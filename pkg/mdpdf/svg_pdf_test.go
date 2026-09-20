package mdpdf

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testSVG = `<svg xmlns="http://www.w3.org/2000/svg" width="120" height="60">` +
	`<rect width="120" height="60" fill="#336699"/></svg>`

// TestResolveInlineSVGBlocks_RealConverter exercises the extraction and
// conversion path with the real rsvg-convert binary and verifies the output is
// a real PDF.
func TestResolveInlineSVGBlocks_RealConverter(t *testing.T) {
	if _, err := exec.LookPath("rsvg-convert"); err != nil {
		t.Skip("optional dependency absent: rsvg-convert")
	}
	cfg := DefaultSVGRendererConfig()
	tmpDir := t.TempDir()
	body := "# Doc\n\n" + testSVG + "\n\nafter\n"

	out, err := ResolveInlineSVGBlocks(context.Background(), body, tmpDir, &cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "./images/svg-001.pdf") {
		t.Fatalf("expected substitution, got: %q", out)
	}
	pdf, err := os.ReadFile(filepath.Join(tmpDir, "images", "svg-001.pdf"))
	if err != nil {
		t.Fatalf("expected converted pdf: %v", err)
	}
	if !strings.HasPrefix(string(pdf), "%PDF-") {
		t.Fatalf("converted output is not a PDF: %q", string(pdf[:min(8, len(pdf))]))
	}
}

// TestSVGPipelinePDF renders a document containing referenced, inline and HTML
// SVG through the full pandoc/XeLaTeX pipeline.
func TestSVGPipelinePDF(t *testing.T) {
	for _, tool := range []string{"pandoc", "xelatex", "rsvg-convert"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("optional PDF integration dependency absent: %s", tool)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "logo.svg"), []byte(testSVG), 0o644); err != nil {
		t.Fatal(err)
	}
	source := strings.Join([]string{
		"# SVG pipeline",
		"",
		"Referenced:",
		"",
		"![ref](./logo.svg)",
		"",
		"Inline:",
		"",
		testSVG,
		"",
		"HTML:",
		"",
		`<img src="./logo.svg" width="100">`,
		"",
		"Fenced (must stay code):",
		"",
		"```xml",
		testSVG,
		"```",
		"",
	}, "\n")
	input := filepath.Join(dir, "input.md")
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "out.pdf")
	if err := ConvertMarkdownFileToPDF(ctx, input, output, DefaultPandocOptions()); err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	pdf, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(pdf), "%PDF-") {
		t.Fatal("conversion did not produce a PDF")
	}
}
