---
Title: Migrate remarquee to the new Glazed facade API
Ticket: RMQ-0017
Status: active
Topics:
    - go
    - cli
    - migration
    - glazed
    - remarquee
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/remarquee/cmds/cloud/account.go
      Note: |-
        Representative simple bare cloud command to use as the first migration slice
        Representative simple cloud command to use as the first migration slice
    - Path: cmd/remarquee/cmds/cloud/find.go
      Note: |-
        Representative dual-mode cloud command with legacy Glazed value APIs and structured-output defaults to preserve
        Representative dual-mode cloud command to preserve while migrating values and output defaults
    - Path: cmd/remarquee/cmds/cloud/rmapi.go
      Note: |-
        Shared cloud auth settings consumed by multiple command files
        Shared cloud auth settings used across the cloud command family
    - Path: cmd/remarquee/cmds/ocr/root.go
      Note: OCR command is the main special case because it also integrates Geppetto settings and middleware
    - Path: cmd/remarquee/cmds/rmdoc/render_v6.go
      Note: |-
        Representative rmdoc command with local/cloud orchestration and helper reuse
        Representative rmdoc command with local/cloud orchestration
    - Path: cmd/remarquee/cmds/rmdoc/render_v6_test.go
      Note: Representative test file still constructing legacy parsed layers directly
ExternalSources: []
Summary: Migrate remarquee command construction, settings decoding, tests, and OCR/Geppetto integration from Glazed's removed layers/parameters API to the current schema/fields/values facade API.
LastUpdated: 2026-04-10T09:30:00-04:00
WhatFor: Track the design and implementation work needed to make remarquee compile and behave correctly after the Glazed facade cutover.
WhenToUse: Use when planning, implementing, reviewing, or continuing the remarquee Glazed migration.
---


# Migrate remarquee to the new Glazed facade API

## Overview

`remarquee` still has broad usage of the removed Glazed `layers` / `parameters` command API. The current scan found 23 affected Go files, including cloud commands, rmdoc commands, two rmdoc tests, and one OCR command with Geppetto integration.

This ticket packages that work into a continuation-friendly migration plan with:

- a concrete implementation guide
- a detailed execution checklist
- explicit validation and delivery steps

The main risk is not the standard command migration itself; it is preserving behavior while updating the plumbing, especially for:

- dual-mode cloud commands that should still default to JSON in glaze mode
- rmdoc command helpers that already share local/cloud orchestration
- OCR, which also depends on Geppetto sections and middleware

## Key Links

- Implementation guide: `design/01-implementation-guide-glazed-facade-migration.md`
- Tasks: `tasks.md`
- Changelog: `changelog.md`
- Primary migration reference: `/home/manuel/code/wesen/go-go-golems/glazed/pkg/doc/tutorials/migrating-to-facade-packages.md` (repo-external reference; see guide for details)
- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- go
- cli
- migration
- glazed
- remarquee

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
