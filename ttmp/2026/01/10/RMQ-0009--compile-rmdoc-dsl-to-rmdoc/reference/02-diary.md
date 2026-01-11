---
Title: Diary
Ticket: RMQ-0009
Status: active
Topics:
    - remarkable
    - rmdoc
    - rendering
    - dsl
    - compiler
    - go
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarquee/cmd/remarquee/cmds/rmdsl/compile.go
      Note: CLI command for rmdsl compile
    - Path: remarquee/cmd/remarquee/cmds/rmdsl/root.go
      Note: rmdsl command group wiring
    - Path: remarquee/cmd/remarquee/main.go
      Note: Registers rmdsl command
    - Path: remarquee/pkg/doc/topics/rmdsl-getting-started.md
      Note: Documented compile usage
    - Path: remarquee/pkg/rmdsl/compile/compile.go
      Note: Compiler entrypoint and archive assembly
    - Path: remarquee/pkg/rmdsl/compile/compile_test.go
      Note: Integration and round-trip tests
    - Path: remarquee/pkg/rmdsl/compile/content.go
      Note: Minimal cPages .content JSON builder
    - Path: remarquee/pkg/rmdsl/compile/lower.go
      Note: Shape/stroke lowering logic
    - Path: remarquee/pkg/rmdsl/compile/metadata.go
      Note: Minimal .metadata JSON builder
    - Path: remarquee/pkg/rmdsl/compile/rmv6_encode.go
      Note: Line payload encoding and CRDT ID generation
    - Path: remarquee/pkg/rmdsl/compile/rmv6_page.go
      Note: RMV6 scene tree + line item block writer
    - Path: remarquee/pkg/rmdsl/compile/rmv6_writer.go
      Note: Tagged-block writer implementation
    - Path: remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/20-ellipse-sweep-generate-upload-review.sh
      Note: Primary script proving the PDF upload path for the red-dash fixture
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/analysis/01-rmq-0006-red-dash-pdf-vs-rmdoc-findings.md
      Note: Step 1 analysis artifact for RMQ-0006 red-dash workflow
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/analysis/02-incremental-rmdoc-compiler-test-plan.md
      Note: Step 4 test ladder doc
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/analysis/03-rmv6-block-semantics-in-rmscene-go-parser.md
      Note: Step 7 analysis doc
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/analysis/04-remarks-rmscene-block-handling-and-semantics.md
      Note: Step 9 analysis doc (remarks + rmscene block handling)
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/cases/01-empty-1page.yaml
      Note: Core validation case
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/cases/02-single-line.yaml
      Note: Core validation case
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/cases/03-rect.yaml
      Note: Core validation case
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/cases/04-ellipse.yaml
      Note: Core validation case
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/cases/05-two-pages-empty.yaml
      Note: Core validation case
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/cases/06-two-pages-mixed.yaml
      Note: Core validation case
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/cases/07-two-layers.yaml
      Note: Core validation case
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/cases/08-tools-colors.yaml
      Note: Core validation case
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/cases/09-complex-spiral-grid.yaml
      Note: Step 10 complicated validation case
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/cases/10-layered-highlighter.yaml
      Note: Step 10 complicated validation case
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/cases/11-two-pages-dense.yaml
      Note: Step 10 complicated validation case
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/cases/12-tools-showcase-advanced.yaml
      Note: Step 10 complicated validation case
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/scripts/01-compile-ellipse-sweep-rmdoc.sh
      Note: Fixture compile helper
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/scripts/02-compile-upload-review.sh
      Note: Compile/upload/editable notebook prompt script
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/scripts/03-batch-compile-upload-tests.sh
      Note: Batch compile/upload script
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/scripts/04-dump-rm-blocks/main.go
      Note: Script used to compare RMV6 block sets
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/scripts/05-batch-compile-render-upload-complicated.sh
      Note: Step 10 batch compile/render/upload script
ExternalSources: []
Summary: Implementation diary tracking RMQ-0009 research, compilation work, and device validation steps.
LastUpdated: 2026-01-10T21:56:36-05:00
WhatFor: Provide step-by-step implementation notes, decisions, and validation commands for RMQ-0009.
WhenToUse: Use to review work history, reproduce validation steps, or continue implementation.
---








# Diary

## Goal

Capture RMQ-0009 progress and research outcomes, with an emphasis on the DSL-to-.rmdoc compiler scope and its relationship to RMQ-0006 artifacts.

## Step 1: Trace the RMQ-0006 red-dash PDF workflow

I reviewed RMQ-0006 docs and scripts to answer whether the red-dash ellipse sweep was a real `.rmdoc` generated from JS or a PDF transport. The result is a new RMQ-0009 analysis doc that records the evidence and implications for the compiler work.

The evidence consistently points to a JS-to-DSL-to-PDF path that was uploaded to the device, with `.rmdoc` compilation explicitly listed as a future task. This clarifies RMQ-0009's missing capability and why we cannot treat the RMQ-0006 red-dash fixture as an editable notebook artifact.

### What I did
- Searched RMQ-0006 docs for "red dashes" and traced the ellipse sweep pipeline.
- Read the JS case generator and the PDF renderer source.
- Confirmed the RMQ-0006 task list still marked DSL -> `.rmdoc` compilation as pending.
- Added RMQ-0009 analysis doc: `remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/analysis/01-rmq-0006-red-dash-pdf-vs-rmdoc-findings.md`.

### Why
- Answer the user question with primary-source evidence and lock in RMQ-0009 scope.

### What worked
- RMQ-0006 diary and scripts explicitly document the PDF transport and device upload path.

### What didn't work
- N/A (no failures encountered).

### What I learned
- The red-dash ellipse sweep in RMQ-0006 was a PDF transport generated from RMDoc-DSL JS, not a `.rmdoc` compiler output.

### What was tricky to build
- Disentangling similar-looking workflows (PDF transport vs `.rmdoc` pipelines) across multiple RMQ-0006 artifacts.

### What warrants a second pair of eyes
- Confirm there were no side experiments elsewhere in the repo that produced `.rmdoc` from DSL during RMQ-0006 (the docs and tasks say no, but a quick repo-wide scan could validate).

### What should be done in the future
- As RMQ-0009 progresses, document clearly which outputs are PDF-only vs `.rmdoc` to avoid mixing up validation artifacts.

### Code review instructions
- Start in `remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/analysis/01-rmq-0006-red-dash-pdf-vs-rmdoc-findings.md`.
- Validate by opening the referenced RMQ-0006 scripts and confirming they generate/upload PDFs, not `.rmdoc`.

### Technical details
- Commands used:
  - `rg -n "red dash|dashes" remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner`
  - `sed -n '1,200p' remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/20-ellipse-sweep-generate-upload-review.sh`
  - `sed -n '1,220p' remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/19-rmdsl-render-to-pdf/main.go`
  - `sed -n '1,200p' remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/cases/03-ellipse-sweep.js`

## Step 2: Build a strokes-only DSL → .rmdoc compiler + CLI

This step implemented the minimal end-to-end compiler: RMDoc-DSL (YAML/JS) now produces a V6 `.rmdoc` zip with `.content`, `.metadata`, and per-page `.rm` files. The compiler lowers shapes into polylines, generates CRDT IDs deterministically, and emits a basic scene tree so `OpenFile` + `ParseRMV6SceneTree` can read it.

I also added a CLI command (`remarquee rmdsl compile`), integration tests, and ticket scripts to compile and upload the ellipse sweep as a real notebook. This establishes the pipeline RMQ-0009 needs before we tackle typed text, glyphs, or templates.

**Commit (code):** dbacadb — "✨ RMQ-0009: add RMDoc-DSL compiler and CLI"

### What I did
- Added a new `pkg/rmdsl/compile` package with a tagged-block writer, V6 line encoder, shape lowering, and zip assembly.
- Implemented a strokes-only scene writer: SceneTreeBlock + TreeNodeBlock + SceneGroupItemBlock + SceneLineItemBlock.
- Generated `.content` (cPages) and `.metadata` JSON for a minimal notebook archive.
- Added `remarquee rmdsl compile` Cobra command and wired it into the CLI.
- Added integration + round-trip tests for compile → open → parse → render.
- Added RMQ-0009 scripts to compile and upload the ellipse sweep.
- Documented the compiler usage in `pkg/doc/topics/rmdsl-getting-started.md`.

### Why
- RMQ-0009 needs a real `.rmdoc` generator so device truth can be based on notebook bytes, not PDF transports.

### What worked
- `go test ./pkg/rmdsl/compile -count=1` passes, including parse and render-v6 checks.
- The compiler produces a valid `.rmdoc` zip that `rmdoc.OpenReaderAt` can open.

### What didn't work
- N/A (no failures encountered).

### What I learned
- Minimal V6 `.rm` files can be accepted by our parser as long as SceneTree and line items are coherent; the rest can be layered later.

### What was tricky to build
- Keeping tagged-block write order aligned with the decoder (tags must be emitted in the expected index order).
- Choosing deterministic CRDT IDs and page UUID policies without breaking parser assumptions.
- Matching highlight color encoding conventions (PenColor.HIGHLIGHT + RGBA marker).

### What warrants a second pair of eyes
- Validate the `.content`/`.metadata` payloads against device acceptance; minimal fields may not be sufficient for cloud upload.
- Review the CRDT id sequencing and cPages `idx` generation for compatibility with real notebooks.

### What should be done in the future
- Confirm device acceptance via `cloud put` and update `.content`/`.metadata` if the tablet rejects or mis-displays the notebook.
- Extend the compiler for glyphs, typed text, and templates once the strokes-only path is proven.

### Code review instructions
- Start in `remarquee/pkg/rmdsl/compile/compile.go` (compiler entrypoint and zip assembly).
- Review `remarquee/pkg/rmdsl/compile/rmv6_page.go` and `remarquee/pkg/rmdsl/compile/rmv6_encode.go` (V6 blocks + line payload).
- Verify tests in `remarquee/pkg/rmdsl/compile/compile_test.go`.
- Try the CLI: `go run ./cmd/remarquee rmdsl compile <case.js> --out /tmp/out.rmdoc`.

### Technical details
- Test command: `go test ./pkg/rmdsl/compile -count=1`
- New scripts:
  - `remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/scripts/01-compile-ellipse-sweep-rmdoc.sh`
  - `remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/scripts/02-compile-upload-review.sh`

## Step 3: Upload compiled `.rmdoc` and request device validation

This step attempted the full protocol: compile the ellipse sweep case, upload it as a notebook, and prompt for device confirmation. The upload succeeded after creating the missing remote directory, but the `plz-confirm` prompt auto-timed out with no response, so editability is still unverified.

The result is that the pipeline is technically working up to upload, yet we still need a human confirmation that the notebook opens as an editable document on-device and that the red dash markers render correctly.

### What I did
- Ran the protocol script and saw the upload fail due to a missing remote directory.
- Created `/remarquee/rendering/rmq-0009-ellipse` via `remarquee cloud mkdir`.
- Uploaded `ellipse-sweep.rmdoc` with `remarquee cloud put`.
- Issued a `plz-confirm` prompt for device verification.

### Why
- Confirm the compiler output is accepted by the cloud and appears on-device as an editable notebook.

### What worked
- Upload succeeded after creating the remote directory.

### What didn't work
- Initial upload failed with `Error: directory doesn't exist`.
- `plz-confirm` timed out with `AUTO_TIMEOUT` (no user response).

### What I learned
- The cloud path must exist before upload; `cloud put` does not create intermediate directories.

### What was tricky to build
- N/A (workflow step; no code changes).

### What warrants a second pair of eyes
- Device-side confirmation that the uploaded `.rmdoc` is editable and renders the red dash markers.

### What should be done in the future
- Re-run the device confirmation prompt when someone is available with the tablet.

### Code review instructions
- N/A (no code changes in this step).

### Technical details
- Commands:
  - `bash remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/scripts/02-compile-upload-review.sh`
  - `go run ./cmd/remarquee cloud ls /remarquee --non-interactive`
  - `go run ./cmd/remarquee cloud mkdir --non-interactive /remarquee/rendering/rmq-0009-ellipse`
  - `go run ./cmd/remarquee cloud put --non-interactive --force ./rendering/rmq-0009-ellipse/ellipse-sweep.rmdoc /remarquee/rendering/rmq-0009-ellipse`
  - `plz-confirm confirm --title "RMQ-0009: .rmdoc uploaded" --message "Open 'ellipse-sweep.rmdoc' on the tablet in /remarquee/rendering/rmq-0009-ellipse. Confirm: is it an editable notebook (can you add strokes) and do the red dash page markers appear?" --approve-text "Yes, editable" --reject-text "Not now" --wait-timeout 300 --output json`
- Errors:
  - `Error: directory doesn't exist`
  - `AUTO_TIMEOUT` from `plz-confirm`

## Step 4: Draft incremental compiler test ladder

I wrote a detailed, incremental testing plan so we can debug the "unable to render document" device error by starting from minimal notebooks and adding features one at a time. The plan includes a test ladder, feature coverage map, and recommended DSL case files.

This gives us a shared testing checklist and avoids random trial-and-error; each rung is meant to isolate a specific failure mode.

### What I did
- Authored the test plan document: `remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/analysis/02-incremental-rmdoc-compiler-test-plan.md`.
- Linked it from the RMQ-0009 index.

### Why
- Provide a deterministic, incremental approach to validate the compiler and isolate render failures.

### What worked
- The document captures a full ladder from empty notebooks through device upload.

### What didn't work
- N/A (documentation only).

### What I learned
- The fastest path to isolate the device render error is to start with an empty notebook and add a single feature per test case.

### What was tricky to build
- Keeping the ladder granular enough to isolate failures without creating redundant test steps.

### What warrants a second pair of eyes
- Review the ladder ordering and ensure no missing "first" feature that could still break render (e.g., empty layers vs no layers).

### What should be done in the future
- Convert the plan into actual DSL fixtures and CI tests as we start validating rungs.

### Code review instructions
- Start in `remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/analysis/02-incremental-rmdoc-compiler-test-plan.md`.
- Validate that the ladder aligns with the current compiler scope (strokes-only).

### Technical details
- N/A (documentation only).

## Step 5: Create and upload core 8 validation notebooks

I created eight RMDoc-DSL YAML cases that map to the core validation ladder (empty, single line, rect, ellipse, multi-page, multi-layer, and tool/color mapping). Each case was compiled to a separate `.rmdoc` and uploaded to a shared remote folder so device review can cover multiple tests in one pass.

The batch upload partly timed out in the CLI harness, so I re-ran the remaining uploads individually. The final remote directory listing confirms all eight notebooks are present and ready for device validation.

**Commit (code):** 63b764f — "📝 RMQ-0009: add test fixtures and docs"

### What I did
- Added 8 DSL cases under `ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/cases/`.
- Added a batch compile+upload script.
- Compiled and uploaded all 8 notebooks to `/remarquee/rendering/rmq-0009-tests`.
- Verified the remote folder contains all 8 documents.

### Why
- Enable user review of multiple incremental tests in one device session and isolate the first failing rung.

### What worked
- Each case compiled and uploaded successfully when run individually.
- `cloud ls` confirms all 8 documents are present.

### What didn't work
- The batch script timed out in the CLI harness; I had to re-run the remaining uploads.
- `cloud mkdir` reports “entry already exists” (expected when re-running).

### What I learned
- Long-running batch uploads need explicit timeouts in the CLI harness.

### What was tricky to build
- N/A (mostly data files + batch script).

### What warrants a second pair of eyes
- Confirm the device error (“unable to render document”) appears on the earliest failing case, so we can isolate the specific feature that breaks rendering.

### What should be done in the future
- Add a follow-up script that requests device feedback on each notebook (pass/fail + notes) once the user reviews them.

### Code review instructions
- Start in `remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/cases/`.
- Review the batch script: `remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/scripts/03-batch-compile-upload-tests.sh`.

### Technical details
- Commands:
  - `bash remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/scripts/03-batch-compile-upload-tests.sh`
  - `go run ./cmd/remarquee cloud ls --non-interactive /remarquee/rendering/rmq-0009-tests`
  - `go run ./cmd/remarquee rmdsl compile <case.yaml> --out ./rendering/rmq-0009-tests/<case>.rmdoc`
  - `go run ./cmd/remarquee cloud put --non-interactive --force ./rendering/rmq-0009-tests/<case>.rmdoc /remarquee/rendering/rmq-0009-tests`
- Remote path:
  - `/remarquee/rendering/rmq-0009-tests`

## Step 6: Compare device-authored empty notebook vs compiled empty

I downloaded a device-authored empty notebook (`device-empty-1page`) to compare its `.content` and `.rm` structure against our compiled empty case. The device `.rm` includes several blocks that our compiler currently omits: AuthorIdsBlock (0x09), MigrationInfoBlock (0x00), PageInfoBlock (0x0A), and SceneInfo (0x0D). The device file also includes a TreeNode block for the root group and uses a specific CRDT id pattern (tree_id = CrdtId(0,11), root group = CrdtId(0,1)).

This likely explains the tablet’s “unable to render document” error: the missing blocks and root group metadata are probably required for notebook acceptance.

**Commit (code):** 63b764f — "📝 RMQ-0009: add test fixtures and docs"

### What I did
- Downloaded the device-authored notebook from the cloud.
- Inspected `.content` and `.metadata` JSON for both device and compiled cases.
- Wrote a small script to list RMV6 block types in a `.rm` file.
- Compared block sets between the two notebooks.

### Why
- Identify the minimum missing RMV6 blocks needed for device render acceptance.

### What worked
- `cloud get` successfully downloaded the device notebook.
- The block dump script confirmed missing blocks in the compiled file.

### What didn't work
- N/A (analysis-only).

### What I learned
- Device empty `.rm` includes:
  - AuthorIdsBlock (0x09)
  - MigrationInfoBlock (0x00)
  - PageInfoBlock (0x0A)
  - SceneInfo (0x0D)
  - SceneTreeBlock with tree_id = CrdtId(0,11), node_id = CrdtId(0,0), is_update = true
  - TreeNodeBlock for root group (CrdtId(0,1)) and for layer group (CrdtId(0,11))
  - SceneGroupItemBlock linking root -> layer
- Our compiled empty `.rm` has only SceneTreeBlock, TreeNodeBlock (layer), and SceneGroupItemBlock.

### What was tricky to build
- Extracting block-level details required a custom script because the Go reader only exposes parsed scene trees.

### What warrants a second pair of eyes
- Verify which of the missing blocks are strictly required by the device (likely MigrationInfo + PageInfo + SceneInfo + AuthorIds).
- Confirm the CRDT id layout used by device notebooks (e.g., root group id, layer group ids).

### What should be done in the future
- Update the compiler to emit AuthorIdsBlock, MigrationInfoBlock, PageInfoBlock, SceneInfo, and a TreeNodeBlock for the root group.
- Re-run the empty notebook upload after those blocks are added.

### Code review instructions
- Start with the block dump script: `remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/scripts/04-dump-rm-blocks/main.go`.
- Review the block type output for both compiled and device notebooks.

### Technical details
- Commands:
  - `go run ./cmd/remarquee cloud ls --non-interactive /remarquee/rendering/rmq-0009-tests`
  - `go run ./cmd/remarquee cloud get --non-interactive /remarquee/rendering/rmq-0009-tests/device-empty-1page --out-dir ./rendering/rmq-0009-tests`
  - `go run ./ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/scripts/04-dump-rm-blocks/main.go --rmdoc ./rendering/rmq-0009-tests/01-empty-1page.rmdoc`
  - `go run ./ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/scripts/04-dump-rm-blocks/main.go --rmdoc ./rendering/rmq-0009-tests/device-empty-1page.rmdoc`

## Step 7: Document RMV6 block semantics (rmscene + Go parser)

I wrote a detailed analysis document explaining what each RMV6 block does, where it is implemented in `rmscene`, and how the Go parser treats (or ignores) those blocks. The doc also captures the observed block list for a device-authored empty notebook and contrasts it with our compiled empty output, highlighting the missing blocks that likely cause device render failures.

This is meant to be the reference for implementing the missing blocks in the compiler without guessing.

### What I did
- Authored the analysis doc: `remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/analysis/03-rmv6-block-semantics-in-rmscene-go-parser.md`.
- Linked it from the RMQ-0009 index.
- Related key rmscene + Go parser files to the doc.

### Why
- We need a grounded understanding of RMV6 block semantics to fix “unable to render document.”

### What worked
- The analysis highlights exact missing blocks and their expected fields.

### What didn't work
- N/A (documentation only).

### What I learned
- Device acceptance likely depends on blocks that the Go parser currently ignores, so local parsing success is not enough.

### What was tricky to build
- Keeping the doc focused on concrete blocks/fields rather than conjecture.

### What warrants a second pair of eyes
- Validate the inferred required blocks against another device-authored notebook (not just empty).

### What should be done in the future
- Implement the missing RMV6 blocks and re-run the empty notebook upload.

### Code review instructions
- Start in `remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/analysis/03-rmv6-block-semantics-in-rmscene-go-parser.md`.

### Technical details
- N/A (documentation only).

## Step 8: Emit device-required RMV6 blocks and re-upload core tests

I updated the RMV6 writer to include the blocks found in device-authored notebooks (AuthorIds, MigrationInfo, PageInfo, SceneInfo) and added a root TreeNode plus device-aligned CRDT id patterns. The empty compiled notebook now matches the device block set, and I recompiled/reuploaded the core 8 validation notebooks to the shared device folder.

This should unblock the “unable to render document” issue and lets us retest from the simplest notebook upward.

**Commit (code):** dbacadb — "✨ RMQ-0009: add RMDoc-DSL compiler and CLI"

### What I did
- Added block writers for AuthorIdsBlock (0x09), MigrationInfoBlock (0x00), PageInfoBlock (0x0A), SceneInfo (0x0D).
- Added a root TreeNodeBlock and device-like CRDT id layout (root = 0,1; layer id starts at 0,11; group item id at 0,13).
- Added author UUID handling and wrote author UUIDs in little-endian format for AuthorIdsBlock.
- Recompiled and reuploaded the core 8 `.rmdoc` validation notebooks.

### Why
- Device notebooks require additional RMV6 blocks not needed by our Go parser; without them the tablet refuses to render.

### What worked
- The compiled empty `.rm` now includes the same block types as the device empty notebook.
- All 8 core notebooks reuploaded successfully.

### What didn't work
- N/A (no new failures observed).

### What I learned
- Matching block presence and CRDT id patterns is critical for device acceptance even when local parsers succeed.

### What was tricky to build
- Correctly encoding author UUIDs in little-endian byte order for AuthorIdsBlock.
- Aligning structural CRDT ids without breaking line item sequences.

### What warrants a second pair of eyes
- Validate the SceneInfo defaults (paper size, visibility flags) against other device-authored notebooks.

### What should be done in the future
- Re-run device validation on `01-empty-1page` and report the first failing case (if any).

### Code review instructions
- Start in `remarquee/pkg/rmdsl/compile/rmv6_blocks.go` (new block writers).
- Then review `remarquee/pkg/rmdsl/compile/rmv6_page.go` (block ordering + CRDT id layout).
- Check `remarquee/pkg/rmdsl/compile/compile.go` + `remarquee/pkg/rmdsl/compile/content.go` for author UUID plumbing.

### Technical details
- Tests: `go test ./pkg/rmdsl/compile -count=1`
- Commands:
  - `go run ./cmd/remarquee rmdsl compile <case.yaml> --out ./rendering/rmq-0009-tests/<case>.rmdoc`
  - `go run ./cmd/remarquee cloud put --non-interactive --force ./rendering/rmq-0009-tests/<case>.rmdoc /remarquee/rendering/rmq-0009-tests`

## Step 9: Document remarks + rmscene block handling

I wrote a new analysis document that focuses on the Python `remarks` reader and the `rmscene` parser library, describing how they parse RMV6 blocks and build the SceneTree. The goal is to make the reader expectations explicit, so the RMQ-0009 writer can mirror the block ordering and fields that these tools rely on.

This step is documentation-only and captures concrete file references, symbol names, and pseudocode for the parsing flow. It also clarifies which blocks are parsed but not used by `remarks` and which ones are critical for tree assembly.

### What I did
- Authored `remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/analysis/04-remarks-rmscene-block-handling-and-semantics.md` with detailed block semantics and parsing pseudocode.
- Updated `remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/index.md` to link the new analysis doc.
- Related core rmscene/remarks files to the new doc for traceability.

### Why
- We need a reader-centric view of block semantics to ensure the compiler outputs the structure that existing tools accept and interpret correctly.

### What worked
- The analysis captures the exact flow from `read_blocks` to `build_tree` and the way `remarks` extracts typed text and highlights.

### What didn't work
- N/A (documentation only).

### What I learned
- `remarks` does not use line items directly in `parse_v6`, but it still relies on a valid SceneTree for glyph ranges and text.
- `SceneGroupItemBlock` is the critical join point that attaches group nodes into the tree; missing it causes tree assembly failures.

### What was tricky to build
- Distilling the block semantics into a concise flow while preserving the exact file and symbol references.

### What warrants a second pair of eyes
- Confirm that the summarized block ordering in `rmscene/tests/test_scene_stream.py` is representative of device notebooks beyond the typed-text sample.

### What should be done in the future
- If we implement highlights or typed text, cross-check the block emission against `rmscene` expectations in the new analysis doc.

### Code review instructions
- Start with `remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/analysis/04-remarks-rmscene-block-handling-and-semantics.md`.
- Spot-check referenced files: `rmscene/src/rmscene/scene_stream.py` and `remarks/remarks/conversion/parsing.py`.

### Technical details
- N/A (documentation only).

## Step 10: Add complicated validation cases and upload paired PDFs

I added a new set of more complex RMDoc-DSL examples that exercise layered strokes, rotated rectangles, ellipses, and highlight colors. The goal is to compare each generated `.rmdoc` against its rendered PDF side-by-side on the device.

I also scripted a batch compile + render-v6 + upload flow so we can quickly iterate on multiple cases and get immediate feedback on discrepancies.

**Commit (code):** 63b764f — "📝 RMQ-0009: add test fixtures and docs"

### What I did
- Added new DSL cases:
  - `remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/cases/09-complex-spiral-grid.yaml`
  - `remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/cases/10-layered-highlighter.yaml`
  - `remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/cases/11-two-pages-dense.yaml`
  - `remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/cases/12-tools-showcase-advanced.yaml`
- Added batch script: `remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/scripts/05-batch-compile-render-upload-complicated.sh`.
- Created remote folder `/remarquee/rendering/rmq-0009-complicated` and uploaded `.rmdoc` + `.pdf` pairs for all 4 cases.

### Why
- We need richer cases to detect rendering differences that don’t appear in the minimal test set.
- Pairing each `.rmdoc` with a rendered PDF on-device makes visual comparison faster.

### What worked
- Batch compile + render-v6 produced PDFs for all cases.
- Uploads succeeded once the remote directory existed.

### What didn't work
- Initial batch upload failed because the remote directory did not exist:
  - Command: `bash .../scripts/05-batch-compile-render-upload-complicated.sh`
  - Error: `Error: directory doesn't exist`
- First attempt to create the folder timed out:
  - Command: `go run ./cmd/remarquee cloud mkdir --non-interactive /remarquee/rendering/rmq-0009-complicated`
  - Error: `command timed out after 120010 milliseconds`

### What I learned
- `remarquee cloud put` requires the target directory to already exist.
- `cloud mkdir` can succeed but may need a longer timeout when rmapi is slow.

### What was tricky to build
- Keeping DSL cases within supported features (strokes, rectangles, ellipses, tool+color map) while still making them visually complex.

### What warrants a second pair of eyes
- Compare the `.rmdoc` and `.pdf` versions for each case to flag any tool/brush or coordinate discrepancies.

### What should be done in the future
- Add follow-on cases that mix highlight strokes with complex shapes once glyph/highlight blocks are implemented.

### Code review instructions
- Start with the new cases under `remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/cases/`.
- Review the batch script at `remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/scripts/05-batch-compile-render-upload-complicated.sh`.

### Technical details
- Commands:
  - `go run ./cmd/remarquee cloud mkdir --non-interactive /remarquee/rendering/rmq-0009-complicated`
  - `bash /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/scripts/05-batch-compile-render-upload-complicated.sh`

## Step 11: Upload PDFs with distinct basenames

After the first upload, PDFs did not appear on the tablet because `cloud put` strips the extension and uses the basename as the document name. That meant `.pdf` uploads collided with existing `.rmdoc` uploads.

I updated the batch script to render PDFs with a `-pdf` suffix in the filename so they appear as separate documents, then re-ran the upload.

**Commit (code):** 63b764f — "📝 RMQ-0009: add test fixtures and docs"

### What I did
- Updated the batch script to render PDFs as `*-pdf.pdf`.
- Re-ran the batch script to upload both `.rmdoc` and `*-pdf` documents.
- Confirmed the remote folder now lists both variants.

### Why
- The reMarkable cloud treats filenames without extension, so `.pdf` and `.rmdoc` with the same basename collide.

### What worked
- The folder now contains both notebook and PDF documents:
  - `09-complex-spiral-grid` and `09-complex-spiral-grid-pdf`
  - `10-layered-highlighter` and `10-layered-highlighter-pdf`
  - `11-two-pages-dense` and `11-two-pages-dense-pdf`
  - `12-tools-showcase-advanced` and `12-tools-showcase-advanced-pdf`

### What didn't work
- N/A (fix applied cleanly).

### What I learned
- `util.DocPathToName` strips extensions, so distinct basenames are required for parallel PDF + notebook uploads.

### What was tricky to build
- N/A (simple rename fix).

### What warrants a second pair of eyes
- Verify that the PDF entries are visible on-device and compare against their paired notebooks.

### What should be done in the future
- When pairing document formats, always encode the type in the basename to avoid rmapi collisions.

### Code review instructions
- Review the change in `remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/scripts/05-batch-compile-render-upload-complicated.sh`.

### Technical details
- Commands:
  - `bash /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/scripts/05-batch-compile-render-upload-complicated.sh`
  - `go run ./cmd/remarquee cloud ls --non-interactive /remarquee/rendering/rmq-0009-complicated`
