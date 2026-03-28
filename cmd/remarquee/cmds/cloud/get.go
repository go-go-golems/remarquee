package cloud

import (
	"context"
	"fmt"

	"github.com/go-go-golems/glazed/pkg/cli"
	glazecmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/remarquee/pkg/rmcloud"
	"github.com/spf13/cobra"
)

type GetCommand struct {
	*glazecmds.CommandDescription
}

type GetSettings struct {
	AuthSettings

	Remote string `glazed.parameter:"remote"`
	OutDir string `glazed.parameter:"out-dir"`
}

var _ glazecmds.BareCommand = &GetCommand{}

func NewGetCommand() (*GetCommand, error) {
	glazedLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}
	commandSettingsLayer, err := cli.NewCommandSettingsLayer()
	if err != nil {
		return nil, err
	}

	cmdDesc := glazecmds.NewCommandDescription(
		"get",
		glazecmds.WithShort("Download a remote document as .rmdoc"),
		glazecmds.WithLong(`
Downloads a remote document as a .rmdoc archive (rmapi-backed).

Examples:
  remarquee cloud get /Books/MyDoc
  remarquee cloud get /Books/MyDoc --out-dir /tmp
`),
		glazecmds.WithFlags(
			// Auth flags
			parameters.NewParameterDefinition(
				"non-interactive",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithHelp("Do not prompt for one-time code; fail if tokens are missing"),
			),
			parameters.NewParameterDefinition(
				"reauth",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithHelp("Force re-authentication (re-fetch user token)"),
			),

			parameters.NewParameterDefinition(
				"out-dir",
				parameters.ParameterTypeString,
				parameters.WithDefault("."),
				parameters.WithHelp("Output directory for the downloaded .rmdoc"),
			),
			parameters.NewParameterDefinition(
				"remote",
				parameters.ParameterTypeString,
				parameters.WithIsArgument(true),
				parameters.WithRequired(true),
				parameters.WithHelp("Remote path to download (must be a file)"),
			),
		),
		glazecmds.WithLayersList(glazedLayer, commandSettingsLayer),
	)

	return &GetCommand{CommandDescription: cmdDesc}, nil
}

func (c *GetCommand) Run(ctx context.Context, parsedLayers *layers.ParsedLayers) error {
	s := &GetSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, s); err != nil {
		return err
	}

	downloaded, err := rmcloud.DownloadDocumentByPath(ctx, rmcloud.AuthSettings{
		NonInteractive: s.NonInteractive,
		Reauth:         s.Reauth,
	}, s.Remote, s.OutDir)
	if err != nil {
		return err
	}

	fmt.Printf("OK: downloaded %s -> %s\n", s.Remote, downloaded.LocalPath)
	return nil
}

func NewGetCobraCommand() (*cobra.Command, error) {
	cmd, err := NewGetCommand()
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
