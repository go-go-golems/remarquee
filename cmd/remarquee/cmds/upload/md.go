package upload

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-go-golems/remarquee/pkg/mdpdf"
	"github.com/go-go-golems/remarquee/pkg/rmcloud"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type uploadMarkdownSettings struct {
	NonInteractive bool
	Reauth         bool

	Force           bool
	DryRun          bool
	PDFOnly         bool
	OutputDir       string
	PreserveDirs    bool
	Flatten         bool
	Name            string
	Date            string
	RemoteDir       string
	Pandoc          string
	PDFEngine       string
	MainFont        string
	MonoFont        string
	Layout          string
	Geometry        string
	LatexHeaderFile string
	Workers         int
}

func NewUploadMarkdownCommand() *cobra.Command {
	s := &uploadMarkdownSettings{}

	cmd := &cobra.Command{
		Use:   "md <path...>",
		Short: "Convert markdown to PDF (pandoc/xelatex) and upload to reMarkable",
		Long: strings.TrimSpace(`
Convert markdown files to PDF (pandoc + xelatex, DejaVu fonts) and upload them to a reMarkable device via rmapi.

Inputs:
- You may pass markdown files (*.md) and/or directories.
- Directories are recursively scanned for *.md files.
- Directory structure is mirrored remotely by default. Use --flatten to upload all files to a single directory.

Destination:
- Default remote directory: /ai/YYYY/MM/DD/ (today's date)
- Override with --date (changes /ai/YYYY/MM/DD/) or --remote-dir (full override)

Safety:
- Existing documents are skipped unless --force is provided.
`),
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			return runUploadMarkdown(ctx, cmd, s, args)
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
	cmd.Flags().BoolVar(&s.PreserveDirs, "preserve-dirs", true, "Recreate the local relative directory structure remotely (default: true)")
	cmd.Flags().BoolVar(&s.Flatten, "flatten", false, "Upload all files to a single flat directory (overrides --preserve-dirs)")
	cmd.Flags().StringVar(&s.Name, "name", "", "Custom output document name (only valid when exactly one markdown file is selected)")
	cmd.Flags().IntVar(&s.Workers, "workers", 1, "Number of parallel pandoc conversions to run before upload (default: 1)")

	// Destination.
	cmd.Flags().StringVar(&s.Date, "date", "", "Destination date folder under /ai (YYYY/MM/DD or YYYY-MM-DD). Default: today")
	cmd.Flags().StringVar(&s.RemoteDir, "remote-dir", "", "Override remote directory (default: /ai/YYYY/MM/DD/)")

	// Pandoc/xelatex.
	cmd.Flags().StringVar(&s.Pandoc, "pandoc", "pandoc", "Pandoc binary to run")
	cmd.Flags().StringVar(&s.PDFEngine, "pdf-engine", "xelatex", "Pandoc PDF engine (default: xelatex)")
	cmd.Flags().StringVar(&s.MainFont, "mainfont", "DejaVu Sans", "Main font for PDF generation")
	cmd.Flags().StringVar(&s.MonoFont, "monofont", "DejaVu Sans Mono", "Monospace font for code blocks")
	cmd.Flags().StringVar(&s.Layout, "layout", mdpdf.MarkdownLayoutDefault, "Markdown layout preset: default|editor (editor adds wider margins and more annotation-friendly spacing)")
	cmd.Flags().StringVar(&s.Geometry, "geometry", "margin=1in", "LaTeX geometry setting passed to pandoc (default: margin=1in)")
	cmd.Flags().StringVar(&s.LatexHeaderFile, "latex-header-file", "", "Optional path to a LaTeX header file to include (overrides built-in header)")

	return cmd
}

func runUploadMarkdown(ctx context.Context, cmd *cobra.Command, s *uploadMarkdownSettings, args []string) error {
	// --flatten is a convenience inverse of --preserve-dirs.
	if s.Flatten {
		s.PreserveDirs = false
	}
	if s.Workers < 1 {
		return errors.New("--workers must be at least 1")
	}

	mdInputs, err := collectMarkdownInputs(args)
	if err != nil {
		return err
	}
	if len(mdInputs) == 0 {
		return errors.New("no markdown files found")
	}

	remoteDir, err := resolveRemoteDir(s.Date, s.RemoteDir)
	if err != nil {
		return err
	}

	// Detect collisions early:
	// - flat mode: docName collisions are always ambiguous
	// - preserve-dirs mode: detect collisions at the computed remote location
	seenRemoteKeys := map[string]string{}
	for _, in := range mdInputs {
		pdfName, err := markdownPDFName(in, s.Name, len(mdInputs))
		if err != nil {
			return err
		}
		docName := strings.TrimSuffix(pdfName, filepath.Ext(pdfName))
		relDir := ""
		if s.PreserveDirs {
			relDir = in.RelDir()
		}
		key := remoteDocKey(remoteDir, relDir, docName)
		if other, ok := seenRemoteKeys[key]; ok {
			return errors.Errorf("duplicate document %q from %q and %q (rename one file or upload to different remote directories)", docName, other, in.AbsPath)
		}
		seenRemoteKeys[key] = in.AbsPath
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
	)
	if err != nil {
		return err
	}

	if s.DryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "DRY: layout=%s\n", mdpdf.NormalizeMarkdownLayout(s.Layout))
		fmt.Fprintf(cmd.OutOrStdout(), "DRY: remote-dir=%s\n", remoteDir)
		fmt.Fprintf(cmd.OutOrStdout(), "DRY: workers=%d\n", s.Workers)
		for _, in := range mdInputs {
			mdPath := in.AbsPath
			pdfName, err := markdownPDFName(in, s.Name, len(mdInputs))
			if err != nil {
				return err
			}
			if s.PDFOnly {
				outDir := s.OutputDir
				if outDir == "" {
					outDir = "."
				}
				outPDF := filepath.Join(outDir, pdfName)
				if s.PreserveDirs {
					outPDF = filepath.Join(outDir, in.RelDir(), pdfName)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "DRY: pandoc %s -> %s\n", mdPath, outPDF)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "DRY: pandoc %s -> <tmp>/%s\n", mdPath, pdfName)
				dst := remoteDir
				if s.PreserveDirs {
					dst = joinRemoteDir(remoteDir, in.RelDir())
				}
				fmt.Fprintf(cmd.OutOrStdout(), "DRY: upload %s -> %s\n", pdfName, dst)
			}
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

		jobs, err := buildMarkdownConversionJobs(mdInputs, outDir, s.Name, s.PreserveDirs)
		if err != nil {
			return err
		}
		if err := convertMarkdownJobs(ctx, jobs, s.Workers, pandocOpts); err != nil {
			return err
		}
		for _, job := range jobs {
			fmt.Fprintf(cmd.OutOrStdout(), "OK: generated %s\n", job.OutPDF)
		}
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

	tmpDir, err := os.MkdirTemp("", "remarquee-upload-md-")
	if err != nil {
		return errors.Wrap(err, "failed to create temp directory")
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	jobs, err := buildMarkdownConversionJobs(mdInputs, tmpDir, s.Name, s.PreserveDirs)
	if err != nil {
		return err
	}
	if s.Workers > 1 {
		if err := convertMarkdownJobs(ctx, jobs, s.Workers, pandocOpts); err != nil {
			return err
		}
	}

	for _, job := range jobs {
		if s.Workers == 1 {
			if err := mdpdf.ConvertMarkdownFileToPDF(ctx, job.Input.AbsPath, job.OutPDF, pandocOpts); err != nil {
				return err
			}
		}

		dst := remoteDir
		if s.PreserveDirs {
			dst = joinRemoteDir(remoteDir, job.Input.RelDir())
		}

		apiCtx, err = uploadPDFToRemoteWithAuthRetry(cmd, authSettings, apiCtx, dst, job.OutPDF, job.PDFName, s.Force)
		if err != nil {
			return err
		}
	}

	return nil
}

func markdownPDFName(in markdownInput, override string, total int) (string, error) {
	override = strings.TrimSpace(override)
	if override != "" {
		if total != 1 {
			return "", errors.New("--name is only valid when exactly one markdown file is selected")
		}
		if strings.ContainsAny(override, `/\`) {
			return "", errors.New("--name must be a basename and may not contain path separators")
		}
		return ensurePDFSuffix(override), nil
	}

	return strings.TrimSuffix(filepath.Base(in.AbsPath), filepath.Ext(in.AbsPath)) + ".pdf", nil
}

type markdownInput struct {
	AbsPath string
	RelPath string
}

func (in markdownInput) RelDir() string {
	d := filepath.Dir(in.RelPath)
	if d == "." || d == string(filepath.Separator) {
		return ""
	}
	return d
}

func remoteDocKey(remoteBase string, relDir string, docName string) string {
	dst := joinRemoteDir(remoteBase, relDir)
	return dst + "/" + docName
}

func joinRemoteDir(remoteBase string, relDir string) string {
	relDir = filepath.ToSlash(relDir)
	relDir = strings.Trim(relDir, "/")
	if relDir == "" || relDir == "." {
		return normalizeRemoteDir(remoteBase)
	}
	return normalizeRemoteDir(normalizeRemoteDir(remoteBase) + "/" + relDir)
}

func collectMarkdownInputs(paths []string) ([]markdownInput, error) {
	seen := map[string]struct{}{}
	var out []markdownInput

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
			err := filepath.WalkDir(abs, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				if strings.EqualFold(filepath.Ext(d.Name()), ".md") {
					if _, ok := seen[path]; !ok {
						rel, err := filepath.Rel(abs, path)
						if err != nil {
							return errors.Wrap(err, "failed to compute relative path")
						}
						seen[path] = struct{}{}
						out = append(out, markdownInput{AbsPath: path, RelPath: rel})
					}
				}
				return nil
			})
			if err != nil {
				return nil, errors.Wrapf(err, "failed to walk directory: %s", abs)
			}
			continue
		}

		if !strings.EqualFold(filepath.Ext(abs), ".md") {
			return nil, errors.Errorf("unsupported file type (expected .md): %s", abs)
		}

		if _, ok := seen[abs]; !ok {
			seen[abs] = struct{}{}
			out = append(out, markdownInput{AbsPath: abs, RelPath: filepath.Base(abs)})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].AbsPath) < strings.ToLower(out[j].AbsPath)
	})
	return out, nil
}

func collectMarkdownFiles(paths []string) ([]string, error) {
	seen := map[string]struct{}{}
	var out []string

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
			err := filepath.WalkDir(abs, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				if strings.EqualFold(filepath.Ext(d.Name()), ".md") {
					if _, ok := seen[path]; !ok {
						seen[path] = struct{}{}
						out = append(out, path)
					}
				}
				return nil
			})
			if err != nil {
				return nil, errors.Wrapf(err, "failed to walk directory: %s", abs)
			}
			continue
		}

		if !strings.EqualFold(filepath.Ext(abs), ".md") {
			return nil, errors.Errorf("unsupported file type (expected .md): %s", abs)
		}

		if _, ok := seen[abs]; !ok {
			seen[abs] = struct{}{}
			out = append(out, abs)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i]) < strings.ToLower(out[j])
	})
	return out, nil
}

func resolveRemoteDir(dateArg string, override string) (string, error) {
	if strings.TrimSpace(override) != "" {
		return normalizeRemoteDir(override), nil
	}

	t := time.Now()
	if strings.TrimSpace(dateArg) != "" {
		normalized := strings.TrimSpace(dateArg)
		normalized = strings.Trim(normalized, "/")
		if strings.Contains(normalized, "-") && !strings.Contains(normalized, "/") {
			parts := strings.Split(normalized, "-")
			if len(parts) == 3 {
				normalized = strings.Join(parts, "/")
			}
		}
		parts := strings.Split(normalized, "/")
		if len(parts) != 3 {
			return "", errors.Errorf("invalid date format: %q (expected YYYY/MM/DD or YYYY-MM-DD)", dateArg)
		}
		yyyy, mm, dd := parts[0], parts[1], parts[2]
		if len(yyyy) != 4 || len(mm) != 2 || len(dd) != 2 {
			return "", errors.Errorf("invalid date format: %q (expected YYYY/MM/DD)", dateArg)
		}
		if !allDigits(yyyy) || !allDigits(mm) || !allDigits(dd) {
			return "", errors.Errorf("invalid date format: %q (expected digits)", dateArg)
		}
		return normalizeRemoteDir("/ai/" + yyyy + "/" + mm + "/" + dd + "/"), nil
	}

	return normalizeRemoteDir(fmt.Sprintf("/ai/%04d/%02d/%02d/", t.Year(), int(t.Month()), t.Day())), nil
}

func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return s != ""
}

func normalizeRemoteDir(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	for len(p) > 1 && strings.HasSuffix(p, "/") {
		p = strings.TrimSuffix(p, "/")
	}
	return p
}
