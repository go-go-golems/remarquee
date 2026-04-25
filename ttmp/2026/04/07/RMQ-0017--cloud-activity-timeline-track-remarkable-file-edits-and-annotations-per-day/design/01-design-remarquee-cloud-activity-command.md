---
title: "Design: remarquee cloud activity Command"
DocType: design-doc
Status: active
Intent: long-term
Ticket: RMQ-0017
Topics:
  - remarquee
  - cloud
  - rmdoc
  - analysis
Created: 2026-04-07
Owners: []
RelatedFiles: []
Tasks: []
---

# Design: `remarquee cloud activity` Command

## Overview

A new `remarquee cloud activity` command that scans reMarkable cloud paths, downloads `.rmdoc` files, and produces a timeline showing which files the user interacted with, when, and how (read vs annotated).

## Proposed Interface

```bash
# Basic usage: show activity for a date range
remarquee cloud activity /ai/2026/04 --since 2026-04-03

# With annotation details (downloads each file)
remarquee cloud activity /ai/2026/04 --since 2026-04-03 --with-annotations

# JSON output for further processing
remarquee cloud activity /ai/2026/04 --since 2026-04-03 --with-annotations \
  --with-glaze-output --output json

# Quick mode: cloud-only timestamps, no downloads
remarquee cloud activity /ai/2026/04 --since 2026-04-03 --cloud-only

# Specific file stat
remarquee cloud activity /ai/2026/04/06/CR-DSL-002
```

## Output Format (Text)

```
📅 2026-04-06 (Monday)
──────────────────────────────────────────────────────────
  11:37→11:45  ✏️ CR-DSL-002 — Internal Cross-Linking Design
                  6 pages annotated, last page 11
  15:48        📖 GOJA-25 Code Review - REPL Architecture
                  read to page 4
  18:15→18:26  📖 PPPP-001 Source Code
                  read, no annotations
  21:59→22:22  ✏️ PPPP-003 Paper Pro Fast E-Ink Investigation Guide
                  annotated
```

## Output Format (JSON / Glaze)

```json
{
  "date": "2026-04-06",
  "files": [
    {
      "path": "/ai/2026/04/06/CR-DSL-002/CR-DSL-002 — Internal Cross-Linking Design",
      "name": "CR-DSL-002 — Internal Cross-Linking Design",
      "last_opened": "2026-04-06T15:37:19Z",
      "last_modified": "2026-04-06T15:45:36Z",
      "last_opened_page": 11,
      "annotated": true,
      "annotated_pages": 6,
      "total_pages": 13,
      "interaction": "annotated"
    },
    {
      "path": "/ai/2026/04/06/GOJA-25-CODE-REVIEW/GOJA-25 Code Review - REPL Architecture",
      "name": "GOJA-25 Code Review - REPL Architecture",
      "last_opened": "2026-04-06T19:48:42Z",
      "last_modified": "2026-04-06T19:48:56Z",
      "last_opened_page": 4,
      "annotated": false,
      "annotated_pages": 0,
      "total_pages": 26,
      "interaction": "read"
    }
  ]
}
```

## Architecture

### Data Flow

```
cloud ls --json (per day dir)
    │
    ├── collect files with modified_client timestamps
    │
    ▼
cloud stat (per file, optional if ls already has enough)
    │
    ├── for --cloud-only: done, emit timeline
    │
    ▼
cloud get (per file, only with --with-annotations)
    │
    ├── unzip → extract .metadata
    │   ├── lastOpened (epoch ms → ISO 8601)
    │   ├── lastModified (epoch ms → ISO 8601)
    │   └── lastOpenedPage
    │
    ├── unzip → count .rm files in <uuid>/ subdir
    │   ├── annotated_pages = count of .rm files
    │   └── annotated = annotated_pages > 0
    │
    ├── cleanup temp download
    │
    ▼
emit per-day timeline
```

### Interaction Classification

| Signal | Classification |
|--------|---------------|
| `lastOpened` = 0 or empty, no `.rm` files | `uploaded` — never opened on device |
| `lastOpened` set, no `.rm` files | `read` — opened but no annotations |
| `.rm` files present | `annotated` — drew/highlighted on device |
| `lastModified` > `lastOpened` + 30s | `annotated` — edit after open suggests active work |

### Key Implementation Details

1. **Batch downloading:** Download files to a temp directory, process, then clean up. Use `--out-dir` with a temp path.

2. **ZIP inspection without full extraction:** Use Go's `archive/zip` to read just `.metadata` and enumerate `.rm` entries without extracting to disk.

3. **Rate limiting:** The rmapi-backed cloud commands are sequential. Add a small delay between downloads to avoid throttling.

4. **Caching:** For repeated runs, cache `.metadata` content keyed by cloud path + `modified_client`. If the cloud timestamp hasn't changed, skip re-download.

5. **Date range filtering:** `--since YYYY-MM-DD` filters on `modified_client` or `lastOpened` (whichever is requested). The `--until YYYY-MM-DD` flag for upper bound.

6. **Progress reporting:** For large scans (100+ files), show progress: `Scanning... 23/141 files (16 annotated)`.

## Code Location

New files under the existing remarquee codebase:

```
pkg/rmdoc/
  activity.go           ← core activity scanning logic
  activity_test.go      ← tests with fixture .rmdoc zips

cmd/
  cloud_activity.go     ← Cobra command wiring

internal/
  metadata.go           ← .metadata JSON parser (extract from zip)
```

The activity scanner reuses existing infrastructure:
- `remarquee cloud ls` → already returns JSON with `modified_client`
- `remarquee cloud get` → already downloads `.rmdoc`
- `archive/zip` → Go stdlib for ZIP inspection
- `pkg/rmdoc/content.go` → `ParseContent()` for schema detection

## Implementation Phases

### Phase 1: Cloud-only timeline (no downloads)

Just wrap `cloud ls --json` and format the output as a per-day timeline using `modified_client`.

```bash
remarquee cloud activity /ai/2026/04 --cloud-only
```

This is immediately useful — shows all files per day sorted by modification time.

**Estimated effort:** Small. Parse the JSON output from cloud ls, sort, format.

### Phase 2: Full activity with metadata (downloads each file)

Download each file, extract `.metadata`, detect annotations:

```bash
remarquee cloud activity /ai/2026/04 --with-annotations
```

Adds: `lastOpened`, `lastOpenedPage`, annotated page count, interaction classification.

**Estimated effort:** Medium. Needs download batching, ZIP inspection, temp file management.

### Phase 3: Caching and incremental mode

Cache results. On re-run, only re-download files whose `modified_client` has changed.

```bash
remarquee cloud activity /ai/2026/04 --with-annotations --cache-dir ~/.cache/remarquee-activity
```

**Estimated effort:** Medium. Needs a simple file-based cache (JSON per cloud path).

## Edge Cases

1. **Files with empty `lastOpened`:** Some files (particularly recently uploaded ones) have `lastOpened: ""` or `"0"`. Handle gracefully as "never opened."

2. **Files with 0 pages:** Some `.rmdoc` files have `pages=0` in inspect output (e.g., PPPP-004). These are likely upload artifacts — show them but don't try to compute annotation ratios.

3. **Duplicate files across days:** The same document name can appear in multiple day folders (e.g., `SQLETON-03-DUCKDB-SUPPORT` on both Apr 4 and Apr 5). These are separate cloud entries with different UUIDs.

4. **Non-ASCII filenames:** Many files use em-dashes, ampersands, etc. Ensure shell quoting is handled correctly in the cloud get/stat calls.

5. **Large directories:** `/ai/2026/03/` has 31 subdirectories with ~1050 entries total. The scanner needs to handle this volume without overwhelming the cloud API.

## Open Questions

1. **Should we also support scanning `/` (root) for non-`/ai/` paths?** The current design assumes the `/ai/YYYY/MM/DD/` structure, but a user might want to scan other folders too.

2. **Should we support `--watch` mode?** Continuous monitoring that re-scans periodically and highlights new activity since last scan.

3. **Should annotation detection be smarter?** Currently we count `.rm` files, but we could parse them to distinguish highlights from freehand drawings, count strokes, etc. The v6 parser infrastructure already exists in `pkg/rmdoc/`.

4. **Interaction with remarquee-ui?** The existing `remarquee-ui` web tool could display this activity timeline as a calendar view or daily log.
