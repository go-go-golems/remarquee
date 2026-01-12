package render

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/go-go-golems/remarquee/pkg/rmdoc"
	"github.com/pkg/errors"
	"github.com/unidoc/unipdf/v3/contentstream"
	"github.com/unidoc/unipdf/v3/contentstream/draw"
	"github.com/unidoc/unipdf/v3/core"
	"github.com/unidoc/unipdf/v3/creator"
	pdf "github.com/unidoc/unipdf/v3/model"
)

type V6MergeOptions struct {
	// StrokeWidthPt is used as a fallback when a stroke doesn't provide enough
	// information to derive a tool-specific width.
	StrokeWidthPt float64
}

func (o V6MergeOptions) withDefaults() V6MergeOptions {
	if o.StrokeWidthPt <= 0 {
		o.StrokeWidthPt = 1.0
	}
	return o
}

func xx(v float64) float64 { return v * rmv6Scale }
func yy(v float64) float64 { return v * rmv6Scale }

type strokeStyle struct {
	WidthScreenUnits float64
	Opacity          float64
	LineCap          int64 // 0=butt, 1=round, 2=square
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func strokeAlphaFromColorID(color uint32) (float64, bool) {
	// Shader colors encode their alpha in the hardcoded RGBA map.
	// For highlight colors, alpha is always 255 (opaque), and the transparency comes from the tool.
	switch rmdoc.PenColor(color) {
	case rmdoc.PenColorShaderGray:
		return 64.0 / 255.0, true
	case rmdoc.PenColorShaderOrange:
		return 115.0 / 255.0, true
	case rmdoc.PenColorShaderMagenta:
		return 128.0 / 255.0, true
	case rmdoc.PenColorShaderBlue:
		return 77.0 / 255.0, true
	case rmdoc.PenColorShaderRed:
		return 102.0 / 255.0, true
	case rmdoc.PenColorShaderGreen:
		return 128.0 / 255.0, true
	case rmdoc.PenColorShaderYellow:
		return 115.0 / 255.0, true
	case rmdoc.PenColorShaderCyan:
		return 102.0 / 255.0, true
	case rmdoc.PenColorBlack,
		rmdoc.PenColorGray,
		rmdoc.PenColorWhite,
		rmdoc.PenColorYellow,
		rmdoc.PenColorGreen,
		rmdoc.PenColorPink,
		rmdoc.PenColorBlue,
		rmdoc.PenColorRed,
		rmdoc.PenColorGrayOverlap,
		rmdoc.PenColorHighlight,
		rmdoc.PenColorGreen2,
		rmdoc.PenColorCyan,
		rmdoc.PenColorMagenta,
		rmdoc.PenColorYellow2,
		rmdoc.PenColorHighlightYellow,
		rmdoc.PenColorHighlightBlue,
		rmdoc.PenColorHighlightPink,
		rmdoc.PenColorHighlightOrange,
		rmdoc.PenColorHighlightGreen,
		rmdoc.PenColorHighlightGray:
		return 0, false
	default:
		return 0, false
	}
}

func strokeStyleForTool(tool uint32, thicknessScale float64, color uint32) strokeStyle {
	// This mirrors rmc.exporters.writing_tools.Pen.create at a coarse level.
	// We intentionally don't implement dynamic point-driven widths yet; the goal is to match
	// major tool differences (highlighter/shader transparency and brush widths) for goldens.
	switch tool {
	case 5, 18: // HIGHLIGHTER_1, HIGHLIGHTER_2
		return strokeStyle{WidthScreenUnits: 25, Opacity: 0.3, LineCap: 2}
	case 23: // SHADER
		opacity := 1.0
		if a, ok := strokeAlphaFromColorID(color); ok {
			opacity = a
		}
		return strokeStyle{WidthScreenUnits: 12, Opacity: clamp01(opacity), LineCap: 1}
	case 4, 17: // FINELINER_1, FINELINER_2
		return strokeStyle{WidthScreenUnits: thicknessScale * 1.8, Opacity: 1.0, LineCap: 1}
	case 7, 13: // MECHANICAL_PENCIL_1, MECHANICAL_PENCIL_2
		return strokeStyle{WidthScreenUnits: thicknessScale * thicknessScale, Opacity: 0.7, LineCap: 1}
	default:
		// Most pens map thickness_scale directly to a base width.
		return strokeStyle{WidthScreenUnits: thicknessScale, Opacity: 1.0, LineCap: 1}
	}
}

func alphaKey(alpha float64) int {
	return int(math.Round(clamp01(alpha) * 1000))
}

func alphaGStateName(alpha float64) core.PdfObjectName {
	return core.PdfObjectName(fmt.Sprintf("RMQGsA%03d", alphaKey(alpha)))
}

func ensureAlphaGState(res *pdf.PdfPageResources, alpha float64) (core.PdfObjectName, error) {
	name := alphaGStateName(alpha)
	if _, ok := res.GetExtGState(name); ok {
		return name, nil
	}

	d := core.MakeDict()
	d.Set(core.PdfObjectName("CA"), core.MakeFloat(clamp01(alpha))) // stroke alpha
	d.Set(core.PdfObjectName("ca"), core.MakeFloat(clamp01(alpha))) // fill alpha
	if err := res.AddExtGState(name, d); err != nil {
		return "", err
	}
	return name, nil
}

func ensureStrokeGStates(res *pdf.PdfPageResources, strokes []rmdoc.Stroke) error {
	// Always ensure opaque state exists so we can reset after transparent strokes.
	if _, err := ensureAlphaGState(res, 1.0); err != nil {
		return err
	}
	for _, s := range strokes {
		st := strokeStyleForTool(s.Tool, s.ThicknessScale, s.Color)
		if _, err := ensureAlphaGState(res, st.Opacity); err != nil {
			return err
		}
	}
	return nil
}

func maxStrokeWidthScreenUnits(strokes []rmdoc.Stroke) float64 {
	maxWidth := 0.0
	for _, s := range strokes {
		st := strokeStyleForTool(s.Tool, s.ThicknessScale, s.Color)
		if st.WidthScreenUnits > maxWidth {
			maxWidth = st.WidthScreenUnits
		}
	}
	return maxWidth
}

func setLineCap(cc *contentstream.ContentCreator, lineCap int64) {
	ops := cc.Operations()
	*ops = append(*ops, &contentstream.ContentStreamOperation{
		Params:  []core.PdfObject{core.MakeInteger(lineCap)},
		Operand: "J",
	})
}

type typedTextFontNames struct {
	Plain   core.PdfObjectName
	Bold    core.PdfObjectName
	Heading core.PdfObjectName
}

func ensureTypedTextFonts(res *pdf.PdfPageResources) (typedTextFontNames, error) {
	plainName := core.PdfObjectName("RMQTxtPlain")
	boldName := core.PdfObjectName("RMQTxtBold")
	headingName := core.PdfObjectName("RMQTxtHeading")

	if !res.HasFontByName(plainName) {
		f, err := pdf.NewStandard14Font(pdf.HelveticaName)
		if err != nil {
			return typedTextFontNames{}, err
		}
		if err := res.SetFontByName(plainName, f.ToPdfObject()); err != nil {
			return typedTextFontNames{}, err
		}
	}
	if !res.HasFontByName(boldName) {
		f, err := pdf.NewStandard14Font(pdf.HelveticaBoldName)
		if err != nil {
			return typedTextFontNames{}, err
		}
		if err := res.SetFontByName(boldName, f.ToPdfObject()); err != nil {
			return typedTextFontNames{}, err
		}
	}
	if !res.HasFontByName(headingName) {
		f, err := pdf.NewStandard14Font(pdf.TimesRomanName)
		if err != nil {
			return typedTextFontNames{}, err
		}
		if err := res.SetFontByName(headingName, f.ToPdfObject()); err != nil {
			return typedTextFontNames{}, err
		}
	}

	return typedTextFontNames{
		Plain:   plainName,
		Bold:    boldName,
		Heading: headingName,
	}, nil
}

func typedTextFontForStyle(style uint8, fonts typedTextFontNames) (core.PdfObjectName, float64) {
	// Mirror rmc's svg exporter style choices:
	// - heading: 14pt serif
	// - bold: 8pt sans-serif bold
	// - default: 7pt sans-serif
	switch style {
	case 2: // HEADING
		return fonts.Heading, 14.0
	case 3: // BOLD
		return fonts.Bold, 8.0
	default:
		return fonts.Plain, 7.0
	}
}

// cairoSVGScale is the implicit "CSS px -> PDF pt" scale factor used by CairoSVG.
// CairoSVG treats unitless SVG width/height as CSS pixels at 96dpi, and writes PDFs in 72pt/in:
// 1px = 72/96 pt = 0.75 pt.
//
// This matters because remarks/rmc generate notebook pages by converting SVG->PDF via CairoSVG
// and then inserting the PDF pages directly, resulting in a 0.75 scale factor for notebook page boxes.
const cairoSVGScale = 72.0 / 96.0

func xxScaled(v, scale float64) float64 { return v * scale }
func yyScaled(v, scale float64) float64 { return v * scale }

type V6MergeResult struct {
	PDF []byte

	// HighlightsXTranslation is per-page (0-based UI order) and matches remarks'
	// highlights_x_translation used when positioning smart highlight rectangles.
	HighlightsXTranslation []float64
}

// MergeRMDocV6OntoBackgroundPDF produces a merged PDF for a (typically PDF-backed) rmdoc:
// background pages from the UI page plan + V6 annotation strokes overlaid using the
// "remarks" merge math (canvas sizing + x/y shifts).
//
// Notes:
//   - This merges strokes + smart highlights + root typed text (minimal).
//   - For pages with no background content, we still keep the background page size (no bbox-cropping),
//     and overlay strokes onto a blank page.
func MergeRMDocV6OntoBackgroundPDF(ctx context.Context, rmdocPath string, opts V6MergeOptions) ([]byte, error) {
	res, err := MergeRMDocV6OntoBackgroundPDFWithInfo(ctx, rmdocPath, opts)
	if err != nil {
		return nil, err
	}
	return res.PDF, nil
}

func MergeRMDocV6OntoBackgroundPDFWithInfo(ctx context.Context, rmdocPath string, opts V6MergeOptions) (*V6MergeResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	opts = opts.withDefaults()

	doc, err := rmdoc.OpenFile(ctx, rmdocPath)
	if err != nil {
		return nil, err
	}

	isNotebook := len(doc.PayloadPDF) == 0

	// For notebooks (no payload PDF) remarks/rmc use the reMarkable screen size scaled to PDF points.
	// Keep our blank background pages aligned with that to avoid mismatched page boxes vs reference PDFs.
	bgPageScale := rmv6Scale
	if isNotebook {
		// For notebook output, remarks ultimately produces pages sized like CairoSVG's PDF output.
		// That means the effective page size is 0.75x the "rmc SVG" point size.
		bgPageScale = rmv6Scale * cairoSVGScale
	}

	bgBytes, err := BuildBackgroundPDF(ctx, doc, BackgroundOptions{
		DefaultPageSize: creator.PageSize{
			float64(rmv6ScreenWidth) * bgPageScale,
			float64(rmv6ScreenHeight) * bgPageScale,
		},
	})
	if err != nil {
		return nil, err
	}

	bgReader, err := pdf.NewPdfReader(bytes.NewReader(bgBytes))
	if err != nil {
		return nil, errors.Wrap(err, "open background pdf reader")
	}

	numPages, err := bgReader.GetNumPages()
	if err != nil {
		return nil, errors.Wrap(err, "get background num pages")
	}

	w := pdf.NewPdfWriter()
	highlightsXTranslation := make([]float64, numPages)

	for i := 0; i < numPages; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		pageNum := i + 1
		bgPage, err := bgReader.GetPage(pageNum)
		if err != nil {
			return nil, errors.Wrapf(err, "get background page %d", pageNum)
		}

		// Try read matching rm annotation file (by pageID from UI plan).
		var rm *rmdoc.RMFile
		if i < len(doc.Pages) && doc.Pages[i].PageID != "" {
			f, ok, err := rmdoc.ReadRMFileFromArchive(ctx, rmdocPath, doc.Pages[i].PageID)
			if err != nil {
				return nil, err
			}
			if ok && f.Version == "V6" {
				rm = f
			}
		}

		if rm == nil || len(rm.Bytes) == 0 {
			// No annotations -> keep background as-is.
			if err := w.AddPage(bgPage); err != nil {
				return nil, err
			}
			continue
		}

		tree, err := rmdoc.ParseRMV6SceneTree(bytes.NewReader(rm.Bytes))
		if err != nil {
			return nil, errors.Wrapf(err, "parse v6 scene tree (page=%d pageID=%s)", i, rm.PageID)
		}

		strokes, err := rmdoc.ExtractRMV6StrokesWithAnchors(tree)
		if err != nil {
			return nil, errors.Wrapf(err, "extract v6 strokes (page=%d pageID=%s)", i, rm.PageID)
		}

		glyphRanges, err := rmdoc.ExtractRMV6GlyphRangesWithAnchors(tree)
		if err != nil {
			return nil, errors.Wrapf(err, "extract v6 glyph ranges (page=%d pageID=%s)", i, rm.PageID)
		}

		textParagraphs, err := rmdoc.BuildRMV6TextDocument(tree.RootText)
		if err != nil {
			return nil, errors.Wrapf(err, "build v6 text document (page=%d pageID=%s)", i, rm.PageID)
		}
		hasTypedText := tree.RootText != nil && rmdoc.HasNonEmptyRMV6Text(textParagraphs)

		if len(strokes) == 0 && len(glyphRanges) == 0 && !hasTypedText {
			// Empty annotations.
			if err := w.AddPage(bgPage); err != nil {
				return nil, err
			}
			continue
		}

		// BBox: match rmc/remarks minimum extents of a full screen. This prevents
		// "annotation bbox" shrinkage from clipping root text/highlights and keeps
		// merge math stable for text-only pages.
		defaultBBox := rmdoc.BBox{
			MinX: -float64(rmv6ScreenWidth) / 2.0,
			MaxX: float64(rmv6ScreenWidth) / 2.0,
			MinY: 0,
			MaxY: float64(rmv6ScreenHeight),
		}
		stBBox, ok := rmdoc.BBoxForStrokes(strokes, 0)
		bbox := defaultBBox
		if ok && !stBBox.IsEmpty() {
			bbox = bbox.Union(stBBox)
		}

		// Background dims + rotation.
		w0, h0, rot, err := pageBoxDims(bgPage)
		if err != nil {
			return nil, errors.Wrapf(err, "background page dims (page=%d)", pageNum)
		}

		wBg, hBg := displayDims(w0, h0, rot)

		bgContent, _ := bgPage.GetAllContentStreams()
		if strings.TrimSpace(bgContent) == "" {
			// Blank-background behavior (remarks):
			// When a background page has no content stream, remarks inserts the rmc-produced SVG PDF
			// directly (no show_pdf_page scaling). That PDF is produced by CairoSVG, which applies the
			// implicit 0.75 px->pt scale factor (72/96).
			//
			// To match remarks across both notebooks and inserted blank pages in PDF-backed docs,
			// render using the bbox-derived page size (so strokes outside the default canvas are kept),
			// with the CairoSVG-effective scale.
			scale := rmv6Scale * cairoSVGScale
			pad := maxStrokeWidthScreenUnits(strokes) / 2.0
			if pad < 1.0 {
				pad = 1.0
			}
			bboxWithPad := bbox.Expand(pad)
			if ok && !stBBox.IsEmpty() {
				defaultMinX := -float64(rmv6ScreenWidth) / 2.0
				defaultMaxX := float64(rmv6ScreenWidth) / 2.0
				leftMargin := stBBox.MinX - defaultMinX
				rightMargin := defaultMaxX - stBBox.MaxX
				if leftMargin < 0 {
					leftMargin = 0
				}
				if rightMargin < 0 {
					rightMargin = 0
				}
				if leftMargin > rightMargin {
					bboxWithPad.MaxX += leftMargin - rightMargin
				} else if rightMargin > leftMargin {
					bboxWithPad.MinX -= rightMargin - leftMargin
				}
			}
			pageW := xxScaled((bboxWithPad.MaxX-bboxWithPad.MinX)+1, scale)
			pageH := yyScaled((bboxWithPad.MaxY-bboxWithPad.MinY)+1, scale)

			mergedPage, err := buildOverlayOnlyPageBBoxScaled(pageW, pageH, strokes, tree.RootText, textParagraphs, bboxWithPad, scale, opts)
			if err != nil {
				return nil, err
			}
			highlightsXTranslation[i] = -xxScaled(bboxWithPad.MinX, scale)
			if err := applySmartHighlightsScaled(mergedPage, glyphRanges, highlightsXTranslation[i], pageH, scale); err != nil {
				return nil, err
			}
			if err := w.AddPage(mergedPage); err != nil {
				return nil, err
			}
			continue
		}

		// Background-present behavior (remarks merge math):
		// Use bbox-based canvas sizing and shifts so annotations align with the payload background.
		xMin, xMax := bbox.MinX, bbox.MaxX
		yMin, yMax := bbox.MinY, bbox.MaxY

		xShift := xx(xMin)
		yShift := yy(yMin)
		wSvg := xx((xMax - xMin) + 1)
		hSvg := yy((yMax - yMin) + 1)

		width := math.Max(wSvg, wBg)
		height := math.Max(hSvg, hBg)

		var xSvg, ySvg, xBg, yBg float64
		highlightsX := wBg / 2.0
		if wSvg > wBg {
			xBg = width/2 - wBg/2 - (wSvg/2 + xShift)
			highlightsX = xBg + wBg/2
		} else if wSvg < wBg {
			xSvg = width/2 - wSvg/2 + (wSvg/2 + xShift)
		}
		highlightsXTranslation[i] = highlightsX

		if hSvg > hBg {
			yBg = -yShift
		} else if hSvg < hBg {
			ySvg = yShift
		}

		mergedPage, err := buildMergedPage(width, height, xBg, yBg, xSvg, ySvg, bgPage, bgContent, rot, w0, h0, strokes, tree.RootText, textParagraphs, bbox, wSvg, hSvg, opts)
		if err != nil {
			return nil, err
		}
		if err := applySmartHighlightsScaled(mergedPage, glyphRanges, highlightsXTranslation[i], height, rmv6Scale); err != nil {
			return nil, err
		}

		if err := w.AddPage(mergedPage); err != nil {
			return nil, err
		}
	}

	var buf bytes.Buffer
	if err := w.Write(&buf); err != nil {
		return nil, errors.Wrap(err, "write merged pdf")
	}
	return &V6MergeResult{
		PDF:                    buf.Bytes(),
		HighlightsXTranslation: highlightsXTranslation,
	}, nil
}

// MergeRMDocV6OntoBackgroundPDFWithInfoForPages merges only the selected UI pages into a PDF.
// pageIndices are 0-based indexes into doc.Pages.
func MergeRMDocV6OntoBackgroundPDFWithInfoForPages(ctx context.Context, rmdocPath string, opts V6MergeOptions, pageIndices []int) (*V6MergeResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(pageIndices) == 0 {
		return nil, errors.New("pageIndices is empty")
	}
	opts = opts.withDefaults()

	doc, err := rmdoc.OpenFile(ctx, rmdocPath)
	if err != nil {
		return nil, err
	}

	for i, idx := range pageIndices {
		if idx < 0 || idx >= len(doc.Pages) {
			return nil, errors.Errorf("pageIndices[%d]=%d out of range (pages=%d)", i, idx, len(doc.Pages))
		}
	}

	isNotebook := len(doc.PayloadPDF) == 0
	bgPageScale := rmv6Scale
	if isNotebook {
		bgPageScale = rmv6Scale * cairoSVGScale
	}

	bgBytes, err := BuildBackgroundPDFForPages(ctx, doc, BackgroundOptions{
		DefaultPageSize: creator.PageSize{
			float64(rmv6ScreenWidth) * bgPageScale,
			float64(rmv6ScreenHeight) * bgPageScale,
		},
	}, pageIndices)
	if err != nil {
		return nil, err
	}

	bgReader, err := pdf.NewPdfReader(bytes.NewReader(bgBytes))
	if err != nil {
		return nil, errors.Wrap(err, "open background pdf reader")
	}

	numPages, err := bgReader.GetNumPages()
	if err != nil {
		return nil, errors.Wrap(err, "get background num pages")
	}
	if numPages != len(pageIndices) {
		return nil, errors.Errorf("background pages=%d does not match requested pages=%d", numPages, len(pageIndices))
	}

	w := pdf.NewPdfWriter()
	highlightsXTranslation := make([]float64, len(pageIndices))

	for i, pageIdx := range pageIndices {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		pageNum := i + 1

		bgPage, err := bgReader.GetPage(pageNum)
		if err != nil {
			return nil, errors.Wrapf(err, "get background page %d (ui idx=%d)", pageNum, pageIdx)
		}

		var rm *rmdoc.RMFile
		if pageIdx < len(doc.Pages) && doc.Pages[pageIdx].PageID != "" {
			f, ok, err := rmdoc.ReadRMFileFromArchive(ctx, rmdocPath, doc.Pages[pageIdx].PageID)
			if err != nil {
				return nil, err
			}
			if ok && f.Version == "V6" {
				rm = f
			}
		}

		if rm == nil || len(rm.Bytes) == 0 {
			if err := w.AddPage(bgPage); err != nil {
				return nil, err
			}
			continue
		}

		tree, err := rmdoc.ParseRMV6SceneTree(bytes.NewReader(rm.Bytes))
		if err != nil {
			return nil, errors.Wrapf(err, "parse v6 scene tree (page=%d pageID=%s)", pageIdx, rm.PageID)
		}

		strokes, err := rmdoc.ExtractRMV6StrokesWithAnchors(tree)
		if err != nil {
			return nil, errors.Wrapf(err, "extract v6 strokes (page=%d pageID=%s)", pageIdx, rm.PageID)
		}

		glyphRanges, err := rmdoc.ExtractRMV6GlyphRangesWithAnchors(tree)
		if err != nil {
			return nil, errors.Wrapf(err, "extract v6 glyph ranges (page=%d pageID=%s)", pageIdx, rm.PageID)
		}

		textParagraphs, err := rmdoc.BuildRMV6TextDocument(tree.RootText)
		if err != nil {
			return nil, errors.Wrapf(err, "build v6 text document (page=%d pageID=%s)", pageIdx, rm.PageID)
		}
		hasTypedText := tree.RootText != nil && rmdoc.HasNonEmptyRMV6Text(textParagraphs)

		if len(strokes) == 0 && len(glyphRanges) == 0 && !hasTypedText {
			if err := w.AddPage(bgPage); err != nil {
				return nil, err
			}
			continue
		}

		defaultBBox := rmdoc.BBox{
			MinX: -float64(rmv6ScreenWidth) / 2.0,
			MaxX: float64(rmv6ScreenWidth) / 2.0,
			MinY: 0,
			MaxY: float64(rmv6ScreenHeight),
		}
		stBBox, ok := rmdoc.BBoxForStrokes(strokes, 0)
		bbox := defaultBBox
		if ok && !stBBox.IsEmpty() {
			bbox = bbox.Union(stBBox)
		}

		w0, h0, rot, err := pageBoxDims(bgPage)
		if err != nil {
			return nil, errors.Wrapf(err, "background page dims (page=%d ui idx=%d)", pageNum, pageIdx)
		}

		wBg, hBg := displayDims(w0, h0, rot)

		bgContent, _ := bgPage.GetAllContentStreams()
		if strings.TrimSpace(bgContent) == "" {
			scale := rmv6Scale * cairoSVGScale
			pad := maxStrokeWidthScreenUnits(strokes) / 2.0
			if pad < 1.0 {
				pad = 1.0
			}
			bboxWithPad := bbox.Expand(pad)
			if ok && !stBBox.IsEmpty() {
				defaultMinX := -float64(rmv6ScreenWidth) / 2.0
				defaultMaxX := float64(rmv6ScreenWidth) / 2.0
				leftMargin := stBBox.MinX - defaultMinX
				rightMargin := defaultMaxX - stBBox.MaxX
				if leftMargin < 0 {
					leftMargin = 0
				}
				if rightMargin < 0 {
					rightMargin = 0
				}
				if leftMargin > rightMargin {
					bboxWithPad.MaxX += leftMargin - rightMargin
				} else if rightMargin > leftMargin {
					bboxWithPad.MinX -= rightMargin - leftMargin
				}
			}
			pageW := xxScaled((bboxWithPad.MaxX-bboxWithPad.MinX)+1, scale)
			pageH := yyScaled((bboxWithPad.MaxY-bboxWithPad.MinY)+1, scale)

			mergedPage, err := buildOverlayOnlyPageBBoxScaled(pageW, pageH, strokes, tree.RootText, textParagraphs, bboxWithPad, scale, opts)
			if err != nil {
				return nil, err
			}
			highlightsXTranslation[i] = -xxScaled(bboxWithPad.MinX, scale)
			if err := applySmartHighlightsScaled(mergedPage, glyphRanges, highlightsXTranslation[i], pageH, scale); err != nil {
				return nil, err
			}
			if err := w.AddPage(mergedPage); err != nil {
				return nil, err
			}
			continue
		}

		xMin, xMax := bbox.MinX, bbox.MaxX
		yMin, yMax := bbox.MinY, bbox.MaxY

		xShift := xx(xMin)
		yShift := yy(yMin)
		wSvg := xx((xMax - xMin) + 1)
		hSvg := yy((yMax - yMin) + 1)

		width := math.Max(wSvg, wBg)
		height := math.Max(hSvg, hBg)

		var xSvg, ySvg, xBg, yBg float64
		highlightsX := wBg / 2.0
		if wSvg > wBg {
			xBg = width/2 - wBg/2 - (wSvg/2 + xShift)
			highlightsX = xBg + wBg/2
		} else if wSvg < wBg {
			xSvg = width/2 - wSvg/2 + (wSvg/2 + xShift)
		}
		highlightsXTranslation[i] = highlightsX

		if hSvg > hBg {
			yBg = -yShift
		} else if hSvg < hBg {
			ySvg = yShift
		}

		mergedPage, err := buildMergedPage(width, height, xBg, yBg, xSvg, ySvg, bgPage, bgContent, rot, w0, h0, strokes, tree.RootText, textParagraphs, bbox, wSvg, hSvg, opts)
		if err != nil {
			return nil, err
		}
		if err := applySmartHighlightsScaled(mergedPage, glyphRanges, highlightsXTranslation[i], height, rmv6Scale); err != nil {
			return nil, err
		}

		if err := w.AddPage(mergedPage); err != nil {
			return nil, err
		}
	}

	var buf bytes.Buffer
	if err := w.Write(&buf); err != nil {
		return nil, errors.Wrap(err, "write merged pdf")
	}
	return &V6MergeResult{
		PDF:                    buf.Bytes(),
		HighlightsXTranslation: highlightsXTranslation,
	}, nil
}

func pageBoxDims(p *pdf.PdfPage) (float64, float64, int64, error) {
	box := p.CropBox
	if box == nil {
		var err error
		box, err = p.GetMediaBox()
		if err != nil {
			return 0, 0, 0, err
		}
	}
	w := box.Urx - box.Llx
	h := box.Ury - box.Lly
	var rotation int64
	if p.Rotate != nil {
		rotation = *p.Rotate
	}
	return w, h, rotation, nil
}

func displayDims(w0, h0 float64, rotation int64) (float64, float64) {
	w, h := w0, h0
	if rotation == 90 || rotation == 270 {
		w, h = h, w
	}
	return w, h
}

func buildOverlayOnlyPageBBoxScaled(width, height float64, strokes []rmdoc.Stroke, rt *rmdoc.RMV6RootText, paragraphs []rmdoc.RMV6TextParagraph, bbox rmdoc.BBox, scale float64, opts V6MergeOptions) (*pdf.PdfPage, error) {
	page := pdf.NewPdfPage()
	page.MediaBox = &pdf.PdfRectangle{Llx: 0, Lly: 0, Urx: width, Ury: height}
	page.CropBox = &pdf.PdfRectangle{Llx: 0, Lly: 0, Urx: width, Ury: height}
	page.Resources = pdf.NewPdfPageResources()

	var fonts typedTextFontNames
	if rt != nil && rmdoc.HasNonEmptyRMV6Text(paragraphs) {
		var err error
		fonts, err = ensureTypedTextFonts(page.Resources)
		if err != nil {
			return nil, err
		}
	}
	if err := ensureStrokeGStates(page.Resources, strokes); err != nil {
		return nil, err
	}

	overlayOps := buildOverlayOpsBBoxScaled(strokes, rt, paragraphs, bbox, 0, 0, width, height, scale, opts, fonts)
	if err := page.SetContentStreams([]string{overlayOps}, core.NewFlateEncoder()); err != nil {
		return nil, err
	}
	return page, nil
}

func buildOverlayOpsBBoxScaled(strokes []rmdoc.Stroke, rt *rmdoc.RMV6RootText, paragraphs []rmdoc.RMV6TextParagraph, bbox rmdoc.BBox, xSvg, ySvg, wSvg, hSvg, scale float64, opts V6MergeOptions, fonts typedTextFontNames) string {
	cc := contentstream.NewContentCreator()
	cc.Add_q()

	cc.Add_w(opts.StrokeWidthPt)
	cc.Add_gs(alphaGStateName(1.0))
	setLineCap(cc, 1)
	cc.Add_RG(0, 0, 0)
	lastColor := uint32(0)
	lastColorSet := false
	lastAlpha := alphaKey(1.0)
	lastLineCap := int64(1)
	lastWidthPt := opts.StrokeWidthPt

	for _, s := range strokes {
		if len(s.Points) == 0 {
			continue
		}

		st := strokeStyleForTool(s.Tool, s.ThicknessScale, s.Color)
		widthPt := st.WidthScreenUnits * scale
		if widthPt <= 0 {
			widthPt = opts.StrokeWidthPt
		}
		if math.Abs(widthPt-lastWidthPt) > 1e-9 {
			cc.Add_w(widthPt)
			lastWidthPt = widthPt
		}
		if ak := alphaKey(st.Opacity); ak != lastAlpha {
			cc.Add_gs(alphaGStateName(st.Opacity))
			lastAlpha = ak
		}
		if st.LineCap != lastLineCap {
			setLineCap(cc, st.LineCap)
			lastLineCap = st.LineCap
		}

		if !lastColorSet || s.Color != lastColor {
			r, g, b := rmdoc.PenColorToRGBForStroke(rmdoc.PenColor(s.Color))
			cc.Add_RG(r, g, b)
			lastColor = s.Color
			lastColorSet = true
		}

		path := draw.NewPath()
		for _, p := range s.Points {
			x := xSvg + xxScaled(float64(p.X)-bbox.MinX, scale)
			y := ySvg + (hSvg - yyScaled(float64(p.Y)-bbox.MinY, scale))
			path = path.AppendPoint(draw.NewPoint(x, y))
		}
		draw.DrawPathWithCreator(path, cc)
		cc.Add_S()
	}

	cc.Add_Q()
	appendTypedTextOpsBBoxScaled(cc, rt, paragraphs, bbox, xSvg, ySvg, wSvg, hSvg, scale, opts, fonts)
	return "% rmv6-overlay\n" + cc.Operations().String()
}

func buildOverlayOps(strokes []rmdoc.Stroke, rt *rmdoc.RMV6RootText, paragraphs []rmdoc.RMV6TextParagraph, bbox rmdoc.BBox, xSvg, ySvg, wSvg, hSvg float64, opts V6MergeOptions) string {
	cc := contentstream.NewContentCreator()
	cc.Add_q()

	cc.Add_w(opts.StrokeWidthPt)
	cc.Add_gs(alphaGStateName(1.0))
	setLineCap(cc, 1)
	cc.Add_RG(0, 0, 0)
	lastColor := uint32(0)
	lastColorSet := false
	lastAlpha := alphaKey(1.0)
	lastLineCap := int64(1)
	lastWidthPt := opts.StrokeWidthPt

	for _, s := range strokes {
		if len(s.Points) == 0 {
			continue
		}

		st := strokeStyleForTool(s.Tool, s.ThicknessScale, s.Color)
		widthPt := xx(st.WidthScreenUnits)
		if widthPt <= 0 {
			widthPt = opts.StrokeWidthPt
		}
		if math.Abs(widthPt-lastWidthPt) > 1e-9 {
			cc.Add_w(widthPt)
			lastWidthPt = widthPt
		}
		if ak := alphaKey(st.Opacity); ak != lastAlpha {
			cc.Add_gs(alphaGStateName(st.Opacity))
			lastAlpha = ak
		}
		if st.LineCap != lastLineCap {
			setLineCap(cc, st.LineCap)
			lastLineCap = st.LineCap
		}

		// Apply per-stroke color. V6 line items store a "color_id" which matches rmscene/rmc's PenColor enum.
		// Historically we rendered everything black; this ensures colored strokes show up correctly.
		if !lastColorSet || s.Color != lastColor {
			r, g, b := rmdoc.PenColorToRGBForStroke(rmdoc.PenColor(s.Color))
			cc.Add_RG(r, g, b)
			lastColor = s.Color
			lastColorSet = true
		}

		path := draw.NewPath()
		for _, p := range s.Points {
			x := xSvg + xx(float64(p.X)-bbox.MinX)
			y := ySvg + (hSvg - yy(float64(p.Y)-bbox.MinY))
			path = path.AppendPoint(draw.NewPoint(x, y))
		}
		draw.DrawPathWithCreator(path, cc)
		cc.Add_S()
	}

	cc.Add_Q()
	appendTypedTextOpsBBox(cc, rt, paragraphs, bbox, xSvg, ySvg, wSvg, hSvg, opts, typedTextFontNames{
		Plain:   core.PdfObjectName("RMQTxtPlain"),
		Bold:    core.PdfObjectName("RMQTxtBold"),
		Heading: core.PdfObjectName("RMQTxtHeading"),
	})
	return "% rmv6-overlay\n" + cc.Operations().String()
}

func appendTypedTextOpsBBox(cc *contentstream.ContentCreator, rt *rmdoc.RMV6RootText, paragraphs []rmdoc.RMV6TextParagraph, bbox rmdoc.BBox, xSvg, ySvg, wSvg, hSvg float64, _ V6MergeOptions, fonts typedTextFontNames) {
	if rt == nil || !rmdoc.HasNonEmptyRMV6Text(paragraphs) {
		return
	}
	if fonts.Plain == "" {
		return
	}

	yOffset := rmdoc.RMV6TextTopY

	cc.Add_BT()
	for _, p := range paragraphs {
		yOffset += rmdoc.RMV6ParagraphLineHeight(p.Style)

		text := strings.TrimSpace(p.Text)
		if text == "" {
			continue
		}

		fontName, fontSize := typedTextFontForStyle(p.Style, fonts)

		xScreen := rt.PosX
		yScreen := rt.PosY + yOffset

		x := xSvg + xx(xScreen-bbox.MinX)
		y := ySvg + (hSvg - yy(yScreen-bbox.MinY))

		cc.Add_Tf(fontName, fontSize)
		cc.Add_Tm(1, 0, 0, 1, x, y)
		cc.Add_Tj(*core.MakeString(text))
	}
	cc.Add_ET()
}

func appendTypedTextOpsBBoxScaled(cc *contentstream.ContentCreator, rt *rmdoc.RMV6RootText, paragraphs []rmdoc.RMV6TextParagraph, bbox rmdoc.BBox, xSvg, ySvg, wSvg, hSvg, scale float64, _ V6MergeOptions, fonts typedTextFontNames) {
	if rt == nil || !rmdoc.HasNonEmptyRMV6Text(paragraphs) {
		return
	}
	if fonts.Plain == "" {
		return
	}

	yOffset := rmdoc.RMV6TextTopY

	cc.Add_BT()
	for _, p := range paragraphs {
		yOffset += rmdoc.RMV6ParagraphLineHeight(p.Style)

		text := strings.TrimSpace(p.Text)
		if text == "" {
			continue
		}

		fontName, fontSize := typedTextFontForStyle(p.Style, fonts)

		xScreen := rt.PosX
		yScreen := rt.PosY + yOffset

		x := xSvg + xxScaled(xScreen-bbox.MinX, scale)
		y := ySvg + (hSvg - yyScaled(yScreen-bbox.MinY, scale))

		cc.Add_Tf(fontName, fontSize)
		cc.Add_Tm(1, 0, 0, 1, x, y)
		cc.Add_Tj(*core.MakeString(text))
	}
	cc.Add_ET()
}

func normalizeRot(rot int64) int64 {
	rot = rot % 360
	if rot < 0 {
		rot += 360
	}
	return rot
}

// backgroundTransform returns an affine transform matrix (a,b,c,d,e,f) for `cm`
// that places the background page content (in its unrotated coordinate system
// with size w0 x h0) into the merged canvas at (xBg,yBg) with rotate=-page_rotation
// semantics (mirroring PyMuPDF show_pdf_page rotate=-page_rotation).
func backgroundTransform(pageRotation int64, w0, h0, xBg, yBg float64) (float64, float64, float64, float64, float64, float64) {
	rot := normalizeRot(-pageRotation)
	switch rot {
	case 0:
		return 1, 0, 0, 1, xBg, yBg
	case 90:
		// +90deg: (x,y)->(-y, x) then translate by (h0,0)
		return 0, 1, -1, 0, xBg + h0, yBg
	case 180:
		// 180deg: (x,y)->(-x, -y) then translate by (w0,h0)
		return -1, 0, 0, -1, xBg + w0, yBg + h0
	case 270:
		// -90deg: (x,y)->(y, -x) then translate by (0,w0)
		return 0, -1, 1, 0, xBg, yBg + w0
	default:
		// Fallback (should not happen: Rotate must be multiple of 90).
		return 1, 0, 0, 1, xBg, yBg
	}
}

func buildMergedPage(width, height, xBg, yBg, xSvg, ySvg float64, bgPage *pdf.PdfPage, bgContent string, pageRotation int64, w0, h0 float64, strokes []rmdoc.Stroke, rt *rmdoc.RMV6RootText, paragraphs []rmdoc.RMV6TextParagraph, bbox rmdoc.BBox, wSvg, hSvg float64, opts V6MergeOptions) (*pdf.PdfPage, error) {
	merged := pdf.NewPdfPage()
	merged.MediaBox = &pdf.PdfRectangle{Llx: 0, Lly: 0, Urx: width, Ury: height}
	merged.CropBox = &pdf.PdfRectangle{Llx: 0, Lly: 0, Urx: width, Ury: height}
	merged.Resources = pdf.NewPdfPageResources()

	// Create background form XObject from the source page content.
	// Normalize its coordinate system so the form bbox starts at (0,0).
	box := bgPage.CropBox
	if box == nil {
		var err error
		box, err = bgPage.GetMediaBox()
		if err != nil {
			return nil, errors.Wrap(err, "get background page box")
		}
	}
	llx, lly := box.Llx, box.Lly

	bgForm := pdf.NewXObjectForm()
	bgForm.Resources = bgPage.Resources
	bgForm.BBox = core.MakeArrayFromFloats([]float64{0, 0, w0, h0})

	normalized := bgContent
	if llx != 0 || lly != 0 {
		normalized = strings.Join([]string{
			"q",
			// move origin to (0,0)
			fmt.Sprintf("1 0 0 1 %.6f %.6f cm", -llx, -lly),
			bgContent,
			"Q",
		}, "\n")
	}
	if err := bgForm.SetContentStream([]byte(normalized), core.NewFlateEncoder()); err != nil {
		return nil, errors.Wrap(err, "set bg form stream")
	}
	if err := merged.Resources.SetXObjectFormByName(core.PdfObjectName("Bg"), bgForm); err != nil {
		return nil, errors.Wrap(err, "set Bg xobject")
	}

	if rt != nil && rmdoc.HasNonEmptyRMV6Text(paragraphs) {
		if _, err := ensureTypedTextFonts(merged.Resources); err != nil {
			return nil, err
		}
	}
	if err := ensureStrokeGStates(merged.Resources, strokes); err != nil {
		return nil, err
	}

	// Compose content: background form first, then overlay strokes.
	cc := contentstream.NewContentCreator()
	cc.Add_q()
	a, b, c, d, e, f := backgroundTransform(pageRotation, w0, h0, xBg, yBg)
	cc.Add_cm(a, b, c, d, e, f)
	cc.Add_Do(core.PdfObjectName("Bg"))
	cc.Add_Q()

	overlayOps := buildOverlayOps(strokes, rt, paragraphs, bbox, xSvg, ySvg, wSvg, hSvg, opts)
	content := cc.Operations().String() + "\n" + overlayOps

	if err := merged.SetContentStreams([]string{content}, core.NewFlateEncoder()); err != nil {
		return nil, err
	}
	return merged, nil
}

func applySmartHighlightsScaled(page *pdf.PdfPage, glyphRanges []rmdoc.RMV6GlyphRange, highlightsXTranslation, pageHeight, scale float64) error {
	_, _ = page.GetAnnotations() // ensure list exists
	for _, gr := range glyphRanges {
		if len(gr.Rectangles) == 0 {
			continue
		}

		r, g, b := rmdoc.PenColorToRGB(gr.Color)

		quadArr := core.MakeArray()
		union := rmdoc.NewEmptyBBox()
		hasAny := false

		for _, rect := range gr.Rectangles {
			x1 := xxScaled(rect.X, scale) + highlightsXTranslation
			x2 := x1 + xxScaled(rect.W, scale)

			// rect.Y is in screen coords (top-origin, y down). Convert to PDF (bottom-origin, y up).
			yTop := pageHeight - yyScaled(rect.Y, scale)
			yBottom := pageHeight - yyScaled(rect.Y+rect.H, scale)

			quadArr.Append(core.MakeFloat(x1))
			quadArr.Append(core.MakeFloat(yTop))
			quadArr.Append(core.MakeFloat(x2))
			quadArr.Append(core.MakeFloat(yTop))
			quadArr.Append(core.MakeFloat(x1))
			quadArr.Append(core.MakeFloat(yBottom))
			quadArr.Append(core.MakeFloat(x2))
			quadArr.Append(core.MakeFloat(yBottom))

			union = union.Union(rmdoc.BBox{MinX: x1, MinY: yBottom, MaxX: x2, MaxY: yTop})
			hasAny = true
		}

		if !hasAny || union.IsEmpty() {
			continue
		}

		hl := pdf.NewPdfAnnotationHighlight()
		hl.C = core.MakeArrayFromFloats([]float64{r, g, b})
		hl.CA = core.MakeFloat(0.3)
		hl.Rect = core.MakeArrayFromFloats([]float64{union.MinX, union.MinY, union.MaxX, union.MaxY})
		hl.QuadPoints = quadArr

		page.AddAnnotation(hl.PdfAnnotation)
	}
	return nil
}
