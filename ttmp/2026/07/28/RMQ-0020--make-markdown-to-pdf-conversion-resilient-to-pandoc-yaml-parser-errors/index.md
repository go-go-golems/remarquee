---
Title: Make Markdown to PDF conversion resilient to Pandoc YAML parser errors
Ticket: RMQ-0020
Status: active
Topics:
    - remarquee
    - upload
    - markdown
    - pdf
    - mdpdf
    - pandoc
    - xelatex
    - cli
    - go
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Intern-ready analysis and implementation guide for preventing Pandoc YAML parser failures caused by ordinary Markdown thematic breaks."
LastUpdated: 2026-07-28T06:20:00-04:00
WhatFor: "Track and implement the shared mdpdf fix for resilient Markdown-to-PDF conversion."
WhenToUse: "Read the design doc before changing Markdown preprocessing, Pandoc arguments, or upload conversion tests."
---

# Make Markdown to PDF conversion resilient to Pandoc YAML parser errors

## Overview

`remarquee upload md` currently fails on long-form Markdown that contains ordinary `---` thematic breaks because Pandoc's default `yaml_metadata_block` extension attempts to parse a later separator as YAML. This ticket documents the evidence, proposes disabling that extension after remarquee strips leading frontmatter, and gives an intern-ready implementation and test plan.

Current status: investigation and design are complete; production implementation is pending.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**. The ticket is ready for implementation review.

## Topics

- remarquee
- upload
- markdown
- pdf
- mdpdf
- pandoc
- xelatex
- cli
- go

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
