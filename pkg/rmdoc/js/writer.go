package js

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	
	"github.com/go-go-golems/remarquee/pkg/rmdoc"
)

// TaggedBlockWriter writes v6 .rm file format
type TaggedBlockWriter struct {
	w io.Writer
}

func NewTaggedBlockWriter(w io.Writer) *TaggedBlockWriter {
	return &TaggedBlockWriter{w: w}
}

func (w *TaggedBlockWriter) WriteHeader() error {
	header := []byte("reMarkable .lines file, version=6          ")
	_, err := w.w.Write(header)
	return err
}

func (w *TaggedBlockWriter) WriteUint8(v uint8) error {
	return binary.Write(w.w, binary.LittleEndian, v)
}

func (w *TaggedBlockWriter) WriteUint32(v uint32) error {
	return binary.Write(w.w, binary.LittleEndian, v)
}

func (w *TaggedBlockWriter) WriteFloat32(v float32) error {
	return binary.Write(w.w, binary.LittleEndian, v)
}

func (w *TaggedBlockWriter) WriteFloat64(v float64) error {
	return binary.Write(w.w, binary.LittleEndian, v)
}

func (w *TaggedBlockWriter) WriteVarUint(v uint64) error {
	for {
		b := uint8(v & 0x7F)
		v >>= 7
		if v != 0 {
			b |= 0x80
		}
		if err := w.WriteUint8(b); err != nil {
			return err
		}
		if v == 0 {
			break
		}
	}
	return nil
}

func (w *TaggedBlockWriter) WriteTag(index uint8, tagType uint8) error {
	tag := (index << 4) | tagType
	return w.WriteUint8(tag)
}

func (w *TaggedBlockWriter) WriteCrdtID(id rmdoc.RMV6CrdtID) error {
	if err := w.WriteUint8(id.Part1); err != nil {
		return err
	}
	return w.WriteVarUint(id.Part2)
}

func (w *TaggedBlockWriter) WriteID(index uint8, id rmdoc.RMV6CrdtID) error {
	if err := w.WriteTag(index, 0xF); err != nil {
		return err
	}
	return w.WriteCrdtID(id)
}

func (w *TaggedBlockWriter) WriteByte(index uint8, v uint8) error {
	if err := w.WriteTag(index, 0x1); err != nil {
		return err
	}
	return w.WriteUint8(v)
}

func (w *TaggedBlockWriter) WriteInt(index uint8, v uint32) error {
	if err := w.WriteTag(index, 0x4); err != nil {
		return err
	}
	return w.WriteUint32(v)
}

func (w *TaggedBlockWriter) WriteFloat(index uint8, v float32) error {
	if err := w.WriteTag(index, 0x4); err != nil {
		return err
	}
	return w.WriteFloat32(v)
}

func (w *TaggedBlockWriter) WriteDouble(index uint8, v float64) error {
	if err := w.WriteTag(index, 0x8); err != nil {
		return err
	}
	return w.WriteFloat64(v)
}

// WriteBlock writes a top-level block
func (w *TaggedBlockWriter) WriteBlock(blockType, minVersion, currentVersion uint8, content []byte) error {
	if err := w.WriteUint32(uint32(len(content))); err != nil {
		return err
	}
	if err := w.WriteUint8(0); err != nil {
		return err
	}
	if err := w.WriteUint8(minVersion); err != nil {
		return err
	}
	if err := w.WriteUint8(currentVersion); err != nil {
		return err
	}
	if err := w.WriteUint8(blockType); err != nil {
		return err
	}
	_, err := w.w.Write(content)
	return err
}

// WriteSubblock writes a subblock with Length4 tag
func (w *TaggedBlockWriter) WriteSubblock(index uint8, content []byte) error {
	if err := w.WriteTag(index, 0xC); err != nil {
		return err
	}
	if err := w.WriteUint32(uint32(len(content))); err != nil {
		return err
	}
	_, err := w.w.Write(content)
	return err
}

// Helper to build subblock content
func BuildSubblock(fn func(*TaggedBlockWriter) error) ([]byte, error) {
	var buf bytes.Buffer
	w := NewTaggedBlockWriter(&buf)
	if err := fn(w); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// WriteSceneTree writes a complete scene tree to a .rm file
func WriteSceneTree(w io.Writer, tree *rmdoc.RMV6SceneTree, strokes []rmdoc.Stroke) error {
	writer := NewTaggedBlockWriter(w)
	
	// Write header
	if err := writer.WriteHeader(); err != nil {
		return err
	}
	
	// Build the scene tree block content
	blockContent, err := BuildSceneTreeBlock(tree, strokes)
	if err != nil {
		return err
	}
	
	// Write the block
	return writer.WriteBlock(1, 0, 0, blockContent)
}

func BuildSceneTreeBlock(tree *rmdoc.RMV6SceneTree, strokes []rmdoc.Stroke) ([]byte, error) {
	var buf bytes.Buffer
	w := NewTaggedBlockWriter(&buf)
	
	// Write tree header (index 1: tree ID)
	treeID := rmdoc.RMV6CrdtID{Part1: 1, Part2: 0}
	if err := w.WriteID(1, treeID); err != nil {
		return nil, err
	}
	
	// Write root node (index 2)
	rootContent, err := BuildRootNode(strokes)
	if err != nil {
		return nil, err
	}
	if err := w.WriteSubblock(2, rootContent); err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

func BuildRootNode(strokes []rmdoc.Stroke) ([]byte, error) {
	var buf bytes.Buffer
	w := NewTaggedBlockWriter(&buf)
	
	// Write node type (0 = Group)
	if err := w.WriteByte(1, 0); err != nil {
		return nil, err
	}
	
	// Write children sequence (index 2)
	childrenContent, err := BuildChildrenSequence(strokes)
	if err != nil {
		return nil, err
	}
	if err := w.WriteSubblock(2, childrenContent); err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

func BuildChildrenSequence(strokes []rmdoc.Stroke) ([]byte, error) {
	var buf bytes.Buffer
	w := NewTaggedBlockWriter(&buf)
	
	// Write each stroke as a sequence item
	for i, stroke := range strokes {
		itemContent, err := BuildSequenceItem(i, stroke)
		if err != nil {
			return nil, err
		}
		if err := w.WriteSubblock(uint8(i+1), itemContent); err != nil {
			return nil, err
		}
	}
	
	return buf.Bytes(), nil
}

func BuildSequenceItem(index int, stroke rmdoc.Stroke) ([]byte, error) {
	var buf bytes.Buffer
	w := NewTaggedBlockWriter(&buf)
	
	// Item ID
	itemID := rmdoc.RMV6CrdtID{Part1: uint8(index + 1), Part2: uint64(index + 1)}
	if err := w.WriteID(1, itemID); err != nil {
		return nil, err
	}
	
	// Left ID
	leftID := rmdoc.RMV6CrdtID{Part1: 0, Part2: 0}
	if index > 0 {
		leftID = rmdoc.RMV6CrdtID{Part1: uint8(index), Part2: uint64(index)}
	}
	if err := w.WriteID(2, leftID); err != nil {
		return nil, err
	}
	
	// Right ID (always 0,0 for end)
	rightID := rmdoc.RMV6CrdtID{Part1: 0, Part2: 0}
	if err := w.WriteID(3, rightID); err != nil {
		return nil, err
	}
	
	// Deleted length (0 = not deleted)
	if err := w.WriteInt(4, 0); err != nil {
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

func BuildLineItem(stroke rmdoc.Stroke) ([]byte, error) {
	var buf bytes.Buffer
	w := NewTaggedBlockWriter(&buf)
	
	// Item type (3 = Line)
	if err := w.WriteByte(1, 3); err != nil {
		return nil, err
	}
	
	// Line data (index 2)
	lineContent, err := BuildLineData(stroke)
	if err != nil {
		return nil, err
	}
	if err := w.WriteSubblock(2, lineContent); err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

func BuildLineData(stroke rmdoc.Stroke) ([]byte, error) {
	var buf bytes.Buffer
	w := NewTaggedBlockWriter(&buf)
	
	// Tool (index 1)
	if err := w.WriteByte(1, uint8(stroke.Tool)); err != nil {
		return nil, err
	}
	
	// Color (index 2)
	if err := w.WriteByte(2, uint8(stroke.Color)); err != nil {
		return nil, err
	}
	
	// Thickness scale (index 4)
	if err := w.WriteFloat(4, float32(stroke.ThicknessScale)); err != nil {
		return nil, err
	}
	
	// Points (index 6)
	pointsContent, err := BuildPoints(stroke.Points)
	if err != nil {
		return nil, err
	}
	if err := w.WriteSubblock(6, pointsContent); err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

func BuildPoints(points []rmdoc.StrokePoint) ([]byte, error) {
	var buf bytes.Buffer
	w := NewTaggedBlockWriter(&buf)
	
	// Number of points
	if err := w.WriteVarUint(uint64(len(points))); err != nil {
		return nil, err
	}
	
	// Write each point
	for _, pt := range points {
		// X coordinate (float64)
		if err := w.WriteFloat64(pt.X); err != nil {
			return nil, err
		}
		// Y coordinate (float64)
		if err := w.WriteFloat64(pt.Y); err != nil {
			return nil, err
		}
		// Speed (float64)
		if err := w.WriteFloat64(pt.Speed); err != nil {
			return nil, err
		}
		// Direction (float64)
		direction := pt.Direction
		if math.IsNaN(direction) {
			direction = 0
		}
		if err := w.WriteFloat64(direction); err != nil {
			return nil, err
		}
		// Width (float64)
		if err := w.WriteFloat64(pt.Width); err != nil {
			return nil, err
		}
		// Pressure (float64)
		if err := w.WriteFloat64(pt.Pressure); err != nil {
			return nil, err
		}
	}
	
	return buf.Bytes(), nil
}
