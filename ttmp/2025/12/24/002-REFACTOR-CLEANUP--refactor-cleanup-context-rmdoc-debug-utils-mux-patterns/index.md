---
Title: 'Refactor cleanup: context, rmdoc debug utils, mux patterns'
Ticket: 002-REFACTOR-CLEANUP
Status: active
Topics:
    - remarquee
    - go
    - refactor
    - cleanup
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarquee/cmd/remarquee-ui/api/inspect.go
      Note: Use request context for OpenFile (commit 4b73281)
    - Path: remarquee/cmd/remarquee-ui/api/render.go
      Note: Use request context for OpenFile+render (commit 4b73281)
    - Path: remarquee/cmd/remarquee/cmds/rmdoc/build_background.go
      Note: Use cobra-derived context for OpenFile/BuildBackgroundPDF (commit 4b73281)
    - Path: remarquee/cmd/remarquee/cmds/rmdoc/inspect.go
      Note: Use cobra-derived context for OpenFile (commit 4b73281)
    - Path: remarquee/pkg/rmdoc/open.go
      Note: Honor ctx in OpenReaderAt/readZipFile (commit 48d822e)
    - Path: remarquee/pkg/rmdoc/render/background.go
      Note: Honor ctx between pages in BuildBackgroundPDF (commit 48d822e)
    - Path: remarquee/ttmp/2025/12/24/002-REFACTOR-CLEANUP--refactor-cleanup-context-rmdoc-debug-utils-mux-patterns/reference/01-diary.md
      Note: Implementation diary for this ticket
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-24T08:57:13.942052994-05:00
WhatFor: ""
WhenToUse: ""
---



# Refactor cleanup: context, rmdoc debug utils, mux patterns

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- remarquee
- go
- refactor
- cleanup

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
