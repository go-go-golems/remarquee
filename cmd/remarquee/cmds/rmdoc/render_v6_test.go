package rmdoc

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-go-golems/remarquee/pkg/rmcloud"
	pdf "github.com/unidoc/unipdf/v3/model"
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

func TestRenderV6Command_PagesSubset(t *testing.T) {
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
		"pages": "1",
	})

	if err := cmd.Run(context.Background(), parsedValues); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got := countPDFPages(t, out); got != 1 {
		t.Fatalf("pages=%d want=1", got)
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

func countPDFPages(t *testing.T, path string) int {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	r, err := pdf.NewPdfReader(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("NewPdfReader: %v", err)
	}
	n, err := r.GetNumPages()
	if err != nil {
		t.Fatalf("GetNumPages: %v", err)
	}
	return n
}

func annotationsOnlyFixturePath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(repoRootFromThisFile(t), "cmd", "remarquee-ui", "testdata", "generated", name)
}

func TestRenderV6Command_AnnotationsOnly(t *testing.T) {
	cmd, err := NewRenderV6Command()
	if err != nil {
		t.Fatalf("NewRenderV6Command: %v", err)
	}

	tmp := t.TempDir()
	out := filepath.Join(tmp, "out.pdf")

	parsedValues := newDefaultParsedValues(t, cmd.CommandDescription, map[string]interface{}{
		"file":             annotationsOnlyFixturePath(t, "fake-cpages-pdf-v6-sample-rm.rmdoc"),
		"out":              out,
		"force":            true,
		"annotations-only": true,
	})

	if err := cmd.Run(context.Background(), parsedValues); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Fixture has 2 UI pages but only page 1 annotated: skip semantics -> 1 page.
	if got := countPDFPages(t, out); got != 1 {
		t.Fatalf("pages=%d want=1 (unannotated page should be skipped)", got)
	}
}

func TestRenderV6Command_AnnotationsOnlyPagesSubset(t *testing.T) {
	cmd, err := NewRenderV6Command()
	if err != nil {
		t.Fatalf("NewRenderV6Command: %v", err)
	}

	tmp := t.TempDir()
	out := filepath.Join(tmp, "out.pdf")

	parsedValues := newDefaultParsedValues(t, cmd.CommandDescription, map[string]interface{}{
		"file":             annotationsOnlyFixturePath(t, "fake-cpages-pdf-v6-sample-rm.rmdoc"),
		"out":              out,
		"force":            true,
		"annotations-only": true,
		"pages":            "1,2",
	})

	if err := cmd.Run(context.Background(), parsedValues); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Explicitly selected pages are emitted even when unannotated (blank page 2).
	if got := countPDFPages(t, out); got != 2 {
		t.Fatalf("pages=%d want=2 (explicitly selected pages must be emitted)", got)
	}
}

func TestRenderV6Command_AnnotationsOnlyDefaultOutName(t *testing.T) {
	cmd, err := NewRenderV6Command()
	if err != nil {
		t.Fatalf("NewRenderV6Command: %v", err)
	}

	tmp := t.TempDir()
	t.Chdir(tmp)

	parsedValues := newDefaultParsedValues(t, cmd.CommandDescription, map[string]interface{}{
		"file":             annotationsOnlyFixturePath(t, "fake-cpages-pdf-v6-sample-rm.rmdoc"),
		"annotations-only": true,
	})

	if err := cmd.Run(context.Background(), parsedValues); err != nil {
		t.Fatalf("Run: %v", err)
	}

	want := "fake-cpages-pdf-v6-sample-rm-v6-annotations.pdf"
	if _, err := os.Stat(filepath.Join(tmp, want)); err != nil {
		t.Fatalf("expected default output %s: %v", want, err)
	}
}
