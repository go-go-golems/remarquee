package main

// Usage:
//   go run ./ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/21-rmdoc-v6-to-dsl-yaml/main.go \
//     --in  /abs/path/to/ellipses-test.rmdoc \
//     --out /abs/path/to/ellipses-test.regen.yaml
//
// Goal:
//   Parse a V6 cPages .rmdoc, extract anchored strokes per page, and emit a RMDoc-DSL YAML
//   that replays those strokes (kind=stroke). This is the core of the “inverse” workflow:
//   device-created doc -> parse -> declarative DSL -> re-render -> compare.

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/remarquee/pkg/rmdoc"
	"github.com/go-go-golems/remarquee/pkg/rmdsl"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

func main() {
	in := flag.String("in", "", "Input .rmdoc path (V6 cPages)")
	out := flag.String("out", "", "Output YAML path")
	flag.Parse()

	if strings.TrimSpace(*in) == "" || strings.TrimSpace(*out) == "" {
		fmt.Fprintln(os.Stderr, "--in and --out are required")
		os.Exit(2)
	}

	ctx := context.Background()
	doc, err := rmdoc.OpenFile(ctx, *in)
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrap(err, "open rmdoc"))
		os.Exit(1)
	}
	if doc.Schema != rmdoc.SchemaCPages {
		fmt.Fprintln(os.Stderr, "unsupported schema (expected cPages/V6):", doc.Schema)
		os.Exit(1)
	}
	if len(doc.Pages) == 0 {
		fmt.Fprintln(os.Stderr, "no pages in document")
		os.Exit(1)
	}

	baseName := strings.TrimSuffix(filepath.Base(*in), filepath.Ext(*in))
	outDoc := rmdsl.Doc{
		RMDSL: rmdsl.DSLVersionV0,
		Document: rmdsl.Document{
			Name:   baseName + "-regen",
			Kind:   "notebook",
			Format: "v6",
			Pages:  make([]rmdsl.Page, 0, len(doc.Pages)),
		},
	}

	for i, p := range doc.Pages {
		rmFile, ok, err := rmdoc.ReadRMFileFromArchive(ctx, *in, p.PageID)
		if err != nil || !ok {
			fmt.Fprintf(os.Stderr, "missing rm file for page %d (%s)\n", i+1, p.PageID)
			os.Exit(1)
		}

		tree, err := rmdoc.ParseRMV6SceneTree(bytes.NewReader(rmFile.Bytes))
		if err != nil {
			fmt.Fprintln(os.Stderr, errors.Wrapf(err, "parse v6 scene tree (page %d)", i+1))
			os.Exit(1)
		}
		strokes, err := rmdoc.ExtractRMV6StrokesWithAnchors(tree)
		if err != nil {
			fmt.Fprintln(os.Stderr, errors.Wrapf(err, "extract strokes (page %d)", i+1))
			os.Exit(1)
		}

		page := rmdsl.Page{
			ID:       fmt.Sprintf("page-%02d", i+1),
			Template: "", // we’re replaying strokes only
			Canvas: rmdsl.Canvas{
				Space:  rmdsl.CanvasSpaceV6,
				Width:  rmdsl.DefaultWidthV6,
				Height: rmdsl.DefaultHeightV6,
			},
			Layers: []rmdsl.Layer{
				{
					Name:  "strokes",
					Items: make([]rmdsl.Item, 0, len(strokes)),
				},
			},
		}

		for _, s := range strokes {
			if len(s.Points) < 2 {
				continue
			}
			it := rmdsl.Item{
				Kind: "stroke",
				Stroke: &rmdsl.StrokeStyle{
					Tool:  fmt.Sprintf("%d", s.Tool),  // keep numeric; we can add name mapping later
					Color: fmt.Sprintf("%d", s.Color), // keep numeric; preserves exact id
					Width: 1,
				},
				Points: make([]rmdsl.Point, 0, len(s.Points)),
			}
			for _, pt := range s.Points {
				it.Points = append(it.Points, rmdsl.Point{X: pt.X, Y: pt.Y})
			}
			page.Layers[0].Items = append(page.Layers[0].Items, it)
		}

		outDoc.Document.Pages = append(outDoc.Document.Pages, page)
	}

	if err := rmdsl.Normalize(&outDoc); err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrap(err, "normalize output DSL"))
		os.Exit(1)
	}

	yb, err := yaml.Marshal(&outDoc)
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrap(err, "marshal yaml"))
		os.Exit(1)
	}

	if err := os.WriteFile(*out, yb, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrap(err, "write output yaml"))
		os.Exit(1)
	}

	fmt.Printf("ok: wrote %s (pages=%d)\n", *out, len(outDoc.Document.Pages))
}
