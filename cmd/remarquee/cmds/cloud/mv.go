package cloud

import (
	"context"
	"path"

	"github.com/go-go-golems/glazed/pkg/cli"
	glazecmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/juruen/rmapi/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type MvCommand struct {
	*glazecmds.CommandDescription
}

type MvSettings struct {
	AuthSettings

	Src string `glazed:"src"`
	Dst string `glazed:"dst"`
}

var _ glazecmds.BareCommand = &MvCommand{}

func NewMvCommand() (*MvCommand, error) {
	glazedLayer, err := settings.NewGlazedSection()
	if err != nil {
		return nil, err
	}
	commandSettingsLayer, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	cmdDesc := glazecmds.NewCommandDescription(
		"mv",
		glazecmds.WithShort("Move or rename a remote entry"),
		glazecmds.WithLong(`
Moves or renames remote files/folders (rmapi-backed).

This mirrors rmapi's shell semantics:
- If <dst> resolves to an existing directory, move <src> into it (keeping the same name).
- Otherwise interpret <dst> as a full destination path (rename/move).

Examples:
  remarquee cloud mv /Books/OldName /Books/NewName
  remarquee cloud mv /Books/Doc /Articles
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
				"src",
				fields.TypeString,
				fields.WithIsArgument(true),
				fields.WithRequired(true),
				fields.WithHelp("Source remote path (can include patterns)"),
			),
			fields.New(
				"dst",
				fields.TypeString,
				fields.WithIsArgument(true),
				fields.WithRequired(true),
				fields.WithHelp("Destination path or directory"),
			),
		),
		glazecmds.WithSections(glazedLayer, commandSettingsLayer),
	)

	return &MvCommand{CommandDescription: cmdDesc}, nil
}

func (c *MvCommand) Run(ctx context.Context, parsedValues *values.Values) error {
	s := &MvSettings{}
	if err := parsedValues.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
		return err
	}

	_, apiCtx, err := createApiCtx(s.AuthSettings)
	if err != nil {
		return err
	}

	srcNodes, err := apiCtx.Filetree().NodesByPath(s.Src, nil, false)
	if err != nil {
		return err
	}
	if len(srcNodes) < 1 {
		return errors.New("no nodes found")
	}

	dstNode, _ := apiCtx.Filetree().NodeByPath(s.Dst, nil)
	if dstNode != nil && dstNode.IsFile() {
		return errors.New("destination entry already exists")
	}

	// Move into existing directory
	if dstNode != nil && dstNode.IsDirectory() {
		for _, node := range srcNodes {
			if isSubdir(node, dstNode) {
				return errors.Errorf("cannot move: %s in itself", node.Name())
			}

			n, err := apiCtx.MoveEntry(node, dstNode, node.Name())
			if err != nil {
				return errors.Wrap(err, "failed to move entry")
			}

			apiCtx.Filetree().MoveNode(node, n)
		}

		if err := apiCtx.SyncComplete(); err != nil {
			return errors.Wrap(err, "cannot notify")
		}
		return nil
	}

	// Rename/move to explicit path (only one source)
	if len(srcNodes) > 1 {
		return errors.New("cannot rename multiple nodes, only first match will be renamed")
	}

	srcNode := srcNodes[0]
	parentDir := path.Dir(s.Dst)
	newEntry := path.Base(s.Dst)

	parentNode, err := apiCtx.Filetree().NodeByPath(parentDir, nil)
	if err != nil || parentNode.IsFile() {
		return errors.Wrap(err, "cannot move")
	}

	n, err := apiCtx.MoveEntry(srcNode, parentNode, newEntry)
	if err != nil {
		return errors.Wrap(err, "failed to move entry")
	}
	if err := apiCtx.SyncComplete(); err != nil {
		return errors.Wrap(err, "cannot notify")
	}

	apiCtx.Filetree().MoveNode(srcNode, n)
	return nil
}

func NewMvCobraCommand() (*cobra.Command, error) {
	cmd, err := NewMvCommand()
	if err != nil {
		return nil, err
	}

	return cli.BuildCobraCommand(cmd,
		cli.WithParserConfig(cli.CobraParserConfig{
			ShortHelpSections: []string{schema.DefaultSlug},
			MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
		}),
	)
}

// isSubdir checks for moves like: a -> a/sub1 which would result in data loss.
func isSubdir(parent *model.Node, child *model.Node) bool {
	for child != nil {
		if parent.Id() == child.Id() {
			return true
		}
		child = child.Parent
	}
	return false
}
