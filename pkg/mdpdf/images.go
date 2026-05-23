package mdpdf

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// imageEmbedRegex matches Markdown image syntax: ![alt](path)
// It captures the alt text (group 1) and the path (group 2).
var imageEmbedRegex = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)

// ResolveImagePaths rewrites relative image paths in Markdown so that pandoc
// can find them from the temp directory where the preprocessed Markdown will
// be written. It copies referenced image files into tmpDir/images/ and
// rewrites the Markdown paths to be relative to the preprocessed file.
//
// Absolute paths and URLs (http://, https://, data:) are left unchanged.
// Missing image files produce a warning but do not cause an error.
//
// Parameters:
//   - body: the Markdown content (after YAML stripping)
//   - sourceDir: the directory of the original Markdown file (for resolving relative paths)
//   - tmpDir: the temp directory where the preprocessed Markdown will be written
//
// Returns the rewritten Markdown body.
func ResolveImagePaths(body string, sourceDir string, tmpDir string) (string, error) {
	imagesDir := filepath.Join(tmpDir, "images")
	if err := os.MkdirAll(imagesDir, 0o755); err != nil {
		return body, fmt.Errorf("failed to create images directory: %w", err)
	}

	// Track used filenames to handle collisions.
	usedNames := map[string]bool{}

	result := imageEmbedRegex.ReplaceAllStringFunc(body, func(match string) string {
		submatch := imageEmbedRegex.FindStringSubmatch(match)
		if len(submatch) < 3 {
			return match
		}
		alt := submatch[1]
		imgPath := submatch[2]

		// Skip URLs and absolute paths.
		if isURL(imgPath) || filepath.IsAbs(imgPath) {
			return match
		}

		// Resolve relative to the original Markdown file's directory.
		resolvedPath := filepath.Join(sourceDir, imgPath)
		resolvedPath = filepath.Clean(resolvedPath)

		// Check if the file exists.
		info, err := os.Stat(resolvedPath)
		if err != nil || info.IsDir() {
			// Image not found: leave unchanged, don't error.
			return match
		}

		// Determine a unique filename in the images directory.
		base := filepath.Base(resolvedPath)
		destName := base
		if usedNames[destName] {
			ext := filepath.Ext(base)
			stem := strings.TrimSuffix(base, ext)
			for i := 1; usedNames[destName]; i++ {
				destName = fmt.Sprintf("%s-%d%s", stem, i, ext)
			}
		}
		usedNames[destName] = true

		destPath := filepath.Join(imagesDir, destName)

		// Copy the image file.
		if err := copyFile(resolvedPath, destPath); err != nil {
			return match
		}

		// Rewrite the Markdown path to be relative to the preprocessed file.
		return fmt.Sprintf("![%s](./images/%s)", alt, destName)
	})

	return result, nil
}

// isURL returns true if the path looks like a URL scheme.
func isURL(path string) bool {
	return strings.HasPrefix(path, "http://") ||
		strings.HasPrefix(path, "https://") ||
		strings.HasPrefix(path, "data:")
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %q: %w", src, err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %q: %w", dst, err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy %q to %q: %w", src, dst, err)
	}

	// Preserve the source file's mode.
	srcInfo, err := os.Stat(src)
	if err != nil {
		return nil // non-fatal
	}
	_ = dstFile.Chmod(srcInfo.Mode())

	return nil
}
