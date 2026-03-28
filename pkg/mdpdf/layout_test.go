package mdpdf

import (
	"strings"
	"testing"
)

func TestNormalizeMarkdownLayout(t *testing.T) {
	if got := NormalizeMarkdownLayout(""); got != MarkdownLayoutDefault {
		t.Fatalf("expected empty layout to normalize to %q, got %q", MarkdownLayoutDefault, got)
	}
	if got := NormalizeMarkdownLayout(" Editor "); got != MarkdownLayoutEditor {
		t.Fatalf("expected mixed-case editor layout to normalize to %q, got %q", MarkdownLayoutEditor, got)
	}
}

func TestApplyMarkdownLayoutPresetEditor(t *testing.T) {
	opts := DefaultPandocOptions()

	if err := ApplyMarkdownLayoutPreset(&opts, MarkdownLayoutEditor); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if opts.Geometry != editorLayoutGeometry {
		t.Fatalf("expected editor layout geometry %q, got %q", editorLayoutGeometry, opts.Geometry)
	}
	if !strings.Contains(opts.ExtraLatexHeader, "\\setstretch{1.18}") {
		t.Fatalf("expected editor header to increase line spacing, got %q", opts.ExtraLatexHeader)
	}
	if !strings.Contains(opts.ExtraLatexHeader, "\\geometry{"+editorLayoutGeometry+"}") {
		t.Fatalf("expected editor header to restate geometry, got %q", opts.ExtraLatexHeader)
	}
}

func TestApplyMarkdownLayoutPresetRejectsUnknownLayout(t *testing.T) {
	opts := DefaultPandocOptions()

	err := ApplyMarkdownLayoutPreset(&opts, "wide")
	if err == nil {
		t.Fatal("expected error for unknown layout")
	}
}
