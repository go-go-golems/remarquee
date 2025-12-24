package rmdoc

import (
	"bytes"
	"testing"
)

func TestBBoxForStroke_Pad(t *testing.T) {
	s := Stroke{
		Points: []StrokePoint{
			{X: 10, Y: 20},
			{X: 15, Y: 25},
			{X: 12, Y: 30},
		},
	}

	b, ok := BBoxForStroke(s, 2)
	if !ok {
		t.Fatalf("expected ok")
	}
	if b.MinX != 8 || b.MinY != 18 || b.MaxX != 17 || b.MaxY != 32 {
		t.Fatalf("unexpected bbox: %+v", b)
	}
}

func TestBBoxForStroke_Empty(t *testing.T) {
	_, ok := BBoxForStroke(Stroke{}, 0)
	if ok {
		t.Fatalf("expected not ok")
	}
}

func TestBBoxForRMV6SceneTree_Smoke(t *testing.T) {
	root := remarqueeRootFromThisFile(t)
	fixture := root + "/cmd/remarquee-ui/testdata/cpage-pdf.rmdoc"

	rmBytes := readFirstRMFileFromRMDoc(t, fixture)
	tree, err := ParseRMV6SceneTree(bytes.NewReader(rmBytes))
	if err != nil {
		t.Fatalf("ParseRMV6SceneTree: %v", err)
	}

	b, err := BBoxForRMV6SceneTree(tree, 0)
	if err != nil {
		t.Fatalf("BBoxForRMV6SceneTree: %v", err)
	}
	if b.IsEmpty() {
		t.Fatalf("expected non-empty bbox")
	}
}
