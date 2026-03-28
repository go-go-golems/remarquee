package rmdoc

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/remarquee/pkg/rmcloud"
)

func TestResolveRMDocInput_Local(t *testing.T) {
	t.Helper()

	res, err := ResolveRMDocInput(context.Background(), "/tmp/test.rmdoc", CloudInputSettings{})
	if err != nil {
		t.Fatalf("ResolveRMDocInput: %v", err)
	}
	if res.Source != "local" {
		t.Fatalf("expected local source, got %q", res.Source)
	}
	if res.LocalPath != "/tmp/test.rmdoc" {
		t.Fatalf("expected original local path, got %q", res.LocalPath)
	}
	if err := res.Cleanup(); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
}

func TestResolveRMDocInput_Cloud(t *testing.T) {
	oldDownloader := downloadDocumentByPath
	t.Cleanup(func() {
		downloadDocumentByPath = oldDownloader
	})

	downloadDocumentByPath = func(ctx context.Context, auth rmcloud.AuthSettings, remotePath string, outDir string) (*rmcloud.DownloadedDocument, error) {
		localPath := filepath.Join(outDir, "Test.rmdoc")
		if err := os.WriteFile(localPath, []byte("fixture"), 0o644); err != nil {
			return nil, err
		}
		return &rmcloud.DownloadedDocument{
			RemotePath: remotePath,
			LocalPath:  localPath,
			Name:       "Test",
		}, nil
	}

	res, err := ResolveRMDocInput(context.Background(), "/Books/Test", CloudInputSettings{
		Cloud:          true,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("ResolveRMDocInput: %v", err)
	}
	if res.Source != "cloud" {
		t.Fatalf("expected cloud source, got %q", res.Source)
	}
	if _, err := os.Stat(res.LocalPath); err != nil {
		t.Fatalf("expected downloaded local path to exist: %v", err)
	}
	parentDir := filepath.Dir(res.LocalPath)
	if err := res.Cleanup(); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if _, err := os.Stat(parentDir); !os.IsNotExist(err) {
		t.Fatalf("expected temp dir to be removed, stat err=%v", err)
	}
}
