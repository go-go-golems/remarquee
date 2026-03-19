---
Title: Fix remarquee OAuth refresh (500 during device→user token exchange)
Ticket: RMQ-01-FIX-OAUTH-REFRESH
Status: active
Topics:
    - backend
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-04T10:05:00.099106019-05:00
WhatFor: ""
WhenToUse: ""
---

# Fix remarquee OAuth refresh (500 during device→user token exchange)

## Overview

This ticket investigates and hardens the “OAuth refresh” path used by `remarquee` (via `rmapi`) when fetching a **user token** from a cached **device token**.

The observed symptom was a hard failure on `remarquee cloud account --reauth`:

- `failed to create user token from device token request failed with status 500`

Key finding: the failure originated in the `rmapi` auth bootstrap code, which (pre-fix) could **hard-exit** the entire process on token refresh failures (including transient HTTP 500s). That made `remarquee` appear “broken” even when the backend error was transient.

## Key Links

- Design / analysis: `design-doc/01-oauth-refresh-failure-analysis-fix-plan.md`
- Diary: `reference/01-diary.md`
- Debugging + recovery playbook: `playbook/01-oauth-refresh-debugging-recovery.md`

## Status

Current status: **active**

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
