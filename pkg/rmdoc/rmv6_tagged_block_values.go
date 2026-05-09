package rmdoc

import (
	"encoding/binary"
	"io"
	"math"

	"github.com/pkg/errors"
)

type RMV6LWW[T any] struct {
	Timestamp RMV6CrdtID
	Value     T
}

// checkTag returns true if the next tag matches (expectedIndex, expectedType) without consuming it.
func (r *rmV6TaggedBlockReader) checkTag(expectedIndex uint32, expectedType rmV6TagType) (bool, error) {
	idx, tt, ok, err := r.peekTag()
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	return idx == expectedIndex && tt == expectedType, nil
}

// readExpectedTag reads a tag and verifies (expectedIndex, expectedType); on mismatch rewinds the stream.
func (r *rmV6TaggedBlockReader) readExpectedTag(expectedIndex uint32, expectedType rmV6TagType) error {
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

func (r *rmV6TaggedBlockReader) readBoolRaw() (bool, error) {
	b, err := r.readUint8()
	if err != nil {
		return false, err
	}
	return b != 0, nil
}

func (r *rmV6TaggedBlockReader) readFloat32Raw() (float32, error) {
	b, err := r.readBytes(4)
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(binary.LittleEndian.Uint32(b)), nil
}

func (r *rmV6TaggedBlockReader) readFloat64Raw() (float64, error) {
	b, err := r.readBytes(8)
	if err != nil {
		return 0, err
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(b)), nil
}

func (r *rmV6TaggedBlockReader) readCrdtIDRaw() (RMV6CrdtID, error) {
	// Mirrors rmscene DataStream.read_crdt_id: part1:uint8, part2:varuint
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

func (r *rmV6TaggedBlockReader) readID(index uint32) (RMV6CrdtID, error) {
	if err := r.readExpectedTag(index, rmV6TagTypeID); err != nil {
		return RMV6CrdtID{}, err
	}
	return r.readCrdtIDRaw()
}

func (r *rmV6TaggedBlockReader) readByte(index uint32) (uint8, error) {
	if err := r.readExpectedTag(index, rmV6TagTypeByte1); err != nil {
		return 0, err
	}
	return r.readUint8()
}

func (r *rmV6TaggedBlockReader) readBool(index uint32) (bool, error) {
	if err := r.readExpectedTag(index, rmV6TagTypeByte1); err != nil {
		return false, err
	}
	return r.readBoolRaw()
}

func (r *rmV6TaggedBlockReader) readUint32(index uint32) (uint32, error) {
	if err := r.readExpectedTag(index, rmV6TagTypeByte4); err != nil {
		return 0, err
	}
	return r.readUint32LE()
}

func (r *rmV6TaggedBlockReader) readFloat32(index uint32) (float32, error) {
	if err := r.readExpectedTag(index, rmV6TagTypeByte4); err != nil {
		return 0, err
	}
	return r.readFloat32Raw()
}

func (r *rmV6TaggedBlockReader) hasSubBlock(index uint32) (bool, error) {
	// Matches rmscene has_subblock: explicit end-of-block guard.
	if r.current != nil {
		rem, err := r.bytesRemainingInBlock()
		if err != nil {
			return false, err
		}
		if rem <= 0 {
			return false, nil
		}
	}
	return r.checkTag(index, rmV6TagTypeLength4)
}

func (r *rmV6TaggedBlockReader) readString(index uint32) (string, error) {
	sb, err := r.readSubBlock(index)
	if err != nil {
		return "", err
	}
	defer func() { _ = r.discardSubBlockRemainder(sb) }()

	strLen, err := r.readVarUint()
	if err != nil {
		return "", err
	}
	isASCII, err := r.readBoolRaw()
	if err != nil {
		return "", err
	}
	if !isASCII {
		return "", errors.New("string is_ascii flag is false (unexpected)")
	}

	// 1 byte for ascii flag + varuint length header are part of subblock; ensure claimed length fits.
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

func (r *rmV6TaggedBlockReader) readStringWithFormat(index uint32) (string, *uint32, error) {
	sb, err := r.readSubBlock(index)
	if err != nil {
		return "", nil, err
	}
	defer func() { _ = r.discardSubBlockRemainder(sb) }()

	strLen, err := r.readVarUint()
	if err != nil {
		return "", nil, err
	}
	isASCII, err := r.readBoolRaw()
	if err != nil {
		return "", nil, err
	}
	if !isASCII {
		return "", nil, errors.New("string is_ascii flag is false (unexpected)")
	}
	if strLen > uint64(math.MaxInt) {
		return "", nil, errors.Errorf("string length %d exceeds max int", strLen)
	}
	if strLen > uint64(sb.Size) {
		return "", nil, errors.Errorf("string length %d exceeds subblock size %d", strLen, sb.Size)
	}

	b, err := r.readBytes(int(strLen))
	if err != nil {
		return "", nil, err
	}
	s := string(b)

	ok, err := r.checkTag(2, rmV6TagTypeByte4)
	if err != nil {
		return "", nil, err
	}
	if ok {
		v, err := r.readUint32(2)
		if err != nil {
			return "", nil, err
		}
		return s, &v, nil
	}

	return s, nil, nil
}

func (r *rmV6TaggedBlockReader) readLwwBool(index uint32) (*RMV6LWW[bool], error) {
	sb, err := r.readSubBlock(index)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.discardSubBlockRemainder(sb) }()

	ts, err := r.readID(1)
	if err != nil {
		return nil, err
	}
	v, err := r.readBool(2)
	if err != nil {
		return nil, err
	}
	return &RMV6LWW[bool]{Timestamp: ts, Value: v}, nil
}

func (r *rmV6TaggedBlockReader) readLwwByte(index uint32) (*RMV6LWW[uint8], error) {
	sb, err := r.readSubBlock(index)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.discardSubBlockRemainder(sb) }()

	ts, err := r.readID(1)
	if err != nil {
		return nil, err
	}
	v, err := r.readByte(2)
	if err != nil {
		return nil, err
	}
	return &RMV6LWW[uint8]{Timestamp: ts, Value: v}, nil
}

func (r *rmV6TaggedBlockReader) readLwwFloat(index uint32) (*RMV6LWW[float32], error) {
	sb, err := r.readSubBlock(index)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.discardSubBlockRemainder(sb) }()

	ts, err := r.readID(1)
	if err != nil {
		return nil, err
	}
	v, err := r.readFloat32(2)
	if err != nil {
		return nil, err
	}
	return &RMV6LWW[float32]{Timestamp: ts, Value: v}, nil
}

func (r *rmV6TaggedBlockReader) readLwwID(index uint32) (*RMV6LWW[RMV6CrdtID], error) {
	sb, err := r.readSubBlock(index)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.discardSubBlockRemainder(sb) }()

	ts, err := r.readID(1)
	if err != nil {
		return nil, err
	}
	v, err := r.readID(2)
	if err != nil {
		return nil, err
	}
	return &RMV6LWW[RMV6CrdtID]{Timestamp: ts, Value: v}, nil
}

func (r *rmV6TaggedBlockReader) readLwwString(index uint32) (*RMV6LWW[string], error) {
	sb, err := r.readSubBlock(index)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.discardSubBlockRemainder(sb) }()

	ts, err := r.readID(1)
	if err != nil {
		return nil, err
	}
	s, err := r.readString(2)
	if err != nil {
		return nil, err
	}
	return &RMV6LWW[string]{Timestamp: ts, Value: s}, nil
}
