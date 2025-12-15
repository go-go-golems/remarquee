package mdpdf

import (
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

	out, err := BuildBundleMarkdown([]BundleInput{
		{Path: a, Title: "Doc A"},
		{Path: b, Title: "Doc B"},
	})
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
