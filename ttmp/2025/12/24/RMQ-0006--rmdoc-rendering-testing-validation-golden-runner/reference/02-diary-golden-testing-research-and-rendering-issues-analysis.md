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
