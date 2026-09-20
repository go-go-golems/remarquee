package mdpdf

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SVGRendererConfig controls how SVG content is detected, extracted and made
// renderable for pandoc/XeLaTeX.
//
// Referenced SVG files (e.g. `![x](./a.svg)`) and `data:image/svg+xml` URIs are
// already rendered by pandoc itself via rsvg-convert, so this config primarily
// governs the parts pandoc cannot handle: inline raw `<svg>` blocks and HTML
// `<img src="...svg">` tags.
//
// All failures are non-fatal. When no converter is available, or a block cannot
// be converted, the source is left unchanged and a warning is emitted rather
// than aborting document conversion.
type SVGRendererConfig struct {
	// Enabled turns the whole inline-SVG feature on/off. Default: true.
	Enabled bool

	// ConverterPath overrides converter auto-detection. When set, it must point
	// to an existing executable.
	ConverterPath string

	// ConverterOrder is the preference list used when ConverterPath is empty.
	// Default: ["rsvg-convert", "inkscape"].
	ConverterOrder []string

	// DefaultWidth is applied to converted inline SVG images as a pandoc
	// `{width=...}` attribute (for example "70%", "12cm"). Empty means use the
	// image's natural size.
	DefaultWidth string

	// MaxWidth caps synthesized widths (same syntax as DefaultWidth). Optional.
	MaxWidth string

	// ImagePrefix is prepended to generated SVG filenames. Bundle generation
	// sets this per input file so repeated svg-001 names do not collide.
	ImagePrefix string

	// WarnWriter receives non-fatal warnings. Default: os.Stderr.
	WarnWriter io.Writer
}

// DefaultSVGRendererConfig returns sensible defaults. The feature is enabled,
// but conversion degrades gracefully when no converter is installed.
func DefaultSVGRendererConfig() SVGRendererConfig {
	return SVGRendererConfig{
		Enabled:        true,
		ConverterOrder: []string{"rsvg-convert", "inkscape"},
		WarnWriter:     os.Stderr,
	}
}

// warnf writes a non-fatal SVG warning to the configured writer.
func (c *SVGRendererConfig) warnf(format string, args ...any) {
	w := c.WarnWriter
	if w == nil {
		w = os.Stderr
	}
	fmt.Fprintf(w, "WARNING: SVG: "+format+"\n", args...)
}

// ResolveSVGConverter finds a usable SVG converter.
//
// It returns ("", nil) when the feature is disabled or no converter is found;
// that is not an error because callers must degrade gracefully rather than abort
// document conversion. An error is only returned when an explicitly configured
// ConverterPath does not exist.
func ResolveSVGConverter(cfg *SVGRendererConfig) (string, error) {
	if cfg == nil || !cfg.Enabled {
		return "", nil
	}

	if cfg.ConverterPath != "" {
		if _, err := os.Stat(cfg.ConverterPath); err == nil {
			return cfg.ConverterPath, nil
		}
		return "", fmt.Errorf("configured svg converter %q not found", cfg.ConverterPath)
	}

	order := cfg.ConverterOrder
	if len(order) == 0 {
		order = []string{"rsvg-convert", "inkscape"}
	}
	for _, name := range order {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", nil
}

// svgConfigWithImagePrefix clones cfg with an added filename prefix, mirroring
// mermaidConfigWithImagePrefix. It returns nil when cfg is nil.
func svgConfigWithImagePrefix(cfg *SVGRendererConfig, prefix string) *SVGRendererConfig {
	if cfg == nil {
		return nil
	}
	cloned := *cfg
	cloned.ImagePrefix = prefix + cloned.ImagePrefix
	return &cloned
}

// looksLikeSVGOpen reports whether a line begins with an <svg ...> opening tag
// (after optional leading whitespace). The character after "svg" must be a tag
// boundary so that <svgfoo> is not matched.
func looksLikeSVGOpen(line string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	if !hasPrefixFold(trimmed, "<svg") {
		return false
	}
	return isTagBoundary(trimmed, len("<svg"))
}

// hasPrefixFold reports whether s begins with prefix, case-insensitively.
func hasPrefixFold(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	return strings.EqualFold(s[:len(prefix)], prefix)
}

// isTagBoundary reports whether the byte at idx is a valid tag boundary
// (whitespace, '>', '/', or end of input).
func isTagBoundary(s string, idx int) bool {
	if idx >= len(s) {
		return true
	}
	switch s[idx] {
	case ' ', '\t', '\r', '\n', '>', '/':
		return true
	default:
		return false
	}
}

// countTagDelta counts net <svg>/</svg> depth on a single line, ignoring tag-like
// text inside quoted attribute values and treating self-closing <svg .../> as
// depth-neutral.
func countTagDelta(line, tag string) int {
	delta := 0
	i := 0
	n := len(line)
	for i < n {
		c := line[i]
		if c == '"' || c == '\'' {
			// Skip a quoted attribute value so '>' inside it is ignored.
			quote := c
			i++
			for i < n && line[i] != quote {
				i++
			}
			if i < n {
				i++
			}
			continue
		}
		if c != '<' {
			i++
			continue
		}

		// Closing tag: </svg...
		if hasPrefixFold(line[i+1:], "/"+tag) && isTagBoundary(line, i+1+1+len(tag)) {
			delta--
			for i < n && line[i] != '>' {
				i++
			}
			if i < n {
				i++
			}
			continue
		}

		// Opening tag: <svg...
		if hasPrefixFold(line[i+1:], tag) && isTagBoundary(line, i+1+len(tag)) {
			end, selfClosing := tagEnd(line, i)
			if !selfClosing {
				delta++
			}
			i = end
			if i < n {
				i++
			}
			continue
		}

		i++
	}
	return delta
}

// tagEnd returns the index of the closing '>' of the tag starting at start, or
// len(line) if there is none. It reports whether the tag is self-closing and
// skips quoted attribute values.
func tagEnd(line string, start int) (int, bool) {
	n := len(line)
	i := start
	var quote byte
	for i < n {
		c := line[i]
		if quote != 0 {
			if c == quote {
				quote = 0
			}
			i++
			continue
		}
		switch c {
		case '"', '\'':
			quote = c
			i++
			continue
		case '>':
			return i, i > start && line[i-1] == '/'
		}
		i++
	}
	return n, false
}

// collectSVGBlock collects the lines of an <svg>...</svg> block starting at
// start. It returns the joined text, the index of the closing line, and whether
// a balanced block was found. A fenced code region beginning inside the block
// aborts collection.
func collectSVGBlock(lines []string, protect []bool, start int) (string, int, bool) {
	depth := 0
	buf := make([]string, 0, 4)
	for j := start; j < len(lines); j++ {
		if j > start && protect[j] {
			return "", start, false
		}
		buf = append(buf, lines[j])
		depth += countTagDelta(lines[j], "svg")
		if depth <= 0 {
			return strings.Join(buf, "\n"), j, true
		}
	}
	return "", start, false
}

// convertSVGToPDF runs the resolved converter to produce a vector PDF next to
// the extracted SVG. It supports rsvg-convert and Inkscape argument styles,
// falling back to the rsvg-convert form for unknown tools.
func convertSVGToPDF(ctx context.Context, converter, svgPath, pdfPath string) error {
	base := filepath.Base(converter)
	var args []string
	switch {
	case strings.Contains(base, "rsvg-convert"):
		args = []string{"-f", "pdf", "-a", "--dpi-x", "96", "--dpi-y", "96", "-o", pdfPath, svgPath}
	case strings.Contains(base, "inkscape"):
		args = []string{svgPath, "--export-type=pdf", "--export-filename=" + pdfPath}
	default:
		args = []string{"-f", "pdf", "-o", pdfPath, svgPath}
	}

	cmd := exec.CommandContext(ctx, converter, args...) // #nosec G204 -- explicit argv, no shell.
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s failed: %s: %w", base, strings.TrimSpace(string(out)), err)
	}
	if _, err := os.Stat(pdfPath); err != nil {
		return fmt.Errorf("%s did not produce %q: %w", base, pdfPath, err)
	}
	return nil
}

// ResolveInlineSVGBlocks finds raw <svg>...</svg> blocks outside fenced code,
// converts each to a vector PDF via the resolved converter, and replaces it
// with a Markdown image reference.
//
// It is non-fatal: on any problem it warns and leaves the source unchanged, so a
// single bad SVG never aborts document conversion. When no converter is
// available the body is returned untouched with one warning.
func ResolveInlineSVGBlocks(ctx context.Context, body, tmpDir string, cfg *SVGRendererConfig) (string, error) {
	if cfg == nil || !cfg.Enabled {
		return body, nil
	}

	converter, err := ResolveSVGConverter(cfg)
	if err != nil {
		return body, err
	}
	if converter == "" {
		if containsInlineSVG(body) {
			cfg.warnf("no SVG converter found; inline <svg> blocks left as-is")
		}
		return body, nil
	}

	lines := strings.Split(body, "\n")
	protect := literalLines(lines)
	out := make([]string, 0, len(lines))
	imagesDir := filepath.Join(tmpDir, "images")
	counter := 0

	for i := 0; i < len(lines); {
		if !protect[i] && looksLikeSVGOpen(lines[i]) {
			svgText, endIdx, ok := collectSVGBlock(lines, protect, i)
			if !ok {
				cfg.warnf("unclosed <svg> block starting at line %d; left as-is", i+1)
				out = append(out, lines[i])
				i++
				continue
			}

			counter++
			name := fmt.Sprintf("%ssvg-%03d", cfg.ImagePrefix, counter)
			if mkErr := os.MkdirAll(imagesDir, 0o755); mkErr != nil {
				return body, fmt.Errorf("failed to create SVG images directory: %w", mkErr)
			}
			svgPath := filepath.Join(imagesDir, name+".svg")
			pdfPath := filepath.Join(imagesDir, name+".pdf")
			if wErr := os.WriteFile(svgPath, []byte(svgText), 0o644); wErr != nil {
				return body, fmt.Errorf("failed to write extracted SVG: %w", wErr)
			}
			if cErr := convertSVGToPDF(ctx, converter, svgPath, pdfPath); cErr != nil {
				cfg.warnf("failed to convert inline <svg> block %d: %v; left as-is", counter, cErr)
				out = append(out, lines[i:endIdx+1]...)
				i = endIdx + 1
				continue
			}

			alt := fmt.Sprintf("svg image %d", counter)
			if cfg.DefaultWidth != "" {
				out = append(out, fmt.Sprintf("![%s](./images/%s.pdf){width=%s}", alt, name, cfg.DefaultWidth))
			} else {
				out = append(out, fmt.Sprintf("![%s](./images/%s.pdf)", alt, name))
			}
			i = endIdx + 1
			continue
		}

		out = append(out, lines[i])
		i++
	}

	return strings.Join(out, "\n"), nil
}

// containsInlineSVG reports whether body contains a line that starts an <svg>
// block. It is a cheap check used only to decide whether to emit a warning.
func containsInlineSVG(body string) bool {
	for _, line := range strings.Split(body, "\n") {
		if looksLikeSVGOpen(line) {
			return true
		}
	}
	return false
}

// ResolveHTMLImages rewrites HTML <img src="*.svg" ...> tags into Markdown image
// references. Only SVG sources are handled; other <img> tags are left unchanged.
// Fenced code regions are skipped.
//
// Like the rest of the SVG handling it is non-fatal: when no converter is
// available the body is returned untouched with a single warning, because a
// rewritten SVG reference would otherwise hard-fail at pandoc.
func ResolveHTMLImages(body string, cfg *SVGRendererConfig) (string, error) {
	if cfg == nil || !cfg.Enabled {
		return body, nil
	}

	converter, err := ResolveSVGConverter(cfg)
	if err != nil {
		return body, err
	}
	if converter == "" {
		if containsHTMLSVGImg(body) {
			cfg.warnf("no SVG converter found; HTML <img src=*.svg> tags left as-is")
		}
		return body, nil
	}

	lines := strings.Split(body, "\n")
	protect := literalLines(lines)
	for i, line := range lines {
		if protect[i] {
			continue
		}
		lines[i] = rewriteHTMLImgTags(line, cfg)
	}
	return strings.Join(lines, "\n"), nil
}

// rewriteHTMLImgTags rewrites every SVG <img> tag on a single line.
func rewriteHTMLImgTags(line string, cfg *SVGRendererConfig) string {
	var b strings.Builder
	i := 0
	n := len(line)
	for i < n {
		idx := indexImgTag(line, i)
		if idx < 0 {
			b.WriteString(line[i:])
			break
		}
		b.WriteString(line[i:idx])
		end, _ := tagEnd(line, idx)
		if end >= n {
			// Unterminated tag on this line: leave it.
			b.WriteString(line[idx:])
			break
		}
		tag := line[idx : end+1]
		if repl, ok := htmlImgToMarkdown(tag, cfg); ok {
			b.WriteString(repl)
		} else {
			b.WriteString(tag)
		}
		i = end + 1
	}
	return b.String()
}

// indexImgTag returns the index of the next <img tag at or after from, or -1.
// The character after "img" must be a tag boundary.
func indexImgTag(s string, from int) int {
	for i := from; i < len(s); i++ {
		if s[i] != '<' {
			continue
		}
		if hasPrefixFold(s[i+1:], "img") && isTagBoundary(s, i+1+len("img")) {
			return i
		}
	}
	return -1
}

// htmlImgToMarkdown converts a single <img> tag to Markdown image syntax when it
// references an SVG. It returns ok=false for non-SVG or malformed tags.
func htmlImgToMarkdown(tag string, cfg *SVGRendererConfig) (string, bool) {
	attrs := parseTagAttrs(tag, "img")
	src := attrs["src"]
	if src == "" || !isSVGSource(src) {
		return "", false
	}
	alt := attrs["alt"]
	if alt == "" {
		alt = "svg image"
	}
	width := normalizeHTMLDimension(attrs["width"])
	if width == "" {
		width = normalizeHTMLDimension(attrs["height"])
	}
	if width == "" {
		width = cfg.DefaultWidth
	}
	if width != "" {
		return fmt.Sprintf("![%s](%s){width=%s}", alt, src, width), true
	}
	return fmt.Sprintf("![%s](%s)", alt, src), true
}

// parseTagAttrs extracts lower-cased attribute name/value pairs from a start tag
// such as `<img src="x" width='100'/>`.
func parseTagAttrs(tag, tagName string) map[string]string {
	attrs := map[string]string{}
	n := len(tag)
	i := 1 + len(tagName)
	limit := n
	if limit > 0 && tag[limit-1] == '>' {
		limit--
	}
	if limit > 0 && tag[limit-1] == '/' {
		limit--
	}
	for i < limit {
		for i < limit && isHTMLSpace(tag[i]) {
			i++
		}
		if i >= limit {
			break
		}
		start := i
		for i < limit && !isHTMLSpace(tag[i]) && tag[i] != '=' {
			i++
		}
		name := strings.ToLower(tag[start:i])
		for i < limit && (isHTMLSpace(tag[i]) || tag[i] == '=') {
			i++
		}
		var value string
		if i < limit && (tag[i] == '"' || tag[i] == '\'') {
			quote := tag[i]
			i++
			vs := i
			for i < limit && tag[i] != quote {
				i++
			}
			value = tag[vs:i]
			if i < limit {
				i++
			}
		} else {
			vs := i
			for i < limit && !isHTMLSpace(tag[i]) {
				i++
			}
			value = tag[vs:i]
		}
		if name != "" {
			attrs[name] = value
		}
	}
	return attrs
}

func isHTMLSpace(c byte) bool {
	switch c {
	case ' ', '\t', '\r', '\n':
		return true
	default:
		return false
	}
}

// normalizeHTMLDimension adds a px unit to bare numeric dimensions and preserves
// explicit units such as "50%" or "10cm".
func normalizeHTMLDimension(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	for i := 0; i < len(v); i++ {
		if v[i] < '0' || v[i] > '9' {
			return v
		}
	}
	return v + "px"
}

// isSVGSource reports whether a URL path refers to an SVG file, ignoring any
// query string or fragment.
func isSVGSource(src string) bool {
	s := src
	if i := strings.IndexAny(s, "?#"); i >= 0 {
		s = s[:i]
	}
	return strings.HasSuffix(strings.ToLower(s), ".svg")
}

// containsHTMLSVGImg reports whether body plausibly contains an HTML <img> tag
// referencing an SVG. It is a cheap check used only to decide whether to warn.
func containsHTMLSVGImg(body string) bool {
	return indexImgTag(body, 0) >= 0 && strings.Contains(strings.ToLower(body), ".svg")
}
