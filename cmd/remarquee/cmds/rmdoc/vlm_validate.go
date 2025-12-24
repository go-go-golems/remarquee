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
)

type VLMValidateCommand struct {
	*glazecmds.CommandDescription
}

type VLMValidateSettings struct {
	PDFA string `glazed.parameter:"pdf-a"`
	PDFB string `glazed.parameter:"pdf-b"`

	Pages string `glazed.parameter:"pages"`

	OutDir string `glazed.parameter:"out-dir"`

	Rasterizer string `glazed.parameter:"rasterizer"`
	DPI        int    `glazed.parameter:"dpi"`
	PDFToPPM   string `glazed.parameter:"pdftoppm"`

	Pinocchio string `glazed.parameter:"pinocchio"`
	Prompt    string `glazed.parameter:"prompt"`
}

var _ glazecmds.BareCommand = &VLMValidateCommand{}

func NewVLMValidateCommand() (*VLMValidateCommand, error) {
	glazedLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}
	commandSettingsLayer, err := cli.NewCommandSettingsLayer()
	if err != nil {
		return nil, err
	}

	cmdDesc := glazecmds.NewCommandDescription(
		"vlm-validate",
		glazecmds.WithShort("Render PDF pages to PNG and ask pinocchio (VLM) to validate/compare"),
		glazecmds.WithLong(`
Render pages from one or two PDFs to PNG images and invoke pinocchio's vision model.

Typical use:
- Render remarquee output and a reference output to images
- Ask the VLM to describe the images and spot misalignment/missing content

Notes:
- This is intended as an optional validation helper (manual/interactive workflow).
- PNG rasterization defaults to Poppler (pdftoppm) because UniDoc's renderer can fail with "type check error" on some PDFs.
`),
		glazecmds.WithFlags(
			parameters.NewParameterDefinition(
				"pdf-a",
				parameters.ParameterTypeString,
				parameters.WithIsArgument(true),
				parameters.WithRequired(true),
				parameters.WithHelp("Path to PDF A (e.g. remarquee output)"),
			),
			parameters.NewParameterDefinition(
				"pdf-b",
				parameters.ParameterTypeString,
				parameters.WithDefault(""),
				parameters.WithHelp("Optional path to PDF B (e.g. reference output)"),
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
				parameters.WithDefault(""),
				parameters.WithHelp("Directory to write PNGs to (default: temp dir)"),
			),
			parameters.NewParameterDefinition(
				"rasterizer",
				parameters.ParameterTypeString,
				parameters.WithDefault("poppler"),
				parameters.WithHelp("PDF->image rasterizer: poppler (pdftoppm) or unidoc (unidoc currently disabled due to type check errors)"),
			),
			parameters.NewParameterDefinition(
				"dpi",
				parameters.ParameterTypeInteger,
				parameters.WithDefault(200),
				parameters.WithHelp("Rasterization DPI (poppler only)"),
			),
			parameters.NewParameterDefinition(
				"pdftoppm",
				parameters.ParameterTypeString,
				parameters.WithDefault("pdftoppm"),
				parameters.WithHelp("pdftoppm executable (poppler rasterizer)"),
			),
			parameters.NewParameterDefinition(
				"pinocchio",
				parameters.ParameterTypeString,
				parameters.WithDefault("pinocchio"),
				parameters.WithHelp("Pinocchio executable (default: pinocchio)"),
			),
			parameters.NewParameterDefinition(
				"prompt",
				parameters.ParameterTypeString,
				parameters.WithDefault("Describe these images. Check for missing/misaligned strokes, missing highlights, missing typed text, incorrect page size, and missing template background. If two PDFs are included, compare A vs B and list differences."),
				parameters.WithHelp("Prompt to send to the VLM"),
			),
		),
		glazecmds.WithLayersList(glazedLayer, commandSettingsLayer),
	)

	return &VLMValidateCommand{CommandDescription: cmdDesc}, nil
}

func (c *VLMValidateCommand) Run(ctx context.Context, parsedLayers *layers.ParsedLayers) error {
	s := &VLMValidateSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, s); err != nil {
		return err
	}

	pages, err := parsePages1Based(s.Pages)
	if err != nil {
		return err
	}

	outDir := strings.TrimSpace(s.OutDir)
	if outDir == "" {
		outDir, err = os.MkdirTemp("", "remarquee-vlm-validate-*")
		if err != nil {
			return errors.Wrap(err, "create temp out dir")
		}
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return errors.Wrap(err, "ensure out dir")
	}

	imgsA, err := renderPDFPagesToPNGs(ctx, s, s.PDFA, outDir, "A", pages)
	if err != nil {
		return err
	}
	var imgsB []string
	if strings.TrimSpace(s.PDFB) != "" {
		imgsB, err = renderPDFPagesToPNGs(ctx, s, s.PDFB, outDir, "B", pages)
		if err != nil {
			return err
		}
	}

	var images []string
	images = append(images, imgsA...)
	images = append(images, imgsB...)

	if _, err := exec.LookPath(s.Pinocchio); err != nil {
		return errors.Wrap(err, "pinocchio not found on PATH")
	}

	imagesArg := strings.Join(images, ",")
	cmd := exec.CommandContext(ctx, s.Pinocchio, "code", "professional", "--images", imagesArg, s.Prompt)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("ok: wrote images to %s\n", outDir)
	fmt.Printf("ok: running: %s\n", strings.Join(cmd.Args, " "))
	return cmd.Run()
}

func NewVLMValidateCobraCommand() (*cobra.Command, error) {
	cmd, err := NewVLMValidateCommand()
	if err != nil {
		return nil, err
	}
	return cli.BuildCobraCommand(cmd,
		cli.WithDualMode(false),
		cli.WithParserConfig(cli.CobraParserConfig{
			ShortHelpLayers: []string{layers.DefaultSlug},
			MiddlewaresFunc: cli.CobraCommandDefaultMiddlewares,
		}),
	)
}

func parsePages1Based(s string) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return []int{1}, nil
	}
	parts := strings.Split(s, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, errors.Wrapf(err, "parse page %q", p)
		}
		if n <= 0 {
			return nil, errors.Errorf("page numbers must be 1-based, got %d", n)
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return []int{1}, nil
	}
	return out, nil
}

func renderPDFPagesToPNGs(ctx context.Context, s *VLMValidateSettings, pdfPath, outDir, prefix string, pages []int) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	switch strings.ToLower(strings.TrimSpace(s.Rasterizer)) {
	case "", "poppler", "pdftoppm":
		return renderPDFPagesToPNGsWithPoppler(ctx, s.PDFToPPM, s.DPI, pdfPath, outDir, prefix, pages)
	case "unidoc":
		return nil, errors.New("unidoc rasterizer is temporarily disabled due to 'type check error' failures; use --rasterizer poppler")
	default:
		return nil, errors.Errorf("unknown rasterizer: %q", s.Rasterizer)
	}
}

func renderPDFPagesToPNGsWithPoppler(ctx context.Context, pdftoppm string, dpi int, pdfPath, outDir, prefix string, pages []int) ([]string, error) {
	if dpi <= 0 {
		dpi = 200
	}
	if _, err := exec.LookPath(pdftoppm); err != nil {
		return nil, errors.Wrap(err, "pdftoppm not found on PATH")
	}

	var out []string
	for _, pageNum := range pages {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// pdftoppm -singlefile writes "<outBase>.png" (no "-1" suffix).
		outBase := filepath.Join(outDir, fmt.Sprintf("%s-page-%03d", prefix, pageNum))
		outFile := outBase + ".png"

		cmd := exec.CommandContext(ctx,
			pdftoppm,
			"-png",
			"-r", strconv.Itoa(dpi),
			"-f", strconv.Itoa(pageNum),
			"-l", strconv.Itoa(pageNum),
			"-singlefile",
			pdfPath,
			outBase,
		)

		// Keep stderr visible to help diagnose Poppler failures.
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return nil, errors.Wrapf(err, "pdftoppm page %d failed: %s", pageNum, strings.TrimSpace(stderr.String()))
		}

		if _, err := os.Stat(outFile); err != nil {
			return nil, errors.Wrapf(err, "pdftoppm did not produce expected output: %s", outFile)
		}
		out = append(out, outFile)
	}
	return out, nil
}


