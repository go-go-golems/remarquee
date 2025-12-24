---
Title: Diary
Ticket: 002-REFACTOR-CLEANUP
Status: active
Topics:
    - remarquee
    - go
    - refactor
    - cleanup
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-24T09:06:56.504564823-05:00
WhatFor: ""
WhenToUse: ""
---

# Diary

## Goal

This diary captures the implementation work for ticket **002-REFACTOR-CLEANUP**: unify context propagation (CLI + UI + rmdoc internals), extract rmdoc introspection helpers into a dedicated package, and clean up `remarquee-ui` routing/path parsing.

## Step 1: Initialize workspace + diary, confirm docmgr task IDs

This step set up the docmgr-managed workspace so subsequent refactors can be tracked precisely (tasks by ID, diary steps, changelog entries). It also confirmed that the checklist in `tasks.md` is recognized by docmgr as 23 actionable tasks, so we can check them off as code lands.

### What I did
- Created `reference/01-diary.md` via `docmgr doc add`.
- Confirmed docmgr is configured via repo root `.ttmp.yaml` (ttmp root is `remarquee/ttmp`).
- Confirmed docmgr parsed the checklist in `tasks.md` into tasks **[1..23]**.

### Why
- We want tight bookkeeping: every code change should map to one or more task IDs and be discoverable later (reverse lookup: docs → code files).

### What worked
- `docmgr task list --ticket 002-REFACTOR-CLEANUP` shows 23 open tasks that match `tasks.md`.
- `docmgr doc add` created the diary doc at `.../reference/01-diary.md`.

### What didn't work
- N/A

### What I learned
- docmgr treats `tasks.md` as the task source-of-truth and assigns stable numeric IDs automatically; we can use `docmgr task check --id ...` without manually maintaining a separate task DB.

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- N/A

### What should be done in the future
- N/A

### Code review instructions
- Start with `remarquee/ttmp/.../002-REFACTOR-CLEANUP.../tasks.md` and `reference/01-diary.md` to see the workplan and execution log.

### Technical details
- Commands used (from repo root):
  - `docmgr status --summary-only`
  - `docmgr ticket list --ticket 002-REFACTOR-CLEANUP`
  - `docmgr task list --ticket 002-REFACTOR-CLEANUP`
  - `docmgr doc add --ticket 002-REFACTOR-CLEANUP --doc-type reference --title "Diary"`

### What I'd do differently next time
- N/A

## Step 2: Replace `context.Background()` with cobra/request context in CLI + UI

This step removed several “context black holes” where CLI commands and HTTP handlers were unconditionally switching to `context.Background()`. The practical impact is that cancellations/timeouts (Cobra context, HTTP client disconnects, reverse proxy deadlines) can now flow through to `pkg/rmdoc` and `pkg/rmdoc/render` calls.

**Commit (code):** 4b73281 — "rmdoc: propagate cobra/request context"

### What I did
- Updated CLI commands to pass the `ctx` they receive into `pkg/rmdoc.OpenFile` and `rmdoc/render.BuildBackgroundPDF`.
- Updated `remarquee-ui` handlers to use `r.Context()` instead of `context.Background()`.

### Why
- We want cancellation to be consistent end-to-end, and to enable later work where `pkg/rmdoc` internals actively honor `ctx` during potentially long reads/renders.

### What worked
- `gofmt` + `go test ./... -count=1` passed for the `remarquee` module.

### What didn't work
- N/A

### What I learned
- In these commands, the “right” context was already available; the bug was simply ignoring it (and in one place explicitly discarding it via `_ = ctx`).

### What was tricky to build
- N/A (straightforward mechanical refactor).

### What warrants a second pair of eyes
- Confirm `cli.BuildCobraCommand` is indeed passing Cobra’s `cmd.Context()` into the `Run(ctx, ...)` methods (it should, but it’s worth verifying once when we later wire cancellations deeper).

### What should be done in the future
- Once `pkg/rmdoc/open.go` and `pkg/rmdoc/render/background.go` honor cancellation, add an integration-style test that exercises the UI handlers with a canceled request context to ensure the server returns promptly.

### Code review instructions
- Start with:
  - `cmd/remarquee/cmds/rmdoc/inspect.go` (`(*InspectCommand).Run`)
  - `cmd/remarquee/cmds/rmdoc/build_background.go` (`(*BuildBackgroundCommand).Run`)
  - `cmd/remarquee-ui/api/inspect.go`, `internal_structure.go`, `render.go`
- Validate with:
  - `cd remarquee && go test ./... -count=1`

### Technical details
- CLI: `pkg_rmdoc.OpenFile(ctx, ...)` now uses the propagated context rather than `context.Background()`.
- UI: `ctx := r.Context()` is now used for `OpenFile` and background rendering.

### What I'd do differently next time
- N/A

## Step 3: Make `pkg/rmdoc` open + background render honor cancellation + add tests

This step implemented real cancellation points in the rmdoc “hot paths”: reading zip entries and building background PDFs. The immediate impact is that callers can now reliably stop work at predictable boundaries (before parsing, during large entry reads, and between per-page render steps).

**Commit (code):** 48d822e — "rmdoc: honor ctx in open/render"

### What I did
- Implemented `ctx.Err()` checks in `OpenReaderAt` and `BuildBackgroundPDF`.
- Replaced `io.ReadAll` with a chunked read loop in `readZipFile` that checks `ctx.Err()` between reads.
- Added deterministic cancellation tests:
  - `pkg/rmdoc/open_cancel_test.go`
  - `pkg/rmdoc/render/background_cancel_test.go`

### Why
- Even after fixing the call sites (Step 2), `pkg/rmdoc` still ignored cancellations internally, so client disconnects and CLI cancellations wouldn’t stop expensive reads or long render loops.

### What worked
- `cd remarquee && go test ./... -count=1` passed.
- Cancellation tests are non-flaky because they use a custom `context.Context` that returns `context.Canceled` after a fixed number of `Err()` calls (no timing-based sleeps).

### What didn't work
- N/A

### What I learned
- Zip entry reads are not natively context-aware; to honor cancellation you need an explicit read loop with periodic `ctx.Err()` checks.

### What was tricky to build
- Making sure cancellations propagate as `ctx.Err()` (not just wrapped I/O errors) so upstream code can reliably detect cancellations.

### What warrants a second pair of eyes
- The `readZipFile` preallocation guard (`<= 64MiB`) is a pragmatic heuristic; confirm it’s acceptable for typical `.rmdoc` sizes and won’t surprise callers with very large docs.

### What should be done in the future
- If we see meaningful performance regressions from chunked reads, consider tuning the buffer size or adding a fast-path for small entries while preserving cancellation checks.

### Code review instructions
- Start with:
  - `pkg/rmdoc/open.go` (`OpenReaderAt`, `readZipFile`)
  - `pkg/rmdoc/render/background.go` (`BuildBackgroundPDF`)
  - `pkg/rmdoc/open_cancel_test.go`
  - `pkg/rmdoc/render/background_cancel_test.go`
- Validate with:
  - `cd remarquee && go test ./... -count=1`

### Technical details
- `readZipFile` reads with a `32*1024` buffer and checks `ctx.Err()` before each `Read`.
- `BuildBackgroundPDF` checks `ctx.Err()` at function entry and at the top of each page iteration.

### What I'd do differently next time
- N/A

## Step 4: Document/implement cancellation semantics for rmapi legacy PDF generation

This step clarifies a key limitation: the rmapi legacy PDF generator API does not accept a `context.Context`, so we cannot truly interrupt generation once it begins. The goal here is to make this behavior explicit (in code and docs) and to avoid kicking off expensive generation when the request is already canceled.

**Commit (code):** 20d6ce3 — "ui: precheck ctx before legacy render"

### What I did
- Added a `ctx.Err()` pre-check in the legacy render handler before calling `generator.Generate()`.
- Added an inline comment documenting the “cannot interrupt” constraint.

### Why
- Without an explicit decision, cancellation semantics are ambiguous and can lead to surprising resource usage (work continues even though the client disconnected).

### What worked
- `cd remarquee && go test ./... -count=1` passed after the change.

### What didn't work
- N/A

### What I learned
- Some third-party APIs (like the rmapi generator here) are intrinsically non-cancelable; we can only:
  - avoid starting the work if already canceled, and
  - rely on the HTTP stack to abort writes once the client is gone (but CPU/disk work may still continue if we started it).

### What was tricky to build
- Choosing the least-surprising behavior without spawning background goroutines that outlive request lifetimes.

### What warrants a second pair of eyes
- Whether the UI should return a structured JSON error when `ctx.Err()!=nil` (vs just returning early). Right now it “best-effort” returns without writing because request cancellation typically means the client is gone anyway.

### What should be done in the future
- If we need true interruptibility, we’ll need either:
  - a fork/wrapper of rmapi that accepts a context and checks it internally, or
  - running generation in a separate process that we can terminate (operationally heavier).

### Code review instructions
- Start in `cmd/remarquee-ui/api/render.go` under `HandleRenderLegacy`.

### Technical details
- The pre-check is intentionally placed immediately before starting generation so it covers any earlier work (manifest lookup, schema checks, output path calculations).

### What I'd do differently next time
- N/A

## Related

<!-- Link to related documents or resources -->
