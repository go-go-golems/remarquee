package rmdoc

import (
	"strings"

	"github.com/pkg/errors"
)

type RMV6TextParagraph struct {
	StartID RMV6CrdtID
	Style   uint8
	Text    string
}

// BuildRMV6TextDocument extracts a minimal per-paragraph text view from RootTextBlock.
//
// It mirrors rmscene.text.TextDocument.from_scene_item closely enough to support rendering and
// anchor debugging, but intentionally ignores inline formatting spans for now.
func BuildRMV6TextDocument(rt *RMV6RootText) ([]RMV6TextParagraph, error) {
	if rt == nil {
		return nil, nil
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

	type charVal struct {
		isFmt bool
		fmt   uint32
		ch    string
	}

	// Expand to single-character items (mirrors rmscene.text.expand_text_items).
	charSeq := NewRMV6CrdtSequence[charVal]()
	for _, item := range ordered {
		if item.DeletedLength > 0 {
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
				ItemID:  item.ItemID,
				LeftID:  item.LeftID,
				RightID: item.RightID,
				Value:   charVal{isFmt: true, fmt: uint32(item.Value.FormatCode)},
			})
			continue
		}

		runes := []rune(item.Value.Text)
		if len(runes) == 0 {
			continue
		}

		itemID := item.ItemID
		leftID := item.LeftID
		for _, r := range runes[:len(runes)-1] {
			rightID := RMV6CrdtID{Part1: itemID.Part1, Part2: itemID.Part2 + 1}
			_ = charSeq.Add(RMV6CrdtSequenceItem[charVal]{
				ItemID:  itemID,
				LeftID:  leftID,
				RightID: rightID,
				Value:   charVal{ch: string(r)},
			})
			leftID = itemID
			itemID = rightID
		}
		_ = charSeq.Add(RMV6CrdtSequenceItem[charVal]{
			ItemID:  itemID,
			LeftID:  leftID,
			RightID: item.RightID,
			Value:   charVal{ch: string(runes[len(runes)-1])},
		})
	}

	charItems, err := charSeq.Items()
	if err != nil {
		return nil, errors.Wrap(err, "order expanded char items")
	}

	// Ensure END_MARKER has a style (matches rmscene behavior).
	styles := rt.Styles
	if styles == nil {
		styles = map[RMV6CrdtID]uint8{}
	}

	var paragraphs []RMV6TextParagraph

	i := 0
	for i < len(charItems) {
		startID := RMV6EndMarker
		if !charItems[i].Value.isFmt && charItems[i].Value.ch == "\n" {
			startID = charItems[i].ItemID
			i++
		}

		var b strings.Builder
		for i < len(charItems) {
			v := charItems[i].Value
			if v.isFmt {
				i++
				continue
			}
			if v.ch == "\n" {
				break
			}
			b.WriteString(v.ch)
			i++
		}

		// Paragraph styles are keyed by paragraph start IDs. If no explicit style exists
		// for this paragraph, rmscene defaults to PLAIN (it does not apply END_MARKER's style).
		style := uint8(1) // PLAIN
		if s, ok := styles[startID]; ok {
			style = s
		}

		paragraphs = append(paragraphs, RMV6TextParagraph{
			StartID: startID,
			Style:   style,
			Text:    b.String(),
		})
	}

	return paragraphs, nil
}

func HasNonEmptyRMV6Text(paragraphs []RMV6TextParagraph) bool {
	for _, p := range paragraphs {
		if strings.TrimSpace(p.Text) != "" {
			return true
		}
	}
	return false
}
