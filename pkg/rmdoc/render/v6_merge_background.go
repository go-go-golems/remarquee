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
	pdf "github.com/unidoc/unipdf/v3/model"
)

type V6MergeOptions struct {
	// StrokeWidthPt is used for all strokes (milestone renderer).
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
// - This currently merges strokes only (no highlights/text).
// - For pages with no background content, it replaces the blank page with an annotation-only page.
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

	bgBytes, err := BuildBackgroundPDF(ctx, doc, BackgroundOptions{})
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

		stBBox, ok := rmdoc.BBoxForStrokes(strokes, 0)
		if !ok || stBBox.IsEmpty() {
			// Empty annotations.
			if err := w.AddPage(bgPage); err != nil {
				return nil, err
			}
			continue
		}

		xMin, xMax := stBBox.MinX, stBBox.MaxX
		yMin, yMax := stBBox.MinY, stBBox.MaxY

		xShift := xx(xMin)
		yShift := yy(yMin)
		wSvg := xx((xMax - xMin) + 1)
		hSvg := yy((yMax - yMin) + 1)

		// Background dims + rotation.
		w0, h0, rot, err := pageBoxDims(bgPage)
		if err != nil {
			return nil, errors.Wrapf(err, "background page dims (page=%d)", pageNum)
		}

		wBg, hBg := displayDims(w0, h0, rot)

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

		// If background has no content, mimic remarks: replace page with annotations only.
		bgContent, _ := bgPage.GetAllContentStreams()
		if strings.TrimSpace(bgContent) == "" {
			mergedPage, err := buildAnnotationOnlyPage(wSvg, hSvg, strokes, stBBox, wSvg, hSvg, opts)
			if err != nil {
				return nil, err
			}
			if err := applySmartHighlights(mergedPage, glyphRanges, highlightsXTranslation[i], hSvg); err != nil {
				return nil, err
			}
			if err := w.AddPage(mergedPage); err != nil {
				return nil, err
			}
			continue
		}

		mergedPage, err := buildMergedPage(width, height, xBg, yBg, xSvg, ySvg, bgPage, bgContent, rot, w0, h0, strokes, stBBox, wSvg, hSvg, opts)
		if err != nil {
			return nil, err
		}
		if err := applySmartHighlights(mergedPage, glyphRanges, highlightsXTranslation[i], height); err != nil {
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

func pageBoxDims(p *pdf.PdfPage) (w, h float64, rotation int64, err error) {
	box := p.CropBox
	if box == nil {
		box, err = p.GetMediaBox()
		if err != nil {
			return 0, 0, 0, err
		}
	}
	w = box.Urx - box.Llx
	h = box.Ury - box.Lly
	if p.Rotate != nil {
		rotation = *p.Rotate
	}
	return w, h, rotation, nil
}

func displayDims(w0, h0 float64, rotation int64) (w, h float64) {
	w, h = w0, h0
	if rotation == 90 || rotation == 270 {
		w, h = h, w
	}
	return w, h
}

func buildAnnotationOnlyPage(width, height float64, strokes []rmdoc.Stroke, bbox rmdoc.BBox, wSvg, hSvg float64, opts V6MergeOptions) (*pdf.PdfPage, error) {
	page := pdf.NewPdfPage()
	page.MediaBox = &pdf.PdfRectangle{Llx: 0, Lly: 0, Urx: width, Ury: height}
	page.CropBox = &pdf.PdfRectangle{Llx: 0, Lly: 0, Urx: width, Ury: height}
	page.Resources = pdf.NewPdfPageResources()

	overlayOps := buildOverlayOps(strokes, bbox, 0, 0, wSvg, hSvg, opts)
	page.SetContentStreams([]string{overlayOps}, core.NewFlateEncoder())
	return page, nil
}

func buildOverlayOps(strokes []rmdoc.Stroke, bbox rmdoc.BBox, xSvg, ySvg, wSvg, hSvg float64, opts V6MergeOptions) string {
	cc := contentstream.NewContentCreator()
	cc.Add_q()

	cc.Add_w(opts.StrokeWidthPt)
	cc.Add_RG(0, 0, 0)
	lastColor := uint32(0)
	lastColorSet := false

	for _, s := range strokes {
		if len(s.Points) == 0 {
			continue
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
	return "% rmv6-overlay\n" + cc.Operations().String()
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
func backgroundTransform(pageRotation int64, w0, h0, xBg, yBg float64) (a, b, c, d, e, f float64) {
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

func buildMergedPage(width, height, xBg, yBg, xSvg, ySvg float64, bgPage *pdf.PdfPage, bgContent string, pageRotation int64, w0, h0 float64, strokes []rmdoc.Stroke, bbox rmdoc.BBox, wSvg, hSvg float64, opts V6MergeOptions) (*pdf.PdfPage, error) {
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

	// Compose content: background form first, then overlay strokes.
	cc := contentstream.NewContentCreator()
	cc.Add_q()
	a, b, c, d, e, f := backgroundTransform(pageRotation, w0, h0, xBg, yBg)
	cc.Add_cm(a, b, c, d, e, f)
	cc.Add_Do(core.PdfObjectName("Bg"))
	cc.Add_Q()

	overlayOps := buildOverlayOps(strokes, bbox, xSvg, ySvg, wSvg, hSvg, opts)
	content := cc.Operations().String() + "\n" + overlayOps

	merged.SetContentStreams([]string{content}, core.NewFlateEncoder())
	return merged, nil
}

func applySmartHighlights(page *pdf.PdfPage, glyphRanges []rmdoc.RMV6GlyphRange, highlightsXTranslation, pageHeight float64) error {
	_, _ = page.GetAnnotations() // ensure list exists
	for _, gr := range glyphRanges {
		if len(gr.Rectangles) == 0 {
			continue
		}

		r, g, b := rmdoc.PenColorToRGB(gr.Color)

		quadArr := core.MakeArray()
		union := rmdoc.NewEmptyBBox()
		any := false

		for _, rect := range gr.Rectangles {
			x1 := xx(rect.X) + highlightsXTranslation
			x2 := x1 + xx(rect.W)

			// rect.Y is in screen coords (top-origin, y down). Convert to PDF (bottom-origin, y up).
			yTop := pageHeight - yy(rect.Y)
			yBottom := pageHeight - yy(rect.Y+rect.H)

			quadArr.Append(core.MakeFloat(x1))
			quadArr.Append(core.MakeFloat(yTop))
			quadArr.Append(core.MakeFloat(x2))
			quadArr.Append(core.MakeFloat(yTop))
			quadArr.Append(core.MakeFloat(x1))
			quadArr.Append(core.MakeFloat(yBottom))
			quadArr.Append(core.MakeFloat(x2))
			quadArr.Append(core.MakeFloat(yBottom))

			union = union.Union(rmdoc.BBox{MinX: x1, MinY: yBottom, MaxX: x2, MaxY: yTop})
			any = true
		}

		if !any || union.IsEmpty() {
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
