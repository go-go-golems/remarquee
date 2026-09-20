---
Title: SVG image support in the Markdown to PDF pipeline
Ticket: RMQ-0024
Status: active
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
Summary: "Evidence-based design and implementation guide for SVG support in remarquee's Markdown-to-PDF pipeline, covering inline raw <svg> extraction, HTML <img> handling, converter fallback, and sizing."
LastUpdated: 2026-09-20T18:12:00-04:00
WhatFor: "Onboarding a new engineer and driving the remaining SVG implementation work in pkg/mdpdf."
WhenToUse: "Before implementing SVG support or when tracing how Markdown becomes a reMarkable PDF."
---

# SVG image support in the Markdown to PDF pipeline

## Overview

This ticket delivers a complete, intern-facing design and implementation guide for
SVG image support in remarquee's Markdown-to-PDF pipeline. The original scope
assumed XeLaTeX cannot render SVG at all; empirical testing (pandoc 3.1.3 +
rsvg-convert, documented in the guide's Appendix A) showed that **referenced SVG
files and `data:image/svg+xml` URIs already render today**, while **inline raw
`<svg>` blocks and HTML `<img src=...svg>` tags are silently dropped**, and a
missing `rsvg-convert` causes a hard XeLaTeX failure. The corrected scope is
inline extraction, HTML handling, graceful degradation, sizing policy, bundle
parity, and tests. No code has been changed yet; this ticket currently contains
the analysis/design/implementation guide plus the evidence diary.

## Key Links

- [Design & implementation guide](design-doc/01-svg-image-support-design-and-implementation-guide-for-a-new-intern.md)
- [Implementation diary](reference/01-implementation-diary.md)
- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

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
