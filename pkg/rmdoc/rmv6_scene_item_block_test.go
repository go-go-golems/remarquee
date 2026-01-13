package rmdoc

import (
	"bytes"
	"testing"
)

func TestParseRMV6SceneItemBlocks_FindsAtLeastOne(t *testing.T) {
	root := remarqueeRootFromThisFile(t)
	fixture := root + "/cmd/remarquee-ui/testdata/cpage-pdf.rmdoc"

	rmBytes := readFirstRMFileFromRMDoc(t, fixture)

	blocks, err := ParseRMV6SceneItemBlocks(bytes.NewReader(rmBytes))
	if err != nil {
		t.Fatalf("ParseRMV6SceneItemBlocks: %v", err)
	}
	if len(blocks) == 0 {
		t.Fatalf("expected at least one scene item block, got 0")
	}

	// Sanity: ensure at least one block has non-zero ids.
	foundNonZero := false
	for _, b := range blocks {
		if b.ParentID != RMV6EndMarker {
			foundNonZero = true
			break
		}
	}
	if !foundNonZero {
		t.Fatalf("expected at least one block with non-zero ParentID")
	}
}
