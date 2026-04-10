package cloud

import (
	"context"
	"fmt"
	"regexp"

	"github.com/go-go-golems/glazed/pkg/cli"
	glazecmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/juruen/rmapi/filetree"
	"github.com/juruen/rmapi/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type FindCommand struct {
	*glazecmds.CommandDescription
}

type FindSettings struct {
	AuthSettings

	Compact bool `glazed.parameter:"compact"`

	Start   string `glazed.parameter:"start"`
	Pattern string `glazed.parameter:"pattern"`
}

var _ glazecmds.BareCommand = &FindCommand{}
var _ glazecmds.GlazeCommand = &FindCommand{}

func NewFindCommand() (*FindCommand, error) {
	glazedLayer, err := settings.NewGlazedParameterLayers(
		// Default to JSON output in glaze mode for machine-readable structured output
		settings.WithOutputParameterLayerOptions(
			layers.WithDefaults(map[string]interface{}{
				"output": "json",
			}),
		),
	)
	if err != nil {
		return nil, err
	}
	commandSettingsLayer, err := cli.NewCommandSettingsLayer()
	if err != nil {
		return nil, err
	}

	cmdDesc := glazecmds.NewCommandDescription(
		"find",
		glazecmds.WithShort("Find entries recursively (optionally by regexp)"),
		glazecmds.WithLong(`
Find entries recursively (rmapi-backed).

Use --with-glaze-output for structured output (JSON, YAML, CSV, table).

Arguments:
  [start]   Start directory (default: /)
  [pattern] Optional regexp to match against the formatted output path

Examples:
  remarquee cloud find
  remarquee cloud find /Books
  remarquee cloud find /Books ".*pdf$"
  remarquee cloud find /Books "Selfish"
  remarquee cloud find /Books --with-glaze-output --output json
  remarquee cloud find /Books --with-glaze-output --output yaml
  remarquee cloud find /Books --with-glaze-output --fields name,type,path
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
				"compact",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithHelp("Compact output (no [d]/[f] prefix; / suffix for directories)"),
				parameters.WithShortFlag("c"),
			),

			parameters.NewParameterDefinition(
				"start",
				parameters.ParameterTypeString,
				parameters.WithIsArgument(true),
				parameters.WithDefault("/"),
				parameters.WithHelp("Start directory (default: /)"),
			),
			parameters.NewParameterDefinition(
				"pattern",
				parameters.ParameterTypeString,
				parameters.WithIsArgument(true),
				parameters.WithDefault(""),
				parameters.WithHelp("Optional regexp pattern"),
			),
		),
		glazecmds.WithLayersList(glazedLayer, commandSettingsLayer),
	)

	return &FindCommand{CommandDescription: cmdDesc}, nil
}

func (c *FindCommand) Run(ctx context.Context, parsedLayers *layers.ParsedLayers) error {
	s := &FindSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, s); err != nil {
		return err
	}

	_, apiCtx, err := createApiCtx(s.AuthSettings)
	if err != nil {
		return err
	}

	startNode, err := apiCtx.Filetree().NodeByPath(s.Start, nil)
	if err != nil {
		return errors.New("start directory doesn't exist")
	}

	var matchRegexp *regexp.Regexp
	if s.Pattern != "" {
		matchRegexp, err = regexp.Compile(s.Pattern)
		if err != nil {
			return errors.New("failed to compile regexp")
		}
	}

	filetree.WalkTree(startNode, filetree.FileTreeVistor{
		Visit: func(node *model.Node, _ []string) bool {
			entry := formatFindEntry(s.Compact, node)
			if matchRegexp == nil || matchRegexp.MatchString(entry) {
				fmt.Println(entry)
			}
			return false
		},
	})

	return nil
}

func (c *FindCommand) RunIntoGlazeProcessor(ctx context.Context, parsedLayers *layers.ParsedLayers, gp middlewares.Processor) error {
	s := &FindSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, s); err != nil {
		return err
	}

	_, apiCtx, err := createApiCtx(s.AuthSettings)
	if err != nil {
		return err
	}

	startNode, err := apiCtx.Filetree().NodeByPath(s.Start, nil)
	if err != nil {
		return errors.New("start directory doesn't exist")
	}

	var matchRegexp *regexp.Regexp
	if s.Pattern != "" {
		matchRegexp, err = regexp.Compile(s.Pattern)
		if err != nil {
			return errors.New("failed to compile regexp")
		}
	}

	filetree.WalkTree(startNode, filetree.FileTreeVistor{
		Visit: func(node *model.Node, _ []string) bool {
			p := buildPathFromParents(node)
			modTime, _ := node.LastModified()

			row := types.NewRow(
				types.MRP("id", node.Id()),
				types.MRP("name", node.Name()),
				types.MRP("type", node.Document.Type),
				types.MRP("is_dir", node.IsDirectory()),
				types.MRP("path", p),
				types.MRP("parent_id", node.Document.Parent),
				types.MRP("version", node.Version()),
				types.MRP("modified_client", node.Document.ModifiedClient),
				types.MRP("modified_time", modTime),
			)

			if matchRegexp == nil || matchRegexp.MatchString(p) {
				if err := gp.AddRow(ctx, row); err != nil {
					// Log error but continue processing
				}
			}
			return false
		},
	})

	return nil
}

func NewFindCobraCommand() (*cobra.Command, error) {
	cmd, err := NewFindCommand()
	if err != nil {
		return nil, err
	}

	return cli.BuildCobraCommand(cmd,
		cli.WithDualMode(true),
		cli.WithGlazeToggleFlag("with-glaze-output"),
		cli.WithParserConfig(cli.CobraParserConfig{
			ShortHelpLayers: []string{layers.DefaultSlug},
			MiddlewaresFunc: cli.CobraCommandDefaultMiddlewares,
		}),
	)
}

func formatFindEntry(compact bool, node *model.Node) string {
	fullpath := buildPathFromParents(node)
	if compact {
		if node.IsDirectory() {
			return fullpath + "/"
		}
		return fullpath
	}

	if node.IsDirectory() {
		return "[d] " + fullpath
	}
	return "[f] " + fullpath
}
