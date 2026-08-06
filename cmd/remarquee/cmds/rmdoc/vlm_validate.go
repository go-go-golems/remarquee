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
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/remarquee/cmd/remarquee/internal/appconfig"
	rmdocrender "github.com/go-go-golems/remarquee/pkg/rmdoc/render"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type VLMValidateCommand struct {
	*glazecmds.CommandDescription
}

type VLMValidateSettings struct {
	PDFA string `glazed:"pdf-a"`
	PDFB string `glazed:"pdf-b"`

	ImageA string `glazed:"image-a"`
	ImageB string `glazed:"image-b"`

	RMDocA string `glazed:"rmdoc-a"`
	RMDocB string `glazed:"rmdoc-b"`

	Pages string `glazed:"pages"`

	OutDir string `glazed:"out-dir"`

	Rasterizer string `glazed:"rasterizer"`
	DPI        int    `glazed:"dpi"`
	PDFToPPM   string `glazed:"pdftoppm"`

	Pinocchio string `glazed:"pinocchio"`
	Prompt    string `glazed:"prompt"`
}

var _ glazecmds.BareCommand = &VLMValidateCommand{}

func NewVLMValidateCommand() (*VLMValidateCommand, error) {
	glazedLayer, err := settings.NewStructuredOutputSection()
	if err != nil {
		return nil, err
	}
	commandSettingsLayer, err := cli.NewCommandSettingsSection()
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
- Inputs can be provided as PDFs, PNGs, or .rmdoc files. For .rmdoc inputs, pages are rendered to PNGs using the V6 merge pipeline.
`),
		glazecmds.WithFlags(
			fields.New(
				"pdf-a",
				fields.TypeString,
				fields.WithIsArgument(true),
				fields.WithHelp("Path to PDF A (e.g. remarquee output)"),
			),
			fields.New(
				"pdf-b",
				fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("Optional path to PDF B (e.g. reference output)"),
			),
			fields.New(
				"image-a",
				fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("Path to PNG A (skip PDF rendering)"),
			),
			fields.New(
				"image-b",
				fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("Path to PNG B (skip PDF rendering)"),
			),
			fields.New(
				"rmdoc-a",
				fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("Path to .rmdoc A (render V6 pages to PNGs)"),
			),
			fields.New(
				"rmdoc-b",
				fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("Path to .rmdoc B (render V6 pages to PNGs)"),
			),
			fields.New(
				"pages",
				fields.TypeString,
				fields.WithDefault("1"),
				fields.WithHelp("Comma-separated 1-based page numbers to render (for PDF or .rmdoc inputs)"),
			),
			fields.New(
				"out-dir",
				fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("Directory to write PNGs to (default: temp dir)"),
			),
			fields.New(
				"rasterizer",
				fields.TypeString,
				fields.WithDefault("poppler"),
				fields.WithHelp("PDF->image rasterizer: poppler (pdftoppm) or unidoc (unidoc currently disabled due to type check errors)"),
			),
			fields.New(
				"dpi",
				fields.TypeInteger,
				fields.WithDefault(200),
				fields.WithHelp("Rasterization DPI (poppler only)"),
			),
			fields.New(
				"pdftoppm",
				fields.TypeString,
				fields.WithDefault("pdftoppm"),
				fields.WithHelp("pdftoppm executable (poppler rasterizer)"),
			),
			fields.New(
				"pinocchio",
				fields.TypeString,
				fields.WithDefault("pinocchio"),
				fields.WithHelp("Pinocchio executable (default: pinocchio)"),
			),
			fields.New(
				"prompt",
				fields.TypeString,
				fields.WithDefault("Describe these images. Check for missing/misaligned strokes, missing highlights, missing typed text, incorrect page size, and missing template background. If two PDFs are included, compare A vs B and list differences."),
				fields.WithHelp("Prompt to send to the VLM"),
			),
		),
		glazecmds.WithSections(glazedLayer, commandSettingsLayer),
	)

	return &VLMValidateCommand{CommandDescription: cmdDesc}, nil
}

func (c *VLMValidateCommand) Run(ctx context.Context, parsedValues *values.Values) error {
	s := &VLMValidateSettings{}
	if err := parsedValues.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
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

	srcA, err := resolveVLMSource("A", s.PDFA, s.ImageA, s.RMDocA, true)
	if err != nil {
		return err
	}
	srcB, err := resolveVLMSource("B", s.PDFB, s.ImageB, s.RMDocB, false)
	if err != nil {
		return err
	}

	imgsA, err := renderVLMSource(ctx, s, srcA, outDir, "A", pages)
	if err != nil {
		return err
	}
	var imgsB []string
	if srcB.Kind != "" {
		imgsB, err = renderVLMSource(ctx, s, srcB, outDir, "B", pages)
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
	// Ensure non-interactive runs (CI/scripts) don't block on "continue in chat?" prompts.
	cmd := exec.CommandContext(ctx, s.Pinocchio, "code", "professional", "--non-interactive", "--output", "text", "--images", imagesArg, s.Prompt) // #nosec G204 -- command intentionally invokes the configured pinocchio binary with explicit argv.
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("ok: wrote images to %s\n", outDir)
	fmt.Printf("ok: running: %s\n", strings.Join(cmd.Args, " "))
	return cmd.Run()
}

type vlmSourceKind string

const (
	vlmSourcePDF   vlmSourceKind = "pdf"
	vlmSourceImage vlmSourceKind = "image"
	vlmSourceRMDoc vlmSourceKind = "rmdoc"
)

type vlmSource struct {
	Kind vlmSourceKind
	Path string
}

func resolveVLMSource(label, pdf, image, rmdoc string, required bool) (vlmSource, error) {
	pdf = strings.TrimSpace(pdf)
	image = strings.TrimSpace(image)
	rmdoc = strings.TrimSpace(rmdoc)

	count := 0
	if pdf != "" {
		count++
	}
	if image != "" {
		count++
	}
	if rmdoc != "" {
		count++
	}

	if count == 0 {
		if required {
			return vlmSource{}, errors.Errorf("missing input for %s (provide --pdf-%s, --image-%s, or --rmdoc-%s)", label, strings.ToLower(label), strings.ToLower(label), strings.ToLower(label))
		}
		return vlmSource{}, nil
	}
	if count > 1 {
		return vlmSource{}, errors.Errorf("multiple inputs for %s; choose only one of --pdf-%s, --image-%s, or --rmdoc-%s", label, strings.ToLower(label), strings.ToLower(label), strings.ToLower(label))
	}

	switch {
	case pdf != "":
		return vlmSource{Kind: vlmSourcePDF, Path: pdf}, nil
	case image != "":
		return vlmSource{Kind: vlmSourceImage, Path: image}, nil
	default:
		return vlmSource{Kind: vlmSourceRMDoc, Path: rmdoc}, nil
	}
}

func renderVLMSource(ctx context.Context, s *VLMValidateSettings, src vlmSource, outDir, prefix string, pages []int) ([]string, error) {
	switch src.Kind {
	case vlmSourceImage:
		if _, err := os.Stat(src.Path); err != nil {
			return nil, errors.Wrapf(err, "image not found: %s", src.Path)
		}
		return []string{src.Path}, nil
	case vlmSourcePDF:
		return renderPDFPagesToPNGs(ctx, s, src.Path, outDir, prefix, pages)
	case vlmSourceRMDoc:
		return renderRMDocPagesToPNGs(ctx, s, src.Path, outDir, prefix, pages)
	default:
		return nil, errors.New("unknown input source")
	}
}

func renderRMDocPagesToPNGs(ctx context.Context, s *VLMValidateSettings, rmdocPath, outDir, prefix string, pages []int) ([]string, error) {
	pageIndices := make([]int, 0, len(pages))
	for _, p := range pages {
		pageIndices = append(pageIndices, p-1)
	}

	res, err := rmdocrender.MergeRMDocV6OntoBackgroundPDFWithInfoForPages(ctx, rmdocPath, rmdocrender.V6MergeOptions{}, pageIndices)
	if err != nil {
		return nil, err
	}

	pdfFile, err := os.CreateTemp(outDir, fmt.Sprintf("%s-v6-*.pdf", strings.ToLower(prefix)))
	if err != nil {
		return nil, errors.Wrap(err, "create temp pdf")
	}
	defer func() {
		if err := pdfFile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: close pdf temp file: %v\n", err)
		}
	}()

	if _, err := pdfFile.Write(res.PDF); err != nil {
		return nil, errors.Wrap(err, "write temp pdf")
	}

	renderPages := make([]pageRenderSpec, 0, len(pages))
	for i, label := range pages {
		renderPages = append(renderPages, pageRenderSpec{
			PDFPage: i + 1,
			Label:   label,
		})
	}

	return renderPDFPagesToPNGsWithPopplerMapped(ctx, s.PDFToPPM, s.DPI, pdfFile.Name(), outDir, prefix, renderPages)
}

func NewVLMValidateCobraCommand() (*cobra.Command, error) {
	cmd, err := NewVLMValidateCommand()
	if err != nil {
		return nil, err
	}
	return cli.BuildCobraCommand(cmd,
		cli.WithDualMode(false),
		cli.WithParserConfig(appconfig.DefaultParserConfig()),
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

		cmd := exec.CommandContext(ctx, // #nosec G204 -- VLM validation intentionally invokes pdftoppm with explicit argv.
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
