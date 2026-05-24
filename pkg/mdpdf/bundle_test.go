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
	}, t.TempDir(), nil, true)
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
	}, tmpDir, nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "./images/bundle-001-photo.png") {
		t.Fatalf("expected rewritten image path, got:\n%s", out)
	}

	// Image should have been copied into tmpDir/images/.
	copiedPath := filepath.Join(tmpDir, "images", "bundle-001-photo.png")
	if _, err := os.Stat(copiedPath); err != nil {
		t.Fatalf("expected copied image at %q: %v", copiedPath, err)
	}
}

func TestBuildBundleMarkdown_AvoidsMermaidFilenameCollisionsAcrossInputs(t *testing.T) {
	td := t.TempDir()
	fakeMmdc := filepath.Join(td, "mmdc")
	if err := os.WriteFile(fakeMmdc, []byte(`#!/bin/sh
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-o" ]; then
    shift
    printf 'fake png' > "$1"
    exit 0
  fi
  shift
done
exit 1
`), 0o755); err != nil {
		t.Fatal(err)
	}

	firstMD := filepath.Join(td, "first.md")
	secondMD := filepath.Join(td, "second.md")
	mermaid := "```mermaid\ngraph TD\n  A --> B\n```\n"
	if err := os.WriteFile(firstMD, []byte(mermaid), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secondMD, []byte(mermaid), 0o644); err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	cfg := &MermaidRendererConfig{Enabled: true, MmdcPath: fakeMmdc}
	out, err := BuildBundleMarkdown(context.Background(), []BundleInput{
		{Path: firstMD, Title: "First"},
		{Path: secondMD, Title: "Second"},
	}, tmpDir, cfg, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "./images/bundle-001-mermaid-001.png") {
		t.Fatalf("expected first prefixed mermaid image path, got:\n%s", out)
	}
	if !strings.Contains(out, "./images/bundle-002-mermaid-001.png") {
		t.Fatalf("expected second prefixed mermaid image path, got:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "images", "bundle-001-mermaid-001.png")); err != nil {
		t.Fatalf("expected first mermaid image: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "images", "bundle-002-mermaid-001.png")); err != nil {
		t.Fatalf("expected second mermaid image: %v", err)
	}
}

func TestBuildBundleMarkdown_AvoidsImageBasenameCollisionsAcrossInputs(t *testing.T) {
	td := t.TempDir()

	firstDir := filepath.Join(td, "first")
	secondDir := filepath.Join(td, "second")
	if err := os.MkdirAll(filepath.Join(firstDir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(secondDir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(firstDir, "assets", "logo.png"), []byte("first"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(secondDir, "assets", "logo.png"), []byte("second"), 0o644); err != nil {
		t.Fatal(err)
	}

	firstMD := filepath.Join(firstDir, "doc.md")
	secondMD := filepath.Join(secondDir, "doc.md")
	if err := os.WriteFile(firstMD, []byte("![first](./assets/logo.png)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secondMD, []byte("![second](./assets/logo.png)\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	out, err := BuildBundleMarkdown(context.Background(), []BundleInput{
		{Path: firstMD, Title: "First"},
		{Path: secondMD, Title: "Second"},
	}, tmpDir, nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "./images/bundle-001-logo.png") {
		t.Fatalf("expected first prefixed image path, got:\n%s", out)
	}
	if !strings.Contains(out, "./images/bundle-002-logo.png") {
		t.Fatalf("expected second prefixed image path, got:\n%s", out)
	}

	firstBytes, err := os.ReadFile(filepath.Join(tmpDir, "images", "bundle-001-logo.png"))
	if err != nil {
		t.Fatal(err)
	}
	secondBytes, err := os.ReadFile(filepath.Join(tmpDir, "images", "bundle-002-logo.png"))
	if err != nil {
		t.Fatal(err)
	}
	if string(firstBytes) != "first" || string(secondBytes) != "second" {
		t.Fatalf("expected distinct copied image contents, got first=%q second=%q", string(firstBytes), string(secondBytes))
	}
}
