package rmdoc

import (
	"math"

	"github.com/pkg/errors"
)

const rmv6TextTopY = -88.0

var rmv6LineHeights = map[uint8]float64{
	1: 70,  // PLAIN
	3: 70,  // BOLD
	2: 150, // HEADING
	4: 35,  // BULLET
	5: 35,  // BULLET2
	6: 35,  // CHECKBOX
	7: 35,  // CHECKBOX_CHECKED
	0: 70,  // BASIC
}

// BuildRMV6AnchorPos builds a mapping from text character IDs to Y positions (screen units),
// using the same approach as `rmc`'s svg exporter (build_anchor_pos + TextDocument.from_scene_item).
func BuildRMV6AnchorPos(rt *RMV6RootText) (map[RMV6CrdtID]float64, error) {
	anchorPos := map[RMV6CrdtID]float64{
		// Special anchors from rmc exporter.
		{Part1: 0, Part2: 281474976710654}: 100,
		{Part1: 0, Part2: 281474976710655}: 100,
	}
	if rt == nil {
		return anchorPos, nil
	}

	// Order original text items via CRDT toposort.
	seq := NewRMV6CrdtSequence[RMV6TextAtom]()
	for _, it := range rt.Items {
		_ = seq.Add(it)
	}
	ordered, err := seq.Items()
	if err != nil {
		return nil, errors.Wrap(err, "order text items")
	}

	// Expand to single-character items (mirrors rmscene.text.expand_text_items).
	type charVal struct {
		isFmt bool
		fmt   uint32
		ch    string
	}

	charSeq := NewRMV6CrdtSequence[charVal]()
	for _, item := range ordered {
		if item.DeletedLength > 0 {
			// Deleted spans: treat as sequence of empty chars.
			itemID := item.ItemID
			leftID := item.LeftID
			for i := uint32(0); i < item.DeletedLength; i++ {
				rightID := item.RightID
				if i < item.DeletedLength-1 {
					rightID = RMV6CrdtID{Part1: itemID.Part1, Part2: itemID.Part2 + 1}
				}
				_ = charSeq.Add(RMV6CrdtSequenceItem[charVal]{
					ItemID:        itemID,
					LeftID:        leftID,
					RightID:       rightID,
					DeletedLength: 1,
					Value:         charVal{ch: ""},
				})
				leftID = itemID
				itemID = rightID
			}
			continue
		}

		if item.Value.Kind == RMV6TextAtomFormatCode {
			_ = charSeq.Add(RMV6CrdtSequenceItem[charVal]{
				ItemID:        item.ItemID,
				LeftID:        item.LeftID,
				RightID:       item.RightID,
				DeletedLength: item.DeletedLength,
				Value:         charVal{isFmt: true, fmt: item.Value.FormatCode},
			})
			continue
		}

		chars := []rune(item.Value.Text)
		if len(chars) == 0 {
			continue
		}

		itemID := item.ItemID
		leftID := item.LeftID
		for _, c := range chars[:len(chars)-1] {
			rightID := RMV6CrdtID{Part1: itemID.Part1, Part2: itemID.Part2 + 1}
			_ = charSeq.Add(RMV6CrdtSequenceItem[charVal]{
				ItemID:  itemID,
				LeftID:  leftID,
				RightID: rightID,
				Value:   charVal{ch: string(c)},
			})
			leftID = itemID
			itemID = rightID
		}
		_ = charSeq.Add(RMV6CrdtSequenceItem[charVal]{
			ItemID:  itemID,
			LeftID:  leftID,
			RightID: item.RightID,
			Value:   charVal{ch: string(chars[len(chars)-1])},
		})
	}

	charItems, err := charSeq.Items()
	if err != nil {
		return nil, errors.Wrap(err, "order expanded char items")
	}

	// Build map id->value and ordered keys slice.
	keys := make([]RMV6CrdtID, 0, len(charItems))
	vals := map[RMV6CrdtID]charVal{}
	for _, it := range charItems {
		keys = append(keys, it.ItemID)
		vals[it.ItemID] = it.Value
	}

	ypos := rt.PosY + rmv6TextTopY

	for len(keys) > 0 {
		startID := RMV6EndMarker
		if v, ok := vals[keys[0]]; ok && !v.isFmt && v.ch == "\n" {
			startID = keys[0]
			keys = keys[1:]
		}

		// Consume this paragraph until newline (formatting codes skipped).
		for len(keys) > 0 {
			v := vals[keys[0]]
			if v.isFmt {
				keys = keys[1:]
				continue
			}
			if v.ch == "\n" {
				break
			}
			anchorPos[keys[0]] = ypos
			keys = keys[1:]
		}

		anchorPos[startID] = ypos

		style := uint8(1) // default PLAIN
		if rt.Styles != nil {
			if s, ok := rt.Styles[startID]; ok {
				style = s
			}
		}
		lh, ok := rmv6LineHeights[style]
		if !ok {
			lh = 70
		}

		ypos += lh
		if math.IsNaN(ypos) || math.IsInf(ypos, 0) {
			return nil, errors.New("invalid ypos while building anchor positions")
		}
	}

	return anchorPos, nil
}
