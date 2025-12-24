package render

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/go-go-golems/remarquee/pkg/rmdoc"
	pdf "github.com/unidoc/unipdf/v3/model"
)

func TestMergeRMDocV6OntoBackgroundPDF_Smoke(t *testing.T) {
	ctx := context.Background()
	path := "/home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee-ui/testdata/cpage-pdf.rmdoc"

	doc, err := rmdoc.OpenFile(ctx, path)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}

	out, err := MergeRMDocV6OntoBackgroundPDF(ctx, path, V6MergeOptions{})
	if err != nil {
		t.Fatalf("MergeRMDocV6OntoBackgroundPDF: %v", err)
	}

	r, err := pdf.NewPdfReader(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("open merged pdf: %v", err)
	}
	n, err := r.GetNumPages()
	if err != nil {
		t.Fatalf("GetNumPages: %v", err)
	}
	if n != len(doc.Pages) {
		t.Fatalf("pages=%d want=%d", n, len(doc.Pages))
	}

	// Assert at least one page had overlay applied.
	seenOverlay := false
	for i := 1; i <= n; i++ {
		p, err := r.GetPage(i)
		if err != nil {
			t.Fatalf("GetPage(%d): %v", i, err)
		}
		cs, _ := p.GetAllContentStreams()
		if strings.Contains(cs, "rmv6-overlay") {
			seenOverlay = true
			break
		}
	}
	if !seenOverlay {
		t.Fatalf("expected at least one page to contain overlay marker")
	}
}
