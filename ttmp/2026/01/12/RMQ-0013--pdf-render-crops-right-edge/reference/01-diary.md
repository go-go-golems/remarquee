---
Title: Diary
Ticket: RMQ-0013
Status: active
Topics:
    - rendering
    - remarkable
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/remarquee/cmds/rmdoc/vlm_validate.go
      Note: Unified inputs and rmdoc rendering
    - Path: pkg/rmdoc/render/v6_merge_background.go
      Note: Blank-page bbox padding + margin balancing logic
    - Path: ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/analysis/01-debug-strategy-right-edge-crop-in-pdf-render.md
      Note: Debug strategy analysis
    - Path: ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/analysis/02-preliminary-questions-cropping-checks-vlm-loop.md
      Note: Preliminary questions and VLM loop
    - Path: ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/01-baseline/README.md
      Note: Baseline comparison writeup
    - Path: ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/02-bbox-expand/README.md
      Note: BBox expansion comparison writeup
    - Path: ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/03-stroke-pad/README.md
      Note: Stroke-pad comparison writeup
    - Path: ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/04-margin-balance/README.md
      Note: Margin-balance comparison writeup
    - Path: ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/scripts/compute_bbox.go
      Note: BBox helper for inspecting stroke extents
ExternalSources: []
Summary: Diary for RMQ-0013 right-edge PDF crop investigation.
LastUpdated: 2026-01-12T13:34:17-05:00
WhatFor: Track progress and validation steps for the RMQ-0013 right-edge PDF crop bug.
WhenToUse: Update after each investigation or validation step.
---





# Diary

## Goal

Track the investigation and documentation for the right-edge PDF crop bug so we can reproduce, fix, and validate it.

## Step 1: Capture debug strategy with VLM + human validation loops

I created the RMQ-0013 diary and wrote a debug strategy that identifies likely sources of right-edge cropping, lays out a step-by-step reproduction loop, and includes both VLM checks and human-in-the-loop confirmation via `plz-confirm`. The strategy captures which functions to inspect and how to validate fixes with side-by-side comparisons.

This gives us a structured approach to narrow the crop to specific layout math and to re-test fixes without losing context.

**Commit (code):** N/A

### What I did
- Read `plz-confirm help how-to-use` to confirm available widgets and flags.
- Authored the debug strategy analysis for the crop issue.
- Created the RMQ-0013 diary entry.

### Why
- The crop issue needs a repeatable loop that mixes automation with human validation.

### What worked
- The analysis captures concrete file targets and verification steps for VLM + human review.

### What didn't work
- N/A

### What I learned
- plz-confirm supports confirm/form/image widgets with explicit wait-timeout and base-url controls.

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- Validate that the recommended debug steps map cleanly to current CLI commands and file paths.

### What should be done in the future
- N/A

### Code review instructions
- Review `remarquee/ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/analysis/01-debug-strategy-right-edge-crop-in-pdf-render.md`.

### Technical details
- Command: `plz-confirm help how-to-use`

## Step 2: Add preliminary VLM loop questions and expand pipeline overview

I added a new analysis doc to capture the earliest questions and a VLM-first validation loop, and I expanded the main debug strategy with an explicit rendering overview and pseudocode showing where cropping can be introduced. The updated guidance stresses that PDF vs PNG comparisons are redundant (same source), and it introduces a plz-confirm gate before any device screenshot is taken.

This ensures we start with fast, repeatable VLM checks on rendered PNGs, then only involve the device once we’ve confirmed the target page is visible.

**Commit (code):** N/A

### What I did
- Wrote the preliminary questions + VLM-first loop analysis.
- Updated the main debug strategy to include rendering overview, pseudocode, and crop injection points.
- Clarified the human gating requirement before device screenshots.

### Why
- The bug needs a staged validation loop that avoids unnecessary device steps and targets the likely crop sources.

### What worked
- The analysis now outlines both the VLM-only phase and the human-confirmed screenshot phase.

### What didn't work
- N/A

### What I learned
- VLM validation can start from rendered PNGs alone, and human gating should precede screenshots to avoid misaligned captures.

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- Confirm the pipeline overview matches the current renderer and file paths.

### What should be done in the future
- N/A

### Code review instructions
- Review `remarquee/ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/analysis/02-preliminary-questions-cropping-checks-vlm-loop.md`.
- Review `remarquee/ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/analysis/01-debug-strategy-right-edge-crop-in-pdf-render.md`.

### Technical details
- N/A

## Step 3: Add RMQ-0013 tasks for unified VLM inputs

I created explicit implementation tasks to unify the `vlm-validate` inputs so it can accept images, PDFs, or rmdoc+pages sources, and to ensure the new flow reuses the render-v6-png logic. This turns the debug strategy into a concrete execution plan.

**Commit (code):** N/A

### What I did
- Added RMQ-0013 tasks covering multi-source inputs, rmdoc rendering, and documentation updates.

### Why
- The new validation loop needs explicit tasks to ensure we implement the correct input precedence and reuse the PNG renderer.

### What worked
- Tasks now capture the implementation scope for the unified VLM inputs.

### What didn't work
- N/A

### What I learned
- N/A

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- N/A

### What should be done in the future
- N/A

### Code review instructions
- Review `remarquee/ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/tasks.md`.

### Technical details
- Commands:
  - `docmgr task add --ticket RMQ-0013 --text "Extend vlm-validate inputs: accept image, pdf, or rmdoc sources (A/B) with page selection"`
  - `docmgr task add --ticket RMQ-0013 --text "Implement rmdoc->PNG path in vlm-validate (reuse render-v6-png logic, subset pages)"`
  - `docmgr task add --ticket RMQ-0013 --text "Document vlm-validate source precedence + plz-confirm gating in debug strategy"`

## Step 4: Implement unified vlm-validate inputs (image/pdf/rmdoc)

I updated `vlm-validate` to accept PNGs, PDFs, or `.rmdoc` inputs for both A and B sources, with mutually exclusive selection and shared page handling. For `.rmdoc` inputs the command now renders only the requested pages using the V6 merge pipeline and then rasterizes them to PNGs before invoking `pinocchio`.

I also refreshed the analysis docs to reflect the new unified flow and updated the VLM-first debug loop language to avoid PDF/PNG comparisons and to gate screenshots behind `plz-confirm`.

**Commit (code):** cdf96b4 — "RMQ-0013: unify vlm-validate inputs"

### What I did
- Added `--image-a/--image-b` and `--rmdoc-a/--rmdoc-b` inputs to `vlm-validate`.
- Implemented source resolution with mutual exclusion per side.
- Reused the `render-v6-png` rasterization path for `.rmdoc` inputs.
- Updated analysis docs to match the unified flow and screenshot gating requirements.
- Marked tasks 2–4 complete.

### Why
- The debug workflow needs a single command that can work with PNGs, PDFs, or `.rmdoc` inputs without extra manual steps.

### What worked
- `vlm-validate` now routes each input through the correct render path and reuses Poppler for rasterization.

### What didn't work
- N/A

### What I learned
- The Poppler rasterizer can be reused for `.rmdoc` outputs by rendering a subset PDF first.

### What was tricky to build
- Preserving original page numbers when rendering a subset PDF for `.rmdoc` inputs.

### What warrants a second pair of eyes
- Confirm the input precedence rules are intuitive and the new flags do not break existing usage.

### What should be done in the future
- N/A

### Code review instructions
- Review `remarquee/cmd/remarquee/cmds/rmdoc/vlm_validate.go` for input routing and rmdoc rendering.
- Review the updated analysis docs for consistency with the CLI behavior.

### Technical details
- Command: `go test ./cmd/remarquee -count=1`

## Step 5: Capture device screenshot and run VLM comparison on page 65

I rendered page 65 to PNG, captured a device screenshot (per your note that the page was already open), and ran the new `vlm-validate` image-to-image comparison. The VLM reported more noticeable right-edge clipping in the rendered PNG than in the device screenshot.

This gives us a concrete cross-check on a real page and confirms the cropping issue is visible in the renderer output.

**Commit (code):** N/A

### What I did
- Rendered `/Journal` page 65 to PNG with `render-v6-png`.
- Captured a device screenshot for page 65.
- Ran `vlm-validate` with `--image-a` and `--image-b` to compare right-edge alignment.

### Why
- Validate the crop issue against a live device screenshot and confirm the VLM can flag it.

### What worked
- The VLM reported right-edge clipping in the rendered PNG and better alignment in the device screenshot.

### What didn't work
- N/A

### What I learned
- The image-to-image VLM path is usable for detecting right-edge cropping.

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- Confirm the device was truly on page 65 at capture time (per human confirmation).

### What should be done in the future
- N/A

### Code review instructions
- N/A (validation-only step).

### Technical details
- Render: `go run ./cmd/remarquee rmdoc render-v6-png /tmp/rmq-0012-journal/Journal.rmdoc --pages 65 --out-dir /tmp/rmq-0013-crop --force`
- Screenshot: `go run ./cmd/remarquee device screenshot --url http://10.11.99.1:2718 --username admin --password password --out /tmp/rmq-0013-crop/device-page-065.png`
- VLM: `go run ./cmd/remarquee rmdoc vlm-validate --image-a /tmp/rmq-0013-crop/Journal-v6-page-065.png --image-b /tmp/rmq-0013-crop/device-page-065.png --prompt "Compare right-edge alignment. Is any content clipped on the right? Describe where."`

## Step 6: Store VLM artifacts in ticket log

I created a log document for VLM comparison runs and copied the rendered/device images into the ticket log directory alongside the exact commands and OCR/VLM outputs. This preserves the evidence needed to reproduce the crop analysis.

**Commit (code):** N/A

### What I did
- Added a log doc for VLM runs.
- Copied the PNGs used in VLM comparisons into the ticket log directory.
- Recorded the command lines and outputs for pages 76 and 65.

### Why
- The investigation needs a permanent, ticket-local record of inputs and outputs for each VLM run.

### What worked
- Images and outputs are now stored under `ttmp/.../log`.

### What didn't work
- N/A

### What I learned
- N/A

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- N/A

### What should be done in the future
- N/A

### Code review instructions
- Review `remarquee/ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/01-vlm-comparison-runs.md`.

### Technical details
- Copies:
  - `/tmp/remarquee-vlm-validate-3080623130/A-page-076.png` -> `remarquee/ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/vlm-A-page-076.png`
  - `/tmp/rmq-0013-crop/Journal-v6-page-065.png` -> `remarquee/ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/render-page-065.png`
  - `/tmp/rmq-0013-crop/device-page-065.png` -> `remarquee/ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/device-page-065.png`

## Step 7: Re-capture page 65 artifacts in ticket log directory

I reran the page 65 render and device screenshot using only the ticket log directory (no `/tmp`) and recorded the fresh VLM output. This ensures the canonical artifacts live inside the ticket and match the confirmed device page.

**Commit (code):** N/A

### What I did
- Rendered page 65 directly into `ttmp/.../log`.
- Captured the device screenshot to the same log directory.
- Ran `vlm-validate` with `--out-dir` inside the log folder.
- Updated the VLM log entry with the new run output.

### Why
- The ticket should store all artifacts in one place and avoid `/tmp`.

### What worked
- The VLM again flagged right-edge clipping in the render vs device screenshot.

### What didn't work
- N/A

### What I learned
- The log directory workflow is sufficient for repeatable runs without `/tmp`.

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- N/A

### What should be done in the future
- N/A

### Code review instructions
- Review `remarquee/ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/01-vlm-comparison-runs.md`.

### Technical details
- Render: `go run ./cmd/remarquee rmdoc render-v6-png /tmp/rmq-0012-journal/Journal.rmdoc --pages 65 --out-dir ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log --force`
- Screenshot: `go run ./cmd/remarquee device screenshot --url http://10.11.99.1:2718 --username admin --password password --out ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/device-page-065.png`
- VLM: `go run ./cmd/remarquee rmdoc vlm-validate --image-a ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/Journal-v6-page-065.png --image-b ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/device-page-065.png --out-dir ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/vlm-20260112-133346 --prompt "Compare right-edge alignment. Is any content clipped on the right? Describe where."`

## Step 8: Balance notebook margins for blank-page renders

I iterated on blank-background (notebook) rendering to address the right-edge crop. After switching to bbox-sized pages, I added stroke-width padding and then introduced a margin-balancing step that extends the tighter side when the stroke bbox overflows the default screen width.

I re-rendered page 65 for each case (bbox-only, stroke-pad, margin-balance), ran VLM comparisons, and captured each run in per-case log folders with writeups. The margin-balanced output now yields symmetric left/right margins, though the VLM still reports right-edge clipping against the device screenshot.

**Commit (code):** 578ab36 — "RMQ-0013: balance notebook margins for blank pages"

### What I did
- Added `maxStrokeWidthScreenUnits` and blank-page bbox padding/margin-balancing logic in `v6_merge_background.go`.
- Verified rendering via `go test ./pkg/rmdoc/render -run TestMergeRMDocV6OntoBackgroundPDF_Smoke -count=1`.
- Rendered page 65 to PNG for each iteration and ran `vlm-validate` comparisons.
- Created per-case log folders (`01-baseline`, `02-bbox-expand`, `03-stroke-pad`, `04-margin-balance`) with writeups.
- Saved the bbox helper script to the ticket scripts folder.

### Why
- The crop appears when notebook pages exceed the default screen width; we need a margin strategy that preserves content without shifting PDF-backed pages.

### What worked
- Margin balancing yields symmetric left/right margins in the rendered PNGs, eliminating the zero-margin right edge seen in earlier attempts.

### What didn't work
- VLM still reports right-edge clipping against the device screenshot after the margin-balance iteration.

### What I learned
- The page 65 stroke bbox exceeds the default screen width, so bbox-based sizing alone still leaves the right edge tight.

### What was tricky to build
- Adjusting notebook-only layout math without perturbing background-aligned pages.

### What warrants a second pair of eyes
- Validate the margin-balancing logic against device expectations (and confirm it does not regress PDF-backed documents).

### What should be done in the future
- Re-check the margin-balanced output with a human-confirmed device screenshot (optionally crop toolbar for a more direct comparison).

### Code review instructions
- Start at `remarquee/pkg/rmdoc/render/v6_merge_background.go` (blank-background branches + `maxStrokeWidthScreenUnits`).
- Review the case writeups under `remarquee/ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/`.

### Technical details
- Commands:
  - `go test ./pkg/rmdoc/render -run TestMergeRMDocV6OntoBackgroundPDF_Smoke -count=1`
  - `go run ./cmd/remarquee rmdoc render-v6-png /tmp/rmq-0012-journal/Journal.rmdoc --pages 65 --out-dir ...`
  - `go run ./cmd/remarquee rmdoc vlm-validate --image-a ... --image-b ... --prompt "Compare right-edge alignment..."`
