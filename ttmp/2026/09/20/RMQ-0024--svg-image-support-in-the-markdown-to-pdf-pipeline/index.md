---
Title: SVG image support in the Markdown to PDF pipeline
Ticket: RMQ-0024
Status: complete
Topics:
    - markdown
    - pdf
    - mdpdf
    - image-embed
    - pandoc
    - xelatex
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Implemented SVG support in the Markdown-to-PDF pipeline: inline raw <svg> extraction/conversion, HTML <img src=*.svg> rewriting, graceful converter fallback, and CLI flags. Validated on real PDFs; all tests pass."
LastUpdated: 2026-09-20T18:20:00-04:00
WhatFor: "Onboarding a new engineer and recording the implemented SVG behavior in pkg/mdpdf."
WhenToUse: "When tracing how Markdown becomes a reMarkable PDF, or reviewing the SVG implementation."
---

# SVG image support in the Markdown to PDF pipeline

## Overview

This ticket delivers a complete, intern-facing design and implementation guide for
SVG image support in remarquee's Markdown-to-PDF pipeline. The original scope
assumed XeLaTeX cannot render SVG at all; empirical testing (pandoc 3.1.3 +
rsvg-convert, documented in the guide's Appendix A) showed that **referenced SVG
files and `data:image/svg+xml` URIs already render today**, while **inline raw
`<svg>` blocks and HTML `<img src=...svg>` tags are silently dropped**, and a
missing `rsvg-convert` causes a hard XeLaTeX failure. The corrected scope was
inline extraction, HTML handling, graceful degradation, sizing policy, bundle
parity, and tests.

**Implementation is complete.** Phases 1–6 landed across commits `a06e76d`,
`2c3ac2b`, `4b6483b`, `7d9bda7`, `ebaf952` (plus docs `5d9f0b9`). New code lives in
`pkg/mdpdf/svg.go` and `cmd/remarquee/cmds/upload/svg_section.go`. Validation on
real PDFs showed inline raw SVG rendering (red pixels 0 -> 3044), referenced and
HTML SVG rendering, fenced code preserved, graceful missing-converter handling,
and working bundle prefixing. `go test ./...` passes.

## Key Links

- [Design & implementation guide](design-doc/01-svg-image-support-design-and-implementation-guide-for-a-new-intern.md)
- [Implementation diary](reference/01-implementation-diary.md)
- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **complete** — implemented, validated, and delivered.

## Topics

- markdown
- pdf
- mdpdf
- image-embed
- pandoc
- xelatex

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
