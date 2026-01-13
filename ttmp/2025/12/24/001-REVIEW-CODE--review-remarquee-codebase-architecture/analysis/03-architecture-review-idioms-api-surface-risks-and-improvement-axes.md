---
Title: Architecture review (idioms, API surface, risks, and improvement axes)
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
Summary: "Deep architecture review of the remarquee module: boundaries, APIs, idiomatic Go considerations, tradeoffs, and what may become problematic as the project grows."
LastUpdated: 2025-12-24T08:44:13.985297993-05:00
WhatFor: "Provide a deep, multi-axis review of the codebase organization and API design, highlighting what’s working, what’s non-idiomatic, and where future complexity/risk may accumulate."
WhenToUse: "Use when planning refactors, adding new major capabilities (new document formats, new renderers, background jobs), or doing a design review before scaling adoption."
---

# Architecture review (idioms, API surface, risks, and improvement axes)

## Executive summary

`remarquee` has a clear, pragmatic architecture: **two shallow entrypoints** (CLI + UI server) built on a set of **small, purpose-specific libraries** (`pkg/rmdoc`, `pkg/rmcloud`, `pkg/mdpdf`, `pkg/doc`). The core success is `pkg/rmdoc`’s “portable model” (`Document` + `[]PageRef`), which allows multiple tools to reuse the same parsing semantics and build different artifacts (inspect output, background PDFs, legacy PDFs, UI JSON APIs).

The main non-idiomatic / future-risk areas are not “bad code”, but places where growth will amplify today’s shortcuts:

- **Context propagation is not unified** (many `context.Background()` call sites; core libs ignore ctx). This is mostly ok for a CLI, but becomes a bigger problem for the UI server as rendering work becomes heavier. See the dedicated context audit doc in this ticket.
- **`remarquee-ui` currently bakes in a ticket directory path** for validation persistence. That’s convenient during development but fragile as ticket structures evolve.
- **Memory usage is “read everything into RAM”** for `.rmdoc` payload PDF bytes. This keeps APIs simple but may become limiting for large PDFs and server use.
- **Inconsistent “framework stack”**: some commands are Glazed-based (dual-mode output), others are Cobra-only. This is workable, but it creates multiple “command authoring styles” and raises the onboarding cost.

Overall: the architecture is coherent and is “ok for now” for a single-team tool. If you plan to grow the UI usage or add more render/transform features, context+resource constraints and API boundaries are the likely first pressure points.

## What exists today (map, but with rationale)

### Entrypoints

- **CLI**: `remarquee/cmd/remarquee/main.go`
  - Cobra root with Glazed help/logging integration.
  - Wires command groups: `cloud`, `rmdoc`, `upload`, `ocr`.
  - Design intent: keep root thin; keep heavy logic in packages and command implementations.

- **UI server**: `remarquee/cmd/remarquee-ui/main.go`
  - `net/http` server with JSON API handlers.
  - Embeds React/Vite frontend in prod (`embed.go`).
  - Design intent: a human-in-the-loop validation harness that exercises `pkg/rmdoc` and persists “review sessions”.

### Library layer (`pkg/*`)

- `pkg/rmdoc`: zip opening + schema/type detection + deterministic UI page plan (`[]PageRef`).
- `pkg/rmdoc/render`: background PDF assembly in UI order (UniPDF).
- `pkg/rmcloud`: rmapi auth bootstrap + “mkdir -p” against rmapi’s filetree.
- `pkg/mdpdf`: markdown preprocessing + pandoc invocation (pdf output).
- `pkg/doc`: embedded markdown help docs used by CLI help system.

## Analysis axis 1: Are the package boundaries sane?

Yes, mostly. The boundaries reflect *how the tools are used*:

- **`pkg/rmdoc` is a domain library**: it is used by both CLI and UI.
- **`pkg/rmcloud` is an integration shim**: it keeps rmapi token bootstrap out of command code.
- **`pkg/mdpdf` is a pipeline helper**: it isolates pandoc quirks and preprocessing.
- **`cmd/*` are glue layers**: parse inputs, call into packages, format outputs.

Where it gets a bit fuzzy:

- `cmd/remarquee-ui/api/*` duplicates some `.rmdoc` zip introspection logic (listing files + sniffing `.rm` header versions) that arguably belongs in a shared library (either in `pkg/rmdoc` or a small `pkg/rmdoc/debug`).
  - This is not “wrong”, but it’s a signal: if the UI grows, it will want shared utilities.

## Analysis axis 2: Is the API surface idiomatic Go?

### The good

- `pkg/rmdoc` has a small API and uses plain structs for results. That’s idiomatic and easy to integrate.
- Errors are consistently wrapped (via `github.com/pkg/errors`), which helps debugging.
- Options structs exist where appropriate (`render.BackgroundOptions`, `mdpdf.PandocOptions`).

### The less-idiomatic / “watch this as you grow”

- Context parameters exist but are currently unused in key places (`OpenReaderAt`, `BuildBackgroundPDF`). In idiomatic Go, a context argument usually signals “this can be canceled / has deadlines”.
  - If you keep the context parameters but don’t honor them, you risk surprising callers later.
  - If you decide context isn’t meaningful, consider removing it (but for servers, it’s usually worth making it meaningful).

- `Document` currently stores raw blobs (`PayloadPDF []byte`, raw JSON bytes). That’s convenient for small docs, but a server that handles large PDFs will feel this quickly.
  - More idiomatic for big payloads would be a closable “document handle” with streaming/lazy access.

## Analysis axis 3: Dependency choices and “API gravity”

### Glazed + Cobra (CLI)

The codebase effectively has two CLI “authoring styles”:

- **Glazed-based commands** (e.g. `cloud refresh`, `rmdoc inspect`)
  - pros: structured output, shared parameter layer pattern, easy to add JSON output
  - cons: extra conceptual overhead for contributors unfamiliar with Glazed

- **Cobra-only commands** (e.g. `upload md/bundle/src`)
  - pros: straightforward Go + Cobra; fewer moving parts
  - cons: harder to standardize output formats; duplicated flag patterns

This is ok for now, but it’s worth deciding long-term whether you want:

- “all commands are Glazed commands wrapped into Cobra”, or
- “Glazed is for a subset of commands where structured output is essential”.

### rmapi

rmapi is the backbone for cloud operations and legacy PDF rendering. That means:

- Your semantics for cloud filetree behavior, naming, and legacy annotation output are coupled to rmapi’s model.
- That coupling is fine as long as rmapi remains stable, but it’s important to isolate it behind adapters (`pkg/rmcloud` is a good start).

### UniPDF

UniPDF does the heavy lifting for PDF assembly. The code already contains a subtle “page duplication” workaround; that’s a sign that:

- You’ll want tests for background PDF assembly and page counts.
- Refactors should preserve these “gotchas” explicitly (they’re easy to lose).

## Analysis axis 4: Operational concerns (server vs CLI)

### UI server: synchronous rendering

`remarquee-ui` handlers currently do work synchronously in the request path (open doc, render pdf, write output). This is ok for a local dev tool. As the server becomes more used:

- You may want a job queue / async model:
  - enqueue render job
  - return job ID
  - poll job status
  - stream outputs

### Cancellation + deadlines

HTTP requests inherently have cancellation (`r.Context()`), but today it’s not used, and core libs ignore ctx anyway. This is ok for a dev harness, but becomes problematic under load or with large documents. (See the dedicated Flow A context audit.)

## Analysis axis 5: Data model correctness and schema/version handling

The `.rmdoc` parsing strategy is pragmatic:

- **Schema detection** by `"cPages"` key presence.
- **Legacy parsing** uses a partial envelope and derives a page plan.
- **cPages parsing** filters deleted pages and reindexes.
- `.pagedata` is applied only if templates aren’t already set.

Risks to watch as you grow:

- Schema detection via a single key is brittle against upstream schema changes.
- Correctness depends on the meaning of `redir.value` for cPages being aligned with “0-based payload PDF page index”.
- Legacy vs cPages differ in how “deleted pages” are represented; any consumer expecting deleted pages to exist in `Pages` will be surprised (cPages deleted pages are currently filtered out).

## Analysis axis 6: Resource usage and scaling risks

### “Read everything into memory”

`pkg/rmdoc.OpenReaderAt` reads:

- `.content` into memory (fine)
- optional `.pdf` payload into memory (can be large)

This simplifies downstream code (background render can use `bytes.NewReader(doc.PayloadPDF)`), but the tradeoff is:

- peak memory usage ~ payload size + output size + overhead
- high GC pressure for large docs

If the UI server becomes multi-user or you process many docs concurrently, you’ll likely want:

- lazy/streaming access to payload PDF
- or a temp-file strategy (extract `.pdf` entry to disk and pass a reader)

## Analysis axis 7: Security and safety posture

Good early choices:

- `cloud rm` requires `--yes` and prints “would delete” preview.
- `outputs` endpoint has a basic path traversal guard.

Things that will matter if the UI server ever becomes remote:

- authentication/authorization (currently none)
- stricter output serving rules (limit file types, restrict directories)
- rate limiting / job limiting (render endpoints can be expensive)

## Analysis axis 8: Testing strategy

Positive signals:

- `pkg/rmdoc` has unit tests and integration tests.
- `pkg/mdpdf` has tests for preprocess/bundle behavior.

Suggested additions if you invest further:

- a cancellation test once contexts are honored (open + background render)
- a golden test for background PDF page count/order for both schema types
- UI handler tests that validate status codes and response schemas

## “Ok for now” vs “fix soon” vs “invest later”

### Ok for now (as a dev-oriented tool)

- synchronous UI handlers
- mixed command authoring styles (Glazed vs Cobra-only)
- in-memory payload PDFs

### Fix soon (low cost, high leverage)

- unify context flow at call sites (`cmd.Context()` / `r.Context()`) and honor it in libraries
- convert `ticketDir` in `remarquee-ui/main.go` into a flag/env var (avoid baking ticket paths into code)
- replace manual path parsing (`strings.Split(...)`) with Go’s newer mux patterns (if you want; Go’s `ServeMux` has improved)

### Invest later (when adoption grows)

- streaming/lazy `.pdf` payload handling in `pkg/rmdoc`
- background job model for UI rendering endpoints
- “debug/introspection” helpers extracted from UI handlers into shared libraries

## Cross-reference

- Context specifics (Flow A): see `analysis/02-flow-a-context-propagation-audit-context-background-usage-unification-plan.md`.

