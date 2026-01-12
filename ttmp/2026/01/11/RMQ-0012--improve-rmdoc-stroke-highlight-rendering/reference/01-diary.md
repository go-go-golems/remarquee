---
Title: Diary
Ticket: RMQ-0012
Status: active
Topics:
    - backend
    - rendering
    - remarkable
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarquee/cmd/remarquee/cmds/rmdoc/render_v6_png.go
      Note: PNG render command
    - Path: remarquee/cmd/remarquee/cmds/rmdoc/root.go
      Note: Register render-v6-png
    - Path: remarquee/ttmp/2026/01/11/RMQ-0012--improve-rmdoc-stroke-highlight-rendering/analysis/01-stroke-highlight-and-color-rendering-analysis.md
      Note: Detailed rendering analysis
    - Path: remarquee/ttmp/2026/01/11/RMQ-0012--improve-rmdoc-stroke-highlight-rendering/playbook/01-iterative-rendering-debug-loop.md
      Note: Iterative debug loop playbook
ExternalSources: []
Summary: Diary for RMQ-0012 rendering fidelity improvements.
LastUpdated: 2026-01-12T12:30:12-05:00
WhatFor: Track RMQ-0012 work to improve stroke/highlight rendering fidelity and validation loops.
WhenToUse: Update after each rendering comparison iteration or tooling change.
---




# Diary

## Goal

Track the analysis, playbooks, and implementation steps for improving rmdoc stroke/highlight rendering fidelity.

## Step 1: Create RMQ-0012 and draft the iterative debug playbook

I created the RMQ-0012 workspace and wrote the first playbook that describes an iterative comparison loop between remarquee's V6 renderer and device screenshots. The playbook includes the user confirmation gate, screenshot capture over HTTP, and a pinocchio comparison step so rendering regressions are detected quickly.

This gives us a repeatable debugging workflow to use while improving stroke thickness, highlight opacity, and color mapping, and it outlines how to keep the device view synchronized with the rendered output.

**Commit (code):** N/A

### What I did
- Created the RMQ-0012 ticket workspace.
- Added the iterative rendering debug playbook.

### Why
- We need a structured workflow to compare rendering fidelity against the device while refining stroke/highlight output.

### What worked
- The playbook captures the intended loop and includes VLM + user confirmation gates.

### What didn't work
- N/A

### What I learned
- N/A

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- Ensure the playbook steps match the actual CLI defaults and device capture endpoints.

### What should be done in the future
- N/A

### Code review instructions
- Review `remarquee/ttmp/2026/01/11/RMQ-0012--improve-rmdoc-stroke-highlight-rendering/playbook/01-iterative-rendering-debug-loop.md`.

### Technical details
- N/A

## Step 2: Analyze stroke, highlight, and color rendering path

I reviewed the V6 decoding and rendering pipeline with a focus on how stroke widths, highlight opacity, and color mapping are applied. The analysis documents the current rendering math, identifies where we ignore per-point stroke dynamics, and captures the exact code paths that set highlight alpha and color.

This gives us a concrete map of which functions to adjust when we start improving fidelity, and it highlights the most likely sources of mismatch against device output.

**Commit (code):** N/A

### What I did
- Read the V6 stroke decode and glyph decode implementations.
- Traced the merge renderer path that draws strokes and applies smart highlight annotations.
- Wrote an analysis doc with code snippets, pseudocode, and pipeline diagrams.

### Why
- We need a detailed baseline of the current rendering path to make targeted improvements to stroke thickness and highlight behavior.

### What worked
- The analysis doc now captures the tool-specific stroke style mapping and highlight annotation pipeline.

### What didn't work
- N/A

### What I learned
- The renderer uses per-stroke `ThicknessScale` but ignores per-point width and pressure.
- Highlight alpha is hard-coded in both stroke styles and PDF highlight annotations.

### What was tricky to build
- N/A (analysis-only step).

### What warrants a second pair of eyes
- Confirm that the tool IDs used in `strokeStyleForTool` match the latest rmscene/rmc definitions.

### What should be done in the future
- Use the analysis doc to define an incremental set of fidelity improvements.

### Code review instructions
- Start with `remarquee/ttmp/2026/01/11/RMQ-0012--improve-rmdoc-stroke-highlight-rendering/analysis/01-stroke-highlight-and-color-rendering-analysis.md`.

### Technical details
- N/A

## Step 3: Add PNG render command and export Journal pages

I added a new `render-v6-png` command that renders a V6 .rmdoc to a merged PDF and rasterizes selected pages to PNGs via Poppler. I then used this command to export the last two pages of `/Journal` into PNG files for review.

This provides the missing PNG extraction workflow the ticket needs and gives us concrete artifacts for comparing stroke and highlight fidelity.

**Commit (code):** 1c64d16 — "RMQ-0012: add rmdoc render-v6-png"

### What I did
- Implemented `remarquee rmdoc render-v6-png` with page selection, output directory, and overwrite control.
- Wired the new command into the rmdoc root command.
- Downloaded `/Journal` and rendered the last two pages to PNGs.

### Why
- We need a CLI verb that produces PNGs directly from a .rmdoc to support fast visual comparisons.

### What worked
- The new command rendered pages 76 and 77 to `/tmp/rmq-0012-journal/png/Journal-v6-page-076.png` and `...077.png`.

### What didn't work
- N/A

### What I learned
- `render-v6-png` can reuse the existing Poppler rasterization flow from `vlm-validate`.

### What was tricky to build
- Ensuring overwrite checks for both the generated PDF and per-page PNG outputs.

### What warrants a second pair of eyes
- Confirm that the output naming convention and overwrite behavior match expected CLI ergonomics.

### What should be done in the future
- N/A

### Code review instructions
- Start in `remarquee/cmd/remarquee/cmds/rmdoc/render_v6_png.go`.
- Check `remarquee/cmd/remarquee/cmds/rmdoc/root.go` for the new command wiring.
- Validate by running `go run ./cmd/remarquee rmdoc render-v6-png <file.rmdoc> --pages 1 --out-dir /tmp/out --force`.

### Technical details
- Download: `go run ./cmd/remarquee cloud get /Journal --out-dir /tmp/rmq-0012-journal`
- Inspect: `go run ./cmd/remarquee rmdoc inspect /tmp/rmq-0012-journal/Journal.rmdoc`
- Render: `go run ./cmd/remarquee rmdoc render-v6-png /tmp/rmq-0012-journal/Journal.rmdoc --pages 76,77 --out-dir /tmp/rmq-0012-journal/png --force`
