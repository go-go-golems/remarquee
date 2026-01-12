package pdfcmp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/unidoc/unipdf/v3/extractor"
	pdf "github.com/unidoc/unipdf/v3/model"
	"github.com/unidoc/unipdf/v3/render"
)

// Options controls PDF comparison behavior.
type Options struct {
	// Tolerance is the fraction of pixels that are allowed to differ (0.01 = 1%).
	Tolerance float64

	// RasterDPI controls PDF->image rasterization DPI when using external rasterizers (pdftoppm fallback).
	// If <= 0, defaults to 200.
	RasterDPI int

	// GenerateDiff enables producing per-page diff images (PNG) for failing pages.
	// If nil, defaults to true (golden-test ergonomics).
	GenerateDiff *bool

	// MaxPages limits the number of pages compared. If <= 0, compare all pages.
	MaxPages int
}

func (o Options) withDefaults() Options {
	if o.Tolerance < 0 {
		o.Tolerance = 0
	}
	if o.RasterDPI <= 0 {
		o.RasterDPI = 200
	}
	if o.GenerateDiff == nil {
		v := true
		o.GenerateDiff = &v
	}
	return o
}

type PageResult struct {
	PageIndex0 int

	// DiffRatio is fraction of differing pixels.
	DiffRatio float64

	// SizeMismatch indicates images rendered to different dimensions.
	SizeMismatch bool

	// DiffPNG is a PNG-encoded diff image (nil if not requested or not available).
	DiffPNG []byte

	// Reason is a short human-readable diagnostic.
	Reason string
}

type Result struct {
	Match       bool
	PageCountA  int
	PageCountB  int
	PageResults []PageResult

	// MaxDiffRatio across pages (ignoring size mismatch pages).
	MaxDiffRatio float64

	// Input fingerprints for debugging/reporting.
	SHA256A string
	SHA256B string
}

// StructuralOptions controls non-visual (structure/text/annotation) comparisons.
type StructuralOptions struct {
	// MaxPages limits the number of pages compared. If <= 0, compare all pages.
	MaxPages int
}

type StructuralPageResult struct {
	PageIndex0 int

	AnnotationCountA int
	AnnotationCountB int

	RotationA int64
	RotationB int64

	TextSHA256A string
	TextSHA256B string

	Match  bool
	Reason string
}

type StructuralResult struct {
	Match      bool
	PageCountA int
	PageCountB int

	PageResults []StructuralPageResult

	SHA256A string
	SHA256B string
}

// CompareFilesVisual compares two PDFs by rendering each page to an image using UniDoc's renderer
// and then doing a per-pixel comparison.
//
// This is intentionally a small, self-contained utility to support golden tests.
func CompareFilesVisual(ctx context.Context, pathA, pathB string, opts Options) (*Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	opts = opts.withDefaults()

	a, err := os.ReadFile(pathA)
	if err != nil {
		return nil, errors.Wrap(err, "read pdf A")
	}
	b, err := os.ReadFile(pathB)
	if err != nil {
		return nil, errors.Wrap(err, "read pdf B")
	}

	// Primary path: pure-Go UniDoc renderer.
	res, err := CompareBytesVisual(ctx, a, b, opts)
	if err == nil {
		return res, nil
	}

	// Fallback: UniDoc rendering can fail with strict "type check error" on some PDFs
	// (notably certain remarks outputs). If Poppler is available, use pdftoppm to rasterize.
	if !strings.Contains(err.Error(), "type check error") {
		return nil, err
	}
	if _, lookErr := exec.LookPath("pdftoppm"); lookErr != nil {
		return nil, err
	}

	return compareFilesVisualWithPDFToPPM(ctx, pathA, pathB, a, b, opts)
}

func compareFilesVisualWithPDFToPPM(ctx context.Context, pathA, pathB string, pdfA, pdfB []byte, opts Options) (*Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	opts = opts.withDefaults()

	rA, err := pdf.NewPdfReader(bytes.NewReader(pdfA))
	if err != nil {
		return nil, errors.Wrap(err, "open pdf A reader")
	}
	rB, err := pdf.NewPdfReader(bytes.NewReader(pdfB))
	if err != nil {
		return nil, errors.Wrap(err, "open pdf B reader")
	}

	nA, err := rA.GetNumPages()
	if err != nil {
		return nil, errors.Wrap(err, "get pdf A num pages")
	}
	nB, err := rB.GetNumPages()
	if err != nil {
		return nil, errors.Wrap(err, "get pdf B num pages")
	}

	res := &Result{
		Match:      true,
		PageCountA: nA,
		PageCountB: nB,
		SHA256A:    sha256Hex(pdfA),
		SHA256B:    sha256Hex(pdfB),
	}

	if nA != nB {
		res.Match = false
		return res, nil
	}

	maxPages := nA
	if opts.MaxPages > 0 && opts.MaxPages < maxPages {
		maxPages = opts.MaxPages
	}

	tmpDir, err := os.MkdirTemp("", "pdfcmp-pdftoppm-*")
	if err != nil {
		return nil, errors.Wrap(err, "create temp dir for pdftoppm")
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	for i := 1; i <= maxPages; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		imgA, err := renderPDFPageWithPDFToPPM(ctx, tmpDir, "A", pathA, i, opts.RasterDPI)
		if err != nil {
			return nil, errors.Wrapf(err, "pdftoppm render pdf A page %d", i)
		}
		imgB, err := renderPDFPageWithPDFToPPM(ctx, tmpDir, "B", pathB, i, opts.RasterDPI)
		if err != nil {
			return nil, errors.Wrapf(err, "pdftoppm render pdf B page %d", i)
		}

		pr := compareImages(imgA, imgB, opts.Tolerance, *opts.GenerateDiff)
		pr.PageIndex0 = i - 1
		pr.Reason = "pdftoppm rasterizer (fallback after UniDoc type check error)"
		res.PageResults = append(res.PageResults, pr)

		if pr.SizeMismatch || pr.DiffRatio > opts.Tolerance {
			res.Match = false
		}
		if pr.DiffRatio > res.MaxDiffRatio {
			res.MaxDiffRatio = pr.DiffRatio
		}
	}

	return res, nil
}

func renderPDFPageWithPDFToPPM(ctx context.Context, tmpDir, prefix, pdfPath string, pageNum, dpi int) (image.Image, error) {
	if dpi <= 0 {
		dpi = 200
	}

	outBase := filepath.Join(tmpDir, fmt.Sprintf("%s-page-%03d", prefix, pageNum))
	outFile := outBase + ".png"

	cmd := exec.CommandContext(ctx,
		"pdftoppm",
		"-png",
		"-r", strconv.Itoa(dpi),
		"-f", strconv.Itoa(pageNum),
		"-l", strconv.Itoa(pageNum),
		"-singlefile",
		pdfPath,
		outBase,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, errors.Wrapf(err, "pdftoppm failed: %s", strings.TrimSpace(stderr.String()))
	}

	b, err := os.ReadFile(outFile)
	if err != nil {
		return nil, errors.Wrapf(err, "read pdftoppm output %s", outFile)
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, errors.Wrap(err, "decode pdftoppm png")
	}
	return img, nil
}

func CompareBytesVisual(ctx context.Context, pdfA, pdfB []byte, opts Options) (*Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	opts = opts.withDefaults()

	rA, err := pdf.NewPdfReader(bytes.NewReader(pdfA))
	if err != nil {
		return nil, errors.Wrap(err, "open pdf A reader")
	}
	rB, err := pdf.NewPdfReader(bytes.NewReader(pdfB))
	if err != nil {
		return nil, errors.Wrap(err, "open pdf B reader")
	}

	nA, err := rA.GetNumPages()
	if err != nil {
		return nil, errors.Wrap(err, "get pdf A num pages")
	}
	nB, err := rB.GetNumPages()
	if err != nil {
		return nil, errors.Wrap(err, "get pdf B num pages")
	}

	res := &Result{
		Match:      true,
		PageCountA: nA,
		PageCountB: nB,
		SHA256A:    sha256Hex(pdfA),
		SHA256B:    sha256Hex(pdfB),
	}

	// Page count mismatch is immediate fail, but we still return a structured result.
	if nA != nB {
		res.Match = false
		return res, nil
	}

	maxPages := nA
	if opts.MaxPages > 0 && opts.MaxPages < maxPages {
		maxPages = opts.MaxPages
	}

	device := render.NewImageDevice()

	for i := 1; i <= maxPages; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		pA, err := rA.GetPage(i)
		if err != nil {
			return nil, errors.Wrapf(err, "get pdf A page %d", i)
		}
		pB, err := rB.GetPage(i)
		if err != nil {
			return nil, errors.Wrapf(err, "get pdf B page %d", i)
		}

		imgA, err := device.Render(pA)
		if err != nil {
			return nil, errors.Wrapf(err, "render pdf A page %d", i)
		}
		imgB, err := device.Render(pB)
		if err != nil {
			return nil, errors.Wrapf(err, "render pdf B page %d", i)
		}

		pr := compareImages(imgA, imgB, opts.Tolerance, *opts.GenerateDiff)
		pr.PageIndex0 = i - 1
		res.PageResults = append(res.PageResults, pr)

		if pr.SizeMismatch || pr.DiffRatio > opts.Tolerance {
			res.Match = false
		}
		if pr.DiffRatio > res.MaxDiffRatio {
			res.MaxDiffRatio = pr.DiffRatio
		}
	}

	return res, nil
}

// CompareFilesStructural compares two PDFs without rendering (fast feedback):
// - page count
// - per-page annotation count
// - per-page extracted text hash (normalized)
func CompareFilesStructural(ctx context.Context, pathA, pathB string, opts StructuralOptions) (*StructuralResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	a, err := os.ReadFile(pathA)
	if err != nil {
		return nil, errors.Wrap(err, "read pdf A")
	}
	b, err := os.ReadFile(pathB)
	if err != nil {
		return nil, errors.Wrap(err, "read pdf B")
	}
	return CompareBytesStructural(ctx, a, b, opts)
}

func CompareBytesStructural(ctx context.Context, pdfA, pdfB []byte, opts StructuralOptions) (*StructuralResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	rA, err := pdf.NewPdfReader(bytes.NewReader(pdfA))
	if err != nil {
		return nil, errors.Wrap(err, "open pdf A reader")
	}
	rB, err := pdf.NewPdfReader(bytes.NewReader(pdfB))
	if err != nil {
		return nil, errors.Wrap(err, "open pdf B reader")
	}

	nA, err := rA.GetNumPages()
	if err != nil {
		return nil, errors.Wrap(err, "get pdf A num pages")
	}
	nB, err := rB.GetNumPages()
	if err != nil {
		return nil, errors.Wrap(err, "get pdf B num pages")
	}

	res := &StructuralResult{
		Match:      true,
		PageCountA: nA,
		PageCountB: nB,
		SHA256A:    sha256Hex(pdfA),
		SHA256B:    sha256Hex(pdfB),
	}

	if nA != nB {
		res.Match = false
		return res, nil
	}

	maxPages := nA
	if opts.MaxPages > 0 && opts.MaxPages < maxPages {
		maxPages = opts.MaxPages
	}

	for i := 1; i <= maxPages; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		pA, err := rA.GetPage(i)
		if err != nil {
			return nil, errors.Wrapf(err, "get pdf A page %d", i)
		}
		pB, err := rB.GetPage(i)
		if err != nil {
			return nil, errors.Wrapf(err, "get pdf B page %d", i)
		}

		ar, err := annotationCount(pA)
		if err != nil {
			return nil, errors.Wrapf(err, "count annotations A page %d", i)
		}
		br, err := annotationCount(pB)
		if err != nil {
			return nil, errors.Wrapf(err, "count annotations B page %d", i)
		}

		rotA := int64(0)
		if pA.Rotate != nil {
			rotA = *pA.Rotate
		}
		rotB := int64(0)
		if pB.Rotate != nil {
			rotB = *pB.Rotate
		}

		txtA, err := extractTextNormalized(pA)
		if err != nil {
			return nil, errors.Wrapf(err, "extract text A page %d", i)
		}
		txtB, err := extractTextNormalized(pB)
		if err != nil {
			return nil, errors.Wrapf(err, "extract text B page %d", i)
		}

		spr := StructuralPageResult{
			PageIndex0:       i - 1,
			AnnotationCountA: ar,
			AnnotationCountB: br,
			RotationA:        rotA,
			RotationB:        rotB,
			TextSHA256A:      sha256Hex([]byte(txtA)),
			TextSHA256B:      sha256Hex([]byte(txtB)),
			Match:            true,
			Reason:           "match",
		}

		if ar != br {
			spr.Match = false
			spr.Reason = "annotation count mismatch"
		} else if rotA != rotB {
			spr.Match = false
			spr.Reason = "rotation mismatch"
		} else if spr.TextSHA256A != spr.TextSHA256B {
			spr.Match = false
			spr.Reason = "extracted text mismatch"
		}

		if !spr.Match {
			res.Match = false
		}
		res.PageResults = append(res.PageResults, spr)
	}

	return res, nil
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func compareImages(a, b image.Image, tolerance float64, generateDiff bool) PageResult {
	ra := a.Bounds()
	rb := b.Bounds()
	// Poppler / PDF renderers can differ by +/- 1px due to rounding.
	// Treat tiny size mismatches as non-fatal by comparing only the overlapping area.
	w, h := ra.Dx(), ra.Dy()
	if ra.Dx() != rb.Dx() || ra.Dy() != rb.Dy() {
		abs := func(x int) int {
			if x < 0 {
				return -x
			}
			return x
		}
		dx := abs(ra.Dx() - rb.Dx())
		dy := abs(ra.Dy() - rb.Dy())
		if dx <= 1 && dy <= 1 {
			if rb.Dx() < w {
				w = rb.Dx()
			}
			if rb.Dy() < h {
				h = rb.Dy()
			}
			// Continue with overlap comparison. Do not mark SizeMismatch, or callers will auto-fail.
		} else {
			return PageResult{
				SizeMismatch: true,
				DiffRatio:    1.0,
				Reason:       "rendered image size mismatch",
			}
		}
	}

	if w == 0 || h == 0 {
		return PageResult{
			DiffRatio: 0,
			Reason:    "empty image bounds",
		}
	}

	total := float64(w * h)
	var diff float64

	// First pass: count diffs.
	threshold := tolerance * total
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !sameColor(a.At(ra.Min.X+x, ra.Min.Y+y), b.At(rb.Min.X+x, rb.Min.Y+y)) {
				diff++
				_ = threshold // keep for readability; no early-exit currently.
			}
		}
	}

	diffRatio := diff / total

	pr := PageResult{
		DiffRatio: diffRatio,
	}
	if diffRatio <= tolerance {
		pr.Reason = "within tolerance"
		return pr
	}
	pr.Reason = "diff exceeds tolerance"

	if generateDiff {
		diffImg := image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				ca := a.At(ra.Min.X+x, ra.Min.Y+y)
				cb := b.At(rb.Min.X+x, rb.Min.Y+y)
				if sameColor(ca, cb) {
					// Dim the original a pixel to make changes pop.
					r, g, bl, _ := ca.RGBA()
					diffImg.SetRGBA(x, y, color.RGBA{
						R: uint8((r >> 8) / 2),
						G: uint8((g >> 8) / 2),
						B: uint8((bl >> 8) / 2),
						A: 255,
					})
					continue
				}
				// Highlight diffs in red.
				diffImg.SetRGBA(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
			}
		}
		var buf bytes.Buffer
		_ = png.Encode(&buf, diffImg)
		pr.DiffPNG = buf.Bytes()
	}

	return pr
}

func sameColor(a, b color.Color) bool {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}

func annotationCount(p *pdf.PdfPage) (int, error) {
	ann, err := p.GetAnnotations()
	if err != nil {
		return 0, err
	}
	if ann == nil {
		return 0, nil
	}
	return len(ann), nil
}

func extractTextNormalized(p *pdf.PdfPage) (string, error) {
	ex, err := extractor.New(p)
	if err != nil {
		return "", err
	}
	txt, err := ex.ExtractText()
	if err != nil {
		return "", err
	}
	// Normalization: collapse whitespace so small differences in spacing don't become noise.
	txt = strings.ReplaceAll(txt, "\r\n", "\n")
	txt = strings.TrimSpace(txt)
	parts := strings.Fields(txt)
	return strings.Join(parts, " "), nil
}

// WritePageDiffPNGs writes diff images for pages that have them.
// The caller decides where to store and how to name files; this helper just encodes.
func WritePageDiffPNGs(w io.Writer, pr PageResult) error {
	_, err := w.Write(pr.DiffPNG)
	return err
}
