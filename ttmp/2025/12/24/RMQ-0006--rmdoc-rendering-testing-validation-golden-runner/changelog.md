# Changelog

## 2025-12-24

- Initial workspace created


## 2025-12-24

Created analysis document on using remarks for golden testing and PDF comparison. Document covers invocation methods, comparison strategies (visual/structural/hybrid), integration approaches, and best practices. Added comprehensive task list for implementing golden testing infrastructure.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/analysis/01-using-remarks-for-golden-testing-and-pdf-comparison.md — Research document on golden testing approach


## 2025-12-24

Added VLM validation section to analysis document. Added investigation tasks for 5 rendering issues discovered through visual inspection. Performed deep code analysis tracing through codebase to identify root causes. Created diary entry documenting research and analysis work.


## 2025-12-24

Implemented Option B PDF comparison utilities (pure Go): UniDoc-based visual comparison (render pages to images + pixel diff with tolerance + diff PNG) and fast structural comparison (page count, annotation count, extracted text hash). Added unit tests generating small PDFs.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/pdfcmp/pdfcmp.go — New pdfcmp package
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/pdfcmp/pdfcmp_test.go — pdfcmp tests
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/tasks.md — Checked off Option B comparison tasks


## 2025-12-24

Implemented remarks reference runner (Go wrapper around the  CLI): run remarks with context, capture stdout/stderr, handle missing binary via ErrNotFound, and locate generated  outputs (including nested UI-path dirs).

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/refimpl/remarks/runner.go — remarks CLI runner
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/refimpl/remarks/runner_test.go — remarks runner tests
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/tasks.md — Checked off remarks wrapper tasks


## 2025-12-24

(Follow-up) Implemented remarks reference runner (Go wrapper around the remarks CLI): run remarks with context, capture stdout/stderr, handle missing binary via ErrNotFound, and locate generated '* _remarks.pdf' outputs (including nested UI-path dirs).

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/refimpl/remarks/runner.go — remarks CLI runner
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/refimpl/remarks/runner_test.go — remarks runner tests


## 2025-12-24

Added first golden test comparing remarquee V6 render against remarks reference for cmd/remarquee-ui/testdata/Test.rmdoc. Test renders with MergeRMDocV6OntoBackgroundPDF, runs remarks via pkg/refimpl/remarks, compares via pkg/pdfcmp with tolerance, emits diff PNGs on mismatch, and skips when remarks is not on PATH.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/rmdoc/render/golden_remarks_test.go — Golden test using remarks reference
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/tasks.md — Checked off first golden test


## 2025-12-24

Added second V6 golden test (cpage-pdf.rmdoc) comparing remarquee output against remarks reference, using pdfcmp visual comparison with tolerance and diff PNGs on mismatch. Test skips when remarks is not on PATH.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/rmdoc/render/golden_remarks_test.go — Added CpagePdf golden test
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/tasks.md — Checked off CpagePdf golden test


## 2025-12-24

Added legacy golden-style smoke test for legacy-pdf-a4.zip using rmapi PdfGenerator backend. Test validates end-to-end legacy rendering and asserts output PDF page count matches parsed UI page plan.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/rmdoc/render/golden_legacy_rmapi_test.go — Legacy rmapi-backed golden smoke test
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/tasks.md — Checked off legacy test


## 2025-12-24

Added basic golden file management: created cmd/remarquee-ui/testdata/golden/ with README and naming convention, and added go test flag -update-golden so remarks-based golden tests can write/update committed reference PDFs when desired (otherwise fall back to running remarks or skip).

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee-ui/testdata/golden/README.md — Golden directory documentation
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/rmdoc/render/golden_remarks_test.go — Update-golden flag + golden reference selection
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/tasks.md — Checked off golden management tasks


## 2025-12-24

Added rmdoc vlm-validate helper CLI: renders selected PDF pages to PNGs and invokes pinocchio (VLM) for semantic validation/comparison. This is an optional manual tool to complement pixel/structural diffs.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee/cmds/rmdoc/root.go — Registers vlm-validate
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee/cmds/rmdoc/vlm_validate.go — New VLM validation CLI
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/tasks.md — VLM helper task


## 2025-12-24

Added decision analysis doc on how to run remarks locally/CI (Nix flake vs Poetry vs pyenv/pip vs pipx/uv), grounded in remarks repo config (flake.nix + pyproject scripts). Captures tradeoffs and recommended path (Nix for CI).

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/analysis/02-running-remarks-locally-in-ci-nix-vs-poetry-vs-pyenv-decision-doc.md — Decision doc


## 2025-12-24

Created bug report for UniDoc rasterizer failure ('type check error') in vlm-validate and switched vlm-validate to Poppler/pdftoppm rasterization. Re-ran VLM A-vs-B successfully; initial VLM feedback highlights missing stroke colors and missing typed text in remarquee vs remarks.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/bug-report-vlm-validate-unidoc-render-type-check-error.md — Bug report
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee/cmds/rmdoc/vlm_validate.go — Poppler rasterization pipeline
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/reference/02-diary-golden-testing-research-and-rendering-issues-analysis.md — Recorded first real VLM run findings


## 2025-12-24

Added human-readable guide for the RMQ-0006 golden testing + validation system (how to use + how it works), including setup, commands, implementation pointers, and troubleshooting for brittle environments.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/reference/03-golden-testing-validation-system-how-to-use-how-it-works.md — New guide


## 2025-12-24

Updated diary with note about the new human-readable golden testing system guide.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/reference/02-diary-golden-testing-research-and-rendering-issues-analysis.md — Diary update


## 2025-12-24

Fixed V6 stroke color rendering: apply per-stroke RG in overlay ops and decode trailing RGBA marker into concrete highlight PenColor ids. Added parsing assertion test + stroke color debugging playbook; validated via vlm-validate vs remarks.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/rmdoc/render/v6_merge_background.go — Per-stroke color rendering in buildOverlayOps
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/rmdoc/rmv6_line_decode.go — Decode trailing RGBA marker to preserve highlighter colors
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/rmdoc/rmv6_stroke_color_test.go — Parsing test for stroke colors
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/reference/04-debugging-playbook-v6-stroke-color-rendering.md — New playbook


## 2025-12-24

Improved pdfcmp robustness: CompareFilesVisual now falls back to Poppler (pdftoppm) rasterization when UniDoc renderer fails with 'type check error' (common with remarks-generated PDFs).

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/pdfcmp/pdfcmp.go — pdftoppm fallback rasterizer


## 2025-12-24

Diagnosed and fixed a major golden brittleness issue: remarks notebook/blank pages are effectively 0.75-scaled due to CairoSVG px→pt behavior; aligned our blank-background V6 rendering to match so goldens don't fail with pure size mismatch. Added debug scripts and made pdfcmp tolerate +/-1px raster rounding. Cpage-pdf golden passes; Test.rmdoc now yields meaningful diffs.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/pdfcmp/pdfcmp.go — +/-1px tolerance and poppler raster use
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/rmdoc/render/v6_merge_background.go — Blank-background sizing aligned to remarks/CairoSVG
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/04-debug-golden-size-mismatch-test-rmdoc.sh — Repro script
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/06-debug-golden-size-mismatch-cpage-pdf.sh — Repro script


## 2025-12-24

Retroactive diary update: imported real-device screenshot for Test.rmdoc page 1, added VLM + plz-confirm human-in-loop scripts for visual review, added page mapping sanity-check widget, and added small debug scripts for stroke tools and group anchor/bbox inspection.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/reference/02-diary-golden-testing-research-and-rendering-issues-analysis.md — Diary update


## 2026-01-04

Add dedicated remarks-golden updater: fix -update-golden to overwrite existing goldens, add TestUpdateRemarksGoldens, and add a script to regenerate all remarks goldens; also mark tasks 47 and 84 done.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/rmdoc/render/golden_remarks_test.go — Make -update-golden regenerate/overwrite existing goldens
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/rmdoc/render/golden_remarks_update_test.go — Dedicated generator test to update all remarks goldens without compare
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/reference/03-golden-testing-validation-system-how-to-use-how-it-works.md — Document new TestUpdateRemarksGoldens workflow
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/16-update-remarks-goldens-all-fixtures.sh — One-shot script to regenerate all committed remarks goldens

