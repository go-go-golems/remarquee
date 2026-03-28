package mdpdf

import (
	"strings"

	"github.com/pkg/errors"
)

const (
	MarkdownLayoutDefault = "default"
	MarkdownLayoutEditor  = "editor"
)

const editorLayoutGeometry = "top=1in,bottom=1.15in,left=1.1in,right=1.9in"

const editorLayoutLatexHeader = `\usepackage{setspace}
\setstretch{1.18}
\setlength{\parskip}{0.7em}
\setlength{\parindent}{0pt}
\geometry{top=1in,bottom=1.15in,left=1.1in,right=1.9in}
`

func NormalizeMarkdownLayout(layout string) string {
	layout = strings.ToLower(strings.TrimSpace(layout))
	if layout == "" {
		return MarkdownLayoutDefault
	}
	return layout
}

func ApplyMarkdownLayoutPreset(opts *PandocOptions, layout string) error {
	switch NormalizeMarkdownLayout(layout) {
	case MarkdownLayoutDefault:
		return nil
	case MarkdownLayoutEditor:
		opts.Geometry = editorLayoutGeometry
		opts.ExtraLatexHeader = editorLayoutLatexHeader
		return nil
	default:
		return errors.Errorf("unknown markdown layout %q (valid: %s, %s)", layout, MarkdownLayoutDefault, MarkdownLayoutEditor)
	}
}
