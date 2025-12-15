package rmdoc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

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

	cmdDesc := glazecmds.NewCommandDescription(
		"render-legacy",
		glazecmds.WithShort("Render a legacy (V3/V5) .rmdoc/.zip to an annotated PDF (rmapi-backed)"),
		glazecmds.WithLong(`
Render a legacy (V3/V5) .rmdoc/.zip to an annotated PDF by delegating to rmapi's annotations PdfGenerator.

Notes:
- This command only supports legacy archives (legacy .content + V3/V5 .rm).
- For cPages/V6 documents, this will return an error (V6 rendering not implemented yet).
`),
		glazecmds.WithFlags(
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
				parameters.WithHelp("Path to the legacy .rmdoc or .zip file"),
			),
		),
		glazecmds.WithLayersList(glazedLayer, commandSettingsLayer),
	)

	return &RenderLegacyCommand{CommandDescription: cmdDesc}, nil
}

func (c *RenderLegacyCommand) Run(ctx context.Context, parsedLayers *layers.ParsedLayers) error {
	s := &RenderLegacySettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, s); err != nil {
		return err
	}

	doc, err := pkg_rmdoc.OpenFile(ctx, s.File)
	if err != nil {
		return err
	}
	if doc.Schema != pkg_rmdoc.SchemaLegacy {
		return errors.Errorf("render-legacy only supports legacy archives; detected schema=%s", schemaString(doc.Schema))
	}
	if doc.Type == pkg_rmdoc.DocTypeEPUB {
		return errors.New("render-legacy: epub not supported")
	}

	out := s.Out
	if out == "" {
		base := filepath.Base(s.File)
		ext := filepath.Ext(base)
		base = base[:len(base)-len(ext)]
		out = base + "-annotations.pdf"
	}

	if !s.Force {
		if _, err := os.Stat(out); err == nil {
			return errors.Errorf("output file exists: %s (use --force to overwrite)", out)
		}
	}

	opts := rmapi_annotations.PdfGeneratorOptions{
		AddPageNumbers:  s.AddPageNumbers,
		AllPages:        s.AllPages,
		AnnotationsOnly: s.AnnotationsOnly,
	}

	g := rmapi_annotations.CreatePdfGenerator(s.File, out, opts)
	if err := g.Generate(); err != nil {
		return err
	}

	fmt.Printf("ok: wrote %s\n", out)
	return nil
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

	doc, err := pkg_rmdoc.OpenFile(ctx, s.File)
	if err != nil {
		return err
	}
	if doc.Schema != pkg_rmdoc.SchemaLegacy {
		return errors.Errorf("render-legacy only supports legacy archives; detected schema=%s", schemaString(doc.Schema))
	}
	if doc.Type == pkg_rmdoc.DocTypeEPUB {
		return errors.New("render-legacy: epub not supported")
	}

	out := s.Out
	if out == "" {
		base := filepath.Base(s.File)
		ext := filepath.Ext(base)
		base = base[:len(base)-len(ext)]
		out = base + "-annotations.pdf"
	}

	if !s.Force {
		if _, err := os.Stat(out); err == nil {
			return errors.Errorf("output file exists: %s (use --force to overwrite)", out)
		}
	}

	opts := rmapi_annotations.PdfGeneratorOptions{
		AddPageNumbers:  s.AddPageNumbers,
		AllPages:        s.AllPages,
		AnnotationsOnly: s.AnnotationsOnly,
	}

	g := rmapi_annotations.CreatePdfGenerator(s.File, out, opts)
	if err := g.Generate(); err != nil {
		return err
	}

	row := types.NewRow(
		types.MRP("input", s.File),
		types.MRP("output", out),
		types.MRP("schema", schemaString(doc.Schema)),
		types.MRP("type", docTypeString(doc.Type)),
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
