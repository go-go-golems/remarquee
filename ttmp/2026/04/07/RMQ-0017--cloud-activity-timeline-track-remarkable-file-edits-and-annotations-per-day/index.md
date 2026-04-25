---
Title: 'Cloud Activity Timeline: track reMarkable file edits and annotations per day'
Ticket: RMQ-0017
Status: active
Topics:
    - remarquee
    - cloud
    - rmdoc
    - analysis
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/rmdoc/content.go
      Note: Legacy vs cPages .content parsing
    - Path: pkg/rmdoc/rmv6_crdt_sequence.go
      Note: CRDT ID type definition (Part1 uint8
    - Path: pkg/rmdoc/rmv6_scene_tree.go
      Note: V6 scene tree parser for .rm stroke files
    - Path: pkg/rmdoc/rmv6_tagged_block_values.go
      Note: LWW values with CRDT timestamps
ExternalSources: []
Summary: "Investigation of reMarkable timestamp sources and design of a `remarquee cloud activity` command to show per-day file interaction timeline (read/annotated/uploaded) from cloud data + rmdoc metadata."
LastUpdated: 2026-04-07T11:11:25.46643217-04:00
WhatFor: "Answering 'which files did I edit on the reMarkable device on which day' by combining cloud-level modified_client timestamps with per-file .metadata (lastOpened, lastModified) and .rm annotation detection."
WhenToUse: ""
---


# Cloud Activity Timeline: track reMarkable file edits and annotations per day

## Overview

**Ticket RMQ-0017** — Investigate what timestamp/edit-history information is available in the reMarkable cloud + rmdoc format, and design a `remarquee cloud activity` command to produce a per-day timeline of file interactions.

### Key Findings

1. **No per-page/per-stroke wall-clock timestamps exist** — CRDT IDs in both cPages JSON ("1:1" strings) and v6 .rm binary (CrdtId structs) are ordering counters only
2. **`modified_client` from cloud == `lastModified` from .metadata** — one reliable wall-clock timestamp per file
3. **`lastOpened` + `lastOpenedPage`** in `.metadata` tells you if/when the user opened a file and how far they read
4. **Annotation detection requires download** — must check for `.rm` files inside the rmdoc ZIP; no cloud-side metadata exists
5. **Interaction classification**: uploaded (never opened) → read (opened, no strokes) → annotated (`.rm` files present)

### Documents

| Doc | Type | Description |
|-----|------|-------------|
| [Investigation](./analysis/01-investigation-remarkable-timestamp-sources-and-activity-tracking.md) | Analysis | Complete inventory of timestamp sources, real-world data samples, what can/cannot be determined |
| [Design](./design/01-design-remarquee-cloud-activity-command.md) | Design | `remarquee cloud activity` command: interface, architecture, 3-phase implementation plan |

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- remarquee
- cloud
- rmdoc
- analysis

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
