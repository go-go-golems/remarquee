package cloud

import (
	"context"
	"fmt"

	"github.com/go-go-golems/glazed/pkg/cli"
	glazecmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type RmCommand struct {
	*glazecmds.CommandDescription
}

type RmSettings struct {
	AuthSettings

	Recursive bool `glazed.parameter:"recursive"`
	Yes       bool `glazed.parameter:"yes"`

	Targets []string `glazed.parameter:"target"`
}

var _ glazecmds.BareCommand = &RmCommand{}

func NewRmCommand() (*RmCommand, error) {
	glazedLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}
	commandSettingsLayer, err := cli.NewCommandSettingsLayer()
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
				"recursive",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithHelp("Remove non-empty folders"),
				parameters.WithShortFlag("r"),
			),
			parameters.NewParameterDefinition(
				"yes",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithHelp("Confirm deletion"),
			),

			parameters.NewParameterDefinition(
				"target",
				parameters.ParameterTypeStringList,
				parameters.WithIsArgument(true),
				parameters.WithRequired(true),
				parameters.WithHelp("One or more remote paths (can include patterns)"),
			),
		),
		glazecmds.WithLayersList(glazedLayer, commandSettingsLayer),
	)

	return &RmCommand{CommandDescription: cmdDesc}, nil
}

func (c *RmCommand) Run(ctx context.Context, parsedLayers *layers.ParsedLayers) error {
	s := &RmSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, s); err != nil {
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
		cli.WithParserConfig(cli.CobraParserConfig{
			ShortHelpLayers: []string{layers.DefaultSlug},
			MiddlewaresFunc: cli.CobraCommandDefaultMiddlewares,
		}),
	)
}
