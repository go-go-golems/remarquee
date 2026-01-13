---
Title: Testing + validation playbook (manual + staged CLI loop)
Ticket: RMQ-0006
Status: active
Topics:
  - go
  - remarkable
  - testing
  - validation
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
  - Path: ../../../../cmd/remarquee/cmds/rmdoc/inspect.go
    Note: Stage 0 inspection (schema + page plan)
  - Path: ../../../../cmd/remarquee/cmds/rmdoc/build_background.go
    Note: Stage 1 background-only PDF builder
  - Path: ../../../../cmd/remarquee/cmds/rmdoc/render_v6.go
    Note: Stage 2 V6 full pipeline renderer (strokes + smart highlights)
  - Path: ../../../../cmd/remarquee/cmds/rmdoc/render_legacy.go
    Note: Legacy renderer (rmapi-backed)
  - Path: ../../../../pkg/rmdoc/render/v6_merge_background.go
    Note: Merge + rotation + highlights_x_translation + smart highlights
  - Path: ../../../../cmd/remarquee-ui/testdata/test-documents.json
    Note: Fixture manifest for quick switching in UI
  - Path: ../../../../cmd/remarquee-ui/testdata/Test.rmdoc
    Note: Real device V6 notebook fixture
  - Path: ../../../../cmd/remarquee-ui/testdata/legacy-pdf-a4.zip
    Note: Legacy PDF-backed fixture (rmapi)
Summary: ""
LastUpdated: 2025-12-24T00:00:00Z
---

# RMQ-0006 — Testing + validation playbook

This document defines a **manual testing loop** to visually validate the features we already built, without overwhelming console output.

## Fixtures to use

All fixtures live under:
- `remarquee/cmd/remarquee-ui/testdata/`

Key fixtures:
- `Test.rmdoc` (device V6 notebook)
- `cpage-pdf.rmdoc` (PDF-backed cPages with duplicates + inserted pages)
- `legacy-pdf-a4.zip` (legacy PDF-backed, V5 .rm)
- `legacy-notebook.zip`
- `generated/*.rmdoc` (synthetic edge cases)

## Manual validation loop (recommended order)

### Stage 0 — Inspect page plan (no rendering)

Goal: confirm schema/type and the UI page plan.

```bash
cd /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee
go run ./cmd/remarquee rmdoc inspect ./cmd/remarquee-ui/testdata/Test.rmdoc
go run ./cmd/remarquee rmdoc inspect ./cmd/remarquee-ui/testdata/cpage-pdf.rmdoc
go run ./cmd/remarquee rmdoc inspect ./cmd/remarquee-ui/testdata/legacy-pdf-a4.zip
```

Checklist:
- schema is expected (cPages vs legacy)
- pages count matches expectation
- for PDF-backed docs, `SourcePDFPage` should be >= 0 or `-1` for inserted

### Stage 1 — Background-only PDF

Goal: isolate background construction / UI ordering bugs (no annotations).

```bash
go run ./cmd/remarquee rmdoc build-background --file ./cmd/remarquee-ui/testdata/cpage-pdf.rmdoc --out /tmp/cpage-pdf-bg.pdf --force
go run ./cmd/remarquee rmdoc build-background --file ./cmd/remarquee-ui/testdata/Test.rmdoc --out /tmp/device-Test-bg.pdf --force
```

Checklist:
- page count == UI page plan count
- duplicated/inserted pages appear in the right order

### Stage 2 — Full V6 pipeline (strokes + smart highlights)

Goal: validate merge math (width/height cases), rotation, highlights_x_translation, and highlight annotation placement.

```bash
go run ./cmd/remarquee rmdoc render-v6 ./cmd/remarquee-ui/testdata/Test.rmdoc --out /tmp/device-Test-v6.pdf --force
go run ./cmd/remarquee rmdoc render-v6 ./cmd/remarquee-ui/testdata/cpage-pdf.rmdoc --out /tmp/cpage-pdf-v6.pdf --force
```

Checklist:
- strokes appear on the expected pages (not shifted off-page)
- highlights appear as selectable PDF highlight annotations in your viewer
- inserted pages still render overlays correctly (blank bg)

### Stage 3 — Legacy pipeline (rmapi-backed)

Goal: validate that our legacy rendering path works end-to-end on a PDF-backed legacy doc.

```bash
go run ./cmd/remarquee rmdoc render-legacy ./cmd/remarquee-ui/testdata/legacy-pdf-a4.zip --out /tmp/legacy-a4.pdf --force
```

Checklist:
- output exists and displays strokes in the correct positions on the background PDF

## Debugging philosophy: don’t spam logs

When something looks wrong, the best next step is usually to add **small staged commands** that print summarized, structured stats for a single page:
- stroke count + bbox
- glyph range count + rectangles + colors
- computed `highlights_x_translation`

These commands don’t exist yet; RMQ-0006 includes tasks to add them.


