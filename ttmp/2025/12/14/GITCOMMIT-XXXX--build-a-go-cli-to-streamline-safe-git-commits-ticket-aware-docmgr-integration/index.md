---
Title: Build a Go CLI to streamline safe git commits (ticket-aware + docmgr integration)
Ticket: GITCOMMIT-XXXX
Status: complete
Topics:
    - devtools
    - go
    - git
    - productivity
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarquee/cmd/remarquee/cmds/rmdoc/inspect.go
      Note: Example motivating ticket-scoped staging and commit guardrails
    - Path: remarquee/ttmp/2025/12/14/GITCOMMIT-XXXX--build-a-go-cli-to-streamline-safe-git-commits-ticket-aware-docmgr-integration/design-doc/01-project-description-git-commit-helper-cli-go.md
      Note: Project description and proposed CLI commands
    - Path: remarquee/ttmp/2025/12/14/GITCOMMIT-XXXX--build-a-go-cli-to-streamline-safe-git-commits-ticket-aware-docmgr-integration/scripts/01-gitcommit-prototype.sh
      Note: |-
        Prototype helper script (safe staging + commit templates)
        Prototype script (preview-first safe staging/commit)
    - Path: remarquee/ttmp/2025/12/14/RMQ-0004--port-rmdoc-parsing-rendering-to-go-v3-v5-v6/reference/01-diary.md
      Note: Recent workflow example where unrelated staged files can slip into commits
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-12T23:49:15.186097884-05:00
WhatFor: ""
WhenToUse: ""
---





# Build a Go CLI to streamline safe git commits (ticket-aware + docmgr integration)

## Overview

This ticket tracks a future Go CLI tool (working name: `gitcommit`) that helps us do safe, ticket-scoped commits efficiently. It is explicitly motivated by mistakes that happen in this repo’s workflow: committing unrelated files, committing in the wrong repo root, and forgetting the docmgr diary/changelog updates that make work reproducible.

The detailed project description lives in:
- `design-doc/01-project-description-git-commit-helper-cli-go.md`

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **complete**

## Topics

- devtools
- go
- git
- productivity

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
