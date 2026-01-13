package cloud

import (
	"context"
	"fmt"

	"github.com/go-go-golems/glazed/pkg/cli"
	glazecmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/juruen/rmapi/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type PutCommand struct {
	*glazecmds.CommandDescription
}

type PutSettings struct {
	AuthSettings

	Local        string `glazed.parameter:"local"`
	RemoteDir    string `glazed.parameter:"remote-dir"`
	Force        bool   `glazed.parameter:"force"`
	ContentOnly  bool   `glazed.parameter:"content-only"`
	CoverpageStr string `glazed.parameter:"coverpage"`
}

var _ glazecmds.BareCommand = &PutCommand{}

func NewPutCommand() (*PutCommand, error) {
	glazedLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}
	commandSettingsLayer, err := cli.NewCommandSettingsLayer()
	if err != nil {
		return nil, err
	}

	cmdDesc := glazecmds.NewCommandDescription(
		"put",
		glazecmds.WithShort("Upload a local document to the cloud"),
		glazecmds.WithLong(`
Uploads a local document to the reMarkable cloud (rmapi-backed).

This mirrors rmapi's semantics:
- If the target name already exists:
  - without --force: error
  - with --force: delete existing and upload new
- With --content-only (PDF only): replace PDF content of an existing document (or create if missing)

Examples:
  remarquee cloud put ./doc.pdf /Books
  remarquee cloud put ./doc.pdf /Books --force
  remarquee cloud put ./doc.pdf /Books --content-only
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
				"force",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithHelp("Overwrite existing file (recreates document)"),
			),
			parameters.NewParameterDefinition(
				"content-only",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithHelp("Replace PDF content only (preserves document metadata)"),
			),
			parameters.NewParameterDefinition(
				"coverpage",
				parameters.ParameterTypeString,
				parameters.WithDefault(""),
				parameters.WithHelp("Set coverpage (0 to disable, 1 to set first page as cover)"),
			),

			parameters.NewParameterDefinition(
				"local",
				parameters.ParameterTypeString,
				parameters.WithIsArgument(true),
				parameters.WithRequired(true),
				parameters.WithHelp("Local file to upload"),
			),
			parameters.NewParameterDefinition(
				"remote-dir",
				parameters.ParameterTypeString,
				parameters.WithIsArgument(true),
				parameters.WithDefault("/"),
				parameters.WithHelp("Remote directory to upload into (default: /)"),
			),
		),
		glazecmds.WithLayersList(glazedLayer, commandSettingsLayer),
	)

	return &PutCommand{CommandDescription: cmdDesc}, nil
}

func (c *PutCommand) Run(ctx context.Context, parsedLayers *layers.ParsedLayers) error {
	s := &PutSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, s); err != nil {
		return err
	}

	if s.Force && s.ContentOnly {
		return errors.New("--force and --content-only cannot be used together")
	}

	coverpageFlag, err := parseCoverpageFlag(s.CoverpageStr)
	if err != nil {
		return err
	}

	_, apiCtx, err := createApiCtx(s.AuthSettings)
	if err != nil {
		return err
	}

	dstNode, err := apiCtx.Filetree().NodeByPath(s.RemoteDir, nil)
	if err != nil || dstNode.IsFile() {
		return errors.New("directory doesn't exist")
	}

	docName, ext := util.DocPathToName(s.Local)

	// content-only mode (PDF only)
	if s.ContentOnly {
		if ext != "pdf" {
			return errors.New("--content-only can only be used with PDF files")
		}

		existingNode, err := apiCtx.Filetree().NodeByPath(docName, dstNode)
		if err != nil {
			document, err := apiCtx.UploadDocument(dstNode.Id(), s.Local, true, coverpageFlag)
			if err != nil {
				return errors.Wrapf(err, "failed to upload file [%s]", s.Local)
			}
			apiCtx.Filetree().AddDocument(document)
			fmt.Printf("OK: uploaded %s -> %s\n", s.Local, buildPathFromParents(dstNode))
			return nil
		}

		if existingNode.IsDirectory() {
			return errors.New("cannot replace directory with file")
		}

		if err := apiCtx.ReplaceDocumentFile(existingNode.Document.ID, s.Local, true); err != nil {
			return errors.Wrap(err, "failed to replace content")
		}

		fmt.Printf("OK: replaced content of %s\n", docName)
		return nil
	}

	// regular upload / --force
	existingNode, err := apiCtx.Filetree().NodeByPath(docName, dstNode)
	if err == nil {
		if !s.Force {
			return errors.New("entry already exists (use --force to recreate, --content-only to replace content)")
		}

		if existingNode.IsDirectory() {
			return errors.New("cannot overwrite directory with file")
		}

		if err := apiCtx.DeleteEntry(existingNode, false, false); err != nil {
			return errors.Wrap(err, "failed to delete existing file")
		}
		apiCtx.Filetree().DeleteNode(existingNode)
	}

	document, err := apiCtx.UploadDocument(dstNode.Id(), s.Local, true, coverpageFlag)
	if err != nil {
		return errors.Wrapf(err, "failed to upload file [%s]", s.Local)
	}
	apiCtx.Filetree().AddDocument(document)

	fmt.Printf("OK: uploaded %s -> %s\n", s.Local, buildPathFromParents(dstNode))
	return nil
}

func NewPutCobraCommand() (*cobra.Command, error) {
	cmd, err := NewPutCommand()
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

func parseCoverpageFlag(s string) (*int, error) {
	if s == "" {
		return nil, nil
	}

	switch s {
	case "0":
		return nil, nil
	case "1":
		val := 0
		return &val, nil
	default:
		return nil, errors.New("--coverpage value must be 0 or 1")
	}
}
