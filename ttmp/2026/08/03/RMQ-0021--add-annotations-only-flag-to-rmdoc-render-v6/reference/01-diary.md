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
- `ask_questions_about_images` on the rasterized pages failed twice with VLM backend errors (`remote error: tls: bad record MAC`, then an aborted call with partial output). Workaround per user instruction ("Use the read() tool"): read the PNGs directly with the `read` tool, which attaches images to the conversation — this produced a full accurate page-by-page description now embedded in the design doc's worked example.

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

**Commit (code):** db38ea1 — "docs(RMQ-0021): analysis/design/implementation guide for render-v6 --annotations-only" (ticket docs; no source changes)

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

## Step 4: Visual verification of the example pages, ticket bookkeeping, and reMarkable upload

This closing step verified what the example pages actually look like (so the guide's worked example describes real annotation content, not guesses), finished docmgr bookkeeping, validated with doctor, committed, and uploaded the bundle to the reMarkable cloud.

### Prompt Context

**User prompt (verbatim):** "Use the read() tool"

**Assistant interpretation:** After the VLM-based image QA tool failed twice, the user wants the example page PNGs inspected by attaching them via the `read` tool instead.

**Inferred user intent:** Get an accurate visual description of workbook pages 1–3 into the design doc using a reliable path.

**Commit (code):** db38ea1 — "docs(RMQ-0021): analysis/design/implementation guide for render-v6 --annotations-only" (from Step 3); final bookkeeping commit follows this step.

### What I did
- Read `/tmp/ttc-workbook/v6-{1,2,3}.png` with the `read` tool (image attachments) and wrote a precise page-by-page annotation description into the design doc's worked example (check mark in score sheet, tall curly braces, red/pink cursive margin notes with arrow leaders, carets, pale red/pink highlighter bands).
- Updated `index.md` overview, `tasks.md` (5 TODO implementation phases, 5 DONE analysis items).
- Related 7 files to the design doc via `docmgr doc relate` (absolute paths, `path:reason` format).
- `docmgr changelog update --ticket RMQ-0021` with the evidence summary.
- `docmgr doctor --ticket RMQ-0021 --stale-after 30` → ✅ all checks passed.
- Committed ticket docs: db38ea1 (6 files, 901 insertions).
- Upload: `remarquee status` ok; `cloud account --non-interactive` ok (user=wesen@ruinwesen.com); `upload bundle --dry-run` clean; real `upload bundle` → `OK: uploaded RMQ-0021 render-v6 annotations-only design guide.pdf -> /ai/2026/08/03/RMQ-0021`; verified with `cloud ls /ai/2026/08/03/RMQ-0021 --long --non-interactive`.

### Why
- The deliverable contract was: ticket + intern-ready guide + diary + reMarkable upload, with doctor validation before upload and dry-run before real upload.

### What worked
- The `read` tool path gave a complete, reliable visual inventory of all three pages; the worked example now describes concrete strokes the implementer can eyeball against their output.
- Doctor passed on the first run; upload succeeded on the first real attempt after a clean dry-run.

### What didn't work
- `ask_questions_about_images` remained unreliable (TLS error, then aborted call with partial output) — superseded by the `read` tool per user instruction.

### What I learned
- The composite render of page 1 shows highlighter bands *behind* the cursive note — a good visual test for the future implementation: annotations-only output must preserve both the dark pen strokes and the pale shader/highlighter strokes with their alpha.

### What was tricky to build
- n/a (verification/bookkeeping phase)

### What warrants a second pair of eyes
- The worked-example descriptions are agent-interpreted visuals; spot-check against the tablet or the PNGs in `/tmp/ttc-workbook/` if precision matters for goldens later.

### What should be done in the future
- Execute implementation Phases 1–4 from the design doc (tracked in tasks.md).
- After implementation, re-render the workbook with `--annotations-only --pages 1-3` and visually compare against the composite PNGs from this session.

### Code review instructions
- Bundle on the tablet: `/ai/2026/08/03/RMQ-0021/RMQ-0021 render-v6 annotations-only design guide` (design doc + diary with ToC depth 2).
- Local docs: `ttmp/2026/08/03/RMQ-0021--add-annotations-only-flag-to-rmdoc-render-v6/`.

### Technical details
- Remote dir: `/ai/2026/08/03/RMQ-0021`
- Bundle contents: design-doc/01 + reference/01-diary

## Step 5: Scope extension — render-v6-png gets the flag too; implementation begins

The user extended the scope mid-flight: `render-v6-png` should also receive `--annotations-only`, and asked to implement the phases task by task with diary and commits. This step records the scope change and the implementation kickoff; the actual phases follow in subsequent steps.

### Prompt Context

**User prompt (verbatim):** "render-v6-png should get it too. 

let's go task by task and implement a keep a detailed diary and ocmmit at appropriate intervals"

**Assistant interpretation:** Add `--annotations-only` to `render-v6-png` as well, then execute the design doc's phases one at a time, keeping the diary current and committing at each phase boundary.

**Inferred user intent:** Move from design to implementation now, with reviewable per-phase commits and a paper trail.

**Commit (code):** none (scope/doc updates only)

### What I did
- Updated `tasks.md`: added Phase 3b (render-v6-png wiring) between Phase 3 and Phase 4; refined Phase 1 wording to the two actual helpers (`annotationCanvasBBox`, `overlayOnlyPageGeometry`).
- Updated the design doc: scope in/out sections (render-v6-png in scope; its legacy-schema restriction explicitly out), new "Phase 3b" section with flag derivation rationale (`includeUnannotated=true` because render-v6-png always has an explicit pages list and needs 1:1 page mapping), open question 1 marked resolved.

### Why
- Keep the design doc and tasks honest before code changes begin, so later phases review against current docs.

### What worked
- The design anticipated this as "open question 1", so the extension slots into the existing plan without re-design: the png verb rasterizes the library renderer's output, so the flag is a thin pass-through.

### What didn't work
- Nothing.

### What I learned
- `render-v6-png` hard-requires `SchemaCPages` (no `archiveHasV6RM` probe), so the workbook example cannot exercise it; recorded as out of scope with a note that relaxing the schema check is a separate change.

### What was tricky to build
- Deciding `includeUnannotated` for the png verb: it always has an explicit pages list (default "1"), and its rasterize loop maps `PDFPage = i+1` to requested labels 1:1, so skipping unannotated pages would corrupt the mapping. Always true.

### What warrants a second pair of eyes
- The prefix change for png outputs (`-v6-annotations-` vs `-v6-`) — confirm it doesn't break any external script (none found in repo).

### What should be done in the future
- Phases 1–4 per tasks.md.

### Code review instructions
- Diff of this step is docs-only: `tasks.md`, design doc scope/Phase 3b sections.

### Technical details
- n/a

## Step 6: Phase 1 — byte-identical extraction of canvas geometry helpers

First code change of the implementation: the annotation canvas bbox computation and the overlay-only page geometry (padding + margin compensation + page dims) were inline-duplicated in both merge loops. They are now two shared helpers, which Phase 2's annotations-only renderer will also call. Proven byte-identical on real documents, not just unit tests.

### Prompt Context

**User prompt (verbatim):** (see Step 5)

**Assistant interpretation:** Execute Phase 1 from the design doc.

**Inferred user intent:** Refactor first, prove no behavior change, then build the feature on the shared helpers.

**Commit (code):** see commit at end of step — "refactor(RMQ-0021): extract annotation canvas geometry helpers (Phase 1)"

### What I did
- Added `annotationCanvasBBox(strokes) (bbox, stBBox, ok)` and `overlayOnlyPageGeometry(strokes, bbox, stBBox, ok) (bboxWithPad, pageW, pageH, scale)` to `pkg/rmdoc/render/v6_merge_background.go` (placed just before `buildOverlayOnlyPageBBoxScaled`), moving the doc comments about rmc/remarks/CairoSVG semantics onto the helpers.
- Replaced both inline copies: loop 1 (`MergeRMDocV6OntoBackgroundPDFWithInfo`) and loop 2 (`...ForPages`) now call the helpers. The two blocks were textually identical except anchors, so `edit` oldText needed distinguishing context (loop 1 has a leading comment block; loop 2 has none).
- Baseline note: `go test ./...` has a pre-existing failure unrelated to this ticket — `cmd/remarquee-ui/embed.go:8:12: pattern frontend/dist: no matching files found` (frontend not built in this worktree). All relevant packages (`./pkg/rmdoc/... ./cmd/remarquee/...`) were green before the change.
- Byte-identity proof: rendered two documents before (git stash) and after (stash pop) the refactor and compared MD5s:
  - workbook `--pages 1-3` (exercises loop 2): `93395dc8d9d1c7c8a9390b33204ab902` == identical.
  - `cpage-pdf.rmdoc` full doc (exercises loop 1 incl. blank-background branch): `ac3ea926a8dc30300b500406802dd25b` == identical.
- `gofmt -l` clean, `go vet ./pkg/rmdoc/render/` clean, `go test ./pkg/rmdoc/... ./cmd/remarquee/cmds/rmdoc/... -count=1` green.

### Why
- DR-1 in the design doc: the annotations-only renderer reuses the exact geometry of the existing overlay-only path; sharing one implementation eliminates drift between the (now three) per-page loops.

### What worked
- The stash/stash-pop MD5 comparison gives much stronger evidence than "tests pass": output bytes on real renders are unchanged.

### What didn't work
- Nothing.

### What I learned
- The repo's test suite has the pre-existing `frontend/dist` embed failure in `cmd/remarquee-ui`; scoping test runs to `./pkg/rmdoc/... ./cmd/remarquee/...` is the right baseline for this ticket.

### What was tricky to build
- The two inline blocks were character-identical, so the edit tool's uniqueness constraint required anchoring on surrounding context differences (comment presence in loop 1, `pageBoxDims` error string wording in loop 2). Symptom: risk of "oldText matches multiple regions"; solution: include the distinguishing lines in each oldText.

### What warrants a second pair of eyes
- Confirm no semantic difference between the extracted helpers and the old inline code (read `annotationCanvasBBox`/`overlayOnlyPageGeometry` side by side with git show HEAD~1 of the two loops).

### What should be done in the future
- Phase 2 builds `RenderRMDocV6AnnotationsOnlyWithInfo` on these helpers.

### Code review instructions
- `git show` this commit; check `pkg/rmdoc/render/v6_merge_background.go` helpers against the removed inline blocks; re-run the MD5 comparison if desired (commands in "What I did").

### Technical details
- Helpers live in `pkg/rmdoc/render/v6_merge_background.go` directly above `buildOverlayOnlyPageBBoxScaled`.

## Step 7: Phase 2 — annotations-only renderer, rewritten stdlib-only after the unipdf watermark discovery (DR-6)

Phase 2 started exactly per the design doc (renderer reusing the unipdf page builders), passed all six new unit tests, and rendered the workbook correctly — but the smoke PDF had a red "Unlicensed UniDoc" watermark on every page. Root-causing that led to a design change (DR-6): the annotations-only path now writes its PDF with a minimal stdlib-only writer and has no unipdf dependency at all.

### Prompt Context

**User prompt (verbatim):** "can we do it without unidoc?" and then "perfect, what's next?"

**Assistant interpretation:** The user wants the annotations-only feature to avoid unipdf if feasible; after confirming feasibility (parser layer already unidoc-free; we only write a simple PDF), implement it that way.

**Inferred user intent:** No AGPL/watermark entanglement for the new feature; keep the change scoped (not a wholesale unipdf removal from the composite pipeline).

**Commit (code):** see commit at end of step — "feat(RMQ-0021): annotations-only V6 renderer with stdlib PDF writer (Phase 2, DR-6)"

### What I did
- Prototyped `RenderRMDocV6AnnotationsOnlyWithInfo` on unipdf builders first: 6/6 new unit tests passed; workbook smoke rendered 3 pages (skip mode) / 4 pages (subset+blank mode).
- Discovered watermark: `pdftotext annonly-all.pdf - | grep -c Unlicensed` → 3; composite renders → 0. Root cause: unipdf's AGPL community watermark is suppressed only by rmapi's `annotations/license.go` `go:linkname` init, which runs only when rmapi is linked into the binary; my standalone smoke program imported only `pkg/rmdoc/render`.
- Feasibility check for dropping unipdf: `go list -deps ./pkg/rmdoc | grep unidoc` → empty (parser layer is unidoc-free); the only unipdf-coupled pieces in the annotations-only path were page/resource/xref assembly — `buildOverlayOpsBBoxScaled` already emits content streams as plain strings.
- Added DR-6 to the design doc and rewrote Phase 2 stdlib-only:
  - `pkg/rmdoc/render/v6_annotations_only_pdf.go` (new): minimal PDF writer — catalog/pages/page objects, zlib content streams, inline ExtGState + base-14 font dicts, indirect Highlight annot dicts, classic xref/trailer; helpers `pdfNum`, `pdfEscapeString`, `alphaGStateNameStr`, `buildResourcesDict`.
  - `pkg/rmdoc/render/v6_annotations_only.go` (rewritten): ops builders mirroring `buildOverlayOpsBBoxScaled` (stroke state machine w/gs/J/RG, m/l/S paths), `appendTypedTextOpsBBoxScaled` (BT/Tf/Tm/Tj), and `applySmartHighlightsScaled` (quad points + union rect); `blankDevicePageSpec()` for DR-2 blank pages.
- Validation: all 6 unit tests pass against the stdlib writer (tests parse output with the unipdf *reader* as an independent implementation); `qpdf --check` clean; `pdfinfo` correct (3 pages, 606.709×515.249pt); `grep -c Unlicensed` → 0; rasterized pages visually identical to the unipdf prototype, watermark gone.
- Flipped DR-1, DR-3, DR-5, DR-6 to "accepted (implemented 2026-08-03)" in the design doc.

### Why
- Watermark-free output without depending on a `go:linkname` hack in a vendored fork; smaller conceptual surface for a feature that only writes a simple PDF; the parser layer being unidoc-free made it cheap.

### What worked
- The mirror-the-existing-ops approach: stroke state machine, text emission, and highlight quads were ported line-by-line from the unipdf versions, so styling (widths, alphas, colors, font sizes) matches the composite pipeline exactly.
- Independent validation: unipdf reader in tests + qpdf + poppler all accept the hand-rolled output.

### What didn't work
- First stdlib-writer test run failed with `GetAllContentStreams(1): zlib: invalid header`. Cause: `compress/flate` emits raw DEFLATE, but PDF `/FlateDecode` expects zlib (RFC 1950, 2-byte header + adler32). Fix: `compress/zlib`. One retry, then green.

### What I learned
- unipdf's watermark behavior is process-global and license-key-driven; any binary that renders via unipdf without rmapi linked gets watermarked output — worth remembering for future remarquee-ui/API work.
- Diary correction for Step 6: the Phase 1 commit initially failed lefthook twice — (1) pre-existing `cmd/remarquee-ui/embed.go: pattern frontend/dist` failure (fixed locally with a gitignored placeholder `frontend/dist/index.html`), (2) `nonamedreturns` lint on the two new helpers (fixed by un-naming returns, which briefly broke the build via `stBBox, ok =` needing `:=`). Final Phase 1 commit (0be07ed) passed hooks cleanly.

### What was tricky to build
- Getting PDF byte-level details right by hand: xref entries must be exactly 20 bytes (`%010d 00000 n \n`), the binary comment line after `%PDF-1.7`, zlib-vs-raw-deflate, and PDF literal string escaping (`\`, `(`, `)`, octal for non-ASCII). Symptom-driven debugging: unipdf's reader error messages pointed at the stream filter first.
- Keeping HighlightsXTranslation aligned with *emitted* pages (blank pages contribute a 0 entry), which differs from the merge path's per-background-page indexing; documented on the function.

### What warrants a second pair of eyes
- `writeSimplePDF` object numbering/xref math (simple but byte-exact).
- Typed-text emission is untested by fixtures (no fixture has RootText); the port mirrors `appendTypedTextOpsBBoxScaled` but has no golden. Consider a DSL-generated fixture (`rmdsl compile`) with typed text in a follow-up.
- Highlight annotations: fixture has no glyph ranges either; logic mirrors `applySmartHighlightsScaled` but is only exercised on the real workbook (which has none on pages 1-3).

### What should be done in the future
- Phase 3/3b CLI wiring (render-v6, render-v6-png), Phase 4 docs + cloud validation.
- Optional: move annotations-only renderer to its own package if binary-level unidoc separation is ever required (currently same package as the unipdf merge code, so unipdf is still *linked*; it is just never *executed* for this path).

### Code review instructions
- Start at `pkg/rmdoc/render/v6_annotations_only.go` (public function + ops builders), then `v6_annotations_only_pdf.go` (writer). Compare ops against `buildOverlayOpsBBoxScaled`/`applySmartHighlightsScaled` in `v6_merge_background.go`. Run `go test ./pkg/rmdoc/render/ -count=1 -run TestRenderV6AnnotationsOnly -v`.

### Technical details
- Blank page size: 1404×1872 units × rmv6Scale × cairoSVGScale ≈ 335.3×447.0 pt.
- Workbook annotations-only page size: 606.709×515.249 pt (bbox widened by margin compensation — the user's right-margin notes extend past the default screen half-width; same math as the composite path).

## Step 8: Phases 3 + 3b — CLI wiring for render-v6 and render-v6-png

With the library renderer in place, this step exposed `--annotations-only` on both v6 verbs, exactly per the design (plus the user's Phase 3b scope extension), and validated end-to-end through the real CLI including the `--cloud` path on the workbook.

### Prompt Context

**User prompt (verbatim):** (see Step 5)

**Assistant interpretation:** Execute Phases 3 and 3b from the design doc.

**Inferred user intent:** Same as Step 5 — per-phase, reviewable, diary-kept implementation.

**Commit (code):** see commit at end of step — "feat(RMQ-0021): --annotations-only flag for render-v6 and render-v6-png (Phases 3+3b)"

### What I did
- `cmd/remarquee/cmds/rmdoc/render_v6.go`: added `AnnotationsOnly` to `RenderV6Settings` and `renderV6Execution`, flag definition (help text mirrors legacy + skip semantics), default output suffix `-v6-annotations.pdf` (DR-4), a `switch` in `execute` routing to `RenderRMDocV6AnnotationsOnlyWithInfo(..., selection.Indices0, !selection.All)` (DR-2 derivation), `annotations_only` in the Glaze row, and long-help bullets.
- `cmd/remarquee/cmds/rmdoc/render_v6_png.go`: same flag on the PNG verb; branch calls the annotations-only renderer with `includeUnannotated=true` (explicit pages list + 1:1 rasterize mapping); output prefix becomes `<base>-v6-annotations` for both the intermediate PDF and PNGs.
- CLI tests (`render_v6_test.go` +3, new `render_v6_png_test.go` +1): skip semantics (1 page from the 2-page/1-annotated fixture), subset blank emission (`--pages 1,2` → 2 pages), default output name via `t.Chdir`, PNG smoke with pdftoppm producing `-v6-annotations-page-001/002.png`.
- End-to-end validation through the built CLI:
  - `render-v6 --cloud --non-interactive "/ai/.../TTC Garden Human Calibration Workbook" --annotations-only --pages 1-3` → 3 pages.
  - `render-v6 workbook.rmdoc --annotations-only` (all 44 pages) → 3 pages (skip semantics).
  - `pdftotext | grep -c Unlicensed` → 0 (DR-6 holds through the CLI too).
  - `--with-glaze-output --output json` → row contains `"annotations_only": true`.
  - `render-v6-png cpage-pdf.rmdoc --pages 1 --annotations-only` → PNG shows handwriting only, white canvas.
- Flipped DR-2 and DR-4 to accepted in the design doc (all six DRs now accepted).

### Why
- Per design doc phases; the png verb needs `includeUnannotated=true` always because its rasterize loop maps requested pages 1:1 to PDF pages.

### What worked
- The glazed plumbing was exactly the three touch points predicted in the design doc (settings struct, flags slice, execute branch).
- Two `edit` anchors missed on the first pass (long-help strings end with `` ` `` `),` not `` ` `` `,`); re-issued with corrected text — second attempt clean.

### What didn't work
- PNG test first failed: `pdftoppm not found on PATH: exec: ""`. Cause: `newDefaultParsedValues` does not apply flag defaults — unset `pdftoppm`/`dpi` decode to zero values. Fix: pass `"pdftoppm": "pdftoppm", "dpi": 100` explicitly in the test map.

### What I learned
- The repo's glazed test helper bypasses the cobra/parser layer where defaults are normally filled in; any CLI test must set every flag the code path reads (or the command's `WithDefault` is silently ignored).

### What was tricky to build
- Nothing beyond the anchor/default-value issues above; the branch logic was a direct application of DR-2's derivation (`includeUnannotated = !selection.All`).

### What warrants a second pair of eyes
- `--pages` + `--annotations-only` on a selection whose pages are ALL unannotated now yields an all-blank PDF (legacy parity) — confirm this is the desired UX vs. an error/empty PDF.
- The PNG prefix change (`-v6-annotations-`) — no in-repo consumers found.

### What should be done in the future
- Phase 4: README examples, final full-suite run, ticket bookkeeping, closing diary step.

### Code review instructions
- `git show` this commit; focus on `render_v6.go` execute switch and `render_v6_png.go` Run branch. Run `go test ./cmd/remarquee/cmds/rmdoc/ -count=1 -run AnnotationsOnly -v` and the end-to-end commands listed above.

### Technical details
- Glaze row gained `annotations_only` (bool).
- Default outputs: `<input>-v6-annotations.pdf` (render-v6), `<base>-v6-annotations.pdf` + `-page-XXX.png` (render-v6-png).

## Step 9: Phase 4 — README, full-suite validation, ticket close-out

Final phase: user-facing docs, whole-repo regression run, and ticket bookkeeping. The feature is complete on this branch: both v6 verbs support `--annotations-only`, validated end-to-end on the real workbook through the `--cloud` path.

### Prompt Context

**User prompt (verbatim):** (see Step 5)

**Assistant interpretation:** Execute Phase 4 and close out the implementation.

**Inferred user intent:** Same as Step 5.

**Commit (code):** see commit at end of step — "docs(RMQ-0021): README examples + final validation (Phase 4)"

### What I did
- README: added `--annotations-only` examples for `render-v6` (plain + `--pages`) and `render-v6-png` in the "Render local .rmdoc archives" section.
- Full suite: `go build ./...` ok; `go test ./... -count=1` → zero failures (including `cmd/remarquee-ui`, which compiles thanks to the gitignored placeholder dist from Step 6).
- Bookkeeping: tasks 5–6 checked; changelog updated with phase results and commit hashes; `docmgr doctor --ticket RMQ-0021` re-run.

### Why
- Close out per the deliverable checklist: docs updated, tests green, ticket state consistent.

### What worked
- No regressions anywhere in the repo test suite after all three code phases.

### What didn't work
- Nothing.

### What I learned
- n/a

### What was tricky to build
- n/a

### What warrants a second pair of eyes
- All open review points from Steps 6–8 (xref math, typed-text/highlight paths lacking fixtures, all-blank-output UX, PNG prefix change).

### What should be done in the future
- Push branch + PR; follow-ups recorded in the design doc (typed-text/highlight fixtures via rmdsl; render-v6-png legacy-schema probe; possible package split for binary-level unidoc separation).
- Optional: re-upload the updated ticket docs to reMarkable (design doc gained DR-6, Phase 3b, accepted statuses; diary gained steps 5–9).

### Code review instructions
- Commits in order: ee1be8e (scope docs), 0be07ed (Phase 1 refactor), 796fd6d (Phase 2 renderer + stdlib writer), 1594eae (Phases 3+3b CLI), this commit (Phase 4 docs). Review design doc DR-1..DR-6 against the code.

### Technical details
- n/a

## Step 10: PR #21 code review — three P2 fixes (text encoding, highlight Y translation, PNG page ranges)

The PR received three P2 review comments from the Codex connector bot. All three were legitimate; all three are fixed in this step, plus one related latent bug in the merge pipeline was identified and documented as a follow-up rather than fixed here (it would alter golden-covered output).

### Prompt Context

**User prompt (verbatim):** "Address code review issues : https://github.com/go-go-golems/remarquee/pull/21"

**Assistant interpretation:** Fetch the PR review comments, fix the issues, validate, commit, push, and respond on the PR.

**Inferred user intent:** Keep the PR mergeable and review-clean.

**Commit (code):** see commit at end of step — "fix(RMQ-0021): address PR #21 review (WinAnsi text encoding, highlight Y translation, PNG page ranges)"

### What I did
- Fetched review comments via `gh api repos/go-go-golems/remarquee/pulls/21/comments`. Three P2s from chatgpt-codex-connector[bot]:
  1. `pdfEscapeString` emitted raw UTF-8 bytes (é → `\303\251`) against base-14 fonts without a Unicode encoding → mojibake.
  2. `buildHighlightAnnots` omitted the canvas `bbox.MinY` Y translation that strokes/typed text apply → highlights shifted vertically.
  3. README example `render-v6-png --pages 2-4` fails: the PNG verb's `parsePages1Based` (vlm_validate.go:338) is comma-only.
- Fix 1: new `encodeWinAnsiText` (rune-by-rune Windows-1252 via `golang.org/x/text/encoding/charmap`, unrepresentable → `?`), font dicts gain `/Encoding /WinAnsiEncoding`, Tj strings go through it. Unit test covers é (0xE9), € (0x80), smart quotes, CJK → `?`.
- Fix 2: `buildAnnotationsOnlyPageSpec`/`buildHighlightAnnots` now take `yTranslation = -yyScaled(bboxWithPad.MinY, scale)`; quads and rect use `pageH - y*scale - yTranslation`. Unit test pins exact quad/rect values with yTranslation=12.
- Fix 3: `render_v6_png.go` now opens the doc first and uses the shared range-aware `parsePageSelection1Based` (same parser as render-v6/render-legacy); labels/indices come from the selection. CLI test with `--pages 1-2` produces both PNGs.
- `go mod tidy` (x/text indirect → direct). Full suite: 12 packages ok, zero failures. Workbook re-validation: 3 pages, qpdf clean.
- Documented latent merge-path issue: `applySmartHighlightsScaled` has the same missing-Y-translation in the blank-background branches of both merge loops; not fixed here because it changes golden-covered composite output — recorded as open question 4 in the design doc.

### Why
- Review-driven quality gate before merge.

### What worked
- x/text was already in go.mod (indirect), so the WinAnsi fix needed no new dependency.
- The reviewer's suggestion "or change the PNG command to use the range-aware page-selection parser" was strictly better than editing the README — it fixes the verb for everyone and makes the example true.

### What didn't work
- Test compile failed once: `undefined: rmdoc` in the test file (new synthetic glyph-range test needed the import). One-line fix.

### What I learned
- The merge pipeline's blank-background branch carries the same highlight/Y-translation misalignment (bbox.Expand(pad) always makes MinY ≤ -1 while highlight math assumed MinY=0). Small (~pad*scale ≈ 3-4pt) but real; needs golden-aware fixing in its own ticket.

### What was tricky to build
- Verifying the sign of the Y translation: stroke y_pdf = pageH − (y − bbox.MinY)·scale = pageH − y·scale − yTranslation with yTranslation = −bbox.MinY·scale (bbox.MinY is negative after padding, so yTranslation is positive and highlights move down to meet the strokes). Pinned by the unit test.

### What warrants a second pair of eyes
- CP1252 replacement-char degradation for non-Latin scripts (documented as deliberate; full Unicode needs a CID font).
- The reorder in render-v6-png Run (doc opened before page parsing) — behavior only differs in error precedence for invalid --pages vs invalid file.

### What should be done in the future
- Ticket the merge-path highlight misalignment (open question 4).
- Merge PR #21 after CI.

### Code review instructions
- Diff this commit: `pkg/rmdoc/render/v6_annotations_only.go` (yTranslation + encodeWinAnsiText), `v6_annotations_only_pdf.go` (WinAnsiEncoding font dicts), `render_v6_png.go` (parser swap), plus 3 new tests. Run `go test ./pkg/rmdoc/render/ ./cmd/remarquee/cmds/rmdoc/ -count=1`.

### Technical details
- Review comments: discussions r3708390590 (encoding), r3708390595 (Y translation), r3708390599 (PNG ranges).

### Addendum (same step): branch rebase mismatch

The first attempt to push the review fixes failed non-fast-forward: the local worktree branch had silently been rebased onto `28edc01` (PR #20, glazed structured-output migration) with all ticket commits duplicated under new hashes, while the GitHub PR head (`wesen/task/remarquee-v6-only-annotations`) still pointed at the pre-rebase line ending in `cf79c72` (confirmed via review-comment commit_id). Resolution: `git reset --hard` to the PR head, then re-applied the review fixes on that line — library/test/docs files checked out verbatim from the fix commit (PR #20 never touched them), the `render_v6_png.go` parser swap re-applied by hand (PR #20 did touch that file), and the `go.mod` x/text direct-dependency edit adapted to the old base's version (v0.37.0, not v0.39.0). Lesson: after any external rebase of the working branch, verify `git status`/base before committing; when pushing fails non-fast-forward, diff both lines before choosing reset-vs-rebase.

## Step 11: Resolve merge conflict with main (glazed migration, PR #20)

The PR became CONFLICTING after main absorbed PR #20 (glazed structured-output migration). Resolved by merging origin/main into the feature branch rather than rebasing — the previous step's rebase confusion argued for the explicit, history-preserving option.

### Prompt Context

**User prompt (verbatim):** "resolve git merge conflict ."

**Assistant interpretation:** Make PR #21 mergeable again by resolving its conflict with main.

**Inferred user intent:** Unblock the merge.

**Commit (code):** merge commit at end of step — "merge: resolve go.mod conflict with origin/main (glazed migration)"

### What I did
- `git merge origin/main` (28edc01): `render_v6.go` and `render_v6_png.go` auto-merged; only `go.mod` conflicted (x/* dependency block: my x/text-made-direct vs main's version bumps).
- Resolution: kept main's newer versions (sync v0.21.0, sys v0.46.0, term v0.44.0, text v0.39.0) and preserved x/text as a direct dependency (my only go.mod change).
- `go mod tidy`; `go build ./...` OK; `go test ./... -count=1` → 12 packages ok, zero failures.
- Post-migration CLI sanity: `--with-glaze-output --output json` no longer exists; migrated flag is `--format json` — verified the glaze row still emits `annotations_only`/`selected_pages`; `--annotations-only --pages 1-3` on the workbook still yields 3 pages.

### Why
- Restore PR mergeability with main's glazed v1.4.2 API included.

### What worked
- Git auto-merged both Go files because PR #20 and RMQ-0021 touched disjoint regions (command constructors vs settings/execute).

### What didn't work
- `--output json` (pre-migration flag) exits 1 — expected behavior change from PR #20, not a regression of this branch; use `--format json`.

### What I learned
- The glazed v1.4.2 migration renamed structured-output flags (`--output` → `--format`, plus `--output-fields`); any docs/scripts using the old flag names break. The RMQ-0021 design guide doesn't reference the old flags, so no doc changes needed.

### What was tricky to build
- n/a (single-file textual conflict)

### What warrants a second pair of eyes
- Confirm no other repo docs/scripts still reference `--output json` for glaze output (out of scope here; noted).

### What should be done in the future
- Merge PR #21.

### Code review instructions
- `git show` the merge commit; only `go.mod` was manual. Re-run the sanity commands above if desired.

### Technical details
- n/a
