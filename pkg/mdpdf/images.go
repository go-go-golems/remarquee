package mdpdf

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// inlineImageRegex matches Markdown inline image syntax: ![alt](destination [title]).
// It captures the alt text (group 1) and the full parenthesized target (group 2).
var inlineImageRegex = regexp.MustCompile(`!\[([^\]]*)\]\(([^)\n]+)\)`)

// referenceImageRegex matches full/collapsed reference-style image uses:
// ![alt][id] and ![alt][]. It captures alt text (group 1) and label (group 2).
var referenceImageRegex = regexp.MustCompile(`!\[([^\]]*)\]\[([^\]]*)\]`)

// referenceDefinitionRegex matches Markdown reference definitions:
// [id]: destination [title]
var referenceDefinitionRegex = regexp.MustCompile(`(?m)^([ \t]{0,3})\[([^\]]+)\]:[ \t]*(<[^>]+>|\S+)([^\n]*)$`)

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
	return ResolveImagePathsWithPrefix(body, sourceDir, tmpDir, "")
}

// ResolveImagePathsWithPrefix is like ResolveImagePaths, but prefixes copied
// image filenames. Bundle generation uses this to avoid collisions when
// different input files contain images with the same basename.
func ResolveImagePathsWithPrefix(body string, sourceDir string, tmpDir string, filenamePrefix string) (string, error) {
	imagesDir := filepath.Join(tmpDir, "images")
	if err := os.MkdirAll(imagesDir, 0o755); err != nil {
		return body, fmt.Errorf("failed to create images directory: %w", err)
	}

	// Track used filenames to handle collisions.
	usedNames := map[string]bool{}

	result := inlineImageRegex.ReplaceAllStringFunc(body, func(match string) string {
		submatch := inlineImageRegex.FindStringSubmatch(match)
		if len(submatch) < 3 {
			return match
		}
		alt := submatch[1]
		target := submatch[2]

		imgPath, suffix, ok := splitInlineImageTarget(target)
		if !ok {
			return match
		}

		rewrittenPath, copied := copyImageToTemp(imgPath, sourceDir, imagesDir, filenamePrefix, usedNames)
		if !copied {
			return match
		}

		// Rewrite the Markdown path to be relative to the preprocessed file,
		// preserving any optional inline title.
		return fmt.Sprintf("![%s](%s%s)", alt, rewrittenPath, suffix)
	})

	imageReferenceLabels := collectImageReferenceLabels(result)
	if len(imageReferenceLabels) == 0 {
		return result, nil
	}

	result = referenceDefinitionRegex.ReplaceAllStringFunc(result, func(match string) string {
		submatch := referenceDefinitionRegex.FindStringSubmatch(match)
		if len(submatch) < 5 {
			return match
		}

		indent := submatch[1]
		label := submatch[2]
		destination := submatch[3]
		suffix := submatch[4]
		if !imageReferenceLabels[normalizeReferenceLabel(label)] {
			return match
		}

		imgPath := strings.Trim(destination, "<>")
		rewrittenPath, copied := copyImageToTemp(imgPath, sourceDir, imagesDir, filenamePrefix, usedNames)
		if !copied {
			return match
		}

		return fmt.Sprintf("%s[%s]: %s%s", indent, label, rewrittenPath, suffix)
	})

	return result, nil
}

// splitInlineImageTarget separates the destination path from an optional title
// in an inline Markdown image target. For example, `./img.png "title"`
// becomes path `./img.png` and suffix ` "title"`.
func splitInlineImageTarget(target string) (string, string, bool) {
	trimmed := strings.TrimSpace(target)
	if trimmed == "" {
		return "", "", false
	}

	if strings.HasPrefix(trimmed, "<") {
		end := strings.Index(trimmed, ">")
		if end <= 0 {
			return "", "", false
		}
		return trimmed[1:end], trimmed[end+1:], true
	}

	for i, r := range trimmed {
		if r == ' ' || r == '\t' || r == '\r' || r == '\n' {
			return trimmed[:i], trimmed[i:], true
		}
	}
	return trimmed, "", true
}

func collectImageReferenceLabels(body string) map[string]bool {
	labels := map[string]bool{}
	for _, match := range referenceImageRegex.FindAllStringSubmatch(body, -1) {
		if len(match) < 3 {
			continue
		}
		label := match[2]
		if label == "" {
			label = match[1]
		}
		labels[normalizeReferenceLabel(label)] = true
	}
	return labels
}

func normalizeReferenceLabel(label string) string {
	return strings.ToLower(strings.Join(strings.Fields(label), " "))
}

func copyImageToTemp(imgPath string, sourceDir string, imagesDir string, filenamePrefix string, usedNames map[string]bool) (string, bool) {
	// Skip URLs and absolute paths.
	if isURL(imgPath) || filepath.IsAbs(imgPath) {
		return "", false
	}

	// Resolve relative to the original Markdown file's directory.
	resolvedPath := filepath.Join(sourceDir, imgPath)
	resolvedPath = filepath.Clean(resolvedPath)

	// Check if the file exists.
	info, err := os.Stat(resolvedPath)
	if err != nil || info.IsDir() {
		// Image not found: leave unchanged, don't error.
		return "", false
	}

	// Determine a unique filename in the images directory. Prefixing is
	// important for bundle mode, where this function is called once per
	// input file while reusing the same tmpDir/images directory.
	base := filepath.Base(resolvedPath)
	destName := filenamePrefix + base
	if usedNames[destName] {
		ext := filepath.Ext(destName)
		stem := strings.TrimSuffix(destName, ext)
		for i := 1; usedNames[destName]; i++ {
			destName = fmt.Sprintf("%s-%d%s", stem, i, ext)
		}
	}
	usedNames[destName] = true

	destPath := filepath.Join(imagesDir, destName)

	// Copy the image file.
	if err := copyFile(resolvedPath, destPath); err != nil {
		return "", false
	}

	return "./images/" + destName, true
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
	defer func() { _ = srcFile.Close() }()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %q: %w", dst, err)
	}
	defer func() { _ = dstFile.Close() }()

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
