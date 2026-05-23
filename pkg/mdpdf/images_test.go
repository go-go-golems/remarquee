package mdpdf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsURL(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"http://example.com/img.png", true},
		{"https://example.com/img.png", true},
		{"data:image/png;base64,abc", true},
		{"./images/img.png", false},
		{"/absolute/path/img.png", false},
		{"relative/img.png", false},
	}
	for _, tt := range tests {
		if got := isURL(tt.path); got != tt.want {
			t.Errorf("isURL(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestResolveImagePaths_RelativePath(t *testing.T) {
	// Set up source directory with an image.
	srcDir := t.TempDir()
	imgPath := filepath.Join(srcDir, "logo.png")
	if err := os.WriteFile(imgPath, []byte("fake-png-data"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Set up tmp directory (simulates pandoc temp dir).
	tmpDir := t.TempDir()

	body := "![logo](./logo.png)\n"
	result, err := ResolveImagePaths(body, srcDir, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should rewrite to ./images/logo.png
	if !strings.Contains(result, "![logo](./images/logo.png)") {
		t.Fatalf("expected rewritten image path, got: %q", result)
	}

	// Image should have been copied.
	copiedPath := filepath.Join(tmpDir, "images", "logo.png")
	if _, err := os.Stat(copiedPath); err != nil {
		t.Fatalf("expected copied image at %q: %v", copiedPath, err)
	}

	// Copied content should match.
	data, err := os.ReadFile(copiedPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "fake-png-data" {
		t.Fatalf("copied content mismatch: got %q", string(data))
	}
}

func TestResolveImagePaths_RelativeSubdirectory(t *testing.T) {
	srcDir := t.TempDir()
	subDir := filepath.Join(srcDir, "assets", "img")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	imgPath := filepath.Join(subDir, "diagram.png")
	if err := os.WriteFile(imgPath, []byte("png-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()

	body := "![diagram](./assets/img/diagram.png)\n"
	result, err := ResolveImagePaths(body, srcDir, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "![diagram](./images/diagram.png)") {
		t.Fatalf("expected rewritten path, got: %q", result)
	}
}

func TestResolveImagePaths_URL_Unchanged(t *testing.T) {
	tmpDir := t.TempDir()

	body := "![remote](https://example.com/img.png)\n"
	result, err := ResolveImagePaths(body, "/nonexistent", tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != body {
		t.Fatalf("URL path should be unchanged, got: %q", result)
	}
}

func TestResolveImagePaths_AbsolutePath_Unchanged(t *testing.T) {
	tmpDir := t.TempDir()

	body := "![abs](/tmp/some-image.png)\n"
	result, err := ResolveImagePaths(body, "/nonexistent", tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != body {
		t.Fatalf("absolute path should be unchanged, got: %q", result)
	}
}

func TestResolveImagePaths_MissingImage(t *testing.T) {
	tmpDir := t.TempDir()

	body := "![missing](./nonexistent.png)\n"
	result, err := ResolveImagePaths(body, "/nonexistent", tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should leave path unchanged (no error for missing images).
	if result != body {
		t.Fatalf("missing image path should be unchanged, got: %q", result)
	}
}

func TestResolveImagePaths_CollisionAvoidance(t *testing.T) {
	// Test that two images with the same basename in the same body get different names.
	srcDir := t.TempDir()
	dir1 := filepath.Join(srcDir, "dir1")
	dir2 := filepath.Join(srcDir, "dir2")
	if err := os.MkdirAll(dir1, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir2, 0o755); err != nil {
		t.Fatal(err)
	}

	// Two different images with the same basename.
	if err := os.WriteFile(filepath.Join(dir1, "icon.png"), []byte("icon-a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir2, "icon.png"), []byte("icon-b"), 0o644); err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()

	// Both images referenced in the same body.
	body := "![icon1](./dir1/icon.png) and ![icon2](./dir2/icon.png)\n"
	result, err := ResolveImagePaths(body, srcDir, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First should get the base name, second should get a suffix.
	if !strings.Contains(result, "./images/icon.png") {
		t.Fatalf("first image should be ./images/icon.png, got: %q", result)
	}
	if !strings.Contains(result, "./images/icon-1.png") {
		t.Fatalf("second image should be ./images/icon-1.png, got: %q", result)
	}

	// Both files should exist with correct content.
	if data, err := os.ReadFile(filepath.Join(tmpDir, "images", "icon.png")); err != nil {
		t.Fatalf("first icon not found: %v", err)
	} else if string(data) != "icon-a" {
		t.Fatalf("first icon content mismatch: got %q", string(data))
	}
	if data, err := os.ReadFile(filepath.Join(tmpDir, "images", "icon-1.png")); err != nil {
		t.Fatalf("second icon not found: %v", err)
	} else if string(data) != "icon-b" {
		t.Fatalf("second icon content mismatch: got %q", string(data))
	}
}

func TestResolveImagePaths_MultipleImagesInOneBody(t *testing.T) {
	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "a.png"), []byte("aaa"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "b.png"), []byte("bbb"), 0o644); err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()

	body := "See ![a](./a.png) and ![b](./b.png).\n"
	result, err := ResolveImagePaths(body, srcDir, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "./images/a.png") {
		t.Fatalf("expected a.png rewrite, got: %q", result)
	}
	if !strings.Contains(result, "./images/b.png") {
		t.Fatalf("expected b.png rewrite, got: %q", result)
	}
}

func TestResolveImagePaths_DataURI_Unchanged(t *testing.T) {
	tmpDir := t.TempDir()

	body := "![inline](data:image/png;base64,iVBOR)\n"
	result, err := ResolveImagePaths(body, "/nonexistent", tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != body {
		t.Fatalf("data URI should be unchanged, got: %q", result)
	}
}

func TestResolveImagePaths_NoImages(t *testing.T) {
	tmpDir := t.TempDir()

	body := "# Title\n\nJust text.\n"
	result, err := ResolveImagePaths(body, "/nonexistent", tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != body {
		t.Fatalf("body with no images should be unchanged, got: %q", result)
	}
}
