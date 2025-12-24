---
Title: Diary
Ticket: RMQ-0004
Status: active
Topics:
    - backend
    - go
    - remarkable
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../rmapi/annotations/pdf.go
      Note: Legacy PDF renderer reference for coordinate inversion and content stream composition
    - Path: ../../../../../../../rmc/src/rmc/exporters/svg.py
      Note: Reference for SCALE/X_SHIFT mapping from screen units to PDF points
    - Path: ../../../../../../../rmscene/src/rmscene/crdt_sequence.py
      Note: Python reference algorithm toposort_items ported to Go
    - Path: ../../../../../../../rmscene/src/rmscene/scene_stream.py
      Note: |-
        Python reference for SceneItemBlock.from_stream (tags 1..6)
        Python reference for build_tree + SceneTreeBlock/TreeNodeBlock formats
        Python reference for line_from_stream + point_from_stream
    - Path: ../../../../../../../rmscene/src/rmscene/tagged_block_common.py
      Note: Python reference for V6 header
    - Path: ../../../../../../../rmscene/src/rmscene/tagged_block_reader.py
      Note: Python reference implementation we’re porting (TaggedBlockReader.read_block/read_subblock)
    - Path: go.mod
      Note: Added unipdf v3.6.1 dependency for V6 PDF rendering
    - Path: pkg/rmdoc/bbox.go
      Note: RMQ-0004 Step 13 (commit cb097e1) - bbox primitives and stroke/tree bbox helpers
    - Path: pkg/rmdoc/bbox_test.go
      Note: Tests for stroke bbox + fixture smoke (commit cb097e1)
    - Path: pkg/rmdoc/render/v6_strokes_pdf.go
      Note: RMQ-0004 Step 12 (commit fee29bd) - render decoded V6 strokes to single-page PDF
    - Path: pkg/rmdoc/render/v6_strokes_pdf_test.go
      Note: Fixture-based PDF smoke test (%PDF header) (commit fee29bd)
    - Path: pkg/rmdoc/rmv6_crdt_sequence.go
      Note: RMQ-0004 Step 8 (commit 6adfc05) - CRDT sequence container + deterministic toposort ordering
    - Path: pkg/rmdoc/rmv6_crdt_sequence_test.go
      Note: Tests for ordering determinism + cycles (commit 6adfc05)
    - Path: pkg/rmdoc/rmv6_line_decode.go
      Note: RMQ-0004 Step 11 (commit b9a1ee9) - DecodeRMV6Line implementation
    - Path: pkg/rmdoc/rmv6_line_decode_test.go
      Note: Fixture-based test for V6 line decoding (commit b9a1ee9)
    - Path: pkg/rmdoc/rmv6_scene_item_block.go
      Note: RMQ-0004 Step 9 (commit f09f78c) - minimal SceneItemBlock header decode (parent_id + CRDT sequence header + raw value subblock)
    - Path: pkg/rmdoc/rmv6_scene_item_block_test.go
      Note: Fixture-based test for SceneItemBlock decoding (commit f09f78c)
    - Path: pkg/rmdoc/rmv6_scene_tree.go
      Note: RMQ-0004 Step 10 (commit 3ce401d) - build minimal V6 scene tree (groups + lines)
    - Path: pkg/rmdoc/rmv6_scene_tree_test.go
      Note: Fixture-based test for V6 scene tree construction (commit 3ce401d)
    - Path: pkg/rmdoc/rmv6_tagged_block_reader.go
      Note: RMQ-0004 Step 7 (commit 1cbf052) - Go port of rmscene tagged-block reader primitives (header + block + subblock boundaries)
    - Path: pkg/rmdoc/rmv6_tagged_block_reader_test.go
      Note: Fixture-based test exercising V6 header/block/subblock parsing (commit 1cbf052)
    - Path: pkg/rmdoc/rmv6_tagged_block_values.go
      Note: RMQ-0004 Step 9 (commit f09f78c) - tagged value readers (read_id/read_uint32/etc)
    - Path: pkg/rmdoc/rmv6_value_reader.go
      Note: RMQ-0004 Step 11 (commit b9a1ee9) - bounded tagged value reader for v6 payloads
    - Path: pkg/rmdoc/strokes.go
      Note: RMQ-0004 Step 11 (commit b9a1ee9) - normalized Stroke primitives
    - Path: ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/reference/06-diary-rmdoc-format-analysis-and-go-reimplementation-prep.md
      Note: Prior research diary that this ticket builds on
    - Path: ttmp/2025/12/14/RMQ-0004--port-rmdoc-parsing-rendering-to-go-v3-v5-v6/design-doc/01-design-go-rmdoc-data-model-and-apis.md
      Note: Design decisions and API proposal for this ticket
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-14T20:59:20.929388092-05:00
WhatFor: ""
WhenToUse: ""
---









# Diary

## Goal

This diary tracks the step-by-step work of porting `.rmdoc` parsing and annotation rendering into Go for remarquee, using RMQ-0001’s research as the baseline.

## Step 1: Bootstrap ticket + write initial design (seeded from RMQ-0001 research)

This step creates a clean implementation ticket (RMQ-0004) and turns the prior research into an actionable design. The point is to stop “re-learning” the `.rmdoc`/`.rm` ecosystem and instead lock down the data model boundaries and APIs we’ll build against.

**Commit:** 0850623 — "RMQ-0004: Seed Go rmdoc port ticket + initial design"

### What I did
- Created the RMQ-0004 docmgr ticket workspace.
- Created a design doc describing proposed Go packages, data structures, and APIs.
- Created this diary document and a first task list for the port.
- Linked the ticket overview back to RMQ-0001 primary research documents.

### Why
- We already did the hard “what’s inside `.rmdoc` and how does `remarks` render” work in RMQ-0001. This ticket should focus on **implementation**, not rediscovery.
- A clear data model + API boundary is the fastest way to make the port tractable and reviewable.

### What worked
- docmgr ticket creation + doc scaffolding makes it easy to keep design + diary + tasks in sync.
- The design doc captures the main architectural split we want: archive parsing vs annotation decoding vs rendering.

### What didn't work
- N/A (no implementation changes yet; this step is scaffolding + design).

### What I learned
- Even at the API-design stage, dual-format support (legacy V3/V5 + modern V6) should be treated as a first-class requirement, not a later “compat” add-on.
- Deterministic page ordering must come from `.content` (not filesystem iteration), because the Python reference implementation uses nondeterministic iteration in at least one place.

### What was tricky to build
- Choosing the right abstraction level: too “normalized” too early risks losing fidelity; too “raw” makes downstream features painful.
- The best compromise seems to be: **format-specific parsing + a small normalized surface** for rendering/extraction.

### What warrants a second pair of eyes
- The proposed package boundaries and the recommended “Proposal B” approach in the design doc:
  - Are we over-committing to unipdf for V6 rendering?
  - Should we keep a transitional Python-backed path for V6 longer than planned?
  - Do the proposed `PageRef` fields cover enough of the `cPages` edge cases (deleted pages, inserted pages, duplicates)?

### What should be done in the future
- Add fixture-based tests for both formats (downloaded from the device) and start a golden-output workflow.
- Start implementation with the archive layer + format detection + page plan, since everything else depends on it.

### Code review instructions
- Start with:
  - `design-doc/01-design-go-rmdoc-data-model-and-apis.md`
  - `tasks.md`
  - `index.md` (links to RMQ-0001 research)

### Technical details
- Primary research lives in RMQ-0001:
  - deep dive: `../RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/analysis/01-deep-dive-rmdoc-format-container-layout-parsing-png-rendering.md`
  - diary: `../RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/reference/06-diary-rmdoc-format-analysis-and-go-reimplementation-prep.md`
  - gap analysis: `../RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/scripts/go_reimplementation_gaps.md`

## Step 2: Implement `.rmdoc` open + `.content` schema detection + deterministic page plan

This step starts the actual Go port: a small `pkg/rmdoc` package that can open `.rmdoc` archives, detect whether the `.content` file is legacy or `cPages`, and produce a deterministic `[]PageRef` plan in UI order. The intent is to make every later stage (rendering, highlights, text extraction) depend on a single “page plan” contract instead of ad-hoc parsing.

**Commit (code):** 49acbde — "RMQ-0004: add rmdoc parser (schema detection + page plan)"

### What I did
- Added `remarquee/pkg/rmdoc` with:
  - `.rmdoc` zip opening (`OpenFile`, `OpenReaderAt`)
  - `.content` parsing (`ParseContent`) and schema detection (`cPages` vs legacy)
  - deterministic `[]PageRef` construction for both schemas
  - `.pagedata` parsing hook (`ApplyPagedataTemplates`) to fill missing templates
- Added unit tests for:
  - V6 `cPages` parsing (including deleted pages + inserted page detection)
  - legacy parsing (pages + redirection map)
  - pagedata template application

### Why
- Everything else (rendering, merge math, highlight placement) depends on having the right page order and PDF redirection map.
- The format varies across real device documents, so we need a single entrypoint that handles both schemas deterministically.

### What worked
- `go test ./...` passes in the `remarquee` module.
- The package yields a stable page plan for both schema families.

### What didn't work
- N/A (no runtime integration yet; only package + tests).

### What I learned
- The cPages schema is easy to detect (`"cPages"` key) and produces a clean UI-ordered list once you filter `deleted.value == 1`.
- Legacy `.content` is still a reality for PDF documents, so this dual-path is required from day 1.

### What was tricky to build
- Picking “just enough” typed schema: we want correctness and a stable `PageRef` output without prematurely modeling every `.content` field.
- Handling legacy archives that only provide `pageCount` (no `pages[]`) required generating synthetic page IDs.

### What warrants a second pair of eyes
- The semantics of “inserted page” for legacy redirection maps: I treated `-1` (InsertedPage) as “no source PDF page”, but we should verify against real legacy `.rmdoc` samples.
- Whether `pagedata` line ordering always matches the filtered cPages ordering (deleted pages could skew indices).

### What should be done in the future
- Integrate `pkg/rmdoc` into a CLI subcommand (even a temporary `debug` command) to print schema + page plan for a downloaded `.rmdoc`.
- Add fixture-based tests using real `.rmdoc` downloads (one legacy PDF doc + one V6 notebook doc).

### Code review instructions
- Start with `remarquee/pkg/rmdoc/open.go` and `remarquee/pkg/rmdoc/content.go`.
- Run:
  - `cd remarquee && go test ./... -count=1`

### Technical details
- Schema detection rule:
  - V6: `.content` contains `cPages`
  - legacy: otherwise; uses `pages`, `pageCount`, `redirectionPageMap`

## Step 3: Add `remarquee rmdoc inspect` CLI for schema + page plan debugging

This step turns the foundational `pkg/rmdoc` work into an actual tool you can run during development. The new CLI command prints schema, doc type, and the computed page plan, which will be invaluable when validating legacy vs cPages behavior and later when debugging V6 rendering issues.

**Commit (code):** c804b36 — "RMQ-0004: add rmdoc inspect CLI (schema + page plan)"

### What I did
- Added a new `remarquee rmdoc` command group with:
  - `remarquee rmdoc inspect <file.rmdoc>`: prints schema + doc type + page plan table
  - `--json`: dumps the parsed `pkg/rmdoc.Document` as JSON for scripts

### Why
- We need a fast, deterministic way to inspect what we’re going to render:
  - Is this legacy or cPages?
  - What is the UI page order?
  - Which UI pages map to which source PDF pages (`redir`)?
  - Which pages are inserted (`-1`)?

### What worked
- Command wiring is minimal and `go test ./...` passes.
- Verified against the repo fixture:
  - `remarquee rmdoc inspect ../remarks/tests/in/copies of different pages.rmdoc`

### What didn't work
- A cloud-based smoke test failed due to expired rmapi token:
  - `Error: failed to parse rmapi user token: token Expired`

### What I learned
- Having an inspect command early prevents a lot of “why is page order wrong?” guesswork later.

### What was tricky to build
- Keeping the output stable and easy to diff (tabular output) while still offering a script-friendly JSON mode.

### What warrants a second pair of eyes
- Whether the JSON output should be a “stable contract” (if so, we should define a dedicated output struct instead of dumping the internal `pkg/rmdoc.Document`).

### What should be done in the future
- Add an inspect mode that also prints the raw `.content` schema summary (keys present, page counts).
- Add a follow-up command that validates invariants (e.g., page indices are contiguous, templates line count matches pages).

### Code review instructions
- Start at:
  - `remarquee/cmd/remarquee/cmds/rmdoc/inspect.go`
  - `remarquee/cmd/remarquee/cmds/rmdoc/root.go`
  - `remarquee/cmd/remarquee/main.go`
- Run:
  - `cd remarquee && go run ./cmd/remarquee rmdoc inspect <some.rmdoc>`

## Step 4: Add integration tests for `pkg/rmdoc.OpenFile` using real fixtures (legacy + cPages)

This step makes the `.rmdoc` parsing layer safer to iterate on by adding integration tests against real `.rmdoc`-style fixtures already present in the repo: a modern `cPages` fixture from `remarks` and a legacy fixture from `rmapi`. This anchors schema detection and page-plan logic to concrete files, so future refactors don’t silently break one of the formats.

**Commit (code):** 3036c7e — "RMQ-0004: rmdoc OpenFile integration tests (legacy + cPages fixtures)"

### What I did
- Added `remarquee/pkg/rmdoc/open_integration_test.go` with tests that open:
  - `../remarks/tests/in/copies of different pages.rmdoc` (cPages + PDF payload)
  - `../rmapi/archive/test.zip` (legacy `.content` + V3 `.rm`)
- Asserted schema detection, doc type detection, and basic page-plan expectations (page counts + inserted pages).

### Why
- We need confidence that we keep supporting both schema families while we build rendering.
- Unit tests against hand-rolled JSON aren’t enough; the zip layout and file naming matter too.

### What worked
- `go test ./pkg/rmdoc -count=1` passes.
- `go test ./... -count=1` in `remarquee/` passes.

### What didn't work
- N/A.

### What I learned
- The rmapi legacy fixture’s `.content` can omit `pages[]` entirely and rely only on `pageCount`.
- The cPages fixture contains inserted pages that surface directly as `SourcePDFPage == -1` in our model.

### What was tricky to build
- Computing fixture paths robustly across `go test` execution contexts; solved via `runtime.Caller` relative path resolution.

### What warrants a second pair of eyes
- Are these tests “too coupled” to the current fixtures, and should we copy minimal fixtures into `remarquee/pkg/rmdoc/testdata/` instead?

### What should be done in the future
- Add a real-device legacy PDF `.rmdoc` fixture (downloaded from cloud) once auth is stable again.
- Add fixture-driven golden PDF rendering tests once the render pipeline exists.

### Code review instructions
- Start in `remarquee/pkg/rmdoc/open_integration_test.go`.
- Run `cd remarquee && go test ./pkg/rmdoc -count=1`.

## Step 5: Add legacy PDF rendering CLI (rmapi-backed) as a bridge milestone

This step unblocks end-to-end output for legacy archives by delegating PDF generation to rmapi’s `annotations.PdfGenerator`. It’s not the final architecture, but it provides a working rendering path for V3/V5 while we build the V6 parser/renderer and the unified renderer layer.

**Commit (code):** 05c257d — "RMQ-0004: add legacy rmdoc PDF renderer (rmapi-backed)"

### What I did
- Added `remarquee rmdoc render-legacy <file.rmdoc|file.zip>`:
  - opens the archive with `pkg/rmdoc` to confirm it’s legacy schema
  - delegates to `rmapi/annotations.CreatePdfGenerator(...).Generate()`
  - writes `<input>-annotations.pdf` by default (or `--out`)

### Why
- Legacy PDFs exist on-device and we need a usable rendering path ASAP.
- This is a pragmatic bridge: it validates the “page plan + render” workflow before we commit to the full unified renderer.

### What worked
- Verified on the legacy fixture without cloud auth:
  - input: `/home/manuel/workspaces/2025-12-14/build-remarquee-tool/rmapi/archive/test.zip`
  - output: a valid 1-page PDF

### What didn't work
- Running `go run` from a temp directory failed (no `go.mod`). Fixed by running from within `remarquee/`.

### What I learned
- rmapi’s PdfGenerator works fine as a drop-in for legacy archives and produces a PDF without additional wiring.

### What was tricky to build
- Ensuring we refuse cPages archives early so users don’t get confusing rmapi parsing errors for V6.

### What warrants a second pair of eyes
- Licensing/footprint implications of using `unipdf` transitively via rmapi inside remarquee.
- Whether we should keep this as a “prototype command” or treat it as supported behavior.

### What should be done in the future
- Replace this bridge with the unified renderer once we have:
  - background PDF assembly based on `PageRef.SourcePDFPage`
  - V6 stroke rendering
  - merge logic and highlights

### Code review instructions
- Start at:
  - `remarquee/cmd/remarquee/cmds/rmdoc/render_legacy.go`
- Run:
  - `cd remarquee && go run ./cmd/remarquee rmdoc render-legacy ../rmapi/archive/test.zip --force`

## Step 6: Implement UI-ordered background PDF assembly (cPages/PDF-backed docs) + convert rmdoc CLIs to Glazed

This step builds the missing foundation for the V6 pipeline: constructing a background PDF in **UI page order** using `PageRef.SourcePDFPage`. It also converts the `rmdoc` CLI commands to the repo’s standard **Glazed command** pattern (as used by the `cloud` command group), so we get consistent parameter parsing and structured output.

### What I did
- Added `pkg/rmdoc/render` with a `BuildBackgroundPDF` helper that:
  - opens the payload PDF (if present) and copies the referenced pages in UI order
  - inserts blank pages for `InsertedPage` (`SourcePDFPage == -1`)
  - duplicates payload pages when the UI references the same PDF page multiple times
- Added a fixture-based test that asserts “one background page per UI page”.
- Added `remarquee rmdoc build-background <file.rmdoc>` as a debug utility to emit the assembled background PDF.
- Converted `remarquee rmdoc inspect` and `remarquee rmdoc render-legacy` to Glazed commands (dual-mode), enabling:
  - default human output
  - structured output via `--with-glaze-output --output json|yaml|csv`

### Why
- The `remarks` algorithm relies on constructing a “background PDF” that already matches UI order (including inserted pages and duplicates). Without this, later merge math becomes fragile.
- Using Glazed for `rmdoc` commands keeps remarquee consistent across command groups and makes scripting easier.

### What worked
- `go test ./... -count=1` passes.
- Smoke-tested:
  - `remarquee rmdoc inspect ../remarks/tests/in/copies of different pages.rmdoc`
  - `remarquee rmdoc inspect ... --with-glaze-output --output json`
  - `remarquee rmdoc build-background ... --out /tmp/rmq4-bg.pdf --force`

### What didn't work
- Initially, the background PDF page count was wrong for the cPages fixture because both UI pages 0 and 1 point to the same payload PDF page (`redir=0`). Adding the same `*PdfPage` instance twice can collapse into fewer pages. Fix: duplicate the page (`PdfPage.Duplicate()`) when adding it to the output.

### What I learned
- `creator.PageSize` is a `[2]float64` (not a `{Width,Height}` struct), so defaults need to treat it like an array.
- In `cPages`-based PDFs, it’s common to see “duplicate pages” at the UI layer (multiple UI pages with identical `redir`), so duplication needs to be a first-class part of background assembly.

### What was tricky to build
- Getting duplication correct without prematurely implementing a full PDF merge engine. The minimal builder still needs to handle “copy, blank insert, duplicate” correctly.

### What warrants a second pair of eyes
- The default blank page size we currently use for inserted pages/templates (445x594 pt) is a stopgap; we should confirm the right size rules for real notebooks/templates once we start V6 rendering.

### What should be done in the future
- Implement template-backed background rendering for notebooks (instead of plain blank pages).
- Move on to V6 `.rm` parsing and stroke rendering now that background assembly exists.

### Code review instructions
- Start with:
  - `remarquee/pkg/rmdoc/render/background.go`
  - `remarquee/cmd/remarquee/cmds/rmdoc/build_background.go`
- Run:
  - `cd remarquee && go test ./... -count=1`

## Step 7: Port the RM v6 tagged-block reader scaffolding (header + block + subblock boundaries)

This step starts the real V6 `.rm` parsing work by porting the *lowest-level* binary reader primitives from `rmscene` into Go. The goal is **not** to decode the scene tree yet, but to establish a solid and testable foundation: validate the V6 header, iterate top-level blocks, and correctly parse Length4 subblock boundaries without overrunning or losing track of offsets.

This unlocks the next V6 milestones (CRDT decoding + scene tree + strokes) by giving us a Go-native reader that can be wired into those higher-level decoders incrementally.

**Commit (code):** 1cbf052 — "RMQ-0004: scaffold RM v6 tagged block reader"

### What I did
- Implemented a minimal RM v6 tagged-block reader in Go:
  - Validates the fixed V6 header (`reMarkable .lines file, version=6 ...`)
  - Reads top-level block headers: `block_length`, `min_version`, `current_version`, `block_type`
  - Tracks block boundaries and captures unread bytes as `ExtraData`
  - Reads Length4 subblock boundaries and captures unread bytes as `ExtraData`
- Added a fixture-based test that extracts the first `.rm` file from:
  - `cmd/remarquee-ui/testdata/cpage-pdf.rmdoc`
  - and validates that we can read the header + iterate blocks and observe at least one Length4 subblock

### Why
- V6 rendering depends on decoding V6 `.rm` files; everything above that (scene tree, strokes, highlights, merge) needs a trustworthy binary reader.
- Starting with boundary correctness (no overreads, correct offset math) prevents subtle corruption bugs later when we add semantic decoding.

### What worked
- `gofmt` + `go test ./... -count=1` passed in `remarquee/`.
- The fixture-based test finds at least one Length4 subblock in real V6 data and exercises block/subblock boundary logic.

### What didn't work
- N/A (no blockers in this step).

### What I learned
- Not all Length4 subblocks necessarily contain nested tagged fields; some subblocks contain *raw* encodings (e.g. string payloads). So the reader API should focus on **subblock boundaries**, not “blind recursive tag parsing”.

### What was tricky to build
- Getting “peek tag” semantics correct without consuming bytes: we use `io.ReadSeeker` so we can read a varuint and seek back.
- Avoiding accidental assumptions about subblock internal structure: we intentionally discard unread bytes as `ExtraData` at boundaries rather than guessing content layouts.

### What warrants a second pair of eyes
- The decision to treat under-read bytes as `ExtraData` and discard them (mirroring rmscene’s behavior) is correctness-critical: confirm this is the right behavior for Go-side iteration and for future decoders.
- Verify the EOF behavior is correct: in particular, `readBlockHeader` maps truncated block-length reads to `io.EOF`.

### What should be done in the future
- Move the V6 reader API toward the shape in the RMQ-0004 design doc (`pkg/rmdoc/anno/v6`) once we start decoding scene blocks (avoid leaving it as an unstructured “misc helper” forever).
- Build the next layer: typed wrappers for “read tag + read ID/uint32/float32” and CRDT sequence decoding.

### Code review instructions
- Start with:
  - `remarquee/pkg/rmdoc/rmv6_tagged_block_reader.go`
  - `remarquee/pkg/rmdoc/rmv6_tagged_block_reader_test.go`
- Validate with:
  - `cd remarquee && go test ./pkg/rmdoc -count=1`
  - `cd remarquee && go test ./... -count=1`

### Technical details
- Header constant mirrors `rmscene/src/rmscene/tagged_block_common.py` `HEADER_V6`.
- Top-level block header mirrors `rmscene/src/rmscene/tagged_block_reader.py` `read_block()`:
  - `uint32 block_length` (little-endian)
  - `uint8 unknown` (assert 0)
  - `uint8 min_version`, `uint8 current_version`
  - `uint8 block_type`

## Step 8: Add CRDT sequence container + deterministic ordering (toposort)

This step ports `rmscene`’s CRDT sequence ordering logic into Go. It doesn’t decode any V6 scene items yet, but it gives us a critical building block: **given a set of CRDT sequence items with left/right neighbor references, produce a deterministic UI/stable order**. That ordering is required for layers, groups, and other scene tree sequences when we start decoding them in task 34.

The implementation intentionally mirrors the Python algorithm (dependency graph + repeated extraction of “no-deps” nodes), including the behavior around unknown neighbor IDs (treated as start/end markers). Tests focus on determinism, simple chains, unknown references, and cycle detection.

**Commit (code):** 6adfc05 — "RMQ-0004: add RM v6 CRDT sequence ordering"

### What I did
- Added `RMV6CrdtID`, `RMV6CrdtSequenceItem`, and `RMV6CrdtSequence` types.
- Implemented deterministic topo ordering (`Keys/Values/Items`) based on `left_id` / `right_id` constraints.
- Added unit tests for:
  - deterministic ordering of independent items
  - linear chains via `left_id`
  - unknown `left_id` treated as start
  - cycle detection

### Why
- V6 scene decoding stores many collections as CRDT sequences; without a deterministic ordering primitive, higher-level decoders can’t build stable scene trees.
- Getting this “pure ordering logic” correct and well-tested is cheaper than debugging ordering bugs inside stroke/highlight rendering later.

### What worked
- `gofmt` + `go test ./... -count=1` passed.
- Ordering is deterministic even when multiple items become “available” simultaneously (ties broken by `(part1, part2)` ID order).

### What didn't work
- N/A.

### What I learned
- The `right_id` edges matter: the Python algorithm models “right comes after item” explicitly (graph edge from right_id node to item_id). Porting that faithfully preserves behavior.

### What was tricky to build
- Representing Python’s `"__start"`/`"__end"` sentinels without using string keys: we modelled a small comparable `rmv6TopoKey` with Kind (start/item/end) plus optional ID.
- Ensuring determinism: we sort the frontier (“no deps”) set before yielding any item IDs.

### What warrants a second pair of eyes
- Confirm the edge model matches `rmscene` exactly (especially treatment of unknown neighbor IDs as start/end).
- Verify the decision to return an explicit error when the topo walk yields fewer items than expected (helps surface malformed/cyclic inputs early).

### What should be done in the future
- Once we start decoding scene blocks, integrate this sequence container so decoded `SceneTree` groups/layers preserve stable order.
- If we discover real-world documents with additional CRDT invariants (e.g., deleted spans), extend tests to cover those cases.

### Code review instructions
- Start with:
  - `remarquee/pkg/rmdoc/rmv6_crdt_sequence.go`
  - `remarquee/pkg/rmdoc/rmv6_crdt_sequence_test.go`
- Run:
  - `cd remarquee && go test ./pkg/rmdoc -count=1`

### Technical details
- This is a direct port of `rmscene/src/rmscene/crdt_sequence.py` (`toposort_items`) adapted to Go’s data structures.

## Step 9: Decode V6 scene-item headers (CRDT sequence item fields) from tagged blocks

This step advances the V6 parser milestone by moving from “we can read block boundaries” to “we can decode real structured data out of those blocks”. Concretely, we now parse the **SceneItemBlock header fields** used to build the scene tree: `parent_id` plus the CRDT sequence item header (`item_id`, `left_id`, `right_id`, `deleted_length`). We also parse the presence of the value subblock (index 6), read its first byte (`item_type`), and keep the remaining value bytes as raw payload for later stroke/glyph decoding.

This is the minimum viable piece of task 34: CRDT sequence decoding as required by scene item blocks.

**Commit (code):** f09f78c — "RMQ-0004: decode V6 scene item headers"

### What I did
- Added tagged-value helpers on top of the tagged-block reader:
  - `checkTag`, `readExpectedTag`
  - `readID`, `readUint32`, `readBool`, `readFloat32`, `readFloat64`, and optional variants
  - `hasSubBlock` and a first `readString` helper (subblock-based)
- Implemented a minimal V6 `SceneItemBlock` decoder:
  - Reads `parent_id` (tag 1)
  - Reads CRDT sequence item header fields (tags 2..5)
  - Optionally reads value subblock 6:
    - reads `item_type` (first byte)
    - stores remaining subblock bytes as raw payload for later decoding
  - Captures unread bytes at block boundaries as `ExtraBlockData`
- Added a fixture-based test that extracts a real V6 `.rm` from `cmd/remarquee-ui/testdata/cpage-pdf.rmdoc` and asserts we can decode at least one scene item block.

### Why
- Scene tree construction (task 35) depends on these exact fields; without them we can’t place items into groups/layers in deterministic order.
- Keeping the value subblock as raw bytes lets us stage the work: first get structural decode correct, then decode strokes/highlights incrementally.

### What worked
- `gofmt` + `go test ./... -count=1` passed.
- The fixture-based test successfully finds and parses at least one scene item block.

### What didn't work
- N/A.

### What I learned
- The scene item “value” is always nested under subblock 6, but its contents are item-type specific. Treating it as raw until we implement per-item decoders is the safest incremental path.

### What was tricky to build
- “Rewind on mismatch” semantics for expected tags: we explicitly seek back when an index/type doesn’t match so optional reads can probe safely.
- Maintaining forward progress on partial parse failures: the parser is structured to discard remaining bytes and continue scanning blocks.

### What warrants a second pair of eyes
- Validate that the block types we classify as “scene item blocks” (0x03/0x04/0x05/0x06/0x08) match `rmscene` for the device documents we care about.
- Confirm that mapping `deleted_length` as `uint32` is correct for all observed files (rmscene uses “unsigned int” but notes uncertainty).

### What should be done in the future
- Implement typed decoding for at least one item type (strokes/lines) so we can hit the “strokes-only” milestone (tasks 35–36).
- Consider moving V6 parsing code into a dedicated subpackage (e.g. `pkg/rmdoc/anno/v6`) once it grows beyond a couple files.

### Code review instructions
- Start with:
  - `remarquee/pkg/rmdoc/rmv6_tagged_block_values.go`
  - `remarquee/pkg/rmdoc/rmv6_scene_item_block.go`
  - `remarquee/pkg/rmdoc/rmv6_scene_item_block_test.go`
- Run:
  - `cd remarquee && go test ./pkg/rmdoc -count=1`

### Technical details
- Based on `rmscene/src/rmscene/scene_stream.py` `SceneItemBlock.from_stream`:
  - tags: 1 (parent), 2 (item_id), 3 (left_id), 4 (right_id), 5 (deleted_length), 6 (value subblock)

## Step 10: Build a minimal V6 scene tree (groups + lines)

This step takes the V6 parsing work from “we can decode individual blocks” to “we can reconstruct the document’s logical structure”. We now build a **scene tree** that mirrors the `rmscene` algorithm: group nodes are created from `SceneTreeBlock` entries, enriched by `TreeNodeBlock` metadata (currently stored as raw extra bytes), and populated with ordered children via CRDT sequence items from scene item blocks.

For now we focus on the critical structural pieces for rendering: **groups and lines**. Line item “value” bytes are carried through as raw payload for the next milestone (task 36: expose strokes as normalized primitives).

**Commit (code):** 3ce401d — "RMQ-0004: build V6 scene tree (groups + lines)"

### What I did
- Implemented `ParseRMV6SceneTree` that reads a V6 `.rm` file and builds:
  - `SceneTreeBlock` (block type `0x01`): creates group nodes (`tree_id`) with `parent_id`.
  - `TreeNodeBlock` (block type `0x02`): attaches raw extra metadata to an existing node (label/visible/etc are not decoded yet).
  - Scene item blocks (0x03 glyph, 0x04 group, 0x05 line, 0x06 text, 0x08 tombstone):
    - group items resolve their referenced node id and attach `*RMV6Group` to the item.
    - line items store `Raw` value bytes plus the block version for future stroke decoding.
    - all items are stored as `RMV6CrdtSequenceItem[RMV6SceneItem]` under their parent group.
- Added a fixture-based test that asserts:
  - the root node exists (`CrdtId(0,1)`),
  - at least one additional group node exists,
  - at least one line item exists somewhere in the tree.

### Why
- Rendering depends on the correct parent/child hierarchy and deterministic ordering of items. Without a tree, “render strokes to PDF” becomes guesswork.
- This matches the proven Python behavior (build_tree) closely enough to validate structure before implementing stroke decoding.

### What worked
- `gofmt` + `go test ./... -count=1` passed.
- The fixture-based test succeeds on real V6 `.rm` data from `cpage-pdf.rmdoc`.

### What didn't work
- N/A.

### What I learned
- `SceneTreeBlock` uses `tree_id` as the identity for the group node (this is what `rmscene` adds to the tree), while `node_id` exists but isn’t used for node identity in build_tree.

### What was tricky to build
- Ensuring we don’t “link groups to parents twice”: `AddNode` is idempotent and parent linkage happens via SceneGroupItemBlock, mirroring the Python algorithm.
- Correctly preserving deterministic child order: children are stored as CRDT sequence items and ordered via our `RMV6CrdtSequence` toposort logic when enumerated.

### What warrants a second pair of eyes
- Verify that treating a missing TreeNodeBlock’s node as a hard error is desirable (Python throws). If real files can violate this ordering, we may need to tolerate it.
- Confirm we’re interpreting “group item” value correctly (`read_id(2)` inside value subblock), and that it always points to a node created by SceneTreeBlock.

### What should be done in the future
- Decode TreeNodeBlock fields into typed metadata (label/visible/anchors) once needed for layout/rendering.
- Implement line value decoding (tool/color/points) and expose normalized stroke primitives (task 36).

### Code review instructions
- Start with:
  - `remarquee/pkg/rmdoc/rmv6_scene_tree.go`
  - `remarquee/pkg/rmdoc/rmv6_scene_tree_test.go`
- Run:
  - `cd remarquee && go test ./pkg/rmdoc -count=1`

### Technical details
- Ported from `rmscene/src/rmscene/scene_stream.py` `build_tree()` and the relevant block formats:
  - `SceneTreeBlock` (BLOCK_TYPE 0x01)
  - `TreeNodeBlock` (BLOCK_TYPE 0x02)

## Step 11: Decode V6 line items into normalized stroke primitives

This step completes the “strokes-only” half of the V6 parser milestone by decoding `SceneLineItemBlock` payloads into a normalized `Stroke` primitive (tool, color, thickness scale, and point list). Rather than jumping straight into PDF rendering, we keep this stage as a pure data transformation with a fixture-based unit test, so later rendering work can focus on geometry/merge math instead of binary parsing.

The decoder is a direct port of `rmscene.scene_stream.line_from_stream`, including the mixed format inside the points subblock (tagged outer structure + raw-packed point records).

**Commit (code):** b9a1ee9 — "RMQ-0004: decode V6 line strokes"

### What I did
- Added a normalized primitive type:
  - `Stroke` + `StrokePoint` in `pkg/rmdoc/strokes.go`
- Implemented a dedicated “tagged values + bounded subblocks” reader:
  - `rmV6ValueReader` in `pkg/rmdoc/rmv6_value_reader.go`
  - supports tags (varuint index/type), primitive reads, and nested Length4 subblocks with enforced bounds
- Implemented `DecodeRMV6Line` in `pkg/rmdoc/rmv6_line_decode.go`:
  - reads tagged fields: tool (1), color (2), thickness_scale (3), starting_length (4)
  - reads subblock 5 containing raw-packed points (v1/v2 point formats)
  - best-effort consumes optional fields (timestamp/move_id) and ignores trailing RGBA marker unless the prefix matches `0x84 0x01`
- Added `rmv6_line_decode_test.go`:
  - builds a scene tree from the V6 fixture, finds the first line item, decodes it, and asserts non-empty points

### Why
- Rendering requires stroke primitives; decoding them now keeps later work focused and testable.
- This makes the eventual “V6 render to PDF” implementation much simpler: it can take `[]Stroke` and draw them.

### What worked
- `gofmt` + `go test ./... -count=1` passed.
- The fixture-based test successfully decodes a real V6 line and yields non-empty points.

### What didn't work
- N/A.

### What I learned
- V6 line payloads are “hybrid”: tagged structure for metadata and subblock boundaries, plus raw-packed point arrays inside the points subblock. A bounded reader is essential to avoid overreads.

### What was tricky to build
- Implementing nested length limits cleanly without relying on top-level block parsing: `rmV6ValueReader` maintains an explicit limit stack.
- Decoding point format v1 vs v2:
  - v2 uses compact `(float32 x,y) + (uint16 speed,width) + (uint8 direction,pressure)`
  - v1 uses float encodings that require post-processing (speed*4, etc) to match rmscene behavior.

### What warrants a second pair of eyes
- Confirm the decision to *not* enforce “read to end” (we allow extra bytes for forward compatibility) matches our goals.
- Validate the interpretation of `direction/pressure/width/speed` fields in both point versions; later rendering may depend on unit assumptions.

### What should be done in the future
- Add decoding for glyph ranges and highlights once we have confidence in the stroke pipeline.
- Plumb decoded strokes into the V6 renderer layer (task 40+), using the same coordinate transform constants as `rmscene`.

### Code review instructions
- Start with:
  - `remarquee/pkg/rmdoc/rmv6_line_decode.go`
  - `remarquee/pkg/rmdoc/rmv6_value_reader.go`
  - `remarquee/pkg/rmdoc/strokes.go`
- Run:
  - `cd remarquee && go test ./pkg/rmdoc -count=1`

### Technical details
- Ported from `rmscene/src/rmscene/scene_stream.py`:
  - `line_from_stream`
  - `point_from_stream`
  - `point_serialized_size`

## Step 12: Render V6 strokes to a PDF page (strokes-only, scaled + centered)

This step connects the V6 “data plane” (decoded strokes) to an actual PDF output. The goal is a minimal, reliable rendering path: given decoded `[]Stroke`, produce a single-page PDF that visibly contains the strokes, using the same baseline coordinate transform as `rmc` (scale from screen units to PDF points and center X).

This is not the final V6 renderer (it doesn’t implement merge math, brush fidelity, or anchors), but it establishes a concrete output artifact that we can iteratively improve while keeping the parsing/stroke pipeline stable.

**Commit (code):** fee29bd — "RMQ-0004: render V6 strokes to PDF"

### What I did
- Added a minimal strokes-only PDF renderer using UniPDF:
  - `pkg/rmdoc/render/v6_strokes_pdf.go`
  - `RenderRMV6StrokesToPDF(ctx, strokes, out)` draws polyline strokes on a single PDF page.
  - `RenderRMV6RMToPDF(ctx, rm, out)` convenience wrapper: parse scene tree → decode line strokes → render.
- Implemented coordinate transform aligned with `rmc`:
  - scale \(72 / 226\) from screen units to PDF points,
  - `xShift = pageWidth/2` to center the coordinate system,
  - `y = pageHeight - scaledY` to map to PDF coordinate origin.
- Added a fixture-based test `v6_strokes_pdf_test.go` that:
  - extracts a real V6 `.rm` from `cmd/remarquee-ui/testdata/cpage-pdf.rmdoc`,
  - renders it to bytes,
  - asserts output starts with `%PDF-`.
- Updated `remarquee/go.mod` to require `github.com/unidoc/unipdf/v3 v3.6.1` (matching rmapi).

### Why
- Having a PDF output loop early makes validation and iteration dramatically faster than waiting until merge/highlights are implemented.
- Keeping this renderer “strokes-only” lets us validate parsing + coordinate mapping independently of background merge behavior.

### What worked
- `go test ./... -count=1` passed.
- PDF bytes are produced for real V6 data with a stable header.

### What didn't work
- N/A.

### What I learned
- rmapi’s legacy renderer and rmc’s SVG exporter use compatible baseline screen-unit conventions:
  - `DeviceWidth=1404`, `DeviceHeight=1872`
  - DPI=226 → `scale=72/226`
  - x-centering via half-page shift.

### What was tricky to build
- Avoiding double-add of pages in UniPDF’s creator API (some APIs add implicitly when creating a new page).
- Keeping rendering logic minimal while still applying the “right” initial coordinate transform.

### What warrants a second pair of eyes
- Confirm the coordinate mapping is correct for real notebooks (especially if anchors are present but not decoded yet).
- Confirm choosing a fixed stroke width (1pt) is acceptable for this milestone, and decide how we want to incorporate thickness/brush width next.

### What should be done in the future
- Decode group anchor metadata (TreeNodeBlock) and apply anchor transforms in rendering.
- Use stroke thickness/pressure and brush types to render closer to device fidelity.
- Implement background merge and page alignment math (tasks 44+).

### Code review instructions
- Start with:
  - `remarquee/pkg/rmdoc/render/v6_strokes_pdf.go`
  - `remarquee/pkg/rmdoc/render/v6_strokes_pdf_test.go`
- Run:
  - `cd remarquee && go test ./pkg/rmdoc/render -count=1`

### Technical details
- Transform constants mirror `rmc/src/rmc/exporters/svg.py`:
  - `SCALE = 72 / 226`
  - `PAGE_WIDTH_PT = 1404 * SCALE`
  - `PAGE_HEIGHT_PT = 1872 * SCALE`

## Step 13: Compute bounding boxes for decoded strokes (partial task 43, no anchors yet)

This step adds basic bounding box computation on top of our normalized `Stroke` primitives. The immediate goal is to support downstream rendering/merge logic that needs “what area does this content occupy?” without yet decoding text anchors or group anchor offsets.

This is intentionally a **partial** implementation of task 43: we can compute bboxes for strokes and for a decoded scene tree by unioning decoded line strokes. Anchor offsets for text-linked groups require RootText/TreeNode anchor decoding and will be implemented in a follow-up step.

**Commit (code):** cb097e1 — "RMQ-0004: add stroke bounding boxes"

### What I did
- Added `BBox` primitives (`MinX/MinY/MaxX/MaxY`) with helpers:
  - `Union`, `Expand`, `IsEmpty`, `Width`, `Height`
- Implemented:
  - `BBoxForStroke(stroke, pad)`
  - `BBoxForStrokes(strokes, pad)`
  - `BBoxForRMV6SceneTree(tree, pad)` (decodes line strokes and unions their bboxes)
- Added unit tests for:
  - padding behavior
  - empty stroke behavior
  - fixture-based smoke test over a decoded V6 scene tree producing a non-empty bbox

### Why
- Merge logic and viewport calculations need bounding boxes; adding them now keeps later steps focused on PDF math rather than recomputing geometry ad-hoc.
- Keeping this “strokes-only bbox” isolated avoids coupling bbox computation to text/anchors until we’re ready.

### What worked
- `gofmt` + `go test ./... -count=1` passed.
- The fixture-based test yields a non-empty bbox from real V6 data.

### What was tricky to build
- Avoiding a premature commitment to brush-width semantics. We accept a `pad` parameter instead of guessing how to expand bounds based on brush dynamics.

### What warrants a second pair of eyes
- Decide how we should expand bbox for real rendering fidelity:
  - based on stroke `Width`/pressure, or
  - based on brush type/thickness_scale, or
  - a conservative fixed pad.

### What should be done in the future
- Implement anchor offsets:
  - decode TreeNodeBlock anchor metadata,
  - decode RootTextBlock and compute anchor positions,
  - apply anchor transforms when computing group/item bboxes.

### Code review instructions
- Start with:
  - `remarquee/pkg/rmdoc/bbox.go`
  - `remarquee/pkg/rmdoc/bbox_test.go`
- Run:
  - `cd remarquee && go test ./pkg/rmdoc -count=1`
