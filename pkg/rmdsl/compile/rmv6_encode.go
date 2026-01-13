package compile

import (
	"bytes"
	"fmt"

	"github.com/go-go-golems/remarquee/pkg/rmdoc"
)

const (
	rmv6LineVersion        = 2
	rmv6LineItemType uint8 = 0x03
)

var rmv6DefaultTimestamp = rmdoc.RMV6CrdtID{Part1: 0, Part2: 1}

type crdtIDGen struct {
	part1 uint8
	next  uint64
}

func newCrdtIDGen(part1 uint8, start uint64) *crdtIDGen {
	if start == 0 {
		start = 1
	}
	return &crdtIDGen{part1: part1, next: start}
}

func (g *crdtIDGen) Next() rmdoc.RMV6CrdtID {
	id := rmdoc.RMV6CrdtID{Part1: g.part1, Part2: g.next}
	g.next++
	return id
}

var highlightRGBA = func() map[rmdoc.PenColor]rmdoc.RGBA {
	out := map[rmdoc.PenColor]rmdoc.RGBA{}
	for rgba, color := range rmdoc.HardcodedColorMap {
		out[color] = rgba
	}
	return out
}()

func rmv6HighlightMarker(color rmdoc.PenColor) (rmdoc.RGBA, bool) {
	rgba, ok := highlightRGBA[color]
	return rgba, ok
}

func encodeRMV6LinePayload(stroke rmdoc.Stroke) ([]byte, error) {
	var buf bytes.Buffer
	w := newRMV6Writer(&buf)

	if err := w.writeUint32(1, stroke.Tool); err != nil {
		return nil, err
	}

	color := rmdoc.PenColor(stroke.Color)
	if rgba, ok := rmv6HighlightMarker(color); ok {
		if err := w.writeUint32(2, uint32(rmdoc.PenColorHighlight)); err != nil {
			return nil, err
		}
		if err := w.writeFloat64(3, stroke.ThicknessScale); err != nil {
			return nil, err
		}
		if err := w.writeFloat32(4, float32(stroke.StartingLength)); err != nil {
			return nil, err
		}
		if err := w.writeSubBlock(5, func(sw *rmv6Writer) error {
			for _, pt := range stroke.Points {
				if err := rmv6WritePoint(sw, pt, rmv6LineVersion); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return nil, err
		}
		if err := w.writeID(6, rmv6DefaultTimestamp); err != nil {
			return nil, err
		}
		if err := w.writeBytes([]byte{0x84, 0x01, rgba.B, rgba.G, rgba.R, rgba.A}); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	if err := w.writeUint32(2, uint32(color)); err != nil {
		return nil, err
	}
	if err := w.writeFloat64(3, stroke.ThicknessScale); err != nil {
		return nil, err
	}
	if err := w.writeFloat32(4, float32(stroke.StartingLength)); err != nil {
		return nil, err
	}
	if err := w.writeSubBlock(5, func(sw *rmv6Writer) error {
		for _, pt := range stroke.Points {
			if err := rmv6WritePoint(sw, pt, rmv6LineVersion); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if err := w.writeID(6, rmv6DefaultTimestamp); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodeRMV6GlyphRange(glyph CompiledGlyph) ([]byte, error) {
	var buf bytes.Buffer
	w := newRMV6Writer(&buf)

	if glyph.Start != nil {
		if err := w.writeUint32(2, *glyph.Start); err != nil {
			return nil, err
		}
	}
	if glyph.Length != nil {
		if err := w.writeUint32(3, *glyph.Length); err != nil {
			return nil, err
		}
	}

	color := glyph.Color
	if rgba, ok := rmv6HighlightMarker(color); ok {
		if err := w.writeUint32(4, uint32(rmdoc.PenColorHighlight)); err != nil {
			return nil, err
		}
		if err := w.writeString(5, glyph.Text); err != nil {
			return nil, err
		}
		if err := w.writeSubBlock(6, func(sw *rmv6Writer) error {
			if err := sw.writeVarUint(uint64(len(glyph.Rects))); err != nil {
				return err
			}
			for _, r := range glyph.Rects {
				if err := sw.writeFloat64LE(r.X); err != nil {
					return err
				}
				if err := sw.writeFloat64LE(r.Y); err != nil {
					return err
				}
				if err := sw.writeFloat64LE(r.W); err != nil {
					return err
				}
				if err := sw.writeFloat64LE(r.H); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return nil, err
		}
		if err := w.writeBytes([]byte{0xA4, 0x01, rgba.B, rgba.G, rgba.R, rgba.A}); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	if err := w.writeUint32(4, uint32(color)); err != nil {
		return nil, err
	}
	if err := w.writeString(5, glyph.Text); err != nil {
		return nil, err
	}
	if err := w.writeSubBlock(6, func(sw *rmv6Writer) error {
		if err := sw.writeVarUint(uint64(len(glyph.Rects))); err != nil {
			return err
		}
		for _, r := range glyph.Rects {
			if err := sw.writeFloat64LE(r.X); err != nil {
				return err
			}
			if err := sw.writeFloat64LE(r.Y); err != nil {
				return err
			}
			if err := sw.writeFloat64LE(r.W); err != nil {
				return err
			}
			if err := sw.writeFloat64LE(r.H); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func rmv6WritePoint(w *rmv6Writer, pt rmdoc.StrokePoint, version uint8) error {
	if err := w.writeFloat32LE(float32(pt.X)); err != nil {
		return err
	}
	if err := w.writeFloat32LE(float32(pt.Y)); err != nil {
		return err
	}
	switch version {
	case 1:
		return fmt.Errorf("rmv6 point version 1 not supported")
	case 2:
		if err := w.writeUint16LE(uint16(pt.Speed)); err != nil {
			return err
		}
		if err := w.writeUint16LE(uint16(pt.Width)); err != nil {
			return err
		}
		if err := w.writeUint8(uint8(pt.Direction)); err != nil {
			return err
		}
		return w.writeUint8(uint8(pt.Pressure))
	default:
		return fmt.Errorf("unsupported rmv6 point version %d", version)
	}
}
