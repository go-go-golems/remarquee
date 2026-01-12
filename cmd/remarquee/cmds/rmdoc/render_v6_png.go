package rmdoc

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cli"
	glazecmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	pkg_rmdoc "github.com/go-go-golems/remarquee/pkg/rmdoc"
	rmdocrender "github.com/go-go-golems/remarquee/pkg/rmdoc/render"
)

type RenderV6PNGCommand struct {
	*glazecmds.CommandDescription
}

type RenderV6PNGSettings struct {
	File   string `glazed.parameter:"file"`
	Pages  string `glazed.parameter:"pages"`
	OutDir string `glazed.parameter:"out-dir"`
	PDFOut string `glazed.parameter:"pdf-out"`

	DPI      int    `glazed.parameter:"dpi"`
	PDFToPPM string `glazed.parameter:"pdftoppm"`
	Force    bool   `glazed.parameter:"force"`
}

var _ glazecmds.BareCommand = &RenderV6PNGCommand{}

func NewRenderV6PNGCommand() (*RenderV6PNGCommand, error) {
	glazedLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}
	commandSettingsLayer, err := cli.NewCommandSettingsLayer()
	if err != nil {
		return nil, err
	}

	cmdDesc := glazecmds.NewCommandDescription(
		"render-v6-png",
		glazecmds.WithShort("Render a V6 (cPages) .rmdoc to PNG images"),
		glazecmds.WithLong(`
Render a V6 (cPages) .rmdoc to PNG images by generating a merged PDF and rasterizing
selected pages with Poppler's pdftoppm.

Notes:
- This command currently uses Poppler for rasterization.
- Output PNG names follow: <basename>-v6-page-XXX.png.
`),
		glazecmds.WithFlags(
			parameters.NewParameterDefinition(
				"file",
				parameters.ParameterTypeString,
				parameters.WithIsArgument(true),
				parameters.WithRequired(true),
				parameters.WithHelp("Path to the V6 .rmdoc file"),
			),
			parameters.NewParameterDefinition(
				"pages",
				parameters.ParameterTypeString,
				parameters.WithDefault("1"),
				parameters.WithHelp("Comma-separated 1-based page numbers to render (e.g. '1,2,5')"),
			),
			parameters.NewParameterDefinition(
				"out-dir",
				parameters.ParameterTypeString,
				parameters.WithDefault("."),
				parameters.WithHelp("Directory to write PNGs (and merged PDF) into"),
			),
			parameters.NewParameterDefinition(
				"pdf-out",
				parameters.ParameterTypeString,
				parameters.WithDefault(""),
				parameters.WithHelp("Optional path for the merged PDF (default: <out-dir>/<basename>-v6.pdf)"),
			),
			parameters.NewParameterDefinition(
				"dpi",
				parameters.ParameterTypeInteger,
				parameters.WithDefault(200),
				parameters.WithHelp("Rasterization DPI (pdftoppm)"),
			),
			parameters.NewParameterDefinition(
				"pdftoppm",
				parameters.ParameterTypeString,
				parameters.WithDefault("pdftoppm"),
				parameters.WithHelp("pdftoppm executable (poppler rasterizer)"),
			),
			parameters.NewParameterDefinition(
				"force",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithHelp("Overwrite existing PNG/PDF outputs"),
			),
		),
		glazecmds.WithLayersList(glazedLayer, commandSettingsLayer),
	)

	return &RenderV6PNGCommand{CommandDescription: cmdDesc}, nil
}

func (c *RenderV6PNGCommand) Run(ctx context.Context, parsedLayers *layers.ParsedLayers) error {
	s := &RenderV6PNGSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, s); err != nil {
		return err
	}

	pages, err := parsePages1Based(s.Pages)
	if err != nil {
		return err
	}

	pageIndices := make([]int, 0, len(pages))
	for _, p := range pages {
		pageIndices = append(pageIndices, p-1)
	}

	doc, err := pkg_rmdoc.OpenFile(ctx, s.File)
	if err != nil {
		return err
	}
	if doc.Schema != pkg_rmdoc.SchemaCPages {
		return errors.Errorf("render-v6-png only supports cPages/V6 archives; detected schema=%s", schemaString(doc.Schema))
	}
	if doc.Type == pkg_rmdoc.DocTypeEPUB {
		return errors.New("render-v6-png: epub not supported")
	}

	outDir := strings.TrimSpace(s.OutDir)
	if outDir == "" {
		outDir = "."
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return errors.Wrap(err, "ensure output dir")
	}

	base := filepath.Base(s.File)
	ext := filepath.Ext(base)
	if ext != "" {
		base = base[:len(base)-len(ext)]
	}
	prefix := base + "-v6"

	pdfOut := strings.TrimSpace(s.PDFOut)
	if pdfOut == "" {
		pdfOut = filepath.Join(outDir, prefix+".pdf")
	}

	if !s.Force {
		if _, err := os.Stat(pdfOut); err == nil {
			return errors.Errorf("output file exists: %s (use --force to overwrite)", pdfOut)
		}
		for _, pageNum := range pages {
			outFile := filepath.Join(outDir, fmt.Sprintf("%s-page-%03d.png", prefix, pageNum))
			if _, err := os.Stat(outFile); err == nil {
				return errors.Errorf("output file exists: %s (use --force to overwrite)", outFile)
			}
		}
	}

	res, err := rmdocrender.MergeRMDocV6OntoBackgroundPDFWithInfoForPages(ctx, s.File, rmdocrender.V6MergeOptions{}, pageIndices)
	if err != nil {
		return err
	}
	if err := os.WriteFile(pdfOut, res.PDF, 0o644); err != nil {
		return errors.Wrap(err, "write output pdf")
	}

	renderPages := make([]pageRenderSpec, 0, len(pages))
	for i, label := range pages {
		renderPages = append(renderPages, pageRenderSpec{
			PDFPage: i + 1,
			Label:   label,
		})
	}
	imgs, err := renderPDFPagesToPNGsWithPopplerMapped(ctx, s.PDFToPPM, s.DPI, pdfOut, outDir, prefix, renderPages)
	if err != nil {
		return err
	}

	fmt.Printf("ok: wrote %s\n", pdfOut)
	for _, img := range imgs {
		fmt.Printf("ok: wrote %s\n", img)
	}
	return nil
}

type pageRenderSpec struct {
	PDFPage int
	Label   int
}

func renderPDFPagesToPNGsWithPopplerMapped(ctx context.Context, pdftoppm string, dpi int, pdfPath, outDir, prefix string, pages []pageRenderSpec) ([]string, error) {
	if dpi <= 0 {
		dpi = 200
	}
	if _, err := exec.LookPath(pdftoppm); err != nil {
		return nil, errors.Wrap(err, "pdftoppm not found on PATH")
	}

	var out []string
	for _, page := range pages {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		outBase := filepath.Join(outDir, fmt.Sprintf("%s-page-%03d", prefix, page.Label))
		outFile := outBase + ".png"

		cmd := exec.CommandContext(ctx,
			pdftoppm,
			"-png",
			"-r", strconv.Itoa(dpi),
			"-f", strconv.Itoa(page.PDFPage),
			"-l", strconv.Itoa(page.PDFPage),
			"-singlefile",
			pdfPath,
			outBase,
		)

		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return nil, errors.Wrapf(err, "pdftoppm page %d failed: %s", page.PDFPage, strings.TrimSpace(stderr.String()))
		}

		if _, err := os.Stat(outFile); err != nil {
			return nil, errors.Wrapf(err, "pdftoppm did not produce expected output: %s", outFile)
		}
		out = append(out, outFile)
	}
	return out, nil
}

func NewRenderV6PNGCobraCommand() (*cobra.Command, error) {
	cmd, err := NewRenderV6PNGCommand()
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
