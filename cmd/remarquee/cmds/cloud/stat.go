package cloud

import (
	"context"
	"fmt"
	"time"

	"github.com/go-go-golems/glazed/pkg/cli"
	glazecmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type StatCommand struct {
	*glazecmds.CommandDescription
}

type StatSettings struct {
	AuthSettings

	Path string `glazed.parameter:"path"`
}

var _ glazecmds.BareCommand = &StatCommand{}
var _ glazecmds.GlazeCommand = &StatCommand{}

func NewStatCommand() (*StatCommand, error) {
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
		"stat",
		glazecmds.WithShort("Show metadata for a remote entry"),
		glazecmds.WithLong(`
Shows metadata for a remote file or folder in the reMarkable cloud (rmapi-backed).

Use --with-glaze-output for structured output (JSON, YAML, table).

Examples:
  remarquee cloud stat /
  remarquee cloud stat /Books
  remarquee cloud stat /Books/paper.pdf
  remarquee cloud stat /Books/paper.pdf --with-glaze-output --output json
  remarquee cloud stat /Books/paper.pdf --with-glaze-output --output yaml
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

			// Argument: path
			parameters.NewParameterDefinition(
				"path",
				parameters.ParameterTypeString,
				parameters.WithIsArgument(true),
				parameters.WithRequired(true),
				parameters.WithHelp("Remote path to inspect"),
			),
		),
		glazecmds.WithLayersList(glazedLayer, commandSettingsLayer),
	)

	return &StatCommand{CommandDescription: cmdDesc}, nil
}

func (c *StatCommand) Run(ctx context.Context, parsedLayers *layers.ParsedLayers) error {
	s := &StatSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, s); err != nil {
		return err
	}

	_, apiCtx, err := createApiCtx(s.AuthSettings)
	if err != nil {
		return err
	}

	node, err := apiCtx.Filetree().NodeByPath(s.Path, nil)
	if err != nil {
		return err
	}

	path := buildPathFromParents(node)
	modLocal := ""
	if node.Document.ModifiedClient != "" {
		mod, err := node.LastModified()
		if err != nil {
			return errors.Wrap(err, "failed to parse modified time")
		}
		modLocal = mod.Local().Format(time.RFC3339)
	}

	fmt.Printf("path=%s\n", path)
	fmt.Printf("name=%s\n", node.Name())
	fmt.Printf("id=%s\n", node.Id())
	fmt.Printf("type=%s\n", node.Document.Type)
	fmt.Printf("is_dir=%v\n", node.IsDirectory())
	fmt.Printf("version=%d\n", node.Version())
	fmt.Printf("parent_id=%s\n", node.Document.Parent)
	fmt.Printf("modified_client=%s\n", node.Document.ModifiedClient)
	fmt.Printf("modified_local=%s\n", modLocal)

	return nil
}

func (c *StatCommand) RunIntoGlazeProcessor(ctx context.Context, parsedLayers *layers.ParsedLayers, gp middlewares.Processor) error {
	s := &StatSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, s); err != nil {
		return err
	}

	_, apiCtx, err := createApiCtx(s.AuthSettings)
	if err != nil {
		return err
	}

	node, err := apiCtx.Filetree().NodeByPath(s.Path, nil)
	if err != nil {
		return err
	}

	path := buildPathFromParents(node)
	var mod interface{} = nil
	if node.Document.ModifiedClient != "" {
		mt, err := node.LastModified()
		if err != nil {
			return errors.Wrap(err, "failed to parse modified time")
		}
		mod = mt
	}

	row := types.NewRow(
		types.MRP("path", path),
		types.MRP("name", node.Name()),
		types.MRP("id", node.Id()),
		types.MRP("type", node.Document.Type),
		types.MRP("is_dir", node.IsDirectory()),
		types.MRP("version", node.Version()),
		types.MRP("parent_id", node.Document.Parent),
		types.MRP("modified_client", node.Document.ModifiedClient),
		types.MRP("modified_time", mod),
	)

	return gp.AddRow(ctx, row)
}

func NewStatCobraCommand() (*cobra.Command, error) {
	cmd, err := NewStatCommand()
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
