package rmdoc

import (
	"math"

	"github.com/pkg/errors"
)

// BBox is an axis-aligned bounding box in the same coordinate space as its inputs.
type BBox struct {
	MinX float64
	MinY float64
	MaxX float64
	MaxY float64
}

func NewEmptyBBox() BBox {
	return BBox{
		MinX: math.Inf(1),
		MinY: math.Inf(1),
		MaxX: math.Inf(-1),
		MaxY: math.Inf(-1),
	}
}

func (b BBox) IsEmpty() bool {
	return b.MinX > b.MaxX || b.MinY > b.MaxY
}

func (b BBox) Width() float64 {
	if b.IsEmpty() {
		return 0
	}
	return b.MaxX - b.MinX
}

func (b BBox) Height() float64 {
	if b.IsEmpty() {
		return 0
	}
	return b.MaxY - b.MinY
}

func (b BBox) Expand(pad float64) BBox {
	if b.IsEmpty() {
		return b
	}
	return BBox{
		MinX: b.MinX - pad,
		MinY: b.MinY - pad,
		MaxX: b.MaxX + pad,
		MaxY: b.MaxY + pad,
	}
}

func (b BBox) Translate(dx, dy float64) BBox {
	if b.IsEmpty() {
		return b
	}
	return BBox{
		MinX: b.MinX + dx,
		MinY: b.MinY + dy,
		MaxX: b.MaxX + dx,
		MaxY: b.MaxY + dy,
	}
}

func (b BBox) Union(o BBox) BBox {
	if b.IsEmpty() {
		return o
	}
	if o.IsEmpty() {
		return b
	}
	return BBox{
		MinX: math.Min(b.MinX, o.MinX),
		MinY: math.Min(b.MinY, o.MinY),
		MaxX: math.Max(b.MaxX, o.MaxX),
		MaxY: math.Max(b.MaxY, o.MaxY),
	}
}

func BBoxFromPoints(points []StrokePoint) (BBox, bool) {
	if len(points) == 0 {
		return BBox{}, false
	}
	b := NewEmptyBBox()
	for _, p := range points {
		if p.X < b.MinX {
			b.MinX = p.X
		}
		if p.Y < b.MinY {
			b.MinY = p.Y
		}
		if p.X > b.MaxX {
			b.MaxX = p.X
		}
		if p.Y > b.MaxY {
			b.MaxY = p.Y
		}
	}
	return b, !b.IsEmpty()
}

// BBoxForStroke computes a bounding box for a single stroke.
//
// NOTE: At this stage we only consider point coordinates; brush width/thickness
// expansion is left to the caller via `pad`.
func BBoxForStroke(s Stroke, pad float64) (BBox, bool) {
	b, ok := BBoxFromPoints(s.Points)
	if !ok {
		return BBox{}, false
	}
	return b.Expand(pad), true
}

func BBoxForStrokes(strokes []Stroke, pad float64) (BBox, bool) {
	out := NewEmptyBBox()
	hasAny := false
	for _, s := range strokes {
		b, ok := BBoxForStroke(s, pad)
		if !ok {
			continue
		}
		out = out.Union(b)
		hasAny = true
	}
	if !hasAny {
		return BBox{}, false
	}
	return out, true
}

// BBoxForRMV6SceneTree computes a bbox for the scene by decoding line strokes and unioning
// their point bounds. This is a partial implementation toward RMQ-0004 task 43:
// anchor offsets for text-linked groups are not applied yet.
func BBoxForRMV6SceneTree(tree *RMV6SceneTree, pad float64) (BBox, error) {
	if tree == nil {
		return BBox{}, errors.New("tree is nil")
	}

	var strokes []Stroke
	for _, g := range tree.Groups() {
		items, err := g.Children.Items()
		if err != nil {
			return BBox{}, errors.Wrap(err, "order group children")
		}
		for _, it := range items {
			if it.Value.Kind != RMV6SceneItemLine || it.Value.Line == nil {
				continue
			}
			st, err := DecodeRMV6Line(it.Value.Line.BlockVersion, it.Value.Line.Raw)
			if err != nil {
				return BBox{}, errors.Wrap(err, "decode line stroke")
			}
			strokes = append(strokes, *st)
		}
	}

	b, ok := BBoxForStrokes(strokes, pad)
	if !ok {
		return BBox{}, nil
	}
	return b, nil
}

// BBoxForRMV6SceneTreeWithAnchors computes a bbox for the scene in global screen coordinates,
// applying group anchor offsets (anchor_origin_x + anchor_pos[anchor_id]).
//
// This is the anchor-aware implementation needed for RMQ-0004 task 43.
func BBoxForRMV6SceneTreeWithAnchors(tree *RMV6SceneTree, pad float64) (BBox, error) {
	if tree == nil {
		return BBox{}, errors.New("tree is nil")
	}

	anchorPos, err := BuildRMV6AnchorPos(tree.RootText)
	if err != nil {
		return BBox{}, err
	}

	var walk func(g *RMV6Group, tx, ty float64) (BBox, bool, error)
	walk = func(g *RMV6Group, tx, ty float64) (BBox, bool, error) {
		if g == nil {
			return BBox{}, false, nil
		}

		ax := 0.0
		if g.AnchorOriginX != nil {
			ax = *g.AnchorOriginX
		}
		ay := 0.0
		if g.AnchorID != nil {
			if v, ok := anchorPos[*g.AnchorID]; ok {
				ay = v
			}
		}

		gx := tx + ax
		gy := ty + ay

		items, err := g.Children.Items()
		if err != nil {
			return BBox{}, false, err
		}

		out := NewEmptyBBox()
		hasAny := false

		for _, it := range items {
			switch it.Value.Kind {
			case RMV6SceneItemLine:
				if it.Value.Line == nil {
					continue
				}
				st, err := DecodeRMV6Line(it.Value.Line.BlockVersion, it.Value.Line.Raw)
				if err != nil {
					return BBox{}, false, errors.Wrap(err, "decode line stroke")
				}
				b, ok := BBoxForStroke(*st, pad)
				if !ok {
					continue
				}
				out = out.Union(b.Translate(gx, gy))
				hasAny = true

			case RMV6SceneItemGroup:
				if it.Value.Group == nil {
					continue
				}
				b, ok, err := walk(it.Value.Group, gx, gy)
				if err != nil {
					return BBox{}, false, err
				}
				if ok {
					out = out.Union(b)
					hasAny = true
				}
			case RMV6SceneItemGlyph, RMV6SceneItemText, RMV6SceneItemTombstone, RMV6SceneItemUnknown:
				continue
			default:
				continue
			}
		}

		if !hasAny {
			return BBox{}, false, nil
		}
		return out, true, nil
	}

	b, ok, err := walk(tree.Root, 0, 0)
	if err != nil {
		return BBox{}, err
	}
	if !ok {
		return BBox{}, nil
	}
	return b, nil
}
