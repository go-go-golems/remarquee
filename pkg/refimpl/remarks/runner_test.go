package remarks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRemarksPDFs(t *testing.T) {
	dir := t.TempDir()

	// Non-matching file.
	if err := os.WriteFile(filepath.Join(dir, "foo.pdf"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Matching file (top-level).
	p1 := filepath.Join(dir, "Doc _remarks.pdf")
	if err := os.WriteFile(p1, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Matching file (nested).
	nested := filepath.Join(dir, "Some", "Path")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	p2 := filepath.Join(nested, "Other _remarks.pdf")
	if err := os.WriteFile(p2, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	pdfs, err := FindRemarksPDFs(dir)
	if err != nil {
		t.Fatalf("FindRemarksPDFs: %v", err)
	}
	if len(pdfs) != 2 {
		t.Fatalf("expected 2 pdfs, got %d: %v", len(pdfs), pdfs)
	}
}


