---
Title: 'Remarkable files web UI: macOS 1 monochrome upload, manage, and find'
Ticket: RMQ-0025
Status: complete
Topics:
    - remarkable
    - cloud
    - upload
    - web
    - frontend
    - ui
    - react
    - go
    - remarquee
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/remarquee/cmds/serve/handlers.go
      Note: F11 Content-Disposition via mime.FormatMediaType
    - Path: repo://cmd/remarquee/cmds/serve/server_test.go
      Note: F11 attachment header parsing regression test
    - Path: repo://pkg/rmfiles/manage.go
      Note: F11 download staging path and attachment name sanitizer
    - Path: repo://pkg/rmfiles/service_test.go
      Note: F11 staging path and sanitizer regression tests
ExternalSources: []
Summary: 'Design, implementation, and PR 29 review for remarquee serve: an embedded no-build files UI backed by a serialized rmfiles service. The intern-facing review identifies correctness and security changes required before merge.'
LastUpdated: 2026-10-04T07:22:39.678261-04:00
WhatFor: Explaining, reviewing, and improving the reMarkable files web UI (Go HTTP API plus embedded vanilla JavaScript frontend).
WhenToUse: When reviewing PR 29, addressing its findings, or debugging the service and browser boundary.
---

# Remarkable files web UI: macOS 1 monochrome upload, manage, and find

## Overview

This ticket delivers a complete, intern-facing design and implementation guide
for a small local web application that lets a user **upload, manage, and quickly
find** their reMarkable cloud files. The UI is styled after classic Macintosh
System 1 (1984): black-on-white, square corners, hairline rules,
**no window chrome and no menu bar**, white text surfaces, and an accent palette used
**only as text highlights**. Fonts are modern — the classic Macintosh bitmap
typeface is explicitly not used.

Architecturally the app reuses almost all existing machinery: `pkg/rmcloud` for
auth/tree/download, `pkg/mdpdf` for Markdown→PDF, the `remarquee cloud …`
commands as the semantic reference. The implemented frontend is independent of
`cmd/remarquee-ui` and requires no Node/Dagger build. Its central backend boundary
is the **single-context, mutex-serialized service** (`pkg/rmfiles`) because
`rmapi`'s `ApiCtx` is stateful and not safe for concurrent mutation.

**Implementation submitted; review requests changes.** PR [#29](https://github.com/go-go-golems/remarquee/pull/29)
implements `pkg/rmfiles` + `cmd/remarquee/cmds/serve` with an embedded no-build
static frontend. Historical happy-path cloud smoke tests are in the diary.
The subsequent evidence-backed review identifies merge-critical recursive-delete,
browser-origin, state-integrity, resource-ownership, and download-staging gaps;
those fixes are not implemented by the review deliverable.

## Key Links

- [Design & implementation guide](design-doc/01-remarkable-files-web-ui-design-and-implementation-guide-for-a-new-intern.md)
- [Second design: move directory browser, file types and sortable columns](design-doc/02-files-ui-additions-directory-picker-column-clarity-and-sorting.md)
- [PR 29 intern-facing architecture and implementation review](analysis/01-pr-29-intern-facing-architecture-design-and-implementation-review.md)
- [Implementation diary](reference/01-implementation-diary.md)
- [P1 remediation implementation diary](reference/02-p1-remediation-implementation-diary.md)
- [Task list](tasks.md)
- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Initial design and implementation are delivered. PR review is documented;
remediation remains open in [tasks.md](tasks.md). Use the review's actual API
reference and phased remediation plan rather than the historical React/API
proposal sections in the original guide.

## Topics

- remarkable
- cloud
- upload
- web
- frontend
- ui
- react
- go
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
