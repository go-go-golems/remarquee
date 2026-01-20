package js

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	
	"github.com/go-go-golems/remarquee/pkg/rmdoc"
	"github.com/pkg/errors"
)

// WriteSceneTreeV2 writes a complete scene tree to a .rm file using the tree's actual items
func WriteSceneTreeV2(w io.Writer, tree *rmdoc.RMV6SceneTree, strokes []rmdoc.Stroke) error {
	writer := NewTaggedBlockWriter(w)
	
	// Write header
	if err := writer.WriteHeader(); err != nil {
		return err
	}
	
	// Build the scene tree block content from the actual tree
	blockContent, err := BuildSceneTreeBlockV2(tree, strokes)
	if err != nil {
		return err
	}
	
	// Write the block
	return writer.WriteBlock(1, 0, 0, blockContent)
}

func BuildSceneTreeBlockV2(tree *rmdoc.RMV6SceneTree, strokes []rmdoc.Stroke) ([]byte, error) {
	var buf bytes.Buffer
	w := NewTaggedBlockWriter(&buf)
	
	// Write tree header (index 1: tree ID)
	treeID := rmdoc.RMV6CrdtID{Part1: 1, Part2: 0}
	if err := w.WriteID(1, treeID); err != nil {
		return nil, err
	}
	
	// Write root node (index 2)
	rootContent, err := BuildRootNodeV2(tree, strokes)
	if err != nil {
		return nil, err
	}
	if err := w.WriteSubblock(2, rootContent); err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

func BuildRootNodeV2(tree *rmdoc.RMV6SceneTree, strokes []rmdoc.Stroke) ([]byte, error) {
	var buf bytes.Buffer
	w := NewTaggedBlockWriter(&buf)
	
	// Write node type (0 = Group)
	if err := w.WriteByte(1, 0); err != nil {
		return nil, err
	}
	
	// Get the actual items from the tree
	items, err := tree.Root.Children.Items()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get items from tree")
	}
	
	// Debug: print item count
	fmt.Printf("[DEBUG] Writing %d items from tree\n", len(items))
	
	// Write children sequence (index 2)
	childrenContent, err := BuildChildrenSequenceV2(items, strokes)
	if err != nil {
		return nil, err
	}
	if err := w.WriteSubblock(2, childrenContent); err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

func BuildChildrenSequenceV2(items []rmdoc.RMV6CrdtSequenceItem[rmdoc.RMV6SceneItem], strokes []rmdoc.Stroke) ([]byte, error) {
	var buf bytes.Buffer
	w := NewTaggedBlockWriter(&buf)
	
	// Write each item from the tree
	for i, item := range items {
		itemContent, err := BuildSequenceItemV2(item, strokes[i])
		if err != nil {
			return nil, err
		}
		// Use index i+1 for the subblock
		if err := w.WriteSubblock(uint8(i+1), itemContent); err != nil {
			return nil, err
		}
	}
	
	return buf.Bytes(), nil
}

func BuildSequenceItemV2(item rmdoc.RMV6CrdtSequenceItem[rmdoc.RMV6SceneItem], stroke rmdoc.Stroke) ([]byte, error) {
	var buf bytes.Buffer
	w := NewTaggedBlockWriter(&buf)
	
	// Write the CRDT IDs from the item
	if err := w.WriteID(1, item.ItemID); err != nil {
		return nil, err
	}
	
	if err := w.WriteID(2, item.LeftID); err != nil {
		return nil, err
	}
	
	if err := w.WriteID(3, item.RightID); err != nil {
		return nil, err
	}
	
	// Deleted length
	if err := w.WriteInt(4, item.DeletedLength); err != nil {
		return nil, err
	}
	
	// Value (the line/stroke item)
	valueContent, err := BuildLineItem(stroke)
	if err != nil {
		return nil, err
	}
	if err := w.WriteSubblock(5, valueContent); err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

// Helper to write varuint (for point count)
func writeVarUint(w io.Writer, v uint64) error {
	for {
		b := uint8(v & 0x7F)
		v >>= 7
		if v != 0 {
			b |= 0x80
		}
		if err := binary.Write(w, binary.LittleEndian, b); err != nil {
			return err
		}
		if v == 0 {
			break
		}
	}
	return nil
}
