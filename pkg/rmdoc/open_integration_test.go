package rmdoc

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
)

func repoRootFromThisFile(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	// This file lives at: remarquee/pkg/rmdoc/open_integration_test.go
	// Repo root is: <root>/remarquee
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}

func TestOpenFile_V6Fixture(t *testing.T) {
	root := repoRootFromThisFile(t)
	fixture := filepath.Join(root, "cmd", "remarquee-ui", "testdata", "cpage-pdf.rmdoc")

	doc, err := OpenFile(context.Background(), fixture)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}

	if doc.UUID == "" {
		t.Fatalf("doc.UUID is empty")
	}
	if doc.Schema != SchemaCPages {
		t.Fatalf("doc.Schema=%v, want %v", doc.Schema, SchemaCPages)
	}
	if doc.Type != DocTypePDF {
		t.Fatalf("doc.Type=%v, want %v", doc.Type, DocTypePDF)
	}
	if len(doc.Pages) == 0 {
		t.Fatalf("expected at least one page")
	}
}

func TestOpenFile_LegacyFixture(t *testing.T) {
	root := repoRootFromThisFile(t)
	fixture := filepath.Join(root, "cmd", "remarquee-ui", "testdata", "legacy-notebook.zip")

	doc, err := OpenFile(context.Background(), fixture)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}

	if doc.UUID == "" {
		t.Fatalf("doc.UUID is empty")
	}
	if doc.Schema != SchemaLegacy {
		t.Fatalf("doc.Schema=%v, want %v", doc.Schema, SchemaLegacy)
	}
	if doc.Type != DocTypeNotebook {
		t.Fatalf("doc.Type=%v, want %v", doc.Type, DocTypeNotebook)
	}
	if len(doc.Pages) == 0 {
		t.Fatalf("expected at least one page")
	}
	if doc.Pages[0].PageID == "" {
		t.Fatalf("doc.Pages[0].PageID is empty: %+v", doc.Pages[0])
	}
}
