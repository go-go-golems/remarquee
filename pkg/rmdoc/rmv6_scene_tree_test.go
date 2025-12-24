package rmdoc

import (
	"bytes"
	"testing"
)

func TestParseRMV6SceneTree_GroupsAndLines(t *testing.T) {
	root := remarqueeRootFromThisFile(t)
	fixture := root + "/cmd/remarquee-ui/testdata/cpage-pdf.rmdoc"

	rmBytes := readFirstRMFileFromRMDoc(t, fixture)

	tree, err := ParseRMV6SceneTree(bytes.NewReader(rmBytes))
	if err != nil {
		t.Fatalf("ParseRMV6SceneTree: %v", err)
	}
	if tree.Root == nil || tree.Root.NodeID != RMV6RootGroupID {
		t.Fatalf("expected root group %v, got %+v", RMV6RootGroupID, tree.Root)
	}

	// Expect at least one non-root node was created by SceneTreeBlock(s).
	nonRootCount := 0
	for id := range tree.nodes {
		if id != RMV6RootGroupID {
			nonRootCount++
		}
	}
	if nonRootCount == 0 {
		t.Fatalf("expected at least one non-root group node")
	}

	// Expect at least one line item exists somewhere.
	foundLine := false
	for _, g := range tree.nodes {
		items, err := g.Children.Items()
		if err != nil {
			t.Fatalf("Children.Items: %v", err)
		}
		for _, it := range items {
			if it.Value.Kind == RMV6SceneItemLine && it.Value.Line != nil {
				foundLine = true
				break
			}
		}
		if foundLine {
			break
		}
	}
	if !foundLine {
		t.Fatalf("expected to find at least one line item in the tree")
	}
}
