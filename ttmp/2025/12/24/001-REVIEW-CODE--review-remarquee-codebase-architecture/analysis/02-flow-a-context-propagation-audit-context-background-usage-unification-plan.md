---
Title: Flow A context propagation audit (context.Background usage + unification plan)
Ticket: 001-REVIEW-CODE
Status: active
Topics:
    - remarquee
    - go
    - architecture
    - review
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Audit of context usage for Flow A (inspect a .rmdoc): where context.Background() is used today, what IO it covers, and a concrete plan for unified context propagation across CLI/UI/pkg."
LastUpdated: 2025-12-24T08:44:13.911841037-05:00
WhatFor: "Make cancellation/timeouts/traceability consistent when inspecting .rmdoc documents; identify precise improvement points and the work needed to unify context flow."
WhenToUse: "Use when adding server-side rendering jobs, introducing timeouts, debugging long-running requests, or refactoring pkg/rmdoc to be more idiomatic and cancellable."
---

# Flow A context propagation audit (context.Background usage + unification plan)

## Executive summary

Flow A (“inspect a `.rmdoc` and show schema + page plan”) exists in both the **CLI** and **UI server**, and today it uses `context.Background()` in several call sites. Even more importantly, the core library functions (`pkg/rmdoc.OpenReaderAt`, `pkg/rmdoc/render.BuildBackgroundPDF`) currently **ignore the context entirely**, so even if call sites pass a meaningful `ctx`, cancellation/timeouts won’t actually interrupt the underlying work.

This document answers two questions:

- **Where is `context.Background()` used today in Flow A, and what I/O does it wrap?**
- **What does it take to have a unified, end-to-end context flow?** (from Cobra / net/http boundaries down into `pkg/rmdoc`)

## Flow A recap (what we mean by “Flow A”)

Flow A is the “inspect” path: open a `.rmdoc` and derive the page plan.

### CLI flavor

- **Command**: `remarquee rmdoc inspect <file.rmdoc>`
- **Implementation**: `remarquee/cmd/remarquee/cmds/rmdoc/inspect.go`
- **Core library call**: `remarquee/pkg/rmdoc.OpenFile(ctx, path)` (which delegates to `OpenReaderAt`)

### UI flavor

- **Endpoint**: `GET /api/document/{id}/inspect`
- **Implementation**: `remarquee/cmd/remarquee-ui/api/inspect.go`
- **Core library call**: `remarquee/pkg/rmdoc.OpenFile(ctx, docPath)`

## Current state: where `context.Background()` is used (Flow A + directly related)

This table is intentionally concrete. When we say “what I/O”, we mean the actual operations that are triggered by the call chain from that location.

### A. Flow A call sites (inspect paths)

- **CLI**: `remarquee/cmd/remarquee/cmds/rmdoc/inspect.go`
  - **Symbol**: `(*InspectCommand).Run`
  - **Code**: `pkg_rmdoc.OpenFile(context.Background(), s.File)`
  - **I/O triggered** (via `pkg/rmdoc/open.go`):
    - `os.Open(p)`
    - `f.Stat()`
    - `zip.NewReader(r, size)`
    - read zip entries via `zip.File.Open()` + `io.ReadAll(...)` for:
      - `.content` (required)
      - optional `.metadata`, `.pagedata`, `.pdf` (if present)
  - **Why it matters**:
    - ignores Cobra’s `cmd.Context()` (so CTRL-C / cancellation semantics can’t propagate)
    - even if it passed `ctx`, `OpenReaderAt` currently ignores it (see below), so it would still not be cancellable.

- **CLI**: `remarquee/cmd/remarquee/cmds/rmdoc/inspect.go`
  - **Symbol**: `(*InspectCommand).RunIntoGlazeProcessor`
  - **Code**: `pkg_rmdoc.OpenFile(ctx, s.File)` (✅ uses provided ctx)
  - **I/O triggered**: same as above
  - **Why it matters**:
    - this method is “closer” to ideal call-site behavior, but the library still ignores `ctx` today.
    - it creates inconsistency within the same command implementation: `Run` vs `RunIntoGlazeProcessor` behave differently wrt cancellation semantics.

- **UI**: `remarquee/cmd/remarquee-ui/api/inspect.go`
  - **Symbol**: `HandleInspect(...)` handler
  - **Code**: `ctx := context.Background(); rmdoc.OpenFile(ctx, docPath)`
  - **I/O triggered**: same as above
  - **Why it matters**:
    - a server boundary already has a context: `r.Context()`. Using `context.Background()` discards request cancellation/timeouts and trace IDs (if introduced later).

### B. Flow A adjacent call sites (same open/parse pipeline)

These are not “Flow A inspect” per se, but they exercise the same open/parse pipeline and are relevant to unification because they show the pattern repeating.

- **UI**: `remarquee/cmd/remarquee-ui/api/internal_structure.go`
  - **Symbol**: `HandleInternalStructure(...)` handler
  - **Code**: `ctx := context.Background(); rmdoc.OpenFile(ctx, docPath)`
  - **I/O triggered**:
    - same `.rmdoc` open pipeline
    - plus zip listing (manual `zip.NewReader(...)` in handler) and header sniff of `.rm` files (`Read` first 43 bytes)

- **UI**: `remarquee/cmd/remarquee-ui/api/render.go`
  - **Symbol**: `HandleRenderBackground(...)`, `HandleRenderLegacy(...)`
  - **Code**: `ctx := context.Background(); rmdoc.OpenFile(ctx, docPath)` and background/legacy render calls
  - **I/O triggered**:
    - `.rmdoc` open pipeline
    - background render: `unipdf` page reads + PDF writing to `outputs/*.pdf`
    - legacy render: rmapi `PdfGenerator.Generate()` writes output PDF

- **CLI**: `remarquee/cmd/remarquee/cmds/rmdoc/build_background.go`
  - **Symbol**: `(*BuildBackgroundCommand).Run`
  - **Code**:
    - `pkg_rmdoc.OpenFile(context.Background(), s.File)`
    - `rmdocrender.BuildBackgroundPDF(context.Background(), doc, ...)`
  - **I/O triggered**:
    - `.rmdoc` open pipeline
    - unipdf parsing/assembly
    - `os.WriteFile(out, bg, ...)`

## The deeper issue: contexts are accepted but ignored

Even if we replace all `context.Background()` usage at call sites, **Flow A still won’t be cancellable** until the libraries actually *honor* the context.

### `pkg/rmdoc/open.go`

- **Symbol**: `OpenReaderAt(ctx context.Context, r io.ReaderAt, size int64) (*Document, error)`
- **Current behavior**: `_ = ctx // reserved for future cancellation / IO abstraction`
- **Impact**:
  - no `ctx.Err()` checks
  - no opportunity to stop mid-read for big `.content`/`.pdf`

### `pkg/rmdoc/render/background.go`

- **Symbol**: `render.BuildBackgroundPDF(ctx context.Context, doc *rmdoc.Document, opts BackgroundOptions) ([]byte, error)`
- **Current behavior**: `_ = ctx // reserved for future cancellation`
- **Impact**:
  - building large PDFs in the UI server cannot be stopped when the HTTP client disconnects.

## What “unified context flow” should mean (design target)

Unifying context isn’t “pass ctx everywhere” as a cosmetic change. The goal is:

- **Boundary contexts** are the root of truth:
  - CLI: `cmd.Context()` is the canonical context.
  - HTTP: `r.Context()` is the canonical context.
- **Every operation that can be slow should accept context** and check it at reasonable boundaries.
- **Cancellation should stop work quickly enough to matter**, even if we can’t interrupt every syscall or third-party library call.

### Diagram: desired propagation (Flow A)

```text
CLI: Cobra cmd.Context()
  -> rmdoc inspect Run(ctx)
     -> pkg/rmdoc.OpenFile(ctx, path)
        -> OpenReaderAt(ctx, ...)
           -> (ctx checks during zip reads)
           -> ParseContent(...)
           -> ApplyPagedataTemplates(...)
     -> output

HTTP: net/http r.Context()
  -> HandleInspect(w,r)
     -> pkg/rmdoc.OpenFile(r.Context(), docPath)
     -> JSON response
```

## Concrete improvement list (filename + symbol)

This section is the “punch list”: it’s designed so a developer can grep the symbols and fix them systematically.

### 1) Fix call sites to pass the right context

- `remarquee/cmd/remarquee/cmds/rmdoc/inspect.go`
  - `(*InspectCommand).Run`: replace `context.Background()` with the passed `ctx` (or `cmd.Context()` higher up, if wiring changes).

- `remarquee/cmd/remarquee/cmds/rmdoc/build_background.go`
  - `(*BuildBackgroundCommand).Run`: replace both `context.Background()` usages with `ctx`.

- `remarquee/cmd/remarquee-ui/api/inspect.go`
  - `HandleInspect`: replace `context.Background()` with `r.Context()`.

- `remarquee/cmd/remarquee-ui/api/internal_structure.go`
  - `HandleInternalStructure`: replace `context.Background()` with `r.Context()`.

- `remarquee/cmd/remarquee-ui/api/render.go`
  - `HandleRenderBackground` and `HandleRenderLegacy`: replace `context.Background()` with `r.Context()`.

### 2) Make the libraries honor the context

- `remarquee/pkg/rmdoc/open.go`
  - `OpenReaderAt`: add `ctx.Err()` checks:
    - before opening zip entries
    - between reading `.content` / `.metadata` / `.pagedata` / `.pdf`
  - `readZipFile`: replace `io.ReadAll` with a context-aware read loop.

- `remarquee/pkg/rmdoc/render/background.go`
  - `BuildBackgroundPDF`: add `ctx.Err()` checks:
    - after opening payload PDF
    - inside the loop over `doc.Pages` (between pages)

### 3) Decide what “cancellation” means for third-party calls

Some third-party APIs are not context-aware:

- rmapi legacy PDF generation: `rmapi/annotations.PdfGenerator.Generate()` has no context.
  - A pragmatic approach: run it in a goroutine and select on `ctx.Done()` to stop waiting.
  - Important limitation: this does **not** interrupt rmapi’s internal work; it only prevents the caller from blocking forever.

## What I/O is actually affected (and what isn’t)

It’s important to be honest about what context can and cannot do in this design:

- **Will improve meaningfully**:
  - aborting long zip reads (especially `.pdf`) and large in-memory operations by checking `ctx` between chunk reads
  - aborting multi-page background PDF assembly between pages
  - terminating HTTP requests quickly on client disconnect (handler returns early)

- **Will not improve fully without deeper changes**:
  - canceling a single blocking read syscall mid-flight (Go can’t generally interrupt all I/O)
  - canceling rmapi legacy PDF generation mid-render (no context support)
  - canceling unipdf operations inside a single expensive call (only coarse-grained checks are realistic)

## Minimal implementation sketch (for future refactor)

This is the core technique needed in `pkg/rmdoc` to honor context during reads:

```go
func readAllWithContext(ctx context.Context, r io.Reader) ([]byte, error) {
    buf := make([]byte, 0, 64*1024)
    tmp := make([]byte, 32*1024)
    for {
        if err := ctx.Err(); err != nil {
            return nil, err
        }
        n, err := r.Read(tmp)
        if n > 0 {
            buf = append(buf, tmp[:n]...)
        }
        if err == io.EOF {
            return buf, nil
        }
        if err != nil {
            return nil, err
        }
    }
}
```

## Risks and review notes

- **Behavioral change**: callers may start seeing `context.Canceled` / `context.DeadlineExceeded` errors where previously the system “just kept working”.
- **Partial outputs**: for background render, cancellation mid-loop should not write a partially complete PDF to disk unless we explicitly decide that’s acceptable (likely not).
- **Test updates**: tests that currently use `context.Background()` can stay, but we should add at least one test that cancels a context and asserts early return (for both open and background render).

