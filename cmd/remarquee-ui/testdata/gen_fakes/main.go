package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/unidoc/unipdf/v3/creator"
)

type cPagesContent struct {
	FileType string `json:"fileType,omitempty"`
	CPages   struct {
		Pages []struct {
			ID      string `json:"id"`
			Redir   *value `json:"redir,omitempty"`
			Deleted *value `json:"deleted,omitempty"`
		} `json:"pages"`
	} `json:"cPages"`
}

type value struct {
	Timestamp string `json:"timestamp,omitempty"`
	Value     int    `json:"value,omitempty"`
}

func main() {
	if err := run(context.Background()); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %+v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	root := "/home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee"
	outDir := filepath.Join(root, "cmd", "remarquee-ui", "testdata", "generated")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	// Extract one real V6 .rm file from cpage-pdf.rmdoc (for “realistic” fixtures).
	sampleRM, err := extractFirstRMFile(filepath.Join(root, "cmd", "remarquee-ui", "testdata", "cpage-pdf.rmdoc"))
	if err != nil {
		return err
	}
	headerOnlyRM := makeV6HeaderOnlyRM()

	pdf2, err := makePDFWithNPages(2)
	if err != nil {
		return err
	}
	pdf1, err := makePDFWithNPages(1)
	if err != nil {
		return err
	}

	// 1) PDF-backed, 4 UI pages: dup page 0, inserted, page 1. No annotations.
	{
		docID := "fake-cpages-pdf-no-annotations"
		content := cPagesForPlan("pdf", []pagePlan{
			{id: "p0", redir: ptrInt(0)},
			{id: "p1", redir: ptrInt(0)},
			{id: "p2", redir: nil},       // inserted
			{id: "p3", redir: ptrInt(1)}, // second pdf page
		})
		files := map[string][]byte{
			docID + ".content":  mustJSON(content),
			docID + ".metadata": []byte(`{"generated":true}`),
			docID + ".pagedata": []byte("Blank\nBlank\nBlank\nBlank\n"),
			docID + ".pdf":      pdf2,
		}
		if err := writeRMDoc(filepath.Join(outDir, docID+".rmdoc"), docID, files, nil); err != nil {
			return err
		}
	}

	// 2) PDF-backed, 2 UI pages; page0 has header-only V6 rm file (parses but no content).
	{
		docID := "fake-cpages-pdf-v6-empty-rm"
		content := cPagesForPlan("pdf", []pagePlan{
			{id: "p0", redir: ptrInt(0)},
			{id: "p1", redir: ptrInt(0)},
		})
		files := map[string][]byte{
			docID + ".content":  mustJSON(content),
			docID + ".metadata": []byte(`{"generated":true}`),
			docID + ".pagedata": []byte("Blank\nBlank\n"),
			docID + ".pdf":      pdf1,
		}
		rmFiles := map[string][]byte{
			"p0.rm": headerOnlyRM,
		}
		if err := writeRMDoc(filepath.Join(outDir, docID+".rmdoc"), docID, files, rmFiles); err != nil {
			return err
		}
	}

	// 3) PDF-backed, 2 UI pages; page0 has real sample V6 rm (strokes/highlights depending on sample).
	{
		docID := "fake-cpages-pdf-v6-sample-rm"
		content := cPagesForPlan("pdf", []pagePlan{
			{id: "p0", redir: ptrInt(0)},
			{id: "p1", redir: ptrInt(0)},
		})
		files := map[string][]byte{
			docID + ".content":  mustJSON(content),
			docID + ".metadata": []byte(`{"generated":true}`),
			docID + ".pagedata": []byte("Blank\nBlank\n"),
			docID + ".pdf":      pdf1,
		}
		rmFiles := map[string][]byte{
			"p0.rm": sampleRM,
		}
		if err := writeRMDoc(filepath.Join(outDir, docID+".rmdoc"), docID, files, rmFiles); err != nil {
			return err
		}
	}

	// 4) Notebook-style: no PDF payload, 2 UI pages; both have header-only V6 rm.
	{
		docID := "fake-cpages-notebook-v6-empty"
		content := cPagesForPlan("", []pagePlan{
			{id: "p0", redir: nil},
			{id: "p1", redir: nil},
		})
		files := map[string][]byte{
			docID + ".content":  mustJSON(content),
			docID + ".metadata": []byte(`{"generated":true}`),
			docID + ".pagedata": []byte("Blank\nBlank\n"),
		}
		rmFiles := map[string][]byte{
			"p0.rm": headerOnlyRM,
			"p1.rm": headerOnlyRM,
		}
		if err := writeRMDoc(filepath.Join(outDir, docID+".rmdoc"), docID, files, rmFiles); err != nil {
			return err
		}
	}

	// 5) PDF-backed with deleted page entries in cPages (should be filtered by parser).
	{
		docID := "fake-cpages-pdf-deleted-pages"
		content := cPagesForPlanWithDeleted("pdf", []pagePlan{
			{id: "keep0", redir: ptrInt(0), deleted: false},
			{id: "del1", redir: ptrInt(0), deleted: true},
			{id: "keep2", redir: ptrInt(0), deleted: false},
		})
		files := map[string][]byte{
			docID + ".content":  mustJSON(content),
			docID + ".metadata": []byte(`{"generated":true}`),
			docID + ".pagedata": []byte("Blank\nBlank\nBlank\n"),
			docID + ".pdf":      pdf1,
		}
		if err := writeRMDoc(filepath.Join(outDir, docID+".rmdoc"), docID, files, nil); err != nil {
			return err
		}
	}

	fmt.Printf("ok: wrote fake fixtures to %s\n", outDir)
	return nil
}

type pagePlan struct {
	id      string
	redir   *int
	deleted bool
}

func cPagesForPlan(fileType string, pages []pagePlan) cPagesContent {
	var c cPagesContent
	c.FileType = fileType
	for _, p := range pages {
		pp := struct {
			ID      string `json:"id"`
			Redir   *value `json:"redir,omitempty"`
			Deleted *value `json:"deleted,omitempty"`
		}{ID: p.id}
		if p.redir != nil {
			pp.Redir = &value{Value: *p.redir}
		}
		c.CPages.Pages = append(c.CPages.Pages, pp)
	}
	return c
}

func cPagesForPlanWithDeleted(fileType string, pages []pagePlan) cPagesContent {
	var c cPagesContent
	c.FileType = fileType
	for _, p := range pages {
		pp := struct {
			ID      string `json:"id"`
			Redir   *value `json:"redir,omitempty"`
			Deleted *value `json:"deleted,omitempty"`
		}{ID: p.id}
		if p.redir != nil {
			pp.Redir = &value{Value: *p.redir}
		}
		if p.deleted {
			pp.Deleted = &value{Value: 1}
		}
		c.CPages.Pages = append(c.CPages.Pages, pp)
	}
	return c
}

func mustJSON(v any) []byte {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	return b
}

func ptrInt(v int) *int { return &v }

func makePDFWithNPages(n int) ([]byte, error) {
	c := creator.New()
	// Use the same default page size as rmapi (445 x 594 points).
	c.SetPageSize(creator.PageSize{445, 594})
	for i := 0; i < n; i++ {
		_ = c.NewPage()
		p := c.NewParagraph(fmt.Sprintf("Fake background page %d", i+1))
		p.SetFontSize(24)
		p.SetPos(72, 72)
		c.Draw(p)
	}
	var buf bytes.Buffer
	if err := c.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func makeV6HeaderOnlyRM() []byte {
	// Must be exactly the v6 header the reader expects (43 bytes).
	return []byte("reMarkable .lines file, version=6 ")
}

func extractFirstRMFile(rmdocPath string) ([]byte, error) {
	f, err := os.Open(rmdocPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	zr, err := zip.NewReader(f, fi.Size())
	if err != nil {
		return nil, err
	}
	for _, zf := range zr.File {
		if !strings.HasSuffix(zf.Name, ".rm") {
			continue
		}
		rc, err := zf.Open()
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return nil, err
		}
		return b, nil
	}
	return nil, errors.New("no .rm files found in fixture")
}

func writeRMDoc(outPath string, docID string, flatFiles map[string][]byte, rmFiles map[string][]byte) error {
	tmp := outPath + ".tmp"
	_ = os.Remove(tmp)
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	zw := zip.NewWriter(f)
	defer func() { _ = zw.Close() }()

	for name, b := range flatFiles {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		if _, err := w.Write(b); err != nil {
			return err
		}
	}

	for name, b := range rmFiles {
		w, err := zw.Create(docID + "/" + name)
		if err != nil {
			return err
		}
		if _, err := w.Write(b); err != nil {
			return err
		}
	}

	if err := zw.Close(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmp, outPath); err != nil {
		return err
	}
	return nil
}


