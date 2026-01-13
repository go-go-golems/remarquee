package upload

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectMarkdownFilesForBundle_Order(t *testing.T) {
	td := t.TempDir()

	// dir structure:
	// td/dir/a.md
	// td/dir/sub/b.md
	// td/x.md
	if err := os.MkdirAll(filepath.Join(td, "dir", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(td, "dir", "a.md"), []byte("# a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(td, "dir", "sub", "b.md"), []byte("# b"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(td, "x.md"), []byte("# x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Explicit file first, then directory. We should keep file first, then dir files by rel-path order.
	files, err := collectMarkdownFilesForBundle([]string{filepath.Join(td, "x.md"), filepath.Join(td, "dir")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d: %#v", len(files), files)
	}

	if filepath.Base(files[0].AbsPath) != "x.md" {
		t.Fatalf("expected first file to be x.md, got: %s", files[0].AbsPath)
	}
	if files[1].Title != "a" {
		t.Fatalf("expected second title a, got: %q", files[1].Title)
	}
	if files[2].Title != filepath.Join("sub", "b") {
		t.Fatalf("expected third title sub/b, got: %q", files[2].Title)
	}
}
