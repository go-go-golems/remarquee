package cloud

import (
	"context"
	"fmt"

	"github.com/go-go-golems/glazed/pkg/cli"
	glazecmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/remarquee/cmd/remarquee/internal/appconfig"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type RefreshCommand struct {
	*glazecmds.CommandDescription
}

type RefreshSettings struct {
	AuthSettings
}

var _ glazecmds.BareCommand = &RefreshCommand{}
var _ glazecmds.GlazeCommand = &RefreshCommand{}

func NewRefreshCommand() (*RefreshCommand, error) {
	glazedLayer, err := settings.NewStructuredOutputSection(
		// Default to JSON output in glaze mode for machine-readable structured output
		schema.WithDefaults(map[string]interface{}{
			"format": "json",
		}),
	)
	if err != nil {
		return nil, err
	}
	commandSettingsLayer, err := cli.NewCommandSettingsSection()
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

Use --with-glaze-output for structured output (JSON, YAML, table).

Examples:
  remarquee cloud refresh
  remarquee cloud refresh --non-interactive
  remarquee cloud refresh --reauth
  remarquee cloud refresh --with-glaze-output --format json
  remarquee cloud refresh --with-glaze-output --format yaml
`),
		glazecmds.WithFlags(
			fields.New(
				"non-interactive",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Do not prompt for one-time code; fail if tokens are missing"),
			),
			fields.New(
				"reauth",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Force re-authentication (re-fetch user token)"),
			),
		),
		glazecmds.WithSections(glazedLayer, commandSettingsLayer),
	)

	return &RefreshCommand{CommandDescription: cmdDesc}, nil
}

func (c *RefreshCommand) Run(ctx context.Context, parsedValues *values.Values) error {
	settings_ := &RefreshSettings{}
	if err := parsedValues.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return err
	}

	userInfo, apiCtx, err := createApiCtx(settings_.AuthSettings)
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
	parsedValues *values.Values,
	gp middlewares.Processor,
) error {
	settings_ := &RefreshSettings{}
	if err := parsedValues.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return err
	}

	userInfo, apiCtx, err := createApiCtx(settings_.AuthSettings)
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
		cli.WithParserConfig(appconfig.DefaultParserConfig()),
	)
	if err != nil {
		return nil, err
	}

	return cobraCmd, nil
}
