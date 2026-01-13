package rmdoc

import (
	"bytes"
	"context"
	"sort"
	"testing"
)

func TestParseRMV6SceneTree_StrokeColorsPresent_TestRmdoc(t *testing.T) {
	ctx := context.Background()
	path := "/home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee-ui/testdata/Test.rmdoc"

	doc, err := OpenFile(ctx, path)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	if len(doc.Pages) == 0 {
		t.Fatalf("expected at least one page in fixture")
	}

	seen := map[uint32]int{}
	seenPages := 0

	for _, p := range doc.Pages {
		if p.PageID == "" {
			continue
		}
		rm, ok, err := ReadRMFileFromArchive(ctx, path, p.PageID)
		if err != nil {
			t.Fatalf("ReadRMFileFromArchive(%s): %v", p.PageID, err)
		}
		if !ok || rm == nil || rm.Version != "V6" || len(rm.Bytes) == 0 {
			continue
		}

		tree, err := ParseRMV6SceneTree(bytes.NewReader(rm.Bytes))
		if err != nil {
			t.Fatalf("ParseRMV6SceneTree(pageID=%s): %v", p.PageID, err)
		}
		strokes, err := ExtractRMV6StrokesWithAnchors(tree)
		if err != nil {
			t.Fatalf("ExtractRMV6StrokesWithAnchors(pageID=%s): %v", p.PageID, err)
		}
		for _, s := range strokes {
			seen[s.Color]++
		}
		seenPages++
	}

	if seenPages == 0 {
		t.Fatalf("fixture had no V6 pages with annotation .rm files")
	}

	colors := make([]int, 0, len(seen))
	for c := range seen {
		colors = append(colors, int(c))
	}
	sort.Ints(colors)
	t.Logf("seen stroke colors: %v (counts=%v)", colors, seen)

	// We expect "Test.rmdoc" to contain colored strokes (not just black).
	// This verifies the parsing layer is carrying a non-zero color_id through.
	nonZero := 0
	for c := range seen {
		if c != 0 {
			nonZero++
		}
	}
	if nonZero == 0 {
		t.Fatalf("expected at least one non-black stroke color_id, got only %v", colors)
	}

	// Test.rmdoc also contains different-colored highlighter strokes. In V6, those can show up
	// as PenColorHighlight (9) in the line header, with the actual color specified by a trailing
	// RGBA marker. We want to ensure we decode that marker into one of the concrete highlight colors.
	hasConcreteHighlight := false
	for c := range seen {
		if c >= uint32(PenColorHighlightYellow) && c <= uint32(PenColorHighlightGray) {
			hasConcreteHighlight = true
			break
		}
	}
	if !hasConcreteHighlight {
		t.Fatalf("expected at least one concrete highlight color_id in [%d..%d], got %v",
			PenColorHighlightYellow, PenColorHighlightGray, colors)
	}
}
