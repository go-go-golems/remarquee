package compile

import (
	"archive/zip"
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/remarquee/pkg/rmdoc"
	"github.com/go-go-golems/remarquee/pkg/rmdoc/render"
	"github.com/go-go-golems/remarquee/pkg/rmdsl"
)

func TestCompileRMDoc_OpenAndParse(t *testing.T) {
	t.Parallel()

	casePath := filepath.Join("..", "..", "..", "ttmp", "2025", "12", "24", "RMQ-0006--rmdoc-rendering-testing-validation-golden-runner", "cases", "01-ellipse-at-bottom.yaml")
	doc, err := rmdsl.LoadFromFile(context.Background(), casePath, rmdsl.LoadOptions{})
	if err != nil {
		t.Fatalf("load case: %v", err)
	}

	compiled, err := Compile(context.Background(), doc, CompileOptions{})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	archiveBytes, err := buildRMDocArchive(compiled, CompileOptions{})
	if err != nil {
		t.Fatalf("build archive: %v", err)
	}

	reader := bytes.NewReader(archiveBytes)
	docParsed, err := rmdoc.OpenReaderAt(context.Background(), reader, int64(len(archiveBytes)))
	if err != nil {
		t.Fatalf("open rmdoc: %v", err)
	}
	if docParsed.Schema != rmdoc.SchemaCPages {
		t.Fatalf("schema=%v want %v", docParsed.Schema, rmdoc.SchemaCPages)
	}
	if len(docParsed.Pages) != 1 {
		t.Fatalf("pages=%d want 1", len(docParsed.Pages))
	}

	rmBytes := readRMFromArchive(t, archiveBytes, docParsed.UUID, docParsed.Pages[0].PageID)
	tree, err := rmdoc.ParseRMV6SceneTree(bytes.NewReader(rmBytes))
	if err != nil {
		t.Fatalf("parse rmv6: %v", err)
	}
	strokes, err := rmdoc.ExtractRMV6StrokesWithAnchors(tree)
	if err != nil {
		t.Fatalf("extract strokes: %v", err)
	}
	if len(strokes) != 4 {
		t.Fatalf("strokes=%d want 4", len(strokes))
	}

	if _, err := render.RenderRMV6RMToPDFBytes(context.Background(), bytes.NewReader(rmBytes)); err != nil {
		t.Fatalf("render pdf: %v", err)
	}
}

func TestEncodeRMV6LinePayload_RoundTrip(t *testing.T) {
	t.Parallel()

	stroke := rmdoc.Stroke{
		Tool:           17,
		Color:          uint32(rmdoc.PenColorRed),
		ThicknessScale: 1.0,
		StartingLength: 0,
		Points: []rmdoc.StrokePoint{
			{X: -10, Y: 20, Speed: 0, Direction: 0, Width: 120, Pressure: 96},
			{X: 30, Y: 40, Speed: 0, Direction: 0, Width: 120, Pressure: 96},
		},
	}

	payload, err := encodeRMV6LinePayload(stroke)
	if err != nil {
		t.Fatalf("encode line payload: %v", err)
	}
	decoded, err := rmdoc.DecodeRMV6Line(rmv6LineVersion, payload)
	if err != nil {
		t.Fatalf("decode line payload: %v", err)
	}
	if decoded.Tool != stroke.Tool {
		t.Fatalf("tool=%d want %d", decoded.Tool, stroke.Tool)
	}
	if decoded.Color != stroke.Color {
		t.Fatalf("color=%d want %d", decoded.Color, stroke.Color)
	}
	if len(decoded.Points) != len(stroke.Points) {
		t.Fatalf("points=%d want %d", len(decoded.Points), len(stroke.Points))
	}
	if len(decoded.Points) > 0 && (decoded.Points[0].X != stroke.Points[0].X || decoded.Points[0].Y != stroke.Points[0].Y) {
		t.Fatalf("first point mismatch: got (%v,%v) want (%v,%v)", decoded.Points[0].X, decoded.Points[0].Y, stroke.Points[0].X, stroke.Points[0].Y)
	}
}

func readRMFromArchive(t *testing.T, archive []byte, docID string, pageID string) []byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatalf("zip reader: %v", err)
	}
	target := filepath.ToSlash(filepath.Join(docID, pageID+".rm"))
	for _, f := range zr.File {
		if f.Name != target {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open rm file: %v", err)
		}
		defer func() { _ = rc.Close() }()
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(rc); err != nil {
			t.Fatalf("read rm file: %v", err)
		}
		return buf.Bytes()
	}
	t.Fatalf("rm file not found: %s", target)
	return nil
}
