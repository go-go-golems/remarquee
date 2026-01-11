package compile

import (
	"math"
	"strings"

	"github.com/go-go-golems/remarquee/pkg/rmdoc"
	"github.com/go-go-golems/remarquee/pkg/rmdsl"
)

const (
	defaultStrokeWidth   = 1.0
	defaultPointSpeed    = 0
	defaultPointWidth    = 120
	defaultPointPressure = 96
)

var toolMap = map[string]uint32{
	"fineliner_2":   17,
	"highlighter_2": 18,
	"marker_2":      16,
	"ballpoint_2":   15,
	"pencil_2":      14,
}

var colorMap = map[string]rmdoc.PenColor{
	"black":           rmdoc.PenColorBlack,
	"red":             rmdoc.PenColorRed,
	"green":           rmdoc.PenColorGreen,
	"blue":            rmdoc.PenColorBlue,
	"highlight_pink":  rmdoc.PenColorHighlightPink,
	"highlight_green": rmdoc.PenColorHighlightGreen,
}

func lowerPageToLayers(p rmdsl.Page) []CompiledLayer {
	layers := make([]CompiledLayer, 0, len(p.Layers))
	for _, layer := range p.Layers {
		out := CompiledLayer{Name: layer.Name}
		for _, item := range layer.Items {
			out.Strokes = append(out.Strokes, lowerItemToStrokes(item)...)
		}
		layers = append(layers, out)
	}
	return layers
}

func lowerItemToStrokes(item rmdsl.Item) []rmdoc.Stroke {
	switch strings.ToLower(strings.TrimSpace(item.Kind)) {
	case "stroke":
		return lowerStroke(item)
	case "shape":
		return lowerShape(item)
	default:
		return nil
	}
}

func lowerStroke(item rmdsl.Item) []rmdoc.Stroke {
	if item.Stroke == nil || len(item.Points) == 0 {
		return nil
	}
	return []rmdoc.Stroke{buildStroke(*item.Stroke, item.Points)}
}

func lowerShape(item rmdsl.Item) []rmdoc.Stroke {
	if item.Stroke == nil {
		return nil
	}
	switch strings.ToLower(strings.TrimSpace(item.Shape)) {
	case "ellipse":
		if item.Center == nil {
			return nil
		}
		pts := ellipsePoints(*item.Center, item.Rx, item.Ry, 256)
		return []rmdoc.Stroke{buildStroke(*item.Stroke, pts)}
	case "rect":
		if item.Rect == nil {
			return nil
		}
		pts := rectPoints(*item.Rect, item.RotateDeg)
		return []rmdoc.Stroke{buildStroke(*item.Stroke, pts)}
	default:
		return nil
	}
}

func buildStroke(style rmdsl.StrokeStyle, pts []rmdsl.Point) rmdoc.Stroke {
	width := style.Width
	if width <= 0 {
		width = defaultStrokeWidth
	}
	tool := toolMap[strings.ToLower(strings.TrimSpace(style.Tool))]
	if tool == 0 {
		tool = toolMap["fineliner_2"]
	}
	color := colorMap[strings.ToLower(strings.TrimSpace(style.Color))]
	if color == 0 {
		color = rmdoc.PenColorBlack
	}

	out := rmdoc.Stroke{
		Tool:           tool,
		Color:          uint32(color),
		ThicknessScale: width,
		StartingLength: 0,
		Points:         make([]rmdoc.StrokePoint, 0, len(pts)),
	}

	for _, p := range pts {
		out.Points = append(out.Points, rmdoc.StrokePoint{
			X:         p.X,
			Y:         p.Y,
			Speed:     defaultPointSpeed,
			Direction: 0,
			Width:     defaultPointWidth,
			Pressure:  defaultPointPressure,
		})
	}

	return out
}

func ellipsePoints(center rmdsl.PointXY, rx, ry float64, steps int) []rmdsl.Point {
	if steps < 8 {
		steps = 8
	}
	out := make([]rmdsl.Point, 0, steps+1)
	for i := 0; i <= steps; i++ {
		t := (float64(i) / float64(steps)) * (2.0 * math.Pi)
		out = append(out, rmdsl.Point{
			X: center.X + rx*math.Cos(t),
			Y: center.Y + ry*math.Sin(t),
		})
	}
	return out
}

func rectPoints(r rmdsl.Rect, rotDeg float64) []rmdsl.Point {
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
	out := make([]rmdsl.Point, 0, len(corners))
	for _, pt := range corners {
		x := pt[0]
		y := pt[1]
		dx := x - cx
		dy := y - cy
		rx := cx + dx*cosr - dy*sinr
		ry := cy + dx*sinr + dy*cosr
		out = append(out, rmdsl.Point{X: rx, Y: ry})
	}
	return out
}
