---
Title: remarquee upload next features (bundle/ToC, preserve dirs, upload src)
Ticket: RMQ-0005
Status: complete
Topics:
    - backend
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/remarquee/cmds/upload/md.go
      Note: Upload baseline; RMQ-0005 extends it
    - Path: pkg/mdpdf/pandoc.go
      Note: Pandoc runner to extend (ToC + syntax highlighting)
    - Path: pkg/rmcloud/dirs.go
      Note: Remote mkdir -p used for preserve-dirs
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-12T16:16:35.149737712-05:00
WhatFor: ""
WhenToUse: ""
---



# remarquee upload next features (bundle/ToC, preserve dirs, upload src)

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **complete**

## Topics

- backend

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
