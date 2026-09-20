package mdpdf

import (
	"os"
	"path/filepath"
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
