package cloud

import (
	"context"
	"fmt"

	"github.com/go-go-golems/glazed/pkg/cli"
	glazecmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/juruen/rmapi/version"
	"github.com/spf13/cobra"
)

type VersionCommand struct {
	*glazecmds.CommandDescription
}

type VersionSettings struct {
	AuthSettings
}

var _ glazecmds.BareCommand = &VersionCommand{}

func NewVersionCommand() (*VersionCommand, error) {
	glazedLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}
	commandSettingsLayer, err := cli.NewCommandSettingsLayer()
	if err != nil {
		return nil, err
	}

	cmdDesc := glazecmds.NewCommandDescription(
		"version",
		glazecmds.WithShort("Show rmapi version"),
		glazecmds.WithLong(`
Shows the bundled rmapi library version.
`),
		glazecmds.WithFlags(
			// Auth flags (kept consistent with other cloud commands, though version doesn't require auth)
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
		),
		glazecmds.WithLayersList(glazedLayer, commandSettingsLayer),
	)

	return &VersionCommand{CommandDescription: cmdDesc}, nil
}

func (c *VersionCommand) Run(ctx context.Context, parsedLayers *layers.ParsedLayers) error {
	fmt.Printf("rmapi_version=%s\n", version.Version)
	return nil
}

func NewVersionCobraCommand() (*cobra.Command, error) {
	cmd, err := NewVersionCommand()
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
