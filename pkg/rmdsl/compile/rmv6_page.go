package compile

import (
	"bytes"

	"github.com/go-go-golems/remarquee/pkg/rmdoc"
	"github.com/google/uuid"
)

type rmv6LayerGroup struct {
	ID          rmdoc.RMV6CrdtID
	LabelTS     rmdoc.RMV6CrdtID
	GroupItemID rmdoc.RMV6CrdtID
	Layer       CompiledLayer
}

func buildRMV6Page(page CompiledPage, structGen *crdtIDGen, lineGen *crdtIDGen, authorID uint8, authorUUID uuid.UUID) ([]byte, error) {
	var buf bytes.Buffer
	w := newRMV6Writer(&buf)

	if err := w.writeHeader(); err != nil {
		return nil, err
	}

	rootID := rmdoc.RMV6RootGroupID
	if err := writeAuthorIdsBlock(w, uint16(authorID), authorUUID); err != nil {
		return nil, err
	}
	if err := writeMigrationInfoBlock(w, rmdoc.RMV6CrdtID{Part1: 1, Part2: 1}, true); err != nil {
		return nil, err
	}
	if err := writePageInfoBlock(w, 1, 0, 0, 0, 0); err != nil {
		return nil, err
	}
	if err := writeSceneInfoBlock(w, rmv6ZeroID, true, true, 1620, 2160); err != nil {
		return nil, err
	}

	layers := make([]rmv6LayerGroup, 0, len(page.Layers))

	for _, layer := range page.Layers {
		groupID := structGen.Next()
		labelTS := structGen.Next()
		groupItemID := structGen.Next()
		if err := writeSceneTreeBlock(w, groupID, rootID); err != nil {
			return nil, err
		}
		layers = append(layers, rmv6LayerGroup{
			ID:          groupID,
			LabelTS:     labelTS,
			GroupItemID: groupItemID,
			Layer:       layer,
		})
	}

	if err := writeTreeNodeBlock(w, rootID, "", rmv6ZeroID); err != nil {
		return nil, err
	}
	for _, g := range layers {
		if err := writeTreeNodeBlock(w, g.ID, layerLabel(g.Layer, len(layers)), g.LabelTS); err != nil {
			return nil, err
		}
	}

	prevGroupItem := rmdoc.RMV6EndMarker
	for _, g := range layers {
		if err := writeSceneGroupItemBlock(w, rootID, g.GroupItemID, prevGroupItem, rmdoc.RMV6EndMarker, g.ID); err != nil {
			return nil, err
		}
		prevGroupItem = g.GroupItemID
	}

	for _, g := range layers {
		prevLine := rmdoc.RMV6EndMarker
		for _, stroke := range g.Layer.Strokes {
			itemID := lineGen.Next()
			payload, err := encodeRMV6LinePayload(stroke)
			if err != nil {
				return nil, err
			}
			if err := writeSceneLineItemBlock(w, g.ID, itemID, prevLine, rmdoc.RMV6EndMarker, payload); err != nil {
				return nil, err
			}
			prevLine = itemID
		}
	}

	return buf.Bytes(), nil
}

func layerLabel(layer CompiledLayer, total int) string {
	if layer.Name != "" {
		return layer.Name
	}
	if total == 1 {
		return "Layer 1"
	}
	return "Layer"
}
