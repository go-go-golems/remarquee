package mdpdf

import (
	"slices"
	"testing"
)

func TestBuildPandocArgsDisablesYAMLMetadataBlocks(t *testing.T) {
	opts := DefaultPandocOptions()
	opts.TOC = true
	opts.TOCDepth = 2
	opts.HighlightStyle = "tango"
	opts.Listings = true

	args := buildPandocArgs("input.md", "/tmp/output.pdf", opts, []string{"header.tex"})

	if len(args) == 0 || args[0] != "--from=markdown-yaml_metadata_block" {
		t.Fatalf("expected YAML metadata extension to be disabled first, got %#v", args)
	}
	count := 0
	for _, arg := range args {
		if arg == "--from=markdown-yaml_metadata_block" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected YAML metadata extension flag exactly once, got %d in %#v", count, args)
	}

	for _, want := range []string{
		"input.md",
		"-o",
		"/tmp/output.pdf",
		"--pdf-engine=xelatex",
		"--toc",
		"--toc-depth=2",
		"--highlight-style=tango",
		"--listings",
		"-H",
		"header.tex",
	} {
		if !slices.Contains(args, want) {
			t.Errorf("expected argument %q in %#v", want, args)
		}
	}
}
