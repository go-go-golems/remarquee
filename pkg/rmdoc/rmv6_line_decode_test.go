package rmdoc

import (
	"bytes"
	"testing"
)

func TestDecodeRMV6Line_FromSceneTree(t *testing.T) {
	root := remarqueeRootFromThisFile(t)
	fixture := root + "/cmd/remarquee-ui/testdata/cpage-pdf.rmdoc"

	rmBytes := readFirstRMFileFromRMDoc(t, fixture)

	tree, err := ParseRMV6SceneTree(bytes.NewReader(rmBytes))
	if err != nil {
		t.Fatalf("ParseRMV6SceneTree: %v", err)
	}

	var (
		found  bool
		stroke *Stroke
	)

	for _, g := range tree.nodes {
		items, err := g.Children.Items()
		if err != nil {
			t.Fatalf("Children.Items: %v", err)
		}
		for _, it := range items {
			if it.Value.Kind == RMV6SceneItemLine && it.Value.Line != nil && len(it.Value.Line.Raw) > 0 {
				stroke, err = DecodeRMV6Line(it.Value.Line.BlockVersion, it.Value.Line.Raw)
				if err != nil {
					t.Fatalf("DecodeRMV6Line: %v", err)
				}
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		t.Fatalf("did not find a line to decode")
	}
	if stroke == nil {
		t.Fatalf("decoded stroke is nil")
	}
	if len(stroke.Points) == 0 {
		t.Fatalf("expected non-empty points")
	}
	if stroke.ThicknessScale == 0 {
		t.Fatalf("expected non-zero thickness scale")
	}
}
