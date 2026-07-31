package cloud

import (
	"context"
	"fmt"

	"github.com/go-go-golems/glazed/pkg/cli"
	glazecmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/remarquee/cmd/remarquee/internal/appconfig"
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
	glazedLayer, err := settings.NewStructuredOutputSection()
	if err != nil {
		return nil, err
	}
	commandSettingsLayer, err := cli.NewCommandSettingsSection()
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

	return &VersionCommand{CommandDescription: cmdDesc}, nil
}

func (c *VersionCommand) Run(ctx context.Context, parsedValues *values.Values) error {
	fmt.Printf("rmapi_version=%s\n", version.Version)
	return nil
}

func NewVersionCobraCommand() (*cobra.Command, error) {
	cmd, err := NewVersionCommand()
	if err != nil {
		return nil, err
	}

	return cli.BuildCobraCommand(cmd,
		cli.WithParserConfig(appconfig.DefaultParserConfig()),
	)
}
