package mdpdf

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

type PandocOptions struct {
	PandocPath       string
	PDFEngine        string
	MainFont         string
	MonoFont         string
	Geometry         string
	LatexHeaderFile  string
	ExtraLatexHeader string

	TOC      bool
	TOCDepth int

	HighlightStyle string
	Listings       bool
}

func DefaultPandocOptions() PandocOptions {
	return PandocOptions{
		PandocPath: "pandoc",
		PDFEngine:  "xelatex",
		MainFont:   "DejaVu Sans",
		MonoFont:   "DejaVu Sans Mono",
		Geometry:   "margin=1in",
	}
}

const defaultLatexHeader = `\usepackage{enumitem}
\setlist[itemize]{leftmargin=*,topsep=0.5em,itemsep=0.3em,parsep=0.2em}
\setlist[enumerate]{leftmargin=*,topsep=0.5em,itemsep=0.3em,parsep=0.2em}
\usepackage{geometry}
\geometry{margin=1in}
`

func ConvertMarkdownFileToPDF(ctx context.Context, mdPath string, outPDF string, opts PandocOptions) error {
	if opts.PandocPath == "" {
		opts.PandocPath = "pandoc"
	}
	if opts.PDFEngine == "" {
		opts.PDFEngine = "xelatex"
	}
	if opts.MainFont == "" {
		opts.MainFont = "DejaVu Sans"
	}
	if opts.MonoFont == "" {
		opts.MonoFont = "DejaVu Sans Mono"
	}
	if opts.Geometry == "" {
		opts.Geometry = "margin=1in"
	}

	mdBytes, err := os.ReadFile(mdPath)
	if err != nil {
		return errors.Wrap(err, "failed to read markdown file")
	}
	body := StripYAMLFrontmatter(string(mdBytes))

	tmpDir, err := os.MkdirTemp("", "remarquee-mdpdf-")
	if err != nil {
		return errors.Wrap(err, "failed to create temp directory")
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Resolve local image paths before other preprocessing so that pandoc
	// can find referenced files from the temp directory.
	sourceDir := filepath.Dir(mdPath)
	body, err = ResolveImagePaths(body, sourceDir, tmpDir)
	if err != nil {
		return errors.Wrap(err, "failed to resolve image paths")
	}

	body = NormalizeListSpacing(body)
	body = FlattenDeepLists(body, 4)

	// Keep temporary helper filenames deliberately boring. Pandoc treats '#'
	// in input paths as a URI fragment separator in some readers, so using
	// filepath.Base(mdPath) here breaks Markdown files whose names contain
	// issue/PR numbers like "PR #6.md".
	inputPath := filepath.Join(tmpDir, "input.md")
	if err := os.WriteFile(inputPath, []byte(body), 0o644); err != nil { // #nosec G703 -- inputPath is a fixed filename inside an os.MkdirTemp directory.
		return errors.Wrap(err, "failed to write preprocessed markdown")
	}

	headerPaths := make([]string, 0, 2)
	if opts.LatexHeaderFile != "" {
		headerPaths = append(headerPaths, opts.LatexHeaderFile)
	} else {
		headerPath := filepath.Join(tmpDir, "header.tex")
		if err := os.WriteFile(headerPath, []byte(defaultLatexHeader), 0o644); err != nil {
			return errors.Wrap(err, "failed to write latex header")
		}
		headerPaths = append(headerPaths, headerPath)
	}
	if strings.TrimSpace(opts.ExtraLatexHeader) != "" {
		extraHeaderPath := filepath.Join(tmpDir, "extra-header.tex")
		if err := os.WriteFile(extraHeaderPath, []byte(opts.ExtraLatexHeader), 0o644); err != nil {
			return errors.Wrap(err, "failed to write extra latex header")
		}
		headerPaths = append(headerPaths, extraHeaderPath)
	}

	argv := []string{
		inputPath,
		"-o", outPDF,
		"--pdf-engine=" + opts.PDFEngine,
		"--standalone",
		"-V", "mainfont=" + opts.MainFont,
		"-V", "monofont=" + opts.MonoFont,
		"-V", "geometry:" + opts.Geometry,
	}
	for _, headerPath := range headerPaths {
		argv = append(argv, "-H", headerPath)
	}

	if opts.TOC {
		argv = append(argv, "--toc")
		if opts.TOCDepth > 0 {
			argv = append(argv, fmt.Sprintf("--toc-depth=%d", opts.TOCDepth))
		}
	}
	if opts.HighlightStyle != "" {
		argv = append(argv, "--highlight-style="+opts.HighlightStyle)
	}
	if opts.Listings {
		argv = append(argv, "--listings")
	}

	cmd := exec.CommandContext(ctx, opts.PandocPath, argv...) // #nosec G204 -- this package intentionally invokes the configured pandoc binary with explicit argv.
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "pandoc failed: %s", string(out))
	}

	return nil
}
