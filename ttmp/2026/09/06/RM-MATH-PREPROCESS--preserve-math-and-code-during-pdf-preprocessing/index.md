---
Title: Preserve math and code during PDF preprocessing
Ticket: RM-MATH-PREPROCESS
Status: complete
Topics:
    - markdown
    - pdf
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Default stmaryrd support and shared protection of display math and fenced code, implemented and tested in a dedicated worktree.
LastUpdated: 2026-09-06T21:04:53.902303075-04:00
WhatFor: ""
WhenToUse: ""
---


# Preserve math and code during PDF preprocessing

## Overview

Implemented both rendering fixes in commit `e9d9e779458849ead417e685e807df93ce24b6b7`, branch `fix/math-safe-markdown`, worktree `/home/manuel/worktrees/remarquee-math-safe-markdown`. Ordinary list behavior remains unchanged outside supported literal regions. Focused tests, real direct/bundle PDF rendering, full repository tests and lint pass. Generated frontend assets remain ignored; the original checkout and installed CLI are unchanged.

## Key Links

- [Quick design](design-doc/01-math-safe-preprocessing-and-default-symbol-support.md)
- [Implementation diary](reference/01-implementation-diary.md)
- [PR #26](https://github.com/go-go-golems/remarquee/pull/26)

## Status

Implementation and requested PR creation are complete; PR review/merge remains separate.

## Topics

- markdown
- pdf

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
