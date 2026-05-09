package cloud

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cli"
	glazecmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/remarquee/cmd/remarquee/internal/appconfig"
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

	Query string `glazed:"query"`
	Start string `glazed:"start"`

	Regex         bool   `glazed:"regex"`
	CaseSensitive bool   `glazed:"case-sensitive"`
	MatchTarget   string `glazed:"match"`
	Compact       bool   `glazed:"compact"`
	IncludeTemps  bool   `glazed:"include-templates"`
	TypeFilter    string `glazed:"type"`
	Limit         int    `glazed:"limit"`
}

var _ glazecmds.BareCommand = &SearchCommand{}

func NewSearchCommand() (*SearchCommand, error) {
	glazedLayer, err := settings.NewGlazedSection()
	if err != nil {
		return nil, err
	}
	commandSettingsLayer, err := cli.NewCommandSettingsSection()
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
				"query",
				fields.TypeString,
				fields.WithIsArgument(true),
				fields.WithRequired(true),
				fields.WithHelp("Search query (substring by default, regex with --regex)"),
			),
			fields.New(
				"start",
				fields.TypeString,
				fields.WithDefault("/"),
				fields.WithHelp("Start directory (default: /)"),
			),
			fields.New(
				"regex",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Treat query as a regexp"),
			),
			fields.New(
				"case-sensitive",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Use case-sensitive matching (substring or regex)"),
			),
			fields.New(
				"match",
				fields.TypeString,
				fields.WithDefault("path"),
				fields.WithHelp("Match target: path or name"),
			),
			fields.New(
				"compact",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithShortFlag("c"),
				fields.WithHelp("Compact output (no [d]/[f] prefix; / suffix for directories)"),
			),
			fields.New(
				"include-templates",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Include template entries in results"),
			),
			fields.New(
				"type",
				fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("Optional filter: dir, file, or template"),
			),
			fields.New(
				"limit",
				fields.TypeInteger,
				fields.WithDefault(0),
				fields.WithHelp("Stop after N matches (0 = no limit)"),
			),
		),
		glazecmds.WithSections(glazedLayer, commandSettingsLayer),
	)

	return &SearchCommand{CommandDescription: cmdDesc}, nil
}

func (c *SearchCommand) Run(ctx context.Context, parsedValues *values.Values) error {
	s := &SearchSettings{}
	if err := parsedValues.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
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
		cli.WithParserConfig(appconfig.DefaultParserConfig()),
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
