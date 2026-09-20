# Changelog

## 2026-09-20

- Initial workspace created

## 2026-09-20

Created ticket and wrote the intern-facing SVG design/implementation guide. Empirical baseline corrected the original scope: referenced SVG and data-URI SVG already render via pandoc+rsvg-convert; inline raw <svg> and HTML <img src=svg> are dropped; missing rsvg-convert hard-fails XeLaTeX.

### Related Files

- /home/manuel/code/wesen/go-go-golems/remarquee/ttmp/2026/09/20/RMQ-0024--svg-image-support-in-the-markdown-to-pdf-pipeline/design-doc/01-svg-image-support-design-and-implementation-guide-for-a-new-intern.md — Main deliverable: system tour, behavior matrix, design, pseudocode, API refs, phased plan
- /home/manuel/code/wesen/go-go-golems/remarquee/ttmp/2026/09/20/RMQ-0024--svg-image-support-in-the-markdown-to-pdf-pipeline/reference/01-implementation-diary.md — Evidence diary and corrected-scope rationale

## 2026-09-20

Uploaded the SVG design/implementation guide (bundled with the evidence diary) to reMarkable at /ai/2026/09/20/RMQ-0024 as 'RMQ-0024 SVG Image Support Guide.pdf'.

## 2026-09-20

Implemented SVG support end-to-end: SVGRendererConfig + converter discovery (P1), inline <svg> extraction/conversion (P2), HTML <img src=*.svg> rewriting (P3), direct+bundle pipeline wiring (P4), and --svg/--svg-converter/--svg-default-width CLI flags on md/bundle/sync (P5). Validated on real PDFs: inline raw SVG now renders (red pixels 0 -> 3044), referenced+HTML render, fenced code preserved, missing-converter warns and continues, bundle prefixing works. All tests pass.

### Related Files

- /home/manuel/code/wesen/go-go-golems/remarquee/cmd/remarquee/cmds/upload/svg_section.go — CLI SVG flags section
- /home/manuel/code/wesen/go-go-golems/remarquee/pkg/mdpdf/bundle.go — Bundle wiring with per-input prefixing
- /home/manuel/code/wesen/go-go-golems/remarquee/pkg/mdpdf/pandoc.go — Pipeline wiring and default SVGRendererConfig
- /home/manuel/code/wesen/go-go-golems/remarquee/pkg/mdpdf/svg.go — SVG config, converter discovery, inline extraction, HTML img rewriting
