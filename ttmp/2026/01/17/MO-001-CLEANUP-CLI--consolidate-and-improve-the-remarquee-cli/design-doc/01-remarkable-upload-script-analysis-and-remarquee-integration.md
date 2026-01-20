---
Title: Remarkable Upload Script Analysis and Remarquee Integration
Ticket: MO-001-CLEANUP-CLI
Status: active
Topics:
    - cli
    - remarquee
    - cleanup
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../.local/bin/remarkable_upload.py
      Note: Source behavior
    - Path: remarquee/cmd/remarquee/cmds/upload/md.go
      Note: Go upload command targeted for integration
    - Path: remarquee/pkg/mdpdf
      Note: Pandoc helpers and PDF conversion pipeline
    - Path: remarquee/pkg/rmcloud
      Note: rmapi library integration for uploads
ExternalSources: []
Summary: Analysis of remarkable_upload.py (pandoc+rmapi workflow) and a design for folding its behavior into remarquee's Go CLI.
LastUpdated: 2026-01-18T00:00:00-05:00
WhatFor: Guide integration of the Python upload script into remarquee's Go upload commands.
WhenToUse: Use when consolidating upload flows or deprecating remarkable_upload.py in favor of remarquee.
---


# Remarkable Upload Script Analysis and Remarquee Integration

## Executive Summary

`remarkable_upload.py` is a pragmatic, ticket-aware Markdown → PDF → upload pipeline that shells out to `pandoc` and `rmapi`. It adds two behaviors that the Go `remarquee upload` commands do not fully mirror today: (1) frontmatter stripping + list-spacing normalization before pandoc, and (2) ticket-directory inference + ticket-structure mirroring for remote paths.

This document explains how the script works, why its behaviors matter for reliability, and a concrete integration plan that ports its semantics into the Go `remarquee upload` commands while keeping the existing rmapi-library-backed upload implementation.

## Problem Statement

The upload surface is currently split:
- The repo has full Go upload commands (`remarquee upload md|bundle|src`) with rmapi library integration and advanced features (bundling, source highlighting).
- A separate Python script (`remarkable_upload.py`) is used operationally for ticket-doc uploads; it shells out to `rmapi` and applies a few Markdown pre-processing steps that reduce pandoc failures and improve list rendering.

This split causes inconsistent behavior and cognitive overhead (two tools, different flags, different preprocessing). It also risks regressions when the Python script is used as the canonical workflow but the Go CLI is evolving separately.

## Current Script: Behavior and Flow

### Inputs and defaults

- Accepts explicit Markdown files as args; if omitted:
  - With `--mirror-ticket-structure`, uploads *all* `*.md` under the ticket directory.
  - Otherwise, uploads two hard-coded docs (a bug report + analysis) within the ticket.
- Ticket discovery:
  - Default `ticket_dir` is `script_path.parent.parent` (assumes a `<ticket>/scripts/remarkable_upload.py` layout).
  - Override with `--ticket-dir`, or with `--ticket` + `--root` (searches for a directory containing `index.md`).
- Date inference:
  - Tries to infer `YYYY/MM/DD` from `.../ttmp/YYYY/MM/DD/<ticket>`.
  - Falls back to today's date.

### Remote path rules

- `--date`: overrides destination date folder under `ai/YYYY/MM/DD/`.
- If `--mirror-ticket-structure`:
  - Remote base = `ai/YYYY/MM/DD/<ticket-dir-name>/` or `--remote-ticket-root`.
  - Each file is placed under its relative subdir (mirrors local structure).
- Otherwise:
  - Remote base = `ai/YYYY/MM/DD/`.

### Markdown preprocessing (important)

- Strips YAML frontmatter delimited by `---` … `---`.
- Normalizes list spacing by inserting blank lines before list items when needed.
- Writes a temporary `.input.md` file with the modified content.
- Injects a LaTeX header file that adjusts list spacing using `enumitem` and `geometry`.

This is defensive: it avoids pandoc failures on invalid YAML and improves list rendering consistency.

### Pandoc invocation

- Uses `pandoc` with:
  - `--pdf-engine=xelatex`
  - `mainfont=DejaVu Sans` and `monofont=DejaVu Sans Mono`
  - `--standalone`
  - `--resource-path` including the markdown parent directory
  - `-H <header.tex>` (custom list/margin formatting)

### rmapi interaction

- Existence checks:
  - Attempts `rmapi ls <dir>` first (string contains check), else falls back to `rmapi get <path>`.
- Ensures remote directory exists by calling `rmapi mkdir` for each path segment.
- Uploads via `rmapi put <pdf> <remote_dir>`.
- Uses `--force` to overwrite.

### Output conventions

- Summary lines printed at start: ticket dir, remote dir, force, dry-run, pdf-only, mirror mode.
- For each file:
  - `DRY: pandoc ...` / `DRY: rmapi put ...` in dry-run mode.
  - `OK: generated ...` for PDF-only mode.
  - `OK: uploaded ...` on success.
  - `SKIP: ... already exists` when not forcing.

## Proposed Solution (Integrate into remarquee)

Port the script’s behavior into the Go CLI, so `remarquee upload` is the single surface for Markdown uploads, with parity to current script semantics.

### Core idea

- Keep rmapi library upload (rmcloud) as the *backend*.
- Bring the Python script’s preprocessing and ticket-aware pathing into Go.
- Preserve current Go features (bundle, src rendering) while adding the script’s behaviors as options for `upload md`.

### Design outline

#### 1) Markdown preprocessing parity

Add a preprocessing step in Go before `mdpdf.ConvertMarkdownFileToPDF`:
- Strip YAML frontmatter using the same delimiter logic.
- Normalize list spacing (blank line before list items).
- Generate a temporary `.input.md` when modifications are made.
- Optionally inject a LaTeX header file or move this into `mdpdf` as a default header option.

**Implementation location:**
- `remarquee/pkg/mdpdf` should grow:
  - `PreprocessMarkdown(text string) (string, modified bool)`
  - optional `PandocOptions.HeaderFile` or `LatexHeaderContent` field.

#### 2) Ticket-aware pathing & mirroring

Extend `remarquee upload md` flags:
- `--ticket-dir` (explicit ticket root)
- `--ticket` + `--root` (best-effort ticket lookup by name)
- `--mirror-ticket-structure`
- `--remote-ticket-root`

Resolve destination in a helper:
- `inferTicketDateFromPath` (mirror Python logic)
- `resolveRemoteBase(date, ticketDir, mirror, remoteRoot)`
- `relativePathWithinTicket` for per-file remote subdir

#### 3) Default file selection parity

When no explicit files are provided:
- In `--mirror-ticket-structure` mode: upload all `*.md` under the ticket dir.
- Otherwise: define explicit defaults.
  - Prefer: upload `analysis/*.md` and `reference/*.md` (more general than the script’s two hard-coded files).
  - If strict compatibility is needed, keep the exact two-file fallback, but hide behind `--legacy-defaults`.

#### 4) Existence checks

Current Go upload uses rmapi Filetree lookups; keep that, but add a fast path to avoid repeated lookups in mirror mode:
- Cache per-destination directory node (`dstNodeCache` already exists in `upload md` for preserve-dirs).
- Use `NodeByPath(docName, dstNode)` to avoid global search.

#### 5) Output contract consistency

Add a structured output option (glazed or JSON) for upload commands while preserving `OK/DRY/SKIP` human logs.
- This makes automation easier and aligns with other glazed-capable commands.

## Design Decisions

1) **Keep rmapi library integration instead of shelling out.**
   - The Go CLI already uses `rmcloud` to upload; the Python script shells out to `rmapi` for simplicity. We retain library calls to avoid shell dependencies and improve error handling.

2) **Port preprocessing into Go rather than embedding Python.**
   - The Python script is a good reference but not a long-term dependency. The Go pipeline should own the normalization logic to keep behavior consistent across platforms.

3) **Add ticket-aware flags to Go rather than replacing upload commands.**
   - `remarquee upload md` already has a robust feature set. Extending it avoids a second parallel CLI surface.

## Alternatives Considered

- **Keep the Python script as the canonical workflow.**
  Rejected: splits behavior, complicates documentation, and prevents consolidation.

- **Wrap the Python script from Go.**
  Rejected: adds a runtime dependency on Python + the script’s location; also weaker error handling and no shared code.

- **Reimplement upload only in Python and drop Go upload.**
  Rejected: Go CLI already supports bundle/src features not present in the script.

## Implementation Plan

1) **Preprocessing parity in `pkg/mdpdf`**
   - Implement frontmatter stripping and list-spacing normalization.
   - Add optional LaTeX header injection (enumitem/geometry) or embed into pandoc options.
   - Update `ConvertMarkdownFileToPDF` to use preprocessing hooks.

2) **Ticket-aware flags + path resolution**
   - Add flags to `cmd/remarquee/cmds/upload/md.go`.
   - Port ticket date inference and mirror-ticket-structure logic.
   - Add `--remote-ticket-root` for collision avoidance.

3) **Default file selection + scan**
   - Implement `--mirror-ticket-structure` to upload all `*.md` under ticket root.
   - Add a new helper for default docs (or explicit `--defaults` mode).

4) **Output normalization (optional)**
   - Add `--with-glaze-output` to upload commands (structured rows for each file).

5) **Deprecation / migration**
   - Update docs to recommend `remarquee upload md` with ticket flags.
   - Keep `remarkable_upload.py` as a reference for one release cycle, then deprecate.

## Open Questions

- Should `remarquee upload md` become ticket-aware by default (implicit ticket inference when CWD is under `ttmp/`), or only when `--ticket-dir/--ticket` is supplied?
- Do we want to keep the “two hard-coded docs” default, or replace with a more generic heuristic (all `analysis/*.md` + `reference/*.md`)?
- Should the list-normalization header be a permanent default for all uploads, or gated behind a flag (e.g., `--normalize-lists`)?

## Related Files

- `/home/manuel/.local/bin/remarkable_upload.py` (script under analysis)
- `/home/manuel/workspaces/2026-01-17/cleanup-remarquee-cli/remarquee/cmd/remarquee/cmds/upload/md.go` (Go upload entrypoint)
- `/home/manuel/workspaces/2026-01-17/cleanup-remarquee-cli/remarquee/pkg/mdpdf` (pandoc helpers, to extend)
- `/home/manuel/workspaces/2026-01-17/cleanup-remarquee-cli/remarquee/pkg/rmcloud` (rmapi library helpers)

