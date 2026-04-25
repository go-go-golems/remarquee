---
title: "Investigation: reMarkable Timestamp Sources and Activity Tracking"
DocType: analysis
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

# Investigation: reMarkable Timestamp Sources and Activity Tracking

**Date:** 2026-04-07
**Ticket:** RMQ-0017
**Status:** Investigation complete, design phase

## Problem Statement

Users want to answer the question: *"Which files did I edit on the reMarkable device on which day?"* — specifically for annotated PDFs stored under `/ai/YYYY/MM/DD/` in the reMarkable cloud.

This investigation maps every available timestamp source in the reMarkable ecosystem, identifies what can and cannot be determined, and proposes a solution.

---

## 1. The reMarkable Archive Structure

Every document on reMarkable is stored as a `.rmdoc` archive (ZIP format). For annotated PDFs, the structure is:

```
<uuid>.rmdoc (ZIP archive)
├── <uuid>.content     ← Page layout metadata (JSON)
├── <uuid>.metadata    ← Document-level timestamps (JSON)
├── <uuid>.pagedata    ← Template per page (text, legacy only)
├── <uuid>.pdf         ← The original PDF
└── <uuid>/
    ├── <page-id>.rm   ← Stroke/annotation data (binary)
    ├── <page-id>.rm
    └── ...
```

**Key finding:** The `.content` file for annotated PDFs uses the **legacy schema** (flat `pages` UUID array), even though the `.rm` stroke files inside use **version=6** binary format.

The **cPages schema** (with structured `value[T].Timestamp` fields) only appears on notebooks created natively on the device, not on uploaded PDFs.

---

## 2. Timestamp Sources — Complete Inventory

### 2.1 Cloud-Level Timestamps (no download required)

| Source | Command | Field | Type | Meaning |
|--------|---------|-------|------|---------|
| `cloud ls --json` | Per-entry | `modified_client` | ISO 8601 UTC | Last edit time synced to cloud |
| `cloud stat <path>` | Per-file | `modified_client` | ISO 8601 UTC | Same as above |
| `cloud stat <path>` | Per-file | `modified_local` | ISO 8601 with TZ | `modified_client` converted to local timezone |

**These are the most accessible timestamps** — available via `remarquee cloud ls` and `cloud stat` without downloading anything.

**Confirmed:** `modified_client` == `.metadata` `lastModified` (same value, synced from device).

### 2.2 Document-Level Timestamps (inside `.rmdoc` ZIP)

Found in `<uuid>.metadata` (JSON):

```json
{
    "createdTime": "0",
    "deleted": false,
    "lastModified": "1775504936250",
    "lastOpened": "1775504922838",
    "lastOpenedPage": 4,
    "metadatamodified": false,
    "modified": false,
    "new": false,
    "parent": "<uuid>",
    "pinned": false,
    "source": "",
    "synced": true,
    "type": "DocumentType",
    "version": 0,
    "visibleName": "Document Name"
}
```

| Field | Type | Meaning |
|-------|------|---------|
| `createdTime` | epoch ms | Creation time (often `"0"` for uploaded PDFs) |
| `lastModified` | epoch ms | Last edit (annotation, highlight, etc.) |
| `lastOpened` | epoch ms | Last time document was opened on device |
| `lastOpenedPage` | int | Page number user was on when they closed it |

**Example conversion:**
- `lastModified: 1775504936250` → `2026-04-06T15:48:56.250Z` → `2026-04-06 11:48:56 EDT`
- `lastOpened: 1775504922838` → `2026-04-06T15:48:42.838Z` → `2026-04-06 11:48:42 EDT`

### 2.3 Annotation Detection (inside `.rmdoc` ZIP)

**No cloud-side metadata exists** to indicate whether a file has annotations. You must download and inspect the ZIP.

Detection method: count `.rm` files in `<uuid>/` subdirectory:

```
annotated → <uuid>/<page-id>.rm files exist (one per annotated page)
read-only → no <uuid>/ directory or no .rm files
```

### 2.4 cPages "Timestamps" — NOT Wall-Clock Time

The v6 cPages `.content` format has `value[T].Timestamp` fields:

```json
{
  "cPages": {
    "pages": [
      {
        "id": "page-uuid",
        "redir": {"timestamp": "1:1", "value": 0},
        "template": {"timestamp": "1:1", "value": "Blank"}
      }
    ]
  }
}
```

Despite the field name, `"1:1"` is a **serialized CRDT ID** (`CrdtId(part1=1, part2=1)`), not a wall-clock timestamp. It's a logical clock for conflict resolution across devices.

The Go type from our codebase:

```go
// pkg/rmdoc/rmv6_crdt_sequence.go
type RMV6CrdtID struct {
    Part1 uint8
    Part2 uint64
}
```

Read from binary as: `part1` = single uint8 byte, `part2` = variable-length unsigned int (varuint).

### 2.5 Binary `.rm` v6 Stroke Files — CRDT IDs Only

The `.rm` files have header `reMarkable .lines file, version=6          ` (43 bytes), followed by tagged blocks containing scene tree items (strokes, groups, glyphs). Each item carries a `RMV6CrdtID` — again, a monotonically incrementing counter, not a timestamp.

**Bottom line:** No per-stroke or per-page wall-clock timestamps exist anywhere in the reMarkable format.

---

## 3. What We Can Determine

| Question | Answerable? | How |
|----------|------------|-----|
| When was a file last uploaded/modified? | ✅ Yes | `cloud stat` → `modified_client` |
| When was a file last opened on device? | ✅ Yes | Download → `.metadata` → `lastOpened` |
| Which page was the user reading? | ✅ Yes | Download → `.metadata` → `lastOpenedPage` |
| Did the user annotate a file? | ✅ Yes | Download → check for `.rm` files in ZIP |
| How many pages were annotated? | ✅ Yes | Download → count `.rm` files |
| When was a specific annotation made? | ❌ No | CRDT IDs only provide ordering, not wall-clock time |
| Per-page edit timestamps? | ❌ No | Not stored in any format |
| How long did the user spend on a file? | ❌ No | No start/duration data |

---

## 4. Real-World Data Sample

Scanning `/ai/2026/03/28` through `/ai/2026/04/06` revealed **141 files** across 10 days.

### Files with confirmed annotations (v6 strokes):

| File | Annotated Pages | Last Modified | Last Opened |
|------|----------------|---------------|-------------|
| CR-DSL-002 — Internal Cross-Linking Design | 6 pages | Apr 6, 11:45 EDT | Apr 6, 11:37 EDT |
| GL-009 vault-backed secrets and redaction | 3 pages | Apr 2, 20:06 EDT | Apr 2, 19:56 EDT |
| GOJA-20 Web REPL Architecture | 4 pages | Apr 3, 17:16 EDT | Apr 3, 16:52 EDT |

### Files read but not annotated:

| File | Last Modified | Last Opened | Last Page |
|------|---------------|-------------|-----------|
| GOJA-25 Code Review - REPL Architecture | Apr 6, 15:48 EDT | Apr 6, 15:48 EDT | Page 4 |
| CR-DSL-001 Design Guide | Apr 4, 17:58 EDT | Apr 4, 17:57 EDT | Page 10 |
| GL-009 vault-backed secrets | Apr 2, 20:06 EDT | Apr 2, 19:56 EDT | Page 18 |

### Files uploaded but never opened on device:

These have `lastOpened` = 0 or empty, and no `.rm` files — pure uploads.

---

## 5. Key Technical Findings

### 5.1 Legacy PDF `.content` vs v6 cPages

- **Annotated PDFs** → `.content` uses legacy schema (`pages` array, `redirectionPageMap`)
- **Device-created notebooks** → `.content` uses cPages schema (`cPages.pages` with `value[T]` wrappers)
- Both can have v6 binary `.rm` stroke files inside

### 5.2 The `.metadata` File is the Gold Standard

For activity tracking, the `.metadata` JSON inside every `.rmdoc` is the single most useful artifact:
- `lastOpened` — confirms the user actually interacted with the file on the device
- `lastModified` — tells you when the last annotation was made
- `lastOpenedPage` — shows reading progress

### 5.3 Cloud `modified_client` Matches `lastModified`

Through direct comparison of downloaded files:

```
cloud stat:  modified_client=2026-04-06T15:45:36Z
.metadata:   lastModified=1775504936250 → 2026-04-06T15:45:36Z
```

They are the same value. The cloud sync preserves the device timestamp.

### 5.4 No Per-Page Timestamps Anywhere

We examined:
- `.content` (legacy: flat array; cPages: CRDT IDs as `"part1:part2"` strings)
- `.metadata` (document-level only)
- `.rm` binary (CRDT IDs: `CrdtId(uint8, varuint)`)
- Cloud API (folder/file level only)

None store per-page or per-stroke wall-clock timestamps.
