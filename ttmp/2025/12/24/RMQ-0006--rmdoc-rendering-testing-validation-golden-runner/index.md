---
Title: 'RMDOC rendering: testing + validation + golden runner'
Ticket: RMQ-0006
Status: complete
Topics:
    - go
    - remarkable
    - testing
    - validation
    - rmdoc
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarquee/pkg/rmdoc/render/v6_merge_background.go
      Note: V6 merge pipeline now renders strokes + typed text
    - Path: remarquee/pkg/rmdoc/rmv6_anchor_pos.go
      Note: Uses shared layout constants
    - Path: remarquee/pkg/rmdoc/rmv6_root_text.go
      Note: RootTextBlock parser now discards subblocks eagerly
    - Path: remarquee/pkg/rmdoc/rmv6_text_document.go
      Note: Extract text paragraphs from RootTextBlock
    - Path: remarquee/pkg/rmdoc/rmv6_text_layout.go
      Note: Text layout constants shared by anchors/rendering
    - Path: remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/11-dump-test-rmdoc-stroke-tools-page1/main.go
      Note: Debug tool/color counts for Test.rmdoc
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-12T16:38:38.274519712-05:00
WhatFor: ""
WhenToUse: ""
---



# RMDOC rendering: testing + validation + golden runner

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **complete**

## Topics

- go
- remarkable
- testing
- validation
- rmdoc

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
