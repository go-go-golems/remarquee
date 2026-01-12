package render_test

import (
	"bytes"
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-go-golems/remarquee/pkg/rmdoc"
	"github.com/go-go-golems/remarquee/pkg/rmdoc/render"
	pdf "github.com/unidoc/unipdf/v3/model"
)

func repoRootFromThisFile(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	// This file lives at: remarquee/pkg/rmdoc/render/background_test.go
	// Repo root is: <root>/remarquee
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
}

func TestBuildBackgroundPDF_CPagesFixture_PageCountAndReadable(t *testing.T) {
	ctx := context.Background()

	root := repoRootFromThisFile(t)
	fixture := filepath.Join(root, "..", "remarks", "tests", "in", "copies of different pages.rmdoc")

	doc, err := rmdoc.OpenFile(ctx, fixture)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	if doc == nil {
		t.Fatalf("doc is nil")
	}
	if len(doc.PayloadPDF) == 0 {
		t.Fatalf("expected fixture to be PDF-backed")
	}
	if len(doc.Pages) == 0 {
		t.Fatalf("expected fixture to contain pages")
	}

	bg, err := render.BuildBackgroundPDF(ctx, doc, render.BackgroundOptions{})
	if err != nil {
		t.Fatalf("BuildBackgroundPDF: %v", err)
	}
	if len(bg) == 0 {
		t.Fatalf("background PDF is empty")
	}

	r, err := pdf.NewPdfReader(bytes.NewReader(bg))
	if err != nil {
		t.Fatalf("NewPdfReader(bg): %v", err)
	}
	n, err := r.GetNumPages()
	if err != nil {
		t.Fatalf("GetNumPages(bg): %v", err)
	}

	if n != len(doc.Pages) {
		t.Fatalf("page count mismatch: got %d, want %d", n, len(doc.Pages))
	}
}
