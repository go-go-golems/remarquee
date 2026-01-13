---
Title: 'RMQ-0009 Tasks — Compile RMDoc-DSL → .rmdoc'
Ticket: RMQ-0009
Status: active
Topics:
  - remarkable
  - rmdoc
  - dsl
  - compiler
  - go
DocType: task-list
Intent: short-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: >
  Task list for implementing a compiler that emits real .rmdoc archives from RMDoc-DSL.
LastUpdated: 2026-01-10
---

# RMQ-0009 Tasks — Compile RMDoc-DSL → .rmdoc

## Milestone 0: Decide the exact target format

- [x] Confirm “target output”:
- [x] V6 cPages notebook `.rmdoc` (zip) with per-page `<page-id>.rm` V6 scene tree
- [x] `.content` / `.metadata` (and optionally `.pagedata`)
- [x] Confirm minimum acceptance criteria:
- [x] `remarquee pkg/rmdoc.OpenFile` can open the produced `.rmdoc`
- [x] `remarquee rmdoc render-v6` can render it
- [x] Device upload results in an **editable notebook** (not just a PDF)

## Milestone 1: Compiler “strokes-only notebook”

- [x] Implement RMDoc-DSL → internal “compiled doc” model:
- [x] page ordering, stable page IDs
- [x] layer→group mapping decision (can start with one group per layer)
- [x] Emit `.content` (cPages) referencing each page ID.
- [x] Emit `.metadata` (minimal viable metadata to be accepted by cloud/device).
- [x] Emit per-page V6 `.rm` bytes for **strokes-only**:
- [x] V6 header
- [x] SceneTree blocks + SceneLineItem blocks (enough for `ParseRMV6SceneTree` to read)
- [x] Zip into `.rmdoc`
- [x] Add a reproducible test fixture generator:
- [x] compile `cases/03-ellipse-sweep.js` into `ellipse-sweep.rmdoc`

## Milestone 2: Upload and “round-trip on device”

- [x] Add a CLI command:
- [x] `remarquee rmdsl compile <case.(yaml|js)> --out <file.rmdoc>`
- [x] Add an upload verb (or prove `cloud put` supports `.rmdoc` as notebook):
- [x] `remarquee cloud put <file.rmdoc> <remote-dir>` works and yields editable doc, OR
  - [ ] add `remarquee cloud put-rmdoc` / `remarquee cloud put-notebook` that sets correct document type
- [x] Add a “protocol script”:
- [x] compile → upload → prompt user to confirm it’s editable and correct

## Milestone 3: Extend beyond strokes (planned)

- [x] Shapes lowering (ellipse/rect) → stroke polylines (if not already compiled as stroke)
- [x] Highlights / glyph rectangles (SceneGlyphItem blocks)
- [x] Typed text (RootTextBlock + anchors)
- [ ] Templates (ties into RMQ-0007)

## Milestone 4: Tests + CI + docs

- [x] Unit tests for the `.rm` writer:
  - [ ] golden bytes tests for small blocks (tag encoding, varuint, crdt ids)
  - [ ] parse-then-reparse invariants (writer output is readable by our parser)
- [x] Integration tests:
- [x] compile DSL → `.rmdoc` → parse → render-v6 (no error)
- [x] Document the compiler in `pkg/doc/topics/`


