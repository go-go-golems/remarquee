---
Title: Incremental RMDoc compiler test plan
Ticket: RMQ-0009
Status: active
Topics:
    - remarkable
    - rmdoc
    - rendering
    - dsl
    - compiler
    - go
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarquee/pkg/rmdsl/compile/compile.go
      Note: Compiler entrypoint referenced by compile steps
    - Path: remarquee/pkg/rmdsl/compile/compile_test.go
      Note: Existing integration tests referenced in the ladder
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/scripts/01-compile-ellipse-sweep-rmdoc.sh
      Note: Example compile script for device validation
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/scripts/02-compile-upload-review.sh
      Note: Upload script referenced in device acceptance steps
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/tasks.md
      Note: Source of RMQ-0009 tasks the ladder maps onto
ExternalSources: []
Summary: 'Incremental test ladder for DSL -> .rmdoc compiler: start from empty notebook and progressively validate lines, shapes, layers, pages, colors, and device acceptance.'
LastUpdated: 2026-01-10T20:23:09.955117028-05:00
WhatFor: Guide RMQ-0009 validation from minimal .rmdoc output to device-level verification.
WhenToUse: When adding compiler features or diagnosing render failures (e.g., 'unable to render document').
---


# Incremental RMDoc compiler test plan

## Goal

Provide a step-by-step test ladder that starts with the simplest possible compiled `.rmdoc` and gradually adds features (lines, shapes, layers, pages, colors, highlights, text) so we can isolate which feature breaks rendering or device acceptance.

## Current compiler scope (baseline)

- Strokes-only `.rm` writer (V6): SceneTreeBlock + TreeNodeBlock + SceneGroupItemBlock + SceneLineItemBlock.
- Shapes (`ellipse`, `rect`) are lowered to polyline strokes.
- `.content` is cPages with minimal metadata; `.metadata` is minimal document metadata.
- No typed text, glyph ranges, or templates yet.

## Principles

- **One change per rung**: each test adds exactly one new feature.
- **Same verification pipeline**: compile -> open -> parse -> extract -> render; then device upload only after local checks pass.
- **Deterministic inputs**: short, static DSL cases; stable IDs via the compiler.
- **Fail fast**: stop at the first failing rung and fix before moving on.

## Validation ladder (ordered)

### Phase 0: Archive sanity

1) **Minimal notebook structure**
   - DSL: single page, single empty layer, no items.
   - Expected:
     - Zip contains `<doc>.content`, `<doc>.metadata`, `<doc>/<page>.rm`.
     - `rmdoc.OpenFile` succeeds; Schema = cPages; 1 page.
   - Automation:
     - `go test ./pkg/rmdsl/compile -run TestCompileRMDoc_OpenAndParse -count=1`
   - Manual: N/A.

2) **Render-v6 on empty page**
   - Input: `.rm` from test 1.
   - Expected: `remarquee rmdoc render-v6` completes without error.
   - Automation: add test or CLI run.
   - Manual: N/A.

### Phase 1: Single-stroke correctness

3) **Single line (2 points)**
   - DSL: one stroke with two points.
   - Expected:
     - 1 stroke extracted via `ExtractRMV6StrokesWithAnchors`.
     - First/last point coordinates match input.
     - `DecodeRMV6Line` round-trip succeeds.
   - Automation: unit test for line payload + integration compile test.

4) **Rectangle lowered to stroke**
   - DSL: single rectangle.
   - Expected:
     - 1 stroke; 5 points (closed rect).
     - Render-v6 draws a rectangle.

5) **Ellipse lowered to stroke**
   - DSL: single ellipse.
   - Expected:
     - 1 stroke; ~257 points (steps+1).
     - Render-v6 draws ellipse without errors.

### Phase 2: Multiple strokes and ordering

6) **Two strokes in one layer**
   - DSL: two strokes in order.
   - Expected:
     - 2 strokes extracted in DSL order.
     - CRDT left/right chain preserved.

7) **Layer order**
   - DSL: two layers with distinct strokes.
   - Expected:
     - Root group contains two group items in layer order.
     - Each stroke parent is the expected layer group.

### Phase 3: Multi-page notebook

8) **Two empty pages**
   - DSL: two pages, empty layers.
   - Expected:
     - cPages pages length = 2; `rmdoc.OpenFile` yields 2 pages.
     - Each page has a `.rm` file.

9) **Two pages with different strokes**
   - DSL: page 1 line; page 2 rectangle.
   - Expected:
     - Render-v6 for each page succeeds.
     - Correct stroke counts per page.

### Phase 4: Tool + color mapping

10) **Tool mapping**
   - DSL: one stroke per tool (`fineliner_2`, `marker_2`, `highlighter_2`).
   - Expected: decoded `Stroke.Tool` matches mapping table.

11) **Basic colors**
   - DSL: black/red/green/blue.
   - Expected: decoded `Stroke.Color` matches `PenColor` constants; render-v6 color changes.

12) **Highlight color marker**
   - DSL: highlight color (pink/green).
   - Expected:
     - Line payload encodes `PenColorHighlight` with RGBA marker.
     - Decoder maps to highlight color.
     - Render-v6 still succeeds.

### Phase 5: Device acceptance

13) **Upload simple notebook**
   - Compile test 3 (single line) to `.rmdoc`.
   - Upload via `remarquee cloud put`.
   - Expected on device:
     - Document opens without “unable to render”.
     - Notebook is editable (can add strokes).

14) **Upload multi-page ellipse sweep**
   - Compile `cases/03-ellipse-sweep.js`.
   - Expected on device:
     - Page markers (red dashes) show.
     - Ellipse moves top -> bottom across pages.
     - Notebook is editable.

### Phase 6: Future features (when implemented)

15) **Glyph highlights**
   - DSL: highlight rectangles (glyph ranges).
   - Expected: SceneGlyphItemBlock decode and render.

16) **Typed text**
   - DSL: text block, paragraph styles.
   - Expected: RootTextBlock + anchors parse; text layout stable.

17) **Templates**
   - DSL: template field per page.
   - Expected: `.pagedata` or cPages template is respected by renderer/device.

## Feature coverage matrix (quick map)

- Archive validity: tests 1–2
- Lines + points: tests 3–6
- Shapes lowering: tests 4–5
- Layers + ordering: tests 6–7
- Multi-page: tests 8–9
- Tools + colors: tests 10–12
- Device acceptance: tests 13–14
- Text/glyph/templates: tests 15–17 (future)

## Suggested test assets

Create RMQ-0009 cases alongside the ticket:

- `ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/cases/`
  - `01-empty-page.yaml`
  - `02-single-line.yaml`
  - `03-rect.yaml`
  - `04-ellipse.yaml`
  - `05-two-pages.yaml`
  - `06-two-layers.yaml`
  - `07-tools-colors.yaml`

## Recommended tooling checklist per test

- Compile: `go run ./cmd/remarquee rmdsl compile <case> --out /tmp/out.rmdoc`
- Open + parse: `remarquee rmdoc render-v6 /tmp/out.rmdoc --out /tmp/out.pdf --force`
- Extract strokes: add or reuse Go test in `pkg/rmdsl/compile`.
- Device upload (only after local pass): `remarquee cloud put /tmp/out.rmdoc /remarquee/rendering/rmq-0009`

## Known failure to isolate

- Current symptom: device shows “unable to render document” for the ellipse sweep `.rmdoc`.
- Use the ladder above to find the first failing rung and correct the compiler before re-testing the sweep.
