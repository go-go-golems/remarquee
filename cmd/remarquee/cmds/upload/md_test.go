package upload

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectMarkdownFiles_FileAndDirectory(t *testing.T) {
	td := t.TempDir()

	// dir structure:
	// td/a.md
	// td/sub/b.md
	// td/sub/c.txt
	if err := os.WriteFile(filepath.Join(td, "a.md"), []byte("# a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(td, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(td, "sub", "b.md"), []byte("# b"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(td, "sub", "c.txt"), []byte("no"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := collectMarkdownFiles([]string{td, filepath.Join(td, "a.md")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %#v", len(files), files)
	}
}

func TestResolveRemoteDir_DateFormats(t *testing.T) {
	got, err := resolveRemoteDir("2025/12/15", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/ai/2025/12/15" {
		t.Fatalf("unexpected: %q", got)
	}

	got2, err := resolveRemoteDir("2025-12-15", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got2 != "/ai/2025/12/15" {
		t.Fatalf("unexpected: %q", got2)
	}
}

func TestResolveRemoteDir_OverrideNormalizes(t *testing.T) {
	got, err := resolveRemoteDir("", "ai/2025/12/15/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/ai/2025/12/15" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestJoinRemoteDir(t *testing.T) {
	base := "/ai/test"

	if got := joinRemoteDir(base, ""); got != "/ai/test" {
		t.Fatalf("unexpected: %q", got)
	}
	if got := joinRemoteDir(base, "."); got != "/ai/test" {
		t.Fatalf("unexpected: %q", got)
	}
	if got := joinRemoteDir(base, "sub"); got != "/ai/test/sub" {
		t.Fatalf("unexpected: %q", got)
	}
	if got := joinRemoteDir(base, "/sub/dir/"); got != "/ai/test/sub/dir" {
		t.Fatalf("unexpected: %q", got)
	}
}
