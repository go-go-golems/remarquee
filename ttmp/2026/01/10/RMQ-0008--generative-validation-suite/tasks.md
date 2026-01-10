---
Title: 'RMQ-0008 Tasks — Generative validation suite (RMDoc-DSL)'
Ticket: RMQ-0008
Status: active
Topics:
  - rmdoc
  - rendering
  - testing
  - validation
  - generative
  - dsl
DocType: task-list
Intent: short-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: >
  Task list for building a reusable generative validation suite using RMDoc-DSL.
LastUpdated: 2026-01-10
---

# RMQ-0008 Tasks — Generative validation suite (RMDoc-DSL)

## Goals (what success looks like)

- A library of **reproducible** DSL case generators (YAML + JS) covering the major renderer features.
- A runner that can:
  - generate cases,
  - render them (programmatic + PDF),
  - compare outputs (visual + structural),
  - optionally upload for device review,
  - and emit artifacts suitable for CI.
- A clear “contract” of invariants: what we expect to remain stable as the renderer evolves.

## Tasks

### 1) Suite shape + conventions

- [ ] Define directory layout for suite assets (cases, libs, outputs, expected).
- [ ] Define naming conventions:
  - [ ] case IDs
  - [ ] deterministic seeds
  - [ ] artifact file names
- [ ] Define how we store results:
  - [ ] local workdir
  - [ ] CI artifacts (diff PNGs, PDFs)

### 2) Generators (JS/goja + YAML)

- [ ] Ellipse sweep (already exists as prototype; move/standardize it under this ticket’s case structure).
- [ ] Rect/rotation sweep (angles + positions).
- [ ] Stroke tool sweep (tool id → expected width/opacity style where applicable).
- [ ] Highlight rectangle sweep:
  - [ ] glyph rectangles + stroke alignment invariants
  - [ ] regression harness for highlight X translation
- [ ] Anchor translation sweep:
  - [ ] groups anchored to synthetic “text anchors” (even before full typed text compile exists)
  - [ ] check that anchored items translate as expected
- [ ] Typed text layout sweep:
  - [ ] paragraph styles and line-height invariants
  - [ ] top offset invariants

### 3) Render targets

- [ ] Programmatic PNG renderer improvements:
  - [ ] per-item bboxes
  - [ ] layer toggles / filters
  - [ ] “shapes-only” and “highlights-only” views
- [ ] PDF transport renderer improvements:
  - [ ] consistent colors/widths
  - [ ] page labels / markers (for human review)

### 4) Comparisons

- [ ] Visual compare (pixel diff) with tolerance (reuse `pkg/pdfcmp` / Poppler pipeline).
- [ ] Structural checks:
  - [ ] page count
  - [ ] stroke count
  - [ ] bbox ranges / hashes
  - [ ] text extraction hashes (when text exists)

### 5) Device loops (when needed)

- [ ] Standard scripts:
  - [ ] generate → upload → plz-confirm review
  - [ ] download device-authored doc → parse → regen → compare
- [ ] Decide the “editable notebook” path:
  - [ ] compile RMDoc-DSL → `.rmdoc`
  - [ ] upload `.rmdoc` as editable document

### 6) CI integration

- [ ] Add a CI job that runs a subset of the suite (fast smoke) and uploads artifacts on failure.
- [ ] Document how to run the full suite locally vs CI.


