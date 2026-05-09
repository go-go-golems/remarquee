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
	"github.com/go-go-golems/remarquee/pkg/rmcloud"
	"github.com/spf13/cobra"
)

type GetCommand struct {
	*glazecmds.CommandDescription
}

type GetSettings struct {
	AuthSettings

	Remote string `glazed:"remote"`
	OutDir string `glazed:"out-dir"`
}

var _ glazecmds.BareCommand = &GetCommand{}

func NewGetCommand() (*GetCommand, error) {
	glazedLayer, err := settings.NewGlazedSection()
	if err != nil {
		return nil, err
	}
	commandSettingsLayer, err := cli.NewCommandSettingsSection()
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

			fields.New(
				"out-dir",
				fields.TypeString,
				fields.WithDefault("."),
				fields.WithHelp("Output directory for the downloaded .rmdoc"),
			),
			fields.New(
				"remote",
				fields.TypeString,
				fields.WithIsArgument(true),
				fields.WithRequired(true),
				fields.WithHelp("Remote path to download (must be a file)"),
			),
		),
		glazecmds.WithSections(glazedLayer, commandSettingsLayer),
	)

	return &GetCommand{CommandDescription: cmdDesc}, nil
}

func (c *GetCommand) Run(ctx context.Context, parsedValues *values.Values) error {
	s := &GetSettings{}
	if err := parsedValues.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
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
		cli.WithParserConfig(appconfig.DefaultParserConfig()),
	)
}
