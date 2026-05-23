package upload

import (
	"github.com/go-go-golems/remarquee/pkg/mdpdf"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

type mermaidFlags struct {
	Mermaid      bool
	MmdcPath     string
	MermaidScale int
	MermaidTheme string
	MermaidBg    string
	MermaidWidth int
}

func (f *mermaidFlags) ToConfig() *mdpdf.MermaidRendererConfig {
	if !f.Mermaid {
		return nil
	}
	cfg := mdpdf.DefaultMermaidRendererConfig()
	cfg.MmdcPath = f.MmdcPath
	if f.MermaidScale > 0 {
		cfg.Scale = f.MermaidScale
	}
	if f.MermaidTheme != "" {
		cfg.Theme = f.MermaidTheme
	}
	if f.MermaidBg != "" {
		cfg.BackgroundColor = f.MermaidBg
	}
	cfg.Width = f.MermaidWidth
	return &cfg
}

func configureMarkdownPandocOptions(
	flags *pflag.FlagSet,
	layout string,
	pandoc string,
	pdfEngine string,
	mainFont string,
	monoFont string,
	geometry string,
	latexHeaderFile string,
	mermaidCfg *mdpdf.MermaidRendererConfig,
) (mdpdf.PandocOptions, error) {
	opts := mdpdf.DefaultPandocOptions()
	opts.PandocPath = pandoc
	opts.PDFEngine = pdfEngine
	opts.MainFont = mainFont
	opts.MonoFont = monoFont
	opts.Mermaid = mermaidCfg

	if err := mdpdf.ApplyMarkdownLayoutPreset(&opts, layout); err != nil {
		return mdpdf.PandocOptions{}, err
	}
	if flags == nil {
		return mdpdf.PandocOptions{}, errors.New("flag set is required")
	}
	if flags.Changed("geometry") {
		opts.Geometry = geometry
	}
	if flags.Changed("latex-header-file") {
		opts.LatexHeaderFile = latexHeaderFile
	}

	return opts, nil
}
