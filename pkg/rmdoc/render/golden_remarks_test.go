package render

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/go-go-golems/remarquee/pkg/pdfcmp"
	remarksref "github.com/go-go-golems/remarquee/pkg/refimpl/remarks"
)

func repoRootFromThisFile(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	// this file: <repo>/pkg/rmdoc/render/golden_remarks_test.go
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
}

func TestRenderV6Golden_RemarksReference_TestRmdoc(t *testing.T) {
	root := repoRootFromThisFile(t)
	fixture := filepath.Join(root, "cmd", "remarquee-ui", "testdata", "Test.rmdoc")
	if _, err := os.Stat(fixture); err != nil {
		t.Skipf("fixture not available: %s (%v)", fixture, err)
	}

	// Keep this bounded; rendering + running remarks can be slow on CI/dev machines.
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// 1) Render with remarquee (Go pipeline).
	actualPDFBytes, err := MergeRMDocV6OntoBackgroundPDF(ctx, fixture, V6MergeOptions{})
	if err != nil {
		t.Fatalf("MergeRMDocV6OntoBackgroundPDF: %v", err)
	}
	work := t.TempDir()
	actualPath := filepath.Join(work, "remarquee-v6.pdf")
	if err := os.WriteFile(actualPath, actualPDFBytes, 0o644); err != nil {
		t.Fatalf("write actual pdf: %v", err)
	}

	// 2) Render with remarks (reference implementation).
	refOutDir := filepath.Join(work, "remarks-out")
	if err := os.MkdirAll(refOutDir, 0o755); err != nil {
		t.Fatalf("mkdir remarks out: %v", err)
	}

	r := remarksref.Runner{LogLevel: "ERROR"}
	_, err = r.Run(ctx, fixture, refOutDir)
	if err != nil {
		if errors.Is(err, remarksref.ErrNotFound) {
			t.Skip("remarks not installed on PATH; skipping golden comparison against reference implementation")
		}
		t.Fatalf("run remarks: %v", err)
	}

	refPath, err := remarksref.FindSingleRemarksPDF(refOutDir)
	if err != nil {
		t.Fatalf("FindSingleRemarksPDF: %v", err)
	}

	// 3) Compare (visual + tolerance). Start loose; tighten once we get aligned.
	cmp, err := pdfcmp.CompareFilesVisual(ctx, actualPath, refPath, pdfcmp.Options{Tolerance: 0.01})
	if err != nil {
		t.Fatalf("CompareFilesVisual: %v", err)
	}

	if !cmp.Match {
		// Emit diff images for failed pages.
		for _, pr := range cmp.PageResults {
			if len(pr.DiffPNG) == 0 {
				continue
			}
			out := filepath.Join(work, fmt.Sprintf("diff-page-%03d.png", pr.PageIndex0))
			_ = os.WriteFile(out, pr.DiffPNG, 0o644)
			t.Logf("diff image: %s (diffRatio=%0.6f)", out, pr.DiffRatio)
		}
		t.Fatalf("pdf mismatch vs remarks reference: maxDiffRatio=%0.6f (A=%s B=%s)", cmp.MaxDiffRatio, cmp.SHA256A, cmp.SHA256B)
	}
}


