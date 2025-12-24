package rmdoc

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
)

func TestRenderV6Command_Smoke(t *testing.T) {
	cmd, err := NewRenderV6Command()
	if err != nil {
		t.Fatalf("NewRenderV6Command: %v", err)
	}

	tmp := t.TempDir()
	out := filepath.Join(tmp, "out.pdf")

	defaultLayer, err := layers.NewParameterLayer(layers.DefaultSlug, "Default")
	if err != nil {
		t.Fatalf("NewParameterLayer: %v", err)
	}

	pp := parameters.NewParsedParameters()
	pp.Set("file", &parameters.ParsedParameter{Value: "/home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee-ui/testdata/cpage-pdf.rmdoc"})
	pp.Set("out", &parameters.ParsedParameter{Value: out})
	pp.Set("force", &parameters.ParsedParameter{Value: true})

	parsedLayers := layers.NewParsedLayers(layers.WithParsedLayer(layers.DefaultSlug, &layers.ParsedLayer{
		Layer:      defaultLayer,
		Parameters: pp,
	}))

	if err := cmd.Run(context.Background(), parsedLayers); err != nil {
		t.Fatalf("Run: %v", err)
	}

	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(b) < 8 || string(b[:5]) != "%PDF-" {
		t.Fatalf("expected PDF output, got prefix %q", string(b[:8]))
	}
}
