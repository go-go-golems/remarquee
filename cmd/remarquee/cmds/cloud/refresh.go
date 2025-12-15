package cloud

import (
	"context"
	"fmt"

	"github.com/go-go-golems/glazed/pkg/cli"
	glazecmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/juruen/rmapi/api"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type RefreshCommand struct {
	*glazecmds.CommandDescription
}

type RefreshSettings struct {
	NonInteractive bool `glazed.parameter:"non-interactive"`
	Reauth         bool `glazed.parameter:"reauth"`
}

var _ glazecmds.BareCommand = &RefreshCommand{}
var _ glazecmds.GlazeCommand = &RefreshCommand{}

func NewRefreshCommand() (*RefreshCommand, error) {
	glazedLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}
	commandSettingsLayer, err := cli.NewCommandSettingsLayer()
	if err != nil {
		return nil, err
	}

	cmdDesc := glazecmds.NewCommandDescription(
		"refresh",
		glazecmds.WithShort("Refresh the remote document tree"),
		glazecmds.WithLong(`
Refreshes the rmapi-backed remote document tree (Sync15).

This is the first cloud command because it validates end-to-end connectivity:
- token discovery/auth (rmapi)
- API context creation (rmapi sync15)
- remote tree refresh

Examples:
  remarquee cloud refresh
  remarquee cloud refresh --non-interactive
  remarquee cloud refresh --reauth
  remarquee cloud refresh --with-glaze-output --output json
`),
		glazecmds.WithFlags(
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

	return &RefreshCommand{CommandDescription: cmdDesc}, nil
}

func (c *RefreshCommand) Run(ctx context.Context, parsedLayers *layers.ParsedLayers) error {
	settings_ := &RefreshSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, settings_); err != nil {
		return err
	}

	userInfo, apiCtx, err := createApiCtx(settings_.Reauth, settings_.NonInteractive)
	if err != nil {
		return err
	}

	hash, gen, err := apiCtx.Refresh()
	if err != nil {
		return errors.Wrap(err, "rmapi refresh failed")
	}

	fmt.Printf("user=%s sync=%s hash=%s generation=%d\n", userInfo.User, userInfo.SyncVersion.String(), hash, gen)
	return nil
}

func (c *RefreshCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
	gp middlewares.Processor,
) error {
	settings_ := &RefreshSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, settings_); err != nil {
		return err
	}

	userInfo, apiCtx, err := createApiCtx(settings_.Reauth, settings_.NonInteractive)
	if err != nil {
		return err
	}

	hash, gen, err := apiCtx.Refresh()
	if err != nil {
		return errors.Wrap(err, "rmapi refresh failed")
	}

	row := types.NewRow(
		types.MRP("user", userInfo.User),
		types.MRP("sync_version", userInfo.SyncVersion.String()),
		types.MRP("hash", hash),
		types.MRP("generation", gen),
	)

	return gp.AddRow(ctx, row)
}

func NewRefreshCobraCommand() (*cobra.Command, error) {
	cmd, err := NewRefreshCommand()
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

	// Attach the glazed help system under this command tree for discoverability.
	// This mirrors the common pattern used elsewhere in this monorepo.
	helpSystem := help.NewHelpSystem()
	help_cmd.SetupCobraRootCommand(helpSystem, cobraCmd)

	return cobraCmd, nil
}

func createApiCtx(reAuth bool, nonInteractive bool) (*api.UserInfo, api.ApiCtx, error) {
	// rmapi does retries in main.go; for now keep this minimal and bubble errors.
	httpCtx := api.AuthHttpCtx(reAuth, nonInteractive)
	userInfo, err := api.ParseToken(httpCtx.Tokens.UserToken)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to parse rmapi user token")
	}

	apiCtx, err := api.CreateApiCtx(httpCtx, userInfo.SyncVersion)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create rmapi api context")
	}

	return userInfo, apiCtx, nil
}
