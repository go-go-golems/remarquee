---
Title: Add editor-friendly markdown upload layout
Ticket: RMQ-0014
Status: complete
Topics:
    - remarkable
    - upload
    - markdown
    - pdf
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/03/28/RMQ-0014--add-editor-friendly-markdown-upload-layout/design-doc/01-analysis-and-design-for-editor-friendly-markdown-upload-layout.md
      Note: Primary architecture and implementation analysis
    - Path: ttmp/2026/03/28/RMQ-0014--add-editor-friendly-markdown-upload-layout/playbook/01-intern-implementation-guide-for-markdown-upload-editor-layout.md
      Note: Intern-facing extension and validation guide
    - Path: ttmp/2026/03/28/RMQ-0014--add-editor-friendly-markdown-upload-layout/reference/01-diary.md
      Note: Chronological implementation record and validation evidence
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-28T09:28:20.058956177-04:00
WhatFor: Track the editor-friendly markdown upload layout feature, its design rationale, and the implementation guidance needed to extend it safely.
WhenToUse: Use this ticket when working on markdown-to-PDF upload presentation, annotation-friendly review layouts, or related pandoc option plumbing.
---


# Add editor-friendly markdown upload layout

## Overview

This ticket adds an annotation-friendly markdown upload layout to `remarquee`. The feature introduces a named `--layout editor` preset for markdown uploads and markdown bundles so uploaded PDFs leave more room for margin notes and feel less cramped during on-device editing.

The implementation keeps the existing default rendering intact. The new behavior is isolated behind an explicit preset in `pkg/mdpdf`, shared through a small upload-layer helper, covered by focused tests, and documented in the embedded upload help docs.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- Design doc: `design-doc/01-analysis-and-design-for-editor-friendly-markdown-upload-layout.md`
- Diary: `reference/01-diary.md`
- Intern guide: `playbook/01-intern-implementation-guide-for-markdown-upload-editor-layout.md`

## Status

Current status: **complete**

Implementation is complete and validated with focused Go tests. Ticket bookkeeping, doc validation, and reMarkable delivery are tracked in the companion docs.

## Topics

- remarkable
- upload
- markdown
- pdf

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design-doc/ - Architecture and design analysis for the layout preset
- reference/ - Chronological diary of commands, validation, and delivery
- playbook/ - Intern-facing implementation and extension guide
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
