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
// preprocessed individually: YAML frontmatter is stripped, HTML SVG <img> tags
// are rewritten, local image paths are resolved (copied into tmpDir/images/),
// inline <svg> blocks are converted, and Mermaid blocks are rendered to images
// (if the corresponding configs are provided).
//
// The resulting body can be passed to ConvertMarkdownFileToPDF, which
// will find the pre-resolved images via its own image resolution step.
func BuildBundleMarkdown(ctx context.Context, inputs []BundleInput, tmpDir string, mermaidCfg *MermaidRendererConfig, svgCfg *SVGRendererConfig, resolveImages bool) (string, error) {
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

		assetPrefix := fmt.Sprintf("bundle-%03d-", i+1)

		// Rewrite SVG <img> tags before image path resolution so the resulting
		// Markdown references are copied from this input's source directory.
		body, err = ResolveHTMLImages(body, svgConfigWithImagePrefix(svgCfg, assetPrefix))
		if err != nil {
			return "", errors.Wrapf(err, "failed to resolve HTML SVG images for %s", in.Path)
		}

		if resolveImages {
			// Resolve local image paths relative to this input's source directory.
			// Prefix filenames by bundle input so same-basename images from
			// different files cannot overwrite each other in tmpDir/images.
			sourceDir := filepath.Dir(in.Path)
			body, err = ResolveImagePathsWithPrefix(body, sourceDir, tmpDir, assetPrefix)
			if err != nil {
				return "", errors.Wrapf(err, "failed to resolve image paths for %s", in.Path)
			}
		}

		// Extract and convert inline <svg> blocks. Writes finished assets directly
		// into tmpDir/images, so it runs after image path resolution. Uses the
		// per-input prefix to avoid svg-001 collisions across files.
		body, err = ResolveInlineSVGBlocks(ctx, body, tmpDir, svgConfigWithImagePrefix(svgCfg, assetPrefix))
		if err != nil {
			return "", errors.Wrapf(err, "failed to resolve inline SVG blocks for %s", in.Path)
		}

		// Render Mermaid blocks for this input. Use the same per-input
		// filename prefix to avoid mermaid-001.png collisions across files.
		body, err = RenderMermaidBlocks(ctx, body, tmpDir, mermaidConfigWithImagePrefix(mermaidCfg, assetPrefix))
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
