package mdpdf

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// MermaidRendererConfig controls how Mermaid code blocks are rendered to images.
type MermaidRendererConfig struct {
	// MmdcPath is the path to the mmdc binary. If empty, "mmdc" is looked up
	// in $PATH. If not found, Mermaid blocks are left as-is with a warning.
	MmdcPath string

	// Enabled controls whether Mermaid rendering is attempted at all.
	// Default: true. Set to false to skip even if mmdc is available.
	Enabled bool

	// Scale controls the pixel scale of rendered diagrams (1x, 2x, 3x).
	// Default: 2 (good balance of quality and file size for PDF embedding).
	Scale int

	// BackgroundColor sets the diagram background. Default: "white".
	BackgroundColor string

	// Width sets the maximum diagram width in pixels. 0 = auto (use Mermaid default).
	Width int

	// Theme sets the Mermaid theme. Default: "default". Options: "default",
	// "dark", "forest", "neutral".
	Theme string

	// NoSandbox passes --no-sandbox to the Puppeteer/Chromium process.
	// Required on systems where the Chrome sandbox is unavailable (e.g.
	// Ubuntu 23.10+ with AppArmor userns restrictions, or CI containers).
	NoSandbox bool

	// PDFWidth sets the display width for mermaid images in the PDF.
	// Uses pandoc attribute syntax: "50%", "400px", "10cm", etc.
	// Empty = no constraint (pandoc default = fill page width).
	PDFWidth string
}

// DefaultMermaidRendererConfig returns sensible defaults.
func DefaultMermaidRendererConfig() MermaidRendererConfig {
	return MermaidRendererConfig{
		Enabled:         true,
		Scale:           2,
		BackgroundColor: "white",
		Theme:           "default",
	}
}

// mermaidBlockRegex matches ```mermaid ... ``` fenced code blocks.
// It uses (?s) so that . matches newlines, and captures the Mermaid source.
var mermaidBlockRegex = regexp.MustCompile(
	"(?s)" +
		"```mermaid\\s*\\n" +
		"(.*?)" +
		"\\n\\s*```",
)

// RenderMermaidBlocks finds Mermaid fenced code blocks in the Markdown body,
// renders each one to a PNG using mmdc, and replaces the block with an image
// embed. If mmdc is not available, blocks are left unchanged.
//
// Parameters:
//   - ctx: context for cancellation of mmdc subprocess
//   - body: the Markdown content (after YAML stripping and image resolution)
//   - tmpDir: the temp directory for writing rendered PNGs
//   - config: Mermaid renderer configuration
//
// Returns the rewritten Markdown body.
func RenderMermaidBlocks(ctx context.Context, body string, tmpDir string, config *MermaidRendererConfig) (string, error) {
	if config == nil || !config.Enabled {
		return body, nil
	}

	mmdcPath, err := resolveMmdcPath(config.MmdcPath)
	if err != nil {
		// mmdc not found: leave blocks as-is. This is not an error —
		// the user may not have mermaid-cli installed.
		return body, nil
	}

	imagesDir := filepath.Join(tmpDir, "images")
	if err := os.MkdirAll(imagesDir, 0o755); err != nil {
		return body, fmt.Errorf("failed to create images directory for mermaid: %w", err)
	}

	counter := 0
	result := mermaidBlockRegex.ReplaceAllStringFunc(body, func(match string) string {
		counter++
		submatch := mermaidBlockRegex.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		mermaidSource := strings.TrimSpace(submatch[1])
		if mermaidSource == "" {
			return match // empty block, skip
		}

		imgFilename := fmt.Sprintf("mermaid-%03d.png", counter)
		imgPath := filepath.Join(imagesDir, imgFilename)

		if err := renderMermaidToPNG(ctx, mmdcPath, mermaidSource, imgPath, config); err != nil {
			// Per-block error: warn and leave as-is so one bad diagram
			// doesn't break the whole document.
			fmt.Fprintf(os.Stderr, "WARNING: failed to render Mermaid block %d: %v\n", counter, err)
			return match
		}

		if config.PDFWidth != "" {
			return fmt.Sprintf("![mermaid diagram %d](./images/%s){width=%s}", counter, imgFilename, config.PDFWidth)
		}

		return fmt.Sprintf("![mermaid diagram %d](./images/%s)", counter, imgFilename)
	})

	return result, nil
}

// renderMermaidToPNG writes the Mermaid source to a temp .mmd file and
// invokes mmdc to render it to a PNG at the specified output path.
func renderMermaidToPNG(ctx context.Context, mmdcPath string, source string, outPath string, config *MermaidRendererConfig) error {
	tmpDir, err := os.MkdirTemp("", "remarquee-mermaid-")
	if err != nil {
		return fmt.Errorf("failed to create temp dir for mermaid: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	mmdFile := filepath.Join(tmpDir, "diagram.mmd")
	if err := os.WriteFile(mmdFile, []byte(source), 0o644); err != nil {
		return fmt.Errorf("failed to write mermaid source: %w", err)
	}

	args := []string{
		"-i", mmdFile,
		"-o", outPath,
	}
	if config.Scale > 0 {
		args = append(args, "-s", strconv.Itoa(config.Scale))
	}
	if config.BackgroundColor != "" {
		args = append(args, "-b", config.BackgroundColor)
	}
	if config.Theme != "" {
		args = append(args, "-t", config.Theme)
	}
	if config.Width > 0 {
		args = append(args, "-w", strconv.Itoa(config.Width))
	}

	// If NoSandbox is set, write a Puppeteer config file and pass it to mmdc.
	if config.NoSandbox {
		puppeteerConfig := filepath.Join(tmpDir, "puppeteer.json")
		puppeteerContent := `{"args": ["--no-sandbox"]}`
		if err := os.WriteFile(puppeteerConfig, []byte(puppeteerContent), 0o644); err != nil {
			return fmt.Errorf("failed to write puppeteer config: %w", err)
		}
		args = append(args, "--puppeteerConfigFile", puppeteerConfig)
	}

	cmd := exec.CommandContext(ctx, mmdcPath, args...) // #nosec G204
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mmdc failed: %s: %w", strings.TrimSpace(string(output)), err)
	}

	// Verify output file exists.
	if _, err := os.Stat(outPath); err != nil {
		return fmt.Errorf("mmdc output not found at %q: %w", outPath, err)
	}

	return nil
}

// resolveMmdcPath finds the mmdc binary. Returns an error if not found.
func resolveMmdcPath(override string) (string, error) {
	if override != "" {
		if _, err := os.Stat(override); err == nil {
			return override, nil
		}
		return "", fmt.Errorf("mmdc path %q not found", override)
	}
	path, err := exec.LookPath("mmdc")
	if err != nil {
		return "", fmt.Errorf("mmdc not found in $PATH: %w (install with: npm install -g @mermaid-js/mermaid-cli)", err)
	}
	return path, nil
}
