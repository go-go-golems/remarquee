package rmdoc

import (
	"bytes"
	"errors"
	"io"
)

// RMV6SceneItemBlockType mirrors the scene item block type constants in rmscene `scene_stream.py`.
type RMV6SceneItemBlockType uint8

const (
	RMV6SceneGlyphItemBlockType     RMV6SceneItemBlockType = 0x03
	RMV6SceneGroupItemBlockType     RMV6SceneItemBlockType = 0x04
	RMV6SceneLineItemBlockType      RMV6SceneItemBlockType = 0x05
	RMV6SceneTextItemBlockType      RMV6SceneItemBlockType = 0x06
	RMV6SceneTombstoneItemBlockType RMV6SceneItemBlockType = 0x08
)

// RMV6SceneItemBlock is a minimal decoded representation of a SceneItemBlock.
//
// For this milestone we decode:
// - parent_id (tag 1)
// - CRDT sequence header fields (tags 2..5)
// - subblock 6 boundary (if present) and item_type (first byte)
// and we keep the remainder of subblock 6 as raw bytes for later decoding.
type RMV6SceneItemBlock struct {
	BlockType RMV6SceneItemBlockType

	ParentID RMV6CrdtID
	Item     RMV6CrdtSequenceItem[RMV6SceneItemValue]

	// ExtraBlockData captures any bytes not decoded inside this top-level block.
	ExtraBlockData []byte
}

type RMV6SceneItemValue struct {
	ItemType uint8
	Raw      []byte
	Extra    []byte
}

func isRMV6SceneItemBlockType(t uint8) bool {
	switch RMV6SceneItemBlockType(t) {
	case RMV6SceneGlyphItemBlockType, RMV6SceneGroupItemBlockType, RMV6SceneLineItemBlockType, RMV6SceneTextItemBlockType, RMV6SceneTombstoneItemBlockType:
		return true
	default:
		return false
	}
}

// ParseRMV6SceneItemBlocks scans a V6 .rm file and returns minimal SceneItemBlock decodes.
func ParseRMV6SceneItemBlocks(r io.ReadSeeker) ([]RMV6SceneItemBlock, error) {
	tr := newRMV6TaggedBlockReader(r)
	if err := tr.readHeader(); err != nil {
		return nil, err
	}

	var out []RMV6SceneItemBlock

	for {
		blk, err := tr.readBlockHeader()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		if !isRMV6SceneItemBlockType(blk.BlockType) {
			if err := tr.endBlock(blk); err != nil {
				return nil, err
			}
			continue
		}

		parentID, err := tr.readID(1)
		if err != nil {
			// Keep forward progress: discard unread block bytes and continue.
			_ = tr.endBlock(blk)
			continue
		}
		itemID, err := tr.readID(2)
		if err != nil {
			_ = tr.endBlock(blk)
			continue
		}
		leftID, err := tr.readID(3)
		if err != nil {
			_ = tr.endBlock(blk)
			continue
		}
		rightID, err := tr.readID(4)
		if err != nil {
			_ = tr.endBlock(blk)
			continue
		}
		deletedLength, err := tr.readUint32(5)
		if err != nil {
			_ = tr.endBlock(blk)
			continue
		}

		var value RMV6SceneItemValue
		hasV, err := tr.hasSubBlock(6)
		if err != nil {
			_ = tr.endBlock(blk)
			continue
		}
		if hasV {
			sb, err := tr.readSubBlock(6)
			if err != nil {
				_ = tr.endBlock(blk)
				continue
			}

			itemType, err := tr.readUint8()
			if err != nil {
				_ = tr.discardSubBlockRemainder(sb)
				_ = tr.endBlock(blk)
				continue
			}

			// Read the remaining bytes of the subblock as raw.
			pos, _ := tr.tell()
			end := sb.Offset + int64(sb.Size)
			remaining := end - pos
			if remaining < 0 {
				remaining = 0
			}
			raw := []byte(nil)
			if remaining > 0 {
				raw, _ = tr.readBytes(int(remaining))
			}
			// discardSubBlockRemainder should be a no-op now, but keep it for correctness.
			_ = tr.discardSubBlockRemainder(sb)

			value = RMV6SceneItemValue{
				ItemType: itemType,
				Raw:      raw,
				Extra:    sb.ExtraData,
			}
		}

		b := RMV6SceneItemBlock{
			BlockType: RMV6SceneItemBlockType(blk.BlockType),
			ParentID:  parentID,
			Item: RMV6CrdtSequenceItem[RMV6SceneItemValue]{
				ItemID:        itemID,
				LeftID:        leftID,
				RightID:       rightID,
				DeletedLength: deletedLength,
				Value:         value,
			},
		}

		// endBlock will capture any remaining bytes in blk.ExtraData.
		if err := tr.endBlock(blk); err != nil {
			return nil, err
		}
		b.ExtraBlockData = bytes.Clone(blk.ExtraData)

		out = append(out, b)
	}

	return out, nil
}
