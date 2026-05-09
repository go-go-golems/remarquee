package rmdoc

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"

	"github.com/pkg/errors"
)

// rmV6Header is the fixed 43-byte header prefix for reMarkable v6 .rm files.
// NOTE: this matches rmscene's HEADER_V6 in `rmscene/src/rmscene/tagged_block_common.py`.
var rmV6Header = []byte("reMarkable .lines file, version=6          ")

type rmV6TagType uint8

const (
	rmV6TagTypeID      rmV6TagType = 0xF
	rmV6TagTypeLength4 rmV6TagType = 0xC
	rmV6TagTypeByte8   rmV6TagType = 0x8
	rmV6TagTypeByte4   rmV6TagType = 0x4
	rmV6TagTypeByte1   rmV6TagType = 0x1
)

type rmV6MainBlockInfo struct {
	// Offset is the absolute file offset where the block payload begins.
	Offset int64
	// Size is the length of the block payload in bytes.
	Size uint32

	BlockType      uint8
	MinVersion     uint8
	CurrentVersion uint8
	ExtraData      []byte
}

type rmV6SubBlockInfo struct {
	Offset int64
	Size   uint32

	Index     uint32
	ExtraData []byte
}

// rmV6TaggedBlockReader is a minimal Go port of rmscene's TaggedBlockReader that focuses on:
// - validating/reading the v6 file header,
// - reading top-level block boundaries,
// - reading subblock boundaries (Length4 tagged blocks),
// while deferring semantic decoding (scene tree / CRDT) to later steps.
//
// This is intended as a foundation for RMQ-0004 task 33.
type rmV6TaggedBlockReader struct {
	rs io.ReadSeeker

	current *rmV6MainBlockInfo
}

func newRMV6TaggedBlockReader(rs io.ReadSeeker) *rmV6TaggedBlockReader {
	return &rmV6TaggedBlockReader{rs: rs}
}

func (r *rmV6TaggedBlockReader) tell() (int64, error) {
	off, err := r.rs.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, errors.Wrap(err, "seek current")
	}
	return off, nil
}

func (r *rmV6TaggedBlockReader) readBytes(n int) ([]byte, error) {
	buf := make([]byte, n)
	_, err := io.ReadFull(r.rs, buf)
	if err != nil {
		return nil, errors.Wrap(err, "read bytes")
	}
	return buf, nil
}

func (r *rmV6TaggedBlockReader) readUint8() (uint8, error) {
	b, err := r.readBytes(1)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

func (r *rmV6TaggedBlockReader) readUint32LE() (uint32, error) {
	b, err := r.readBytes(4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(b), nil
}

func (r *rmV6TaggedBlockReader) readVarUint() (uint64, error) {
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
		// Protect against malformed inputs causing an infinite loop / overflow.
		if shift > 63 {
			return 0, errors.Errorf("varuint too large (shift=%d)", shift)
		}
	}
	return out, nil
}

func (r *rmV6TaggedBlockReader) readHeader() error {
	header, err := r.readBytes(len(rmV6Header))
	if err != nil {
		return errors.Wrap(err, "read rm v6 header")
	}
	if !bytes.Equal(header, rmV6Header) {
		return errors.Errorf("wrong rm v6 header: got %q", string(header))
	}
	return nil
}

// peekTag reads the next tag varuint and rewinds the stream back to its original position.
func (r *rmV6TaggedBlockReader) peekTag() (uint32, rmV6TagType, bool, error) {
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

// readTag reads a single tagged field header (varuint) and returns (index, tagType).
func (r *rmV6TaggedBlockReader) readTag() (uint32, rmV6TagType, error) {
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

func (r *rmV6TaggedBlockReader) skipTaggedValue(tagType rmV6TagType) error {
	switch tagType {
	case rmV6TagTypeByte1:
		_, err := r.readBytes(1)
		return err
	case rmV6TagTypeByte4:
		_, err := r.readBytes(4)
		return err
	case rmV6TagTypeByte8:
		_, err := r.readBytes(8)
		return err
	case rmV6TagTypeLength4:
		size, err := r.readUint32LE()
		if err != nil {
			return err
		}
		_, err = r.rs.Seek(int64(size), io.SeekCurrent)
		return err
	case rmV6TagTypeID:
		// Matches rmscene DataStream.read_crdt_id:
		// part1: uint8, part2: varuint
		if _, err := r.readUint8(); err != nil {
			return err
		}
		_, err := r.readVarUint()
		return err
	default:
		return errors.Errorf("cannot skip unsupported tag type 0x%X", uint8(tagType))
	}
}

// readBlockHeader reads the next top-level block header and enters the block payload.
// The caller should call endBlock on the returned info to ensure the stream is positioned
// at the end of the block payload (unread bytes become ExtraData).
func (r *rmV6TaggedBlockReader) readBlockHeader() (*rmV6MainBlockInfo, error) {
	if r.current != nil {
		return nil, errors.New("already in a block")
	}

	// The python reader treats EOF while reading block_length as "no more blocks".
	// We mirror that behavior by mapping io.EOF to io.EOF here.
	blockLength, err := r.readUint32LE()
	if err != nil {
		// unwrap readBytes -> io.ReadFull errors; treat EOF/UnexpectedEOF as EOF
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, io.EOF
		}
		return nil, errors.Wrap(err, "read block length")
	}

	unknown, err := r.readUint8()
	if err != nil {
		return nil, errors.Wrap(err, "read block header unknown")
	}
	minVersion, err := r.readUint8()
	if err != nil {
		return nil, errors.Wrap(err, "read block header min_version")
	}
	currentVersion, err := r.readUint8()
	if err != nil {
		return nil, errors.Wrap(err, "read block header current_version")
	}
	blockType, err := r.readUint8()
	if err != nil {
		return nil, errors.Wrap(err, "read block header block_type")
	}

	if unknown != 0 {
		return nil, errors.Errorf("unexpected rm v6 block header byte: got %d, want 0", unknown)
	}
	if minVersion > currentVersion {
		return nil, errors.Errorf("invalid rm v6 block versions: min=%d current=%d", minVersion, currentVersion)
	}

	offset, err := r.tell()
	if err != nil {
		return nil, err
	}

	info := &rmV6MainBlockInfo{
		Offset:         offset,
		Size:           blockLength,
		BlockType:      blockType,
		MinVersion:     minVersion,
		CurrentVersion: currentVersion,
	}
	r.current = info
	return info, nil
}

func (r *rmV6TaggedBlockReader) bytesRemainingInBlock() (int64, error) {
	if r.current == nil {
		return 0, errors.New("not in a block")
	}
	pos, err := r.tell()
	if err != nil {
		return 0, err
	}
	end := r.current.Offset + int64(r.current.Size)
	return end - pos, nil
}

// endBlock verifies we did not read past the end, and if under-read, discards the remainder
// into info.ExtraData.
func (r *rmV6TaggedBlockReader) endBlock(info *rmV6MainBlockInfo) error {
	if r.current == nil || r.current != info {
		return errors.New("endBlock called for non-current block")
	}

	pos, err := r.tell()
	if err != nil {
		return err
	}
	end := info.Offset + int64(info.Size)
	if pos > end {
		return errors.Errorf("read past end of rm v6 block (pos=%d end=%d overflow=%d)", pos, end, pos-end)
	}
	if pos < end {
		remaining := end - pos
		extra, err := r.readBytes(int(remaining))
		if err != nil {
			return errors.Wrap(err, "discard remaining block bytes")
		}
		info.ExtraData = extra
	}

	r.current = nil
	return nil
}

// readSubBlock reads a Length4 tagged subblock boundary and returns its info.
// Like endBlock, it is up to the caller to consume exactly Size bytes from the stream,
// or to explicitly discard the remainder via discardSubBlockRemainder.
func (r *rmV6TaggedBlockReader) readSubBlock(expectedIndex uint32) (*rmV6SubBlockInfo, error) {
	idx, tt, err := r.readTag()
	if err != nil {
		return nil, errors.Wrap(err, "read subblock tag")
	}
	if idx != expectedIndex {
		return nil, errors.Errorf("unexpected subblock index: got %d want %d", idx, expectedIndex)
	}
	if tt != rmV6TagTypeLength4 {
		return nil, errors.Errorf("unexpected subblock tag type: got 0x%X want 0x%X", uint8(tt), uint8(rmV6TagTypeLength4))
	}
	subLen, err := r.readUint32LE()
	if err != nil {
		return nil, errors.Wrap(err, "read subblock length")
	}
	off, err := r.tell()
	if err != nil {
		return nil, err
	}
	return &rmV6SubBlockInfo{
		Offset: off,
		Size:   subLen,
		Index:  expectedIndex,
	}, nil
}

func (r *rmV6TaggedBlockReader) discardSubBlockRemainder(info *rmV6SubBlockInfo) error {
	pos, err := r.tell()
	if err != nil {
		return err
	}
	end := info.Offset + int64(info.Size)
	if pos > end {
		return errors.Errorf("read past end of rm v6 subblock (pos=%d end=%d overflow=%d)", pos, end, pos-end)
	}
	if pos < end {
		remaining := end - pos
		extra, err := r.readBytes(int(remaining))
		if err != nil {
			return errors.Wrap(err, "discard remaining subblock bytes")
		}
		info.ExtraData = extra
	}
	return nil
}
