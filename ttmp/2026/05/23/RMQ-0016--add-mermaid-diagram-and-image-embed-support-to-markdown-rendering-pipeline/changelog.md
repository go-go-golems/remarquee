# Changelog

## 2026-05-23

- Initial workspace created
- Created design document: design/01-mermaid-image-embed-design.md — comprehensive intern-ready guide covering current architecture, gap analysis, proposed solution, pseudocode, phased implementation plan, test strategy, and API reference
- Created investigation diary: reference/01-diary.md — Steps 1-7: full investigation and implementation
- Related key source files: pkg/mdpdf/pandoc.go, pkg/mdpdf/bundle.go, pkg/mdpdf/preprocess.go, cmd/.../upload/md.go, cmd/.../upload/bundle.go
- Updated index.md with summary and overview
- Updated tasks.md with phased implementation checklist
- Recorded open question decisions: --mermaid-width yes, no upload src mermaid, no bundle dedup
- Uploaded design doc bundle to reMarkable at /ai/2026/05/23/RMQ-0016
- Phase 1 (6a960c9): Image path resolution — pkg/mdpdf/images.go, 10 unit tests + integration test
- Phase 2 (162a75c): Mermaid block rendering — pkg/mdpdf/mermaid.go, 12 tests, MermaidRendererConfig
- Phase 3 (2ccac3e): CLI flags — 7 new flags on upload md/bundle, mermaidFlags helper type
- Phase 4 (5606cb8): Bundle mode — per-file image/mermaid preprocessing in BuildBundleMarkdown
- Phase 5 (cd216c6): README updated with mermaid/image examples

## 2026-05-23: Fix image resolver for valid Markdown image forms

- Updated `pkg/mdpdf/images.go` so local image copying/rewrite handles inline image titles like `![alt](./img.png "title")`.
- Added support for reference-style image definitions used by `![alt][id]` and collapsed `![alt][]` references.
- Preserved non-image reference definitions unchanged.
- Added regression tests in `pkg/mdpdf/images_test.go`.
