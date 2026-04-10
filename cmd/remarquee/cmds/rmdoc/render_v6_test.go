package rmdoc

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-go-golems/remarquee/pkg/rmcloud"
)

func repoRootFromThisFile(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// This file lives at: remarquee/cmd/remarquee/cmds/rmdoc/render_v6_test.go
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "..", ".."))
}

func TestRenderV6Command_Smoke(t *testing.T) {
	cmd, err := NewRenderV6Command()
	if err != nil {
		t.Fatalf("NewRenderV6Command: %v", err)
	}

	tmp := t.TempDir()
	out := filepath.Join(tmp, "out.pdf")

	parsedValues := newDefaultParsedValues(t, cmd.CommandDescription, map[string]interface{}{
		"file":  filepath.Join(repoRootFromThisFile(t), "cmd", "remarquee-ui", "testdata", "cpage-pdf.rmdoc"),
		"out":   out,
		"force": true,
	})

	if err := cmd.Run(context.Background(), parsedValues); err != nil {
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

func TestRenderV6Command_CloudSmoke(t *testing.T) {
	oldDownloader := downloadDocumentByPath
	t.Cleanup(func() {
		downloadDocumentByPath = oldDownloader
	})

	downloadDocumentByPath = func(ctx context.Context, auth rmcloud.AuthSettings, remotePath string, outDir string) (*rmcloud.DownloadedDocument, error) {
		src := filepath.Join(repoRootFromThisFile(t), "cmd", "remarquee-ui", "testdata", "cpage-pdf.rmdoc")
		dst := filepath.Join(outDir, "CloudFixture.rmdoc")
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
			Name:       "CloudFixture",
		}, nil
	}

	cmd, err := NewRenderV6Command()
	if err != nil {
		t.Fatalf("NewRenderV6Command: %v", err)
	}

	tmp := t.TempDir()
	out := filepath.Join(tmp, "out.pdf")

	parsedValues := newDefaultParsedValues(t, cmd.CommandDescription, map[string]interface{}{
		"file":            "/Books/CloudFixture",
		"out":             out,
		"force":           true,
		"cloud":           true,
		"non-interactive": true,
	})

	if err := cmd.Run(context.Background(), parsedValues); err != nil {
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
