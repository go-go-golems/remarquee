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

- [ ] Confirm “target output”:
  - [ ] V6 cPages notebook `.rmdoc` (zip) with per-page `<page-id>.rm` V6 scene tree
  - [ ] `.content` / `.metadata` (and optionally `.pagedata`)
- [ ] Confirm minimum acceptance criteria:
  - [ ] `remarquee pkg/rmdoc.OpenFile` can open the produced `.rmdoc`
  - [ ] `remarquee rmdoc render-v6` can render it
  - [ ] Device upload results in an **editable notebook** (not just a PDF)

## Milestone 1: Compiler “strokes-only notebook”

- [ ] Implement RMDoc-DSL → internal “compiled doc” model:
  - [ ] page ordering, stable page IDs
  - [ ] layer→group mapping decision (can start with one group per layer)
- [ ] Emit `.content` (cPages) referencing each page ID.
- [ ] Emit `.metadata` (minimal viable metadata to be accepted by cloud/device).
- [ ] Emit per-page V6 `.rm` bytes for **strokes-only**:
  - [ ] V6 header
  - [ ] SceneTree blocks + SceneLineItem blocks (enough for `ParseRMV6SceneTree` to read)
- [ ] Zip into `.rmdoc`
- [ ] Add a reproducible test fixture generator:
  - [ ] compile `cases/03-ellipse-sweep.js` into `ellipse-sweep.rmdoc`

## Milestone 2: Upload and “round-trip on device”

- [ ] Add a CLI command:
  - [ ] `remarquee rmdsl compile <case.(yaml|js)> --out <file.rmdoc>`
- [ ] Add an upload verb (or prove `cloud put` supports `.rmdoc` as notebook):
  - [ ] `remarquee cloud put <file.rmdoc> <remote-dir>` works and yields editable doc, OR
  - [ ] add `remarquee cloud put-rmdoc` / `remarquee cloud put-notebook` that sets correct document type
- [ ] Add a “protocol script”:
  - [ ] compile → upload → prompt user to confirm it’s editable and correct

## Milestone 3: Extend beyond strokes (planned)

- [ ] Shapes lowering (ellipse/rect) → stroke polylines (if not already compiled as stroke)
- [ ] Highlights / glyph rectangles (SceneGlyphItem blocks)
- [ ] Typed text (RootTextBlock + anchors)
- [ ] Templates (ties into RMQ-0007)

## Milestone 4: Tests + CI + docs

- [ ] Unit tests for the `.rm` writer:
  - [ ] golden bytes tests for small blocks (tag encoding, varuint, crdt ids)
  - [ ] parse-then-reparse invariants (writer output is readable by our parser)
- [ ] Integration tests:
  - [ ] compile DSL → `.rmdoc` → parse → render-v6 (no error)
- [ ] Document the compiler in `pkg/doc/topics/`


