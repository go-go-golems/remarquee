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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type RmCommand struct {
	*glazecmds.CommandDescription
}

type RmSettings struct {
	AuthSettings

	Recursive bool `glazed:"recursive"`
	Yes       bool `glazed:"yes"`

	Targets []string `glazed:"target"`
}

var _ glazecmds.BareCommand = &RmCommand{}

func NewRmCommand() (*RmCommand, error) {
	glazedLayer, err := settings.NewGlazedSection()
	if err != nil {
		return nil, err
	}
	commandSettingsLayer, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	cmdDesc := glazecmds.NewCommandDescription(
		"rm",
		glazecmds.WithShort("Delete remote entries (requires --yes)"),
		glazecmds.WithLong(`
Deletes files/folders in the reMarkable cloud (rmapi-backed).

Safety:
- This command refuses to delete unless you pass --yes.

Examples:
  remarquee cloud rm /Books/Doc --yes
  remarquee cloud rm /Books/Folder --recursive --yes
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
				"recursive",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Remove non-empty folders"),
				fields.WithShortFlag("r"),
			),
			fields.New(
				"yes",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Confirm deletion"),
			),

			fields.New(
				"target",
				fields.TypeStringList,
				fields.WithIsArgument(true),
				fields.WithRequired(true),
				fields.WithHelp("One or more remote paths (can include patterns)"),
			),
		),
		glazecmds.WithSections(glazedLayer, commandSettingsLayer),
	)

	return &RmCommand{CommandDescription: cmdDesc}, nil
}

func (c *RmCommand) Run(ctx context.Context, parsedValues *values.Values) error {
	s := &RmSettings{}
	if err := parsedValues.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
		return err
	}

	_, apiCtx, err := createApiCtx(s.AuthSettings)
	if err != nil {
		return err
	}

	// Resolve targets first so we can print what would be deleted if --yes is missing.
	toDelete := make([]string, 0)
	for _, target := range s.Targets {
		nodes, err := apiCtx.Filetree().NodesByPath(target, nil, false)
		if err != nil {
			return err
		}
		for _, node := range nodes {
			toDelete = append(toDelete, buildPathFromParents(node))
		}
	}

	if !s.Yes {
		if len(toDelete) == 0 {
			return errors.New("nothing to delete (and --yes not provided)")
		}
		for _, p := range toDelete {
			fmt.Printf("would delete: %s\n", p)
		}
		return errors.New("refusing to delete without --yes")
	}

	// Execute deletions.
	for _, target := range s.Targets {
		nodes, err := apiCtx.Filetree().NodesByPath(target, nil, false)
		if err != nil {
			return err
		}
		for _, node := range nodes {
			fmt.Printf("deleting: %s\n", buildPathFromParents(node))
			if err := apiCtx.DeleteEntry(node, s.Recursive, true); err != nil {
				return errors.Wrap(err, "failed to delete entry")
			}
			apiCtx.Filetree().DeleteNode(node)
		}
	}

	return nil
}

func NewRmCobraCommand() (*cobra.Command, error) {
	cmd, err := NewRmCommand()
	if err != nil {
		return nil, err
	}

	return cli.BuildCobraCommand(cmd,
		cli.WithParserConfig(appconfig.DefaultParserConfig()),
	)
}
