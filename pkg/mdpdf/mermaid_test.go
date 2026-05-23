package mdpdf

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderMermaidBlocks_NilConfig(t *testing.T) {
	body := "```mermaid\ngraph TD\n  A --> B\n```\n"
	result, err := RenderMermaidBlocks(context.Background(), body, t.TempDir(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != body {
		t.Fatalf("nil config should leave body unchanged, got: %q", result)
	}
}

func TestRenderMermaidBlocks_Disabled(t *testing.T) {
	cfg := &MermaidRendererConfig{Enabled: false}
	body := "```mermaid\ngraph TD\n  A --> B\n```\n"
	result, err := RenderMermaidBlocks(context.Background(), body, t.TempDir(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != body {
		t.Fatalf("disabled config should leave body unchanged, got: %q", result)
	}
}

func TestRenderMermaidBlocks_MmdcNotFound(t *testing.T) {
	cfg := &MermaidRendererConfig{
		Enabled: true,
		MmdcPath: "/nonexistent/mmdc",
	}
	body := "```mermaid\ngraph TD\n  A --> B\n```\n"
	result, err := RenderMermaidBlocks(context.Background(), body, t.TempDir(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should leave body unchanged when mmdc is not found.
	if result != body {
		t.Fatalf("mmdc not found should leave body unchanged, got: %q", result)
	}
}

func TestRenderMermaidBlocks_NoMermaidBlocks(t *testing.T) {
	cfg := &MermaidRendererConfig{Enabled: true}
	body := "# Title\n\nJust text.\n"
	result, err := RenderMermaidBlocks(context.Background(), body, t.TempDir(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != body {
		t.Fatalf("body without mermaid should be unchanged, got: %q", result)
	}
}

func TestRenderMermaidBlocks_EmptyBlock(t *testing.T) {
	cfg := &MermaidRendererConfig{Enabled: true}
	body := "```mermaid\n```\n"
	result, err := RenderMermaidBlocks(context.Background(), body, t.TempDir(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty block should remain unchanged.
	if result != body {
		t.Fatalf("empty mermaid block should be unchanged, got: %q", result)
	}
}

func TestResolveMmdcPath_NotFound(t *testing.T) {
	_, err := resolveMmdcPath("")
	if err == nil {
		t.Fatal("expected error when mmdc is not in $PATH")
	}
	if !strings.Contains(err.Error(), "mmdc") {
		t.Fatalf("error should mention mmdc, got: %v", err)
	}
}

func TestResolveMmdcPath_InvalidOverride(t *testing.T) {
	_, err := resolveMmdcPath("/nonexistent/path/mmdc")
	if err == nil {
		t.Fatal("expected error for nonexistent override path")
	}
}

func TestResolveMmdcPath_ValidOverride(t *testing.T) {
	tmpDir := t.TempDir()
	fakeMmdc := filepath.Join(tmpDir, "mmdc")
	if err := os.WriteFile(fakeMmdc, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	path, err := resolveMmdcPath(fakeMmdc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != fakeMmdc {
		t.Fatalf("expected %q, got %q", fakeMmdc, path)
	}
}

func TestDefaultMermaidRendererConfig(t *testing.T) {
	cfg := DefaultMermaidRendererConfig()
	if !cfg.Enabled {
		t.Error("default config should have Enabled=true")
	}
	if cfg.Scale != 2 {
		t.Errorf("default Scale should be 2, got %d", cfg.Scale)
	}
	if cfg.BackgroundColor != "white" {
		t.Errorf("default BackgroundColor should be white, got %q", cfg.BackgroundColor)
	}
	if cfg.Theme != "default" {
		t.Errorf("default Theme should be default, got %q", cfg.Theme)
	}
	if cfg.Width != 0 {
		t.Errorf("default Width should be 0 (auto), got %d", cfg.Width)
	}
}

func TestMermaidBlockRegex_Simple(t *testing.T) {
	body := "before\n```mermaid\ngraph TD\n  A --> B\n```\nafter"
	matches := mermaidBlockRegex.FindAllStringSubmatch(body, -1)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	source := strings.TrimSpace(matches[0][1])
	if source != "graph TD\n  A --> B" {
		t.Fatalf("unexpected mermaid source: %q", source)
	}
}

func TestMermaidBlockRegex_Multiple(t *testing.T) {
	body := "```mermaid\ngraph TD\n  A --> B\n```\n\ntext\n\n```mermaid\nsequenceDiagram\n  A->>B: hi\n```\n"
	matches := mermaidBlockRegex.FindAllStringSubmatch(body, -1)
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}
}

func TestMermaidBlockRegex_NoMatchForOtherLangs(t *testing.T) {
	body := "```go\nfmt.Println(\"hi\")\n```\n"
	if mermaidBlockRegex.MatchString(body) {
		t.Fatal("should not match non-mermaid code blocks")
	}
}
