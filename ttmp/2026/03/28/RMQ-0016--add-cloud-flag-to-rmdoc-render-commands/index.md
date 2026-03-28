---
Title: Add --cloud flag to rmdoc render commands
Ticket: RMQ-0016
Status: active
Topics:
    - cli
    - cloud
    - rendering
    - rmdoc
    - remarkable
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/remarquee/cmds/rmdoc/render_v6.go
      Note: Existing V6 render command
    - Path: cmd/remarquee/cmds/rmdoc/render_legacy.go
      Note: Existing legacy render command
    - Path: cmd/remarquee/cmds/cloud/get.go
      Note: Existing cloud download command
    - Path: pkg/rmcloud/auth.go
      Note: Shared rmapi auth bootstrap
    - Path: pkg/rmdoc/open.go
      Note: Local archive opening API
    - Path: pkg/rmdoc/render/v6_merge_background.go
      Note: V6 renderer entrypoint that still takes an archive path
ExternalSources: []
Summary: Ticket for documenting and later implementing `--cloud` support on the `rmdoc` render commands, plus an adjacent review of CLI verb cleanup opportunities.
LastUpdated: 2026-03-28T10:35:12.47526385-04:00
WhatFor: Track design work for rendering `.rmdoc` files directly from reMarkable cloud paths without forcing a manual `cloud get` step first.
WhenToUse: Use when implementing, reviewing, or continuing RMQ-0016.
---

# Add --cloud flag to rmdoc render commands

## Overview

This ticket captures the design for a small but important UX improvement: allowing `remarquee rmdoc render-v6` and `remarquee rmdoc render-legacy` to accept remote cloud paths directly via `--cloud`. The central design choice is to keep rendering packages local-file-oriented and introduce a shared command-layer resolver that downloads remote documents into temporary local `.rmdoc` files before rendering.

The ticket also includes a separate CLI review document because the inspection work exposed a broader pattern: the current command tree mixes stable user-facing verbs with internal/debug tooling, and a few verbs overlap enough that they should eventually be tightened or deprecated.

## Key Links

- Design doc: `design-doc/01-design-and-implementation-guide-for-cloud-backed-rmdoc-rendering.md`
- Analysis doc: `analysis/01-cli-verb-review-and-tightening-recommendations.md`
- Diary: `reference/01-diary.md`
- Tasks: `tasks.md`
- Changelog: `changelog.md`

## Status

Current status: **active**

Deliverables produced in this ticket so far:

- detailed design and implementation guide
- CLI verb review and tightening recommendations
- investigation diary
- `docmgr doctor` passes cleanly for `RMQ-0016`
- reMarkable bundle uploaded to `/ai/2026/03/28/RMQ-0016`

## Topics

- cli
- cloud
- rendering
- rmdoc
- remarkable

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design-doc/ - Architecture and implementation guidance
- analysis/ - Review documents and broader codebase assessments
- reference/ - Diary and continuation-friendly notes
- playbooks/ - Command sequences and validation procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
