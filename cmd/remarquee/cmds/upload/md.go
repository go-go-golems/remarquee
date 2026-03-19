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
	"github.com/juruen/rmapi/model"
	"github.com/juruen/rmapi/util"
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
	Date            string
	RemoteDir       string
	Pandoc          string
	PDFEngine       string
	MainFont        string
	MonoFont        string
	Geometry        string
	LatexHeaderFile string
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

	// Destination.
	cmd.Flags().StringVar(&s.Date, "date", "", "Destination date folder under /ai (YYYY/MM/DD or YYYY-MM-DD). Default: today")
	cmd.Flags().StringVar(&s.RemoteDir, "remote-dir", "", "Override remote directory (default: /ai/YYYY/MM/DD/)")

	// Pandoc/xelatex.
	cmd.Flags().StringVar(&s.Pandoc, "pandoc", "pandoc", "Pandoc binary to run")
	cmd.Flags().StringVar(&s.PDFEngine, "pdf-engine", "xelatex", "Pandoc PDF engine (default: xelatex)")
	cmd.Flags().StringVar(&s.MainFont, "mainfont", "DejaVu Sans", "Main font for PDF generation")
	cmd.Flags().StringVar(&s.MonoFont, "monofont", "DejaVu Sans Mono", "Monospace font for code blocks")
	cmd.Flags().StringVar(&s.Geometry, "geometry", "margin=1in", "LaTeX geometry setting passed to pandoc (default: margin=1in)")
	cmd.Flags().StringVar(&s.LatexHeaderFile, "latex-header-file", "", "Optional path to a LaTeX header file to include (overrides built-in header)")

	return cmd
}

func runUploadMarkdown(ctx context.Context, cmd *cobra.Command, s *uploadMarkdownSettings, args []string) error {
	// --flatten is a convenience inverse of --preserve-dirs.
	if s.Flatten {
		s.PreserveDirs = false
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
		docName := strings.TrimSuffix(filepath.Base(in.AbsPath), filepath.Ext(in.AbsPath))
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

	pandocOpts := mdpdf.DefaultPandocOptions()
	pandocOpts.PandocPath = s.Pandoc
	pandocOpts.PDFEngine = s.PDFEngine
	pandocOpts.MainFont = s.MainFont
	pandocOpts.MonoFont = s.MonoFont
	pandocOpts.Geometry = s.Geometry
	pandocOpts.LatexHeaderFile = s.LatexHeaderFile

	if s.DryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "DRY: remote-dir=%s\n", remoteDir)
		for _, in := range mdInputs {
			mdPath := in.AbsPath
			pdfName := strings.TrimSuffix(filepath.Base(mdPath), filepath.Ext(mdPath)) + ".pdf"
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

		for _, in := range mdInputs {
			mdPath := in.AbsPath
			pdfName := strings.TrimSuffix(filepath.Base(mdPath), filepath.Ext(mdPath)) + ".pdf"
			outPDF := filepath.Join(outDir, pdfName)
			if s.PreserveDirs {
				outPDF = filepath.Join(outDir, in.RelDir(), pdfName)
				if err := os.MkdirAll(filepath.Dir(outPDF), 0o755); err != nil {
					return errors.Wrap(err, "failed to create output directory")
				}
			}
			if err := mdpdf.ConvertMarkdownFileToPDF(ctx, mdPath, outPDF, pandocOpts); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "OK: generated %s\n", outPDF)
		}
		return nil
	}

	// Upload mode.
	_, apiCtx, err := rmcloud.CreateApiCtx(rmcloud.AuthSettings{
		NonInteractive: s.NonInteractive,
		Reauth:         s.Reauth,
	})
	if err != nil {
		return err
	}

	dstNodeCache := map[string]*model.Node{}

	tmpDir, err := os.MkdirTemp("", "remarquee-upload-md-")
	if err != nil {
		return errors.Wrap(err, "failed to create temp directory")
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	for _, in := range mdInputs {
		mdPath := in.AbsPath
		pdfName := strings.TrimSuffix(filepath.Base(mdPath), filepath.Ext(mdPath)) + ".pdf"
		outPDF := filepath.Join(tmpDir, pdfName)
		if s.PreserveDirs {
			outPDF = filepath.Join(tmpDir, in.RelDir(), pdfName)
			if err := os.MkdirAll(filepath.Dir(outPDF), 0o755); err != nil {
				return errors.Wrap(err, "failed to create temp directory structure")
			}
		}

		if err := mdpdf.ConvertMarkdownFileToPDF(ctx, mdPath, outPDF, pandocOpts); err != nil {
			return err
		}

		docName, _ := util.DocPathToName(outPDF)

		dst := remoteDir
		if s.PreserveDirs {
			dst = joinRemoteDir(remoteDir, in.RelDir())
		}

		// Ensure remote directory exists (cache MkdirAll results).
		dstNode, ok := dstNodeCache[dst]
		if !ok {
			node, err := rmcloud.MkdirAll(apiCtx, dst)
			if err != nil {
				return err
			}
			dstNode = node
			dstNodeCache[dst] = node
		}

		// Existence check.
		existingNode, err := apiCtx.Filetree().NodeByPath(docName, dstNode)
		if err == nil {
			if !s.Force {
				fmt.Fprintf(cmd.OutOrStdout(), "SKIP: %s already exists in %s (use --force to overwrite)\n", docName, dst)
				continue
			}

			if existingNode.IsDirectory() {
				return errors.Errorf("cannot overwrite directory %q in %s", docName, dst)
			}

			if err := apiCtx.DeleteEntry(existingNode, false, false); err != nil {
				return errors.Wrap(err, "failed to delete existing file")
			}
			apiCtx.Filetree().DeleteNode(existingNode)
		}

		document, err := apiCtx.UploadDocument(dstNode.Id(), outPDF, true, nil)
		if err != nil {
			return errors.Wrapf(err, "failed to upload file [%s]", outPDF)
		}
		apiCtx.Filetree().AddDocument(document)
		fmt.Fprintf(cmd.OutOrStdout(), "OK: uploaded %s -> %s\n", pdfName, dst)
	}

	return nil
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
