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
	pkg_rmdoc "github.com/go-go-golems/remarquee/pkg/rmdoc"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type InspectCommand struct {
	*glazecmds.CommandDescription
}

type InspectSettings struct {
	File string `glazed.parameter:"file"`
}

var _ glazecmds.BareCommand = &InspectCommand{}
var _ glazecmds.GlazeCommand = &InspectCommand{}

func NewInspectCommand() (*InspectCommand, error) {
	glazedLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}
	commandSettingsLayer, err := cli.NewCommandSettingsLayer()
	if err != nil {
		return nil, err
	}

	cmdDesc := glazecmds.NewCommandDescription(
		"inspect",
		glazecmds.WithShort("Inspect a local .rmdoc and print detected schema + page plan"),
		glazecmds.WithLong(`
Inspect a local .rmdoc file and print a deterministic page plan derived from .content.

Examples:
  remarquee rmdoc inspect file.rmdoc
  remarquee rmdoc inspect file.rmdoc --with-glaze-output --output json
`),
		glazecmds.WithFlags(
			parameters.NewParameterDefinition(
				"file",
				parameters.ParameterTypeString,
				parameters.WithIsArgument(true),
				parameters.WithRequired(true),
				parameters.WithHelp("Path to the .rmdoc file"),
			),
		),
		glazecmds.WithLayersList(glazedLayer, commandSettingsLayer),
	)

	return &InspectCommand{CommandDescription: cmdDesc}, nil
}

func (c *InspectCommand) Run(ctx context.Context, parsedLayers *layers.ParsedLayers) error {
	_ = ctx

	s := &InspectSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, s); err != nil {
		return err
	}

	doc, err := pkg_rmdoc.OpenFile(context.Background(), s.File)
	if err != nil {
		return err
	}

	fmt.Printf("uuid=%s schema=%s type=%s pages=%d\n", doc.UUID, schemaString(doc.Schema), docTypeString(doc.Type), len(doc.Pages))
	fmt.Printf("idx\tpage_id\tsrc_pdf\ttemplate\n")
	for _, p := range doc.Pages {
		fmt.Printf("%d\t%s\t%d\t%s\n", p.Index, p.PageID, p.SourcePDFPage, p.Template)
	}
	return nil
}

func (c *InspectCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
	gp middlewares.Processor,
) error {
	s := &InspectSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, s); err != nil {
		return err
	}

	doc, err := pkg_rmdoc.OpenFile(ctx, s.File)
	if err != nil {
		return err
	}

	for _, p := range doc.Pages {
		row := types.NewRow(
			types.MRP("uuid", doc.UUID),
			types.MRP("schema", schemaString(doc.Schema)),
			types.MRP("type", docTypeString(doc.Type)),
			types.MRP("idx", p.Index),
			types.MRP("page_id", p.PageID),
			types.MRP("src_pdf", p.SourcePDFPage),
			types.MRP("template", p.Template),
			types.MRP("deleted", p.Deleted),
		)
		if err := gp.AddRow(ctx, row); err != nil {
			return err
		}
	}

	if len(doc.Pages) == 0 {
		return errors.New("document has no pages")
	}

	return nil
}

func NewInspectCobraCommand() (*cobra.Command, error) {
	cmd, err := NewInspectCommand()
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
