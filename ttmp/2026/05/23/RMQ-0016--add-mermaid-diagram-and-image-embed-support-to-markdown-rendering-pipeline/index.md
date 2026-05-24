---
Title: Add Mermaid Diagram and Image Embed Support to Markdown Rendering Pipeline
Ticket: RMQ-0016
Status: active
Topics:
    - mdpdf
    - mermaid
    - image-embed
    - pandoc
    - xelatex
    - upload
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Add Mermaid diagram rendering and local image embed resolution to the pkg/mdpdf markdown-to-PDF pipeline. Design doc is intern-ready with full architecture, pseudocode, and phased implementation plan."
LastUpdated: 2026-05-23T16:02:47.667311506-04:00
WhatFor: "Implement Mermaid + image embed support in remarquee upload commands"
WhenToUse: "Read the design doc before implementing changes to pkg/mdpdf"
---

# Add Mermaid Diagram and Image Embed Support to Markdown Rendering Pipeline

## Overview

remarquee converts Markdown to PDF via pandoc/xelatex and uploads to reMarkable. Today it cannot render Mermaid diagrams (```mermaid blocks appear as plain text) and cannot resolve local image paths (images referenced in Markdown break because the preprocessed file lives in a temp directory).

This ticket adds two preprocessing steps to `pkg/mdpdf`:
1. **Image path resolution** — copy referenced images into the temp dir and rewrite paths.
2. **Mermaid rendering** — detect mermaid blocks, render via `mmdc` CLI, replace with image embeds.

Current status: **design complete, implementation not started**.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- mdpdf
- mermaid
- image-embed
- pandoc
- xelatex
- upload

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
