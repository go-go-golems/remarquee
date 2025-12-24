package render

import (
	"bytes"
	"context"
	"io"

	"github.com/go-go-golems/remarquee/pkg/rmdoc"
	"github.com/pkg/errors"
	"github.com/unidoc/unipdf/v3/contentstream"
	"github.com/unidoc/unipdf/v3/contentstream/draw"
	"github.com/unidoc/unipdf/v3/core"
	"github.com/unidoc/unipdf/v3/creator"
)

const (
	rmv6ScreenWidth  = 1404
	rmv6ScreenHeight = 1872
	rmv6ScreenDPI    = 226.0
)

// rmv6Scale maps reMarkable screen units to PDF points. Mirrors rmc's SCALE=72/226.
const rmv6Scale = 72.0 / rmv6ScreenDPI

// RenderRMV6StrokesToPDF renders decoded V6 strokes to a single-page PDF.
// This is a minimal “strokes-only” renderer that:
// - applies the rmc coordinate transform (scale + X shift),
// - inverts Y to PDF coordinates,
// - draws polyline strokes with a fixed stroke width.
//
// NOTE: This does not yet implement highlight/tapered brushes, group anchors, or merge math.
func RenderRMV6StrokesToPDF(ctx context.Context, strokes []rmdoc.Stroke, out io.Writer) error {
	_ = ctx

	c := creator.New()
	pageWidth := float64(rmv6ScreenWidth) * rmv6Scale
	pageHeight := float64(rmv6ScreenHeight) * rmv6Scale
	c.SetPageSize(creator.PageSize{pageWidth, pageHeight})

	page := c.NewPage()
	if page == nil {
		return errors.New("creator.NewPage returned nil")
	}

	xShift := pageWidth / 2.0

	cc := contentstream.NewContentCreator()
	cc.Add_q()

	// Fixed stroke style for now (black, 1pt).
	cc.Add_w(1.0)
	cc.Add_RG(0, 0, 0)

	for _, s := range strokes {
		if len(s.Points) == 0 {
			continue
		}
		path := draw.NewPath()
		for _, p := range s.Points {
			x := p.X*rmv6Scale + xShift
			y := pageHeight - (p.Y * rmv6Scale)
			path = path.AppendPoint(draw.NewPoint(x, y))
		}
		draw.DrawPathWithCreator(path, cc)
		cc.Add_S()
	}

	cc.Add_Q()

	page.SetContentStreams([]string{cc.Operations().String()}, core.NewFlateEncoder())

	return c.Write(out)
}

// RenderRMV6RMToPDF is a convenience wrapper that:
// - parses the v6 .rm file into a scene tree,
// - decodes line items into strokes,
// - renders those strokes to a single-page PDF.
func RenderRMV6RMToPDF(ctx context.Context, rm io.ReadSeeker, out io.Writer) error {
	tree, err := rmdoc.ParseRMV6SceneTree(rm)
	if err != nil {
		return errors.Wrap(err, "parse v6 scene tree")
	}

	var strokes []rmdoc.Stroke
	for _, g := range tree.Groups() {
		items, err := g.Children.Items()
		if err != nil {
			return errors.Wrap(err, "order group children")
		}
		for _, it := range items {
			if it.Value.Kind != rmdoc.RMV6SceneItemLine || it.Value.Line == nil {
				continue
			}
			st, err := rmdoc.DecodeRMV6Line(it.Value.Line.BlockVersion, it.Value.Line.Raw)
			if err != nil {
				return errors.Wrap(err, "decode v6 line")
			}
			strokes = append(strokes, *st)
		}
	}

	return RenderRMV6StrokesToPDF(ctx, strokes, out)
}

// RenderRMV6RMToPDFBytes is a convenience function for tests/callers.
func RenderRMV6RMToPDFBytes(ctx context.Context, rm io.ReadSeeker) ([]byte, error) {
	var buf bytes.Buffer
	if err := RenderRMV6RMToPDF(ctx, rm, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
