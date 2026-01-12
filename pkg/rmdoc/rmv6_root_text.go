package rmdoc

import "github.com/pkg/errors"

// RMV6RootText is a minimal representation of RootTextBlock.value from rmscene.
type RMV6RootText struct {
	Items  []RMV6CrdtSequenceItem[RMV6TextAtom]
	Styles map[RMV6CrdtID]uint8 // ParagraphStyle code (see rmscene.scene_items.ParagraphStyle)
	PosX   float64
	PosY   float64
	Width  float64
}

type RMV6TextAtomKind uint8

const (
	RMV6TextAtomString RMV6TextAtomKind = iota
	RMV6TextAtomFormatCode
)

type RMV6TextAtom struct {
	Kind RMV6TextAtomKind

	Text       string
	FormatCode uint32
}

func rmv6ReadTextItemFromStream(tr *rmV6TaggedBlockReader) (RMV6CrdtSequenceItem[RMV6TextAtom], error) {
	sb, err := tr.readSubBlock(0)
	if err != nil {
		return RMV6CrdtSequenceItem[RMV6TextAtom]{}, err
	}
	defer func() { _ = tr.discardSubBlockRemainder(sb) }()

	itemID, err := tr.readID(2)
	if err != nil {
		return RMV6CrdtSequenceItem[RMV6TextAtom]{}, err
	}
	leftID, err := tr.readID(3)
	if err != nil {
		return RMV6CrdtSequenceItem[RMV6TextAtom]{}, err
	}
	rightID, err := tr.readID(4)
	if err != nil {
		return RMV6CrdtSequenceItem[RMV6TextAtom]{}, err
	}
	deletedLength, err := tr.readUint32(5)
	if err != nil {
		return RMV6CrdtSequenceItem[RMV6TextAtom]{}, err
	}

	atom := RMV6TextAtom{Kind: RMV6TextAtomString, Text: ""}

	has, err := tr.hasSubBlock(6)
	if err != nil {
		return RMV6CrdtSequenceItem[RMV6TextAtom]{}, err
	}
	if has {
		text, fmt, err := tr.readStringWithFormat(6)
		if err != nil {
			return RMV6CrdtSequenceItem[RMV6TextAtom]{}, err
		}
		if fmt != nil {
			atom = RMV6TextAtom{Kind: RMV6TextAtomFormatCode, FormatCode: *fmt}
		} else {
			atom = RMV6TextAtom{Kind: RMV6TextAtomString, Text: text}
		}
	}

	return RMV6CrdtSequenceItem[RMV6TextAtom]{
		ItemID:        itemID,
		LeftID:        leftID,
		RightID:       rightID,
		DeletedLength: deletedLength,
		Value:         atom,
	}, nil
}

func rmv6ReadTextFormatFromStream(tr *rmV6TaggedBlockReader) (RMV6CrdtID, uint8, error) {
	charID, err := tr.readCrdtIDRaw()
	if err != nil {
		return RMV6CrdtID{}, 0, err
	}
	// timestamp
	if _, err := tr.readID(1); err != nil {
		return RMV6CrdtID{}, 0, err
	}

	sb, err := tr.readSubBlock(2)
	if err != nil {
		return RMV6CrdtID{}, 0, err
	}
	defer func() { _ = tr.discardSubBlockRemainder(sb) }()

	// c (17) then format_code
	_, err = tr.readUint8()
	if err != nil {
		return RMV6CrdtID{}, 0, err
	}
	formatCode, err := tr.readUint8()
	if err != nil {
		return RMV6CrdtID{}, 0, err
	}

	return charID, formatCode, nil
}

// ParseRMV6RootTextBlock parses RootTextBlock (block type 0x07) from the current block.
func ParseRMV6RootTextBlock(tr *rmV6TaggedBlockReader) (*RMV6RootText, error) {
	blockID, err := tr.readID(1)
	if err != nil {
		return nil, err
	}
	// rmscene asserts (0,0)
	if blockID != RMV6EndMarker {
		// tolerate for now, but still parse
		_ = blockID
	}

	sb2, err := tr.readSubBlock(2)
	if err != nil {
		return nil, err
	}

	// Text items section: subblock(1)->subblock(1)->varuint count -> repeated text_item_from_stream()
	sbItemsOuter, err := tr.readSubBlock(1)
	if err != nil {
		return nil, err
	}

	sbItemsInner, err := tr.readSubBlock(1)
	if err != nil {
		return nil, err
	}

	nItems, err := tr.readVarUint()
	if err != nil {
		return nil, err
	}
	items := make([]RMV6CrdtSequenceItem[RMV6TextAtom], 0, nItems)
	for i := 0; i < int(nItems); i++ {
		it, err := rmv6ReadTextItemFromStream(tr)
		if err != nil {
			return nil, errors.Wrap(err, "read text item")
		}
		items = append(items, it)
	}
	if err := tr.discardSubBlockRemainder(sbItemsInner); err != nil {
		return nil, err
	}
	if err := tr.discardSubBlockRemainder(sbItemsOuter); err != nil {
		return nil, err
	}

	// Formatting section: subblock(2)->subblock(1)->varuint count -> repeated text_format_from_stream()
	sbFmtOuter, err := tr.readSubBlock(2)
	if err != nil {
		return nil, err
	}

	sbFmtInner, err := tr.readSubBlock(1)
	if err != nil {
		return nil, err
	}

	nFmt, err := tr.readVarUint()
	if err != nil {
		return nil, err
	}
	styles := map[RMV6CrdtID]uint8{}
	for i := 0; i < int(nFmt); i++ {
		charID, code, err := rmv6ReadTextFormatFromStream(tr)
		if err != nil {
			return nil, errors.Wrap(err, "read text format")
		}
		styles[charID] = code
	}
	if err := tr.discardSubBlockRemainder(sbFmtInner); err != nil {
		return nil, err
	}
	if err := tr.discardSubBlockRemainder(sbFmtOuter); err != nil {
		return nil, err
	}
	if err := tr.discardSubBlockRemainder(sb2); err != nil {
		return nil, err
	}

	// Last section: subblock(3) pos_x,pos_y float64
	sb3, err := tr.readSubBlock(3)
	if err != nil {
		return nil, err
	}

	posX, err := tr.readFloat64Raw()
	if err != nil {
		return nil, err
	}
	posY, err := tr.readFloat64Raw()
	if err != nil {
		return nil, err
	}
	if err := tr.discardSubBlockRemainder(sb3); err != nil {
		return nil, err
	}

	// width is tagged float32 at index 4
	w, err := tr.readFloat32(4)
	if err != nil {
		return nil, err
	}

	return &RMV6RootText{
		Items:  items,
		Styles: styles,
		PosX:   posX,
		PosY:   posY,
		Width:  float64(w),
	}, nil
}
