package mdpdf

import "testing"

func TestListPassesPreserveLiteralRegions(t *testing.T) {
	cases := map[string]string{
		"dollar math":            "$$\nA = B\n+ C\n        - D\n$$\n",
		"dollar opening content": "$$ A = B\n+ C\n$$\n",
		"dollar closing content": "$$\nA = B\n+ C $$\n",
		"bracket math":           "\\[\nA = B\n+ C\n        - D\n\\]\n",
		"one line dollar":        "$$ A + B $$\n",
		"one line bracket":       "\\[ A + B \\]\n",
		"escaped dollar closer":  "$$\n\\$$\n+ C\n$$\n",
		"escaped bracket closer": "\\[\n\\\\]\n+ C\n\\]\n",
		"even backslashes close": "$$\nA \\\\$$\n",
		"backticks":              "```text\nparagraph\n+ code\n        1. code\n```\n",
		"tildes":                 "~~~text\nparagraph\n- code\n        * code\n~~~\n",
		"long fence":             "````text\n```\n+ code\n        - code\n`````\n",
		"wrong closer":           "```text\n~~~\n+ code\n        - code\n```\n",
		"closer with text":       "```text\n``` not a closer\n+ code\n```\n",
		"indented fence":         "  ~~~text\n  paragraph\n        - code\n  ~~~\n",
		"math inside code":       "```text\n$$\n+ code\n```\n",
		"code inside math":       "$$\n```\n+ C\n$$\n",
		"crlf":                   "$$\r\nA = B\r\n+ C\r\n        - D\r\n$$\r\n",
	}
	for name, block := range cases {
		t.Run(name, func(t *testing.T) {
			if got := NormalizeListSpacing(block); got != block {
				t.Fatalf("normalization changed literal text:\nwant %q\ngot  %q", block, got)
			}
			if got := FlattenDeepLists(block, 1); got != block {
				t.Fatalf("flattening changed literal text:\nwant %q\ngot  %q", block, got)
			}
			in := block + "- outside\n  - nested\n"
			want := block + "\n- outside\n- nested\n"
			got := FlattenDeepLists(NormalizeListSpacing(in), 1)
			if got != want {
				t.Fatalf("list handling did not resume after literal:\nwant %q\ngot  %q", want, got)
			}
			if again := FlattenDeepLists(NormalizeListSpacing(got), 1); again != got {
				t.Fatalf("repeated preprocessing is not stable: %q", again)
			}
		})
	}
}

func TestListPassesPreserveUnclosedRegions(t *testing.T) {
	for _, opener := range []string{"$$", `\[`, "```text", "~~~text"} {
		in := opener + "\nparagraph\n+ literal\n        - literal"
		if got := FlattenDeepLists(NormalizeListSpacing(in), 1); got != in {
			t.Errorf("unclosed %q changed: %q", opener, got)
		}
	}
}

func TestInlineCodeIsNotAnOpeningFence(t *testing.T) {
	in := "```inline```\n+ outside\n"
	want := "```inline```\n\n+ outside\n"
	if got := NormalizeListSpacing(in); got != want {
		t.Fatalf("inline backticks swallowed list: %q", got)
	}
}
