package main

// Usage:
//   go run ./ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/11-dump-test-rmdoc-stroke-tools-page1/main.go

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

	toolCounts := map[uint32]int{}
	colorCounts := map[uint32]int{}
	for _, s := range strokes {
		toolCounts[s.Tool]++
		colorCounts[s.Color]++
	}

	fmt.Println("toolCounts:")
	for k, v := range toolCounts {
		fmt.Printf("  tool=%d count=%d\n", k, v)
	}
	fmt.Println("colorCounts:")
	for k, v := range colorCounts {
		fmt.Printf("  color=%d count=%d\n", k, v)
	}
}
