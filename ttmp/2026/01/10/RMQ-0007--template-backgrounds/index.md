---
Title: 'RMDoc templates: background rendering'
Ticket: RMQ-0007
Status: active
Topics:
  - remarkable
  - rmdoc
  - rendering
  - templates
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: >
  Implement template backgrounds for RMDoc notebooks (P Dots S, grid, lined, etc.)
  so that V6 renders match device/remarks expectations and goldens become more faithful.
LastUpdated: 2026-01-10
---

# RMDoc templates: background rendering

## Overview

This ticket focuses on rendering notebook template backgrounds (dots/grid/lines/etc.) for `.rmdoc` pages. Today, templates are present in `.content` (cPages) but our background generation largely ignores them. That makes “blank notebook pages” hard to compare visually (and makes some device-vs-export comparisons misleading).

## Key Links

- **Parent context**: RMQ-0006 (golden tests + debugging framework)

## Status

Current status: **active**

## Topics

- remarkable
- rmdoc
- rendering
- templates

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.


