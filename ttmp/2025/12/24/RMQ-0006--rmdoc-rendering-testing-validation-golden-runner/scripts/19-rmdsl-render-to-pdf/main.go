package main

// Usage:
//   go run ./ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/19-rmdsl-render-to-pdf/main.go \
//     --in  ./ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/cases/03-ellipse-sweep.js \
//     --out /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/rendering/rmq-0006-ellipse/ellipse-sweep.pdf
//
// Goal:
//   Render RMDoc-DSL (YAML or JS) into a multi-page PDF for easy on-device review.
//   This is a pragmatic “debug transport” (PDF), not a `.rmdoc` notebook compiler.

import (
	"context"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/remarquee/pkg/rmdsl"
	"github.com/pkg/errors"
	"github.com/unidoc/unipdf/v3/contentstream"
	"github.com/unidoc/unipdf/v3/contentstream/draw"
	"github.com/unidoc/unipdf/v3/core"
	"github.com/unidoc/unipdf/v3/creator"
)

func parseColor(name string) (r, g, b float64) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "black":
		return 0, 0, 0
	case "red":
		return 0.86, 0.20, 0.20
	case "green":
		return 0.15, 0.70, 0.25
	default:
		return 0.1, 0.1, 0.1
	}
}

func toPDF(canvasW int, pageH float64, scale float64, x, y float64) (float64, float64) {
	// rm_screen_v6: x in [-W/2..+W/2], y in [0..H] with y top-down.
	// PDF: origin bottom-left.
	xShift := float64(canvasW) / 2.0
	px := (x + xShift) * scale
	py := pageH - (y * scale)
	return px, py
}

func ellipsePath(canvasW int, pageH, scale float64, cx, cy, rx, ry float64) draw.Path {
	const steps = 256
	p := draw.NewPath()
	for i := 0; i <= steps; i++ {
		t := (float64(i) / steps) * (2.0 * math.Pi)
		x := cx + rx*math.Cos(t)
		y := cy + ry*math.Sin(t)
		px, py := toPDF(canvasW, pageH, scale, x, y)
		p = p.AppendPoint(draw.NewPoint(px, py))
	}
	return p
}

func rectPath(canvasW int, pageH, scale float64, r rmdsl.Rect, rotDeg float64) draw.Path {
	cx := r.X + r.W/2.0
	cy := r.Y + r.H/2.0
	rot := rotDeg * math.Pi / 180.0
	cosr := math.Cos(rot)
	sinr := math.Sin(rot)
	corners := [][2]float64{
		{r.X, r.Y},
		{r.X + r.W, r.Y},
		{r.X + r.W, r.Y + r.H},
		{r.X, r.Y + r.H},
		{r.X, r.Y},
	}
	p := draw.NewPath()
	for _, pt := range corners {
		x := pt[0]
		y := pt[1]
		dx := x - cx
		dy := y - cy
		rx := cx + dx*cosr - dy*sinr
		ry := cy + dx*sinr + dy*cosr
		px, py := toPDF(canvasW, pageH, scale, rx, ry)
		p = p.AppendPoint(draw.NewPoint(px, py))
	}
	return p
}

func strokePath(canvasW int, pageH, scale float64, pts []rmdsl.Point) (draw.Path, bool) {
	if len(pts) < 2 {
		return draw.Path{}, false
	}
	p := draw.NewPath()
	for _, pt := range pts {
		px, py := toPDF(canvasW, pageH, scale, pt.X, pt.Y)
		p = p.AppendPoint(draw.NewPoint(px, py))
	}
	return p, true
}

func main() {
	in := flag.String("in", "", "Input RMDoc-DSL case file (.yaml/.yml/.js)")
	out := flag.String("out", "", "Output PDF path")
	scale := flag.Float64("scale", 0.5, "Scale factor (rm screen units to PDF points)")
	flag.Parse()

	if strings.TrimSpace(*in) == "" || strings.TrimSpace(*out) == "" {
		fmt.Fprintln(os.Stderr, "--in and --out are required")
		os.Exit(2)
	}

	ctx := context.Background()
	doc, err := rmdsl.LoadFromFile(ctx, *in, rmdsl.LoadOptions{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "load case:", err)
		os.Exit(1)
	}

	c := creator.New()

	// Render each page.
	for _, p := range doc.Document.Pages {
		pageW := float64(p.Canvas.Width) * (*scale)
		pageH := float64(p.Canvas.Height) * (*scale)
		c.SetPageSize(creator.PageSize{pageW, pageH})

		page := c.NewPage()
		if page == nil {
			fmt.Fprintln(os.Stderr, "creator.NewPage returned nil")
			os.Exit(1)
		}

		cc := contentstream.NewContentCreator()
		cc.Add_q()
		cc.Add_w(1.0)

		for _, layer := range p.Layers {
			_ = layer // preserve order
			for _, it := range layer.Items {
				switch it.Kind {
				case "stroke":
					if it.Stroke == nil {
						continue
					}
					path, ok := strokePath(p.Canvas.Width, pageH, *scale, it.Points)
					if !ok {
						continue
					}
					r, g, b := parseColor(it.Stroke.Color)
					cc.Add_RG(r, g, b)
					draw.DrawPathWithCreator(path, cc)
					cc.Add_S()

				case "shape":
					if it.Stroke == nil {
						continue
					}
					r, g, b := parseColor(it.Stroke.Color)
					cc.Add_RG(r, g, b)

					switch it.Shape {
					case "ellipse":
						if it.Center == nil {
							continue
						}
						path := ellipsePath(p.Canvas.Width, pageH, *scale, it.Center.X, it.Center.Y, it.Rx, it.Ry)
						draw.DrawPathWithCreator(path, cc)
						cc.Add_S()

					case "rect":
						if it.Rect == nil {
							continue
						}
						path := rectPath(p.Canvas.Width, pageH, *scale, *it.Rect, it.RotateDeg)
						draw.DrawPathWithCreator(path, cc)
						cc.Add_S()
					default:
						continue
					}
				default:
					continue
				}
			}
		}

		cc.Add_Q()
		page.SetContentStreams([]string{cc.Operations().String()}, core.NewFlateEncoder())
	}

	// Ensure parent dir exists (we do not mkdir here; caller should choose an existing dir).
	if _, err := os.Stat(filepath.Dir(*out)); err != nil {
		fmt.Fprintln(os.Stderr, "output directory does not exist:", filepath.Dir(*out))
		os.Exit(2)
	}

	f, err := os.Create(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrap(err, "create output"))
		os.Exit(1)
	}
	defer func() { _ = f.Close() }()

	if err := c.Write(f); err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrap(err, "write pdf"))
		os.Exit(1)
	}

	fmt.Printf("ok: wrote %s\n", *out)
}


