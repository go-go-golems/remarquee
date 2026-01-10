package cloud

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cli"
	glazecmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/juruen/rmapi/filetree"
	"github.com/juruen/rmapi/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type SearchCommand struct {
	*glazecmds.CommandDescription
}

type SearchSettings struct {
	AuthSettings

	Query string `glazed.parameter:"query"`
	Start string `glazed.parameter:"start"`

	Regex         bool   `glazed.parameter:"regex"`
	CaseSensitive bool   `glazed.parameter:"case-sensitive"`
	MatchTarget   string `glazed.parameter:"match"`
	Compact       bool   `glazed.parameter:"compact"`
	IncludeTemps  bool   `glazed.parameter:"include-templates"`
	TypeFilter    string `glazed.parameter:"type"`
	Limit         int    `glazed.parameter:"limit"`
}

var _ glazecmds.BareCommand = &SearchCommand{}

func NewSearchCommand() (*SearchCommand, error) {
	glazedLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}
	commandSettingsLayer, err := cli.NewCommandSettingsLayer()
	if err != nil {
		return nil, err
	}

	cmdDesc := glazecmds.NewCommandDescription(
		"search",
		glazecmds.WithShort("Search entries by name or path (rmapi-backed)"),
		glazecmds.WithLong(`
Search entries by name or path (rmapi-backed).

Examples:
  remarquee cloud search electronics
  remarquee cloud search ".*pdf$" --regex
  remarquee cloud search notes --match name --start /Books
  remarquee cloud search sketch --type dir --limit 5
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
				"query",
				parameters.ParameterTypeString,
				parameters.WithIsArgument(true),
				parameters.WithRequired(true),
				parameters.WithHelp("Search query (substring by default, regex with --regex)"),
			),
			parameters.NewParameterDefinition(
				"start",
				parameters.ParameterTypeString,
				parameters.WithDefault("/"),
				parameters.WithHelp("Start directory (default: /)"),
			),
			parameters.NewParameterDefinition(
				"regex",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithHelp("Treat query as a regexp"),
			),
			parameters.NewParameterDefinition(
				"case-sensitive",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithHelp("Use case-sensitive matching (substring or regex)"),
			),
			parameters.NewParameterDefinition(
				"match",
				parameters.ParameterTypeString,
				parameters.WithDefault("path"),
				parameters.WithHelp("Match target: path or name"),
			),
			parameters.NewParameterDefinition(
				"compact",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithShortFlag("c"),
				parameters.WithHelp("Compact output (no [d]/[f] prefix; / suffix for directories)"),
			),
			parameters.NewParameterDefinition(
				"include-templates",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithHelp("Include template entries in results"),
			),
			parameters.NewParameterDefinition(
				"type",
				parameters.ParameterTypeString,
				parameters.WithDefault(""),
				parameters.WithHelp("Optional filter: dir, file, or template"),
			),
			parameters.NewParameterDefinition(
				"limit",
				parameters.ParameterTypeInteger,
				parameters.WithDefault(0),
				parameters.WithHelp("Stop after N matches (0 = no limit)"),
			),
		),
		glazecmds.WithLayersList(glazedLayer, commandSettingsLayer),
	)

	return &SearchCommand{CommandDescription: cmdDesc}, nil
}

func (c *SearchCommand) Run(ctx context.Context, parsedLayers *layers.ParsedLayers) error {
	s := &SearchSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, s); err != nil {
		return err
	}

	if s.Query == "" {
		return errors.New("query is required")
	}

	matchTarget := strings.ToLower(strings.TrimSpace(s.MatchTarget))
	switch matchTarget {
	case "path", "name":
	default:
		return errors.New("match must be 'path' or 'name'")
	}

	switch strings.ToLower(strings.TrimSpace(s.TypeFilter)) {
	case "", "dir", "file", "template":
	default:
		return errors.New("type must be one of: dir, file, template")
	}

	_, apiCtx, err := createApiCtx(s.AuthSettings)
	if err != nil {
		return err
	}

	startNode, err := apiCtx.Filetree().NodeByPath(s.Start, nil)
	if err != nil {
		return errors.New("start directory doesn't exist")
	}

	var matchFunc func(string) bool
	if s.Regex {
		pattern := s.Query
		if !s.CaseSensitive {
			pattern = "(?i)" + pattern
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return errors.New("failed to compile regexp")
		}
		matchFunc = re.MatchString
	} else {
		query := s.Query
		if !s.CaseSensitive {
			query = strings.ToLower(query)
		}
		matchFunc = func(candidate string) bool {
			if !s.CaseSensitive {
				candidate = strings.ToLower(candidate)
			}
			return strings.Contains(candidate, query)
		}
	}

	typeFilter := strings.ToLower(strings.TrimSpace(s.TypeFilter))
	matchesType := func(node *model.Node) bool {
		switch typeFilter {
		case "":
			return true
		case "dir":
			return node.IsDirectory()
		case "template":
			return node.Document.Type == model.TemplateType
		case "file":
			return node.IsFile() && node.Document.Type == model.DocumentType
		default:
			return true
		}
	}

	count := 0
	filetree.WalkTree(startNode, filetree.FileTreeVistor{
		Visit: func(node *model.Node, _ []string) bool {
			if !s.IncludeTemps && node.Document.Type == model.TemplateType {
				return false
			}
			if !matchesType(node) {
				return false
			}

			var candidate string
			if matchTarget == "name" {
				candidate = node.Name()
			} else {
				candidate = buildPathFromParents(node)
			}

			if !matchFunc(candidate) {
				return false
			}

			fmt.Println(formatSearchEntry(s.Compact, node))
			count++
			if s.Limit > 0 && count >= s.Limit {
				return true
			}
			return false
		},
	})

	return nil
}

func NewSearchCobraCommand() (*cobra.Command, error) {
	cmd, err := NewSearchCommand()
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

func formatSearchEntry(compact bool, node *model.Node) string {
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
