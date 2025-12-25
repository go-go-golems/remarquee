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
  - [x] Create Go function to invoke remarks CLI (subprocess approach)
  - [x] Handle output path resolution (`{doc_name} _remarks.pdf` naming convention)
  - [x] Add error handling for missing remarks installation
  - [x] Add logging/verbose mode for debugging

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
  - [x] Make visual comparison robust to remarks PDFs:
    - [x] Fall back to Poppler `pdftoppm` rasterization when UniDoc renderer hits "type check error"
    - [x] Tolerate tiny (+/-1px) raster dimension differences (rounding) by comparing the overlap area

- [ ] Set up golden file management:
  - [x] Create `testdata/golden/` directory structure
  - [ ] Script to generate golden PDFs from remarks for all fixtures
  - [x] Document golden file naming convention
  - [x] Add `-update-golden` flag to tests for intentional changes
  - [x] Store reusable ticket scripts for generating A/B PDFs + diagnostics (avoid brittle one-liners)

- [ ] Create golden test cases:
  - [x] `TestRenderV6Golden_TestRmdoc` (device V6 notebook fixture)
  - [x] `TestRenderV6Golden_CpagePdf` (PDF-backed cPages fixture)
  - [x] `TestRenderLegacyGolden_Rmapi_Backend_LegacyPdfA4` (legacy PDF-backed fixture; rmapi-backed)
  - [ ] Each test should:
    - [x] Render with remarquee
    - [x] Generate/load golden from remarks
    - [x] Compare using chosen strategy (visual/structural/hybrid)
    - [x] Save diff images on failure
    - [x] Provide clear failure messages

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

- [x] **Stroke color rendering**:
  - [x] Investigate why strokes render without color (all black)
  - [x] Trace `Stroke.Color` field usage through rendering pipeline
  - [x] Fix: apply per-stroke RGB in `buildOverlayOps` (no more global black RG)
  - [x] Fix: decode trailing highlight/shader RGBA marker in `DecodeRMV6Line` into concrete highlight PenColor ids
  - [x] Validate with `vlm-validate` (A vs B on `Test.rmdoc`)

- [ ] **Highlighter stroke misalignment**:
  - [ ] Investigate coordinate system differences between strokes and highlights
  - [ ] Trace `highlightsXTranslation` calculation vs stroke positioning
  - [ ] Check if highlight rectangles use same coordinate transform as strokes
  - [ ] Compare highlight positioning logic with remarks
  - [ ] Document alignment issues and root causes
  - [x] Add human-in-the-loop validation loop to confirm whether this is actually an issue (device screenshot + plz-confirm image widget)
  - [ ] Decide/record conclusion: acceptable vs needs fix (and why)

- [ ] **Ellipse/oval shape appears misaligned (Test.rmdoc page 1, vs device screenshot)**:
  - [ ] Confirm the issue is real (page mapping sanity: A1 matches device; then do a focused crop/side-by-side review)
  - [ ] Identify which stroke corresponds to the oval in our data (tool/color + bbox; correlate to on-page position)
  - [ ] Verify group anchor + root text anchor math for that stroke (dump group anchors + bbox after transforms)
  - [ ] Compare with `remarks` rendering for the same page (A vs B vs device)
  - [ ] Fix if needed, and add a short note explaining root cause (anchor translation vs scale vs coordinate inversion)

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
  - [x] Fix: align notebook/blank-page sizing with remarks (CairoSVG 0.75 px→pt factor) so goldens don't fail with pure size mismatch
  - [ ] Decide what "correct" notebook/blank page sizing should be long-term (match remarks vs match reMarkable desktop export vs always full-screen)

- [ ] **Template backgrounds not rendered**:
  - [ ] Investigate `PageRef.Template` field usage
  - [ ] Check `BuildBackgroundPDF` template handling (comment says "later milestone")
  - [ ] Trace template name extraction from `.pagedata` or `cPages.template`
  - [ ] Compare with remarks template rendering approach
  - [ ] Document template rendering requirements and implementation plan


### VLM validation helper

- [x] Add `remarquee rmdoc vlm-validate` helper to render PDF pages to PNGs and invoke pinocchio VLM for semantic validation/comparison
- [x] Switch `vlm-validate` PNG rasterization to Poppler (`pdftoppm`) to avoid UniDoc "type check error" failures (see bug report)
 - [x] Ensure pinocchio runs are non-interactive by default (avoid “continue in chat?” prompts)

### Human-in-the-loop visual review (new)

- [x] Import real-device screenshot reference for `Test.rmdoc` page 1 into ticket `reference/`
- [x] Add `plz-confirm image` scripts to ask humans to compare rendered output vs device screenshot
- [ ] Add a small “image review playbook” (how to interpret answers; where to store results)

