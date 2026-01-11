package compile

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-go-golems/remarquee/pkg/rmdsl"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type CompileOptions struct {
	DocUUID    string
	Now        time.Time
	Author     uint8
	AuthorUUID string
}

func Compile(ctx context.Context, doc *rmdsl.Doc, opts CompileOptions) (*CompiledDoc, error) {
	_ = ctx
	if err := rmdsl.Normalize(doc); err != nil {
		return nil, err
	}

	docUUID := opts.DocUUID
	if docUUID == "" {
		docUUID = deriveDocUUID(doc.Document.Name)
	}

	var docUUIDObj uuid.UUID
	if parsed, err := uuid.Parse(docUUID); err == nil {
		docUUIDObj = parsed
	} else {
		docUUIDObj = uuid.NewSHA1(compilerNamespaceUUID(), []byte(docUUID))
		docUUID = docUUIDObj.String()
	}

	pages := make([]CompiledPage, 0, len(doc.Document.Pages))
	for _, p := range doc.Document.Pages {
		pageID := p.ID
		if _, err := uuid.Parse(pageID); err != nil {
			pageID = uuid.NewSHA1(docUUIDObj, []byte(p.ID)).String()
		}
		text := compileTextBlock(p.Text, p.Canvas.Width)
		pages = append(pages, CompiledPage{
			ID:       pageID,
			Template: p.Template,
			CanvasW:  p.Canvas.Width,
			CanvasH:  p.Canvas.Height,
			Layers:   lowerPageToLayers(p),
			Text:     text,
		})
	}

	return &CompiledDoc{
		UUID:  docUUID,
		Name:  doc.Document.Name,
		Pages: pages,
	}, nil
}

func CompileToRMDoc(ctx context.Context, doc *rmdsl.Doc, outPath string, opts CompileOptions) error {
	compiled, err := Compile(ctx, doc, opts)
	if err != nil {
		return err
	}

	buf, err := buildRMDocArchive(compiled, opts)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return errors.Wrap(err, "mkdir output dir")
	}
	if err := os.WriteFile(outPath, buf, 0o644); err != nil {
		return errors.Wrap(err, "write output")
	}
	return nil
}

func buildRMDocArchive(doc *CompiledDoc, opts CompileOptions) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}

	authorID := opts.Author
	if authorID == 0 {
		authorID = 1
	}
	authorUUID := resolveAuthorUUID(opts, doc.UUID)

	canvasW, canvasH := defaultCanvasDims(doc)

	content, err := buildContentJSON(doc, now, canvasW, canvasH, authorID, authorUUID.String())
	if err != nil {
		_ = zw.Close()
		return nil, err
	}
	metadata, err := buildMetadataJSON(doc, now)
	if err != nil {
		_ = zw.Close()
		return nil, err
	}

	if err := writeZipFile(zw, fmt.Sprintf("%s.content", doc.UUID), content); err != nil {
		_ = zw.Close()
		return nil, err
	}
	if err := writeZipFile(zw, fmt.Sprintf("%s.metadata", doc.UUID), metadata); err != nil {
		_ = zw.Close()
		return nil, err
	}

	for _, p := range doc.Pages {
		structGen := newCrdtIDGen(0, 11)
		lineGen := newCrdtIDGen(authorID, 1)
		rmBytes, err := buildRMV6Page(p, structGen, lineGen, authorID, authorUUID)
		if err != nil {
			_ = zw.Close()
			return nil, err
		}
		name := fmt.Sprintf("%s/%s.rm", doc.UUID, p.ID)
		if err := writeZipFile(zw, name, rmBytes); err != nil {
			_ = zw.Close()
			return nil, err
		}
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeZipFile(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func defaultCanvasDims(doc *CompiledDoc) (int, int) {
	if len(doc.Pages) == 0 {
		return rmdsl.DefaultWidthV6, rmdsl.DefaultHeightV6
	}
	p := doc.Pages[0]
	if p.CanvasW > 0 && p.CanvasH > 0 {
		return p.CanvasW, p.CanvasH
	}
	return rmdsl.DefaultWidthV6, rmdsl.DefaultHeightV6
}

func deriveDocUUID(name string) string {
	return uuid.NewSHA1(compilerNamespaceUUID(), []byte(name)).String()
}

func compilerNamespaceUUID() uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte("remarquee-rmdsl"))
}

func resolveAuthorUUID(opts CompileOptions, docUUID string) uuid.UUID {
	if opts.AuthorUUID != "" {
		if parsed, err := uuid.Parse(opts.AuthorUUID); err == nil {
			return parsed
		}
	}
	return uuid.NewSHA1(compilerNamespaceUUID(), []byte("author:"+docUUID))
}
