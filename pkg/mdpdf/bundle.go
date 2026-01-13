package mdpdf

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
)

type BundleInput struct {
	Path  string
	Title string
}

func BuildBundleMarkdown(inputs []BundleInput) (string, error) {
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
