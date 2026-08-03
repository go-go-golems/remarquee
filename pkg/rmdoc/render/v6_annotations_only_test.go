package render

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	pdf "github.com/unidoc/unipdf/v3/model"
)

func annotationsOnlyFixture(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(repoRootInternal(t), "cmd", "remarquee-ui", "testdata", "generated", name)
}

func openPDFBytes(t *testing.T, b []byte) *pdf.PdfReader {
	t.Helper()
	if len(b) < 8 || !bytes.HasPrefix(b, []byte("%PDF-")) {
		t.Fatalf("expected PDF header, got %q", string(b[:min(8, len(b))]))
	}
	r, err := pdf.NewPdfReader(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("NewPdfReader: %v", err)
	}
	return r
}

func pageCount(t *testing.T, r *pdf.PdfReader) int {
	t.Helper()
	n, err := r.GetNumPages()
	if err != nil {
		t.Fatalf("GetNumPages: %v", err)
	}
	return n
}

func pageContent(t *testing.T, r *pdf.PdfReader, pageNum int) string {
	t.Helper()
	p, err := r.GetPage(pageNum)
	if err != nil {
		t.Fatalf("GetPage(%d): %v", pageNum, err)
	}
	cs, err := p.GetAllContentStreams()
	if err != nil {
		t.Fatalf("GetAllContentStreams(%d): %v", pageNum, err)
	}
	return cs
}

// fake-cpages-pdf-v6-sample-rm.rmdoc: 2 UI pages, but only page 1 (p0) has a V6 .rm file.
func TestRenderV6AnnotationsOnly_SkipsUnannotatedPages(t *testing.T) {
	ctx := context.Background()
	path := annotationsOnlyFixture(t, "fake-cpages-pdf-v6-sample-rm.rmdoc")

	res, err := RenderRMDocV6AnnotationsOnlyWithInfo(ctx, path, V6MergeOptions{}, []int{0, 1}, false)
	if err != nil {
		t.Fatalf("RenderRMDocV6AnnotationsOnlyWithInfo: %v", err)
	}

	r := openPDFBytes(t, res.PDF)
	if n := pageCount(t, r); n != 1 {
		t.Fatalf("pages=%d want=1 (unannotated page should be skipped)", n)
	}
	if cs := pageContent(t, r, 1); !strings.Contains(cs, "rmv6-overlay") {
		t.Fatalf("expected overlay marker on emitted page")
	}
	if len(res.HighlightsXTranslation) != 1 {
		t.Fatalf("HighlightsXTranslation=%d want=1", len(res.HighlightsXTranslation))
	}
}

func TestRenderV6AnnotationsOnly_EmitsBlanksWhenIncluded(t *testing.T) {
	ctx := context.Background()
	path := annotationsOnlyFixture(t, "fake-cpages-pdf-v6-sample-rm.rmdoc")

	res, err := RenderRMDocV6AnnotationsOnlyWithInfo(ctx, path, V6MergeOptions{}, []int{0, 1}, true)
	if err != nil {
		t.Fatalf("RenderRMDocV6AnnotationsOnlyWithInfo: %v", err)
	}

	r := openPDFBytes(t, res.PDF)
	if n := pageCount(t, r); n != 2 {
		t.Fatalf("pages=%d want=2 (blank page should be emitted for unannotated selection)", n)
	}
	if cs := pageContent(t, r, 1); !strings.Contains(cs, "rmv6-overlay") {
		t.Fatalf("expected overlay marker on page 1")
	}
	if cs := strings.TrimSpace(pageContent(t, r, 2)); cs != "" {
		t.Fatalf("expected blank page 2, got content stream %q", cs)
	}
	if len(res.HighlightsXTranslation) != 2 {
		t.Fatalf("HighlightsXTranslation=%d want=2 (one entry per emitted page)", len(res.HighlightsXTranslation))
	}
	if res.HighlightsXTranslation[1] != 0 {
		t.Fatalf("blank page translation=%v want=0", res.HighlightsXTranslation[1])
	}
}

// fake-cpages-pdf-no-annotations.rmdoc: 4 UI pages, no .rm files at all.
func TestRenderV6AnnotationsOnly_NoAnnotationsAtAll(t *testing.T) {
	ctx := context.Background()
	path := annotationsOnlyFixture(t, "fake-cpages-pdf-no-annotations.rmdoc")

	res, err := RenderRMDocV6AnnotationsOnlyWithInfo(ctx, path, V6MergeOptions{}, []int{0, 1, 2, 3}, false)
	if err != nil {
		t.Fatalf("RenderRMDocV6AnnotationsOnlyWithInfo: %v", err)
	}
	r := openPDFBytes(t, res.PDF)
	if n := pageCount(t, r); n != 0 {
		t.Fatalf("pages=%d want=0 (no annotated pages in fixture)", n)
	}

	res, err = RenderRMDocV6AnnotationsOnlyWithInfo(ctx, path, V6MergeOptions{}, []int{0, 1, 2, 3}, true)
	if err != nil {
		t.Fatalf("RenderRMDocV6AnnotationsOnlyWithInfo: %v", err)
	}
	r = openPDFBytes(t, res.PDF)
	if n := pageCount(t, r); n != 4 {
		t.Fatalf("pages=%d want=4 (all blank)", n)
	}
	for i := 1; i <= 4; i++ {
		if cs := strings.TrimSpace(pageContent(t, r, i)); cs != "" {
			t.Fatalf("expected blank page %d, got content stream %q", i, cs)
		}
	}
}

// fake-cpages-pdf-v6-empty-rm.rmdoc: 2 UI pages, one header-only (empty) V6 .rm file.
func TestRenderV6AnnotationsOnly_EmptyRMFile(t *testing.T) {
	ctx := context.Background()
	path := annotationsOnlyFixture(t, "fake-cpages-pdf-v6-empty-rm.rmdoc")

	res, err := RenderRMDocV6AnnotationsOnlyWithInfo(ctx, path, V6MergeOptions{}, []int{0, 1}, false)
	if err != nil {
		t.Fatalf("RenderRMDocV6AnnotationsOnlyWithInfo: %v", err)
	}
	r := openPDFBytes(t, res.PDF)
	if n := pageCount(t, r); n != 0 {
		t.Fatalf("pages=%d want=0 (header-only .rm carries no renderable content)", n)
	}

	res, err = RenderRMDocV6AnnotationsOnlyWithInfo(ctx, path, V6MergeOptions{}, []int{0, 1}, true)
	if err != nil {
		t.Fatalf("RenderRMDocV6AnnotationsOnlyWithInfo: %v", err)
	}
	r = openPDFBytes(t, res.PDF)
	if n := pageCount(t, r); n != 2 {
		t.Fatalf("pages=%d want=2 (all blank)", n)
	}
}

func TestRenderV6AnnotationsOnly_PageIndexValidation(t *testing.T) {
	ctx := context.Background()
	path := annotationsOnlyFixture(t, "fake-cpages-pdf-v6-sample-rm.rmdoc")

	if _, err := RenderRMDocV6AnnotationsOnlyWithInfo(ctx, path, V6MergeOptions{}, []int{2}, false); err == nil ||
		!strings.Contains(err.Error(), "pageIndices") {
		t.Fatalf("expected out-of-range pageIndices error, got %v", err)
	}

	if _, err := RenderRMDocV6AnnotationsOnlyWithInfo(ctx, path, V6MergeOptions{}, nil, false); err == nil ||
		!strings.Contains(err.Error(), "pageIndices is empty") {
		t.Fatalf("expected empty pageIndices error, got %v", err)
	}
}

// The annotations-only path must never composite the background form XObject.
func TestRenderV6AnnotationsOnly_NoBackgroundXObject(t *testing.T) {
	ctx := context.Background()
	path := annotationsOnlyFixture(t, "fake-cpages-pdf-v6-sample-rm.rmdoc")

	res, err := RenderRMDocV6AnnotationsOnlyWithInfo(ctx, path, V6MergeOptions{}, []int{0}, false)
	if err != nil {
		t.Fatalf("RenderRMDocV6AnnotationsOnlyWithInfo: %v", err)
	}

	r := openPDFBytes(t, res.PDF)
	if n := pageCount(t, r); n != 1 {
		t.Fatalf("pages=%d want=1", n)
	}
	cs := pageContent(t, r, 1)
	if strings.Contains(cs, "/Bg") {
		t.Fatalf("annotations-only page must not reference the background XObject, got %q", cs)
	}
	if !strings.Contains(cs, "rmv6-overlay") {
		t.Fatalf("expected overlay marker on emitted page")
	}
}
