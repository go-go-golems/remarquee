// RMQ-0006 helper: dump stroke tool/color counts for Test.rmdoc page 1.
//
// Usage:
//   go run ./ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/11-dump-test-rmdoc-stroke-tools-page1.go
package main

import (
	"bytes"
	"context"
	"fmt"
	"sort"

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
	strokes, err := rmdoc.ExtractRMV6StrokesWithAnchors(tree)
	must(err)

	type key struct {
		Tool  uint32
		Color uint32
	}
	counts := map[key]int{}
	for _, s := range strokes {
		counts[key{Tool: s.Tool, Color: s.Color}]++
	}

	keys := make([]key, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Tool != keys[j].Tool {
			return keys[i].Tool < keys[j].Tool
		}
		return keys[i].Color < keys[j].Color
	})

	fmt.Printf("pageID=%s strokes=%d\n", doc.Pages[0].PageID, len(strokes))
	for _, k := range keys {
		fmt.Printf("tool=%d color=%d count=%d\n", k.Tool, k.Color, counts[k])
	}
}

func must(err error) {
	if err == nil {
		return
	}
	panic(fmt.Sprintf("%+v", err))
}


