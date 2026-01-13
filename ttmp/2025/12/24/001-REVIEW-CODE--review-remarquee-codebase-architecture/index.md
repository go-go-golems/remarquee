---
Title: Review remarquee codebase architecture
Ticket: 001-REVIEW-CODE
Status: complete
Topics:
    - remarquee
    - go
    - architecture
    - review
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarquee/cmd/remarquee/cmds/cloud/root.go
      Note: Cloud command group; same init-error placeholder pattern
    - Path: remarquee/cmd/remarquee/cmds/rmdoc/build_background.go
      Note: Background-PDF builder; calls pkg/rmdoc/render.BuildBackgroundPDF
    - Path: remarquee/cmd/remarquee/cmds/rmdoc/inspect.go
      Note: Glazed-based inspect command; exercises pkg/rmdoc.OpenFile + prints page plan
    - Path: remarquee/cmd/remarquee/cmds/rmdoc/render_legacy.go
      Note: Legacy renderer; schema gating + rmapi annotations PDF generation
    - Path: remarquee/cmd/remarquee/cmds/rmdoc/root.go
      Note: rmdoc command group; demonstrates runtime init-error placeholder pattern
    - Path: remarquee/cmd/remarquee/cmds/status.go
      Note: Simple sanity-check command; validates CLI wiring
    - Path: remarquee/cmd/remarquee/main.go
      Note: CLI entrypoint; wires logging/help + command groups
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-24T09:06:03.647003847-05:00
WhatFor: ""
WhenToUse: ""
---




# Review remarquee codebase architecture

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
- architecture
- review

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
