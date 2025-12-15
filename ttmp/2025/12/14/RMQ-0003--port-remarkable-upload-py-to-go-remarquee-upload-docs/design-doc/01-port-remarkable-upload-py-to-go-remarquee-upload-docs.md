---
Title: Port remarkable_upload.py to Go (remarquee upload docs)
Ticket: RMQ-0003
Status: active
Topics:
    - backend
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Design proposal to port remarkable_upload.py into remarquee as a Go CLI command: Markdown preprocessing, pandoc/xelatex PDF generation, and rmapi-backed upload into /ai/YYYY/MM/DD/ with ticket-aware mirroring."
LastUpdated: 2025-12-14T20:39:43.523522929-05:00
---

# Port remarkable_upload.py to Go (remarquee upload docs)

## Executive Summary

We will port the functionality of `remarkable_upload.py` (Markdown → PDF via pandoc/xelatex, then upload to reMarkable using rmapi, organized under `ai/YYYY/MM/DD/`) into the `remarquee` Go CLI.

The proposal adds a new command group:

- `remarquee upload md`: convert one or more Markdown files to PDF (after docmgr-style frontmatter stripping + list spacing normalization), then upload the resulting PDFs to the reMarkable cloud **using rmapi as a library** (same approach as `remarquee cloud put/mkdir`).

Key design goals:
- Keep **output fidelity** identical to the Python script by continuing to use **pandoc + xelatex** (not a new renderer).
- Reuse existing rmapi-backed plumbing already present in `remarquee cloud` commands.
- Provide a **ticket-aware bulk workflow** for docmgr (`--ticket`, `--ticket-dir`, `--mirror-ticket-structure`).
- Be **safe by default**: do not overwrite existing documents unless `--force` is provided; support `--dry-run`.

## Problem Statement

Today we have a working Python utility script:
- `remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/scripts/remarkable_upload.py`

It solves an important “write path” workflow: pushing documentation (often docmgr-managed Markdown) onto a reMarkable tablet for reading/annotation.

However, the script is:
- Not integrated into the `remarquee` CLI UX (flags, auth flow, help pages, structured output conventions).
- Not easily distributable as part of the Go binary (dependency discovery, consistent error handling, shared auth).
- Duplicating cloud operations that `remarquee` already implements in Go (`cloud put`, `cloud mkdir`, etc.).

We want a first-class `remarquee` command that:
- Works with the same rmapi token handling as the cloud commands.
- Provides the same conversion behavior (including docmgr-frontmatter quirks).
- Is implemented in Go, in the same “one file per command” style used for RMQ-0002.

## Proposed Solution

### User-facing CLI

Add a new top-level command group `remarquee upload` and at least one subcommand:

- `remarquee upload md [<file1.md> <file2.md> ...]`

Core modes and flags (porting Python behavior, with small Go-idiomatic adjustments):

- **Input selection**
  - Positional args: explicit Markdown paths.
  - `--ticket <RMQ-xxxx>`: locate a ticket directory under `--root` (best-effort match, requires `index.md` like the Python script).
  - `--ticket-dir <path>`: explicit ticket directory (highest priority).
  - `--root <path>`: docs root used for `--ticket` lookup (default: `remarquee/ttmp` resolved from CWD).
  - `--mirror-ticket-structure`: upload all `*.md` under the ticket directory and mirror subdirectories on-device.
  - `--remote-ticket-root <name>`: override the on-device folder name for the ticket when mirroring (default: ticket directory name).
  - (Optional, proposed) `--include-globs` / `--exclude-globs`: refine which markdown files are included when mirroring.

- **Destination directory**
  - `--date <YYYY/MM/DD|YYYY-MM-DD>`: choose upload folder date.
  - Default: infer from ticket path `.../ttmp/YYYY/MM/DD/<ticket>`; fallback to today (same as Python).
  - Upload base: `/ai/YYYY/MM/DD/` (note leading `/` to match `remarquee cloud` path semantics).

- **Behavior**
  - `--force`: overwrite if exists (mirrors `remarquee cloud put --force` semantics).
  - `--dry-run`: print what would happen.
  - `--pdf-only`: generate PDFs but do not upload.
  - `--output-dir <path>`: where to write PDFs in `--pdf-only` mode.
  - (Optional, proposed) `--keep-going`: continue processing other files if one fails.

- **Auth (same as cloud)**
  - `--non-interactive`
  - `--reauth`

- **Pandoc/xelatex knobs (advanced)**
  - `--pandoc <path>` (default: `pandoc`)
  - `--pdf-engine <engine>` (default: `xelatex`)
  - `--mainfont <font>` (default: `DejaVu Sans`)
  - `--monofont <font>` (default: `DejaVu Sans Mono`)
  - `--geometry <spec>` (default: `margin=1in`)
  - (Optional, proposed) `--latex-header-file <path>`: allow providing a custom header instead of the built-in `enumitem` header.

### What gets uploaded (naming)

The Python script talks about uploading `<name>.pdf`, but rmapi’s naming semantics typically use the document name without the extension (see `rmapi/util.DocPathToName`). The Go port will align with existing rmapi-backed commands:
- Local file: `foo.md` → temp PDF: `foo.pdf`
- Remote document name: `foo` (stem), created by rmapi during upload.

### High-level flow

For each target Markdown file:
1. Read markdown, preprocess:
   - strip docmgr-style YAML frontmatter block (`---` … `---`) at the top
   - normalize list spacing so pandoc recognizes lists reliably
2. Convert to PDF via `pandoc` + `xelatex` with the same LaTeX header used by the Python script (`enumitem` list spacing + geometry).
3. If `--pdf-only`: write PDF to output dir and stop.
4. Else:
   - compute remote directory:
     - base: `/ai/YYYY/MM/DD/`
     - if mirroring: `/ai/YYYY/MM/DD/<ticket-root>/<relative-subdir>/`
   - ensure remote directory exists (recursive mkdir)
   - if not `--force`: check existence and skip or error (TBD; see “Design Decisions”)
   - upload PDF using rmapi library calls (same primitives used by `remarquee cloud put`)
5. Cleanup temp artifacts.

### Where the code lives

Follow the established “one file per command” approach:

- `remarquee/cmd/remarquee/cmds/upload/root.go` (new command group)
- `remarquee/cmd/remarquee/cmds/upload/md.go` (the actual implementation)

Create small internal packages so logic is testable and reusable:

- `remarquee/pkg/mdpdf`:
  - preprocessing helpers (frontmatter stripping, list normalization)
  - pandoc runner (build argv, manage temp files)
- `remarquee/pkg/rmcloud` (or similar):
  - shared rmapi bootstrap (move `createApiCtx` out of `cmd/.../cloud/rmapi.go`)
  - “mkdir -p” helper implemented via rmapi `CreateDir` and file tree resolution
  - upload helper that mirrors `cloud put` behavior

This keeps command files small and makes it easier to reuse this functionality later (e.g., “upload directory”, “upload docmgr ticket”, “upload from stdin”).

## Design Decisions

### Decision: keep pandoc/xelatex instead of a pure-Go renderer

We will keep pandoc + xelatex as external dependencies for the initial Go port.

Rationale:
- The Python script already converged on this combination for **high-quality typography** and **Unicode safety** (DejaVu fonts).
- A pure-Go Markdown → PDF renderer would be a separate project with significant fidelity risk.
- The value here is integration and automation, not reinventing typesetting.

### Decision: use rmapi as a library (not shelling out to `rmapi`)

We will reuse the same approach as `remarquee cloud put/mkdir`:
- call rmapi’s Go APIs directly (`api.AuthHttpCtx`, `api.CreateApiCtx`, `UploadDocument`, etc.).

Rationale:
- Consistent auth behavior and error handling with existing `remarquee cloud` commands.
- No need to parse `rmapi` stdout formats.
- Easier to unit-test at the “argv building” layer and to evolve toward richer behaviors.

### Decision: path semantics use leading `/` (cloud-style)

The Python script uses `ai/YYYY/MM/DD/` without a leading slash. The `remarquee cloud` commands use paths like `/Books` and `/ai/...`.

We will normalize:
- Accept both `ai/...` and `/ai/...` as inputs, but internally operate on `/...`.

Rationale:
- Avoid surprising differences between `remarquee upload` and `remarquee cloud`.

### Decision: duplicate/existence behavior

The Python script:
- checks existence and **skips** if the PDF exists unless `--force`

For the Go port we have two viable options:

- **Option A (skip)**: if remote entry exists and not `--force`, print `SKIP` and continue (Python-compatible).
- **Option B (error)**: if remote entry exists and not `--force`, return an error (matches current `cloud put` behavior).

Recommendation:
- Default to **Option A** for `upload md` to preserve the “batch upload” UX.
- Keep `cloud put` as-is (error) because it’s a lower-level primitive.
- Add `--fail-on-existing` if we want a strict mode.

### Decision: recursive mkdir

The Python script ensures intermediate directories exist.

We should implement a true recursive mkdir helper in Go (a `mkdir -p` equivalent) in `pkg/rmcloud`, because:
- `cloud mkdir` is explicitly non-recursive today.
- `upload md` needs recursive behavior to mirror ticket structures.

We should consider later whether to extend `cloud mkdir` to support `-p` as well.

## Alternatives Considered

### Alternative 1: keep Python and call it from Go

We could ship a `remarquee upload md` command that spawns the existing Python script.

Rejected for initial port because:
- Adds runtime Python dependency and packaging complexity.
- Leaves rmapi operations duplicated (Go already has cloud commands).
- Harder to integrate with `remarquee help` / embedded docs.

### Alternative 2: implement Markdown → PDF fully in Go

Rejected for initial port because:
- High effort and high fidelity risk vs a known-good pandoc pipeline.
- Hard to match LaTeX-quality typesetting quickly.

### Alternative 3: shell out to `rmapi`

Rejected because:
- Inconsistent with existing `remarquee cloud` design (already uses rmapi as a library).
- Requires parsing command output; brittle.

## Implementation Plan

### Phase 0: shared rmapi helpers (small refactor)
- Move `createApiCtx` out of `cmd/remarquee/cmds/cloud/rmapi.go` into `remarquee/pkg/rmcloud/auth.go`.
- Update cloud commands to use it (no behavior change).

### Phase 1: add `remarquee upload md` (MVP parity with Python)
- Add new command group + command file:
  - `cmd/remarquee/cmds/upload/root.go`
  - `cmd/remarquee/cmds/upload/md.go`
- Implement:
  - ticket resolution (`--ticket-dir`, `--ticket`, `--root`)
  - date inference/normalization
  - file selection (positional args or `--mirror-ticket-structure`)
  - preprocessing (frontmatter strip + list spacing normalization)
  - pandoc runner (generate temp `.input.md` and `.header.tex`, call pandoc)
  - `--pdf-only`, `--output-dir`, `--dry-run`
  - remote dir computation + recursive mkdir + upload via rmapi
  - `--force` behavior consistent with chosen “existence” decision

### Phase 2: tests
- Unit tests:
  - frontmatter stripping behavior (docmgr-delimiter-accurate)
  - list spacing normalization
  - date parsing/normalization
  - ticket date inference from path segments
  - remote dir computation for mirroring
- Integration tests (optional):
  - pandoc invocation test guarded behind env var (only run if pandoc present)

### Phase 3: docs + polish
- Add embedded help docs under `remarquee/pkg/doc/`:
  - `pkg/doc/upload/01-remarquee-upload-getting-started.md`
  - `pkg/doc/upload/02-remarquee-upload-reference.md`
- Update `pkg/doc/doc.go` embed pattern to include `upload/*.md`.

### Phase 4: extend cloud mkdir (optional)
- Add `remarquee cloud mkdir -p` by reusing the recursive mkdir helper.

## Open Questions

- Should `upload md` default to “skip existing” (Python UX) or “error on existing” (cloud put UX)? Proposed: skip by default.
- Should we also support uploading `.md` files **without pandoc** (e.g., via a built-in renderer) when pandoc/xelatex isn’t installed? Proposed: no for MVP; keep explicit dependency.
- Should we preserve the Python script’s exact remote directory formatting (`ai/.../` without leading slash), or normalize to `/ai/...`? Proposed: accept both, normalize to `/ai/...`.
- When mirroring ticket structure, should we exclude certain directories by default (`archive/`, `.meta/`, `sources/`)? Proposed: yes, exclude `archive/` and `.meta/` at minimum.
- Should the command emit structured output rows (Glazed) in addition to human output? Proposed: later; MVP can be human output consistent with `cloud put`.

## References

- RMQ-0001 script: `remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/scripts/remarkable_upload.py`
- RMQ-0001 analysis: `remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/reference/03-remarkable-upload-py-script-analysis-markdown-to-pdf-conversion-and-upload.md`
- Existing rmapi-backed upload command: `remarquee/cmd/remarquee/cmds/cloud/put.go`
- Existing mkdir command: `remarquee/cmd/remarquee/cmds/cloud/mkdir.go`
