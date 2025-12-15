package rmdoc

import (
	"archive/zip"
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
	_ = ctx // reserved for future cancellation / IO abstraction

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

	contentJSON, err := readZipFile(contentFile)
	if err != nil {
		return nil, errors.Wrap(err, "read .content")
	}

	var metadataJSON []byte
	if metadataFile != nil {
		metadataJSON, err = readZipFile(metadataFile)
		if err != nil {
			return nil, errors.Wrap(err, "read .metadata")
		}
	}

	var pagedata string
	if pagedataFile != nil {
		b, err := readZipFile(pagedataFile)
		if err != nil {
			return nil, errors.Wrap(err, "read .pagedata")
		}
		pagedata = string(b)
	}

	var payloadPDF []byte
	if pdfFile != nil {
		payloadPDF, err = readZipFile(pdfFile)
		if err != nil {
			return nil, errors.Wrap(err, "read .pdf payload")
		}
	}

	docUUID := strings.TrimSuffix(path.Base(contentFile.Name), ".content")
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

func readZipFile(f *zip.File) ([]byte, error) {
	r, err := f.Open()
	if err != nil {
		return nil, errors.Wrap(err, "open zip entry")
	}
	defer func() { _ = r.Close() }()

	b, err := io.ReadAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "read zip entry")
	}

	return b, nil
}
