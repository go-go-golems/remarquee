---
Title: Add structured output to remarquee cloud find
Ticket: RMQ-0014
Status: active
Topics:
    - cli
    - glazed
    - implementation
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - cmd/remarquee/cmds/cloud/find.go:Target file to add structured output
    - cmd/remarquee/cmds/cloud/ls.go:Reference implementation (complete)
    - glazed/pkg/doc/tutorials/05-build-first-command.md:Glazed dual-mode tutorial
ExternalSources: []
Summary: |-
    Add Glazed structured output (JSON, YAML, CSV, table) to `remarquee cloud find`,
    following the dual-mode pattern established in `cloud ls`. Includes implementation
    guide for new team members.
LastUpdated: 2026-04-10T08:10:40.548634158-04:00
WhatFor: |-
    Enable scripting and automation for the `cloud find` command by adding machine-readable
    output formats while maintaining backward compatibility with human-readable output.
WhenToUse: |-
    Use when working on cloud command structured output, when adding GlazeCommand interfaces
    to existing commands, or when implementing dual-mode CLI patterns.
---

# Add structured output to remarquee cloud find

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- cli
- glazed
- implementation

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
