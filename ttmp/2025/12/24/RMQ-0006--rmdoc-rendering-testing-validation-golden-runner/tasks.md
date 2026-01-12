# Tasks

## TODO

### Carry-over from RMQ-0004 (unclosed)

- [x] Confirm scope + acceptance criteria:
- [x] Required outputs: PDF only vs PDF+PNG
- [x] Fidelity target: pixel-perfect vs “good enough” (strokes-only vs highlights vs typed text)
- [x] Supported inputs: `.rmdoc` only vs unpacked exports
- [x] Validation workflow: where PASS/FAIL + notes are stored (UI sessions vs markdown logs)

- [x] Template backgrounds:
- [x] Define template-to-page-size strategy (blank page constants vs render templates)

- [x] Typed text:
- [x] Implement typed text parsing/render/extraction plan (RootTextBlock is parsed enough for anchors; decide next outputs)

- [x] Golden test runner (task 53):
- [x] Decide baseline strategy:
- [x] Compare against `remarks`/`rmc` (reference implementation)
- [x] and/or commit goldens and regression-test against them
- [x] and/or structured diff of primitives (strokes/highlights/anchor offsets)
- [x] Implement `remarquee rmdoc validate` (or similar) that:
- [x] renders target fixture(s) through our pipeline
- [x] optionally renders through reference implementation (when available)
- [x] produces per-page PNGs and a diff report (tolerance-based)

- [x] Interactive validation UI (RMQ-RMDOC-WEB-001 follow-up):
- [x] Decide whether validation sessions live in RMQ-RMDOC-WEB-001 or here; keep one source of truth

### New validation work (this ticket)

- [x] Write a detailed testing/validation playbook doc (feature-by-feature) in `reference/`
- [x] Add stage-level debug CLIs to avoid console spam:
- [x] `remarquee rmdoc v6-stats <file>` (counts: strokes, glyph ranges, groups, anchors)
- [x] `remarquee rmdoc v6-dump-highlights <file> [--page N]` (rects + color + x_translation)
- [x] `remarquee rmdoc v6-dump-strokes <file> [--page N]` (bbox + sample points)

### Golden testing with remarks (from analysis research)

- [x] Implement remarks invocation wrapper:
  - [x] Create Go function to invoke remarks CLI (subprocess approach)
  - [x] Handle output path resolution (`{doc_name} _remarks.pdf` naming convention)
  - [x] Add error handling for missing remarks installation
  - [x] Add logging/verbose mode for debugging

- [x] Implement PDF comparison utilities:
- [x] Option A: Python script wrapper (`scripts/compare_with_remarks.py`)
- [x] Visual comparison using PyMuPDF `get_pixmap()` + numpy
- [x] Structural comparison (page count, annotations, text content)
- [x] JSON output for Go test integration
- [x] Configurable tolerance for visual diff
- [x] Option B: Pure Go implementation (if preferred)
    - [x] Use UniDoc for PDF reading
    - [x] Render pages to images for visual comparison
    - [x] Compare annotations and text content structurally
  - [x] Generate diff images on mismatch (save to test output directory)
  - [x] Make visual comparison robust to remarks PDFs:
    - [x] Fall back to Poppler `pdftoppm` rasterization when UniDoc renderer hits "type check error"
    - [x] Tolerate tiny (+/-1px) raster dimension differences (rounding) by comparing the overlap area

- [x] Set up golden file management:
  - [x] Create `testdata/golden/` directory structure
- [x] Script to generate golden PDFs from remarks for all fixtures
  - [x] Document golden file naming convention
  - [x] Add `-update-golden` flag to tests for intentional changes
  - [x] Store reusable ticket scripts for generating A/B PDFs + diagnostics (avoid brittle one-liners)

- [x] Create golden test cases:
  - [x] `TestRenderV6Golden_TestRmdoc` (device V6 notebook fixture)
  - [x] `TestRenderV6Golden_CpagePdf` (PDF-backed cPages fixture)
  - [x] `TestRenderLegacyGolden_Rmapi_Backend_LegacyPdfA4` (legacy PDF-backed fixture; rmapi-backed)
- [x] Each test should:
    - [x] Render with remarquee
    - [x] Generate/load golden from remarks
    - [x] Compare using chosen strategy (visual/structural/hybrid)
    - [x] Save diff images on failure
    - [x] Provide clear failure messages

- [x] CI integration:
- [x] Ensure remarks is available in CI environment (or skip golden tests if not)
- [x] Add golden tests to CI pipeline
- [x] Configure artifact storage for diff images
- [x] Document CI setup requirements

- [x] Documentation and validation:
- [x] Update testing playbook with golden test procedures
- [x] Document tolerance settings and when to adjust them
- [x] Add troubleshooting guide for common comparison issues
- [x] Validate analysis document completeness

### Rendering issue investigations (from visual inspection)

- [x] **Stroke color rendering**:
  - [x] Investigate why strokes render without color (all black)
  - [x] Trace `Stroke.Color` field usage through rendering pipeline
  - [x] Fix: apply per-stroke RGB in `buildOverlayOps` (no more global black RG)
  - [x] Fix: decode trailing highlight/shader RGBA marker in `DecodeRMV6Line` into concrete highlight PenColor ids
  - [x] Validate with `vlm-validate` (A vs B on `Test.rmdoc`)

- [x] **Highlighter stroke misalignment**:
- [x] Investigate coordinate system differences between strokes and highlights
- [x] Trace `highlightsXTranslation` calculation vs stroke positioning
- [x] Check if highlight rectangles use same coordinate transform as strokes
- [x] Compare highlight positioning logic with remarks
- [x] Document alignment issues and root causes
  - [x] Add human-in-the-loop validation loop to confirm whether this is actually an issue (device screenshot + plz-confirm image widget)
- [x] Decide/record conclusion: acceptable vs needs fix (and why)

- [x] **Ellipse/oval shape appears misaligned (Test.rmdoc page 1, vs device screenshot)**:
  - [x] Confirm the issue is real (page mapping sanity: A1 matches device; then do a focused crop/side-by-side review)
  - [x] Compare with `remarks` rendering for the same page (A vs B vs device)
  - [x] Run controlled calibration: ellipse sweep (known Y positions) on device
  - [x] Run inverse test: device-authored `ellipses-test` → parse → regen DSL → compare (A vs B)
  - [x] Conclusion: coordinate model is consistent; earlier mismatch was fixture/view confusion (no transform bug to fix)

- [x] **Typed text not rendered**:
- [x] Investigate `RootTextBlock` parsing vs rendering
- [x] Check if `ParseRMV6RootTextBlock` output is used anywhere
- [x] Trace typed text extraction from scene tree
- [x] Compare with remarks typed text rendering approach
- [x] Document where typed text rendering should be added

- [x] **Page format uses annotation bbox instead of full page**:
- [x] Investigate page size calculation in `MergeRMDocV6OntoBackgroundPDFWithInfo`
- [x] Check `width = math.Max(wSvg, wBg)` logic (line 155)
- [x] Trace when `wSvg`/`hSvg` (annotation bbox) vs `wBg`/`hBg` (background) is used
- [x] Check `buildAnnotationOnlyPage` page size calculation
- [x] Compare with remarks page size handling
- [x] Document expected vs actual page dimensions
  - [x] Fix: align notebook/blank-page sizing with remarks (CairoSVG 0.75 px→pt factor) so goldens don't fail with pure size mismatch
- [x] Decide what "correct" notebook/blank page sizing should be long-term (match remarks vs match reMarkable desktop export vs always full-screen)

- [x] **Template backgrounds not rendered**:
- [x] Investigate `PageRef.Template` field usage
- [x] Check `BuildBackgroundPDF` template handling (comment says "later milestone")
- [x] Trace template name extraction from `.pagedata` or `cPages.template`
- [x] Compare with remarks template rendering approach
- [x] Document template rendering requirements and implementation plan


### VLM validation helper

- [x] Add `remarquee rmdoc vlm-validate` helper to render PDF pages to PNGs and invoke pinocchio VLM for semantic validation/comparison
- [x] Switch `vlm-validate` PNG rasterization to Poppler (`pdftoppm`) to avoid UniDoc "type check error" failures (see bug report)
 - [x] Ensure pinocchio runs are non-interactive by default (avoid “continue in chat?” prompts)

### Human-in-the-loop visual review (new)

- [x] Import real-device screenshot reference for `Test.rmdoc` page 1 into ticket `reference/`
- [x] Add `plz-confirm image` scripts to ask humans to compare rendered output vs device screenshot
- [x] Add a small “image review playbook” (how to interpret answers; where to store results)

### RMDoc-DSL fixtures + scriptability (new)

- [x] Define RMDoc-DSL v0 spec (YAML) + design notes (models “supported now” vs “planned”)
- [x] Add JS scriptability (goja) with a minimal `rm` builder API + `rm.include()`
- [x] Add user-facing docs page in `pkg/doc/topics/` for YAML+JS usage
- [x] Add canonical ellipse sweep generator and end-to-end device upload + plz-confirm review script (PDF transport)
- [x] Add inverse workflow tools:
  - [x] Export V6 `.rmdoc` strokes → RMDoc-DSL YAML
  - [x] Parse → regen → A/B compare (PDF + PNG raster + plz-confirm)

### Next (recommended)

- [x] Compile RMDoc-DSL → real `.rmdoc` (editable notebook) + add upload command (so device truth uses the exact same fixture bytes, not just PDFs)
- [x] Add “sweep generators” beyond ellipses:
- [x] anchor translation sweeps (groups anchored to text)
- [x] highlight rectangle sweeps (glyph ranges vs stroke transforms)
- [x] typed-text layout sweeps (paragraph styles, line heights, top offsets)

