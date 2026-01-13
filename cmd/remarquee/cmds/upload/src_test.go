package upload

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestBuildSourceBundleMarkdown_HasHeadingsAndPageBreaks(t *testing.T) {
	td := t.TempDir()
	a := filepath.Join(td, "a.go")
	b := filepath.Join(td, "sub", "b.c")
	if err := os.MkdirAll(filepath.Dir(b), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(a, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("#include <stdio.h>\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := buildSourceBundleMarkdown([]sourceInput{
		{AbsPath: a, RelPath: "a.go"},
		{AbsPath: b, RelPath: filepath.Join("sub", "b.c")},
	}, "path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "# a.go\n") {
		t.Fatalf("expected heading for a.go, got:\n%s", out)
	}
	if !strings.Contains(out, "# "+filepath.ToSlash(filepath.Join("sub", "b.c"))+"\n") {
		// RelPath is platform-specific; title uses RelPath as-is.
		t.Fatalf("expected heading for sub/b.c, got:\n%s", out)
	}
	if !strings.Contains(out, "\\newpage") {
		t.Fatalf("expected page break, got:\n%s", out)
	}
}
