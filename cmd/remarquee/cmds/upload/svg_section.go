package upload

import (
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/remarquee/pkg/mdpdf"
	"github.com/spf13/cobra"
)

const SVGSectionSlug = "svg"

// NewSVGSection creates a Glazed section for SVG-handling flags. When added to a
// cobra command, these flags appear under an "SVG flags" heading in --help.
func NewSVGSection() (*schema.SectionImpl, error) {
	return schema.NewSection(
		SVGSectionSlug,
		"SVG flags",
		schema.WithDescription("Control how inline <svg> blocks and HTML <img src=*.svg> tags are rendered"),
		schema.WithFields(
			fields.New(
				"svg",
				fields.TypeBool,
				fields.WithDefault(true),
				fields.WithHelp("Render inline <svg> blocks and HTML <img src=*.svg> tags as vector images"),
			),
			fields.New(
				"svg-converter",
				fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("Path to rsvg-convert or inkscape (default: auto-detect)"),
			),
			fields.New(
				"svg-default-width",
				fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("Width for extracted inline SVGs, e.g. 70% or 12cm (default: natural size)"),
			),
		),
	)
}

// svgFlags is the parsed representation of the svg section flags.
type svgFlags struct {
	SVG             bool   `glazed:"svg"`
	SVGConverter    string `glazed:"svg-converter"`
	SVGDefaultWidth string `glazed:"svg-default-width"`
}

// ToConfig builds an mdpdf.SVGRendererConfig. It returns nil when SVG handling
// is disabled so the pipeline treats it as a no-op.
func (f *svgFlags) ToConfig() *mdpdf.SVGRendererConfig {
	if !f.SVG {
		return nil
	}
	cfg := mdpdf.DefaultSVGRendererConfig()
	cfg.ConverterPath = f.SVGConverter
	cfg.DefaultWidth = f.SVGDefaultWidth
	return &cfg
}

// addSVGFlagsToCommand adds the SVG flags to a cobra command and registers them
// as a Glazed flag group for grouped help output.
func addSVGFlagsToCommand(cmd *cobra.Command) error {
	section, err := NewSVGSection()
	if err != nil {
		return err
	}
	return section.AddSectionToCobraCommand(cmd)
}

// svgConfigFromCommand parses the SVG section flags into a renderer config.
func svgConfigFromCommand(cmd *cobra.Command) (*mdpdf.SVGRendererConfig, error) {
	flags := cmd.Flags()

	enabled, err := flags.GetBool("svg")
	if err != nil {
		return nil, err
	}
	converter, err := flags.GetString("svg-converter")
	if err != nil {
		return nil, err
	}
	defaultWidth, err := flags.GetString("svg-default-width")
	if err != nil {
		return nil, err
	}

	f := svgFlags{SVG: enabled, SVGConverter: converter, SVGDefaultWidth: defaultWidth}
	return f.ToConfig(), nil
}
