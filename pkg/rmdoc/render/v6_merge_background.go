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

// MergeRMDocV6OntoBackgroundPDF produces a merged PDF for a (typically PDF-backed) rmdoc:
// background pages from the UI page plan + V6 annotation strokes overlaid using the
// "remarks" merge math (canvas sizing + x/y shifts).
//
// Notes:
// - This currently merges strokes only (no highlights/text).
// - For pages with no background content, it replaces the blank page with an annotation-only page.
func MergeRMDocV6OntoBackgroundPDF(ctx context.Context, rmdocPath string, opts V6MergeOptions) ([]byte, error) {
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

		// Background dims.
		wBg, hBg, pageRotation, err := pageDimsAndRotation(bgPage)
		if err != nil {
			return nil, errors.Wrapf(err, "background page dims (page=%d)", pageNum)
		}
		_ = pageRotation // rotation handling is still a later refinement (task 46)

		width := math.Max(wSvg, wBg)
		height := math.Max(hSvg, hBg)

		var xSvg, ySvg, xBg, yBg float64
		if wSvg > wBg {
			xBg = width/2 - wBg/2 - (wSvg/2 + xShift)
		} else if wSvg < wBg {
			xSvg = width/2 - wSvg/2 + (wSvg/2 + xShift)
		}

		if hSvg > hBg {
			yBg = -yShift
		} else if hSvg < hBg {
			ySvg = yShift
		}

		// If background has no content, mimic remarks: replace page with annotations only.
		bgContent, _ := bgPage.GetAllContentStreams()
		if strings.TrimSpace(bgContent) == "" {
			if err := applyAnnotationOnly(bgPage, strokes, stBBox, wSvg, hSvg, opts); err != nil {
				return nil, err
			}
			if err := w.AddPage(bgPage); err != nil {
				return nil, err
			}
			continue
		}

		if err := applyMerged(bgPage, width, height, xBg, yBg, xSvg, ySvg, strokes, stBBox, wSvg, hSvg, opts); err != nil {
			return nil, err
		}

		if err := w.AddPage(bgPage); err != nil {
			return nil, err
		}
	}

	var buf bytes.Buffer
	if err := w.Write(&buf); err != nil {
		return nil, errors.Wrap(err, "write merged pdf")
	}
	return buf.Bytes(), nil
}

func pageDimsAndRotation(p *pdf.PdfPage) (w, h float64, rotation int64, err error) {
	crop := p.CropBox
	if crop == nil {
		crop, err = p.GetMediaBox()
		if err != nil {
			return 0, 0, 0, err
		}
	}
	w = crop.Urx - crop.Llx
	h = crop.Ury - crop.Lly
	if p.Rotate != nil {
		rotation = *p.Rotate
	}
	// mirror remarks: swap dims for landscape rotation
	if rotation == 90 || rotation == 270 {
		w, h = h, w
	}
	return w, h, rotation, nil
}

func applyAnnotationOnly(page *pdf.PdfPage, strokes []rmdoc.Stroke, bbox rmdoc.BBox, wSvg, hSvg float64, opts V6MergeOptions) error {
	page.MediaBox = &pdf.PdfRectangle{Llx: 0, Lly: 0, Urx: wSvg, Ury: hSvg}
	page.CropBox = &pdf.PdfRectangle{Llx: 0, Lly: 0, Urx: wSvg, Ury: hSvg}

	ops := buildOverlayOps(strokes, bbox, 0, 0, wSvg, hSvg, opts)
	page.SetContentStreams([]string{ops}, core.NewFlateEncoder())
	return nil
}

func applyMerged(page *pdf.PdfPage, width, height, xBg, yBg, xSvg, ySvg float64, strokes []rmdoc.Stroke, bbox rmdoc.BBox, wSvg, hSvg float64, opts V6MergeOptions) error {
	page.MediaBox = &pdf.PdfRectangle{Llx: 0, Lly: 0, Urx: width, Ury: height}
	page.CropBox = &pdf.PdfRectangle{Llx: 0, Lly: 0, Urx: width, Ury: height}

	bgContent, err := page.GetAllContentStreams()
	if err != nil {
		return errors.Wrap(err, "get page content streams")
	}

	overlayOps := buildOverlayOps(strokes, bbox, xSvg, ySvg, wSvg, hSvg, opts)

	bgTranslate := fmt.Sprintf("1 0 0 1 %.6f %.6f cm", xBg, yBg)

	// Wrap background with translation, then add overlay ops.
	wrapper := []string{
		"q",
		bgTranslate,
		bgContent,
		"Q",
		overlayOps,
	}
	page.SetContentStreams(wrapper, core.NewFlateEncoder())
	return nil
}

func buildOverlayOps(strokes []rmdoc.Stroke, bbox rmdoc.BBox, xSvg, ySvg, wSvg, hSvg float64, opts V6MergeOptions) string {
	cc := contentstream.NewContentCreator()
	cc.Add_q()

	cc.Add_w(opts.StrokeWidthPt)
	cc.Add_RG(0, 0, 0)

	for _, s := range strokes {
		if len(s.Points) == 0 {
			continue
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
