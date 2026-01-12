package render

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	pkg_rmdoc "github.com/go-go-golems/remarquee/pkg/rmdoc"
	rmapi_annotations "github.com/juruen/rmapi/annotations"
	pdf "github.com/unidoc/unipdf/v3/model"
)

func repoRootFromThisFileLegacy(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
}

func TestRenderLegacyGolden_Rmapi_Backend_LegacyPdfA4(t *testing.T) {
	root := repoRootFromThisFileLegacy(t)
	fixture := filepath.Join(root, "cmd", "remarquee-ui", "testdata", "legacy-pdf-a4.zip")
	if _, err := os.Stat(fixture); err != nil {
		t.Skipf("fixture not available: %s (%v)", fixture, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Use our own rmdoc parser to determine the expected UI page count.
	doc, err := pkg_rmdoc.OpenFile(ctx, fixture)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	if doc.Schema != pkg_rmdoc.SchemaLegacy {
		t.Fatalf("unexpected schema=%s", doc.Schema.String())
	}

	out := filepath.Join(t.TempDir(), "legacy-a4-annotations.pdf")
	opts := rmapi_annotations.PdfGeneratorOptions{
		AddPageNumbers:  false,
		AllPages:        true,
		AnnotationsOnly: false,
	}
	g := rmapi_annotations.CreatePdfGenerator(fixture, out, opts)
	if err := g.Generate(); err != nil {
		t.Fatalf("rmapi PdfGenerator.Generate: %v", err)
	}

	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	r, err := pdf.NewPdfReader(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("open output as pdf: %v", err)
	}
	n, err := r.GetNumPages()
	if err != nil {
		t.Fatalf("GetNumPages: %v", err)
	}

	if n != len(doc.Pages) {
		t.Fatalf("pages=%d want=%d", n, len(doc.Pages))
	}
}
