package upload

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-go-golems/remarquee/pkg/mdpdf"
	"github.com/go-go-golems/remarquee/pkg/rmcloud"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type uploadBundleSettings struct {
	NonInteractive bool
	Reauth         bool

	Force     bool
	DryRun    bool
	PDFOnly   bool
	OutputDir string

	// Destination.
	Date      string
	RemoteDir string

	// Bundle output.
	Name     string
	TOCDepth int

	// Pandoc/xelatex.
	Pandoc          string
	PDFEngine       string
	MainFont        string
	MonoFont        string
	Layout          string
	Geometry        string
	LatexHeaderFile string

	// Image flags.
	ResolveImages bool
}

type bundleMarkdownFile struct {
	AbsPath string
	Title   string
}

func NewUploadBundleCommand() *cobra.Command {
	s := &uploadBundleSettings{}

	cmd := &cobra.Command{
		Use:   "bundle <path...>",
		Short: "Bundle multiple markdown inputs into a single PDF (with ToC) and upload to reMarkable",
		Long: strings.TrimSpace(`
Bundle multiple markdown inputs into a single PDF with a clickable Table of Contents (ToC), then upload to a reMarkable device via rmapi.

Inputs:
- You may pass markdown files (*.md) and/or directories.
- Directories are recursively scanned for *.md files.

Ordering:
- Files passed explicitly preserve the user-given order.
- Directory contents are ordered lexicographically by relative path (case-insensitive).

Destination:
- Default remote directory: /ai/YYYY/MM/DD/ (today's date)
- Override with --date (changes /ai/YYYY/MM/DD/) or --remote-dir (full override)

Safety:
- Existing documents are skipped unless --force is provided.
`),
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			return runUploadBundle(ctx, cmd, s, args)
		},
	}

	// Auth flags (match cloud commands).
	cmd.Flags().BoolVar(&s.NonInteractive, "non-interactive", false, "Do not prompt for one-time code; fail if tokens are missing")
	cmd.Flags().BoolVar(&s.Reauth, "reauth", false, "Force re-authentication (re-fetch user token)")

	// Behavior.
	cmd.Flags().BoolVar(&s.Force, "force", false, "Overwrite existing documents (WARNING: deletes existing document + annotations)")
	cmd.Flags().BoolVar(&s.DryRun, "dry-run", false, "Print what would be done, but do not run pandoc or upload")
	cmd.Flags().BoolVar(&s.PDFOnly, "pdf-only", false, "Only generate PDFs, do not upload")
	cmd.Flags().StringVar(&s.OutputDir, "output-dir", "", "Output directory for PDFs in --pdf-only mode (default: current directory)")

	// Destination.
	cmd.Flags().StringVar(&s.Date, "date", "", "Destination date folder under /ai (YYYY/MM/DD or YYYY-MM-DD). Default: today")
	cmd.Flags().StringVar(&s.RemoteDir, "remote-dir", "", "Override remote directory (default: /ai/YYYY/MM/DD/)")

	// Bundle output options.
	cmd.Flags().StringVar(&s.Name, "name", "", "Output document name (default: <dirname> for a single directory input, else bundle)")
	cmd.Flags().IntVar(&s.TOCDepth, "toc-depth", 1, "Table of contents depth (pandoc --toc-depth)")

	// Pandoc/xelatex.
	cmd.Flags().StringVar(&s.Pandoc, "pandoc", "pandoc", "Pandoc binary to run")
	cmd.Flags().StringVar(&s.PDFEngine, "pdf-engine", "xelatex", "Pandoc PDF engine (default: xelatex)")
	cmd.Flags().StringVar(&s.MainFont, "mainfont", "DejaVu Sans", "Main font for PDF generation")
	cmd.Flags().StringVar(&s.MonoFont, "monofont", "DejaVu Sans Mono", "Monospace font for code blocks")
	cmd.Flags().StringVar(&s.Layout, "layout", mdpdf.MarkdownLayoutDefault, "Markdown layout preset: default|editor (editor adds wider margins and more annotation-friendly spacing)")
	cmd.Flags().StringVar(&s.Geometry, "geometry", "margin=1in", "LaTeX geometry setting passed to pandoc (default: margin=1in)")
	cmd.Flags().StringVar(&s.LatexHeaderFile, "latex-header-file", "", "Optional path to a LaTeX header file to include (overrides built-in header)")

	// Mermaid flags (Glazed section — shows in "Mermaid flags" help group).
	if err := addMermaidFlagsToCommand(cmd); err != nil {
		panic(err)
	}

	// Image flags.
	addResolveImagesFlag(cmd, &s.ResolveImages)

	return cmd
}

func runUploadBundle(ctx context.Context, cmd *cobra.Command, s *uploadBundleSettings, args []string) error {
	files, err := collectMarkdownFilesForBundle(args)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return errors.New("no markdown files found")
	}

	name, err := defaultBundleName(args, s.Name)
	if err != nil {
		return err
	}
	pdfFileName := ensurePDFSuffix(name)

	remoteDir, err := resolveRemoteDir(s.Date, s.RemoteDir)
	if err != nil {
		return err
	}

	mermaidCfg, err := mermaidConfigFromCommand(cmd)
	if err != nil {
		return err
	}

	pandocOpts, err := configureMarkdownPandocOptions(
		cmd.Flags(),
		s.Layout,
		s.Pandoc,
		s.PDFEngine,
		s.MainFont,
		s.MonoFont,
		s.Geometry,
		s.LatexHeaderFile,
		mermaidCfg,
	)
	if err != nil {
		return err
	}
	pandocOpts.ResolveImages = s.ResolveImages
	pandocOpts.TOC = true
	pandocOpts.TOCDepth = s.TOCDepth

	if s.DryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "DRY: layout=%s\n", mdpdf.NormalizeMarkdownLayout(s.Layout))
		fmt.Fprintf(cmd.OutOrStdout(), "DRY: bundle name=%s\n", name)
		fmt.Fprintf(cmd.OutOrStdout(), "DRY: remote-dir=%s\n", remoteDir)
		for _, f := range files {
			fmt.Fprintf(cmd.OutOrStdout(), "DRY: include %s (title=%q)\n", f.AbsPath, f.Title)
		}
		if s.PDFOnly {
			outDir := s.OutputDir
			if outDir == "" {
				outDir = "."
			}
			fmt.Fprintf(cmd.OutOrStdout(), "DRY: pandoc <bundle> -> %s\n", filepath.Join(outDir, pdfFileName))
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "DRY: pandoc <bundle> -> <tmp>/%s\n", pdfFileName)
			fmt.Fprintf(cmd.OutOrStdout(), "DRY: upload %s -> %s\n", pdfFileName, remoteDir)
		}
		return nil
	}

	// Ensure output dir for pdf-only.
	if s.PDFOnly {
		outDir := s.OutputDir
		if outDir == "" {
			outDir = "."
		}
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return errors.Wrap(err, "failed to create output directory")
		}

		outPDF := filepath.Join(outDir, pdfFileName)
		if err := writeBundlePDF(ctx, files, outPDF, pandocOpts); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "OK: generated %s\n", outPDF)
		return nil
	}

	// Upload mode.
	authSettings := rmcloud.AuthSettings{
		NonInteractive: s.NonInteractive,
		Reauth:         s.Reauth,
	}
	_, apiCtx, err := rmcloud.CreateApiCtx(authSettings)
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "remarquee-upload-bundle-")
	if err != nil {
		return errors.Wrap(err, "failed to create temp directory")
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	outPDF := filepath.Join(tmpDir, pdfFileName)
	if err := writeBundlePDF(ctx, files, outPDF, pandocOpts); err != nil {
		return err
	}

	_, err = uploadPDFToRemoteWithAuthRetry(cmd, authSettings, apiCtx, remoteDir, outPDF, pdfFileName, s.Force)
	return err
}

func writeBundlePDF(ctx context.Context, files []bundleMarkdownFile, outPDF string, pandocOpts mdpdf.PandocOptions) error {
	inputs := make([]mdpdf.BundleInput, 0, len(files))
	for _, f := range files {
		inputs = append(inputs, mdpdf.BundleInput{
			Path:  f.AbsPath,
			Title: f.Title,
		})
	}

	// Create the temp directory early so that BuildBundleMarkdown can
	// place resolved images into tmpDir/images/ before concatenation.
	tmpDir, err := os.MkdirTemp("", "remarquee-bundle-md-")
	if err != nil {
		return errors.Wrap(err, "failed to create temp directory")
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	body, err := mdpdf.BuildBundleMarkdown(ctx, inputs, tmpDir, pandocOpts.Mermaid, pandocOpts.ResolveImages)
	if err != nil {
		return err
	}

	bundleMD := filepath.Join(tmpDir, "bundle.md")
	if err := os.WriteFile(bundleMD, []byte(body), 0o644); err != nil {
		return errors.Wrap(err, "failed to write bundle markdown")
	}

	return mdpdf.ConvertMarkdownFileToPDF(ctx, bundleMD, outPDF, pandocOpts)
}

func ensurePDFSuffix(name string) string {
	name = strings.TrimSpace(name)
	if strings.HasSuffix(strings.ToLower(name), ".pdf") {
		return name
	}
	return name + ".pdf"
}

func defaultBundleName(args []string, override string) (string, error) {
	if strings.TrimSpace(override) != "" {
		return strings.TrimSpace(override), nil
	}

	if len(args) == 1 {
		abs, err := filepath.Abs(strings.TrimSpace(args[0]))
		if err != nil {
			return "", errors.Wrap(err, "failed to resolve input path")
		}
		info, err := os.Stat(abs)
		if err != nil {
			return "", errors.Wrap(err, "failed to stat input path")
		}
		if info.IsDir() {
			return filepath.Base(abs), nil
		}
	}

	return "bundle", nil
}

func collectMarkdownFilesForBundle(paths []string) ([]bundleMarkdownFile, error) {
	seen := map[string]struct{}{}
	var out []bundleMarkdownFile

	for _, p := range paths {
		pp := strings.TrimSpace(p)
		if pp == "" {
			continue
		}
		abs, err := filepath.Abs(pp)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to resolve path: %s", p)
		}

		info, err := os.Stat(abs)
		if err != nil {
			return nil, errors.Wrapf(err, "path not found: %s", abs)
		}

		if info.IsDir() {
			var dirFiles []bundleMarkdownFile

			err := filepath.WalkDir(abs, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				if !strings.EqualFold(filepath.Ext(d.Name()), ".md") {
					return nil
				}
				rel, err := filepath.Rel(abs, path)
				if err != nil {
					return errors.Wrap(err, "failed to compute relative path")
				}
				title := strings.TrimSuffix(rel, filepath.Ext(rel))
				dirFiles = append(dirFiles, bundleMarkdownFile{
					AbsPath: path,
					Title:   title,
				})
				return nil
			})
			if err != nil {
				return nil, errors.Wrapf(err, "failed to walk directory: %s", abs)
			}

			sort.Slice(dirFiles, func(i, j int) bool {
				return strings.ToLower(dirFiles[i].Title) < strings.ToLower(dirFiles[j].Title)
			})

			for _, f := range dirFiles {
				if _, ok := seen[f.AbsPath]; ok {
					continue
				}
				seen[f.AbsPath] = struct{}{}
				out = append(out, f)
			}
			continue
		}

		if !strings.EqualFold(filepath.Ext(abs), ".md") {
			return nil, errors.Errorf("unsupported file type (expected .md): %s", abs)
		}

		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}

		out = append(out, bundleMarkdownFile{
			AbsPath: abs,
			Title:   strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs)),
		})
	}

	return out, nil
}
