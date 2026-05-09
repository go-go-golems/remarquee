package compile

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/go-go-golems/remarquee/pkg/rmdoc"
)

type rmv6TagType uint8

const (
	rmv6TagTypeID      rmv6TagType = 0xF
	rmv6TagTypeLength4 rmv6TagType = 0xC
	rmv6TagTypeByte8   rmv6TagType = 0x8
	rmv6TagTypeByte4   rmv6TagType = 0x4
	rmv6TagTypeByte1   rmv6TagType = 0x1
)

type rmv6Writer struct {
	w io.Writer
}

func newRMV6Writer(w io.Writer) *rmv6Writer {
	return &rmv6Writer{w: w}
}

func (w *rmv6Writer) writeHeader() error {
	_, err := w.w.Write(rmV6HeaderBytes())
	return err
}

func (w *rmv6Writer) writeTag(index uint32, tagType rmv6TagType) error {
	x := (uint64(index) << 4) | uint64(tagType)
	return w.writeVarUint(x)
}

func (w *rmv6Writer) writeVarUint(v uint64) error {
	for {
		b := byte(v & 0x7F)
		v >>= 7
		if v == 0 {
			if _, err := w.w.Write([]byte{b}); err != nil {
				return err
			}
			break
		}
		if _, err := w.w.Write([]byte{b | 0x80}); err != nil {
			return err
		}
	}
	return nil
}

func (w *rmv6Writer) writeUint8(v uint8) error {
	_, err := w.w.Write([]byte{v})
	return err
}

func (w *rmv6Writer) writeUint16LE(v uint16) error {
	return binary.Write(w.w, binary.LittleEndian, v)
}

func (w *rmv6Writer) writeUint32LE(v uint32) error {
	return binary.Write(w.w, binary.LittleEndian, v)
}

func (w *rmv6Writer) writeFloat32LE(v float32) error {
	return binary.Write(w.w, binary.LittleEndian, v)
}

func (w *rmv6Writer) writeFloat64LE(v float64) error {
	return binary.Write(w.w, binary.LittleEndian, v)
}

func (w *rmv6Writer) writeBytes(b []byte) error {
	_, err := w.w.Write(b)
	return err
}

func (w *rmv6Writer) writeIntPair(index uint32, a uint32, b uint32) error {
	return w.writeSubBlock(index, func(sw *rmv6Writer) error {
		if err := sw.writeUint32LE(a); err != nil {
			return err
		}
		return sw.writeUint32LE(b)
	})
}

func (w *rmv6Writer) writeID(index uint32, id rmdoc.RMV6CrdtID) error {
	if err := w.writeTag(index, rmv6TagTypeID); err != nil {
		return err
	}
	return w.writeCrdtIDRaw(id)
}

func (w *rmv6Writer) writeCrdtIDRaw(id rmdoc.RMV6CrdtID) error {
	if err := w.writeUint8(id.Part1); err != nil {
		return err
	}
	return w.writeVarUint(id.Part2)
}

func (w *rmv6Writer) writeBool(index uint32, v bool) error {
	if err := w.writeTag(index, rmv6TagTypeByte1); err != nil {
		return err
	}
	if v {
		return w.writeUint8(1)
	}
	return w.writeUint8(0)
}

func (w *rmv6Writer) writeByte(index uint32, v uint8) error {
	if err := w.writeTag(index, rmv6TagTypeByte1); err != nil {
		return err
	}
	return w.writeUint8(v)
}

func (w *rmv6Writer) writeUint32(index uint32, v uint32) error {
	if err := w.writeTag(index, rmv6TagTypeByte4); err != nil {
		return err
	}
	return w.writeUint32LE(v)
}

func (w *rmv6Writer) writeFloat32(index uint32, v float32) error {
	if err := w.writeTag(index, rmv6TagTypeByte4); err != nil {
		return err
	}
	return w.writeFloat32LE(v)
}

func (w *rmv6Writer) writeFloat64(index uint32, v float64) error {
	if err := w.writeTag(index, rmv6TagTypeByte8); err != nil {
		return err
	}
	return w.writeFloat64LE(v)
}

func intToUint32(v int) (uint32, error) {
	if v < 0 || v > math.MaxUint32 {
		return 0, fmt.Errorf("value %d exceeds uint32", v)
	}
	return uint32(v), nil // #nosec G115 -- v is range-checked above.
}

func (w *rmv6Writer) writeSubBlock(index uint32, fn func(sw *rmv6Writer) error) error {
	var buf bytes.Buffer
	sw := newRMV6Writer(&buf)
	if err := fn(sw); err != nil {
		return err
	}
	if err := w.writeTag(index, rmv6TagTypeLength4); err != nil {
		return err
	}
	bufLen, err := intToUint32(buf.Len())
	if err != nil {
		return err
	}
	if err := w.writeUint32LE(bufLen); err != nil {
		return err
	}
	return w.writeBytes(buf.Bytes())
}

func (w *rmv6Writer) writeString(index uint32, value string) error {
	return w.writeSubBlock(index, func(sw *rmv6Writer) error {
		b := []byte(value)
		if err := sw.writeVarUint(uint64(len(b))); err != nil {
			return err
		}
		if err := sw.writeUint8(1); err != nil {
			return err
		}
		return sw.writeBytes(b)
	})
}

func (w *rmv6Writer) writeStringWithFormat(index uint32, value string, format *uint32) error {
	return w.writeSubBlock(index, func(sw *rmv6Writer) error {
		b := []byte(value)
		if err := sw.writeVarUint(uint64(len(b))); err != nil {
			return err
		}
		if err := sw.writeUint8(1); err != nil {
			return err
		}
		if err := sw.writeBytes(b); err != nil {
			return err
		}
		if format != nil {
			return sw.writeUint32(2, *format)
		}
		return nil
	})
}

func (w *rmv6Writer) writeLWWBool(index uint32, ts rmdoc.RMV6CrdtID, value bool) error {
	return w.writeSubBlock(index, func(sw *rmv6Writer) error {
		if err := sw.writeID(1, ts); err != nil {
			return err
		}
		if value {
			return sw.writeByte(2, 1)
		}
		return sw.writeByte(2, 0)
	})
}

func (w *rmv6Writer) writeLWWID(index uint32, ts rmdoc.RMV6CrdtID, value rmdoc.RMV6CrdtID) error {
	return w.writeSubBlock(index, func(sw *rmv6Writer) error {
		if err := sw.writeID(1, ts); err != nil {
			return err
		}
		return sw.writeID(2, value)
	})
}

func (w *rmv6Writer) writeLWWString(index uint32, ts rmdoc.RMV6CrdtID, value string) error {
	return w.writeSubBlock(index, func(sw *rmv6Writer) error {
		if err := sw.writeID(1, ts); err != nil {
			return err
		}
		return sw.writeString(2, value)
	})
}

func (w *rmv6Writer) writeBlock(blockType uint8, minVersion uint8, currentVersion uint8, fn func(bw *rmv6Writer) error) error {
	var buf bytes.Buffer
	bw := newRMV6Writer(&buf)
	if err := fn(bw); err != nil {
		return err
	}
	bufLen, err := intToUint32(buf.Len())
	if err != nil {
		return err
	}
	if err := w.writeUint32LE(bufLen); err != nil {
		return err
	}
	if err := w.writeUint8(0); err != nil {
		return err
	}
	if err := w.writeUint8(minVersion); err != nil {
		return err
	}
	if err := w.writeUint8(currentVersion); err != nil {
		return err
	}
	if err := w.writeUint8(blockType); err != nil {
		return err
	}
	return w.writeBytes(buf.Bytes())
}

func rmV6HeaderBytes() []byte {
	// Keep in sync with rmdoc.rmV6Header; duplicated here to avoid package export.
	return []byte("reMarkable .lines file, version=6          ")
}
