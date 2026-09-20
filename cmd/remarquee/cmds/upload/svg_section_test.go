package upload

import (
	"strings"
	"testing"
)

func TestUploadMarkdownHelpGroupsSVGFlags(t *testing.T) {
	cmd := NewUploadMarkdownCommand()
	groups := cmd.Annotations
	found := false
	for k, v := range groups {
		if strings.Contains(k, "glazed:flag-group:svg:SVG flags") {
			found = true
			if !strings.Contains(v, "svg-converter") || !strings.Contains(v, "svg-default-width") {
				t.Fatalf("svg flag group missing expected flags: %q", v)
			}
		}
	}
	if !found {
		t.Fatalf("expected SVG flags group annotation, got %#v", groups)
	}
}

func TestParseSVGFlagsDefaults(t *testing.T) {
	cmd := NewUploadMarkdownCommand()
	cfg, err := svgConfigFromCommand(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected default svg config")
	}
	if !cfg.Enabled {
		t.Fatal("expected svg enabled by default")
	}
	if cfg.ConverterPath != "" || cfg.DefaultWidth != "" {
		t.Fatalf("unexpected defaults: %#v", cfg)
	}
}

func TestParseSVGFlagsOverrides(t *testing.T) {
	cmd := NewUploadMarkdownCommand()
	if err := cmd.Flags().Parse([]string{"--svg-default-width", "70%", "--svg-converter", "/opt/rsvg-convert"}); err != nil {
		t.Fatalf("unexpected flag parse error: %v", err)
	}
	cfg, err := svgConfigFromCommand(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.DefaultWidth != "70%" || cfg.ConverterPath != "/opt/rsvg-convert" {
		t.Fatalf("unexpected overrides: %#v", cfg)
	}
}

func TestParseSVGFlagsDisabled(t *testing.T) {
	cmd := NewUploadMarkdownCommand()
	if err := cmd.Flags().Parse([]string{"--svg=false"}); err != nil {
		t.Fatalf("unexpected flag parse error: %v", err)
	}
	cfg, err := svgConfigFromCommand(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Fatalf("expected nil config when --svg=false, got %#v", cfg)
	}
}
