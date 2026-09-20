package mdpdf

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeExecutable(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("failed to write fake executable %q: %v", path, err)
	}
	return path
}

func TestResolveSVGConverter_NilConfig(t *testing.T) {
	got, err := ResolveSVGConverter(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty converter for nil config, got %q", got)
	}
}

func TestResolveSVGConverter_Disabled(t *testing.T) {
	cfg := DefaultSVGRendererConfig()
	cfg.Enabled = false
	got, err := ResolveSVGConverter(&cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty converter when disabled, got %q", got)
	}
}

func TestResolveSVGConverter_ExplicitPathExists(t *testing.T) {
	dir := t.TempDir()
	bin := writeExecutable(t, dir, "my-svg-converter")

	cfg := DefaultSVGRendererConfig()
	cfg.ConverterPath = bin
	got, err := ResolveSVGConverter(&cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != bin {
		t.Fatalf("expected %q, got %q", bin, got)
	}
}

func TestResolveSVGConverter_ExplicitPathMissing(t *testing.T) {
	cfg := DefaultSVGRendererConfig()
	cfg.ConverterPath = filepath.Join(t.TempDir(), "does-not-exist")
	if _, err := ResolveSVGConverter(&cfg); err == nil {
		t.Fatal("expected error for missing explicit converter path")
	}
}

func TestResolveSVGConverter_PathOrder(t *testing.T) {
	dir := t.TempDir()
	writeExecutable(t, dir, "rsvg-convert")
	t.Setenv("PATH", dir)

	cfg := DefaultSVGRendererConfig()
	got, err := ResolveSVGConverter(&cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(dir, "rsvg-convert")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestResolveSVGConverter_PrefersFirstInOrder(t *testing.T) {
	dir := t.TempDir()
	writeExecutable(t, dir, "rsvg-convert")
	writeExecutable(t, dir, "inkscape")
	t.Setenv("PATH", dir)

	cfg := DefaultSVGRendererConfig()
	got, err := ResolveSVGConverter(&cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Base(got) != "rsvg-convert" {
		t.Fatalf("expected rsvg-convert to win, got %q", got)
	}
}

func TestResolveSVGConverter_FallsBackToInkscape(t *testing.T) {
	dir := t.TempDir()
	writeExecutable(t, dir, "inkscape")
	t.Setenv("PATH", dir)

	cfg := DefaultSVGRendererConfig()
	got, err := ResolveSVGConverter(&cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Base(got) != "inkscape" {
		t.Fatalf("expected inkscape fallback, got %q", got)
	}
}

func TestResolveSVGConverter_NoneFoundIsNotError(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	cfg := DefaultSVGRendererConfig()
	got, err := ResolveSVGConverter(&cfg)
	if err != nil {
		t.Fatalf("expected no error when no converter found, got %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty converter, got %q", got)
	}
}

func TestSVGConfigWithImagePrefix(t *testing.T) {
	if got := svgConfigWithImagePrefix(nil, "bundle-001-"); got != nil {
		t.Fatalf("expected nil for nil config, got %#v", got)
	}

	cfg := DefaultSVGRendererConfig()
	cfg.ImagePrefix = "extra-"
	got := svgConfigWithImagePrefix(&cfg, "bundle-001-")
	if got == nil {
		t.Fatal("expected non-nil cloned config")
	}
	if got.ImagePrefix != "bundle-001-extra-" {
		t.Fatalf("unexpected prefix: %q", got.ImagePrefix)
	}
	// Original must be untouched.
	if cfg.ImagePrefix != "extra-" {
		t.Fatalf("original config was mutated: %q", cfg.ImagePrefix)
	}
}

func TestSVGRendererConfigWarnf(t *testing.T) {
	var buf testWriter
	cfg := DefaultSVGRendererConfig()
	cfg.WarnWriter = &buf
	cfg.warnf("could not convert %d blocks", 2)
	if got := buf.String(); got != "WARNING: SVG: could not convert 2 blocks\n" {
		t.Fatalf("unexpected warning: %q", got)
	}
}

type testWriter struct{ b []byte }

func (w *testWriter) Write(p []byte) (int, error) { w.b = append(w.b, p...); return len(p), nil }
func (w *testWriter) String() string              { return string(w.b) }

// writeFakeConverter writes a shell converter that reads a `-o OUT` argument and
// writes a dummy PDF there. It supports the rsvg-convert/inkscape argv shapes.
func writeFakeConverter(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	script := `#!/bin/sh
out=""
while [ $# -gt 0 ]; do
  if [ "$1" = "-o" ]; then out="$2"; shift 2; continue; fi
  in="$1"; shift
done
[ -n "$out" ] && printf 'PDF' > "$out"
exit 0
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write fake converter: %v", err)
	}
	return path
}

func writeFailingConverter(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho boom >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("failed to write failing converter: %v", err)
	}
	return path
}

func TestLooksLikeSVGOpen(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"<svg xmlns=\"...\">", true},
		{"  <svg>", true},
		{"<SVG viewBox=\"0 0 10 10\">", true},
		{"<svg/>", true},
		{"<svgfoo>", false},
		{"text <svg>", false},
		{"no svg here", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := looksLikeSVGOpen(tt.line); got != tt.want {
			t.Errorf("looksLikeSVGOpen(%q) = %v, want %v", tt.line, got, tt.want)
		}
	}
}

func TestCountTagDelta(t *testing.T) {
	tests := []struct {
		line string
		want int
	}{
		{"<svg>", 1},
		{"</svg>", -1},
		{"<svg/>", 0},
		{"<svg x=\"1\"/>", 0},
		{"<svg data-x=\">\">", 1},
		{"<svg data-x=\"</svg>\">", 1},
		{"<svg></svg>", 0},
		{"nope", 0},
		{"<svgfoo>", 0},
	}
	for _, tt := range tests {
		if got := countTagDelta(tt.line, "svg"); got != tt.want {
			t.Errorf("countTagDelta(%q) = %d, want %d", tt.line, got, tt.want)
		}
	}
}

func newInlineTestConfig(t *testing.T, tmpDir string) (SVGRendererConfig, string) {
	t.Helper()
	dir := t.TempDir()
	conv := writeFakeConverter(t, dir, "rsvg-convert")
	cfg := DefaultSVGRendererConfig()
	cfg.ConverterPath = conv
	cfg.WarnWriter = &testWriter{}
	return cfg, tmpDir
}

func TestResolveInlineSVGBlocks_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	cfg, _ := newInlineTestConfig(t, tmpDir)
	body := "# Title\n\n<svg xmlns=\"http://www.w3.org/2000/svg\"><rect/></svg>\n\nafter\n"

	out, err := ResolveInlineSVGBlocks(context.Background(), body, tmpDir, &cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "<svg") {
		t.Fatalf("expected <svg> to be replaced, got: %q", out)
	}
	if !strings.Contains(out, "![svg image 1](./images/svg-001.pdf)") {
		t.Fatalf("expected markdown image substitution, got: %q", out)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "images", "svg-001.pdf")); err != nil {
		t.Fatalf("expected converted pdf: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "images", "svg-001.svg")); err != nil {
		t.Fatalf("expected extracted svg: %v", err)
	}
}

func TestResolveInlineSVGBlocks_Multiline(t *testing.T) {
	tmpDir := t.TempDir()
	cfg, _ := newInlineTestConfig(t, tmpDir)
	body := "<svg xmlns=\"http://www.w3.org/2000/svg\">\n  <circle r=\"5\"/>\n  <rect/>\n</svg>\n"

	out, err := ResolveInlineSVGBlocks(context.Background(), body, tmpDir, &cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "![svg image 1](./images/svg-001.pdf)") {
		t.Fatalf("expected substitution, got: %q", out)
	}
}

func TestResolveInlineSVGBlocks_NestedAndCount(t *testing.T) {
	tmpDir := t.TempDir()
	cfg, _ := newInlineTestConfig(t, tmpDir)
	body := "<svg>\n<svg>\n</svg>\n</svg>\n\n<svg/>\n"

	out, err := ResolveInlineSVGBlocks(context.Background(), body, tmpDir, &cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "svg-001.pdf") || !strings.Contains(out, "svg-002.pdf") {
		t.Fatalf("expected two substitutions, got: %q", out)
	}
}

func TestResolveInlineSVGBlocks_SkipsFencedCode(t *testing.T) {
	tmpDir := t.TempDir()
	cfg, _ := newInlineTestConfig(t, tmpDir)
	body := "```xml\n<svg xmlns=\"http://www.w3.org/2000/svg\"><rect/></svg>\n```\n"

	out, err := ResolveInlineSVGBlocks(context.Background(), body, tmpDir, &cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != body {
		t.Fatalf("fenced svg must be untouched, got: %q", out)
	}
}

func TestResolveInlineSVGBlocks_UnclosedLeavesAsIs(t *testing.T) {
	tmpDir := t.TempDir()
	var buf testWriter
	cfg, _ := newInlineTestConfig(t, tmpDir)
	cfg.WarnWriter = &buf
	body := "<svg>\n<rect/>\n"

	out, err := ResolveInlineSVGBlocks(context.Background(), body, tmpDir, &cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != body {
		t.Fatalf("unclosed svg must be untouched, got: %q", out)
	}
	if !strings.Contains(buf.String(), "unclosed") {
		t.Fatalf("expected unclosed warning, got: %q", buf.String())
	}
}

func TestResolveInlineSVGBlocks_NoConverter(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	tmpDir := t.TempDir()
	var buf testWriter
	cfg := DefaultSVGRendererConfig()
	cfg.WarnWriter = &buf
	body := "<svg><rect/></svg>\n"

	out, err := ResolveInlineSVGBlocks(context.Background(), body, tmpDir, &cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != body {
		t.Fatalf("expected unchanged body without converter, got: %q", out)
	}
	if !strings.Contains(buf.String(), "no SVG converter") {
		t.Fatalf("expected no-converter warning, got: %q", buf.String())
	}
}

func TestResolveInlineSVGBlocks_ConversionFailureLeavesAsIs(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultSVGRendererConfig()
	cfg.ConverterPath = writeFailingConverter(t, dir, "rsvg-convert")
	var buf testWriter
	cfg.WarnWriter = &buf
	tmpDir := t.TempDir()
	body := "<svg><rect/></svg>\n"

	out, err := ResolveInlineSVGBlocks(context.Background(), body, tmpDir, &cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != body {
		t.Fatalf("expected unchanged body on failure, got: %q", out)
	}
	if !strings.Contains(buf.String(), "failed to convert") {
		t.Fatalf("expected conversion warning, got: %q", buf.String())
	}
}

func TestResolveInlineSVGBlocks_DefaultWidthAndPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	cfg, _ := newInlineTestConfig(t, tmpDir)
	cfg.DefaultWidth = "70%"
	cfg.ImagePrefix = "bundle-002-"
	body := "<svg><rect/></svg>\n"

	out, err := ResolveInlineSVGBlocks(context.Background(), body, tmpDir, &cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "![svg image 1](./images/bundle-002-svg-001.pdf){width=70%}"
	if !strings.Contains(out, want) {
		t.Fatalf("expected %q, got: %q", want, out)
	}
}
