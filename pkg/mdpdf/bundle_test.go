package mdpdf

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildBundleMarkdown_StripsFrontmatterAndAddsHeadings(t *testing.T) {
	td := t.TempDir()

	a := filepath.Join(td, "a.md")
	b := filepath.Join(td, "b.md")

	if err := os.WriteFile(a, []byte("---\nTitle: A\n---\n# Inner\n\nHello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("# B\n\nWorld\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := BuildBundleMarkdown(context.Background(), []BundleInput{
		{Path: a, Title: "Doc A"},
		{Path: b, Title: "Doc B"},
	}, t.TempDir(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(out, "Title: A") {
		t.Fatalf("expected YAML frontmatter stripped, got:\n%s", out)
	}
	if !strings.Contains(out, "# Doc A\n") || !strings.Contains(out, "# Doc B\n") {
		t.Fatalf("expected stable section headings, got:\n%s", out)
	}
	if !strings.Contains(out, "\\newpage") {
		t.Fatalf("expected pagebreak between docs, got:\n%s", out)
	}
}

func TestBuildBundleMarkdown_ResolvesImages(t *testing.T) {
	td := t.TempDir()
	imgDir := filepath.Join(td, "assets")
	if err := os.MkdirAll(imgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(imgDir, "photo.png"), []byte("png-data"), 0o644); err != nil {
		t.Fatal(err)
	}

	md := filepath.Join(td, "doc.md")
	if err := os.WriteFile(md, []byte("# Doc\n\n![photo](./assets/photo.png)\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	out, err := BuildBundleMarkdown(context.Background(), []BundleInput{
		{Path: md, Title: "Doc"},
	}, tmpDir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "./images/photo.png") {
		t.Fatalf("expected rewritten image path, got:\n%s", out)
	}

	// Image should have been copied into tmpDir/images/.
	copiedPath := filepath.Join(tmpDir, "images", "photo.png")
	if _, err := os.Stat(copiedPath); err != nil {
		t.Fatalf("expected copied image at %q: %v", copiedPath, err)
	}
}
