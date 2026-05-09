---
Title: 'Improve remarquee skill/CLI: reduce tool-call churn in agent sessions'
Ticket: REMARQUEE-IMPROVE
Status: active
Topics:
    - remarquee
    - minitrace
    - transcript-analysis
    - tool-churn
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../.pi/agent/skills/remarkable-upload/SKILL.md
      Note: P0 skill rewrite - eliminated status pre-check
    - Path: ../../../../../../../../../code/wesen/go-go-golems/remarquee/cmd/remarquee/cmds/upload/root.go
      Note: P1 filename sanitization function
    - Path: ../../../../../../../../../code/wesen/obsidian-vault/Projects/2026/05/08/ARTICLE - Transcript Mining - Using go-minitrace to Find and Fix Tool-Call Churn in Agent Sessions.md
      Note: Obsidian vault article teaching the method
    - Path: go-minitrace/pkg/query/engine.go
      Note: NormalizeValue is the single fix point for BigInt→Number coercion at the DuckDB→Goja boundary
    - Path: remarquee/ttmp/2026/05/08/REMARQUEE-IMPROVE--improve-remarquee-skill-cli-reduce-tool-call-churn-in-agent-sessions/scripts/js/remarquee-analysis/05-remarquee-sequences.js
      Note: Temporal sequence detection
    - Path: remarquee/ttmp/2026/05/08/REMARQUEE-IMPROVE--improve-remarquee-skill-cli-reduce-tool-call-churn-in-agent-sessions/scripts/js/remarquee-analysis/06-remarquee-subcommand-summary.js
      Note: Subcommand distribution query command
    - Path: remarquee/ttmp/2026/05/08/REMARQUEE-IMPROVE--improve-remarquee-skill-cli-reduce-tool-call-churn-in-agent-sessions/scripts/js/remarquee-analysis/07-remarquee-failures.js
      Note: Failure classification
    - Path: remarquee/ttmp/2026/05/08/REMARQUEE-IMPROVE--improve-remarquee-skill-cli-reduce-tool-call-churn-in-agent-sessions/scripts/js/remarquee-analysis/09-remarquee-churn-metrics.js
      Note: Churn score computation
    - Path: remarquee/ttmp/2026/05/08/REMARQUEE-IMPROVE--improve-remarquee-skill-cli-reduce-tool-call-churn-in-agent-sessions/scripts/js/remarquee-analysis/10-remarquee-failure-mode-summary.js
      Note: Failure mode aggregation
ExternalSources: []
Summary: ""
LastUpdated: 2026-05-08T06:10:46.55559561-04:00
WhatFor: ""
WhenToUse: ""
---










# Improve remarquee skill/CLI: reduce tool-call churn in agent sessions

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- remarquee
- minitrace
- transcript-analysis
- tool-churn

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
