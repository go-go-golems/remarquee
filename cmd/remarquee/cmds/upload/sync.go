package upload

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-go-golems/remarquee/pkg/mdpdf"
	"github.com/go-go-golems/remarquee/pkg/rmcloud"
	"github.com/juruen/rmapi/api"
	"github.com/juruen/rmapi/filetree"
	"github.com/juruen/rmapi/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type uploadSyncSettings struct {
	NonInteractive bool
	Reauth         bool

	Force         bool
	DryRun        bool
	PreserveDirs  bool
	Flatten       bool
	Name          string
	Date          string
	RemoteDir     string
	CompareMTime  bool
	DeleteOrphans bool

	Pandoc          string
	PDFEngine       string
	MainFont        string
	MonoFont        string
	Layout          string
	Geometry        string
	LatexHeaderFile string
}

func NewUploadSyncCommand() *cobra.Command {
	s := &uploadSyncSettings{}

	cmd := &cobra.Command{
		Use:   "sync <path...>",
		Short: "Plan a Markdown-to-reMarkable sync with pre-flight remote delta detection",
		Long: strings.TrimSpace(`
Plan a one-way sync from local Markdown files to reMarkable PDF documents.

Inputs match upload md: pass markdown files (*.md) and/or directories, and directories are scanned recursively.
The sync command builds a remote index before conversion so it can report which files would upload, skip, or become stale without running pandoc for unchanged files.

Current implementation supports dry-run planning. Mutating upload execution will be added after the plan output is stable.
`),
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUploadSync(cmd.Context(), cmd, s, args)
		},
	}

	cmd.Flags().BoolVar(&s.NonInteractive, "non-interactive", false, "Do not prompt for one-time code; fail if tokens are missing")
	cmd.Flags().BoolVar(&s.Reauth, "reauth", false, "Force re-authentication (re-fetch user token)")

	cmd.Flags().BoolVar(&s.Force, "force", false, "Overwrite stale existing documents when execution is enabled (WARNING: deletes existing document + annotations)")
	cmd.Flags().BoolVar(&s.DryRun, "dry-run", false, "Print the sync plan, but do not run pandoc or upload")
	cmd.Flags().BoolVar(&s.PreserveDirs, "preserve-dirs", true, "Recreate the local relative directory structure remotely (default: true)")
	cmd.Flags().BoolVar(&s.Flatten, "flatten", false, "Upload all files to a single flat directory (overrides --preserve-dirs)")
	cmd.Flags().StringVar(&s.Name, "name", "", "Custom output document name (only valid when exactly one markdown file is selected)")
	cmd.Flags().BoolVar(&s.CompareMTime, "compare-mtime", false, "Mark remote documents stale when the local markdown file is newer than the remote document")
	cmd.Flags().BoolVar(&s.DeleteOrphans, "delete-orphans", false, "Report remote documents with no matching local markdown input")

	cmd.Flags().StringVar(&s.Date, "date", "", "Destination date folder under /ai (YYYY/MM/DD or YYYY-MM-DD). Default: today")
	cmd.Flags().StringVar(&s.RemoteDir, "remote-dir", "", "Override remote directory (default: /ai/YYYY/MM/DD/)")

	cmd.Flags().StringVar(&s.Pandoc, "pandoc", "pandoc", "Pandoc binary to run")
	cmd.Flags().StringVar(&s.PDFEngine, "pdf-engine", "xelatex", "Pandoc PDF engine (default: xelatex)")
	cmd.Flags().StringVar(&s.MainFont, "mainfont", "DejaVu Sans", "Main font for PDF generation")
	cmd.Flags().StringVar(&s.MonoFont, "monofont", "DejaVu Sans Mono", "Monospace font for code blocks")
	cmd.Flags().StringVar(&s.Layout, "layout", mdpdf.MarkdownLayoutDefault, "Markdown layout preset: default|editor (editor adds wider margins and more annotation-friendly spacing)")
	cmd.Flags().StringVar(&s.Geometry, "geometry", "margin=1in", "LaTeX geometry setting passed to pandoc (default: margin=1in)")
	cmd.Flags().StringVar(&s.LatexHeaderFile, "latex-header-file", "", "Optional path to a LaTeX header file to include (overrides built-in header)")

	return cmd
}

func runUploadSync(ctx context.Context, cmd *cobra.Command, s *uploadSyncSettings, args []string) error {
	if s.Flatten {
		s.PreserveDirs = false
	}

	if !s.DryRun {
		return errors.New("upload sync execution is not implemented yet; rerun with --dry-run to inspect the sync plan")
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

	if _, err := configureMarkdownPandocOptions(
		cmd.Flags(),
		s.Layout,
		s.Pandoc,
		s.PDFEngine,
		s.MainFont,
		s.MonoFont,
		s.Geometry,
		s.LatexHeaderFile,
	); err != nil {
		return err
	}

	_, apiCtx, err := rmcloud.CreateApiCtx(rmcloud.AuthSettings{
		NonInteractive: s.NonInteractive,
		Reauth:         s.Reauth,
	})
	if err != nil {
		return err
	}

	remoteIndex, err := buildSyncRemoteIndex(apiCtx, remoteDir)
	if err != nil {
		return err
	}

	plan, err := buildSyncPlan(mdInputs, remoteIndex, syncPlanSettings{
		RemoteDir:     remoteDir,
		PreserveDirs:  s.PreserveDirs,
		CompareMTime:  s.CompareMTime,
		DeleteOrphans: s.DeleteOrphans,
		Name:          s.Name,
	})
	if err != nil {
		return err
	}

	printSyncPlan(cmd, remoteDir, plan)
	_ = ctx
	return nil
}

func buildSyncRemoteIndex(apiCtx api.ApiCtx, remoteDir string) (map[string]syncRemoteEntry, error) {
	remoteIndex := map[string]syncRemoteEntry{}

	startNode, err := apiCtx.Filetree().NodeByPath(remoteDir, nil)
	if err != nil {
		return remoteIndex, nil
	}

	filetree.WalkTree(startNode, filetree.FileTreeVistor{
		Visit: func(node *model.Node, _ []string) bool {
			if node == nil || node.IsRoot() {
				return filetree.ContinueVisiting
			}

			p := buildUploadSyncPathFromParents(node)
			modTime, _ := node.LastModified()
			remoteIndex[p] = syncRemoteEntry{
				Name:         node.Name(),
				Path:         p,
				IsDir:        node.IsDirectory(),
				ModifiedTime: modTime,
			}
			return filetree.ContinueVisiting
		},
	})

	return remoteIndex, nil
}

func printSyncPlan(cmd *cobra.Command, remoteDir string, plan syncPlan) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "SYNC: remote-dir=%s\n", remoteDir)
	fmt.Fprintf(out, "SUMMARY: upload=%d skip=%d stale=%d orphan=%d\n",
		plan.Count(syncActionUpload),
		plan.Count(syncActionSkip),
		plan.Count(syncActionStale),
		plan.Count(syncActionOrphan),
	)

	for _, item := range plan.Items {
		switch item.Action {
		case syncActionUpload:
			fmt.Fprintf(out, "UPLOAD: %s <- %s\n", item.Local.RemoteKey, item.Local.Input.AbsPath)
		case syncActionSkip:
			fmt.Fprintf(out, "SKIP: %s (%s)\n", item.Local.RemoteKey, item.Reason)
		case syncActionStale:
			fmt.Fprintf(out, "STALE: %s (%s)\n", item.Local.RemoteKey, item.Reason)
		case syncActionOrphan:
			fmt.Fprintf(out, "ORPHAN: %s (%s)\n", item.Remote.Path, item.Reason)
		}
	}
}

func buildUploadSyncPathFromParents(n *model.Node) string {
	if n == nil {
		return ""
	}
	if n.IsRoot() {
		return "/"
	}

	parts := []string{n.Name()}
	cur := n.Parent
	for cur != nil && !cur.IsRoot() {
		parts = append(parts, cur.Name())
		cur = cur.Parent
	}

	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}

	return "/" + filepath.ToSlash(strings.Join(parts, "/"))
}

func parseRemoteModifiedTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, value); err == nil {
			return t
		}
	}
	return time.Time{}
}
