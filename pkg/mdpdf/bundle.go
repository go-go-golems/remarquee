package mdpdf

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

type BundleInput struct {
	Path  string
	Title string
}

// BuildBundleMarkdown concatenates multiple Markdown inputs into a single
// document with stable section headings and page breaks. Each input is
// preprocessed individually: YAML frontmatter is stripped, local image
// paths are resolved (copied into tmpDir/images/), and Mermaid blocks
// are rendered to images (if mermaidCfg is provided).
//
// The resulting body can be passed to ConvertMarkdownFileToPDF, which
// will find the pre-resolved images via its own image resolution step.
func BuildBundleMarkdown(ctx context.Context, inputs []BundleInput, tmpDir string, mermaidCfg *MermaidRendererConfig) (string, error) {
	var b strings.Builder

	for i, in := range inputs {
		if strings.TrimSpace(in.Path) == "" {
			return "", errors.New("bundle input path is empty")
		}
		title := strings.TrimSpace(in.Title)
		if title == "" {
			title = in.Path
		}

		mdBytes, err := os.ReadFile(in.Path)
		if err != nil {
			return "", errors.Wrapf(err, "failed to read markdown file: %s", in.Path)
		}
		body := StripYAMLFrontmatter(string(mdBytes))

		// Resolve local image paths relative to this input's source directory.
		sourceDir := filepath.Dir(in.Path)
		body, err = ResolveImagePaths(body, sourceDir, tmpDir)
		if err != nil {
			return "", errors.Wrapf(err, "failed to resolve image paths for %s", in.Path)
		}

		// Render Mermaid blocks for this input.
		body, err = RenderMermaidBlocks(ctx, body, tmpDir, mermaidCfg)
		if err != nil {
			// Non-fatal: mermaid rendering errors are logged per-block.
			_ = err
		}

		body = NormalizeListSpacing(body)

		// Stable section heading for predictable ToC entries.
		fmt.Fprintf(&b, "# %s\n\n", title)
		b.WriteString(body)
		if !strings.HasSuffix(body, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("\n")

		// Insert a page break between documents for readability.
		if i < len(inputs)-1 {
			b.WriteString("```{=latex}\n\\newpage\n```\n\n")
		}
	}

	return b.String(), nil
}
