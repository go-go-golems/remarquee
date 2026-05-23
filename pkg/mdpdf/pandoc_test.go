package mdpdf

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func pandocAvailable(t *testing.T) {
	t.Helper()
	for _, p := range []string{"/usr/bin/pandoc", "/usr/local/bin/pandoc"} {
		if _, err := os.Stat(p); err == nil {
			return
		}
	}
	t.Skip("pandoc is not installed in a standard location")
}

func TestConvertMarkdownFileToPDFHandlesHashInInputFilename(t *testing.T) {
	pandocAvailable(t)

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

func TestConvertMarkdownFileToPDFWithImage(t *testing.T) {
	pandocAvailable(t)

	td := t.TempDir()

	// Create a minimal valid PNG (1x1 pixel, black).
	pngData := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, // PNG signature
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // 1x1
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, // 8-bit RGB
		0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41, // IDAT chunk
		0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x00, 0x02, 0x00, 0x01, 0xe2, 0x21, 0xbc,
		0x33, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, // IEND chunk
		0x44, 0xae, 0x42, 0x60, 0x82,
	}

	imgDir := filepath.Join(td, "assets")
	if err := os.MkdirAll(imgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	imgPath := filepath.Join(imgDir, "pixel.png")
	if err := os.WriteFile(imgPath, pngData, 0o644); err != nil {
		t.Fatal(err)
	}

	mdContent := "# Image Test\n\n![pixel](./assets/pixel.png)\n"
	mdPath := filepath.Join(td, "doc.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0o644); err != nil {
		t.Fatal(err)
	}

	outPDF := filepath.Join(td, "out.pdf")
	if err := ConvertMarkdownFileToPDF(context.Background(), mdPath, outPDF, DefaultPandocOptions()); err != nil {
		t.Fatalf("conversion with image failed: %v", err)
	}
	if _, err := os.Stat(outPDF); err != nil {
		t.Fatalf("expected output PDF: %v", err)
	}
}
