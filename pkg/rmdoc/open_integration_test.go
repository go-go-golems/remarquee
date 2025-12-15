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
	fixture := filepath.Join(root, "..", "remarks", "tests", "in", "copies of different pages.rmdoc")

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
	if len(doc.Pages) != 4 {
		t.Fatalf("len(doc.Pages)=%d, want 4", len(doc.Pages))
	}
	// cPages should include inserted pages in the plan. For this fixture page 2+3 are inserted.
	if doc.Pages[2].SourcePDFPage != InsertedPage || doc.Pages[3].SourcePDFPage != InsertedPage {
		t.Fatalf("expected inserted pages at indices 2 and 3, got: %+v %+v", doc.Pages[2], doc.Pages[3])
	}
}

func TestOpenFile_LegacyFixture(t *testing.T) {
	root := repoRootFromThisFile(t)
	fixture := filepath.Join(root, "..", "rmapi", "archive", "test.zip")

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
	// rmapi/archive/test.zip has fileType == "" (notebook)
	if doc.Type != DocTypeNotebook {
		t.Fatalf("doc.Type=%v, want %v", doc.Type, DocTypeNotebook)
	}
	if len(doc.Pages) != 1 {
		t.Fatalf("len(doc.Pages)=%d, want 1", len(doc.Pages))
	}
	if doc.Pages[0].PageID == "" {
		t.Fatalf("doc.Pages[0].PageID is empty: %+v", doc.Pages[0])
	}
}
