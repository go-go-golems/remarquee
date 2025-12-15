---
Title: Port .rmdoc parsing + rendering to Go (V3/V5 + V6)
Ticket: RMQ-0004
Status: active
Topics:
    - backend
    - go
    - remarkable
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarquee/cmd/remarquee/cmds/rmdoc/render_legacy.go
      Note: Legacy PDF renderer command using rmapi annotations (commit 05c257d)
    - Path: remarquee/pkg/rmdoc/content.go
      Note: Parse .content (legacy + cPages) and build deterministic PageRef plan (commit 49acbde)
    - Path: remarquee/pkg/rmdoc/content_test.go
      Note: Unit tests for schema detection and page plan generation (commit 49acbde)
    - Path: remarquee/pkg/rmdoc/open.go
      Note: Open .rmdoc zip and extract content/metadata/pagedata/pdf (commit 49acbde)
    - Path: remarquee/pkg/rmdoc/open_integration_test.go
      Note: Integration tests for OpenFile against legacy + cPages fixtures (commit 3036c7e)
    - Path: remarquee/pkg/rmdoc/pagedata.go
      Note: Apply .pagedata templates to PageRefs (commit 49acbde)
    - Path: remarquee/pkg/rmdoc/types.go
      Note: Document/PageRef types for the rmdoc package (commit 49acbde)
    - Path: remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/analysis/01-deep-dive-rmdoc-format-container-layout-parsing-png-rendering.md
      Note: Primary deep dive on .rmdoc ZIP layout
    - Path: remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/playbook/02-intern-guide-continuing-rmdoc-format-and-algorithm-research.md
      Note: Research playbook that guided the investigation
    - Path: remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/reference/06-diary-rmdoc-format-analysis-and-go-reimplementation-prep.md
      Note: Chronological research log + key findings (dual-format requirement)
    - Path: remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/scripts/color_map.go
      Note: PenColor and RGBA mappings needed for V6 highlights
    - Path: remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/scripts/go_reimplementation_gaps.md
      Note: Gap analysis and incremental porting strategy
    - Path: rmapi/annotations/pdf.go
      Note: Underlying legacy annotations PDF generator (unipdf)
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-14T20:59:12.658149443-05:00
---





# Port .rmdoc parsing + rendering to Go (V3/V5 + V6)

## Overview

This ticket is the implementation follow-up to the deep-dive research done in RMQ-0001. The goal is to **port reMarkable `.rmdoc` parsing and annotation rendering into Go**, so remarquee can:

- download `.rmdoc` via `remarquee cloud get`, and
- produce a merged, annotated PDF (and later PNGs / extracted highlights / typed text) without relying on the Python toolchain.

**Key discovery driving scope**: documents on the tablet use **both** formats:

- **V3/V5 (legacy)**: common for older PDF-based documents
- **V6 (modern)**: common for notebooks and many newer documents

So the port must include format detection + dual pipelines (at least initially).

## Key Links

- Design doc: `design-doc/01-design-go-rmdoc-data-model-and-apis.md`
- Diary: `reference/01-diary.md`
- Prior research (RMQ-0001):
  - `../RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/analysis/01-deep-dive-rmdoc-format-container-layout-parsing-png-rendering.md`
  - `../RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/reference/06-diary-rmdoc-format-analysis-and-go-reimplementation-prep.md`
  - `../RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/scripts/go_reimplementation_gaps.md`

## Status

Current status: **active**

## Topics

- backend
- go
- remarkable

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
