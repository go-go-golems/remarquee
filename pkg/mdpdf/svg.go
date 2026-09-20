package mdpdf

import (
	"fmt"
	"io"
	"os"
	"os/exec"
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
