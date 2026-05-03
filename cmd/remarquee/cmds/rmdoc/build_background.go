package rmdoc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-go-golems/glazed/pkg/cli"
	glazecmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/remarquee/cmd/remarquee/internal/appconfig"
	pkg_rmdoc "github.com/go-go-golems/remarquee/pkg/rmdoc"
	rmdocrender "github.com/go-go-golems/remarquee/pkg/rmdoc/render"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type BuildBackgroundCommand struct {
	*glazecmds.CommandDescription
}

type BuildBackgroundSettings struct {
	File  string `glazed:"file"`
	Out   string `glazed:"out"`
	Force bool   `glazed:"force"`
}

var _ glazecmds.BareCommand = &BuildBackgroundCommand{}

func NewBuildBackgroundCommand() (*BuildBackgroundCommand, error) {
	glazedLayer, err := settings.NewGlazedSection()
	if err != nil {
		return nil, err
	}
	commandSettingsLayer, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	cmdDesc := glazecmds.NewCommandDescription(
		"build-background",
		glazecmds.WithShort("Build a UI-ordered background PDF using PageRef.SourcePDFPage (debug utility)"),
		glazecmds.WithLong(`
Build a background PDF in UI page order based on the parsed page plan.

For PDF-backed docs it copies payload PDF pages and inserts blank pages for InsertedPage.
For notebooks (no payload) it currently creates blank pages using a default size.
`),
		glazecmds.WithFlags(
			fields.New(
				"out",
				fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("Output PDF path (default: <input>-background.pdf in current dir)"),
			),
			fields.New(
				"force",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Overwrite output file if it exists"),
			),
			fields.New(
				"file",
				fields.TypeString,
				fields.WithIsArgument(true),
				fields.WithRequired(true),
				fields.WithHelp("Path to the .rmdoc file"),
			),
		),
		glazecmds.WithSections(glazedLayer, commandSettingsLayer),
	)

	return &BuildBackgroundCommand{CommandDescription: cmdDesc}, nil
}

func (c *BuildBackgroundCommand) Run(ctx context.Context, parsedValues *values.Values) error {
	s := &BuildBackgroundSettings{}
	if err := parsedValues.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
		return err
	}

	doc, err := pkg_rmdoc.OpenFile(ctx, s.File)
	if err != nil {
		return err
	}
	if doc.Type == pkg_rmdoc.DocTypeEPUB {
		return errors.New("build-background: epub not supported")
	}

	out := s.Out
	if out == "" {
		base := filepath.Base(s.File)
		ext := filepath.Ext(base)
		base = base[:len(base)-len(ext)]
		out = base + "-background.pdf"
	}

	if !s.Force {
		if _, err := os.Stat(out); err == nil {
			return errors.Errorf("output file exists: %s (use --force to overwrite)", out)
		}
	}

	bg, err := rmdocrender.BuildBackgroundPDF(ctx, doc, rmdocrender.BackgroundOptions{})
	if err != nil {
		return err
	}

	if err := os.WriteFile(out, bg, 0o644); err != nil {
		return errors.Wrap(err, "write output pdf")
	}

	fmt.Printf("ok: wrote %s\n", out)
	return nil
}

func NewBuildBackgroundCobraCommand() (*cobra.Command, error) {
	cmd, err := NewBuildBackgroundCommand()
	if err != nil {
		return nil, err
	}

	return cli.BuildCobraCommand(cmd,
		cli.WithParserConfig(appconfig.DefaultParserConfig()),
	)
}
