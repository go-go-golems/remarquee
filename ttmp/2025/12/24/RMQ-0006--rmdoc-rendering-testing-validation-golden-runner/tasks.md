# Tasks

## TODO

### Carry-over from RMQ-0004 (unclosed)

- [ ] Confirm scope + acceptance criteria:
  - [ ] Required outputs: PDF only vs PDF+PNG
  - [ ] Fidelity target: pixel-perfect vs “good enough” (strokes-only vs highlights vs typed text)
  - [ ] Supported inputs: `.rmdoc` only vs unpacked exports
  - [ ] Validation workflow: where PASS/FAIL + notes are stored (UI sessions vs markdown logs)

- [ ] Template backgrounds:
  - [ ] Define template-to-page-size strategy (blank page constants vs render templates)

- [ ] Typed text:
  - [ ] Implement typed text parsing/render/extraction plan (RootTextBlock is parsed enough for anchors; decide next outputs)

- [ ] Golden test runner (task 53):
  - [ ] Decide baseline strategy:
    - [ ] Compare against `remarks`/`rmc` (reference implementation)
    - [ ] and/or commit goldens and regression-test against them
    - [ ] and/or structured diff of primitives (strokes/highlights/anchor offsets)
  - [ ] Implement `remarquee rmdoc validate` (or similar) that:
    - [ ] renders target fixture(s) through our pipeline
    - [ ] optionally renders through reference implementation (when available)
    - [ ] produces per-page PNGs and a diff report (tolerance-based)

- [ ] Interactive validation UI (RMQ-RMDOC-WEB-001 follow-up):
  - [ ] Decide whether validation sessions live in RMQ-RMDOC-WEB-001 or here; keep one source of truth

### New validation work (this ticket)

- [ ] Write a detailed testing/validation playbook doc (feature-by-feature) in `reference/`
- [ ] Add stage-level debug CLIs to avoid console spam:
  - [ ] `remarquee rmdoc v6-stats <file>` (counts: strokes, glyph ranges, groups, anchors)
  - [ ] `remarquee rmdoc v6-dump-highlights <file> [--page N]` (rects + color + x_translation)
  - [ ] `remarquee rmdoc v6-dump-strokes <file> [--page N]` (bbox + sample points)

### Golden testing with remarks (from analysis research)

- [ ] Implement remarks invocation wrapper:
  - [ ] Create Go function to invoke remarks CLI (subprocess approach)
  - [ ] Handle output path resolution (`{doc_name} _remarks.pdf` naming convention)
  - [ ] Add error handling for missing remarks installation
  - [ ] Add logging/verbose mode for debugging

- [ ] Implement PDF comparison utilities:
  - [ ] Option A: Python script wrapper (`scripts/compare_with_remarks.py`)
    - [ ] Visual comparison using PyMuPDF `get_pixmap()` + numpy
    - [ ] Structural comparison (page count, annotations, text content)
    - [ ] JSON output for Go test integration
    - [ ] Configurable tolerance for visual diff
  - [ ] Option B: Pure Go implementation (if preferred)
    - [x] Use UniDoc for PDF reading
    - [x] Render pages to images for visual comparison
    - [x] Compare annotations and text content structurally
  - [x] Generate diff images on mismatch (save to test output directory)

- [ ] Set up golden file management:
  - [ ] Create `testdata/golden/` directory structure
  - [ ] Script to generate golden PDFs from remarks for all fixtures
  - [ ] Document golden file naming convention
  - [ ] Add `-update-golden` flag to tests for intentional changes

- [ ] Create golden test cases:
  - [ ] `TestRenderV6Golden_TestRmdoc` (device V6 notebook fixture)
  - [ ] `TestRenderV6Golden_CpagePdf` (PDF-backed cPages fixture)
  - [ ] `TestRenderV6Golden_LegacyPdfA4` (legacy PDF-backed fixture)
  - [ ] Each test should:
    - Render with remarquee
    - Generate/load golden from remarks
    - Compare using chosen strategy (visual/structural/hybrid)
    - Save diff images on failure
    - Provide clear failure messages

- [ ] CI integration:
  - [ ] Ensure remarks is available in CI environment (or skip golden tests if not)
  - [ ] Add golden tests to CI pipeline
  - [ ] Configure artifact storage for diff images
  - [ ] Document CI setup requirements

- [ ] Documentation and validation:
  - [ ] Update testing playbook with golden test procedures
  - [ ] Document tolerance settings and when to adjust them
  - [ ] Add troubleshooting guide for common comparison issues
  - [ ] Validate analysis document completeness

### Rendering issue investigations (from visual inspection)

- [ ] **Stroke color rendering**:
  - [ ] Investigate why strokes render without color (all black)
  - [ ] Trace `Stroke.Color` field usage through rendering pipeline
  - [ ] Check if `buildOverlayOps` in `v6_merge_background.go` uses stroke color
  - [ ] Compare with remarks implementation for color handling
  - [ ] Document where color information is lost or ignored

- [ ] **Highlighter stroke misalignment**:
  - [ ] Investigate coordinate system differences between strokes and highlights
  - [ ] Trace `highlightsXTranslation` calculation vs stroke positioning
  - [ ] Check if highlight rectangles use same coordinate transform as strokes
  - [ ] Compare highlight positioning logic with remarks
  - [ ] Document alignment issues and root causes

- [ ] **Typed text not rendered**:
  - [ ] Investigate `RootTextBlock` parsing vs rendering
  - [ ] Check if `ParseRMV6RootTextBlock` output is used anywhere
  - [ ] Trace typed text extraction from scene tree
  - [ ] Compare with remarks typed text rendering approach
  - [ ] Document where typed text rendering should be added

- [ ] **Page format uses annotation bbox instead of full page**:
  - [ ] Investigate page size calculation in `MergeRMDocV6OntoBackgroundPDFWithInfo`
  - [ ] Check `width = math.Max(wSvg, wBg)` logic (line 155)
  - [ ] Trace when `wSvg`/`hSvg` (annotation bbox) vs `wBg`/`hBg` (background) is used
  - [ ] Check `buildAnnotationOnlyPage` page size calculation
  - [ ] Compare with remarks page size handling
  - [ ] Document expected vs actual page dimensions

- [ ] **Template backgrounds not rendered**:
  - [ ] Investigate `PageRef.Template` field usage
  - [ ] Check `BuildBackgroundPDF` template handling (comment says "later milestone")
  - [ ] Trace template name extraction from `.pagedata` or `cPages.template`
  - [ ] Compare with remarks template rendering approach
  - [ ] Document template rendering requirements and implementation plan


