---
Title: Make upload sync/md resilient to pandoc and upload errors
Ticket: RMQ-0015
Status: active
Topics:
    - remarquee
    - upload
    - pandoc
    - resilience
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/remarquee/cmds/upload/sync.go
      Note: executeSyncPlan now collects errors instead of aborting (commit c30268c)
ExternalSources: []
Summary: ""
LastUpdated: 2026-05-15T13:16:23.037677387-04:00
WhatFor: ""
WhenToUse: ""
---


# Make upload sync/md resilient to pandoc and upload errors

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- remarquee
- upload
- pandoc
- resilience

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
