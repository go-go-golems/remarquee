package pdfcmp

import (
	"bytes"
	"context"
	"testing"

	"github.com/unidoc/unipdf/v3/contentstream"
	"github.com/unidoc/unipdf/v3/contentstream/draw"
	"github.com/unidoc/unipdf/v3/core"
	pdf "github.com/unidoc/unipdf/v3/model"
)

func buildTestPDFLine(x2 float64) ([]byte, error) {
	page := pdf.NewPdfPage()
	page.MediaBox = &pdf.PdfRectangle{Llx: 0, Lly: 0, Urx: 200, Ury: 200}
	page.CropBox = &pdf.PdfRectangle{Llx: 0, Lly: 0, Urx: 200, Ury: 200}
	page.Resources = pdf.NewPdfPageResources()

	cc := contentstream.NewContentCreator()
	cc.Add_q()
	cc.Add_w(2.0)
	cc.Add_RG(0, 0, 0)

	path := draw.NewPath()
	path = path.AppendPoint(draw.NewPoint(10, 10))
	path = path.AppendPoint(draw.NewPoint(x2, 190))
	draw.DrawPathWithCreator(path, cc)
	cc.Add_S()
	cc.Add_Q()

	page.SetContentStreams([]string{cc.Operations().String()}, core.NewFlateEncoder())

	w := pdf.NewPdfWriter()
	if err := w.AddPage(page); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := w.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func TestCompareBytesVisual_Identical(t *testing.T) {
	ctx := context.Background()
	pdfA, err := buildTestPDFLine(150)
	if err != nil {
		t.Fatalf("build pdf: %v", err)
	}

	res, err := CompareBytesVisual(ctx, pdfA, pdfA, Options{Tolerance: 0})
	if err != nil {
		t.Fatalf("CompareBytesVisual: %v", err)
	}
	if !res.Match {
		t.Fatalf("expected match, got mismatch: maxDiff=%v pages=%v", res.MaxDiffRatio, res.PageResults)
	}
}

func TestCompareBytesVisual_Different(t *testing.T) {
	ctx := context.Background()
	pdfA, err := buildTestPDFLine(150)
	if err != nil {
		t.Fatalf("build pdf A: %v", err)
	}
	pdfB, err := buildTestPDFLine(140)
	if err != nil {
		t.Fatalf("build pdf B: %v", err)
	}

	res, err := CompareBytesVisual(ctx, pdfA, pdfB, Options{Tolerance: 0})
	if err != nil {
		t.Fatalf("CompareBytesVisual: %v", err)
	}
	if res.Match {
		t.Fatalf("expected mismatch")
	}
	if len(res.PageResults) != 1 {
		t.Fatalf("expected 1 page result, got %d", len(res.PageResults))
	}
	if res.PageResults[0].DiffRatio <= 0 {
		t.Fatalf("expected diffRatio > 0, got %v", res.PageResults[0].DiffRatio)
	}
	// By default we generate diff PNG bytes for failing pages.
	if len(res.PageResults[0].DiffPNG) == 0 {
		t.Fatalf("expected diff PNG bytes to be generated")
	}
}

func TestCompareBytesStructural_Identical(t *testing.T) {
	ctx := context.Background()
	pdfA, err := buildTestPDFLine(150)
	if err != nil {
		t.Fatalf("build pdf: %v", err)
	}

	res, err := CompareBytesStructural(ctx, pdfA, pdfA, StructuralOptions{})
	if err != nil {
		t.Fatalf("CompareBytesStructural: %v", err)
	}
	if !res.Match {
		t.Fatalf("expected match, got mismatch: %+v", res.PageResults)
	}
}


