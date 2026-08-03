---
Title: Diary
Ticket: RMQ-0021
Status: active
Topics:
    - cli
    - rmdoc
    - rendering
    - remarkable
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Chronological investigation and implementation diary for adding --annotations-only to rmdoc render-v6.
LastUpdated: 2026-08-03T18:06:17.662214553-04:00
WhatFor: Resume work on RMQ-0021; understand what was tried, what failed, and what is next.
WhenToUse: Before modifying render-v6 or the pkg/rmdoc/render pipeline for annotations-only output.
---

# Diary

## Goal

Add an `--annotations-only` flag to `remarquee rmdoc render-v6` with parity to `render-legacy --annotations-only`, and produce an intern-ready analysis/design/implementation guide in this ticket, then upload the ticket docs to reMarkable.

## Step 1: Orient in the repo and map the two render verbs

This step established the lay of the land before any design claims were made. The goal was to locate both render verbs, the shared page-selection machinery, and the library primitives they call, so that the eventual guide could be evidence-anchored rather than speculative.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to add a --annotations-only fag to the v6 render verb so that we can render only annotations just like for the render-legacy verb. Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.


Keep a detailed diary as you work (see diary format skill) and commit at appropriate intervals."

**Assistant interpretation:** Create a docmgr ticket for the `--annotations-only` feature on `render-v6`, write a long-form intern-ready design guide (prose + bullets + pseudocode + diagrams + API/file references), keep a diary, commit at sensible checkpoints, and upload the resulting documents to the reMarkable.

**Inferred user intent:** They want the feature scoped and designed so a new engineer can implement it without prior codebase knowledge, and they want the deliverable readable on the tablet.

**Commit (code):** none (investigation only)

### What I did
- Ran `docmgr status --summary-only`, `git log`, and listed the repo; confirmed docmgr root at `ttmp/` with 30 tickets and confirmed the next free ticket number is RMQ-0021 (RMQ-0020 is the newest).
- Located the verbs: `cmd/remarquee/cmds/rmdoc/render_v6.go` (251 lines) and `render_legacy.go` (265 lines); confirmed `render-legacy` already has `AnnotationsOnly bool \`glazed:"annotations-only"\`` at line 36, passed to rmapi at lines 192-197.
- Read the render pipeline: `pkg/rmdoc/render/v6_merge_background.go` (1122 lines), `background.go`, `v6_strokes_pdf.go`; the CLI helpers `pages.go`, `input_resolver.go`, `root.go`; tests `render_v6_test.go`, `v6_merge_background_test.go`.
- Read the vendored rmapi fork's `annotations/pdf.go` to pin down legacy `AnnotationsOnly`/`AllPages` semantics (skip-unannotated at line 96; blank-page swap at line 226).
- Read prior ticket docs for house style: RMQ-0019 design doc (`--pages`) and the writing-style/deliverable-checklist references of the ticket-research skill.

### Why
- The guide must teach the whole system to an intern; that requires accurate architecture mapping with file:line anchors, not guesses.

### What worked
- Found the pivotal implementation fact early: the merge pipeline **already has an "overlay-only" page path** (`buildOverlayOnlyPageBBoxScaled`, `v6_merge_background.go:755`) used when a background page has no content; the feature is essentially "force that path for every page and skip the background".
- The two merge loops (`:272` and `:506`) are near-copies — directly informs DR-1 (dedicated function vs. threading a bool).

### What didn't work
- Nothing blocking.

### What I learned
- Glazed command anatomy: settings struct + `fields.New` definitions + `CommandDescription` + `execute` shared by `Run`/`RunIntoGlazeProcessor` + `cli.BuildCobraCommand`.
- Page selection is pre-render for v6 (`MergeRMDocV6OntoBackgroundPDFWithInfoForPages` takes 0-based indices) but post-render extraction for legacy (`extractPDFPages`, `pages.go:109`) — which is why legacy forces `AllPages` when `--pages` is used.

### What was tricky to build
- n/a (reading phase)

### What warrants a second pair of eyes
- n/a (no code changes)

### What should be done in the future
- n/a

### Code review instructions
- Start at `cmd/remarquee/cmds/rmdoc/render_v6.go:144` (`execute`) and follow into `pkg/rmdoc/render/v6_merge_background.go:272`.

### Technical details
- Branch: `task/remarquee-v6-only-annotations` (worktree at `/home/manuel/workspaces/2026-08-03/remarquee-v6-only-annotations/remarquee`).

## Step 2: Reproduce the real-world failure with the TTC Garden workbook

The user pointed at a concrete document on the tablet to use as the running example. Downloading it turned the ticket from "parity feature" into "currently impossible workflow": the legacy verb hard-fails on this document because rmapi cannot parse V6 `.rm` files.

### Prompt Context

**User prompt (verbatim):** "/ai/2026/08/03/TTC-GARDEN-UXQA-001/TTC Garden Human Calibration Workbook Use this page 1 - 2 - 3 as example" and "(on the tablet)"

**Assistant interpretation:** Use the named reMarkable cloud document (pages 1–3) as the worked example for the design guide.

**Inferred user intent:** Ground the design in a real document they care about; pages 1–3 are the annotated pages worth exercising.

**Commit (code):** none

### What I did
- Listed the cloud dir: `go run ./cmd/remarquee cloud ls "/ai/2026/08/03/TTC-GARDEN-UXQA-001" --long --non-interactive` → found the workbook file.
- First download attempt failed: `cloud get ... --out /tmp/ttc-workbook/workbook.rmdoc` → `exit status 1` (the verb has no `--out`; only `--out-dir`). Re-ran with `--out-dir /tmp/ttc-workbook --non-interactive` → `OK: downloaded ... -> /tmp/ttc-workbook/TTC Garden Human Calibration Workbook.rmdoc` (286,022 bytes).
- Inspected: `rmdoc inspect` → `schema=legacy type=pdf pages=44`.
- Verified `.rm` headers with a Python zip read: exactly 3 `.rm` files (for UI pages 1–3), each starting `reMarkable .lines file, version=6`.
- Ran `rmdoc render-v6 workbook.rmdoc --pages 1-3` → ✅ `ok: wrote /tmp/ttc-workbook/workbook-v6-p1-3.pdf` (3 pages, 612×792 pt letter per `pdfinfo`).
- Ran `rmdoc render-legacy workbook.rmdoc --pages 1-3 --annotations-only` → ❌ `Error: Unknown header` (also fails without `--annotations-only`).

### Why
- A design doc for an intern needs a concrete, reproducible motivation; "the other verb crashes on real documents from current devices" is the strongest possible justification.

### What worked
- The hybrid nature of the document (legacy `.content` schema + V6 `.rm` files) exercises `archiveHasV6RM` (`render_v6.go:128`), and `render-v6` handled it fine.
- The fact that only pages 1–3 have `.rm` files makes it a perfect case for the skip-vs-blank page-emission decision (DR-2).

### What didn't work
- `cloud get --out <file>` is not a valid flag combination; error output was terse. Fix: use `--out-dir` and accept the document-name-derived filename.
- `ask_questions_about_images` on the rasterized pages failed with a VLM backend TLS error (`remote error: tls: bad record MAC`) — visual description of the example pages deferred; will retry or fall back to pdfinfo/structure-level facts.

### What I learned
- Modern devices write V6 `.rm` files even into legacy-schema archives; schema detection alone cannot pick the renderer — the `archiveHasV6RM` probe exists precisely for this.

### What was tricky to build
- n/a (reproduction phase)

### What warrants a second pair of eyes
- n/a

### What should be done in the future
- Retry the visual description of pages 1–3 for the final handoff bundle (non-blocking for the design).

### Code review instructions
- Re-run the reproduction block in the design doc's "Verified reproduction" section; all commands are copy-pasteable.

### Technical details
- Local copy: `/tmp/ttc-workbook/TTC Garden Human Calibration Workbook.rmdoc`; renders in `/tmp/ttc-workbook/workbook-v6-p1-3.pdf` and `v6-{1,2,3}.png`.

## Step 3: Create the ticket workspace and write the design guide

With evidence in hand, this step created RMQ-0021 and wrote the primary deliverable: a 10-section, intern-ready analysis/design/implementation guide with pseudocode, ASCII dataflow diagrams, five decision records, a 4-phase implementation plan, and a fixture-based test matrix.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Proceed with ticket creation and the long-form guide.

**Inferred user intent:** Same as Step 1.

**Commit (code):** none yet (docs to be committed next)

### What I did
- `docmgr ticket create-ticket --ticket RMQ-0021 --title "Add --annotations-only flag to rmdoc render-v6" --topics cli,rmdoc,rendering,remarkable` → `ttmp/2026/08/03/RMQ-0021--add-annotations-only-flag-to-rmdoc-render-v6`.
- `docmgr doc add --doc-type design-doc ...` and `--doc-type reference --title "Diary"`.
- Collected exact line numbers for every referenced symbol (`grep -n` across `render_v6.go`, `render_legacy.go`, `v6_merge_background.go`, `background.go`, `pages.go`, `input_resolver.go`, `types.go`, `open.go`, rmapi's `pdf.go`).
- Checked test fixtures: `fake-cpages-pdf-v6-sample-rm.rmdoc` has 2 UI pages but only `<uuid>/p0.rm` — ideal for skip/blank semantics tests; `fake-cpages-pdf-no-annotations.rmdoc` (4 pages, no `.rm`) and `fake-cpages-pdf-v6-empty-rm.rmdoc` (header-only) cover the other branches.
- Wrote the design doc: `design-doc/01-analysis-design-and-implementation-guide-for-render-v6-annotations-only.md`.

### Why
- The skill workflow requires evidence-first writing; the guide must stand alone for an intern (background sections on .rmdoc, glazed, V6 parsing, both render pipelines) and must justify every non-obvious choice (DR-1..DR-5).

### What worked
- The design fell out cleanly: one new library function `RenderRMDocV6AnnotationsOnlyWithInfo` in a new file `pkg/rmdoc/render/v6_annotations_only.go`, one new CLI flag, default-out suffix `-v6-annotations.pdf`, page emission derived from `--pages` presence (`includeUnannotated = !selection.All`), matching legacy's user-visible semantics.

### What didn't work
- Nothing blocking.

### What I learned
- Legacy's forced `AllPages` under `--pages` exists for index alignment of post-render extraction; v6's pre-selection architecture gets the same user-visible behavior for free by deriving `includeUnannotated` from the selection instead of adding an `--all-pages` flag.

### What was tricky to build
- Deciding page-emission semantics (DR-2): three defensible options (always emit / always skip / skip-unless-selected). Resolved by mirroring rmapi's user-visible behavior, which the user explicitly asked to match ("just like for the render-legacy verb").
- Deciding page geometry (DR-3): reusing the existing overlay-only canvas math avoids new rendering code and keeps visual consistency with current blank-page output; fixed device size or rmapi's 445×594 pt constant were rejected.

### What warrants a second pair of eyes
- DR-2's blank-page emission for explicitly selected unannotated pages (vs. skipping them too).
- DR-4's new default suffix `-v6-annotations.pdf` (vs. keeping `-v6.pdf`).
- Phase 1's helper extraction must be byte-identical on the golden-covered default path.

### What should be done in the future
- Implement Phases 1–4 (this ticket's implementation phase).
- Consider the same flag for `render-v6-png` and the remarquee-ui API as follow-up tickets.

### Code review instructions
- Read the design doc top-to-bottom; then verify each file:line reference with `grep -n`. Cross-check DR-1..DR-5 against the pseudocode in "Proposed Solution".

### Technical details
- Ticket dir: `ttmp/2026/08/03/RMQ-0021--add-annotations-only-flag-to-rmdoc-render-v6/`
- Design doc: `design-doc/01-analysis-design-and-implementation-guide-for-render-v6-annotations-only.md` (~42 KB)
