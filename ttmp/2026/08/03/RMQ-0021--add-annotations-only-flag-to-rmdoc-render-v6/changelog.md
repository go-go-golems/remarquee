# Changelog

## 2026-08-03

- Initial workspace created


## 2026-08-03

Created ticket; mapped both render verbs, V6 render pipeline, and rmapi legacy annotations-only semantics; reproduced motivating failure on TTC Garden Human Calibration Workbook (render-legacy: Error: Unknown header; render-v6 works on pages 1-3); wrote intern-ready analysis/design/implementation guide with pseudocode, diagrams, DR-1..DR-5, 4-phase plan, fixture-based test matrix

### Related Files

- /home/manuel/workspaces/2026-08-03/remarquee-v6-only-annotations/remarquee/ttmp/2026/08/03/RMQ-0021--add-annotations-only-flag-to-rmdoc-render-v6/design-doc/01-analysis-design-and-implementation-guide-for-render-v6-annotations-only.md — Primary deliverable
- /home/manuel/workspaces/2026-08-03/remarquee-v6-only-annotations/remarquee/ttmp/2026/08/03/RMQ-0021--add-annotations-only-flag-to-rmdoc-render-v6/reference/01-diary.md — Investigation diary steps 1-3


## 2026-08-03

Verified workbook pages 1-3 visually (read tool); updated worked example with concrete annotation inventory; doctor passed; committed ticket docs (db38ea1); uploaded bundle to reMarkable /ai/2026/08/03/RMQ-0021 and verified remote listing

### Related Files

- /home/manuel/workspaces/2026-08-03/remarquee-v6-only-annotations/remarquee/ttmp/2026/08/03/RMQ-0021--add-annotations-only-flag-to-rmdoc-render-v6/reference/01-diary.md — Diary step 4


## 2026-08-03

Implemented --annotations-only for render-v6 and render-v6-png across 4 phases: helper extraction (byte-identical, MD5-verified), stdlib-only annotations-only renderer (DR-6, no unipdf watermark), CLI wiring with legacy-parity page semantics, README + full-suite validation (commits 0be07ed, 796fd6d, 1594eae)

### Related Files

- /home/manuel/workspaces/2026-08-03/remarquee-v6-only-annotations/remarquee/cmd/remarquee/cmds/rmdoc/render_v6.go — --annotations-only flag
- /home/manuel/workspaces/2026-08-03/remarquee-v6-only-annotations/remarquee/cmd/remarquee/cmds/rmdoc/render_v6_png.go — --annotations-only flag (Phase 3b)
- /home/manuel/workspaces/2026-08-03/remarquee-v6-only-annotations/remarquee/pkg/rmdoc/render/v6_annotations_only.go — New renderer
- /home/manuel/workspaces/2026-08-03/remarquee-v6-only-annotations/remarquee/pkg/rmdoc/render/v6_annotations_only_pdf.go — Stdlib PDF writer (DR-6)


## 2026-08-03

Re-uploaded updated design guide + diary bundle (DR-6, Phases 1-4, diary steps 5-9) to reMarkable /ai/2026/08/03/RMQ-0021 after implementation completed


## 2026-08-05

PR #21 review fixes: WinAnsi/CP1252 typed-text encoding with WinAnsiEncoding font dicts (no more mojibake); smart-highlight Y translation aligned with stroke canvas; render-v6-png switched to range-aware parsePageSelection1Based; documented latent merge-path highlight misalignment as follow-up open question

### Related Files

- /home/manuel/workspaces/2026-08-03/remarquee-v6-only-annotations/remarquee/cmd/remarquee/cmds/rmdoc/render_v6_png.go — range-aware page parser
- /home/manuel/workspaces/2026-08-03/remarquee-v6-only-annotations/remarquee/pkg/rmdoc/render/v6_annotations_only.go — encodeWinAnsiText + yTranslation

