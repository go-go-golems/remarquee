package cloud

import (
	"context"
	"path"

	"github.com/go-go-golems/glazed/pkg/cli"
	glazecmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type MkdirCommand struct {
	*glazecmds.CommandDescription
}

type MkdirSettings struct {
	AuthSettings

	Path string `glazed.parameter:"path"`
}

var _ glazecmds.BareCommand = &MkdirCommand{}

func NewMkdirCommand() (*MkdirCommand, error) {
	glazedLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}
	commandSettingsLayer, err := cli.NewCommandSettingsLayer()
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
				"path",
				parameters.ParameterTypeString,
				parameters.WithIsArgument(true),
				parameters.WithRequired(true),
				parameters.WithHelp("Remote directory path to create"),
			),
		),
		glazecmds.WithLayersList(glazedLayer, commandSettingsLayer),
	)

	return &MkdirCommand{CommandDescription: cmdDesc}, nil
}

func (c *MkdirCommand) Run(ctx context.Context, parsedLayers *layers.ParsedLayers) error {
	s := &MkdirSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, s); err != nil {
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
		cli.WithParserConfig(cli.CobraParserConfig{
			ShortHelpLayers: []string{layers.DefaultSlug},
			MiddlewaresFunc: cli.CobraCommandDefaultMiddlewares,
		}),
	)
}
