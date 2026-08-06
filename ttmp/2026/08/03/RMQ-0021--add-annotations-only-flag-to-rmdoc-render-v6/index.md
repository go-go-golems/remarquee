---
Title: Add --annotations-only flag to rmdoc render-v6
Ticket: RMQ-0021
Status: active
Topics:
    - cli
    - rmdoc
    - rendering
    - remarkable
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Add --annotations-only to rmdoc render-v6 (annotations-only PDF export for V6 documents); design guide + diary."
LastUpdated: 2026-08-03T18:06:10.822185365-04:00
WhatFor: "Track design and implementation of annotations-only rendering for the V6 render verb."
WhenToUse: "When implementing or reviewing render-v6 --annotations-only or touching the pkg/rmdoc/render pipeline."
---

# Add --annotations-only flag to rmdoc render-v6

## Overview

Add an `--annotations-only` flag to `remarquee rmdoc render-v6` so V6 (cPages) documents can be exported as annotation-only PDFs (blank pages carrying only strokes, smart highlights, and typed text), with parity to `render-legacy --annotations-only`.

Motivation verified on 2026-08-03 with the real tablet document "TTC Garden Human Calibration Workbook" (legacy schema + V6 .rm files, annotations on pages 1-3 only): `render-legacy --annotations-only` fails with `Error: Unknown header`, while `render-v6` works but always composites the background.

Deliverables:
- Design/implementation guide: `design-doc/01-analysis-design-and-implementation-guide-for-render-v6-annotations-only.md`
- Diary: `reference/01-diary.md`

Status: design/analysis complete; implementation (Phases 1-4 in the design doc) not yet started.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- cli
- rmdoc
- rendering
- remarkable

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
