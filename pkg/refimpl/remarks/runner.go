package remarks

import (
	"bytes"
	"context"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// ErrNotFound indicates the `remarks` executable is not available on PATH (or at Runner.Bin).
var ErrNotFound = errors.New("remarks executable not found")

// Runner shells out to the Python `remarks` CLI as a reference implementation.
//
// This is used for golden testing / validation (Option B PDF comparison stays in Go, but we still
// want a convenient way to produce reference PDFs).
type Runner struct {
	// Bin is the executable name/path. If empty, defaults to "remarks".
	Bin string

	// LogLevel sets remarks' --log_level (DEBUG/INFO/WARNING/ERROR). Empty means omit flag.
	LogLevel string

	// ExtraArgs are appended to the command line (rare; keep empty unless needed).
	ExtraArgs []string
}

type Result struct {
	Args   []string
	Stdout string
	Stderr string
}

func (r Runner) bin() string {
	if strings.TrimSpace(r.Bin) != "" {
		return r.Bin
	}
	return "remarks"
}

// Run executes: remarks <input> <outputDir> [--log_level X]
func (r Runner) Run(ctx context.Context, inputPath, outputDir string) (*Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	bin := r.bin()
	if _, err := exec.LookPath(bin); err != nil {
		return nil, ErrNotFound
	}

	args := []string{inputPath, outputDir}
	if strings.TrimSpace(r.LogLevel) != "" {
		args = append(args, "--log_level", strings.TrimSpace(r.LogLevel))
	}
	if len(r.ExtraArgs) > 0 {
		args = append(args, r.ExtraArgs...)
	}

	cmd := exec.CommandContext(ctx, bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return &Result{Args: append([]string{bin}, args...), Stdout: stdout.String(), Stderr: stderr.String()},
			errors.Wrap(err, "run remarks")
	}

	return &Result{Args: append([]string{bin}, args...), Stdout: stdout.String(), Stderr: stderr.String()}, nil
}

// FindRemarksPDFs finds PDFs created by remarks under outputDir.
//
// remarks typically writes files with suffix " _remarks.pdf" and may place them in nested
// directories matching the device UI path.
func FindRemarksPDFs(outputDir string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(outputDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), " _remarks.pdf") {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "walk output dir")
	}
	return out, nil
}

// FindSingleRemarksPDF finds exactly one remarks PDF in outputDir.
func FindSingleRemarksPDF(outputDir string) (string, error) {
	pdfs, err := FindRemarksPDFs(outputDir)
	if err != nil {
		return "", err
	}
	if len(pdfs) == 0 {
		return "", errors.Errorf("no remarks pdf found under %q", outputDir)
	}
	if len(pdfs) > 1 {
		return "", errors.Errorf("expected 1 remarks pdf under %q, found %d", outputDir, len(pdfs))
	}
	return pdfs[0], nil
}
