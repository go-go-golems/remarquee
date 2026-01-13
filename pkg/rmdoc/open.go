package rmdoc

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
)

func OpenFile(ctx context.Context, p string) (*Document, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, errors.Wrap(err, "open rmdoc file")
	}
	defer func() { _ = f.Close() }()

	fi, err := f.Stat()
	if err != nil {
		return nil, errors.Wrap(err, "stat rmdoc file")
	}

	return OpenReaderAt(ctx, f, fi.Size())
}

func OpenReaderAt(ctx context.Context, r io.ReaderAt, size int64) (*Document, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, errors.Wrap(err, "open zip reader")
	}

	contentFile, err := findUniqueByExt(zr, ".content")
	if err != nil {
		return nil, err
	}
	metadataFile, _ := findOptionalByExt(zr, ".metadata")
	pagedataFile, _ := findOptionalByExt(zr, ".pagedata")
	pdfFile, _ := findOptionalByExt(zr, ".pdf")

	contentJSON, err := readZipFile(ctx, contentFile)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, errors.Wrap(err, "read .content")
	}

	var metadataJSON []byte
	if metadataFile != nil {
		metadataJSON, err = readZipFile(ctx, metadataFile)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, ctxErr
			}
			return nil, errors.Wrap(err, "read .metadata")
		}
	}

	var pagedata string
	if pagedataFile != nil {
		b, err := readZipFile(ctx, pagedataFile)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, ctxErr
			}
			return nil, errors.Wrap(err, "read .pagedata")
		}
		pagedata = string(b)
	}

	var payloadPDF []byte
	if pdfFile != nil {
		payloadPDF, err = readZipFile(ctx, pdfFile)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, ctxErr
			}
			return nil, errors.Wrap(err, "read .pdf payload")
		}
	}

	docUUID := strings.TrimSuffix(path.Base(contentFile.Name), ".content")
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	schema, docType, pages, err := ParseContent(contentJSON)
	if err != nil {
		return nil, err
	}
	pages = ApplyPagedataTemplates(pages, pagedata)

	return &Document{
		UUID:         docUUID,
		Schema:       schema,
		Type:         docType,
		ContentJSON:  contentJSON,
		MetadataJSON: metadataJSON,
		Pagedata:     pagedata,
		PayloadPDF:   payloadPDF,
		Pages:        pages,
	}, nil
}

func findUniqueByExt(zr *zip.Reader, ext string) (*zip.File, error) {
	var out *zip.File
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, ext) {
			if out != nil {
				return nil, errors.Errorf("archive contains multiple %s files (%s, %s)", ext, out.Name, f.Name)
			}
			out = f
		}
	}
	if out == nil {
		return nil, errors.Errorf("archive does not contain a %s file", ext)
	}
	return out, nil
}

func findOptionalByExt(zr *zip.Reader, ext string) (*zip.File, error) {
	var out *zip.File
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, ext) {
			out = f
			break
		}
	}
	return out, nil
}

func readZipFile(ctx context.Context, f *zip.File) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r, err := f.Open()
	if err != nil {
		return nil, errors.Wrap(err, "open zip entry")
	}
	defer func() { _ = r.Close() }()

	var buf bytes.Buffer
	// Avoid unbounded allocations; only pre-grow for reasonably sized entries.
	if f.UncompressedSize64 > 0 && f.UncompressedSize64 <= 64*1024*1024 {
		buf.Grow(int(f.UncompressedSize64))
	}

	tmp := make([]byte, 32*1024)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		n, err := r.Read(tmp)
		if n > 0 {
			_, _ = buf.Write(tmp[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "read zip entry")
		}
	}

	return buf.Bytes(), nil
}
