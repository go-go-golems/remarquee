package upload

import (
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/remarquee/pkg/mdpdf"
	"github.com/spf13/cobra"
)

const MermaidSectionSlug = "mermaid"

// NewMermaidSection creates a Glazed section for all mermaid-related flags.
// When added to a cobra command, these flags appear under a "Mermaid flags"
// heading in --help instead of mixed into the default "Flags" section.
func NewMermaidSection() (*schema.SectionImpl, error) {
	return schema.NewSection(
		MermaidSectionSlug,
		"Mermaid flags",
		schema.WithDescription("Control how Mermaid code blocks are rendered to diagrams in the PDF"),
		schema.WithFields(
			fields.New(
				"mermaid",
				fields.TypeBool,
				fields.WithDefault(true),
				fields.WithHelp("Render Mermaid code blocks as diagrams (requires mmdc)"),
			),
			fields.New(
				"mmdc-path",
				fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("Path to mmdc binary (default: auto-detect from $PATH)"),
			),
			fields.New(
				"mermaid-scale",
				fields.TypeInteger,
				fields.WithDefault(2),
				fields.WithHelp("Pixel scale for rendered Mermaid diagrams"),
			),
			fields.New(
				"mermaid-theme",
				fields.TypeString,
				fields.WithDefault("default"),
				fields.WithHelp("Mermaid theme: default, dark, forest, neutral"),
			),
			fields.New(
				"mermaid-bg",
				fields.TypeString,
				fields.WithDefault("white"),
				fields.WithHelp("Background color for Mermaid diagrams"),
			),
			fields.New(
				"mermaid-width",
				fields.TypeInteger,
				fields.WithDefault(0),
				fields.WithHelp("Max width in pixels for Mermaid diagrams (0 = auto)"),
			),
			fields.New(
				"mermaid-no-sandbox",
				fields.TypeBool,
				fields.WithDefault(true),
				fields.WithHelp("Pass --no-sandbox to Puppeteer/Chromium (default: true, safe for CLI use)"),
			),
			fields.New(
				"mermaid-pdf-width",
				fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("Display width for Mermaid diagrams in PDF (e.g. 50%, 400px, 10cm). Empty = fill page width"),
			),
		),
	)
}

// mermaidFlags is the parsed representation of the mermaid section flags.
type mermaidFlags struct {
	Mermaid          bool   `glazed:"mermaid"`
	MmdcPath         string `glazed:"mmdc-path"`
	MermaidScale     int    `glazed:"mermaid-scale"`
	MermaidTheme     string `glazed:"mermaid-theme"`
	MermaidBg        string `glazed:"mermaid-bg"`
	MermaidWidth     int    `glazed:"mermaid-width"`
	MermaidNoSandbox bool   `glazed:"mermaid-no-sandbox"`
	MermaidPDFWidth  string `glazed:"mermaid-pdf-width"`
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
	cfg.NoSandbox = f.MermaidNoSandbox
	cfg.PDFWidth = f.MermaidPDFWidth
	return &cfg
}

// parseMermaidFlags parses the mermaid section from cobra flags. The flags are
// registered via a Glazed section for grouped help output, but plain Cobra
// accessors keep parsing simple and include defaults for unchanged flags.
func parseMermaidFlags(cmd *cobra.Command) (*mermaidFlags, error) {
	flags := cmd.Flags()

	mermaid, err := flags.GetBool("mermaid")
	if err != nil {
		return nil, err
	}
	mmdcPath, err := flags.GetString("mmdc-path")
	if err != nil {
		return nil, err
	}
	mermaidScale, err := flags.GetInt("mermaid-scale")
	if err != nil {
		return nil, err
	}
	mermaidTheme, err := flags.GetString("mermaid-theme")
	if err != nil {
		return nil, err
	}
	mermaidBg, err := flags.GetString("mermaid-bg")
	if err != nil {
		return nil, err
	}
	mermaidWidth, err := flags.GetInt("mermaid-width")
	if err != nil {
		return nil, err
	}
	mermaidNoSandbox, err := flags.GetBool("mermaid-no-sandbox")
	if err != nil {
		return nil, err
	}
	mermaidPDFWidth, err := flags.GetString("mermaid-pdf-width")
	if err != nil {
		return nil, err
	}

	return &mermaidFlags{
		Mermaid:          mermaid,
		MmdcPath:         mmdcPath,
		MermaidScale:     mermaidScale,
		MermaidTheme:     mermaidTheme,
		MermaidBg:        mermaidBg,
		MermaidWidth:     mermaidWidth,
		MermaidNoSandbox: mermaidNoSandbox,
		MermaidPDFWidth:  mermaidPDFWidth,
	}, nil
}

func mermaidConfigFromCommand(cmd *cobra.Command) (*mdpdf.MermaidRendererConfig, error) {
	f, err := parseMermaidFlags(cmd)
	if err != nil {
		return nil, err
	}
	return f.ToConfig(), nil
}

// addMermaidFlagsToCommand adds the mermaid section flags to a cobra command
// and registers them as a Glazed flag group for grouped help output.
func addMermaidFlagsToCommand(cmd *cobra.Command) error {
	section, err := NewMermaidSection()
	if err != nil {
		return err
	}
	return section.AddSectionToCobraCommand(cmd)
}

// addResolveImagesFlag adds the --resolve-images flag to the "Flags" group
// (not the mermaid section, since it's a general image feature).
func addResolveImagesFlag(cmd *cobra.Command, target *bool) {
	cmd.Flags().BoolVar(target, "resolve-images", true, "Resolve and embed local image references")
}
