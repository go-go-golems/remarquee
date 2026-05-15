package mdpdf

import "testing"

func TestStripYAMLFrontmatter_NoFrontmatter(t *testing.T) {
	in := "# Title\n\nHello\n"
	out := StripYAMLFrontmatter(in)
	if out != in {
		t.Fatalf("expected input unchanged, got: %q", out)
	}
}

func TestStripYAMLFrontmatter_StripsBlock(t *testing.T) {
	in := "---\nTitle: Test\nTopics:\n  - backend\n---\n# Body\n\nHello\n"
	out := StripYAMLFrontmatter(in)
	want := "# Body\n\nHello\n"
	if out != want {
		t.Fatalf("unexpected output.\nwant:\n%q\ngot:\n%q", want, out)
	}
}

func TestStripYAMLFrontmatter_RequiresDelimiterLine(t *testing.T) {
	// Starts with --- but first line is not exactly "---" after trimming.
	in := "----\nTitle: Test\n---\n# Body\n"
	out := StripYAMLFrontmatter(in)
	if out != in {
		t.Fatalf("expected input unchanged, got: %q", out)
	}
}

func TestNormalizeListSpacing_InsertsBlankLineBeforeList(t *testing.T) {
	in := "Paragraph\n- item 1\n- item 2\n"
	out := NormalizeListSpacing(in)
	want := "Paragraph\n\n- item 1\n- item 2\n"
	if out != want {
		t.Fatalf("unexpected output.\nwant:\n%q\ngot:\n%q", want, out)
	}
}

func TestNormalizeListSpacing_DoesNotInsertBetweenListItems(t *testing.T) {
	in := "Paragraph\n\n- item 1\n- item 2\n"
	out := NormalizeListSpacing(in)
	if out != in {
		t.Fatalf("expected unchanged, got: %q", out)
	}
}

func TestNormalizeListSpacing_OrderedList(t *testing.T) {
	in := "Paragraph\n1. first\n2. second\n"
	out := NormalizeListSpacing(in)
	want := "Paragraph\n\n1. first\n2. second\n"
	if out != want {
		t.Fatalf("unexpected output.\nwant:\n%q\ngot:\n%q", want, out)
	}
}

func TestFlattenDeepLists_CapsNestingAt4(t *testing.T) {
	in := "- level 1\n  - level 2\n    - level 3\n      - level 4\n        - level 5\n          - level 6\n"
	out := FlattenDeepLists(in, 4)
	want := "- level 1\n  - level 2\n    - level 3\n      - level 4\n      - level 5\n      - level 6\n"
	if out != want {
		t.Fatalf("unexpected output.\nwant:\n%q\ngot:\n%q", want, out)
	}
}

func TestFlattenDeepLists_NonListLinesUnchanged(t *testing.T) {
	if FlattenDeepLists("hello\n", 4) != "hello\n" {
		t.Fatal("non-list lines should be unchanged")
	}
}

func TestFlattenDeepLists_MaxDepth1(t *testing.T) {
	in := "- level 1\n  - level 2\n    - level 3\n"
	out := FlattenDeepLists(in, 1)
	// All items should be at depth 1 (0 spaces)
	want := "- level 1\n- level 2\n- level 3\n"
	if out != want {
		t.Fatalf("unexpected output.\nwant:\n%q\ngot:\n%q", want, out)
	}
}
