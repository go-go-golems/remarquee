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
