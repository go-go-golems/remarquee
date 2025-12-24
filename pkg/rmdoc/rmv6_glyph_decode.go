package rmdoc

import (
	"bytes"
	"io"

	"github.com/pkg/errors"
)

type RMV6Rectangle struct {
	X float64
	Y float64
	W float64
	H float64
}

type RMV6GlyphRange struct {
	Start  *uint32
	Length *uint32
	Text   string

	Color PenColor

	Rectangles []RMV6Rectangle
}

// DecodeRMV6GlyphRange decodes the value payload of a SceneGlyphItemBlock (after the item_type byte)
// into a GlyphRange with rectangles suitable for smart highlight annotations.
//
// It mirrors rmscene.scene_stream.glyph_range_from_stream.
func DecodeRMV6GlyphRange(payload []byte) (*RMV6GlyphRange, error) {
	vr := newRMV6ValueReader(bytes.NewReader(payload), int64(len(payload)))

	var start *uint32
	if ok, _ := vr.checkTag(2, rmV6TagTypeByte4); ok {
		v, err := vr.readUint32(2)
		if err != nil {
			return nil, errors.Wrap(err, "read start")
		}
		start = &v
	}

	var length *uint32
	if ok, _ := vr.checkTag(3, rmV6TagTypeByte4); ok {
		v, err := vr.readUint32(3)
		if err != nil {
			return nil, errors.Wrap(err, "read length")
		}
		length = &v
	}

	colorRaw, err := vr.readUint32(4)
	if err != nil {
		return nil, errors.Wrap(err, "read color")
	}

	text, err := vr.readString(5)
	if err != nil {
		return nil, errors.Wrap(err, "read text")
	}

	sb, err := vr.readSubBlock(6)
	if err != nil {
		return nil, errors.Wrap(err, "read rectangles subblock")
	}

	numRects, err := vr.readVarUint()
	if err != nil {
		_ = vr.endSubBlock(sb)
		return nil, errors.Wrap(err, "read rectangles count")
	}

	rects := make([]RMV6Rectangle, 0, numRects)
	for i := 0; i < int(numRects); i++ {
		x, err := vr.readFloat64LE()
		if err != nil {
			_ = vr.endSubBlock(sb)
			return nil, err
		}
		y, err := vr.readFloat64LE()
		if err != nil {
			_ = vr.endSubBlock(sb)
			return nil, err
		}
		w, err := vr.readFloat64LE()
		if err != nil {
			_ = vr.endSubBlock(sb)
			return nil, err
		}
		h, err := vr.readFloat64LE()
		if err != nil {
			_ = vr.endSubBlock(sb)
			return nil, err
		}
		rects = append(rects, RMV6Rectangle{X: x, Y: y, W: w, H: h})
	}

	if err := vr.endSubBlock(sb); err != nil {
		return nil, err
	}

	color := PenColor(colorRaw)

	// Trailing highlight/shader RGBA marker: 2 bytes prefix + 4 bytes (b,g,r,a).
	// In python this is b"\xa4\x01"; in some line payloads we've also seen 0x84 0x01.
	rem, _ := vr.bytesRemaining()
	if rem >= 6 {
		b6, err := vr.peekBytes(6)
		if err == nil && len(b6) == 6 && (b6[0] == 0xA4 || b6[0] == 0x84) && b6[1] == 0x01 {
			_, _ = vr.readBytes(2) // prefix
			b, _ := vr.readUint8()
			g, _ := vr.readUint8()
			r, _ := vr.readUint8()
			a, _ := vr.readUint8()
			if c, ok := HardcodedColorMap[RGBA{R: r, G: g, B: b, A: a}]; ok {
				color = c
			} else {
				// rewind -6, mirroring python behavior
				_, _ = vr.rs.Seek(-6, io.SeekCurrent)
			}
		}
	}

	return &RMV6GlyphRange{
		Start:      start,
		Length:     length,
		Text:       text,
		Color:      color,
		Rectangles: rects,
	}, nil
}
