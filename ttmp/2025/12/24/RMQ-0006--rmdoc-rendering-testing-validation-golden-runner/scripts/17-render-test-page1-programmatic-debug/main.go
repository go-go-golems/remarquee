package main

// Usage:
//   go run ./ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/17-render-test-page1-programmatic-debug/main.go
//
// Output:
//   Writes PNG(s) into:
//     remarquee/rendering/rmq-0006-ellipse/
//
// Goal:
//   Create "programmatic" debug images directly from parsed V6 strokes (no PDF renderer),
//   so we can reason about coordinate transforms quickly.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"github.com/go-go-golems/remarquee/pkg/rmdoc"
)

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func repoRoot() string {
	wd := must(os.Getwd())
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			panic("could not find go.mod above cwd")
		}
		wd = parent
	}
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func drawPoint(img *image.RGBA, x, y int, c color.Color) {
	b := img.Bounds()
	if x < b.Min.X || x >= b.Max.X || y < b.Min.Y || y >= b.Max.Y {
		return
	}
	img.Set(x, y, c)
}

// Bresenham line.
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

func drawRect(img *image.RGBA, x0, y0, x1, y1 int, c color.Color) {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	x0 = clampInt(x0, 0, img.Bounds().Dx()-1)
	x1 = clampInt(x1, 0, img.Bounds().Dx()-1)
	y0 = clampInt(y0, 0, img.Bounds().Dy()-1)
	y1 = clampInt(y1, 0, img.Bounds().Dy()-1)
	drawLine(img, x0, y0, x1, y0, c)
	drawLine(img, x1, y0, x1, y1, c)
	drawLine(img, x1, y1, x0, y1, c)
	drawLine(img, x0, y1, x0, y0, c)
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

type strokeBBoxDump struct {
	Index int        `json:"index"`
	Tool  uint32     `json:"tool"`
	Color uint32     `json:"color"`
	NPts  int        `json:"npts"`
	BBox  rmdoc.BBox `json:"bbox"`
}

func savePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return png.Encode(f, img)
}

func main() {
	root := repoRoot()

	// Inputs.
	fixture := filepath.Join(root, "cmd", "remarquee-ui", "testdata", "Test.rmdoc")
	ctx := context.Background()
	doc := must(rmdoc.OpenFile(ctx, fixture))
	if len(doc.Pages) < 1 {
		panic("no pages")
	}

	rm, ok, err := rmdoc.ReadRMFileFromArchive(ctx, fixture, doc.Pages[0].PageID)
	if err != nil || !ok {
		panic("missing rm file for page 1")
	}

	tree := must(rmdoc.ParseRMV6SceneTree(bytes.NewReader(rm.Bytes)))
	strokes := must(rmdoc.ExtractRMV6StrokesWithAnchors(tree))
	if len(strokes) == 0 {
		panic("no strokes")
	}

	// Output dir: stable in-repo location (already used elsewhere in this session).
	outDir := filepath.Join(root, "rendering", "rmq-0006-ellipse")
	// (No mkdir here per user preference; assume it exists.)

	// Canvas is "RM screen pixels". X is centered ([-W/2, +W/2]), Y is top-down ([0..H]).
	// Keep in sync with `pkg/rmdoc/render/v6_strokes_pdf.go`.
	w := 1404
	h := 1872
	scale := 1.0
	xShift := float64(w) / 2.0

	toPx := func(x, y float64) (int, int) {
		px := int(math.Round(x*scale + xShift))
		py := int(math.Round(y * scale))
		return px, py
	}

	// Colors.
	bg := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	grid := color.RGBA{R: 245, G: 245, B: 245, A: 255}
	axis := color.RGBA{R: 220, G: 220, B: 220, A: 255}
	bboxC := color.RGBA{R: 40, G: 90, B: 220, A: 255}
	shapeBBoxC := color.RGBA{R: 220, G: 60, B: 60, A: 255}

	// 1) Full strokes + bboxes + grid.
	imgAll := image.NewRGBA(image.Rect(0, 0, w, h))
	fillBackground(imgAll, bg)
	drawGrid(imgAll, 72, grid) // ~1 inch-ish on 200dpi; just a convenient step
	// axes (x=0 in RM coords, y=0 at top)
	drawLine(imgAll, int(xShift), 0, int(xShift), h-1, axis)
	drawLine(imgAll, 0, 0, w-1, 0, axis)

	dumps := make([]strokeBBoxDump, 0, len(strokes))

	for i, s := range strokes {
		if len(s.Points) == 0 {
			continue
		}
		// stroke color
		r, g, b := rmdoc.PenColorToRGBForStroke(rmdoc.PenColor(s.Color))
		stC := color.RGBA{
			R: uint8(clampInt(int(r*255.0), 0, 255)),
			G: uint8(clampInt(int(g*255.0), 0, 255)),
			B: uint8(clampInt(int(b*255.0), 0, 255)),
			A: 255,
		}

		// polyline
		prevX, prevY := toPx(s.Points[0].X, s.Points[0].Y)
		for _, p := range s.Points[1:] {
			x, y := toPx(p.X, p.Y)
			drawLine(imgAll, prevX, prevY, x, y, stC)
			prevX, prevY = x, y
		}

		// bbox (screen coords)
		bb, ok := rmdoc.BBoxForStroke(s, 0)
		if ok {
			x0, y0 := toPx(bb.MinX, bb.MinY)
			x1, y1 := toPx(bb.MaxX, bb.MaxY)
			drawRect(imgAll, x0, y0, x1, y1, bboxC)
			dumps = append(dumps, strokeBBoxDump{
				Index: i,
				Tool:  s.Tool,
				Color: s.Color,
				NPts:  len(s.Points),
				BBox:  bb,
			})
		}
	}

	outAll := filepath.Join(outDir, "Test-rmdoc-page1-programmatic-strokes+bbox.png")
	fmt.Printf("write %s\n", outAll)
	must(struct{}{}, savePNG(outAll, imgAll))

	// 2) Shapes-only (fineliner_2) bboxes emphasized.
	// In rmscene, tool=17 is FINELINER_2 (often used for "shape tool" strokes).
	imgShapes := image.NewRGBA(image.Rect(0, 0, w, h))
	fillBackground(imgShapes, bg)
	drawGrid(imgShapes, 72, grid)
	drawLine(imgShapes, int(xShift), 0, int(xShift), h-1, axis)
	drawLine(imgShapes, 0, 0, w-1, 0, axis)

	for _, s := range strokes {
		if s.Tool != 17 || len(s.Points) == 0 {
			continue
		}
		// draw stroke in black for maximum contrast
		prevX, prevY := toPx(s.Points[0].X, s.Points[0].Y)
		for _, p := range s.Points[1:] {
			x, y := toPx(p.X, p.Y)
			drawLine(imgShapes, prevX, prevY, x, y, color.Black)
			prevX, prevY = x, y
		}
		if bb, ok := rmdoc.BBoxForStroke(s, 0); ok {
			x0, y0 := toPx(bb.MinX, bb.MinY)
			x1, y1 := toPx(bb.MaxX, bb.MaxY)
			drawRect(imgShapes, x0, y0, x1, y1, shapeBBoxC)
		}
	}

	outShapes := filepath.Join(outDir, "Test-rmdoc-page1-programmatic-shapes-only.png")
	fmt.Printf("write %s\n", outShapes)
	must(struct{}{}, savePNG(outShapes, imgShapes))

	// 3) Dump bbox JSON (so we can refer to specific strokes without needing text labels).
	outJSON := filepath.Join(outDir, "Test-rmdoc-page1-stroke-bboxes.json")
	fmt.Printf("write %s\n", outJSON)
	jb, _ := json.MarshalIndent(dumps, "", "  ")
	must(struct{}{}, os.WriteFile(outJSON, jb, 0o644))
}


