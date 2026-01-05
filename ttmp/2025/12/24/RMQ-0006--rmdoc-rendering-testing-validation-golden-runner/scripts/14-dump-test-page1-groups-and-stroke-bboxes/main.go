package main

// Usage:
//   go run ./ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/14-dump-test-page1-groups-and-stroke-bboxes/main.go

import (
	"bytes"
	"context"
	"fmt"
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
	strokes := must(rmdoc.ExtractRMV6StrokesWithAnchors(tree))
	bbox, _ := rmdoc.BBoxForStrokes(strokes, 0)

	fmt.Printf("strokes=%d bbox=[%.1f %.1f %.1f %.1f]\n", len(strokes), bbox.MinX, bbox.MinY, bbox.MaxX, bbox.MaxY)

	for _, g := range tree.Groups() {
		ax := 0.0
		ay := 0.0
		if g.AnchorOriginX != nil {
			ax = *g.AnchorOriginX
		}
		if g.AnchorID != nil {
			anchorPos, _ := rmdoc.BuildRMV6AnchorPos(tree.RootText)
			if v, ok := anchorPos[*g.AnchorID]; ok {
				ay = v
			}
		}
		children, err := g.Children.Items()
		if err != nil {
			panic(err)
		}
		fmt.Printf("group=%s anchor_x=%.1f anchor_y=%.1f children=%d\n", g.NodeID, ax, ay, len(children))
	}
}
