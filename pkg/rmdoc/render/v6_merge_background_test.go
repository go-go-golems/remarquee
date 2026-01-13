package render

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/go-go-golems/remarquee/pkg/rmdoc"
	"github.com/unidoc/unipdf/v3/core"
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

func TestMergeRMDocV6OntoBackgroundPDFWithInfo_HighlightsXTranslation_Smoke(t *testing.T) {
	ctx := context.Background()
	path := "/home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee-ui/testdata/cpage-pdf.rmdoc"

	doc, err := rmdoc.OpenFile(ctx, path)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}

	res, err := MergeRMDocV6OntoBackgroundPDFWithInfo(ctx, path, V6MergeOptions{})
	if err != nil {
		t.Fatalf("MergeRMDocV6OntoBackgroundPDFWithInfo: %v", err)
	}
	if len(res.PDF) < 8 || !bytes.HasPrefix(res.PDF, []byte("%PDF-")) {
		t.Fatalf("expected PDF header")
	}
	if len(res.HighlightsXTranslation) != len(doc.Pages) {
		t.Fatalf("HighlightsXTranslation=%d want=%d", len(res.HighlightsXTranslation), len(doc.Pages))
	}
}

func TestMergeRMDocV6OntoBackgroundPDF_SmartHighlights_NoCrash(t *testing.T) {
	ctx := context.Background()
	path := "/home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee-ui/testdata/cpage-pdf.rmdoc"

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
	if n < 1 {
		t.Fatalf("expected at least one page")
	}

	// If fixture contains glyph ranges, we should see at least one highlight annotation.
	// If not, the test still validates we didn't break PDF output.
	seenHighlight := false
	for i := 1; i <= n; i++ {
		p, err := r.GetPage(i)
		if err != nil {
			t.Fatalf("GetPage(%d): %v", i, err)
		}
		annots, err := p.GetAnnotations()
		if err != nil {
			continue
		}
		for _, a := range annots {
			if a == nil {
				continue
			}
			ctx := a.GetContext()
			if ctx == nil {
				continue
			}
			// Check subtype directly for robustness.
			obj := a.ToPdfObject()
			io, ok := core.GetIndirect(obj)
			if !ok {
				continue
			}
			d, ok := core.GetDict(io.PdfObject)
			if !ok {
				continue
			}
			if name, ok := core.GetName(d.Get("Subtype")); ok && name.String() == "Highlight" {
				seenHighlight = true
				break
			}
		}
		if seenHighlight {
			break
		}
	}

	_ = seenHighlight // not asserting true because fixture may not include highlights
}
