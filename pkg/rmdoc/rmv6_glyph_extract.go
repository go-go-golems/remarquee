package rmdoc

import "github.com/pkg/errors"

// ExtractRMV6GlyphRangesWithAnchors traverses the scene tree and returns all decoded glyph ranges
// in global screen coordinates, applying group anchor translations (anchor_origin_x + anchor_pos).
func ExtractRMV6GlyphRangesWithAnchors(tree *RMV6SceneTree) ([]RMV6GlyphRange, error) {
	if tree == nil {
		return nil, errors.New("tree is nil")
	}

	anchorPos, err := BuildRMV6AnchorPos(tree.RootText)
	if err != nil {
		return nil, err
	}

	var out []RMV6GlyphRange

	var walk func(g *RMV6Group, tx, ty float64) error
	walk = func(g *RMV6Group, tx, ty float64) error {
		if g == nil {
			return nil
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
			return err
		}

		for _, it := range items {
			switch it.Value.Kind {
			case RMV6SceneItemGlyph:
				if it.Value.Glyph == nil {
					continue
				}
				gr := *it.Value.Glyph
				gr.Rectangles = make([]RMV6Rectangle, 0, len(it.Value.Glyph.Rectangles))
				for _, r := range it.Value.Glyph.Rectangles {
					gr.Rectangles = append(gr.Rectangles, RMV6Rectangle{
						X: r.X + gx,
						Y: r.Y + gy,
						W: r.W,
						H: r.H,
					})
				}
				out = append(out, gr)

			case RMV6SceneItemGroup:
				if it.Value.Group == nil {
					continue
				}
				if err := walk(it.Value.Group, gx, gy); err != nil {
					return err
				}
			case RMV6SceneItemLine, RMV6SceneItemText, RMV6SceneItemTombstone, RMV6SceneItemUnknown:
				continue
			default:
				continue
			}
		}

		return nil
	}

	if err := walk(tree.Root, 0, 0); err != nil {
		return nil, err
	}
	return out, nil
}
