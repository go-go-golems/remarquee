package main

// Usage:
//   go run ./ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/14-dump-test-page1-groups-and-stroke-bboxes/main.go

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

func main() {
	root := repoRoot()
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

	// Recompute strokes+bbox just as a quick sanity check for "global screen coords".
	strokes := must(rmdoc.ExtractRMV6StrokesWithAnchors(tree))
	bbox, _ := rmdoc.BBoxForStrokes(strokes, 0)
	fmt.Printf("strokes=%d (with anchors) bbox=[%.1f %.1f %.1f %.1f]\n", len(strokes), bbox.MinX, bbox.MinY, bbox.MaxX, bbox.MaxY)

	anchorPos := must(rmdoc.BuildRMV6AnchorPos(tree.RootText))

	strokeIdx := 0
	var walk func(g *rmdoc.RMV6Group, tx, ty float64, indent int) error
	walk = func(g *rmdoc.RMV6Group, tx, ty float64, indent int) error {
		if g == nil {
			return nil
		}

		pad := strings.Repeat("  ", indent)

		ax := 0.0
		if g.AnchorOriginX != nil {
			ax = *g.AnchorOriginX
		}
		ay := 0.0
		ayOK := false
		var aid rmdoc.RMV6CrdtID
		if g.AnchorID != nil {
			aid = *g.AnchorID
			if v, ok := anchorPos[aid]; ok {
				ay = v
				ayOK = true
			}
		}

		gx := tx + ax
		gy := ty + ay

		atype := "nil"
		if g.AnchorType != nil {
			atype = fmt.Sprintf("%d", *g.AnchorType)
		}
		ath := "nil"
		if g.AnchorThreshold != nil {
			ath = fmt.Sprintf("%.3f", *g.AnchorThreshold)
		}
		aidStr := "nil"
		if g.AnchorID != nil {
			aidStr = g.AnchorID.String()
		}

		fmt.Printf(
			"%sgroup node_id=%s anchor_id=%s anchor_type=%s anchor_threshold=%s anchor_origin_x=%.1f anchor_y=%.1f(anchorPosHit=%v) tx=%.1f ty=%.1f -> gx=%.1f gy=%.1f\n",
			pad,
			g.NodeID,
			aidStr,
			atype,
			ath,
			ax,
			ay,
			ayOK,
			tx,
			ty,
			gx,
			gy,
		)

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
					return err
				}
				rawBBox, _ := rmdoc.BBoxForStroke(*st, 0)

				// Apply translation (same logic as ExtractRMV6StrokesWithAnchors).
				for i := range st.Points {
					st.Points[i].X += gx
					st.Points[i].Y += gy
				}
				anchBBox, _ := rmdoc.BBoxForStroke(*st, 0)

				w := anchBBox.MaxX - anchBBox.MinX
				h := anchBBox.MaxY - anchBBox.MinY

				fmt.Printf(
					"%s  stroke[%02d] tool=%d color=%d npts=%d raw_bbox=[%.1f %.1f %.1f %.1f] -> bbox=[%.1f %.1f %.1f %.1f] (w=%.1f h=%.1f)\n",
					pad,
					strokeIdx,
					st.Tool,
					st.Color,
					len(st.Points),
					rawBBox.MinX, rawBBox.MinY, rawBBox.MaxX, rawBBox.MaxY,
					anchBBox.MinX, anchBBox.MinY, anchBBox.MaxX, anchBBox.MaxY,
					w,
					h,
				)
				strokeIdx++

			case rmdoc.RMV6SceneItemGroup:
				if it.Value.Group == nil {
					continue
				}
				if err := walk(it.Value.Group, gx, gy, indent+1); err != nil {
					return err
				}
			default:
				continue
			}
		}

		return nil
	}

	if err := walk(tree.Root, 0, 0, 0); err != nil {
		panic(err)
	}
}
