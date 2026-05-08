package upload

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

func NewUploadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload content to a reMarkable device",
	}

	cmd.AddCommand(NewUploadMarkdownCommand())
	cmd.AddCommand(NewUploadBundleCommand())
	cmd.AddCommand(NewUploadSourceCommand())
	return cmd
}

// sanitizePDFName replaces characters that commonly cause rmapi upload failures
// (HTTP 400) with safe alternatives. Spaces become underscores, parentheses
// and other problematic characters are removed.
//
// This was added after transcript analysis showed that agents spend 3-5 extra
// tool calls retrying uploads when PDF filenames contain spaces or special chars
// that the reMarkable cloud API rejects.
func sanitizePDFName(name string) string {
	base := filepath.Base(name)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)

	// Replace spaces with underscores (the most common 400 trigger).
	stem = strings.ReplaceAll(stem, " ", "_")

	// Remove characters that rmapi/cloud sync rejects or mangles.
	// Keep: alphanumeric, underscores, dashes, dots.
	re := regexp.MustCompile(`[^a-zA-Z0-9_.\-]`)
	stem = re.ReplaceAllString(stem, "")

	// Collapse multiple underscores.
	for strings.Contains(stem, "__") {
		stem = strings.ReplaceAll(stem, "__", "_")
	}

	// Strip leading/trailing underscores and dashes.
	stem = strings.Trim(stem, "_-")

	if stem == "" {
		stem = "document"
	}

	return stem + ext
}

// sanitizeAndCheckOutputPath applies sanitizePDFName to the filename portion of
// outPDF and returns the cleaned path. If the name was sanitized, prints a
// notice so the agent knows the upload target differs from the input name.
func sanitizeAndCheckOutputPath(outPDF string) string {
	dir := filepath.Dir(outPDF)
	origBase := filepath.Base(outPDF)
	cleanBase := sanitizePDFName(origBase)

	if origBase != cleanBase {
		fmt.Fprintf(os.Stderr, "NOTE: sanitized PDF filename: %q -> %q\n", origBase, cleanBase)
	}

	return filepath.Join(dir, cleanBase)
}
