---
Title: Migrate remarquee to the new Glazed facade API
Ticket: RMQ-0017
Status: complete
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
    - Path: cmd/remarquee/cmds/rmdoc/input_resolver.go
      Note: Shared cloud-input resolver migrated from parsed layers to parsed values
    - Path: cmd/remarquee/cmds/rmdoc/render_legacy.go
      Note: Legacy renderer command migrated to the new section/value API
    - Path: cmd/remarquee/cmds/rmdoc/render_v6.go
      Note: |-
        Representative rmdoc command with local/cloud orchestration and helper reuse
        Representative rmdoc command with local/cloud orchestration
    - Path: cmd/remarquee/cmds/rmdoc/render_v6_test.go
      Note: Representative test file still constructing legacy parsed layers directly
    - Path: cmd/remarquee/cmds/rmdoc/test_values_helpers_test.go
      Note: New helper for building values-based test fixtures from command sections
    - Path: cmd/remarquee/main.go
      Note: Final logging helper rename required for full-repo validation
    - Path: go.mod
      Note: Dependency bump required to access modern Glazed and Geppetto APIs
ExternalSources: []
Summary: Migrate remarquee command construction, settings decoding, tests, and OCR/Geppetto integration from Glazed's removed layers/parameters API to the current schema/fields/values facade API.
LastUpdated: 2026-04-10T10:20:00-04:00
WhatFor: Track the design and implementation work needed to make remarquee compile and behave correctly after the Glazed facade cutover.
WhenToUse: Use when planning, implementing, reviewing, or continuing the remarquee Glazed migration.
---



# Migrate remarquee to the new Glazed facade API

## Overview

This migration is complete. `remarquee` no longer uses the removed Glazed `layers` / `parameters` command API in its command code, rmdoc tests, or OCR integration.

The finished work included:

- upgrading `glazed` from `v0.7.8` to `v1.2.1`
- upgrading `geppetto` from `v0.6.0` to `v0.11.9`
- migrating the full `cloud` command package to `schema` / `fields` / `values`
- preserving dual-mode cloud behavior, including JSON default output in glaze mode
- migrating the full `rmdoc` command family and replacing legacy parsed-layer test helpers
- migrating OCR to Geppetto sections plus `factory.NewEngineFromParsedValues(...)`
- fixing the final root-command logging helper rename
- validating the whole repository with `go test ./...`

The most important implementation caveat discovered during the work is that this migration required a dependency bump as well as source changes: the older pinned Glazed and Geppetto versions did not expose the needed APIs.
## Key Links

- Implementation guide: `design/01-implementation-guide-glazed-facade-migration.md`
- Diary: `reference/01-diary.md`
- Tasks: `tasks.md`
- Changelog: `changelog.md`
- Primary migration reference: `/home/manuel/code/wesen/go-go-golems/glazed/pkg/doc/tutorials/migrating-to-facade-packages.md` (repo-external reference; see guide for details)
- Code commits:
  - `472aba7` — cloud command migration + dependency bump start
  - `1341fbb` — rmdoc command and test migration
  - `1034de5` — OCR migration + entrypoint cleanup
- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **complete**

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

## Final Validation

Completed during implementation:

- package-level validation for `cloud`
- package-level validation for `rmdoc`
- OCR command initialization/help validation
- representative `--help` smoke checks across cloud, rmdoc, and OCR commands
- repo-wide legacy symbol grep
- full-repo validation with `go test ./...`
- `docmgr doctor --ticket RMQ-0017 --stale-after 30`

## Continuation Note

There is no required follow-up for the Glazed facade migration itself.

Optional future cleanup only:

- normalize repeated cloud auth flag definitions if desired
- optionally standardize command builder style further (`WithParserConfig(...)` vs convenience helpers) now that the migration is complete

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
