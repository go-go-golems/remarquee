package render

// Annotations-only V6 renderer (ticket RMQ-0021).
//
// This file and v6_annotations_only_pdf.go are deliberately free of unipdf
// imports (design doc DR-6): the annotations-only path only *writes* a simple
// PDF from scratch, so a minimal stdlib writer covers it, avoiding the AGPL
// watermark that unipdf emits when rmapi's go:linkname community-license init
// is not linked into the binary. The composite merge pipeline
// (v6_merge_background.go) keeps unipdf because it must read and composite
// existing PDF pages.
//
// The stroke/text/highlight emission logic mirrors buildOverlayOpsBBoxScaled,
// appendTypedTextOpsBBoxScaled, and applySmartHighlightsScaled in
// v6_merge_background.go — keep the two implementations in sync for styling
// (widths, alphas, colors, font sizes).

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/go-go-golems/remarquee/pkg/rmdoc"
	"github.com/pkg/errors"
	"golang.org/x/text/encoding/charmap"
)

// RenderRMDocV6AnnotationsOnlyWithInfo renders only V6 annotation content
// (strokes, smart highlights, and typed text) onto blank pages, without reading
// or compositing the background payload PDF. This is the V6 equivalent of
// rmapi's PdfGeneratorOptions{AnnotationsOnly: true} used by render-legacy.
//
// pageIndices are 0-based indexes into doc.Pages (UI order), the same convention
// as MergeRMDocV6OntoBackgroundPDFWithInfoForPages.
//
// Page emission semantics (mirroring render-legacy):
//   - When includeUnannotated is false, pages without any V6 annotation content
//     are skipped (rmapi's default with AnnotationsOnly: pages you did not draw
//     on produce no output page).
//   - When includeUnannotated is true, such pages are emitted as blank
//     device-sized pages (the render-legacy CLI's forced AllPages behavior when
//     an explicit --pages subset is selected).
//
// Page geometry reuses the existing overlay-only canvas math
// (annotationCanvasBBox + overlayOnlyPageGeometry), so output is consistent
// with what the merge pipeline already produces for blank-background pages.
//
// V6MergeResult.HighlightsXTranslation has one entry per emitted page (in
// emission order); blank pages contributed by includeUnannotated have a 0 entry.
func RenderRMDocV6AnnotationsOnlyWithInfo(
	ctx context.Context,
	rmdocPath string,
	opts V6MergeOptions,
	pageIndices []int,
	includeUnannotated bool,
) (*V6MergeResult, error) {
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

	pages := make([]pdfPageSpec, 0, len(pageIndices))
	highlightsXTranslation := make([]float64, 0, len(pageIndices))

	for _, pageIdx := range pageIndices {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		var rm *rmdoc.RMFile
		if pageID := doc.Pages[pageIdx].PageID; pageID != "" {
			f, ok, err := rmdoc.ReadRMFileFromArchive(ctx, rmdocPath, pageID)
			if err != nil {
				return nil, err
			}
			if ok && f.Version == "V6" {
				rm = f
			}
		}

		if rm == nil || len(rm.Bytes) == 0 {
			// No V6 annotation file for this page.
			if includeUnannotated {
				pages = append(pages, blankDevicePageSpec())
				highlightsXTranslation = append(highlightsXTranslation, 0)
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
			// Annotation file exists but carries no renderable content.
			if includeUnannotated {
				pages = append(pages, blankDevicePageSpec())
				highlightsXTranslation = append(highlightsXTranslation, 0)
			}
			continue
		}

		bbox, stBBox, ok := annotationCanvasBBox(strokes)
		bboxWithPad, pageW, pageH, scale := overlayOnlyPageGeometry(strokes, bbox, stBBox, ok)

		xTranslation := -xxScaled(bboxWithPad.MinX, scale)
		// Highlights must be translated by the same canvas origin as strokes/text:
		// strokes use (p.Y - bbox.MinY) but glyph rectangles are absolute screen
		// coords, so pass the vertical translation explicitly (PR #21 review).
		yTranslation := -yyScaled(bboxWithPad.MinY, scale)
		spec := buildAnnotationsOnlyPageSpec(strokes, tree.RootText, textParagraphs, glyphRanges,
			bboxWithPad, pageW, pageH, scale, xTranslation, yTranslation, opts)
		pages = append(pages, spec)
		highlightsXTranslation = append(highlightsXTranslation, xTranslation)
	}

	pdfBytes, err := writeSimplePDF(pages)
	if err != nil {
		return nil, errors.Wrap(err, "write annotations-only pdf")
	}
	return &V6MergeResult{
		PDF:                    pdfBytes,
		HighlightsXTranslation: highlightsXTranslation,
	}, nil
}

// blankDevicePageSpec returns a blank page sized like the reMarkable screen at
// the CairoSVG-effective scale, matching the blank pages the merge pipeline
// produces for notebook backgrounds.
func blankDevicePageSpec() pdfPageSpec {
	scale := rmv6Scale * cairoSVGScale
	return pdfPageSpec{
		width:  float64(rmv6ScreenWidth) * scale,
		height: float64(rmv6ScreenHeight) * scale,
	}
}

// buildAnnotationsOnlyPageSpec builds one page worth of content-stream
// operators plus highlight annotations for an annotated page.
// Mirrors buildOverlayOpsBBoxScaled + appendTypedTextOpsBBoxScaled (stroke and
// text ops) and applySmartHighlightsScaled (highlight dicts).
func buildAnnotationsOnlyPageSpec(
	strokes []rmdoc.Stroke,
	rt *rmdoc.RMV6RootText,
	paragraphs []rmdoc.RMV6TextParagraph,
	glyphRanges []rmdoc.RMV6GlyphRange,
	bbox rmdoc.BBox,
	pageW, pageH, scale, xTranslation, yTranslation float64,
	opts V6MergeOptions,
) pdfPageSpec {
	var b strings.Builder
	b.WriteString("% rmv6-overlay\n")

	// Stroke overlay (mirror of buildOverlayOpsBBoxScaled with xSvg=ySvg=0,
	// wSvg=pageW, hSvg=pageH).
	b.WriteString("q\n")
	fmt.Fprintf(&b, "%s w\n", pdfNum(opts.StrokeWidthPt))
	fmt.Fprintf(&b, "/%s gs\n", alphaGStateNameStr(1.0))
	b.WriteString("1 J\n")
	b.WriteString("0 0 0 RG\n")

	alphas := []float64{1.0}
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
		alphas = appendAlphaOnce(alphas, st.Opacity)

		widthPt := st.WidthScreenUnits * scale
		if widthPt <= 0 {
			widthPt = opts.StrokeWidthPt
		}
		if math.Abs(widthPt-lastWidthPt) > 1e-9 {
			fmt.Fprintf(&b, "%s w\n", pdfNum(widthPt))
			lastWidthPt = widthPt
		}
		if ak := alphaKey(st.Opacity); ak != lastAlpha {
			fmt.Fprintf(&b, "/%s gs\n", alphaGStateNameStr(st.Opacity))
			lastAlpha = ak
		}
		if st.LineCap != lastLineCap {
			fmt.Fprintf(&b, "%d J\n", st.LineCap)
			lastLineCap = st.LineCap
		}

		if !lastColorSet || s.Color != lastColor {
			r, g, bl := rmdoc.PenColorToRGBForStroke(rmdoc.PenColor(s.Color))
			fmt.Fprintf(&b, "%s %s %s RG\n", pdfNum(r), pdfNum(g), pdfNum(bl))
			lastColor = s.Color
			lastColorSet = true
		}

		for i, p := range s.Points {
			x := xxScaled(float64(p.X)-bbox.MinX, scale)
			y := pageH - yyScaled(float64(p.Y)-bbox.MinY, scale)
			if i == 0 {
				fmt.Fprintf(&b, "%s %s m\n", pdfNum(x), pdfNum(y))
			} else {
				fmt.Fprintf(&b, "%s %s l\n", pdfNum(x), pdfNum(y))
			}
		}
		b.WriteString("S\n")
	}
	b.WriteString("Q\n")

	// Typed text (mirror of appendTypedTextOpsBBoxScaled).
	needsFonts := false
	if rt != nil && rmdoc.HasNonEmptyRMV6Text(paragraphs) {
		needsFonts = true
		yOffset := rmdoc.RMV6TextTopY
		b.WriteString("BT\n")
		for _, p := range paragraphs {
			yOffset += rmdoc.RMV6ParagraphLineHeight(p.Style)

			text := strings.TrimSpace(p.Text)
			if text == "" {
				continue
			}

			fontName, fontSize := typedTextFontForStyleStr(p.Style)

			xScreen := rt.PosX
			yScreen := rt.PosY + yOffset

			x := xxScaled(xScreen-bbox.MinX, scale)
			y := pageH - yyScaled(yScreen-bbox.MinY, scale)

			fmt.Fprintf(&b, "/%s %s Tf\n", fontName, pdfNum(fontSize))
			fmt.Fprintf(&b, "1 0 0 1 %s %s Tm\n", pdfNum(x), pdfNum(y))
			fmt.Fprintf(&b, "(%s) Tj\n", pdfEscapeString(string(encodeWinAnsiText(text))))
		}
		b.WriteString("ET\n")
	}

	return pdfPageSpec{
		width:      pageW,
		height:     pageH,
		content:    b.String(),
		alphas:     alphas,
		needsFonts: needsFonts,
		annots:     buildHighlightAnnots(glyphRanges, xTranslation, yTranslation, pageH, scale),
	}
}

// appendAlphaOnce appends a to alphas if its quantized key is not present yet.
func appendAlphaOnce(alphas []float64, a float64) []float64 {
	key := alphaKey(a)
	for _, existing := range alphas {
		if alphaKey(existing) == key {
			return alphas
		}
	}
	return append(alphas, a)
}

// typedTextFontForStyleStr mirrors typedTextFontForStyle without unipdf types.
func typedTextFontForStyleStr(style uint8) (string, float64) {
	switch style {
	case 2: // HEADING
		return "RMQTxtHeading", 14.0
	case 3: // BOLD
		return "RMQTxtBold", 8.0
	default:
		return "RMQTxtPlain", 7.0
	}
}

// encodeWinAnsiText converts UTF-8 text to Windows-1252 (WinAnsiEncoding)
// bytes for use with the base-14 fonts. Runes that WinAnsi cannot represent
// (CJK, emoji, ...) are replaced with '?' — mojibake-free degradation instead
// of emitting raw UTF-8 bytes, which PDF viewers would misread as separate
// single-byte character codes (PR #21 review). Full Unicode support would
// require an embedded CID font with Identity-H, a deliberate non-goal for the
// stdlib writer.
func encodeWinAnsiText(s string) []byte {
	enc := charmap.Windows1252.NewEncoder()
	out := make([]byte, 0, len(s))
	for _, r := range s {
		b, err := enc.Bytes([]byte(string(r)))
		if err != nil || len(b) != 1 {
			out = append(out, '?')
			continue
		}
		out = append(out, b[0])
	}
	return out
}

// buildHighlightAnnots mirrors applySmartHighlightsScaled: glyph-range
// rectangles become PDF Highlight annotations with quad points.
//
// Note: unlike applySmartHighlightsScaled, this function takes yTranslation so
// highlights land on the same canvas origin as strokes and typed text
// ((coord - bbox.MinY) * scale). The merge pipeline's blank-background branch
// has the same missing-translation issue; fixing it there changes
// golden-covered composite output, so it is tracked as a follow-up.
func buildHighlightAnnots(glyphRanges []rmdoc.RMV6GlyphRange, xTranslation, yTranslation, pageHeight, scale float64) []pdfHighlightAnnot {
	var annots []pdfHighlightAnnot
	for _, gr := range glyphRanges {
		if len(gr.Rectangles) == 0 {
			continue
		}

		r, g, b := rmdoc.PenColorToRGB(gr.Color)

		var quads []float64
		union := rmdoc.NewEmptyBBox()
		hasAny := false

		for _, rect := range gr.Rectangles {
			x1 := xxScaled(rect.X, scale) + xTranslation
			x2 := x1 + xxScaled(rect.W, scale)

			// rect.Y is in screen coords (top-origin, y down). Convert to PDF
			// (bottom-origin, y up), applying the canvas origin translation so
			// highlights align with strokes ((y - bbox.MinY) * scale).
			yTop := pageHeight - yyScaled(rect.Y, scale) - yTranslation
			yBottom := pageHeight - yyScaled(rect.Y+rect.H, scale) - yTranslation

			quads = append(quads, x1, yTop, x2, yTop, x1, yBottom, x2, yBottom)

			union = union.Union(rmdoc.BBox{MinX: x1, MinY: yBottom, MaxX: x2, MaxY: yTop})
			hasAny = true
		}

		if !hasAny || union.IsEmpty() {
			continue
		}

		annots = append(annots, pdfHighlightAnnot{
			r: r, g: g, b: b,
			ca:    0.3,
			rect:  [4]float64{union.MinX, union.MinY, union.MaxX, union.MaxY},
			quads: quads,
		})
	}
	return annots
}
