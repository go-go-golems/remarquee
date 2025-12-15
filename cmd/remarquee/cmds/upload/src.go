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

	// Source rendering.
	Theme     string
	Listings  bool
	TitleMode string // name|path

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

	// Source rendering.
	cmd.Flags().StringVar(&s.Theme, "theme", "tango", "Pandoc highlight style (pandoc --highlight-style)")
	cmd.Flags().BoolVar(&s.Listings, "listings", false, "Use LaTeX listings for code blocks (pandoc --listings)")
	cmd.Flags().StringVar(&s.TitleMode, "title-mode", "path", "Title mode: name|path (default: path)")

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
	inputs, err := collectSourceInputs(args)
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

	// Detect collisions early: two inputs producing the same document name.
	seenDocNames := map[string]string{}
	for _, in := range inputs {
		name := strings.TrimSuffix(filepath.Base(in.AbsPath), filepath.Ext(in.AbsPath))
		if other, ok := seenDocNames[name]; ok {
			return errors.Errorf("duplicate document name %q from %q and %q (rename one file or upload to different remote directories)", name, other, in.AbsPath)
		}
		seenDocNames[name] = in.AbsPath
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

	if s.DryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "DRY: remote-dir=%s\n", remoteDir)
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
	_, apiCtx, err := rmcloud.CreateApiCtx(rmcloud.AuthSettings{
		NonInteractive: s.NonInteractive,
		Reauth:         s.Reauth,
	})
	if err != nil {
		return err
	}

	dstNode, err := rmcloud.MkdirAll(apiCtx, remoteDir)
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "remarquee-upload-src-")
	if err != nil {
		return errors.Wrap(err, "failed to create temp directory")
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	for _, in := range inputs {
		src := in.AbsPath
		pdfName := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src)) + ".pdf"
		outPDF := filepath.Join(tmpDir, pdfName)

		if err := convertSourceFileToPDF(ctx, src, in.Title(s.TitleMode), outPDF, pandocOpts); err != nil {
			return err
		}

		docName, _ := util.DocPathToName(outPDF)

		// Existence check.
		existingNode, err := apiCtx.Filetree().NodeByPath(docName, dstNode)
		if err == nil {
			if !s.Force {
				fmt.Fprintf(cmd.OutOrStdout(), "SKIP: %s already exists in %s (use --force to overwrite)\n", docName, remoteDir)
				continue
			}

			if existingNode.IsDirectory() {
				return errors.Errorf("cannot overwrite directory %q in %s", docName, remoteDir)
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
		fmt.Fprintf(cmd.OutOrStdout(), "OK: uploaded %s -> %s\n", pdfName, remoteDir)
	}

	return nil
}

func collectSourceInputs(paths []string) ([]sourceInput, error) {
	seen := map[string]struct{}{}
	var out []sourceInput

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
				// Skip obvious markdown inputs; those belong to `upload md` / `upload bundle`.
				if strings.EqualFold(filepath.Ext(d.Name()), ".md") {
					return nil
				}

				if _, ok := seen[path]; ok {
					return nil
				}

				rel, err := filepath.Rel(abs, path)
				if err != nil {
					return errors.Wrap(err, "failed to compute relative path")
				}

				seen[path] = struct{}{}
				out = append(out, sourceInput{AbsPath: path, RelPath: rel})
				return nil
			})
			if err != nil {
				return nil, errors.Wrapf(err, "failed to walk directory: %s", abs)
			}
			continue
		}

		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}
		out = append(out, sourceInput{AbsPath: abs, RelPath: filepath.Base(abs)})
	}

	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].AbsPath) < strings.ToLower(out[j].AbsPath)
	})
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
