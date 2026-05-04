package cloud

import (
	"context"
	"fmt"

	"github.com/go-go-golems/glazed/pkg/cli"
	glazecmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/remarquee/cmd/remarquee/internal/appconfig"
	"github.com/juruen/rmapi/util"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type PutCommand struct {
	*glazecmds.CommandDescription
}

type PutSettings struct {
	AuthSettings

	Local        string `glazed:"local"`
	RemoteDir    string `glazed:"remote-dir"`
	Force        bool   `glazed:"force"`
	ContentOnly  bool   `glazed:"content-only"`
	CoverpageStr string `glazed:"coverpage"`
}

var _ glazecmds.BareCommand = &PutCommand{}

func NewPutCommand() (*PutCommand, error) {
	glazedLayer, err := settings.NewGlazedSection()
	if err != nil {
		return nil, err
	}
	commandSettingsLayer, err := cli.NewCommandSettingsSection()
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
				"force",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Overwrite existing file (recreates document)"),
			),
			fields.New(
				"content-only",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Replace PDF content only (preserves document metadata)"),
			),
			fields.New(
				"coverpage",
				fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("Set coverpage (0 to disable, 1 to set first page as cover)"),
			),

			fields.New(
				"local",
				fields.TypeString,
				fields.WithIsArgument(true),
				fields.WithRequired(true),
				fields.WithHelp("Local file to upload"),
			),
			fields.New(
				"remote-dir",
				fields.TypeString,
				fields.WithIsArgument(true),
				fields.WithDefault("/"),
				fields.WithHelp("Remote directory to upload into (default: /)"),
			),
		),
		glazecmds.WithSections(glazedLayer, commandSettingsLayer),
	)

	return &PutCommand{CommandDescription: cmdDesc}, nil
}

func (c *PutCommand) Run(ctx context.Context, parsedValues *values.Values) error {
	s := &PutSettings{}
	if err := parsedValues.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
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
	log.Debug().
		Str("local", s.Local).
		Str("remote_dir", s.RemoteDir).
		Str("doc_name", docName).
		Str("ext", ext).
		Str("parent_id", dstNode.Id()).
		Bool("force", s.Force).
		Bool("content_only", s.ContentOnly).
		Interface("coverpage", coverpageFlag).
		Msg("cloud put: starting upload")

	// content-only mode (PDF only)
	if s.ContentOnly {
		if ext != "pdf" {
			return errors.New("--content-only can only be used with PDF files")
		}

		existingNode, err := apiCtx.Filetree().NodeByPath(docName, dstNode)
		if err != nil {
			log.Debug().Str("doc_name", docName).Msg("cloud put: content-only: document not found, creating new")
			document, err := apiCtx.UploadDocument(dstNode.Id(), s.Local, true, coverpageFlag, nil, nil, nil)
			if err != nil {
				log.Error().Err(err).Str("file", s.Local).Str("parent_id", dstNode.Id()).Msg("cloud put: content-only upload failed")
				return errors.Wrapf(err, "failed to upload file [%s]", s.Local)
			}
			apiCtx.Filetree().AddDocument(document)
			fmt.Printf("OK: uploaded %s -> %s\n", s.Local, buildPathFromParents(dstNode))
			return nil
		}

		if existingNode.IsDirectory() {
			return errors.New("cannot replace directory with file")
		}

		log.Debug().Str("doc_id", existingNode.Document.ID).Str("file", s.Local).Msg("cloud put: content-only: replacing document file")
		if err := apiCtx.ReplaceDocumentFile(existingNode.Document.ID, s.Local, true); err != nil {
			log.Error().Err(err).Str("doc_id", existingNode.Document.ID).Str("file", s.Local).Msg("cloud put: replace content failed")
			return errors.Wrap(err, "failed to replace content")
		}

		fmt.Printf("OK: replaced content of %s\n", docName)
		return nil
	}

	// regular upload / --force
	existingNode, err := apiCtx.Filetree().NodeByPath(docName, dstNode)
	if err == nil {
		log.Debug().Str("doc_name", docName).Msg("cloud put: entry already exists")
		if !s.Force {
			return errors.New("entry already exists (use --force to recreate, --content-only to replace content)")
		}

		if existingNode.IsDirectory() {
			return errors.New("cannot overwrite directory with file")
		}

		log.Debug().Str("doc_id", existingNode.Document.ID).Msg("cloud put: deleting existing entry for --force")
		if err := apiCtx.DeleteEntry(existingNode, false, false); err != nil {
			return errors.Wrap(err, "failed to delete existing file")
		}
		apiCtx.Filetree().DeleteNode(existingNode)
	} else {
		log.Debug().Str("doc_name", docName).Msg("cloud put: no existing entry found")
	}

	log.Debug().Str("file", s.Local).Str("parent_id", dstNode.Id()).Interface("coverpage", coverpageFlag).Msg("cloud put: uploading document")
	document, err := apiCtx.UploadDocument(dstNode.Id(), s.Local, true, coverpageFlag, nil, nil, nil)
	if err != nil {
		log.Error().Err(err).Str("file", s.Local).Str("parent_id", dstNode.Id()).Msg("cloud put: upload failed")
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
		cli.WithParserConfig(appconfig.DefaultParserConfig()),
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
