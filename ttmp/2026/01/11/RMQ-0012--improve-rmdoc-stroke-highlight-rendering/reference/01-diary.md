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
    - Path: remarquee/ttmp/2026/01/11/RMQ-0012--improve-rmdoc-stroke-highlight-rendering/analysis/01-stroke-highlight-and-color-rendering-analysis.md
      Note: Detailed rendering analysis
    - Path: remarquee/ttmp/2026/01/11/RMQ-0012--improve-rmdoc-stroke-highlight-rendering/playbook/01-iterative-rendering-debug-loop.md
      Note: Iterative debug loop playbook
ExternalSources: []
Summary: Diary for RMQ-0012 rendering fidelity improvements.
LastUpdated: 2026-01-11T21:32:22-05:00
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
