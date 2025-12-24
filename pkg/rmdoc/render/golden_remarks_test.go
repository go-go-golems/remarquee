package render

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/go-go-golems/remarquee/pkg/pdfcmp"
	remarksref "github.com/go-go-golems/remarquee/pkg/refimpl/remarks"
)

var updateGolden = flag.Bool("update-golden", false, "Update golden reference PDFs (writes into cmd/remarquee-ui/testdata/golden/)")

func repoRootFromThisFile(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	// this file: <repo>/pkg/rmdoc/render/golden_remarks_test.go
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
}

func goldenDir(root string) string {
	return filepath.Join(root, "cmd", "remarquee-ui", "testdata", "golden")
}

func remarksGoldenPath(root, fixturePath string) string {
	return filepath.Join(goldenDir(root), filepath.Base(fixturePath)+".remarks.pdf")
}

func copyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	_, err = io.Copy(out, in)
	return err
}

func ensureRemarksReferencePDF(
	ctx context.Context,
	t *testing.T,
	workspaceDir string,
	fixture string,
) (refPath string) {
	t.Helper()

	root := repoRootFromThisFile(t)
	golden := remarksGoldenPath(root, fixture)

	if _, err := os.Stat(golden); err == nil {
		return golden
	}

	// No committed golden: try generating with remarks.
	refOutDir := filepath.Join(workspaceDir, "remarks-out")
	if err := os.MkdirAll(refOutDir, 0o755); err != nil {
		t.Fatalf("mkdir remarks out: %v", err)
	}

	r := remarksref.Runner{LogLevel: "ERROR"}
	_, err := r.Run(ctx, fixture, refOutDir)
	if err != nil {
		if errors.Is(err, remarksref.ErrNotFound) {
			t.Skip("no golden reference present and remarks not installed on PATH; skipping")
		}
		t.Fatalf("run remarks: %v", err)
	}

	refPath, err = remarksref.FindSingleRemarksPDF(refOutDir)
	if err != nil {
		t.Fatalf("FindSingleRemarksPDF: %v", err)
	}

	// If requested, write the generated reference into the repo's golden dir.
	if *updateGolden {
		if err := copyFile(refPath, golden, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("updated golden: %s", golden)
		return golden
	}

	return refPath
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
	refPath := ensureRemarksReferencePDF(ctx, t, work, fixture)

	// 3) Compare (visual + tolerance). Start loose; tighten once we get aligned.
	cmp, err := pdfcmp.CompareFilesVisual(ctx, actualPath, refPath, pdfcmp.Options{Tolerance: 0.01})
	if err != nil {
		t.Fatalf("CompareFilesVisual: %v", err)
	}

	if !cmp.Match {
		// Emit diff images for failed pages.
		for _, pr := range cmp.PageResults {
			if pr.SizeMismatch {
				t.Logf("page %d size mismatch: %s", pr.PageIndex0, pr.Reason)
			}
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

func TestRenderV6Golden_RemarksReference_CpagePdf(t *testing.T) {
	root := repoRootFromThisFile(t)
	fixture := filepath.Join(root, "cmd", "remarquee-ui", "testdata", "cpage-pdf.rmdoc")
	if _, err := os.Stat(fixture); err != nil {
		t.Skipf("fixture not available: %s (%v)", fixture, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	actualPDFBytes, err := MergeRMDocV6OntoBackgroundPDF(ctx, fixture, V6MergeOptions{})
	if err != nil {
		t.Fatalf("MergeRMDocV6OntoBackgroundPDF: %v", err)
	}
	work := t.TempDir()
	actualPath := filepath.Join(work, "remarquee-v6.pdf")
	if err := os.WriteFile(actualPath, actualPDFBytes, 0o644); err != nil {
		t.Fatalf("write actual pdf: %v", err)
	}

	refPath := ensureRemarksReferencePDF(ctx, t, work, fixture)

	cmp, err := pdfcmp.CompareFilesVisual(ctx, actualPath, refPath, pdfcmp.Options{Tolerance: 0.01})
	if err != nil {
		t.Fatalf("CompareFilesVisual: %v", err)
	}

	if !cmp.Match {
		for _, pr := range cmp.PageResults {
			if pr.SizeMismatch {
				t.Logf("page %d size mismatch: %s", pr.PageIndex0, pr.Reason)
			}
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


