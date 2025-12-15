package mdpdf

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
)

type PandocOptions struct {
	PandocPath      string
	PDFEngine       string
	MainFont        string
	MonoFont        string
	Geometry        string
	LatexHeaderFile string
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
	body = NormalizeListSpacing(body)

	tmpDir, err := os.MkdirTemp("", "remarquee-mdpdf-")
	if err != nil {
		return errors.Wrap(err, "failed to create temp directory")
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	inputPath := filepath.Join(tmpDir, filepath.Base(mdPath)+".input.md")
	if err := os.WriteFile(inputPath, []byte(body), 0o644); err != nil {
		return errors.Wrap(err, "failed to write preprocessed markdown")
	}

	headerPath := opts.LatexHeaderFile
	if headerPath == "" {
		headerPath = filepath.Join(tmpDir, filepath.Base(mdPath)+".header.tex")
		if err := os.WriteFile(headerPath, []byte(defaultLatexHeader), 0o644); err != nil {
			return errors.Wrap(err, "failed to write latex header")
		}
	}

	argv := []string{
		inputPath,
		"-o", outPDF,
		"--pdf-engine=" + opts.PDFEngine,
		"--standalone",
		"-H", headerPath,
		"-V", "mainfont=" + opts.MainFont,
		"-V", "monofont=" + opts.MonoFont,
		"-V", "geometry:" + opts.Geometry,
	}

	cmd := exec.CommandContext(ctx, opts.PandocPath, argv...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "pandoc failed: %s", string(out))
	}

	return nil
}
