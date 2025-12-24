package rmdoc

import "github.com/pkg/errors"

// ExtractRMV6StrokesWithAnchors traverses the scene tree and returns all decoded line strokes
// in global screen coordinates, applying group anchor translations (anchor_origin_x + anchor_pos).
func ExtractRMV6StrokesWithAnchors(tree *RMV6SceneTree) ([]Stroke, error) {
	if tree == nil {
		return nil, errors.New("tree is nil")
	}

	anchorPos, err := BuildRMV6AnchorPos(tree.RootText)
	if err != nil {
		return nil, err
	}

	var out []Stroke

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
			case RMV6SceneItemLine:
				if it.Value.Line == nil {
					continue
				}
				st, err := DecodeRMV6Line(it.Value.Line.BlockVersion, it.Value.Line.Raw)
				if err != nil {
					return errors.Wrap(err, "decode line")
				}
				for i := range st.Points {
					st.Points[i].X += gx
					st.Points[i].Y += gy
				}
				out = append(out, *st)
			case RMV6SceneItemGroup:
				if it.Value.Group == nil {
					continue
				}
				if err := walk(it.Value.Group, gx, gy); err != nil {
					return err
				}
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
