package rmdoc

import (
	"context"
	"fmt"

	"github.com/go-go-golems/glazed/pkg/cli"
	glazecmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	rmapi_annotations "github.com/juruen/rmapi/annotations"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	pkg_rmdoc "github.com/go-go-golems/remarquee/pkg/rmdoc"
)

type RenderLegacyCommand struct {
	*glazecmds.CommandDescription
}

type RenderLegacySettings struct {
	File string `glazed.parameter:"file"`
	Out  string `glazed.parameter:"out"`

	Force           bool `glazed.parameter:"force"`
	AddPageNumbers  bool `glazed.parameter:"add-page-numbers"`
	AllPages        bool `glazed.parameter:"all-pages"`
	AnnotationsOnly bool `glazed.parameter:"annotations-only"`

	CloudInputSettings
}

var _ glazecmds.BareCommand = &RenderLegacyCommand{}
var _ glazecmds.GlazeCommand = &RenderLegacyCommand{}

func NewRenderLegacyCommand() (*RenderLegacyCommand, error) {
	glazedLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}
	commandSettingsLayer, err := cli.NewCommandSettingsLayer()
	if err != nil {
		return nil, err
	}

	flags := []*parameters.ParameterDefinition{
		parameters.NewParameterDefinition(
			"out",
			parameters.ParameterTypeString,
			parameters.WithDefault(""),
			parameters.WithHelp("Output PDF path (default: <input>-annotations.pdf in current dir)"),
		),
		parameters.NewParameterDefinition(
			"force",
			parameters.ParameterTypeBool,
			parameters.WithDefault(false),
			parameters.WithHelp("Overwrite output file if it exists"),
		),
		parameters.NewParameterDefinition(
			"add-page-numbers",
			parameters.ParameterTypeBool,
			parameters.WithDefault(false),
			parameters.WithHelp("Add page numbers"),
		),
		parameters.NewParameterDefinition(
			"all-pages",
			parameters.ParameterTypeBool,
			parameters.WithDefault(false),
			parameters.WithHelp("Include pages without annotations"),
		),
		parameters.NewParameterDefinition(
			"annotations-only",
			parameters.ParameterTypeBool,
			parameters.WithDefault(false),
			parameters.WithHelp("Export annotations only (no background PDF)"),
		),
		parameters.NewParameterDefinition(
			"file",
			parameters.ParameterTypeString,
			parameters.WithIsArgument(true),
			parameters.WithRequired(true),
			parameters.WithHelp("Path to the legacy .rmdoc/.zip file, or a remote cloud path when used with --cloud"),
		),
	}
	flags = append(flags, cloudInputParameterDefinitions()...)

	cmdDesc := glazecmds.NewCommandDescription(
		"render-legacy",
		glazecmds.WithShort("Render a legacy (V3/V5) .rmdoc/.zip to an annotated PDF (rmapi-backed)"),
		glazecmds.WithLong(`
Render a legacy (V3/V5) .rmdoc/.zip to an annotated PDF by delegating to rmapi's annotations PdfGenerator.

Notes:
- This command only supports legacy archives (legacy .content + V3/V5 .rm).
- For cPages/V6 documents, this will return an error (V6 rendering not implemented yet).
`),
		glazecmds.WithFlags(flags...),
		glazecmds.WithLayersList(glazedLayer, commandSettingsLayer),
	)

	return &RenderLegacyCommand{CommandDescription: cmdDesc}, nil
}

func (c *RenderLegacyCommand) Run(ctx context.Context, parsedLayers *layers.ParsedLayers) error {
	s := &RenderLegacySettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, s); err != nil {
		return err
	}
	if err := initializeCloudInputSettings(parsedLayers, &s.CloudInputSettings); err != nil {
		return err
	}

	res, err := c.execute(ctx, s)
	if err != nil {
		return err
	}

	fmt.Printf("ok: wrote %s\n", res.Output)
	return nil
}

type renderLegacyExecution struct {
	Input       string
	InputSource string
	Output      string
	Schema      string
	Type        string
}

func (c *RenderLegacyCommand) execute(ctx context.Context, s *RenderLegacySettings) (*renderLegacyExecution, error) {
	input, err := ResolveRMDocInput(ctx, s.File, s.CloudInputSettings)
	if err != nil {
		return nil, err
	}
	defer func() { _ = input.Cleanup() }()

	doc, err := pkg_rmdoc.OpenFile(ctx, input.LocalPath)
	if err != nil {
		return nil, err
	}
	if doc.Schema != pkg_rmdoc.SchemaLegacy {
		return nil, errors.Errorf("render-legacy only supports legacy archives; detected schema=%s", schemaString(doc.Schema))
	}
	if doc.Type == pkg_rmdoc.DocTypeEPUB {
		return nil, errors.New("render-legacy: epub not supported")
	}

	out := s.Out
	if out == "" {
		out = defaultOutputPath(input.LocalPath, "-annotations.pdf")
	}
	if err := ensureOutputWritable(out, s.Force); err != nil {
		return nil, err
	}
	opts := rmapi_annotations.PdfGeneratorOptions{
		AddPageNumbers:  s.AddPageNumbers,
		AllPages:        s.AllPages,
		AnnotationsOnly: s.AnnotationsOnly,
	}

	g := rmapi_annotations.CreatePdfGenerator(input.LocalPath, out, opts)
	if err := g.Generate(); err != nil {
		return nil, err
	}

	return &renderLegacyExecution{
		Input:       input.RequestedPath,
		InputSource: input.Source,
		Output:      out,
		Schema:      schemaString(doc.Schema),
		Type:        docTypeString(doc.Type),
	}, nil
}

func (c *RenderLegacyCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
	gp middlewares.Processor,
) error {
	s := &RenderLegacySettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, s); err != nil {
		return err
	}
	if err := initializeCloudInputSettings(parsedLayers, &s.CloudInputSettings); err != nil {
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
	)
	return gp.AddRow(ctx, row)
}

func NewRenderLegacyCobraCommand() (*cobra.Command, error) {
	cmd, err := NewRenderLegacyCommand()
	if err != nil {
		return nil, err
	}

	cobraCmd, err := cli.BuildCobraCommand(cmd,
		cli.WithDualMode(true),
		cli.WithGlazeToggleFlag("with-glaze-output"),
		cli.WithParserConfig(cli.CobraParserConfig{
			ShortHelpLayers: []string{layers.DefaultSlug},
			MiddlewaresFunc: cli.CobraCommandDefaultMiddlewares,
		}),
	)
	if err != nil {
		return nil, err
	}

	return cobraCmd, nil
}
