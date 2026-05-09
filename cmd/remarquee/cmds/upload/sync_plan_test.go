package upload

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildSyncPlan_UploadAndSkip(t *testing.T) {
	td := t.TempDir()
	a := writeMarkdownFixture(t, td, "a.md")
	b := writeMarkdownFixture(t, td, "b.md")

	inputs := []markdownInput{
		{AbsPath: a, RelPath: "a.md"},
		{AbsPath: b, RelPath: "b.md"},
	}
	remoteIndex := map[string]syncRemoteEntry{
		"/ai/sync/a": {Name: "a", Path: "/ai/sync/a"},
	}

	plan, err := buildSyncPlan(inputs, remoteIndex, syncPlanSettings{
		RemoteDir:    "/ai/sync",
		PreserveDirs: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := plan.Count(syncActionSkip); got != 1 {
		t.Fatalf("expected 1 skip, got %d: %#v", got, plan.Items)
	}
	if got := plan.Count(syncActionUpload); got != 1 {
		t.Fatalf("expected 1 upload, got %d: %#v", got, plan.Items)
	}
}

func TestBuildSyncPlan_PreserveDirsUsesFullRemotePath(t *testing.T) {
	td := t.TempDir()
	rootNote := writeMarkdownFixture(t, td, "note.md")
	subNote := writeMarkdownFixture(t, td, filepath.Join("sub", "note.md"))

	inputs := []markdownInput{
		{AbsPath: rootNote, RelPath: "note.md"},
		{AbsPath: subNote, RelPath: filepath.Join("sub", "note.md")},
	}
	remoteIndex := map[string]syncRemoteEntry{
		"/ai/sync/note": {Name: "note", Path: "/ai/sync/note"},
	}

	plan, err := buildSyncPlan(inputs, remoteIndex, syncPlanSettings{
		RemoteDir:    "/ai/sync",
		PreserveDirs: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := plan.Count(syncActionSkip); got != 1 {
		t.Fatalf("expected root note to skip, got %d skips", got)
	}
	if got := plan.Count(syncActionUpload); got != 1 {
		t.Fatalf("expected sub note to upload, got %d uploads", got)
	}

	var uploadKey string
	for _, item := range plan.Items {
		if item.Action == syncActionUpload {
			uploadKey = item.Local.RemoteKey
		}
	}
	if uploadKey != "/ai/sync/sub/note" {
		t.Fatalf("expected upload key to preserve directory, got %q", uploadKey)
	}
}

func TestBuildSyncPlan_FlattenDetectsDuplicateRemoteKeys(t *testing.T) {
	td := t.TempDir()
	rootNote := writeMarkdownFixture(t, td, "note.md")
	subNote := writeMarkdownFixture(t, td, filepath.Join("sub", "note.md"))

	_, err := buildSyncPlan([]markdownInput{
		{AbsPath: rootNote, RelPath: "note.md"},
		{AbsPath: subNote, RelPath: filepath.Join("sub", "note.md")},
	}, nil, syncPlanSettings{
		RemoteDir:    "/ai/sync",
		PreserveDirs: false,
	})
	if err == nil {
		t.Fatal("expected duplicate remote key error")
	}
}

func TestBuildSyncPlan_CompareMTimeMarksStale(t *testing.T) {
	td := t.TempDir()
	note := writeMarkdownFixture(t, td, "note.md")
	localTime := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	remoteTime := localTime.Add(-time.Hour)
	if err := os.Chtimes(note, localTime, localTime); err != nil {
		t.Fatal(err)
	}

	plan, err := buildSyncPlan([]markdownInput{{AbsPath: note, RelPath: "note.md"}}, map[string]syncRemoteEntry{
		"/ai/sync/note": {Name: "note", Path: "/ai/sync/note", ModifiedTime: remoteTime},
	}, syncPlanSettings{
		RemoteDir:     "/ai/sync",
		PreserveDirs:  true,
		CompareMTime:  true,
		DeleteOrphans: false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := plan.Count(syncActionStale); got != 1 {
		t.Fatalf("expected 1 stale item, got %d: %#v", got, plan.Items)
	}
}

func TestNewUploadCommandRegistersSync(t *testing.T) {
	cmd := NewUploadCommand()
	for _, child := range cmd.Commands() {
		if child.Name() == "sync" {
			return
		}
	}
	t.Fatal("expected upload root command to register sync subcommand")
}

func TestIsFiletreeNotFoundError(t *testing.T) {
	if !isFiletreeNotFoundError(errors.New("entry 'missing' doesnt exist")) {
		t.Fatal("expected rmapi missing-entry error to be classified as not found")
	}
	if isFiletreeNotFoundError(errors.New("network timeout")) {
		t.Fatal("expected non-notfound error to propagate")
	}
}

func TestUploadSyncHelpDocumentsForceForOrphanDeletion(t *testing.T) {
	cmd := NewUploadSyncCommand()
	help := cmd.Long + "\n" + cmd.Flag("delete-orphans").Usage + "\n" + cmd.Flag("force").Usage
	if !strings.Contains(help, "ORPHAN files are deleted only when both --delete-orphans and --force are set") {
		t.Fatalf("expected long help to document orphan deletion safety, got:\n%s", help)
	}
	if !strings.Contains(help, "delete them during execution only with --force") {
		t.Fatalf("expected delete-orphans flag help to mention --force, got:\n%s", help)
	}
}

func TestPrintSyncPlanSummary(t *testing.T) {
	td := t.TempDir()
	note := writeMarkdownFixture(t, td, "note.md")
	plan, err := buildSyncPlan([]markdownInput{{AbsPath: note, RelPath: "note.md"}}, nil, syncPlanSettings{
		RemoteDir:    "/ai/sync",
		PreserveDirs: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd := NewUploadSyncCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	printSyncPlan(cmd, "/ai/sync", plan)

	if !strings.Contains(out.String(), "SUMMARY: upload=1 skip=0 stale=0 orphan=0") {
		t.Fatalf("expected summary in output, got:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "UPLOAD: /ai/sync/note <- "+note) {
		t.Fatalf("expected upload line in output, got:\n%s", out.String())
	}
}

func TestBuildSyncPlan_DeleteOrphans(t *testing.T) {
	td := t.TempDir()
	note := writeMarkdownFixture(t, td, "note.md")

	plan, err := buildSyncPlan([]markdownInput{{AbsPath: note, RelPath: "note.md"}}, map[string]syncRemoteEntry{
		"/ai/sync/note":   {Name: "note", Path: "/ai/sync/note"},
		"/ai/sync/orphan": {Name: "orphan", Path: "/ai/sync/orphan"},
		"/ai/sync/folder": {Name: "folder", Path: "/ai/sync/folder", IsDir: true},
	}, syncPlanSettings{
		RemoteDir:     "/ai/sync",
		PreserveDirs:  true,
		DeleteOrphans: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := plan.Count(syncActionOrphan); got != 1 {
		t.Fatalf("expected 1 orphan, got %d: %#v", got, plan.Items)
	}
}

func writeMarkdownFixture(t *testing.T, root string, rel string) string {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("# note\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
