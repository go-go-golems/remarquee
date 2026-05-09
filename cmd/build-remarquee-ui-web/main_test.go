package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyDirCopiesNestedFilesToRequestedOutput(t *testing.T) {
	td := t.TempDir()
	src := filepath.Join(td, "src")
	dst := filepath.Join(td, "custom-dist")
	if err := os.MkdirAll(filepath.Join(src, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "assets", "app.js"), []byte("console.log('ok')"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir failed: %v", err)
	}

	for _, rel := range []string{"index.html", filepath.Join("assets", "app.js")} {
		if _, err := os.Stat(filepath.Join(dst, rel)); err != nil {
			t.Fatalf("expected copied file %s: %v", rel, err)
		}
	}
}
