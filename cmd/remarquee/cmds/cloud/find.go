package cloud

import (
	"context"
	"fmt"
	"regexp"

	"github.com/go-go-golems/glazed/pkg/cli"
	glazecmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/remarquee/cmd/remarquee/internal/appconfig"
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

	Compact bool `glazed:"compact"`

	Start   string `glazed:"start"`
	Pattern string `glazed:"pattern"`
}

var _ glazecmds.BareCommand = &FindCommand{}
var _ glazecmds.GlazeCommand = &FindCommand{}

func NewFindCommand() (*FindCommand, error) {
	glazedLayer, err := settings.NewGlazedSection(
		// Default to JSON output in glaze mode for machine-readable structured output
		settings.WithOutputSectionOptions(
			schema.WithDefaults(map[string]interface{}{
				"output": "json",
			}),
		),
	)
	if err != nil {
		return nil, err
	}
	commandSettingsLayer, err := cli.NewCommandSettingsSection()
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
				"compact",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Compact output (no [d]/[f] prefix; / suffix for directories)"),
				fields.WithShortFlag("c"),
			),

			fields.New(
				"start",
				fields.TypeString,
				fields.WithIsArgument(true),
				fields.WithDefault("/"),
				fields.WithHelp("Start directory (default: /)"),
			),
			fields.New(
				"pattern",
				fields.TypeString,
				fields.WithIsArgument(true),
				fields.WithDefault(""),
				fields.WithHelp("Optional regexp pattern"),
			),
		),
		glazecmds.WithSections(glazedLayer, commandSettingsLayer),
	)

	return &FindCommand{CommandDescription: cmdDesc}, nil
}

func (c *FindCommand) Run(ctx context.Context, parsedValues *values.Values) error {
	s := &FindSettings{}
	if err := parsedValues.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
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

func (c *FindCommand) RunIntoGlazeProcessor(ctx context.Context, parsedValues *values.Values, gp middlewares.Processor) error {
	s := &FindSettings{}
	if err := parsedValues.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
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

	var addRowErr error
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
					addRowErr = err
					return filetree.StopVisiting
				}
			}
			return filetree.ContinueVisiting
		},
	})

	if addRowErr != nil {
		return addRowErr
	}

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
		cli.WithParserConfig(appconfig.DefaultParserConfig()),
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
