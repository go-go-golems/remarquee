package cloud

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-go-golems/glazed/pkg/cli"
	glazecmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/juruen/rmapi/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type LsCommand struct {
	*glazecmds.CommandDescription
}

type LsSettings struct {
	AuthSettings

	// Arguments
	Path string `glazed.parameter:"path"`

	// Flags (mirrors rmapi ls options)
	Long          bool `glazed.parameter:"long"`
	Compact       bool `glazed.parameter:"compact"`
	Reverse       bool `glazed.parameter:"reverse"`
	DirFirst      bool `glazed.parameter:"group-directories"`
	ByTime        bool `glazed.parameter:"time"`
	ShowTemplates bool `glazed.parameter:"show-templates"`
}

var _ glazecmds.BareCommand = &LsCommand{}
var _ glazecmds.GlazeCommand = &LsCommand{}

func NewLsCommand() (*LsCommand, error) {
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
		"ls",
		glazecmds.WithShort("List remote directory entries"),
		glazecmds.WithLong(`
Lists files and folders in the reMarkable cloud (rmapi-backed).

Use --with-glaze-output for structured output (JSON, YAML, CSV, table).

Examples:
  remarquee cloud ls
  remarquee cloud ls /
  remarquee cloud ls /Books
  remarquee cloud ls /Books --long --time
  remarquee cloud ls /Books --with-glaze-output --output json
  remarquee cloud ls /Books --with-glaze-output --output yaml
  remarquee cloud ls /Books --with-glaze-output --fields name,type,modified_time
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

			// ls flags
			parameters.NewParameterDefinition(
				"compact",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithHelp("Compact output (just names, / suffix for directories)"),
				parameters.WithShortFlag("c"),
			),
			parameters.NewParameterDefinition(
				"long",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithHelp("Long output (includes modified time)"),
				parameters.WithShortFlag("l"),
			),
			parameters.NewParameterDefinition(
				"reverse",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithHelp("Reverse sort order"),
				parameters.WithShortFlag("r"),
			),
			parameters.NewParameterDefinition(
				"group-directories",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithHelp("Group directories first"),
				parameters.WithShortFlag("d"),
			),
			parameters.NewParameterDefinition(
				"time",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithHelp("Sort by modified time"),
				parameters.WithShortFlag("t"),
			),
			parameters.NewParameterDefinition(
				"show-templates",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithHelp("Include template files (rmapi hides these by default)"),
				parameters.WithShortFlag("s"),
			),

			// Argument: path
			parameters.NewParameterDefinition(
				"path",
				parameters.ParameterTypeString,
				parameters.WithIsArgument(true),
				parameters.WithDefault("/"),
				parameters.WithHelp("Remote path (folder or pattern). Defaults to '/'"),
			),
		),
		glazecmds.WithLayersList(glazedLayer, commandSettingsLayer),
	)

	return &LsCommand{CommandDescription: cmdDesc}, nil
}

func (c *LsCommand) Run(ctx context.Context, parsedLayers *layers.ParsedLayers) error {
	s := &LsSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, s); err != nil {
		return err
	}

	_, apiCtx, err := createApiCtx(s.AuthSettings)
	if err != nil {
		return err
	}

	nodes, err := apiCtx.Filetree().NodesByPath(s.Path, nil, true)
	if err != nil {
		return err
	}

	nodes = sortNodes(filterNodes(nodes, s), s)

	for _, n := range nodes {
		if err := displayNodeHuman(n, s); err != nil {
			return err
		}
	}

	return nil
}

func (c *LsCommand) RunIntoGlazeProcessor(ctx context.Context, parsedLayers *layers.ParsedLayers, gp middlewares.Processor) error {
	s := &LsSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, s); err != nil {
		return err
	}

	_, apiCtx, err := createApiCtx(s.AuthSettings)
	if err != nil {
		return err
	}

	nodes, err := apiCtx.Filetree().NodesByPath(s.Path, nil, true)
	if err != nil {
		return err
	}

	nodes = sortNodes(filterNodes(nodes, s), s)

	for _, n := range nodes {
		p := buildPathFromParents(n)
		modTime, _ := n.LastModified()

		row := types.NewRow(
			types.MRP("id", n.Id()),
			types.MRP("name", n.Name()),
			types.MRP("type", n.Document.Type),
			types.MRP("is_dir", n.IsDirectory()),
			types.MRP("path", p),
			types.MRP("parent_id", n.Document.Parent),
			types.MRP("version", n.Version()),
			types.MRP("modified_client", n.Document.ModifiedClient),
			types.MRP("modified_time", modTime),
		)

		if err := gp.AddRow(ctx, row); err != nil {
			return err
		}
	}

	return nil
}

func NewLsCobraCommand() (*cobra.Command, error) {
	cmd, err := NewLsCommand()
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

func filterNodes(in []*model.Node, options *LsSettings) []*model.Node {
	if options.ShowTemplates {
		return in
	}

	filtered := make([]*model.Node, 0, len(in))
	for _, node := range in {
		if node.Document.Type != model.TemplateType {
			filtered = append(filtered, node)
		}
	}
	return filtered
}

func sortNodes(in []*model.Node, options *LsSettings) []*model.Node {
	sort.SliceStable(in, func(i, j int) bool {
		if options.DirFirst {
			if in[i].IsDirectory() && in[j].IsFile() {
				return true
			}
			if in[j].IsDirectory() && in[i].IsFile() {
				return false
			}
		}

		left := strings.ToLower(in[i].Name())
		right := strings.ToLower(in[j].Name())
		if options.ByTime {
			left = in[i].Document.ModifiedClient
			right = in[j].Document.ModifiedClient
		}

		if options.Reverse {
			return left > right
		}
		return left < right
	})

	return in
}

func displayNodeHuman(e *model.Node, d *LsSettings) error {
	if !d.Compact {
		eType := "d"
		if e.IsFile() {
			eType = "f"
		}
		fmt.Printf("[%s]\t%s\n", eType, e.Name())
		return nil
	}

	isFolder := ""
	if e.IsDirectory() {
		isFolder = "/"
	}
	if d.Long {
		t, err := e.LastModified()
		if err != nil {
			return errors.Wrap(err, "failed to parse modified time")
		}
		fmt.Printf("%s %s%s\n", t.Local().Format(time.RFC3339), e.Name(), isFolder)
		return nil
	}

	fmt.Printf("%s%s\n", e.Name(), isFolder)
	return nil
}

func buildPathFromParents(n *model.Node) string {
	if n == nil {
		return ""
	}
	if n.IsRoot() {
		return "/"
	}

	parts := []string{n.Name()}
	cur := n.Parent
	for cur != nil && !cur.IsRoot() {
		parts = append(parts, cur.Name())
		cur = cur.Parent
	}

	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}

	return "/" + strings.Join(parts, "/")
}
