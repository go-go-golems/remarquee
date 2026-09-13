package mdpdf

import (
	"slices"
	"strings"
	"testing"
)

func TestBuildPandocArgsDisablesYAMLMetadataBlocks(t *testing.T) {
	opts := DefaultPandocOptions()
	opts.TOC = true
	opts.TOCDepth = 2
	opts.HighlightStyle = "tango"
	opts.Listings = true

	args := buildPandocArgs("input.md", "/tmp/output.pdf", opts, []string{"header.tex"})

	if len(args) == 0 || !strings.HasPrefix(args[0], "--from=markdown") {
		t.Fatalf("expected --from first, got %#v", args)
	}
	if !strings.Contains(args[0], "-yaml_metadata_block") {
		t.Fatalf("expected YAML metadata extension to be disabled, got %q", args[0])
	}
	count := 0
	for _, arg := range args {
		if strings.HasPrefix(arg, "--from=") {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected --from flag exactly once, got %d in %#v", count, args)
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

func TestBuildPandocArgsCustomFromFormat(t *testing.T) {
	opts := DefaultPandocOptions()
	// Deliberately different from DefaultFromFormat: this test proves a
	// custom format fully replaces the default, not that any specific
	// default exists.
	opts.FromFormat = "commonmark-yaml_metadata_block"

	args := buildPandocArgs("input.md", "/tmp/output.pdf", opts, nil)

	if len(args) == 0 || args[0] != "--from=commonmark-yaml_metadata_block" {
		t.Fatalf("expected custom --from first, got %#v", args)
	}
	count := 0
	for _, arg := range args {
		if arg == "--from=commonmark-yaml_metadata_block" {
			count++
		}
		if arg == "--from="+DefaultFromFormat {
			t.Fatalf("default --from must not appear when a custom format is set: %#v", args)
		}
	}
	if count != 1 {
		t.Fatalf("expected custom --from exactly once, got %d in %#v", count, args)
	}
}

func TestBuildPandocArgsEmptyFromFormatFallsBackToDefault(t *testing.T) {
	opts := DefaultPandocOptions()
	opts.FromFormat = ""

	args := buildPandocArgs("input.md", "/tmp/output.pdf", opts, nil)

	if len(args) == 0 || args[0] != "--from="+DefaultFromFormat {
		t.Fatalf("expected default --from on empty FromFormat, got %#v", args)
	}
}
