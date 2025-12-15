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
	"github.com/go-go-golems/glazed/pkg/settings"
	pkg_rmdoc "github.com/go-go-golems/remarquee/pkg/rmdoc"
	rmdocrender "github.com/go-go-golems/remarquee/pkg/rmdoc/render"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type BuildBackgroundCommand struct {
	*glazecmds.CommandDescription
}

type BuildBackgroundSettings struct {
	File  string `glazed.parameter:"file"`
	Out   string `glazed.parameter:"out"`
	Force bool   `glazed.parameter:"force"`
}

var _ glazecmds.BareCommand = &BuildBackgroundCommand{}

func NewBuildBackgroundCommand() (*BuildBackgroundCommand, error) {
	glazedLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}
	commandSettingsLayer, err := cli.NewCommandSettingsLayer()
	if err != nil {
		return nil, err
	}

	cmdDesc := glazecmds.NewCommandDescription(
		"build-background",
		glazecmds.WithShort("Build a UI-ordered background PDF using PageRef.SourcePDFPage (debug utility)"),
		glazecmds.WithLong(`
Build a background PDF in UI page order based on the parsed page plan.

For PDF-backed docs it copies payload PDF pages and inserts blank pages for InsertedPage.
For notebooks (no payload) it currently creates blank pages using a default size.
`),
		glazecmds.WithFlags(
			parameters.NewParameterDefinition(
				"out",
				parameters.ParameterTypeString,
				parameters.WithDefault(""),
				parameters.WithHelp("Output PDF path (default: <input>-background.pdf in current dir)"),
			),
			parameters.NewParameterDefinition(
				"force",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithHelp("Overwrite output file if it exists"),
			),
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

	return &BuildBackgroundCommand{CommandDescription: cmdDesc}, nil
}

func (c *BuildBackgroundCommand) Run(ctx context.Context, parsedLayers *layers.ParsedLayers) error {
	_ = ctx

	s := &BuildBackgroundSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, s); err != nil {
		return err
	}

	doc, err := pkg_rmdoc.OpenFile(context.Background(), s.File)
	if err != nil {
		return err
	}
	if doc.Type == pkg_rmdoc.DocTypeEPUB {
		return errors.New("build-background: epub not supported")
	}

	out := s.Out
	if out == "" {
		base := filepath.Base(s.File)
		ext := filepath.Ext(base)
		base = base[:len(base)-len(ext)]
		out = base + "-background.pdf"
	}

	if !s.Force {
		if _, err := os.Stat(out); err == nil {
			return errors.Errorf("output file exists: %s (use --force to overwrite)", out)
		}
	}

	bg, err := rmdocrender.BuildBackgroundPDF(context.Background(), doc, rmdocrender.BackgroundOptions{})
	if err != nil {
		return err
	}

	if err := os.WriteFile(out, bg, 0o644); err != nil {
		return errors.Wrap(err, "write output pdf")
	}

	fmt.Printf("ok: wrote %s\n", out)
	return nil
}

func NewBuildBackgroundCobraCommand() (*cobra.Command, error) {
	cmd, err := NewBuildBackgroundCommand()
	if err != nil {
		return nil, err
	}

	return cli.BuildCobraCommand(cmd,
		cli.WithParserConfig(cli.CobraParserConfig{
			ShortHelpLayers: []string{layers.DefaultSlug},
			MiddlewaresFunc: cli.CobraCommandDefaultMiddlewares,
		}),
	)
}
