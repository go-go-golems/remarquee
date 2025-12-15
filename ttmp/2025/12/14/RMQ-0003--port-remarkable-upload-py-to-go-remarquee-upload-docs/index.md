---
Title: Port remarkable_upload.py to Go (remarquee upload docs)
Ticket: RMQ-0003
Status: complete
Topics:
    - backend
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarquee/cmd/remarquee/cmds/cloud/mkdir.go
      Note: Existing mkdir command; upload needs a recursive mkdir helper
    - Path: remarquee/cmd/remarquee/cmds/cloud/put.go
      Note: Existing rmapi-backed upload primitive to mirror/reuse
    - Path: remarquee/cmd/remarquee/cmds/cloud/rmapi.go
      Note: Auth/apiCtx bootstrap helper to factor out for reuse
    - Path: remarquee/pkg/doc/doc.go
      Note: Embedded help doc mechanism to extend with upload docs
    - Path: remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/reference/03-remarkable-upload-py-script-analysis-markdown-to-pdf-conversion-and-upload.md
      Note: Behavioral and architectural analysis of the Python script
    - Path: remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/scripts/remarkable_upload.py
      Note: Python reference implementation we are porting
ExternalSources: []
Summary: 'Port remarkable_upload.py into remarquee as a Go command (upload md): docmgr-friendly markdown preprocessing + pandoc/xelatex PDF generation + rmapi-backed upload to /ai/YYYY/MM/DD/.'
LastUpdated: 2025-12-14T21:20:57.157487274-05:00
---




# Port remarkable_upload.py to Go (remarquee upload docs)

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

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
