package upload

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectMarkdownFiles_FileAndDirectory(t *testing.T) {
	td := t.TempDir()

	// dir structure:
	// td/a.md
	// td/sub/b.md
	// td/sub/c.txt
	if err := os.WriteFile(filepath.Join(td, "a.md"), []byte("# a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(td, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(td, "sub", "b.md"), []byte("# b"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(td, "sub", "c.txt"), []byte("no"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := collectMarkdownFiles([]string{td, filepath.Join(td, "a.md")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %#v", len(files), files)
	}
}

func TestResolveRemoteDir_DateFormats(t *testing.T) {
	got, err := resolveRemoteDir("2025/12/15", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/ai/2025/12/15" {
		t.Fatalf("unexpected: %q", got)
	}

	got2, err := resolveRemoteDir("2025-12-15", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got2 != "/ai/2025/12/15" {
		t.Fatalf("unexpected: %q", got2)
	}
}

func TestResolveRemoteDir_OverrideNormalizes(t *testing.T) {
	got, err := resolveRemoteDir("", "ai/2025/12/15/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/ai/2025/12/15" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestJoinRemoteDir(t *testing.T) {
	base := "/ai/test"

	if got := joinRemoteDir(base, ""); got != "/ai/test" {
		t.Fatalf("unexpected: %q", got)
	}
	if got := joinRemoteDir(base, "."); got != "/ai/test" {
		t.Fatalf("unexpected: %q", got)
	}
	if got := joinRemoteDir(base, "sub"); got != "/ai/test/sub" {
		t.Fatalf("unexpected: %q", got)
	}
	if got := joinRemoteDir(base, "/sub/dir/"); got != "/ai/test/sub/dir" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestCollectMarkdownInputs_PreservesRelativePaths(t *testing.T) {
	td := t.TempDir()

	// dir structure:
	// td/a.md
	// td/sub/b.md
	// td/sub/deep/c.md
	if err := os.WriteFile(filepath.Join(td, "a.md"), []byte("# a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(td, "sub", "deep"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(td, "sub", "b.md"), []byte("# b"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(td, "sub", "deep", "c.md"), []byte("# c"), 0o644); err != nil {
		t.Fatal(err)
	}

	inputs, err := collectMarkdownInputs([]string{td})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(inputs) != 3 {
		t.Fatalf("expected 3 inputs, got %d: %#v", len(inputs), inputs)
	}

	// Build a map of RelPath -> RelDir for easy lookup.
	relDirs := map[string]string{}
	for _, in := range inputs {
		relDirs[in.RelPath] = in.RelDir()
	}

	// a.md is at root level: RelPath="a.md", RelDir=""
	if rd, ok := relDirs["a.md"]; !ok || rd != "" {
		t.Fatalf("expected a.md with RelDir empty, got ok=%v rd=%q", ok, rd)
	}
	// sub/b.md: RelDir="sub"
	if rd, ok := relDirs[filepath.Join("sub", "b.md")]; !ok || rd != "sub" {
		t.Fatalf("expected sub/b.md with RelDir 'sub', got ok=%v rd=%q", ok, rd)
	}
	// sub/deep/c.md: RelDir="sub/deep"
	if rd, ok := relDirs[filepath.Join("sub", "deep", "c.md")]; !ok || rd != filepath.Join("sub", "deep") {
		t.Fatalf("expected sub/deep/c.md with RelDir 'sub/deep', got ok=%v rd=%q", ok, rd)
	}
}

func TestCollectMarkdownInputs_SingleFileHasEmptyRelDir(t *testing.T) {
	td := t.TempDir()

	mdFile := filepath.Join(td, "note.md")
	if err := os.WriteFile(mdFile, []byte("# note"), 0o644); err != nil {
		t.Fatal(err)
	}

	inputs, err := collectMarkdownInputs([]string{mdFile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(inputs) != 1 {
		t.Fatalf("expected 1 input, got %d", len(inputs))
	}
	if inputs[0].RelDir() != "" {
		t.Fatalf("expected empty RelDir for single file, got %q", inputs[0].RelDir())
	}
}

func TestRemoteDocKey(t *testing.T) {
	if got := remoteDocKey("/ai/test", "sub/dir", "doc"); got != "/ai/test/sub/dir/doc" {
		t.Fatalf("unexpected: %q", got)
	}
	if got := remoteDocKey("/ai/test/", "/sub/dir/", "doc"); got != "/ai/test/sub/dir/doc" {
		t.Fatalf("unexpected: %q", got)
	}
	if got := remoteDocKey("/ai/test", "", "doc"); got != "/ai/test/doc" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestUploadMarkdownDryRunShowsEditorLayout(t *testing.T) {
	td := t.TempDir()
	md := filepath.Join(td, "draft.md")
	if err := os.WriteFile(md, []byte("# Draft\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := NewUploadMarkdownCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--dry-run", "--pdf-only", "--layout", "editor", md})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out.String(), "DRY: layout=editor") {
		t.Fatalf("expected dry-run output to mention editor layout, got:\n%s", out.String())
	}
}

func TestUploadMarkdownRejectsUnknownLayout(t *testing.T) {
	td := t.TempDir()
	md := filepath.Join(td, "draft.md")
	if err := os.WriteFile(md, []byte("# Draft\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := NewUploadMarkdownCommand()
	cmd.SetArgs([]string{"--dry-run", "--pdf-only", "--layout", "wide", md})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for unknown layout")
	}
	if !strings.Contains(err.Error(), "unknown markdown layout") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUploadMarkdownDryRunUsesCustomName(t *testing.T) {
	td := t.TempDir()
	md := filepath.Join(td, "draft.md")
	if err := os.WriteFile(md, []byte("# Draft\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := NewUploadMarkdownCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--dry-run", "--pdf-only", "--name", "editor-copy", md})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out.String(), filepath.Join(".", "editor-copy.pdf")) {
		t.Fatalf("expected dry-run output to use custom pdf name, got:\n%s", out.String())
	}
}

func TestUploadMarkdownRejectsNameWithMultipleFiles(t *testing.T) {
	td := t.TempDir()
	a := filepath.Join(td, "a.md")
	b := filepath.Join(td, "b.md")
	if err := os.WriteFile(a, []byte("# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("# B\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := NewUploadMarkdownCommand()
	cmd.SetArgs([]string{"--dry-run", "--pdf-only", "--name", "combined", a, b})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for --name with multiple markdown files")
	}
	if !strings.Contains(err.Error(), "exactly one markdown file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUploadMarkdownRejectsSanitizedNameCollisions(t *testing.T) {
	td := t.TempDir()
	a := filepath.Join(td, "a b.md")
	b := filepath.Join(td, "a_b.md")
	if err := os.WriteFile(a, []byte("# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("# B\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := NewUploadMarkdownCommand()
	cmd.SetArgs([]string{"--dry-run", "--pdf-only", a, b})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for sanitized markdown filename collision")
	}
	if !strings.Contains(err.Error(), "duplicate document") || !strings.Contains(err.Error(), "a_b") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUploadMarkdownRejectsInvalidWorkers(t *testing.T) {
	td := t.TempDir()
	md := filepath.Join(td, "draft.md")
	if err := os.WriteFile(md, []byte("# Draft\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := NewUploadMarkdownCommand()
	cmd.SetArgs([]string{"--dry-run", "--workers", "0", md})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for --workers=0")
	}
	if !strings.Contains(err.Error(), "--workers must be at least 1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUploadMarkdownDryRunShowsWorkers(t *testing.T) {
	td := t.TempDir()
	md := filepath.Join(td, "draft.md")
	if err := os.WriteFile(md, []byte("# Draft\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := NewUploadMarkdownCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--dry-run", "--workers", "4", md})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out.String(), "DRY: workers=4") {
		t.Fatalf("expected dry-run output to mention workers, got:\n%s", out.String())
	}
}

func TestBuildMarkdownConversionJobsPreservesRelativeOutput(t *testing.T) {
	td := t.TempDir()
	out := filepath.Join(td, "out")
	input := markdownInput{AbsPath: filepath.Join(td, "src", "sub", "note.md"), RelPath: filepath.Join("sub", "note.md")}

	jobs, err := buildMarkdownConversionJobs([]markdownInput{input}, out, "", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	want := filepath.Join(out, "sub", "note.pdf")
	if jobs[0].OutPDF != want {
		t.Fatalf("expected output %q, got %q", want, jobs[0].OutPDF)
	}
}

func TestUploadMarkdownRejectsNameWithPathSeparators(t *testing.T) {
	td := t.TempDir()
	md := filepath.Join(td, "draft.md")
	if err := os.WriteFile(md, []byte("# Draft\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"../report", "nested/report", `nested\report`} {
		cmd := NewUploadMarkdownCommand()
		cmd.SetArgs([]string{"--dry-run", "--pdf-only", "--name", name, md})

		err := cmd.Execute()
		if err == nil {
			t.Fatalf("expected error for --name=%q", name)
		}
		if !strings.Contains(err.Error(), "may not contain path separators") {
			t.Fatalf("unexpected error for %q: %v", name, err)
		}
	}
}
