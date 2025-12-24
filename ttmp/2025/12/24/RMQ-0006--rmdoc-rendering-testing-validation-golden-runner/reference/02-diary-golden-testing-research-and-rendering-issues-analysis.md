---
Title: Diary - Golden testing research and rendering issues analysis
Ticket: RMQ-0006
Status: active
Topics:
    - go
    - remarkable
    - testing
    - validation
    - rmdoc
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Diary of research into using remarks for golden testing and analysis of rendering issues discovered through visual inspection.
LastUpdated: 2025-12-24T15:00:00Z
WhatFor: ""
WhenToUse: ""
---

# Diary - Golden testing research and rendering issues analysis

## 2025-12-24

### Morning: Golden testing research

Started researching how to use `remarks` as a reference implementation for golden testing of remarquee's PDF rendering output.

**Key findings:**
- `remarks` is a Python tool that processes `.rmdoc` files and outputs PDFs with naming pattern `{doc_name} _remarks.pdf`
- Can be invoked via CLI (`remarks <input> <output>`) or programmatically (`run_remarks()`)
- Uses PyMuPDF (fitz) for PDF manipulation and rendering
- Has comprehensive test infrastructure in `remarks/test_pdf.py` using pytest fixtures

**Comparison strategies identified:**
1. Visual pixel comparison (PyMuPDF `get_pixmap()` + numpy)
2. Structural comparison (page count, annotations, text content)
3. Hybrid approach (structural for fast feedback, visual for deep validation)

**Integration approaches:**
- Subprocess invocation (simplest, requires Python/remarks installed)
- Python script wrapper (more control, JSON output)
- Pure Go implementation (no Python dependency, but more complex)

Created comprehensive analysis document: `analysis/01-using-remarks-for-golden-testing-and-pdf-comparison.md`

### Afternoon: VLM validation and rendering issues

Added VLM/LLM OCR validation section to analysis document. Can use `pinocchio code professional --images` to validate rendered PDFs semantically.

**Visual inspection findings:**
After comparing remarquee output with remarks output, identified 5 major rendering issues:

1. **Strokes render without color (all black)**
   - Traced to `buildOverlayOps()` in `v6_merge_background.go:253` hardcoding `cc.Add_RG(0, 0, 0)`
   - `Stroke.Color` field exists but is ignored
   - Need to extract color from stroke and convert to RGB

2. **Highlighter strokes misaligned**
   - Different coordinate systems: strokes use `xSvg + xx(p.X - bbox.MinX)`, highlights use `xx(rect.X) + highlightsXTranslation`
   - Y-coordinate transforms also differ
   - Need consistent coordinate transforms

3. **Typed text not rendered**
   - `ParseRMV6RootTextBlock` exists but output never used
   - No rendering function for `RMV6RootText`
   - Comment says "strokes only" but highlights are implemented, only text missing

4. **Page format uses annotation bbox instead of full page**
   - `buildAnnotationOnlyPage` uses `wSvg, hSvg` (annotation bbox) instead of full page size
   - Should use full page dimensions (1404*72/226 x 1872*72/226 for notebooks)

5. **Template backgrounds not rendered**
   - `PageRef.Template` field exists but `BuildBackgroundPDF` comment says "Template backgrounds are a later milestone"
   - Creates blank pages instead of rendering templates

**Code analysis:**
Traced through codebase to understand root causes:
- `remarquee/pkg/rmdoc/render/v6_merge_background.go`: Main merge logic
- `remarquee/pkg/rmdoc/render/background.go`: Background PDF construction
- `remarquee/pkg/rmdoc/strokes.go`: Stroke struct definition
- `remarquee/pkg/rmdoc/pen_color.go`: Color handling
- `remarquee/pkg/rmdoc/rmv6_root_text.go`: Typed text parsing

Added detailed code analysis section to analysis document with file paths, line numbers, and code snippets.

**Tasks created:**
Added comprehensive investigation tasks for each rendering issue:
- Stroke color rendering investigation
- Highlighter alignment investigation  
- Typed text rendering investigation
- Page format investigation
- Template rendering investigation

Each task includes specific code locations to check and comparisons with remarks implementation.

### Next steps

1. Implement golden testing infrastructure (comparison utilities, golden file management)
2. Investigate each rendering issue in depth
3. Compare with remarks implementation for each issue
4. Create fixes for identified issues
5. Add VLM validation to test suite

### Late afternoon: Implemented Option B PDF comparison (pure Go)

Implemented a pure-Go PDF comparison helper package for golden testing:

- **Package**: `remarquee/pkg/pdfcmp`
- **Visual comparison**: `CompareFilesVisual` / `CompareBytesVisual`
  - Uses UniDoc renderer (`render.NewImageDevice().Render(page)`) similar to `rmapi/archive/zipdoc.go:makeThumbnail`
  - Per-page pixel diff with a configurable tolerance
  - Generates a diff PNG (red-highlighted changes) on mismatch
- **Structural comparison**: `CompareFilesStructural` / `CompareBytesStructural`
  - Page count check
  - Per-page annotation count (`PdfPage.GetAnnotations`)
  - Per-page extracted text hash via `unipdf/v3/extractor` (whitespace-normalized)

Added self-contained unit tests in `remarquee/pkg/pdfcmp/pdfcmp_test.go` that generate small PDFs and validate:
- identical PDFs match
- slightly different PDFs fail and produce a diff image

### Evening: Added a Go wrapper to invoke `remarks` (reference implementation runner)

Added a small helper package to run the Python `remarks` CLI and locate the resulting reference PDF(s):

- **Package**: `remarquee/pkg/refimpl/remarks`
- **Runner**: `Runner.Run(ctx, inputPath, outputDir)` shells out to `remarks <input> <outputDir> [--log_level ...]`
- **Missing tool handling**: returns sentinel `ErrNotFound` when `remarks` is not available on `PATH`
- **Output discovery**: `FindRemarksPDFs(outputDir)` and `FindSingleRemarksPDF(outputDir)` search recursively for `* _remarks.pdf`

This gives us a deterministic way to generate a reference PDF for later golden tests (paired with the pure-Go comparator from commit `a93a0856a49559893dd97d661fd4ca3c71646f0f`).

### Evening: First golden test wired up (remarks reference + Go comparator)

Added the first end-to-end golden test for the device V6 notebook fixture:

- **Test**: `TestRenderV6Golden_RemarksReference_TestRmdoc`
- **Location**: `remarquee/pkg/rmdoc/render/golden_remarks_test.go`
- **Behavior**:
  - renders the fixture via `render.MergeRMDocV6OntoBackgroundPDF`
  - runs the `remarks` CLI via `pkg/refimpl/remarks.Runner` to produce a reference PDF
  - compares PDFs via `pkg/pdfcmp.CompareFilesVisual` with a tolerance (currently 1%)
  - writes diff PNGs to the test temp dir on mismatch
  - skips if `remarks` is not available on PATH

### Evening: Second golden test wired up (cpage-pdf.rmdoc)

Added a second V6 golden test for the PDF-backed cPages fixture:

- **Test**: `TestRenderV6Golden_RemarksReference_CpagePdf`
- **Fixture**: `cmd/remarquee-ui/testdata/cpage-pdf.rmdoc`
- **Location**: `remarquee/pkg/rmdoc/render/golden_remarks_test.go`

### Evening: Legacy golden smoke test (rmapi-backed)

Added a legacy (V3/V5) golden-style smoke test for the legacy PDF-backed fixture:

- **Test**: `TestRenderLegacyGolden_Rmapi_Backend_LegacyPdfA4`
- **Fixture**: `cmd/remarquee-ui/testdata/legacy-pdf-a4.zip`
- **Location**: `remarquee/pkg/rmdoc/render/golden_legacy_rmapi_test.go`
- **Notes**: This does not use `remarks` (which is V6-oriented). Instead it validates the rmapi-backed legacy renderer end-to-end and asserts output PDF page count matches our parsed UI page plan.

### Evening: Golden file management (committed reference PDFs)

Added basic golden file management so the remarks-based golden tests can use a committed reference PDF when available:

- **Golden dir**: `cmd/remarquee-ui/testdata/golden/` (contains a README and naming convention)
- **Go test flag**: `-update-golden`
  - updates `*.remarks.pdf` reference files under the golden dir (opt-in)
  - tests use the committed golden when present; otherwise they fall back to running `remarks` (and skip if neither is available)

### Night: VLM helper CLI (pinocchio)

Wired an optional helper CLI to run a VLM check on rendered PDFs without writing a bunch of custom scripts:

- **Command**: `remarquee rmdoc vlm-validate`
- **Location**: `cmd/remarquee/cmds/rmdoc/vlm_validate.go` (registered in `cmd/remarquee/cmds/rmdoc/root.go`)
- **Behavior**:
  - renders selected pages from PDF A (and optional PDF B) to PNGs using Poppler `pdftoppm` (default)
  - invokes `pinocchio code professional --images <pngA>,<pngB>,... <prompt>`
  - prints the temp output dir so you can open the PNGs locally

### Night: First real VLM run (A vs B) + UniDoc renderer failure

We attempted to run `vlm-validate` using UniDoc’s renderer and hit:

- `render page 1: type check error`

Created bug report:
- `bug-report-vlm-validate-unidoc-render-type-check-error.md`

Then switched `vlm-validate` to Poppler (`pdftoppm`) rasterization and re-ran successfully with:
- A = `remarquee rmdoc render-v6` output (Go pipeline)
- B = `remarks` output (Python reference)

High-signal VLM findings from the first run (pages 1–2):
- **Color strokes**: reference output includes colored strokes; remarquee output appears black-only (matches our “stroke color missing” investigation).
- **Typed text**: reference output shows typed text (e.g. “Test”); remarquee output is missing it (matches “typed text not rendered” investigation).

### Night: Wrote a human guide for the golden testing system

Added a readable guide describing how to use and debug the golden system (golden tests + pdfcmp + remarks runner + vlm-validate), including the “brittle setup” failure modes and what to check.

- `reference/03-golden-testing-validation-system-how-to-use-how-it-works.md`

### Late night: Fixed V6 stroke color rendering (with a tight feedback loop)

We tackled the “all strokes are black” issue and built a clean loop to verify parsing + rendering:

- **Parsing verification**:
  - Added `TestParseRMV6SceneTree_StrokeColorsPresent_TestRmdoc` which asserts `Test.rmdoc` contains non-black stroke colors and at least one **concrete highlight color id** (14..19).
- **Root cause**:
  - Renderer was setting `RG(0,0,0)` once globally and never applying per-stroke color.
  - Highlighter strokes in V6 can store real color in an optional trailing RGBA marker; we were consuming it but discarding it, collapsing highlight strokes into a generic color id.
- **Fixes**:
  - Apply per-stroke `RG` in `buildOverlayOps` (so colored strokes render).
  - Decode the trailing `(b,g,r,a)` marker in `DecodeRMV6Line` and map via `HardcodedColorMap` to a concrete highlight PenColor id.
- **Validation**:
  - Re-ran `vlm-validate` (A vs B, page 1) and got a “colors match” result vs `remarks`.

Also wrote a dedicated debugging playbook for the next developer:
- `reference/04-debugging-playbook-v6-stroke-color-rendering.md`

### After midnight: Page size mismatch vs `remarks` (root cause + fix)

We hit a confusing golden failure mode: `maxDiffRatio=1.0` with no useful diffs. That turned out to be a **pure raster dimension mismatch** (A and B were different pixel sizes), not “everything is wrong”.

We made it reproducible with ticket scripts:

- `scripts/04-debug-golden-size-mismatch-test-rmdoc.sh`
- `scripts/06-debug-golden-size-mismatch-cpage-pdf.sh`

Root cause (high-level):

- `remarks` uses `rmc` to convert `.rm` → SVG → PDF via **CairoSVG**.
- CairoSVG treats SVG width/height as CSS px at 96dpi, then emits PDF points at 72pt/in → **0.75 scale**.
- In `remarks`, when a background page has no content stream, it inserts that SVG-PDF directly (no scaling), so notebook/blank pages can end up at the 0.75-sized box.

Fix (current pragmatic behavior for goldens):

- For pages with no background content, render overlays onto a fixed “rm screen” canvas using the CairoSVG-effective scale, to match `remarks` output sizing.
- Make `pdfcmp` tolerant to +/-1px raster rounding differences (compare overlap area).

Outcome:

- `TestRenderV6Golden_RemarksReference_CpagePdf` now passes reliably (no more size-mismatch-driven failure).
- `Test.rmdoc` golden now fails with a meaningful diff ratio (~4.2%) and diff PNGs, i.e. we’re back to “real diffs” (typed text, etc.) instead of “dimension mismatch”.
