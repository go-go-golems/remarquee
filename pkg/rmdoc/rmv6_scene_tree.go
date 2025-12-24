package rmdoc

import (
	"bytes"
	"io"

	"github.com/pkg/errors"
)

// RMV6SceneTree mirrors the high-level structure built by rmscene's build_tree().
// For this milestone (task 35) we focus on groups + lines and keep all other items as raw placeholders.
type RMV6SceneTree struct {
	RootID RMV6CrdtID
	Root   *RMV6Group

	nodes map[RMV6CrdtID]*RMV6Group
}

// RMV6RootGroupID matches rmscene.scene_tree.ROOT_ID = CrdtId(0,1).
var RMV6RootGroupID = RMV6CrdtID{Part1: 0, Part2: 1}

type RMV6Group struct {
	NodeID RMV6CrdtID

	Children *RMV6CrdtSequence[RMV6SceneItem]

	// Extra captures bytes not decoded for this node (from TreeNodeBlock, etc).
	Extra []byte
}

type RMV6SceneItemKind uint8

const (
	RMV6SceneItemUnknown RMV6SceneItemKind = iota
	RMV6SceneItemGlyph
	RMV6SceneItemGroup
	RMV6SceneItemLine
	RMV6SceneItemText
	RMV6SceneItemTombstone
)

type RMV6Line struct {
	// BlockVersion is the main-block current_version for this line item block.
	BlockVersion uint8

	// Raw holds the undecoded value bytes (subblock 6 payload after item_type).
	Raw []byte
}

type RMV6SceneItem struct {
	Kind RMV6SceneItemKind

	Group *RMV6Group
	Line  *RMV6Line

	Raw []byte
}

func NewRMV6SceneTree() *RMV6SceneTree {
	root := &RMV6Group{
		NodeID:   RMV6RootGroupID,
		Children: NewRMV6CrdtSequence[RMV6SceneItem](),
	}
	return &RMV6SceneTree{
		RootID: RMV6RootGroupID,
		Root:   root,
		nodes: map[RMV6CrdtID]*RMV6Group{
			RMV6RootGroupID: root,
		},
	}
}

func (t *RMV6SceneTree) HasNode(id RMV6CrdtID) bool {
	_, ok := t.nodes[id]
	return ok
}

func (t *RMV6SceneTree) Node(id RMV6CrdtID) (*RMV6Group, bool) {
	n, ok := t.nodes[id]
	return n, ok
}

func (t *RMV6SceneTree) AddNode(nodeID RMV6CrdtID, parentID RMV6CrdtID) error {
	_ = parentID // parent linkage happens via SceneGroupItemBlock
	if t.nodes == nil {
		t.nodes = map[RMV6CrdtID]*RMV6Group{}
	}
	if _, ok := t.nodes[nodeID]; ok {
		// Node already exists; allow idempotent add (device emits updates).
		return nil
	}
	t.nodes[nodeID] = &RMV6Group{
		NodeID:   nodeID,
		Children: NewRMV6CrdtSequence[RMV6SceneItem](),
	}
	return nil
}

func (t *RMV6SceneTree) AddItem(item RMV6CrdtSequenceItem[RMV6SceneItem], parentID RMV6CrdtID) error {
	parent, ok := t.nodes[parentID]
	if !ok {
		return errors.Errorf("parent node not known: %s", parentID.String())
	}
	return parent.Children.Add(item)
}

// ParseRMV6SceneTree reads a v6 .rm file (reMarkable lines format, version=6)
// and builds a minimal scene tree with group nodes and ordered children sequences.
//
// It is a direct, simplified port of rmscene.scene_stream.build_tree().
func ParseRMV6SceneTree(r io.ReadSeeker) (*RMV6SceneTree, error) {
	tr := newRMV6TaggedBlockReader(r)
	if err := tr.readHeader(); err != nil {
		return nil, err
	}

	tree := NewRMV6SceneTree()

	for {
		blk, err := tr.readBlockHeader()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		switch blk.BlockType {
		case 0x01: // SceneTreeBlock
			treeID, err := tr.readID(1)
			if err != nil {
				_ = tr.endBlock(blk)
				continue
			}
			_, _ = tr.readID(2)   // node_id (unused)
			_, _ = tr.readBool(3) // is_update (unused)

			sb, err := tr.readSubBlock(4)
			if err != nil {
				_ = tr.endBlock(blk)
				continue
			}
			parentID, err := tr.readID(1)
			_ = tr.discardSubBlockRemainder(sb)
			if err != nil {
				_ = tr.endBlock(blk)
				continue
			}
			_ = tree.AddNode(treeID, parentID)

			_ = tr.endBlock(blk)

		case 0x02: // TreeNodeBlock
			nodeID, err := tr.readID(1)
			if err != nil {
				_ = tr.endBlock(blk)
				continue
			}
			// Ensure node exists; TreeNodeBlock enriches metadata for an existing node.
			if !tree.HasNode(nodeID) {
				// In python this is an error; keep it as an error because it indicates corrupted ordering.
				_ = tr.endBlock(blk)
				return nil, errors.Errorf("node does not exist for TreeNodeBlock: %s", nodeID.String())
			}
			// Capture remaining bytes as Extra on the node.
			if err := tr.endBlock(blk); err != nil {
				return nil, err
			}
			if n, ok := tree.Node(nodeID); ok {
				n.Extra = bytes.Clone(blk.ExtraData)
			}

		default:
			// Scene item blocks: 0x03 glyph, 0x04 group, 0x05 line, 0x06 text, 0x08 tombstone
			if !isRMV6SceneItemBlockType(blk.BlockType) {
				_ = tr.endBlock(blk)
				continue
			}

			parentID, err := tr.readID(1)
			if err != nil {
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

			sceneItem := RMV6SceneItem{Kind: RMV6SceneItemUnknown}

			hasValue, err := tr.hasSubBlock(6)
			if err != nil {
				_ = tr.endBlock(blk)
				continue
			}
			if hasValue {
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

				switch itemType {
				case 0x01:
					sceneItem.Kind = RMV6SceneItemGlyph
				case 0x02:
					sceneItem.Kind = RMV6SceneItemGroup
				case 0x03:
					sceneItem.Kind = RMV6SceneItemLine
				case 0x05:
					sceneItem.Kind = RMV6SceneItemText
				default:
					sceneItem.Kind = RMV6SceneItemUnknown
				}

				switch sceneItem.Kind {
				case RMV6SceneItemGroup:
					// SceneGroupItemBlock.value_from_stream reads id(2) within the value subblock.
					nodeID, err := tr.readID(2)
					if err == nil {
						if n, ok := tree.Node(nodeID); ok {
							sceneItem.Group = n
						} else {
							// Mirror python behavior: missing node id is an error.
							_ = tr.discardSubBlockRemainder(sb)
							_ = tr.endBlock(blk)
							return nil, errors.Errorf("node does not exist for SceneGroupItemBlock: %s", nodeID.String())
						}
					}
				case RMV6SceneItemLine:
					// Keep raw bytes for later stroke decoding.
					pos, _ := tr.tell()
					end := sb.Offset + int64(sb.Size)
					remaining := end - pos
					raw := []byte(nil)
					if remaining > 0 {
						raw, _ = tr.readBytes(int(remaining))
					}
					sceneItem.Line = &RMV6Line{
						BlockVersion: blk.CurrentVersion,
						Raw:          raw,
					}
				default:
					// For other item kinds, keep the raw subblock payload for later.
					pos, _ := tr.tell()
					end := sb.Offset + int64(sb.Size)
					remaining := end - pos
					if remaining > 0 {
						sceneItem.Raw, _ = tr.readBytes(int(remaining))
					}
				}

				_ = tr.discardSubBlockRemainder(sb)
			} else if blk.BlockType == uint8(RMV6SceneTombstoneItemBlockType) {
				sceneItem.Kind = RMV6SceneItemTombstone
			}

			seqItem := RMV6CrdtSequenceItem[RMV6SceneItem]{
				ItemID:        itemID,
				LeftID:        leftID,
				RightID:       rightID,
				DeletedLength: deletedLength,
				Value:         sceneItem,
			}

			if err := tree.AddItem(seqItem, parentID); err != nil {
				_ = tr.endBlock(blk)
				return nil, err
			}

			if err := tr.endBlock(blk); err != nil {
				return nil, err
			}
		}
	}

	return tree, nil
}
