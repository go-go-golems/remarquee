// RMQ-0006 helper: dump per-page PDF box sizes (MediaBox/CropBox) + rotation.
//
// Usage:
//   go run ./ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/03-pdf-box-dump.go /path/to/file.pdf
//
// Why:
// - When pdfcmp reports maxDiffRatio=1.0, the first suspicion is a *raster dimension mismatch*.
// - Raster dimensions typically mismatch when PDF page boxes differ (or rotation differs).
//
// This tool is deterministic and does not require external tools like pdfinfo.
package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/pkg/errors"
	pdf "github.com/unidoc/unipdf/v3/model"
)

func main() {
	if len(os.Args) != 2 {
		_, _ = fmt.Fprintf(os.Stderr, "usage: %s /path/to/file.pdf\n", os.Args[0])
		os.Exit(2)
	}
	path := os.Args[1]
	b, err := os.ReadFile(path)
	must(err)

	r, err := pdf.NewPdfReader(bytes.NewReader(b))
	must(errors.Wrap(err, "open reader"))

	n, err := r.GetNumPages()
	must(errors.Wrap(err, "GetNumPages"))

	fmt.Printf("PDF=%s\n", path)
	fmt.Printf("Pages=%d\n", n)
	for i := 1; i <= n; i++ {
		p, err := r.GetPage(i)
		must(errors.Wrapf(err, "GetPage(%d)", i))

		rot := int64(0)
		if p.Rotate != nil {
			rot = *p.Rotate
		}

		var mboxW, mboxH float64
		if mb, err := p.GetMediaBox(); err == nil && mb != nil {
			mboxW = mb.Urx - mb.Llx
			mboxH = mb.Ury - mb.Lly
		}

		var cboxW, cboxH float64
		if p.CropBox != nil {
			cboxW = p.CropBox.Urx - p.CropBox.Llx
			cboxH = p.CropBox.Ury - p.CropBox.Lly
		}

		fmt.Printf("page=%d rot=%d media=%0.3fx%0.3f crop=%0.3fx%0.3f\n", i, rot, mboxW, mboxH, cboxW, cboxH)
	}
}

func must(err error) {
	if err == nil {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "error: %+v\n", err)
	os.Exit(1)
}


