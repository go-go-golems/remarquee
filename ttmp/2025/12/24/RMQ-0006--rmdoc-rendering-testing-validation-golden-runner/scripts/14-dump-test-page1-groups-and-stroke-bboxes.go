// RMQ-0006 helper: dump group anchor info and per-stroke bboxes for Test.rmdoc page 1.
//
// This is meant to debug "a specific shape is way off" issues by showing:
// - which group a stroke lives under
// - what anchor translation was applied
// - the resulting bbox in screen coordinates
//
// Usage:
//   go run ./ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/14-dump-test-page1-groups-and-stroke-bboxes.go
package main

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/go-go-golems/remarquee/pkg/rmdoc"
	"github.com/pkg/errors"
)

func main() {
	ctx := context.Background()
	path := "/home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee-ui/testdata/Test.rmdoc"

	doc, err := rmdoc.OpenFile(ctx, path)
	must(err)
	if len(doc.Pages) < 1 || doc.Pages[0].PageID == "" {
		must(errors.New("Test.rmdoc has no page 0/PageID"))
	}

	rm, ok, err := rmdoc.ReadRMFileFromArchive(ctx, path, doc.Pages[0].PageID)
	must(err)
	if !ok || rm == nil || rm.Version != "V6" {
		must(errors.Errorf("missing V6 rm for page 0 (pageID=%s)", doc.Pages[0].PageID))
	}

	tree, err := rmdoc.ParseRMV6SceneTree(bytes.NewReader(rm.Bytes))
	must(err)

	anchorPos, err := rmdoc.BuildRMV6AnchorPos(tree.RootText)
	must(err)

	fmt.Printf("pageID=%s\n", doc.Pages[0].PageID)
	fmt.Printf("rootTextPresent=%v anchors=%d\n", tree.RootText != nil, len(anchorPos))
	if tree.RootText != nil {
		fmt.Printf("rootText posX=%0.3f posY=%0.3f width=%0.3f\n", tree.RootText.PosX, tree.RootText.PosY, tree.RootText.Width)
	}

	type strokeInfo struct {
		path string
		tool uint32
		col  uint32
		bbox rmdoc.BBox
		npts int
	}

	var out []strokeInfo

	var walk func(g *rmdoc.RMV6Group, tx, ty float64, pathParts []string) error
	walk = func(g *rmdoc.RMV6Group, tx, ty float64, pathParts []string) error {
		if g == nil {
			return nil
		}

		ax := 0.0
		if g.AnchorOriginX != nil {
			ax = *g.AnchorOriginX
		}
		ay := 0.0
		aid := "none"
		if g.AnchorID != nil {
			aid = fmt.Sprintf("(%d,%d)", g.AnchorID.Part1, g.AnchorID.Part2)
			if v, ok := anchorPos[*g.AnchorID]; ok {
				ay = v
			}
		}

		gx := tx + ax
		gy := ty + ay

		label := fmt.Sprintf("group(%d,%d) aid=%s ax=%0.1f ay=%0.1f tx=%0.1f ty=%0.1f",
			g.NodeID.Part1, g.NodeID.Part2, aid, ax, ay, gx, gy)

		items, err := g.Children.Items()
		if err != nil {
			return err
		}
		for _, it := range items {
			switch it.Value.Kind {
			case rmdoc.RMV6SceneItemLine:
				if it.Value.Line == nil {
					continue
				}
				st, err := rmdoc.DecodeRMV6Line(it.Value.Line.BlockVersion, it.Value.Line.Raw)
				if err != nil {
					return errors.Wrap(err, "decode line")
				}
				for i := range st.Points {
					st.Points[i].X += gx
					st.Points[i].Y += gy
				}
				bb := bboxForPoints(st.Points)
				out = append(out, strokeInfo{
					path: strings.Join(append(pathParts, label, "line"), " / "),
					tool: st.Tool,
					col:  st.Color,
					bbox: bb,
					npts: len(st.Points),
				})
			case rmdoc.RMV6SceneItemGroup:
				if it.Value.Group == nil {
					continue
				}
				if err := walk(it.Value.Group, gx, gy, append(pathParts, label)); err != nil {
					return err
				}
			default:
				continue
			}
		}
		return nil
	}

	if err := walk(tree.Root, 0, 0, nil); err != nil {
		must(err)
	}

	sort.Slice(out, func(i, j int) bool {
		// Sort by bbox top-left to make it easier to scan.
		if out[i].bbox.MinY != out[j].bbox.MinY {
			return out[i].bbox.MinY < out[j].bbox.MinY
		}
		if out[i].bbox.MinX != out[j].bbox.MinX {
			return out[i].bbox.MinX < out[j].bbox.MinX
		}
		return out[i].npts < out[j].npts
	})

	fmt.Printf("strokes=%d\n", len(out))
	for idx, s := range out {
		fmt.Printf("[%02d] tool=%d color=%d pts=%d bbox=(%0.1f,%0.1f)-(%0.1f,%0.1f) %s\n",
			idx, s.tool, s.col, s.npts, s.bbox.MinX, s.bbox.MinY, s.bbox.MaxX, s.bbox.MaxY, s.path)
	}
}

func bboxForPoints(pts []rmdoc.StrokePoint) rmdoc.BBox {
	b := rmdoc.NewEmptyBBox()
	for _, p := range pts {
		b = b.Union(rmdoc.BBox{MinX: p.X, MinY: p.Y, MaxX: p.X, MaxY: p.Y})
	}
	if b.IsEmpty() {
		return b
	}
	// Normalize NaNs just in case.
	if math.IsNaN(b.MinX) || math.IsNaN(b.MinY) || math.IsNaN(b.MaxX) || math.IsNaN(b.MaxY) {
		return rmdoc.NewEmptyBBox()
	}
	return b
}

func must(err error) {
	if err == nil {
		return
	}
	panic(fmt.Sprintf("%+v", err))
}


