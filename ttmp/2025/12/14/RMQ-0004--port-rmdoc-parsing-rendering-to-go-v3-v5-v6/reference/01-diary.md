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
    - Path: remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/reference/06-diary-rmdoc-format-analysis-and-go-reimplementation-prep.md
      Note: Prior research diary that this ticket builds on
    - Path: remarquee/ttmp/2025/12/14/RMQ-0004--port-rmdoc-parsing-rendering-to-go-v3-v5-v6/design-doc/01-design-go-rmdoc-data-model-and-apis.md
      Note: Design decisions and API proposal for this ticket
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-14T20:59:20.929388092-05:00
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
