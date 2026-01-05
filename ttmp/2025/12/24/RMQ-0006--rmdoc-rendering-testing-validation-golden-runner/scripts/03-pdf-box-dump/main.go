package main

// Usage:
//   go run ./ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/03-pdf-box-dump/main.go /path/to/file.pdf

import (
	"fmt"
	"os"

	"github.com/unidoc/unipdf/v3/model"
)

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s /path/to/file.pdf\n", os.Args[0])
		os.Exit(2)
	}

	path := os.Args[1]
	f := must(os.Open(path))
	defer func() { _ = f.Close() }()

	r := must(model.NewPdfReader(f))
	n := must(r.GetNumPages())

	for i := 1; i <= n; i++ {
		p := must(r.GetPage(i))
		mb := must(p.GetMediaBox())
		cb := p.CropBox
		rot := int64(0)
		if p.Rotate != nil {
			rot = *p.Rotate
		}

		fmt.Printf("page=%d rotate=%d media=[%.2f %.2f %.2f %.2f]", i, rot, mb.Llx, mb.Lly, mb.Urx, mb.Ury)
		if cb != nil {
			fmt.Printf(" crop=[%.2f %.2f %.2f %.2f]", cb.Llx, cb.Lly, cb.Urx, cb.Ury)
		}
		fmt.Println()
	}
}
