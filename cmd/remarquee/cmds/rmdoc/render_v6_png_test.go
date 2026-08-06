package rmdoc

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRenderV6PNGCommand_AnnotationsOnlySmoke(t *testing.T) {
	if _, err := os.Stat("/usr/bin/pdftoppm"); err != nil {
		t.Skip("pdftoppm not available")
	}

	cmd, err := NewRenderV6PNGCommand()
	if err != nil {
		t.Fatalf("NewRenderV6PNGCommand: %v", err)
	}

	tmp := t.TempDir()

	parsedValues := newDefaultParsedValues(t, cmd.CommandDescription, map[string]interface{}{
		"file":             annotationsOnlyFixturePath(t, "fake-cpages-pdf-v6-sample-rm.rmdoc"),
		"pages":            "1,2",
		"out-dir":          tmp,
		"dpi":              100,
		"pdftoppm":         "pdftoppm",
		"annotations-only": true,
	})

	if err := cmd.Run(context.Background(), parsedValues); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// One PNG per requested page, using the -v6-annotations prefix; page 2 is blank.
	for _, name := range []string{
		"fake-cpages-pdf-v6-sample-rm-v6-annotations-page-001.png",
		"fake-cpages-pdf-v6-sample-rm-v6-annotations-page-002.png",
		"fake-cpages-pdf-v6-sample-rm-v6-annotations.pdf",
	} {
		if _, err := os.Stat(filepath.Join(tmp, name)); err != nil {
			t.Fatalf("expected output %s: %v", name, err)
		}
	}
}

// PR #21 review: render-v6-png must accept range selectors like "1-2".
func TestRenderV6PNGCommand_PagesRange(t *testing.T) {
	cmd, err := NewRenderV6PNGCommand()
	if err != nil {
		t.Fatalf("NewRenderV6PNGCommand: %v", err)
	}

	tmp := t.TempDir()

	parsedValues := newDefaultParsedValues(t, cmd.CommandDescription, map[string]interface{}{
		"file":     annotationsOnlyFixturePath(t, "fake-cpages-pdf-v6-sample-rm.rmdoc"),
		"pages":    "1-2",
		"out-dir":  tmp,
		"dpi":      50,
		"pdftoppm": "pdftoppm",
	})

	if err := cmd.Run(context.Background(), parsedValues); err != nil {
		t.Fatalf("Run: %v", err)
	}

	for _, name := range []string{
		"fake-cpages-pdf-v6-sample-rm-v6-page-001.png",
		"fake-cpages-pdf-v6-sample-rm-v6-page-002.png",
	} {
		if _, err := os.Stat(filepath.Join(tmp, name)); err != nil {
			t.Fatalf("expected output %s: %v", name, err)
		}
	}
}
