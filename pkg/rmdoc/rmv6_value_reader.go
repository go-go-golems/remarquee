package rmdoc

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"

	"github.com/pkg/errors"
)

// rmV6ValueReader is a small helper for parsing tagged values inside a V6 block or subblock.
//
// Unlike rmV6TaggedBlockReader, this reader does not understand top-level blocks. It only
// enforces nested length limits (subblocks) and supports reading:
// - tags (varuint index/type),
// - primitive typed values,
// - Length4 subblocks.
//
// This is the building block for decoding SceneLineItemBlock payloads (task 36).
type rmV6ValueReader struct {
	rs io.ReadSeeker

	// limitStack holds absolute end offsets; the active limit is the last element.
	limitStack []int64
}

func newRMV6ValueReader(rs io.ReadSeeker, size int64) *rmV6ValueReader {
	return &rmV6ValueReader{
		rs:         rs,
		limitStack: []int64{size},
	}
}

func (r *rmV6ValueReader) tell() (int64, error) {
	off, err := r.rs.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, errors.Wrap(err, "seek current")
	}
	return off, nil
}

func (r *rmV6ValueReader) limitEnd() int64 {
	return r.limitStack[len(r.limitStack)-1]
}

func (r *rmV6ValueReader) bytesRemaining() (int64, error) {
	pos, err := r.tell()
	if err != nil {
		return 0, err
	}
	return r.limitEnd() - pos, nil
}

func (r *rmV6ValueReader) readBytes(n int) ([]byte, error) {
	rem, err := r.bytesRemaining()
	if err != nil {
		return nil, err
	}
	if int64(n) > rem {
		return nil, io.ErrUnexpectedEOF
	}
	buf := make([]byte, n)
	_, err = io.ReadFull(r.rs, buf)
	if err != nil {
		return nil, errors.Wrap(err, "read bytes")
	}
	return buf, nil
}

func (r *rmV6ValueReader) peekBytes(n int) ([]byte, error) {
	pos, err := r.tell()
	if err != nil {
		return nil, err
	}
	defer func() { _, _ = r.rs.Seek(pos, io.SeekStart) }()
	return r.readBytes(n)
}

func (r *rmV6ValueReader) readUint8() (uint8, error) {
	b, err := r.readBytes(1)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

func (r *rmV6ValueReader) readUint16LE() (uint16, error) {
	b, err := r.readBytes(2)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(b), nil
}

func (r *rmV6ValueReader) readUint32LE() (uint32, error) {
	b, err := r.readBytes(4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(b), nil
}

func (r *rmV6ValueReader) readFloat32LE() (float32, error) {
	u, err := r.readUint32LE()
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(u), nil
}

func (r *rmV6ValueReader) readFloat64LE() (float64, error) {
	b, err := r.readBytes(8)
	if err != nil {
		return 0, err
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(b)), nil
}

func (r *rmV6ValueReader) readVarUint() (uint64, error) {
	var (
		shift uint
		out   uint64
	)
	for {
		b, err := r.readUint8()
		if err != nil {
			return 0, err
		}
		out |= uint64(b&0x7F) << shift
		if (b & 0x80) == 0 {
			break
		}
		shift += 7
		if shift > 63 {
			return 0, errors.Errorf("varuint too large (shift=%d)", shift)
		}
	}
	return out, nil
}

func (r *rmV6ValueReader) readTag() (uint32, rmV6TagType, error) {
	x, err := r.readVarUint()
	if err != nil {
		return 0, 0, err
	}
	index64 := x >> 4
	if index64 > math.MaxUint32 {
		return 0, 0, errors.Errorf("rm v6 tag index %d exceeds uint32", index64)
	}
	index := uint32(index64)
	tagType := rmV6TagType(uint8(x & 0xF))
	switch tagType {
	case rmV6TagTypeID, rmV6TagTypeLength4, rmV6TagTypeByte8, rmV6TagTypeByte4, rmV6TagTypeByte1:
		return index, tagType, nil
	default:
		return 0, 0, errors.Errorf("unknown rm v6 tag type 0x%X (index=%d)", uint8(tagType), index)
	}
}

func (r *rmV6ValueReader) peekTag() (uint32, rmV6TagType, bool, error) {
	pos, err := r.tell()
	if err != nil {
		return 0, 0, false, err
	}
	defer func() { _, _ = r.rs.Seek(pos, io.SeekStart) }()

	idx, tt, err := r.readTag()
	if err != nil {
		return 0, 0, false, nil
	}
	return idx, tt, true, nil
}

func (r *rmV6ValueReader) checkTag(expectedIndex uint32, expectedType rmV6TagType) (bool, error) {
	idx, tt, ok, err := r.peekTag()
	if err != nil || !ok {
		return false, err
	}
	return idx == expectedIndex && tt == expectedType, nil
}

func (r *rmV6ValueReader) readExpectedTag(expectedIndex uint32, expectedType rmV6TagType) error {
	pos, err := r.tell()
	if err != nil {
		return err
	}
	idx, tt, err := r.readTag()
	if err != nil {
		return err
	}
	if idx != expectedIndex || tt != expectedType {
		_, _ = r.rs.Seek(pos, io.SeekStart)
		return errors.Errorf("unexpected tag: got (index=%d type=0x%X) want (index=%d type=0x%X)", idx, uint8(tt), expectedIndex, uint8(expectedType))
	}
	return nil
}

func (r *rmV6ValueReader) readCrdtIDRaw() (RMV6CrdtID, error) {
	p1, err := r.readUint8()
	if err != nil {
		return RMV6CrdtID{}, err
	}
	p2, err := r.readVarUint()
	if err != nil {
		return RMV6CrdtID{}, err
	}
	return RMV6CrdtID{Part1: p1, Part2: p2}, nil
}

func (r *rmV6ValueReader) readID(index uint32) (RMV6CrdtID, error) {
	if err := r.readExpectedTag(index, rmV6TagTypeID); err != nil {
		return RMV6CrdtID{}, err
	}
	return r.readCrdtIDRaw()
}

func (r *rmV6ValueReader) readUint32(index uint32) (uint32, error) {
	if err := r.readExpectedTag(index, rmV6TagTypeByte4); err != nil {
		return 0, err
	}
	return r.readUint32LE()
}

func (r *rmV6ValueReader) readFloat32(index uint32) (float32, error) {
	if err := r.readExpectedTag(index, rmV6TagTypeByte4); err != nil {
		return 0, err
	}
	return r.readFloat32LE()
}

func (r *rmV6ValueReader) readFloat64(index uint32) (float64, error) {
	if err := r.readExpectedTag(index, rmV6TagTypeByte8); err != nil {
		return 0, err
	}
	return r.readFloat64LE()
}

func (r *rmV6ValueReader) readString(index uint32) (string, error) {
	sb, err := r.readSubBlock(index)
	if err != nil {
		return "", err
	}
	defer func() { _ = r.endSubBlock(sb) }()

	strLen, err := r.readVarUint()
	if err != nil {
		return "", err
	}
	isASCII, err := r.readUint8()
	if err != nil {
		return "", err
	}
	if isASCII == 0 {
		return "", errors.New("string is_ascii flag is false (unexpected)")
	}
	if strLen > uint64(math.MaxInt) {
		return "", errors.Errorf("string length %d exceeds max int", strLen)
	}
	if strLen > uint64(sb.Size) {
		return "", errors.Errorf("string length %d exceeds subblock size %d", strLen, sb.Size)
	}
	b, err := r.readBytes(int(strLen))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

type rmV6ValueSubBlock struct {
	Offset int64
	Size   uint32

	ExtraData []byte
}

func (r *rmV6ValueReader) readSubBlock(expectedIndex uint32) (*rmV6ValueSubBlock, error) {
	if err := r.readExpectedTag(expectedIndex, rmV6TagTypeLength4); err != nil {
		return nil, err
	}
	subLen, err := r.readUint32LE()
	if err != nil {
		return nil, err
	}
	off, err := r.tell()
	if err != nil {
		return nil, err
	}
	end := off + int64(subLen)
	r.limitStack = append(r.limitStack, end)
	return &rmV6ValueSubBlock{Offset: off, Size: subLen}, nil
}

func (r *rmV6ValueReader) endSubBlock(sb *rmV6ValueSubBlock) error {
	pos, err := r.tell()
	if err != nil {
		return err
	}
	end := sb.Offset + int64(sb.Size)
	if pos > end {
		return errors.Errorf("read past end of v6 subblock (pos=%d end=%d overflow=%d)", pos, end, pos-end)
	}
	if pos < end {
		remaining := end - pos
		extra, err := r.readBytes(int(remaining))
		if err != nil {
			return err
		}
		sb.ExtraData = bytes.Clone(extra)
	}
	// pop limit
	r.limitStack = r.limitStack[:len(r.limitStack)-1]
	return nil
}
