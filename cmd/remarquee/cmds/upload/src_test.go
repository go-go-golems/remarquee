package upload

import "testing"

func TestFenceForCode_GrowsPastBackticks(t *testing.T) {
	if got := fenceForCode("no backticks"); got != "```" {
		t.Fatalf("unexpected fence: %q", got)
	}
	if got := fenceForCode("```\ncode\n```"); got != "````" {
		t.Fatalf("unexpected fence: %q", got)
	}
	if got := fenceForCode("``````"); got != "```````" {
		t.Fatalf("unexpected fence: %q", got)
	}
}

func TestLanguageForPath_CommonExts(t *testing.T) {
	cases := map[string]string{
		"main.go":     "go",
		"foo.py":      "python",
		"foo.ts":      "typescript",
		"foo.tsx":     "typescript",
		"foo.c":       "c",
		"foo.hpp":     "cpp",
		"foo.rs":      "rust",
		"foo.sh":      "bash",
		"foo.unknown": "",
	}
	for p, want := range cases {
		if got := languageForPath(p); got != want {
			t.Fatalf("languageForPath(%q): want %q, got %q", p, want, got)
		}
	}
}
