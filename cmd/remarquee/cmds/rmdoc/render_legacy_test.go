package rmdoc

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/remarquee/pkg/rmcloud"
)

func TestRenderLegacyCommand_Smoke(t *testing.T) {
	cmd, err := NewRenderLegacyCommand()
	if err != nil {
		t.Fatalf("NewRenderLegacyCommand: %v", err)
	}

	tmp := t.TempDir()
	out := filepath.Join(tmp, "out.pdf")

	defaultLayer, err := layers.NewParameterLayer(layers.DefaultSlug, "Default")
	if err != nil {
		t.Fatalf("NewParameterLayer: %v", err)
	}

	pp := parameters.NewParsedParameters()
	pp.Set("file", &parameters.ParsedParameter{Value: filepath.Join(repoRootFromThisFile(t), "cmd", "remarquee-ui", "testdata", "legacy-pdf-a4.zip")})
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

func TestRenderLegacyCommand_CloudSmoke(t *testing.T) {
	oldDownloader := downloadDocumentByPath
	t.Cleanup(func() {
		downloadDocumentByPath = oldDownloader
	})

	downloadDocumentByPath = func(ctx context.Context, auth rmcloud.AuthSettings, remotePath string, outDir string) (*rmcloud.DownloadedDocument, error) {
		src := filepath.Join(repoRootFromThisFile(t), "cmd", "remarquee-ui", "testdata", "legacy-pdf-a4.zip")
		dst := filepath.Join(outDir, "CloudLegacy.rmdoc")
		in, err := os.Open(src)
		if err != nil {
			return nil, err
		}
		defer func() { _ = in.Close() }()
		out, err := os.Create(dst)
		if err != nil {
			return nil, err
		}
		defer func() { _ = out.Close() }()
		if _, err := io.Copy(out, in); err != nil {
			return nil, err
		}
		return &rmcloud.DownloadedDocument{
			RemotePath: remotePath,
			LocalPath:  dst,
			Name:       "CloudLegacy",
		}, nil
	}

	cmd, err := NewRenderLegacyCommand()
	if err != nil {
		t.Fatalf("NewRenderLegacyCommand: %v", err)
	}

	tmp := t.TempDir()
	out := filepath.Join(tmp, "out.pdf")

	defaultLayer, err := layers.NewParameterLayer(layers.DefaultSlug, "Default")
	if err != nil {
		t.Fatalf("NewParameterLayer: %v", err)
	}

	pp := parameters.NewParsedParameters()
	pp.Set("file", &parameters.ParsedParameter{Value: "/Archive/CloudLegacy"})
	pp.Set("out", &parameters.ParsedParameter{Value: out})
	pp.Set("force", &parameters.ParsedParameter{Value: true})
	pp.Set("cloud", &parameters.ParsedParameter{Value: true})
	pp.Set("non-interactive", &parameters.ParsedParameter{Value: true})

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
