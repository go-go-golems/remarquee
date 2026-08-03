package render

// Minimal stdlib-only PDF writer for the annotations-only render path (design
// doc DR-6, ticket RMQ-0021). It exists so that annotations-only output does
// not depend on unipdf (AGPL) or on rmapi's go:linkname community-license init
// to avoid watermarking. Only the annotations-only path uses this writer; the
// composite merge pipeline still uses unipdf because it must read and composite
// existing PDF pages, which is out of scope to replace.
//
// The writer supports exactly what the annotations-only renderer emits:
//   - pages with MediaBox/CropBox
//   - one FlateDecode content stream per page (stroke/text operators)
//   - inline ExtGState (alpha) and base-14 font resource dicts
//   - indirect Highlight annotation dicts with QuadPoints
//   - a classic xref table + trailer

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"strconv"
	"strings"
)

// pdfHighlightAnnot is a Highlight text markup annotation (PDF spec 12.5.6.10).
type pdfHighlightAnnot struct {
	r, g, b float64
	ca      float64
	rect    [4]float64 // llx lly urx ury
	quads   []float64  // 8 numbers per quad: x1 y1 x2 y2 x3 y3 x4 y4
}

// pdfPageSpec describes one output page for writeSimplePDF.
type pdfPageSpec struct {
	width, height float64
	content       string    // content stream operators; empty means a blank page
	alphas        []float64 // ExtGState alphas referenced by content
	needsFonts    bool      // whether content references the typed-text fonts
	annots        []pdfHighlightAnnot
}

// pdfNum formats a float as a PDF number with up to 4 decimals, trimming
// trailing zeros for compact output.
func pdfNum(v float64) string {
	s := strconv.FormatFloat(v, 'f', 4, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	if s == "" || s == "-0" {
		return "0"
	}
	return s
}

// pdfEscapeString escapes a Go string for use as a PDF literal string.
// Non-ASCII/control bytes are emitted as octal escapes (base-14 fonts are
// byte-oriented; this mirrors what unipdf's MakeString did with raw bytes).
func pdfEscapeString(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\', '(', ')':
			b.WriteByte('\\')
			b.WriteByte(c)
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		case '\t':
			b.WriteString("\\t")
		default:
			if c < 0x20 || c > 0x7e {
				fmt.Fprintf(&b, "\\%03o", c)
			} else {
				b.WriteByte(c)
			}
		}
	}
	return b.String()
}

// alphaGStateNameStr mirrors alphaGStateName without depending on unipdf types.
func alphaGStateNameStr(alpha float64) string {
	return fmt.Sprintf("RMQGsA%03d", alphaKey(alpha))
}

func buildResourcesDict(alphas []float64, needsFonts bool) string {
	var parts []string
	if len(alphas) > 0 {
		var gs strings.Builder
		gs.WriteString("/ExtGState <<")
		for _, a := range alphas {
			fmt.Fprintf(&gs, " /%s << /CA %s /ca %s >>", alphaGStateNameStr(a), pdfNum(clamp01(a)), pdfNum(clamp01(a)))
		}
		gs.WriteString(" >>")
		parts = append(parts, gs.String())
	}
	if needsFonts {
		parts = append(parts, "/Font <<"+
			" /RMQTxtPlain << /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"+
			" /RMQTxtBold << /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold >>"+
			" /RMQTxtHeading << /Type /Font /Subtype /Type1 /BaseFont /Times-Roman >>"+
			" >>")
	}
	if len(parts) == 0 {
		return ""
	}
	return "<< " + strings.Join(parts, " ") + " >>"
}

// flateStream compresses content with zlib (RFC 1950), which is what the
// PDF /FlateDecode filter expects (a raw compress/flate stream would be
// missing the zlib header and checksum).
func flateStream(content string) ([]byte, error) {
	var cb bytes.Buffer
	zw := zlib.NewWriter(&cb)
	if _, err := zw.Write([]byte(content)); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}

	var b bytes.Buffer
	fmt.Fprintf(&b, "<< /Length %d /Filter /FlateDecode >>\nstream\n", cb.Len())
	b.Write(cb.Bytes())
	b.WriteString("\nendstream")
	return b.Bytes(), nil
}

func highlightAnnotDict(a pdfHighlightAnnot) string {
	quads := make([]string, 0, len(a.quads))
	for _, v := range a.quads {
		quads = append(quads, pdfNum(v))
	}
	rect := make([]string, 0, 4)
	for _, v := range a.rect {
		rect = append(rect, pdfNum(v))
	}
	return fmt.Sprintf("<< /Type /Annot /Subtype /Highlight /C [%s %s %s] /CA %s /Rect [%s] /QuadPoints [%s] >>",
		pdfNum(a.r), pdfNum(a.g), pdfNum(a.b), pdfNum(a.ca),
		strings.Join(rect, " "), strings.Join(quads, " "))
}

// writeSimplePDF serializes pages into a complete, valid PDF document.
// Object layout: 1 = Catalog, 2 = Pages tree, then per page: page object,
// optional content stream object, optional annotation objects.
func writeSimplePDF(pages []pdfPageSpec) ([]byte, error) {
	type pageRefs struct {
		pageID    int
		contentID int
		annotIDs  []int
	}

	refs := make([]pageRefs, len(pages))
	kids := make([]string, 0, len(pages))
	nextID := 3
	for i, p := range pages {
		r := pageRefs{pageID: nextID}
		nextID++
		if p.content != "" {
			r.contentID = nextID
			nextID++
		}
		for range p.annots {
			r.annotIDs = append(r.annotIDs, nextID)
			nextID++
		}
		refs[i] = r
		kids = append(kids, fmt.Sprintf("%d 0 R", r.pageID))
	}

	objects := make(map[int][]byte, nextID-1)
	objects[1] = []byte("<< /Type /Catalog /Pages 2 0 R >>")
	objects[2] = []byte(fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), len(kids)))

	for i, p := range pages {
		r := refs[i]

		var b strings.Builder
		fmt.Fprintf(&b, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %s %s] /CropBox [0 0 %s %s]",
			pdfNum(p.width), pdfNum(p.height), pdfNum(p.width), pdfNum(p.height))
		if res := buildResourcesDict(p.alphas, p.needsFonts); res != "" {
			b.WriteString(" /Resources " + res)
		}
		if len(r.annotIDs) > 0 {
			ids := make([]string, 0, len(r.annotIDs))
			for _, id := range r.annotIDs {
				ids = append(ids, fmt.Sprintf("%d 0 R", id))
			}
			b.WriteString(" /Annots [" + strings.Join(ids, " ") + "]")
		}
		if r.contentID > 0 {
			fmt.Fprintf(&b, " /Contents %d 0 R", r.contentID)
		}
		b.WriteString(" >>")
		objects[r.pageID] = []byte(b.String())

		if r.contentID > 0 {
			stream, err := flateStream(p.content)
			if err != nil {
				return nil, err
			}
			objects[r.contentID] = stream
		}

		for j, a := range p.annots {
			objects[r.annotIDs[j]] = []byte(highlightAnnotDict(a))
		}
	}

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.7\n%\xe2\xe3\xcf\xd3\n")

	offsets := make([]int, nextID)
	for id := 1; id < nextID; id++ {
		body, ok := objects[id]
		if !ok {
			return nil, fmt.Errorf("internal error: missing pdf object %d", id)
		}
		offsets[id] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n", id)
		buf.Write(body)
		buf.WriteString("\nendobj\n")
	}

	xrefPos := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", nextID)
	buf.WriteString("0000000000 65535 f \n")
	for id := 1; id < nextID; id++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[id])
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", nextID, xrefPos)

	return buf.Bytes(), nil
}
