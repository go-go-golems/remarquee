package rmdoc

import (
	"bytes"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	pdf "github.com/unidoc/unipdf/v3/model"
)

type pageSelection struct {
	All      bool
	Pages1   []int
	Indices0 []int
}

func parsePageSelection1Based(spec string, totalPages int) (pageSelection, error) {
	if totalPages <= 0 {
		return pageSelection{}, errors.Errorf("document has no pages")
	}

	spec = strings.TrimSpace(spec)
	if spec == "" {
		pages := make([]int, totalPages)
		indices := make([]int, totalPages)
		for i := 0; i < totalPages; i++ {
			pages[i] = i + 1
			indices[i] = i
		}
		return pageSelection{All: true, Pages1: pages, Indices0: indices}, nil
	}

	parts := strings.Split(spec, ",")
	pages := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "-") {
			bounds := strings.Split(part, "-")
			if len(bounds) != 2 {
				return pageSelection{}, errors.Errorf("invalid page range %q", part)
			}
			start, err := parsePositivePage(strings.TrimSpace(bounds[0]))
			if err != nil {
				return pageSelection{}, err
			}
			end, err := parsePositivePage(strings.TrimSpace(bounds[1]))
			if err != nil {
				return pageSelection{}, err
			}
			if end < start {
				return pageSelection{}, errors.Errorf("invalid descending page range %q", part)
			}
			for page := start; page <= end; page++ {
				pages = append(pages, page)
			}
			continue
		}

		page, err := parsePositivePage(part)
		if err != nil {
			return pageSelection{}, err
		}
		pages = append(pages, page)
	}

	if len(pages) == 0 {
		return pageSelection{}, errors.New("no pages selected")
	}

	indices := make([]int, 0, len(pages))
	for _, page := range pages {
		if page > totalPages {
			return pageSelection{}, errors.Errorf("page %d out of range (pages=%d)", page, totalPages)
		}
		indices = append(indices, page-1)
	}

	return pageSelection{All: false, Pages1: pages, Indices0: indices}, nil
}

func parsePositivePage(s string) (int, error) {
	if s == "" {
		return 0, errors.New("empty page number")
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, errors.Wrapf(err, "parse page %q", s)
	}
	if n <= 0 {
		return 0, errors.Errorf("page numbers must be 1-based, got %d", n)
	}
	return n, nil
}

func formatPages1Based(pages []int) string {
	parts := make([]string, 0, len(pages))
	for _, page := range pages {
		parts = append(parts, strconv.Itoa(page))
	}
	return strings.Join(parts, ",")
}

func extractPDFPages(inputPDF, outputPDF string, pages1 []int) error {
	b, err := os.ReadFile(inputPDF)
	if err != nil {
		return errors.Wrap(err, "read input pdf")
	}
	r, err := pdf.NewPdfReader(bytes.NewReader(b))
	if err != nil {
		return errors.Wrap(err, "open input pdf")
	}
	n, err := r.GetNumPages()
	if err != nil {
		return errors.Wrap(err, "get input pdf page count")
	}

	w := pdf.NewPdfWriter()
	for _, pageNum := range pages1 {
		if pageNum < 1 || pageNum > n {
			return errors.Errorf("page %d out of range for generated pdf (pages=%d)", pageNum, n)
		}
		page, err := r.GetPage(pageNum)
		if err != nil {
			return errors.Wrapf(err, "get input pdf page %d", pageNum)
		}
		if err := w.AddPage(page); err != nil {
			return errors.Wrapf(err, "add output pdf page %d", pageNum)
		}
	}

	f, err := os.Create(outputPDF)
	if err != nil {
		return errors.Wrap(err, "create output pdf")
	}
	defer func() { _ = f.Close() }()
	if err := w.Write(f); err != nil {
		return errors.Wrap(err, "write output pdf")
	}
	return nil
}
