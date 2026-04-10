package rmdoc

import (
	"context"
	"fmt"
	"os"

	"github.com/go-go-golems/glazed/pkg/cli"
	glazecmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	pkg_rmdoc "github.com/go-go-golems/remarquee/pkg/rmdoc"
	rmdocrender "github.com/go-go-golems/remarquee/pkg/rmdoc/render"
)

type RenderV6Command struct {
	*glazecmds.CommandDescription
}

type RenderV6Settings struct {
	File string `glazed:"file"`
	Out  string `glazed:"out"`

	Force bool `glazed:"force"`

	CloudInputSettings
}

var _ glazecmds.BareCommand = &RenderV6Command{}
var _ glazecmds.GlazeCommand = &RenderV6Command{}

func NewRenderV6Command() (*RenderV6Command, error) {
	glazedLayer, err := settings.NewGlazedSection()
	if err != nil {
		return nil, err
	}
	commandSettingsLayer, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	flags := []*fields.Definition{
		fields.New(
			"out",
			fields.TypeString,
			fields.WithDefault(""),
			fields.WithHelp("Output PDF path (default: <input>-v6.pdf in current dir)"),
		),
		fields.New(
			"force",
			fields.TypeBool,
			fields.WithDefault(false),
			fields.WithHelp("Overwrite output file if it exists"),
		),
		fields.New(
			"file",
			fields.TypeString,
			fields.WithIsArgument(true),
			fields.WithRequired(true),
			fields.WithHelp("Path to the V6 .rmdoc file, or a remote cloud path when used with --cloud"),
		),
	}
	flags = append(flags, cloudInputParameterDefinitions()...)

	cmdDesc := glazecmds.NewCommandDescription(
		"render-v6",
		glazecmds.WithShort("Render a V6 (cPages) .rmdoc to an annotated PDF (strokes + smart highlights)"),
		glazecmds.WithLong(`
Render a V6 (cPages) .rmdoc to an annotated PDF using the Go V6 parser + merge pipeline:
- background PDF is assembled in UI page order from .content (cPages)
- V6 strokes are merged on top using remarks-style positioning math
- V6 GlyphRange rectangles are emitted as PDF Highlight annotations ("smart highlights")

Notes:
- Only supports PDF-backed/notebook cPages archives (not EPUB).
- This is still a milestone renderer (brush fidelity, typed text output, and PNGs are future work).
`),
		glazecmds.WithFlags(flags...),
		glazecmds.WithSections(glazedLayer, commandSettingsLayer),
	)

	return &RenderV6Command{CommandDescription: cmdDesc}, nil
}

func (c *RenderV6Command) Run(ctx context.Context, parsedValues *values.Values) error {
	s := &RenderV6Settings{}
	if err := parsedValues.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
		return err
	}
	if err := initializeCloudInputSettings(parsedValues, &s.CloudInputSettings); err != nil {
		return err
	}

	res, err := c.execute(ctx, s)
	if err != nil {
		return err
	}

	fmt.Printf("ok: wrote %s\n", res.Output)
	return nil
}

type renderV6Execution struct {
	Input       string
	InputSource string
	Output      string
	Schema      string
	Type        string
	Pages       int
}

func (c *RenderV6Command) execute(ctx context.Context, s *RenderV6Settings) (*renderV6Execution, error) {
	input, err := ResolveRMDocInput(ctx, s.File, s.CloudInputSettings)
	if err != nil {
		return nil, err
	}
	defer func() { _ = input.Cleanup() }()

	doc, err := pkg_rmdoc.OpenFile(ctx, input.LocalPath)
	if err != nil {
		return nil, err
	}
	if doc.Schema != pkg_rmdoc.SchemaCPages {
		return nil, errors.Errorf("render-v6 only supports cPages/V6 archives; detected schema=%s", schemaString(doc.Schema))
	}
	if doc.Type == pkg_rmdoc.DocTypeEPUB {
		return nil, errors.New("render-v6: epub not supported")
	}

	out := s.Out
	if out == "" {
		out = defaultOutputPath(input.LocalPath, "-v6.pdf")
	}
	if err := ensureOutputWritable(out, s.Force); err != nil {
		return nil, err
	}

	res, err := rmdocrender.MergeRMDocV6OntoBackgroundPDFWithInfo(ctx, input.LocalPath, rmdocrender.V6MergeOptions{})
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(out, res.PDF, 0o644); err != nil {
		return nil, errors.Wrap(err, "write output pdf")
	}

	return &renderV6Execution{
		Input:       input.RequestedPath,
		InputSource: input.Source,
		Output:      out,
		Schema:      schemaString(doc.Schema),
		Type:        docTypeString(doc.Type),
		Pages:       len(doc.Pages),
	}, nil
}

func (c *RenderV6Command) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedValues *values.Values,
	gp middlewares.Processor,
) error {
	s := &RenderV6Settings{}
	if err := parsedValues.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
		return err
	}
	if err := initializeCloudInputSettings(parsedValues, &s.CloudInputSettings); err != nil {
		return err
	}

	res, err := c.execute(ctx, s)
	if err != nil {
		return err
	}

	row := types.NewRow(
		types.MRP("input", res.Input),
		types.MRP("input_source", res.InputSource),
		types.MRP("output", res.Output),
		types.MRP("schema", res.Schema),
		types.MRP("type", res.Type),
		types.MRP("pages", res.Pages),
	)
	return gp.AddRow(ctx, row)
}

func NewRenderV6CobraCommand() (*cobra.Command, error) {
	cmd, err := NewRenderV6Command()
	if err != nil {
		return nil, err
	}

	cobraCmd, err := cli.BuildCobraCommand(cmd,
		cli.WithDualMode(true),
		cli.WithGlazeToggleFlag("with-glaze-output"),
		cli.WithParserConfig(cli.CobraParserConfig{
			ShortHelpSections: []string{schema.DefaultSlug},
			MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
		}),
	)
	if err != nil {
		return nil, err
	}

	return cobraCmd, nil
}
