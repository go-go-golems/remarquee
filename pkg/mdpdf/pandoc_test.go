package mdpdf

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestConvertMarkdownFileToPDFHandlesHashInInputFilename(t *testing.T) {
	if _, err := os.Stat("/usr/bin/pandoc"); err != nil {
		if _, err := os.Stat("/usr/local/bin/pandoc"); err != nil {
			t.Skip("pandoc is not installed in a standard location")
		}
	}

	td := t.TempDir()
	mdPath := filepath.Join(td, "PROJ - Example PR #6.md")
	outPDF := filepath.Join(td, "out.pdf")
	if err := os.WriteFile(mdPath, []byte("# Example\n\nBody.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := ConvertMarkdownFileToPDF(context.Background(), mdPath, outPDF, DefaultPandocOptions()); err != nil {
		t.Fatalf("unexpected conversion error: %v", err)
	}
	if _, err := os.Stat(outPDF); err != nil {
		t.Fatalf("expected output PDF: %v", err)
	}
}
