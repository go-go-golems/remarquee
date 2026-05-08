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
	"github.com/juruen/rmapi/api"
	"github.com/juruen/rmapi/model"
	"github.com/juruen/rmapi/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type uploadSourceSettings struct {
	NonInteractive bool
	Reauth         bool

	Force     bool
	DryRun    bool
	PDFOnly   bool
	OutputDir string

	// Destination.
	Date      string
	RemoteDir string

	// Bundle mode.
	Bundle   bool
	Name     string
	TOCDepth int

	// Source rendering.
	Theme      string
	Listings   bool
	TitleMode  string // name|path
	IncludeExt []string

	// Pandoc/xelatex.
	Pandoc          string
	PDFEngine       string
	MainFont        string
	MonoFont        string
	Geometry        string
	LatexHeaderFile string
}

type sourceInput struct {
	AbsPath string
	RelPath string
}

func (in sourceInput) Title(titleMode string) string {
	switch strings.ToLower(strings.TrimSpace(titleMode)) {
	case "path":
		return in.RelPath
	case "name", "":
		return filepath.Base(in.AbsPath)
	default:
		return filepath.Base(in.AbsPath)
	}
}

func NewUploadSourceCommand() *cobra.Command {
	s := &uploadSourceSettings{}

	cmd := &cobra.Command{
		Use:   "src <path...>",
		Short: "Render source files as syntax-highlighted PDFs and upload to reMarkable",
		Long: strings.TrimSpace(`
Render source code files as PDFs (pandoc + xelatex) with syntax highlighting, then upload them to a reMarkable device via rmapi.

Inputs:
- You may pass files and/or directories.
- Directories are recursively scanned for files.

Destination:
- Default remote directory: /ai/YYYY/MM/DD/ (today's date)
- Override with --date (changes /ai/YYYY/MM/DD/) or --remote-dir (full override)
`),
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			return runUploadSource(ctx, cmd, s, args)
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

	// Bundle mode.
	cmd.Flags().BoolVar(&s.Bundle, "bundle", false, "Bundle multiple source inputs into a single PDF (with ToC) instead of one PDF per file")
	cmd.Flags().StringVar(&s.Name, "name", "", "Bundle document name (only valid with --bundle)")
	cmd.Flags().IntVar(&s.TOCDepth, "toc-depth", 1, "Table of contents depth for bundle mode (pandoc --toc-depth)")

	// Source rendering.
	cmd.Flags().StringVar(&s.Theme, "theme", "tango", "Pandoc highlight style (pandoc --highlight-style)")
	cmd.Flags().BoolVar(&s.Listings, "listings", false, "Use LaTeX listings for code blocks (pandoc --listings)")
	cmd.Flags().StringVar(&s.TitleMode, "title-mode", "path", "Title mode: name|path (default: path)")
	cmd.Flags().StringSliceVar(&s.IncludeExt, "include-ext", nil, "Only include files with these extensions (repeatable, e.g. --include-ext .go). Empty means include all (except .md)")

	// Pandoc/xelatex.
	cmd.Flags().StringVar(&s.Pandoc, "pandoc", "pandoc", "Pandoc binary to run")
	cmd.Flags().StringVar(&s.PDFEngine, "pdf-engine", "xelatex", "Pandoc PDF engine (default: xelatex)")
	cmd.Flags().StringVar(&s.MainFont, "mainfont", "DejaVu Sans", "Main font for PDF generation")
	cmd.Flags().StringVar(&s.MonoFont, "monofont", "DejaVu Sans Mono", "Monospace font for code blocks")
	cmd.Flags().StringVar(&s.Geometry, "geometry", "margin=1in", "LaTeX geometry setting passed to pandoc (default: margin=1in)")
	cmd.Flags().StringVar(&s.LatexHeaderFile, "latex-header-file", "", "Optional path to a LaTeX header file to include (overrides built-in header)")

	return cmd
}

func runUploadSource(ctx context.Context, cmd *cobra.Command, s *uploadSourceSettings, args []string) error {
	if !s.Bundle && strings.TrimSpace(s.Name) != "" {
		return errors.New("--name is only valid with --bundle")
	}

	inputs, err := collectSourceInputs(args, s.IncludeExt, s.Bundle)
	if err != nil {
		return err
	}
	if len(inputs) == 0 {
		return errors.New("no source files found")
	}

	remoteDir, err := resolveRemoteDir(s.Date, s.RemoteDir)
	if err != nil {
		return err
	}

	if !s.Bundle {
		// Detect collisions early: two inputs producing the same document name (one-PDF-per-file mode).
		seenDocNames := map[string]string{}
		for _, in := range inputs {
			name := strings.TrimSuffix(filepath.Base(in.AbsPath), filepath.Ext(in.AbsPath))
			if other, ok := seenDocNames[name]; ok {
				return errors.Errorf("duplicate document name %q from %q and %q (rename one file or upload to different remote directories)", name, other, in.AbsPath)
			}
			seenDocNames[name] = in.AbsPath
		}
	}

	pandocOpts := mdpdf.DefaultPandocOptions()
	pandocOpts.PandocPath = s.Pandoc
	pandocOpts.PDFEngine = s.PDFEngine
	pandocOpts.MainFont = s.MainFont
	pandocOpts.MonoFont = s.MonoFont
	pandocOpts.Geometry = s.Geometry
	pandocOpts.LatexHeaderFile = s.LatexHeaderFile
	pandocOpts.HighlightStyle = strings.TrimSpace(s.Theme)
	pandocOpts.Listings = s.Listings

	if s.Bundle {
		pandocOpts.TOC = true
		pandocOpts.TOCDepth = s.TOCDepth
	}

	if s.DryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "DRY: remote-dir=%s\n", remoteDir)
		if s.Bundle {
			bundleName := strings.TrimSpace(s.Name)
			if bundleName == "" {
				bundleName = "src-bundle"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "DRY: bundle name=%s\n", bundleName)
			for _, in := range inputs {
				src := in.AbsPath
				title := in.Title(s.TitleMode)
				lang := languageForPath(src)
				fmt.Fprintf(cmd.OutOrStdout(), "DRY: include %s (lang=%q, title=%q)\n", src, lang, title)
			}
			pdfName := ensurePDFSuffix(bundleName)
			if s.PDFOnly {
				outDir := s.OutputDir
				if outDir == "" {
					outDir = "."
				}
				fmt.Fprintf(cmd.OutOrStdout(), "DRY: render bundle -> %s\n", filepath.Join(outDir, pdfName))
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "DRY: render bundle -> <tmp>/%s\n", pdfName)
				fmt.Fprintf(cmd.OutOrStdout(), "DRY: upload %s -> %s\n", pdfName, remoteDir)
			}
			return nil
		}

		for _, in := range inputs {
			src := in.AbsPath
			pdfName := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src)) + ".pdf"
			title := in.Title(s.TitleMode)
			lang := languageForPath(src)
			if s.PDFOnly {
				outDir := s.OutputDir
				if outDir == "" {
					outDir = "."
				}
				fmt.Fprintf(cmd.OutOrStdout(), "DRY: render %s (lang=%q, title=%q) -> %s\n", src, lang, title, filepath.Join(outDir, pdfName))
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "DRY: render %s (lang=%q, title=%q) -> <tmp>/%s\n", src, lang, title, pdfName)
				fmt.Fprintf(cmd.OutOrStdout(), "DRY: upload %s -> %s\n", pdfName, remoteDir)
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

		if s.Bundle {
			bundleName := strings.TrimSpace(s.Name)
			if bundleName == "" {
				bundleName = "src-bundle"
			}
			outPDF := filepath.Join(outDir, ensurePDFSuffix(bundleName))
			if err := convertSourceBundleToPDF(ctx, inputs, s.TitleMode, outPDF, pandocOpts); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "OK: generated %s\n", outPDF)
			return nil
		}

		for _, in := range inputs {
			src := in.AbsPath
			pdfName := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src)) + ".pdf"
			outPDF := filepath.Join(outDir, pdfName)
			if err := convertSourceFileToPDF(ctx, src, in.Title(s.TitleMode), outPDF, pandocOpts); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "OK: generated %s\n", outPDF)
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

	tmpDir, err := os.MkdirTemp("", "remarquee-upload-src-")
	if err != nil {
		return errors.Wrap(err, "failed to create temp directory")
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	if s.Bundle {
		bundleName := strings.TrimSpace(s.Name)
		if bundleName == "" {
			bundleName = "src-bundle"
		}
		pdfName := ensurePDFSuffix(bundleName)
		outPDF := filepath.Join(tmpDir, pdfName)

		if err := convertSourceBundleToPDF(ctx, inputs, s.TitleMode, outPDF, pandocOpts); err != nil {
			return err
		}

		docName, _ := util.DocPathToName(outPDF)

		// Upload with auto-reauth on 401/403.
		_, err = rmcloud.WithAuthRetry(authSettings, apiCtx, func(currentCtx api.ApiCtx) (api.ApiCtx, error) {
			dstNode, mkdirErr := rmcloud.MkdirAll(currentCtx, remoteDir)
			if mkdirErr != nil {
				return currentCtx, mkdirErr
			}

			existingNode, err := currentCtx.Filetree().NodeByPath(docName, dstNode)
			if err == nil {
				if !s.Force {
					fmt.Fprintf(cmd.OutOrStdout(), "SKIP: %s already exists in %s (use --force to overwrite)\n", docName, remoteDir)
					return currentCtx, nil
				}

				if existingNode.IsDirectory() {
					return currentCtx, errors.Errorf("cannot overwrite directory %q in %s", docName, remoteDir)
				}

				if err := currentCtx.DeleteEntry(existingNode, false, false); err != nil {
					return currentCtx, errors.Wrap(err, "failed to delete existing file")
				}
				currentCtx.Filetree().DeleteNode(existingNode)
			}

			document, err := currentCtx.UploadDocument(dstNode.Id(), outPDF, true, nil)
			if err != nil {
				return currentCtx, errors.Wrapf(err, "failed to upload file [%s]", outPDF)
			}
			currentCtx.Filetree().AddDocument(document)
			fmt.Fprintf(cmd.OutOrStdout(), "OK: uploaded %s -> %s\n", pdfName, remoteDir)

			return currentCtx, nil
		})
		return err
	}

	dstNodeCache := map[string]*model.Node{}

	for _, in := range inputs {
		src := in.AbsPath
		pdfName := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src)) + ".pdf"
		outPDF := filepath.Join(tmpDir, pdfName)

		if err := convertSourceFileToPDF(ctx, src, in.Title(s.TitleMode), outPDF, pandocOpts); err != nil {
			return err
		}

		docName, _ := util.DocPathToName(outPDF)

		// Upload with auto-reauth on 401/403.
		apiCtx, err = rmcloud.WithAuthRetry(authSettings, apiCtx, func(currentCtx api.ApiCtx) (api.ApiCtx, error) {
			dstNode, ok := dstNodeCache[remoteDir]
			if !ok {
				node, mkdirErr := rmcloud.MkdirAll(currentCtx, remoteDir)
				if mkdirErr != nil {
					return currentCtx, mkdirErr
				}
				dstNode = node
				dstNodeCache[remoteDir] = node
			}

			existingNode, err := currentCtx.Filetree().NodeByPath(docName, dstNode)
			if err == nil {
				if !s.Force {
					fmt.Fprintf(cmd.OutOrStdout(), "SKIP: %s already exists in %s (use --force to overwrite)\n", docName, remoteDir)
					return currentCtx, nil
				}

				if existingNode.IsDirectory() {
					return currentCtx, errors.Errorf("cannot overwrite directory %q in %s", docName, remoteDir)
				}

				if err := currentCtx.DeleteEntry(existingNode, false, false); err != nil {
					return currentCtx, errors.Wrap(err, "failed to delete existing file")
				}
				currentCtx.Filetree().DeleteNode(existingNode)
			}

			document, err := currentCtx.UploadDocument(dstNode.Id(), outPDF, true, nil)
			if err != nil {
				return currentCtx, errors.Wrapf(err, "failed to upload file [%s]", outPDF)
			}
			currentCtx.Filetree().AddDocument(document)
			fmt.Fprintf(cmd.OutOrStdout(), "OK: uploaded %s -> %s\n", pdfName, remoteDir)

			return currentCtx, nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func collectSourceInputs(paths []string, includeExt []string, deterministic bool) ([]sourceInput, error) {
	seen := map[string]struct{}{}
	var out []sourceInput

	allowed := normalizeExtList(includeExt)

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
			var dirFiles []sourceInput
			err := filepath.WalkDir(abs, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				// Skip obvious markdown inputs; those belong to `upload md` / `upload bundle`.
				if strings.EqualFold(filepath.Ext(d.Name()), ".md") {
					return nil
				}
				if len(allowed) > 0 && !extAllowed(path, allowed) {
					return nil
				}

				if _, ok := seen[path]; ok {
					return nil
				}

				rel, err := filepath.Rel(abs, path)
				if err != nil {
					return errors.Wrap(err, "failed to compute relative path")
				}

				dirFiles = append(dirFiles, sourceInput{AbsPath: path, RelPath: rel})
				return nil
			})
			if err != nil {
				return nil, errors.Wrapf(err, "failed to walk directory: %s", abs)
			}

			if deterministic {
				sort.Slice(dirFiles, func(i, j int) bool {
					return strings.ToLower(dirFiles[i].RelPath) < strings.ToLower(dirFiles[j].RelPath)
				})
			}

			for _, f := range dirFiles {
				if _, ok := seen[f.AbsPath]; ok {
					continue
				}
				seen[f.AbsPath] = struct{}{}
				out = append(out, f)
			}
			continue
		}

		if strings.EqualFold(filepath.Ext(abs), ".md") {
			continue
		}
		if len(allowed) > 0 && !extAllowed(abs, allowed) {
			continue
		}

		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}
		out = append(out, sourceInput{AbsPath: abs, RelPath: filepath.Base(abs)})
	}

	if !deterministic {
		sort.Slice(out, func(i, j int) bool {
			return strings.ToLower(out[i].AbsPath) < strings.ToLower(out[j].AbsPath)
		})
	}
	return out, nil
}

func convertSourceFileToPDF(ctx context.Context, srcPath string, title string, outPDF string, pandocOpts mdpdf.PandocOptions) error {
	b, err := buildSourceMarkdown(srcPath, title)
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "remarquee-src-md-")
	if err != nil {
		return errors.Wrap(err, "failed to create temp directory")
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	mdPath := filepath.Join(tmpDir, filepath.Base(srcPath)+".md")
	if err := os.WriteFile(mdPath, []byte(b), 0o644); err != nil {
		return errors.Wrap(err, "failed to write source markdown wrapper")
	}

	return mdpdf.ConvertMarkdownFileToPDF(ctx, mdPath, outPDF, pandocOpts)
}

func convertSourceBundleToPDF(ctx context.Context, inputs []sourceInput, titleMode string, outPDF string, pandocOpts mdpdf.PandocOptions) error {
	body, err := buildSourceBundleMarkdown(inputs, titleMode)
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "remarquee-src-bundle-md-")
	if err != nil {
		return errors.Wrap(err, "failed to create temp directory")
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	mdPath := filepath.Join(tmpDir, "src-bundle.md")
	if err := os.WriteFile(mdPath, []byte(body), 0o644); err != nil {
		return errors.Wrap(err, "failed to write source bundle markdown")
	}

	return mdpdf.ConvertMarkdownFileToPDF(ctx, mdPath, outPDF, pandocOpts)
}

func buildSourceBundleMarkdown(inputs []sourceInput, titleMode string) (string, error) {
	var b strings.Builder

	for i, in := range inputs {
		srcBytes, err := os.ReadFile(in.AbsPath)
		if err != nil {
			return "", errors.Wrapf(err, "failed to read source file: %s", in.AbsPath)
		}
		src := string(srcBytes)

		title := in.Title(titleMode)
		lang := languageForPath(in.AbsPath)
		fence := fenceForCode(src)

		fmt.Fprintf(&b, "# %s\n\n", title)
		if lang != "" {
			b.WriteString(fence + lang + "\n")
		} else {
			b.WriteString(fence + "\n")
		}
		b.WriteString(src)
		if !strings.HasSuffix(src, "\n") {
			b.WriteString("\n")
		}
		b.WriteString(fence + "\n\n")

		if i < len(inputs)-1 {
			b.WriteString("```{=latex}\n\\newpage\n```\n\n")
		}
	}

	return b.String(), nil
}

func buildSourceMarkdown(srcPath string, title string) (string, error) {
	srcBytes, err := os.ReadFile(srcPath)
	if err != nil {
		return "", errors.Wrap(err, "failed to read source file")
	}
	src := string(srcBytes)

	lang := languageForPath(srcPath)
	fence := fenceForCode(src)

	var b strings.Builder
	if strings.TrimSpace(title) == "" {
		title = filepath.Base(srcPath)
	}
	fmt.Fprintf(&b, "# %s\n\n", title)
	if lang != "" {
		b.WriteString(fence + lang + "\n")
	} else {
		b.WriteString(fence + "\n")
	}
	b.WriteString(src)
	if !strings.HasSuffix(src, "\n") {
		b.WriteString("\n")
	}
	b.WriteString(fence + "\n")
	return b.String(), nil
}

func fenceForCode(code string) string {
	maxRun := 0
	run := 0
	for i := 0; i < len(code); i++ {
		if code[i] == '`' {
			run++
			if run > maxRun {
				maxRun = run
			}
		} else {
			run = 0
		}
	}
	n := 3
	if maxRun+1 > n {
		n = maxRun + 1
	}
	return strings.Repeat("`", n)
}

func languageForPath(p string) string {
	ext := strings.ToLower(filepath.Ext(p))
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".tsx":
		return "typescript"
	case ".c", ".h":
		return "c"
	case ".cc", ".cpp", ".cxx", ".hpp", ".hh":
		return "cpp"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".sh", ".bash", ".zsh":
		return "bash"
	case ".json":
		return "json"
	case ".yml", ".yaml":
		return "yaml"
	case ".toml":
		return "toml"
	case ".sql":
		return "sql"
	case ".html":
		return "html"
	case ".css":
		return "css"
	case ".xml":
		return "xml"
	default:
		return ""
	}
}

func normalizeExtList(exts []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, e := range exts {
		ee := strings.TrimSpace(strings.ToLower(e))
		if ee == "" {
			continue
		}
		if !strings.HasPrefix(ee, ".") {
			ee = "." + ee
		}
		out[ee] = struct{}{}
	}
	return out
}

func extAllowed(path string, allowed map[string]struct{}) bool {
	ext := strings.ToLower(filepath.Ext(path))
	_, ok := allowed[ext]
	return ok
}
