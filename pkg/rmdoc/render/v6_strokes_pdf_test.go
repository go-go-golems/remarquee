package render

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readFirstRMFileFromRMDoc(t *testing.T, rmdocPath string) []byte {
	t.Helper()

	f, err := os.Open(rmdocPath)
	if err != nil {
		t.Fatalf("open rmdoc fixture: %v", err)
	}
	defer func() { _ = f.Close() }()

	fi, err := f.Stat()
	if err != nil {
		t.Fatalf("stat rmdoc fixture: %v", err)
	}

	zr, err := zip.NewReader(f, fi.Size())
	if err != nil {
		t.Fatalf("zip.NewReader: %v", err)
	}

	for _, zf := range zr.File {
		if !strings.HasSuffix(zf.Name, ".rm") {
			continue
		}
		rc, err := zf.Open()
		if err != nil {
			t.Fatalf("open zip entry: %v", err)
		}
		defer func() { _ = rc.Close() }()

		b, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("read .rm entry: %v", err)
		}
		return b
	}

	t.Fatalf("no .rm files found in fixture %s", rmdocPath)
	return nil
}

func TestRenderRMV6RMToPDFBytes(t *testing.T) {
	rmBytes := readFirstRMFileFromRMDoc(t, filepath.Join(repoRootInternal(t), "cmd", "remarquee-ui", "testdata", "cpage-pdf.rmdoc"))

	pdfBytes, err := RenderRMV6RMToPDFBytes(context.Background(), bytes.NewReader(rmBytes))
	if err != nil {
		t.Fatalf("RenderRMV6RMToPDFBytes: %v", err)
	}
	if len(pdfBytes) < 8 {
		t.Fatalf("pdf output too small: %d bytes", len(pdfBytes))
	}
	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Fatalf("expected PDF header, got prefix: %q", string(pdfBytes[:8]))
	}
}
