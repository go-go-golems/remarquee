package cloud

import (
	"context"
	"fmt"

	"github.com/go-go-golems/glazed/pkg/cli"
	glazecmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/spf13/cobra"
)

type AccountCommand struct {
	*glazecmds.CommandDescription
}

type AccountSettings struct {
	AuthSettings
}

var _ glazecmds.BareCommand = &AccountCommand{}

func NewAccountCommand() (*AccountCommand, error) {
	glazedLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}
	commandSettingsLayer, err := cli.NewCommandSettingsLayer()
	if err != nil {
		return nil, err
	}

	cmdDesc := glazecmds.NewCommandDescription(
		"account",
		glazecmds.WithShort("Show cloud account info"),
		glazecmds.WithLong(`
Shows basic account info as detected by rmapi token parsing.

Examples:
  remarquee cloud account

If auth fails:
  - retry with: remarquee cloud account --reauth
  - if it still fails, run: rmapi reset (then re-register the device with rmapi account)
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
		),
		glazecmds.WithLayersList(glazedLayer, commandSettingsLayer),
	)

	return &AccountCommand{CommandDescription: cmdDesc}, nil
}

func (c *AccountCommand) Run(ctx context.Context, parsedLayers *layers.ParsedLayers) error {
	s := &AccountSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, s); err != nil {
		return err
	}

	userInfo, _, err := createApiCtx(s.AuthSettings)
	if err != nil {
		return err
	}

	fmt.Printf("user=%s sync_version=%s\n", userInfo.User, userInfo.SyncVersion.String())
	return nil
}

func NewAccountCobraCommand() (*cobra.Command, error) {
	cmd, err := NewAccountCommand()
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
