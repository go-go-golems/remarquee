package render

import (
	"bytes"
	"context"
	"fmt"

	"github.com/go-go-golems/remarquee/pkg/rmdoc"
	"github.com/pkg/errors"
	"github.com/unidoc/unipdf/v3/creator"
	pdf "github.com/unidoc/unipdf/v3/model"
)

// BackgroundOptions controls how background PDFs are constructed.
type BackgroundOptions struct {
	// DefaultPageSize is used when no PDF payload is available to infer page size.
	// This is also used for "inserted pages" when we cannot infer a size from the payload.
	//
	// If zero, we default to the rmapi legacy constant (445 x 594 points).
	DefaultPageSize creator.PageSize
}

func (o BackgroundOptions) withDefaults() BackgroundOptions {
	if o.DefaultPageSize[0] == 0 || o.DefaultPageSize[1] == 0 {
		// Keep this aligned with rmapi/annotations/pdf.go (rmPageSize).
		o.DefaultPageSize = creator.PageSize{445, 594}
	}
	return o
}

// BuildBackgroundPDF assembles a "UI-ordered" background PDF based on the rmdoc page plan.
//
// For PDF-backed documents:
// - SourcePDFPage >= 0 copies that page from doc.PayloadPDF (SourcePDFPage is 0-based).
// - SourcePDFPage == rmdoc.InsertedPage inserts a blank page matching the payload page size.
//
// For notebook documents (no payload PDF), it currently creates blank pages for each UI page
// using opts.DefaultPageSize. Template backgrounds are a later milestone.
func BuildBackgroundPDF(ctx context.Context, doc *rmdoc.Document, opts BackgroundOptions) ([]byte, error) {
	if doc == nil {
		return nil, errors.New("doc is nil")
	}
	indices := make([]int, len(doc.Pages))
	for i := range doc.Pages {
		indices[i] = i
	}
	return BuildBackgroundPDFForPages(ctx, doc, opts, indices)
}

// BuildBackgroundPDFForPages assembles a UI-ordered background PDF for a subset of pages.
func BuildBackgroundPDFForPages(ctx context.Context, doc *rmdoc.Document, opts BackgroundOptions, pageIndices []int) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if doc == nil {
		return nil, errors.New("doc is nil")
	}
	if len(pageIndices) == 0 {
		return nil, errors.New("pageIndices is empty")
	}
	opts = opts.withDefaults()

	c := creator.New()

	var payloadReader *pdf.PdfReader
	if len(doc.PayloadPDF) > 0 {
		r, err := pdf.NewPdfReader(bytes.NewReader(doc.PayloadPDF))
		if err != nil {
			return nil, errors.Wrap(err, "open payload PDF reader")
		}

		encrypted, err := r.IsEncrypted()
		if err != nil {
			return nil, errors.Wrap(err, "check payload PDF encryption")
		}
		if encrypted {
			ok, err := r.Decrypt([]byte(""))
			if err != nil {
				return nil, errors.Wrap(err, "decrypt payload PDF")
			}
			if !ok {
				return nil, errors.New("cannot decrypt payload PDF with empty password")
			}
		}

		payloadReader = r
	}

	payloadPageSize, err := inferPayloadPageSize(payloadReader, doc.Pages, opts.DefaultPageSize)
	if err != nil {
		return nil, err
	}

	for i, idx := range pageIndices {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if idx < 0 || idx >= len(doc.Pages) {
			return nil, errors.Errorf("pageIndices[%d]=%d out of range (pages=%d)", i, idx, len(doc.Pages))
		}

		page := doc.Pages[idx]
		if payloadReader != nil && page.SourcePDFPage != rmdoc.InsertedPage {
			pageNum := page.SourcePDFPage + 1 // unipdf is 1-based
			if pageNum <= 0 {
				return nil, errors.Errorf("page[%d]: invalid SourcePDFPage=%d", idx, page.SourcePDFPage)
			}

			srcPage, err := payloadReader.GetPage(pageNum)
			if err != nil {
				return nil, errors.Wrapf(err, "get payload page %d (ui idx=%d page_id=%s)", pageNum, page.Index, page.PageID)
			}

			// Important: if a source PDF page is referenced multiple times in the UI plan,
			// we must duplicate it. Adding the same *PdfPage instance twice can result in
			// fewer pages in the output.
			dupPage := srcPage.Duplicate()

			mbox, err := dupPage.GetMediaBox()
			if err != nil {
				return nil, errors.Wrapf(err, "get payload page %d media box", pageNum)
			}

			w := mbox.Urx - mbox.Llx
			h := mbox.Ury - mbox.Lly
			c.SetPageSize(creator.PageSize{w, h})
			if err := c.AddPage(dupPage); err != nil {
				return nil, errors.Wrapf(err, "add payload page %d", pageNum)
			}
			continue
		}

		c.SetPageSize(payloadPageSize)
		_ = c.NewPage()
	}

	var buf bytes.Buffer
	if err := c.Write(&buf); err != nil {
		return nil, errors.Wrap(err, "write background PDF")
	}

	return buf.Bytes(), nil
}

func inferPayloadPageSize(payloadReader *pdf.PdfReader, pages []rmdoc.PageRef, defaultSize creator.PageSize) (creator.PageSize, error) {
	if payloadReader == nil {
		return defaultSize, nil
	}

	// Prefer a page size from an actually-referenced page in the page plan.
	for _, p := range pages {
		if p.SourcePDFPage == rmdoc.InsertedPage {
			continue
		}
		pageNum := p.SourcePDFPage + 1
		if pageNum <= 0 {
			continue
		}
		srcPage, err := payloadReader.GetPage(pageNum)
		if err != nil {
			continue
		}
		mbox, err := srcPage.GetMediaBox()
		if err != nil {
			continue
		}
		w := mbox.Urx - mbox.Llx
		h := mbox.Ury - mbox.Lly
		if w <= 0 || h <= 0 {
			continue
		}
		return creator.PageSize{w, h}, nil
	}

	// Fall back to payload page 1 if nothing is referenced (edge case).
	srcPage, err := payloadReader.GetPage(1)
	if err != nil {
		return creator.PageSize{}, errors.Wrap(err, "payload PDF has no pages")
	}
	mbox, err := srcPage.GetMediaBox()
	if err != nil {
		return creator.PageSize{}, errors.Wrap(err, "get payload page 1 media box")
	}

	w := mbox.Urx - mbox.Llx
	h := mbox.Ury - mbox.Lly
	if w <= 0 || h <= 0 {
		return creator.PageSize{}, fmt.Errorf("invalid payload page size w=%v h=%v", w, h)
	}
	return creator.PageSize{w, h}, nil
}
