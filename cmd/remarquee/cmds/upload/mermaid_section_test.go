package upload

import (
	"strings"
	"testing"
)

func TestUploadMarkdownHelpGroupsMermaidFlags(t *testing.T) {
	cmd := NewUploadMarkdownCommand()
	help := cmd.HelpTemplate()
	if help == "" {
		t.Fatal("expected help template")
	}

	// The actual grouping is driven by Glazed flag-group annotations.
	groups := cmd.Annotations
	if groups["glazed:flag-group-count"] == "" {
		t.Fatalf("expected glazed flag group annotations, got %#v", groups)
	}
	found := false
	for k, v := range groups {
		if strings.Contains(k, "glazed:flag-group:mermaid:Mermaid flags") {
			found = true
			if !strings.Contains(v, "mermaid-pdf-width") || !strings.Contains(v, "mermaid-no-sandbox") {
				t.Fatalf("mermaid flag group missing expected flags: %q", v)
			}
		}
	}
	if !found {
		t.Fatalf("expected Mermaid flags group annotation, got %#v", groups)
	}
}

func TestParseMermaidFlagsDefaults(t *testing.T) {
	cmd := NewUploadMarkdownCommand()
	cfg, err := mermaidConfigFromCommand(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected default mermaid config")
	}
	if !cfg.NoSandbox {
		t.Fatal("expected no-sandbox default to true")
	}
	if cfg.Scale != 2 {
		t.Fatalf("expected scale 2, got %d", cfg.Scale)
	}
}

func TestParseMermaidFlagsOverrides(t *testing.T) {
	cmd := NewUploadMarkdownCommand()
	if err := cmd.Flags().Parse([]string{"--mermaid-pdf-width", "50%", "--mermaid-scale", "3", "--mermaid=false"}); err != nil {
		t.Fatalf("unexpected flag parse error: %v", err)
	}
	cfg, err := mermaidConfigFromCommand(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Fatalf("expected nil config when --mermaid=false, got %#v", cfg)
	}
}
