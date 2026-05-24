package upload

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

	// Image flags.
	ResolveImages bool
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

Use --dry-run to inspect the delta without converting, uploading, or deleting. Without --dry-run, files classified as UPLOAD are converted and uploaded. STALE files require --force before they are replaced. ORPHAN files are deleted only when both --delete-orphans and --force are set.
`),
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUploadSync(cmd.Context(), cmd, s, args)
		},
	}

	cmd.Flags().BoolVar(&s.NonInteractive, "non-interactive", false, "Do not prompt for one-time code; fail if tokens are missing")
	cmd.Flags().BoolVar(&s.Reauth, "reauth", false, "Force re-authentication (re-fetch user token)")

	cmd.Flags().BoolVar(&s.Force, "force", false, "Overwrite stale documents and delete orphans when requested (WARNING: deletes existing documents + annotations)")
	cmd.Flags().BoolVar(&s.DryRun, "dry-run", false, "Print the sync plan, but do not run pandoc or upload")
	cmd.Flags().BoolVar(&s.PreserveDirs, "preserve-dirs", true, "Recreate the local relative directory structure remotely (default: true)")
	cmd.Flags().BoolVar(&s.Flatten, "flatten", false, "Upload all files to a single flat directory (overrides --preserve-dirs)")
	cmd.Flags().StringVar(&s.Name, "name", "", "Custom output document name (only valid when exactly one markdown file is selected)")
	cmd.Flags().BoolVar(&s.CompareMTime, "compare-mtime", false, "Mark remote documents stale when the local markdown file is newer than the remote document")
	cmd.Flags().BoolVar(&s.DeleteOrphans, "delete-orphans", false, "Report remote documents with no matching local markdown input; delete them during execution only with --force")

	cmd.Flags().StringVar(&s.Date, "date", "", "Destination date folder under /ai (YYYY/MM/DD or YYYY-MM-DD). Default: today")
	cmd.Flags().StringVar(&s.RemoteDir, "remote-dir", "", "Override remote directory (default: /ai/YYYY/MM/DD/)")

	cmd.Flags().StringVar(&s.Pandoc, "pandoc", "pandoc", "Pandoc binary to run")
	cmd.Flags().StringVar(&s.PDFEngine, "pdf-engine", "xelatex", "Pandoc PDF engine (default: xelatex)")
	cmd.Flags().StringVar(&s.MainFont, "mainfont", "DejaVu Sans", "Main font for PDF generation")
	cmd.Flags().StringVar(&s.MonoFont, "monofont", "DejaVu Sans Mono", "Monospace font for code blocks")
	cmd.Flags().StringVar(&s.Layout, "layout", mdpdf.MarkdownLayoutDefault, "Markdown layout preset: default|editor (editor adds wider margins and more annotation-friendly spacing)")
	cmd.Flags().StringVar(&s.Geometry, "geometry", "margin=1in", "LaTeX geometry setting passed to pandoc (default: margin=1in)")
	cmd.Flags().StringVar(&s.LatexHeaderFile, "latex-header-file", "", "Optional path to a LaTeX header file to include (overrides built-in header)")

	// Mermaid flags (match upload md/bundle).
	if err := addMermaidFlagsToCommand(cmd); err != nil {
		panic(err) // should never happen with static definitions
	}

	// Image flags.
	addResolveImagesFlag(cmd, &s.ResolveImages)

	return cmd
}

func runUploadSync(ctx context.Context, cmd *cobra.Command, s *uploadSyncSettings, args []string) error {
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
	if s.DryRun {
		return nil
	}

	return executeSyncPlan(ctx, cmd, apiCtx, plan, pandocOpts, s.Force, s.PreserveDirs)
}

func executeSyncPlan(ctx context.Context, cmd *cobra.Command, apiCtx api.ApiCtx, plan syncPlan, pandocOpts mdpdf.PandocOptions, force bool, preserveDirs bool) error {
	dstNodeCache := map[string]*model.Node{}

	tmpDir, err := os.MkdirTemp("", "remarquee-upload-sync-")
	if err != nil {
		return errors.Wrap(err, "failed to create temp directory")
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	type failedItem struct {
		key   string
		err   error
		phase string // "convert" or "upload"
	}
	var failures []failedItem

	for _, item := range plan.Items {
		switch item.Action {
		case syncActionSkip, syncActionError:
			continue
		case syncActionOrphan:
			if !force {
				fmt.Fprintf(cmd.OutOrStdout(), "SKIP-ORPHAN: %s (use --force with --delete-orphans to delete)\n", item.Remote.Path)
				continue
			}
			if err := deleteSyncRemoteEntry(apiCtx, item.Remote, "orphan remote file"); err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "ERROR-DELETE: %s — %v\n", item.Remote.Path, err)
				failures = append(failures, failedItem{key: item.Remote.Path, err: err, phase: "delete"})
				continue
			}
			fmt.Fprintf(cmd.OutOrStdout(), "OK: deleted orphan %s\n", item.Remote.Path)
			continue
		case syncActionStale:
			if !force {
				fmt.Fprintf(cmd.OutOrStdout(), "SKIP-STALE: %s (use --force to overwrite)\n", item.Local.RemoteKey)
				continue
			}
			if err := deleteSyncRemoteEntry(apiCtx, item.Remote, "stale remote file"); err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "ERROR-DELETE: %s — %v\n", item.Local.RemoteKey, err)
				failures = append(failures, failedItem{key: item.Local.RemoteKey, err: err, phase: "delete"})
				continue
			}
		case syncActionUpload:
			// Continue below and upload the new document.
		}

		if item.Local == nil {
			continue
		}

		outPDF := filepath.Join(tmpDir, item.Local.PDFName)
		if preserveDirs {
			outPDF = filepath.Join(tmpDir, item.Local.Input.RelDir(), item.Local.PDFName)
			if err := os.MkdirAll(filepath.Dir(outPDF), 0o755); err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "ERROR-CONVERT: %s — %v\n", item.Local.RemoteKey, err)
				failures = append(failures, failedItem{key: item.Local.RemoteKey, err: err, phase: "convert"})
				continue
			}
		}

		if err := mdpdf.ConvertMarkdownFileToPDF(ctx, item.Local.Input.AbsPath, outPDF, pandocOpts); err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "ERROR-CONVERT: %s — %v\n", item.Local.RemoteKey, err)
			failures = append(failures, failedItem{key: item.Local.RemoteKey, err: err, phase: "convert"})
			continue
		}

		if err := uploadPDFToRemote(cmd, apiCtx, dstNodeCache, item.Local.RemoteDir, outPDF, item.Local.PDFName); err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "ERROR-UPLOAD: %s — %v\n", item.Local.RemoteKey, err)
			failures = append(failures, failedItem{key: item.Local.RemoteKey, err: err, phase: "upload"})
			continue
		}
	}

	if len(failures) > 0 {
		convertFailed := 0
		uploadFailed := 0
		deleteFailed := 0
		for _, f := range failures {
			switch f.phase {
			case "convert":
				convertFailed++
			case "upload":
				uploadFailed++
			case "delete":
				deleteFailed++
			}
		}
		fmt.Fprintf(cmd.OutOrStdout(), "ERRORS: convert-failed=%d upload-failed=%d delete-failed=%d\n", convertFailed, uploadFailed, deleteFailed)
		return errors.Errorf("%d file(s) failed during sync (convert=%d, upload=%d, delete=%d)",
			len(failures), convertFailed, uploadFailed, deleteFailed)
	}

	return nil
}

func deleteSyncRemoteEntry(apiCtx api.ApiCtx, remote *syncRemoteEntry, label string) error {
	if remote == nil || remote.Node == nil {
		return errors.Errorf("cannot delete %s: missing remote node", label)
	}
	remoteNode, ok := remote.Node.(*model.Node)
	if !ok {
		return errors.Errorf("cannot delete %s: unexpected remote node type", label)
	}
	if remoteNode.IsDirectory() {
		return errors.Errorf("cannot delete directory %q as %s", remote.Path, label)
	}
	if err := apiCtx.DeleteEntry(remoteNode, false, false); err != nil {
		return errors.Wrapf(err, "failed to delete %s", label)
	}
	apiCtx.Filetree().DeleteNode(remoteNode)
	return nil
}

func buildSyncRemoteIndex(apiCtx api.ApiCtx, remoteDir string) (map[string]syncRemoteEntry, error) {
	remoteIndex := map[string]syncRemoteEntry{}

	startNode, err := apiCtx.Filetree().NodeByPath(remoteDir, nil)
	if err != nil {
		if isFiletreeNotFoundError(err) {
			return remoteIndex, nil
		}
		return nil, errors.Wrapf(err, "failed to inspect remote sync root %q", remoteDir)
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
				Node:         node,
			}
			return filetree.ContinueVisiting
		},
	})

	return remoteIndex, nil
}

func printSyncPlan(cmd *cobra.Command, remoteDir string, plan syncPlan) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "SYNC: remote-dir=%s\n", remoteDir)
	fmt.Fprintf(out, "SUMMARY: upload=%d skip=%d stale=%d orphan=%d error=%d\n",
		plan.Count(syncActionUpload),
		plan.Count(syncActionSkip),
		plan.Count(syncActionStale),
		plan.Count(syncActionOrphan),
		plan.Count(syncActionError),
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
		case syncActionError:
			if item.Local != nil {
				fmt.Fprintf(out, "ERROR: %s (%s)\n", item.Local.RemoteKey, item.Reason)
			} else {
				fmt.Fprintf(out, "ERROR: %s\n", item.Reason)
			}
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

func isFiletreeNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "doesnt exist") || strings.Contains(msg, "doesn't exist") || strings.Contains(msg, "not found")
}
