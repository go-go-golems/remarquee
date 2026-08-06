package cloud

import (
	"context"
	"fmt"

	"github.com/go-go-golems/glazed/pkg/cli"
	glazecmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/remarquee/cmd/remarquee/internal/appconfig"
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
	glazedLayer, err := settings.NewStructuredOutputSection()
	if err != nil {
		return nil, err
	}
	commandSettingsLayer, err := cli.NewCommandSettingsSection()
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

	return &AccountCommand{CommandDescription: cmdDesc}, nil
}

func (c *AccountCommand) Run(ctx context.Context, parsedValues *values.Values) error {
	s := &AccountSettings{}
	if err := parsedValues.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
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
		cli.WithParserConfig(appconfig.DefaultParserConfig()),
	)
}
