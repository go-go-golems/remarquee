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

	// FromFormat is the pandoc input format string (the value of --from).
	// Pandoc Markdown extensions can only be toggled inside this string, and
	// pandoc's "last --from wins" rule means a hardcoded value silently
	// overrides any wrapper script's own -f flags — so it is configurable.
	// Empty means DefaultFromFormat.
	FromFormat string

	TOC      bool
	TOCDepth int

	HighlightStyle string
	Listings       bool

	// Mermaid configures Mermaid diagram rendering. If nil, Mermaid blocks
	// are left as plain-text code listings.
	Mermaid *MermaidRendererConfig

	// SVG configures inline <svg> block and HTML <img src=...svg> handling.
	// If nil, inline SVG content is left to pandoc (which drops it for LaTeX).
	SVG *SVGRendererConfig

	// ResolveImages controls whether local Markdown image paths are copied into
	// the pandoc temp directory and rewritten. DefaultPandocOptions enables it.
	ResolveImages bool
}

// DefaultFromFormat is the pandoc input format used for markdown
// conversion. The yaml_metadata_block extension is disabled so ordinary
// Markdown thematic breaks (---) are not interpreted as metadata delimiters
// (the converter strips docmgr-style frontmatter itself before invoking
// pandoc).
//
// tex_math_single_backslash is enabled so ChatGPT-style \(...\) and \[...\]
// math delimiters parse as math. Without it the reader consumes the delimiter
// tokens and hands the inner TeX commands to LaTeX in text mode, where they
// are fatal errors ("Command \mathcal allowed only in math mode", "Missing $
// inserted") — the dominant failure class of the 2026-09-13 sync. surf-cli
// now normalizes delimiters to dollar math at extraction and backfill time;
// this keeps remarquee safe for any legacy file that bypassed that pipeline.
const DefaultFromFormat = "markdown-yaml_metadata_block+tex_math_single_backslash"

func DefaultPandocOptions() PandocOptions {
	svg := DefaultSVGRendererConfig()
	return PandocOptions{
		PandocPath:    "pandoc",
		PDFEngine:     "xelatex",
		MainFont:      "DejaVu Sans",
		MonoFont:      "DejaVu Sans Mono",
		Geometry:      "margin=1in",
		FromFormat:    DefaultFromFormat,
		ResolveImages: true,
		SVG:           &svg,
	}
}

const defaultLatexHeader = `\usepackage{stmaryrd}
\usepackage{centernot}
\usepackage{mathtools}
\usepackage{amscd}
\newcommand{\require}[1]{}
\usepackage{enumitem}
\setlist[itemize]{leftmargin=*,topsep=0.5em,itemsep=0.3em,parsep=0.2em}
\setlist[enumerate]{leftmargin=*,topsep=0.5em,itemsep=0.3em,parsep=0.2em}
\usepackage{geometry}
\geometry{margin=1in}
`

// The header comments for the math packages, in order:
//   - stmaryrd: \llbracket/\rrbracket, \rightsquigarrow (ChatGPT category theory)
//   - centernot: \centernot\Longrightarrow (negated arrows)
//   - mathtools: \xRightarrow and friends (extensible arrows)
//   - amscd: the CD commutative-diagram environment, the real-LaTeX
//     equivalent of MathJax's AMScd extension
//   - \require: MathJax's extension loader (\require{AMScd}) appears inside
//     ChatGPT math spans; it has no LaTeX meaning, so a no-op definition
//     lets the surrounding math compile instead of dying with
//     "Undefined control sequence".

func buildPandocArgs(inputPath string, outputPath string, opts PandocOptions, headerPaths []string) []string {
	fromFormat := opts.FromFormat
	if fromFormat == "" {
		fromFormat = DefaultFromFormat
	}
	argv := []string{
		"--from=" + fromFormat,
		inputPath,
		"-o", outputPath,
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

	return argv
}

func ConvertMarkdownFileToPDF(ctx context.Context, mdPath string, outPDF string, opts PandocOptions) error {
	absOutPDF, err := filepath.Abs(outPDF)
	if err != nil {
		return errors.Wrap(err, "failed to resolve output PDF path")
	}
	if opts.LatexHeaderFile != "" {
		absHeader, err := filepath.Abs(opts.LatexHeaderFile)
		if err != nil {
			return errors.Wrap(err, "failed to resolve latex header path")
		}
		opts.LatexHeaderFile = absHeader
	}

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

	if opts.ResolveImages {
		// Rewrite SVG <img> tags to Markdown image syntax before image path
		// resolution, so the resulting references are copied from the source dir.
		// Gated on ResolveImages because the rewrite produces a relative Markdown
		// reference that must be staged by ResolveImagePaths.
		body, err = ResolveHTMLImages(body, opts.SVG)
		if err != nil {
			return errors.Wrap(err, "failed to resolve HTML SVG images")
		}

		// Resolve local image paths before other preprocessing so that pandoc
		// can find referenced files from the temp directory.
		sourceDir := filepath.Dir(mdPath)
		body, err = ResolveImagePaths(body, sourceDir, tmpDir)
		if err != nil {
			return errors.Wrap(err, "failed to resolve image paths")
		}
	}

	// Extract and convert inline <svg> blocks. This writes finished assets
	// directly into <tmp>/images, so it runs after ResolveImagePaths (which
	// resolves relative paths against the source directory, not the temp dir).
	body, err = ResolveInlineSVGBlocks(ctx, body, tmpDir, opts.SVG)
	if err != nil {
		return errors.Wrap(err, "failed to resolve inline SVG blocks")
	}

	// Render Mermaid code blocks to images (if mmdc is available).
	body, err = RenderMermaidBlocks(ctx, body, tmpDir, opts.Mermaid)
	if err != nil {
		// Non-fatal: mermaid rendering errors are logged per-block.
		// Continue with unrendered blocks.
		_ = err
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

	argv := buildPandocArgs(inputPath, absOutPDF, opts, headerPaths)

	cmd := exec.CommandContext(ctx, opts.PandocPath, argv...) // #nosec G204 -- this package intentionally invokes the configured pandoc binary with explicit argv.
	// Set working directory to tmpDir so pandoc resolves relative image
	// paths (like ./images/mermaid-001.png) from the temp directory where
	// ResolveImagePaths and RenderMermaidBlocks placed them.
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "pandoc failed: %s", string(out))
	}

	return nil
}
