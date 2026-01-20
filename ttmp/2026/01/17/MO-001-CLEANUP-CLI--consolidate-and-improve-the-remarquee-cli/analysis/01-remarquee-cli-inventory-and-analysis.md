---
Title: Remarquee CLI Inventory and Analysis
Ticket: MO-001-CLEANUP-CLI
Status: active
Topics:
    - cli
    - remarquee
    - cleanup
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarquee/cmd/remarquee/cmds/cloud/ls.go
      Note: Cloud command implementation and dual-mode glazed output
    - Path: remarquee/cmd/remarquee/cmds/device/serve.go
      Note: Device capture server endpoints and flags
    - Path: remarquee/cmd/remarquee/cmds/ocr/root.go
      Note: OCR command definition and geppetto integration
    - Path: remarquee/cmd/remarquee/cmds/rmdoc/render_v6.go
      Note: V6 render command flags/output (rmdoc group)
    - Path: remarquee/cmd/remarquee/cmds/upload/md.go
      Note: Upload markdown command flags and behaviors
    - Path: remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/design-doc/01-product-design-remarquee-capability-scope-and-ux-surfaces-cli-repl-web.md
      Note: Historical command taxonomy and product intent
    - Path: remarquee/ttmp/2025/12/14/RMQ-0005--remarquee-upload-next-features-bundle-toc-preserve-dirs-upload-src/design-doc/01-upload-next-features-bundle-pdfs-w-toc-mirror-dirs-syntax-highlight-source.md
      Note: Upload command origins and flags
ExternalSources: []
Summary: Full inventory of the remarquee CLI verbs, their flags/outputs, glazed usage, and historical context.
LastUpdated: 2026-01-17T17:23:52.726331222-05:00
WhatFor: Inventory and analysis to guide CLI consolidation work.
WhenToUse: Use when auditing or refactoring remarquee CLI verbs and output conventions.
---


# Remarquee CLI Inventory and Analysis

## Goal

Provide a complete inventory of the existing remarquee CLI verbs, including: glazed usage, flags, outputs, and the context in which each command likely originated. The aim is to build a shared map before consolidation work.

## Method

I treated the CLI as a small language whose verbs are defined in `cmd/remarquee/cmds` and whose grammar is declared in Cobra/glazed metadata. For historical context, I cross-referenced the design notes and prior tickets that explicitly introduced new command surfaces (notably RMQ-0001 for overall taxonomy and RMQ-0005 for upload features). The analysis is code-first and treats docs as explanatory footnotes.

## Inventory at a glance

Top-level verbs (7): `status`, `cloud`, `device`, `ocr`, `rmdsl`, `rmdoc`, `upload`.

Leaf verbs (31):
- status: 1
- cloud: 12
- device: 7
- ocr: 1
- rmdsl: 1
- rmdoc: 6
- upload: 3

Glazed usage summary:
- **Pure Cobra (no glazed)**: `status`, all `device/*`, all `upload/*`, `rmdsl compile`.
- **Glazed parameters only (human output)**: `cloud account/find/search/mv/mkdir/rm/put/get/version`, `rmdoc build-background/render-v6-png/vlm-validate`.
- **Glazed dual-mode output** (`--with-glaze-output`): `cloud ls/stat/refresh`, `rmdoc inspect/render-legacy/render-v6`.
- **Glazed writer command**: `ocr` (outputs LLM text).

High-level context (why these verbs exist):
- **Cloud verbs** align with RMQ-0001's rmapi-backed cloud management taxonomy: a filesystem-like shell of `ls/get/put/mv/rm/mkdir/find/refresh/stat/account/version`. This explains why the CLI mirrors rmapi semantics and naming. (See RMQ-0001 design doc.)
- **Upload verbs** (`upload md|bundle|src`) map directly to the RMQ-0005 design doc that added bundling, ToC, preserve-dirs, and source-code rendering. (See RMQ-0005 design doc.)
- **RMDoc + RMDSL verbs** (`rmdoc *`, `rmdsl compile`) correspond to the rendering/DSL compiler work in RMQ-0009 and related rendering investigations; they exist to debug and validate rendering pipelines with deterministic fixtures.
- **Device verbs** (`device serve/info/screenshot/raw/stream/events/gestures`) fit the "device-side capture" effort (RMQ-0011) and the "stream" surface from RMQ-0001.
- **OCR** is the geppetto-powered "intelligence" surface called out in RMQ-0001, implemented as a direct vision-model call.

## Detailed inventory

### `status`
- **Glazed**: no.
- **Purpose**: simplest possible CLI health check.
- **Flags**: none.
- **Output**: `remarquee: ok`.
- **Likely context**: first "is the binary wired" command from RMQ-0001's taxonomy.

### `cloud` (rmapi-backed remote tree)

Common auth flags (nearly all commands):
- `--non-interactive` (bool) do not prompt for one-time code
- `--reauth` (bool) force re-authentication

`cloud account`
- **Glazed**: yes (parameters only).
- **Args/flags**: auth flags above.
- **Output**: `user=<email> sync_version=<version>`.
- **User value**: confirm rmapi credentials and sync version.

`cloud refresh`
- **Glazed**: yes (dual-mode output with `--with-glaze-output`).
- **Args/flags**: auth flags above.
- **Output (human)**: `user=<email> sync=<version> hash=<hash> generation=<n>`.
- **Output (glaze)**: row with `user`, `sync_version`, `hash`, `generation`.
- **User value**: sanity check connectivity and refresh Sync15 filetree cache.

`cloud ls [path]`
- **Glazed**: yes (dual-mode output with `--with-glaze-output`).
- **Args**: `path` (default `/`).
- **Flags**: `--compact/-c`, `--long/-l`, `--reverse/-r`, `--group-directories/-d`, `--time/-t`, `--show-templates/-s`, plus auth flags.
- **Output (human)**:
  - Default: `[d]\tName` or `[f]\tName`.
  - `--compact`: `Name/` for dirs.
  - `--compact --long`: `<RFC3339> Name/`.
- **Output (glaze)**: rows with `id`, `name`, `type`, `is_dir`, `path`, `parent_id`, `version`, `modified_client`, `modified_time`.
- **User value**: rmapi-style browsing, with optional machine-readable output.

`cloud find [start] [pattern]`
- **Glazed**: yes (parameters only).
- **Args**: `start` (default `/`), `pattern` (regexp, optional).
- **Flags**: `--compact/-c`, auth flags.
- **Output**: one line per match, `[d] /path` or `/path/` in compact mode.
- **User value**: recursive listing with optional regex filter.

`cloud search <query>`
- **Glazed**: yes (parameters only).
- **Args**: `query` (required).
- **Flags**: `--start`, `--regex`, `--case-sensitive`, `--match (path|name)`, `--compact/-c`, `--include-templates`, `--type (dir|file|template)`, `--limit`, plus auth flags.
- **Output**: one line per match, same format as find.
- **User value**: controlled search by name/path with filters.

`cloud get <remote>`
- **Glazed**: yes (parameters only).
- **Args**: `remote` (required).
- **Flags**: `--out-dir` (default `.`), auth flags.
- **Output**: `OK: downloaded <remote> -> <path>`.
- **User value**: download .rmdoc archive for local inspection.

`cloud put <local> [remote-dir]`
- **Glazed**: yes (parameters only).
- **Args**: `local` (required), `remote-dir` (default `/`).
- **Flags**: `--force`, `--content-only` (PDF only), `--coverpage (0|1)`, auth flags.
- **Output**: `OK: uploaded ...` or `OK: replaced content ...`.
- **User value**: upload PDFs/epubs with rmapi semantics; content-only update for PDFs.

`cloud mv <src> <dst>`
- **Glazed**: yes (parameters only).
- **Args**: `src`, `dst`.
- **Output**: none (errors on failure).
- **User value**: rename/move entries; mirrors rmapi shell semantics.

`cloud mkdir <path>`
- **Glazed**: yes (parameters only).
- **Args**: `path`.
- **Output**: none (errors on failure).
- **User value**: create a single directory (no `-p`).

`cloud rm <target...>`
- **Glazed**: yes (parameters only).
- **Args**: `target` list (required).
- **Flags**: `--recursive/-r`, `--yes` (required for delete), auth flags.
- **Output**:
  - Without `--yes`: `would delete: ...` lines and then error.
  - With `--yes`: `deleting: ...` lines.
- **User value**: explicit, safe deletion with preview.

`cloud stat <path>`
- **Glazed**: yes (dual-mode output with `--with-glaze-output`).
- **Args**: `path` (required).
- **Output (human)**: key=value lines for path/id/type/etc.
- **Output (glaze)**: row with `path`, `name`, `id`, `type`, `is_dir`, `version`, `parent_id`, `modified_client`, `modified_time`.
- **User value**: metadata lookup; machine-readable for scripting.

`cloud version`
- **Glazed**: yes (parameters only).
- **Flags**: auth flags (present but not used).
- **Output**: `rmapi_version=<version>`.
- **User value**: confirm rmapi library version.

Context note: the `cloud` subtree is a direct translation of rmapi's filesystem-like commands into a structured CLI. RMQ-0001 explicitly calls out this taxonomy and positions it as a stable surface.

### `device` (device-side capture server and clients)

All `device/*` commands are pure Cobra and speak HTTP to a local device server.

`device serve`
- **Purpose**: run the device capture server on the tablet.
- **Flags**: `--bind`, `--username`, `--password`, `--unsafe`.
- **Output**: `listening on <addr>`.
- **Endpoints**: `/api/v1/info`, `/api/v1/screenshot.raw`, `/api/v1/screenshot.png`, `/api/v1/stream`, `/api/v1/events`, `/api/v1/gestures`.

`device info`
- **Flags**: `--url`, `--username`, `--password`, `--insecure`.
- **Output**: JSON (pretty-printed) from `/api/v1/info`.

`device screenshot`
- **Flags**: `--url`, `--username`, `--password`, `--insecure`, `--out`.
- **Output**: `OK: wrote <path>` and writes PNG to disk.

`device raw`
- **Flags**: `--url`, `--username`, `--password`, `--insecure`, `--out`.
- **Output**: `OK: wrote <path>` and writes raw framebuffer bytes to disk.

`device stream`
- **Flags**: `--url`, `--username`, `--password`, `--insecure`, `--out`, `--rate`, `--duration`.
- **Output**: `OK: wrote <path> (<bytes> bytes)` (or `... timeout`).
- **Behavior**: streams raw framebuffer bytes to a file for a duration.

`device events`
- **Flags**: `--url`, `--username`, `--password`, `--insecure`, `--out` (file or `-`), `--duration`.
- **Output**: raw SSE stream to file/stdout.

`device gestures`
- **Flags**: `--url`, `--username`, `--password`, `--insecure`, `--out` (file or `-`), `--duration`.
- **Output**: NDJSON stream to file/stdout.

Context note: these verbs correspond to the "device-side capture" surface referenced in RMQ-0001 and later device-capture investigations; they operate entirely outside rmapi, directly on the tablet.

### `ocr` (LLM vision OCR)

`ocr <image>`
- **Glazed**: yes (writer command + geppetto layers).
- **Args**: `image` (required).
- **Flags**: `--system`, `--prompt`, `--media-type`, plus geppetto provider/model flags (e.g., `--ai-engine`, `--openai-api-key`, profiles).
- **Output**: plain text from the model (last assistant text block).
- **Default behavior**: uses `gpt-4o-mini`, max tokens 4096, temperature 0, and a strict OCR system prompt.

Context note: this is the "intelligence" surface called out in RMQ-0001, implemented as a direct geppetto inference request.

### `rmdsl` (RMDoc-DSL compiler)

`rmdsl compile <case.(yaml|yml|js)>`
- **Glazed**: no.
- **Args**: `case` path (required).
- **Flags**: `--out` (required), `--doc-uuid`, `--author-id`, `--case-root`.
- **Output**: none (writes `.rmdoc`).

Context note: RMQ-0009 describes the DSL-to-.rmdoc compiler as a core artifact for generating editable fixtures and validating rendering.

### `rmdoc` (inspect + render)

`rmdoc inspect <file.rmdoc>`
- **Glazed**: yes (dual-mode output with `--with-glaze-output`).
- **Args**: `file` (required).
- **Output (human)**:
  - Header: `uuid=<uuid> schema=<schema> type=<type> pages=<n>`
  - Table: `idx`, `page_id`, `src_pdf`, `template` for each page.
- **Output (glaze)**: rows with `uuid`, `schema`, `type`, `idx`, `page_id`, `src_pdf`, `template`, `deleted`.

`rmdoc build-background <file.rmdoc>`
- **Glazed**: yes (parameters only).
- **Flags**: `--out`, `--force`.
- **Output**: `ok: wrote <pdf>`.
- **Purpose**: reconstruct background PDF in UI order for debugging.

`rmdoc render-legacy <file.rmdoc|zip>`
- **Glazed**: yes (dual-mode output with `--with-glaze-output`).
- **Flags**: `--out`, `--force`, `--add-page-numbers`, `--all-pages`, `--annotations-only`.
- **Output (human)**: `ok: wrote <pdf>`.
- **Output (glaze)**: row with `input`, `output`, `schema`, `type`.
- **Note**: legacy-only (V3/V5). Errors on V6.

`rmdoc render-v6 <file.rmdoc>`
- **Glazed**: yes (dual-mode output with `--with-glaze-output`).
- **Flags**: `--out`, `--force`.
- **Output (human)**: `ok: wrote <pdf>`.
- **Output (glaze)**: row with `input`, `output`, `schema`, `type`, `pages`.
- **Note**: cPages/V6 only.

`rmdoc render-v6-png <file.rmdoc>`
- **Glazed**: yes (parameters only).
- **Flags**: `--pages`, `--out-dir`, `--pdf-out`, `--dpi`, `--pdftoppm`, `--force`.
- **Output**: `ok: wrote <pdf>` and `ok: wrote <png>` per page.
- **Note**: uses poppler `pdftoppm` for rasterization.

`rmdoc vlm-validate` (validation helper)
- **Glazed**: yes (parameters only).
- **Inputs**: `--pdf-a` (arg), `--pdf-b`, `--image-a`, `--image-b`, `--rmdoc-a`, `--rmdoc-b` (mutually exclusive per side).
- **Flags**: `--pages`, `--out-dir`, `--rasterizer`, `--dpi`, `--pdftoppm`, `--pinocchio`, `--prompt`.
- **Output**: prints where images were written and the pinocchio command, then runs it.
- **Note**: uses poppler; unidoc is disabled due to "type check error".

Context note: `rmdoc` verbs form the internal validation toolbox for rendering correctness, driven by RMQ-0009's compiler work and the broader rendering/test harness efforts.

### `upload` (publish docs to the cloud)

`upload md <path...>`
- **Glazed**: no.
- **Purpose**: convert markdown to PDF via pandoc/xelatex and upload.
- **Flags**:
  - Auth: `--non-interactive`, `--reauth`.
  - Behavior: `--force`, `--dry-run`, `--pdf-only`, `--output-dir`, `--preserve-dirs`.
  - Destination: `--date`, `--remote-dir`.
  - Pandoc: `--pandoc`, `--pdf-engine`, `--mainfont`, `--monofont`, `--geometry`, `--latex-header-file`.
- **Output**: `DRY:` lines, `OK: generated`, `OK: uploaded`, `SKIP:` (exists).
- **Default dest**: `/ai/YYYY/MM/DD/`.

`upload bundle <path...>`
- **Glazed**: no.
- **Purpose**: bundle multiple markdown inputs into a single PDF with ToC and upload.
- **Flags**: same as `upload md`, plus `--name`, `--toc-depth`.
- **Output**: `DRY:` lines, `OK: generated`, `OK: uploaded`, `SKIP:`.

`upload src <path...>`
- **Glazed**: no.
- **Purpose**: render source files as syntax-highlighted PDFs (pandoc) and upload.
- **Flags**:
  - Auth + behavior + destination: same as above.
  - Bundle: `--bundle`, `--name`, `--toc-depth`.
  - Source rendering: `--theme`, `--listings`, `--title-mode`, `--include-ext`.
- **Output**: `DRY:` lines, `OK: generated`, `OK: uploaded`, `SKIP:`.

Context note: these three commands are a direct realization of RMQ-0005's "next features" doc (bundle + ToC, preserve dirs, and source highlighting). The flags mirror the design document almost line for line.

## Cross-cutting observations

1) **Two output philosophies** coexist.
   - Human-oriented logs (plain text, OK/DRY/SKIP prefixes) dominate in `upload` and `device`.
   - Glazed-style structured output is available only for select `cloud` and `rmdoc` commands.
   This suggests CLI consolidation should normalize when machine-readable output is expected.

2) **Glazed is used primarily for parameter parsing, not output.**
   Only a subset of commands implement `GlazeCommand`; the rest use glazed for flag/arg parsing but print ad-hoc strings. This leads to uneven UX for `--with-glaze-output` across commands.

3) **Command groups map cleanly to original tool boundaries.**
   - `cloud` mirrors rmapi.
   - `upload` mirrors remarkable_upload.py workflows + RMQ-0005 feature expansion.
   - `device` mirrors goMarkableStream-like functionality.
   - `rmdoc`/`rmdsl` are internal rendering/fixture tools.
   - `ocr` reflects the geppetto LLM layer.
   This helps explain why flag naming and output conventions vary across groups: they were composed from different tool lineages.

4) **The help system is glazed-backed but only when cgo is enabled.**
   `cmd/remarquee/help_setup.go` wires the help system into Cobra when docs load; the non-cgo build is a no-op. This may matter when consolidating docs and command help.

## References (internal)

- RMQ-0001: Product design and CLI taxonomy (overall command family layout).
- RMQ-0005: Upload feature expansion (bundle/preserve-dirs/src + pandoc flags).
- RMQ-0009: RMDoc-DSL compiler and rendering validation context.

