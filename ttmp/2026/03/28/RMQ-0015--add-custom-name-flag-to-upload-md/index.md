---
Title: Add custom name flag to upload md
Ticket: RMQ-0015
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
    - Path: ttmp/2026/03/28/RMQ-0015--add-custom-name-flag-to-upload-md/design-doc/01-analysis-and-implementation-plan-for-upload-md-custom-naming.md
      Note: Primary analysis and implementation plan
    - Path: ttmp/2026/03/28/RMQ-0015--add-custom-name-flag-to-upload-md/reference/01-diary.md
      Note: Chronological implementation and commit record
ExternalSources: []
Summary: Track the addition of `upload md --name`, including the single-document naming semantics, validation strategy, and ticket execution history.
LastUpdated: 2026-03-28T10:01:52.686673362-04:00
WhatFor: Document the design and implementation of a custom output-name override for `remarquee upload md`.
WhenToUse: Use this ticket when changing markdown upload naming behavior or extending `upload md` flag semantics.
---


# Add custom name flag to upload md

## Overview

This ticket adds `--name` to `remarquee upload md` so a user can choose the output document name instead of being forced to use the source markdown filename. The implemented behavior is intentionally narrow: the flag is valid only when the expanded input set contains exactly one markdown file.

That constraint keeps the feature unambiguous across dry-run, `--pdf-only`, and upload mode. It also avoids inventing surprising behavior for directory inputs or multi-file uploads where a single name cannot sensibly map to multiple generated PDFs.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- Design doc: `design-doc/01-analysis-and-implementation-plan-for-upload-md-custom-naming.md`
- Diary: `reference/01-diary.md`

## Status

Current status: **complete**

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

- design-doc/ - Architecture and implementation analysis
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
