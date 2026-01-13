package debug

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func testdataPath(t *testing.T, filename string) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}

	remarqueeDir := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	return filepath.Join(remarqueeDir, "cmd", "remarquee-ui", "testdata", filename)
}

func TestDetectRMVersionFromHeader(t *testing.T) {
	for _, tc := range []struct {
		name   string
		header string
		wantV  string
		wantOK bool
	}{
		{name: "v3", header: "reMarkable .lines file, version=3      ", wantV: "V3", wantOK: true},
		{name: "v5", header: "reMarkable .lines file, version=5      ", wantV: "V5", wantOK: true},
		{name: "v6", header: "reMarkable .lines file, version=6      ", wantV: "V6", wantOK: true},
		{name: "unknown", header: "reMarkable .lines file, version=99     ", wantV: "", wantOK: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gotV, gotOK := DetectRMVersionFromHeader([]byte(tc.header))
			if gotV != tc.wantV || gotOK != tc.wantOK {
				t.Fatalf("DetectRMVersionFromHeader(%q) = (%q, %v), want (%q, %v)", tc.header, gotV, gotOK, tc.wantV, tc.wantOK)
			}
		})
	}
}

func TestListArchiveFiles(t *testing.T) {
	ctx := context.Background()

	for _, name := range []string{"cpage-pdf.rmdoc", "legacy-notebook.zip"} {
		t.Run(name, func(t *testing.T) {
			files, err := ListArchiveFiles(ctx, testdataPath(t, name))
			if err != nil {
				t.Fatalf("ListArchiveFiles: %v", err)
			}
			if len(files) == 0 {
				t.Fatalf("expected some files")
			}

			hasContent := false
			for _, f := range files {
				if strings.HasSuffix(f, ".content") {
					hasContent = true
					break
				}
			}
			if !hasContent {
				t.Fatalf("expected at least one .content entry, got %v", files)
			}
		})
	}
}

func TestInspectRMFiles(t *testing.T) {
	ctx := context.Background()

	for _, name := range []string{"cpage-pdf.rmdoc", "legacy-notebook.zip"} {
		t.Run(name, func(t *testing.T) {
			rmFiles, err := InspectRMFiles(ctx, testdataPath(t, name))
			if err != nil {
				t.Fatalf("InspectRMFiles: %v", err)
			}
			if len(rmFiles) == 0 {
				t.Fatalf("expected some .rm files")
			}

			for _, f := range rmFiles {
				if f.PageID == "" {
					t.Fatalf("expected non-empty PageID for %+v", f)
				}
				if f.Filename == "" || !strings.HasSuffix(f.Filename, ".rm") {
					t.Fatalf("expected .rm filename, got %+v", f)
				}
				// Version can be "unknown" (we only sniff headers), but should never be empty.
				if f.Version == "" {
					t.Fatalf("expected non-empty Version for %+v", f)
				}
			}
		})
	}
}
