package main

// Usage:
//   go run ./ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/18-rmdsl-render-to-png/main.go \
//     --in  ./ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/cases/01-ellipse-at-bottom.yaml \
//     --out /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/rendering/rmq-0006-dsl
//
// Goal:
//   Parse the RMDoc-DSL YAML and generate programmatic PNG debug images (no PDF renderer).

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/remarquee/pkg/rmdsl"
)

func drawPoint(img *image.RGBA, x, y int, c color.Color) {
	b := img.Bounds()
	if x < b.Min.X || x >= b.Max.X || y < b.Min.Y || y >= b.Max.Y {
		return
	}
	img.Set(x, y, c)
}

func drawLine(img *image.RGBA, x0, y0, x1, y1 int, c color.Color) {
	dx := int(math.Abs(float64(x1 - x0)))
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	dy := -int(math.Abs(float64(y1 - y0)))
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		drawPoint(img, x0, y0, c)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func fillBackground(img *image.RGBA, c color.Color) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			img.Set(x, y, c)
		}
	}
}

func drawGrid(img *image.RGBA, step int, c color.Color) {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	for x := 0; x < w; x += step {
		drawLine(img, x, 0, x, h-1, c)
	}
	for y := 0; y < h; y += step {
		drawLine(img, 0, y, w-1, y, c)
	}
}

func parseColor(name string) color.Color {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "black":
		return color.Black
	case "red":
		return color.RGBA{R: 220, G: 50, B: 50, A: 255}
	case "green":
		return color.RGBA{R: 40, G: 170, B: 60, A: 255}
	default:
		return color.RGBA{R: 30, G: 30, B: 30, A: 255}
	}
}

func toPxRMV6(canvasW int, x, y float64) (int, int) {
	// rm_screen_v6: x in [-W/2,+W/2], y in [0..H]
	xShift := float64(canvasW) / 2.0
	px := int(math.Round(x + xShift))
	py := int(math.Round(y))
	return px, py
}

func savePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return png.Encode(f, img)
}

func renderPage(doc rmdsl.Doc, page rmdsl.Page, outDir string) error {
	img := image.NewRGBA(image.Rect(0, 0, page.Canvas.Width, page.Canvas.Height))
	fillBackground(img, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	drawGrid(img, 72, color.RGBA{R: 245, G: 245, B: 245, A: 255})
	drawLine(img, page.Canvas.Width/2, 0, page.Canvas.Width/2, page.Canvas.Height-1, color.RGBA{R: 220, G: 220, B: 220, A: 255})

	for _, layer := range page.Layers {
		_ = layer // future: z-order controls
		for _, it := range layer.Items {
			switch it.Kind {
			case "stroke":
				if it.Stroke == nil || len(it.Points) < 2 {
					continue
				}
				c := parseColor(it.Stroke.Color)
				x0, y0 := toPxRMV6(page.Canvas.Width, it.Points[0].X, it.Points[0].Y)
				for _, p := range it.Points[1:] {
					x1, y1 := toPxRMV6(page.Canvas.Width, p.X, p.Y)
					drawLine(img, x0, y0, x1, y1, c)
					x0, y0 = x1, y1
				}

			case "shape":
				if it.Stroke == nil {
					continue
				}
				c := parseColor(it.Stroke.Color)
				switch it.Shape {
				case "ellipse":
					if it.Center == nil {
						continue
					}
					// very simple ellipse raster: sample angle steps
					const steps = 256
					var prevX, prevY int
					for i := 0; i <= steps; i++ {
						t := (float64(i) / steps) * (2.0 * math.Pi)
						x := it.Center.X + it.Rx*math.Cos(t)
						y := it.Center.Y + it.Ry*math.Sin(t)
						px, py := toPxRMV6(page.Canvas.Width, x, y)
						if i > 0 {
							drawLine(img, prevX, prevY, px, py, c)
						}
						prevX, prevY = px, py
					}

				case "rect":
					if it.Rect == nil {
						continue
					}
					// handle optional rotation around rect center
					cx := it.Rect.X + it.Rect.W/2.0
					cy := it.Rect.Y + it.Rect.H/2.0
					rot := it.RotateDeg * math.Pi / 180.0
					cosr := math.Cos(rot)
					sinr := math.Sin(rot)
					corners := [][2]float64{
						{it.Rect.X, it.Rect.Y},
						{it.Rect.X + it.Rect.W, it.Rect.Y},
						{it.Rect.X + it.Rect.W, it.Rect.Y + it.Rect.H},
						{it.Rect.X, it.Rect.Y + it.Rect.H},
						{it.Rect.X, it.Rect.Y},
					}
					var prevX, prevY int
					for i, pt := range corners {
						x := pt[0]
						y := pt[1]
						// rotate about center
						dx := x - cx
						dy := y - cy
						rx := cx + dx*cosr - dy*sinr
						ry := cy + dx*sinr + dy*cosr
						px, py := toPxRMV6(page.Canvas.Width, rx, ry)
						if i > 0 {
							drawLine(img, prevX, prevY, px, py, c)
						}
						prevX, prevY = px, py
					}

				default:
					continue
				}

			default:
				continue
			}
		}
	}

	base := fmt.Sprintf("%s-%s", doc.Document.Name, page.ID)
	base = strings.Trim(base, "-")
	if base == "" {
		base = "rmdsl"
	}
	outPath := filepath.Join(outDir, base+".png")
	return savePNG(outPath, img)
}

func main() {
	in := flag.String("in", "", "Input YAML DSL file (absolute or relative to repo root)")
	out := flag.String("out", "", "Output directory (must exist)")
	flag.Parse()

	if strings.TrimSpace(*in) == "" {
		fmt.Fprintln(os.Stderr, "--in is required")
		os.Exit(2)
	}
	if strings.TrimSpace(*out) == "" {
		fmt.Fprintln(os.Stderr, "--out is required")
		os.Exit(2)
	}
	if st, err := os.Stat(*out); err != nil || !st.IsDir() {
		fmt.Fprintln(os.Stderr, "output directory must exist:", *out)
		os.Exit(2)
	}

	ctx := context.Background()
	docPtr, err := rmdsl.LoadFromFile(ctx, *in, rmdsl.LoadOptions{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "load case:", err)
		os.Exit(1)
	}
	doc := *docPtr

	for i, p := range doc.Document.Pages {
		if p.ID == "" {
			p.ID = fmt.Sprintf("page-%d", i+1)
		}
		if err := renderPage(doc, p, *out); err != nil {
			fmt.Fprintln(os.Stderr, "render:", err)
			os.Exit(1)
		}
	}

	_ = errors.New // keep placeholder for future strict validation
	fmt.Printf("ok: rendered %d page(s) to %s\n", len(doc.Document.Pages), *out)
}
