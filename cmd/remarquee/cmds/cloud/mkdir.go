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
	"github.com/go-go-golems/remarquee/cmd/remarquee/internal/appconfig"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type MkdirCommand struct {
	*glazecmds.CommandDescription
}

type MkdirSettings struct {
	AuthSettings

	Path string `glazed:"path"`
}

var _ glazecmds.BareCommand = &MkdirCommand{}

func NewMkdirCommand() (*MkdirCommand, error) {
	glazedLayer, err := settings.NewStructuredOutputSection()
	if err != nil {
		return nil, err
	}
	commandSettingsLayer, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	cmdDesc := glazecmds.NewCommandDescription(
		"mkdir",
		glazecmds.WithShort("Create a remote directory"),
		glazecmds.WithLong(`
Creates a directory in the reMarkable cloud (rmapi-backed).

Note: this does not currently implement recursive mkdir (-p). The parent directory must exist.

Examples:
  remarquee cloud mkdir /ai/2025/12/14/notes
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
				"path",
				fields.TypeString,
				fields.WithIsArgument(true),
				fields.WithRequired(true),
				fields.WithHelp("Remote directory path to create"),
			),
		),
		glazecmds.WithSections(glazedLayer, commandSettingsLayer),
	)

	return &MkdirCommand{CommandDescription: cmdDesc}, nil
}

func (c *MkdirCommand) Run(ctx context.Context, parsedValues *values.Values) error {
	s := &MkdirSettings{}
	if err := parsedValues.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
		return err
	}

	_, apiCtx, err := createApiCtx(s.AuthSettings)
	if err != nil {
		return err
	}

	// Does it already exist?
	if _, err := apiCtx.Filetree().NodeByPath(s.Path, nil); err == nil {
		return errors.New("entry already exists")
	}

	parentDir := path.Dir(s.Path)
	newDir := path.Base(s.Path)
	if newDir == "/" || newDir == "." {
		return errors.New("invalid directory name")
	}

	parentNode, err := apiCtx.Filetree().NodeByPath(parentDir, nil)
	if err != nil || parentNode.IsFile() {
		return errors.New("directory doesn't exist")
	}

	parentId := parentNode.Id()
	if parentNode.IsRoot() {
		parentId = ""
	}

	document, err := apiCtx.CreateDir(parentId, newDir, true)
	if err != nil {
		return errors.Wrap(err, "failed to create directory")
	}

	apiCtx.Filetree().AddDocument(document)
	return nil
}

func NewMkdirCobraCommand() (*cobra.Command, error) {
	cmd, err := NewMkdirCommand()
	if err != nil {
		return nil, err
	}

	return cli.BuildCobraCommand(cmd,
		cli.WithParserConfig(appconfig.DefaultParserConfig()),
	)
}
