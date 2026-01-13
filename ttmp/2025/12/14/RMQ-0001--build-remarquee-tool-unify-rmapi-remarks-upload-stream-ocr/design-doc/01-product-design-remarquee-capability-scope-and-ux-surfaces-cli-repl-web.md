---
Title: 'Product design: remarquee capability scope and UX surfaces (CLI/REPL/Web)'
Ticket: RMQ-0001
Status: active
Topics:
    - backend
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: goMarkableStream/main.go
      Note: goMarkableStream server entrypoint (live streaming component)
    - Path: remarks/remarks/__main__.py
      Note: remarks CLI entrypoint (extraction pipeline interface to wrap)
    - Path: remarquee/remarkable_upload.py
      Note: markdown→PDF conversion + upload workflow to unify
    - Path: rmapi/api/api.go
      Note: ApiCtx interface (core rmapi abstraction remarquee should build on)
    - Path: rmapi/shell/shell.go
      Note: rmapi shell command taxonomy + REPL implementation reference
ExternalSources: []
Summary: 'Top-level product design for remarquee: capability scope, UX surfaces (CLI/REPL/Web), command taxonomy, and phased rollout plan grounded in rmapi/remarks/remarkable_upload/goMarkableStream.'
LastUpdated: 2025-12-14T19:02:40.391009112-05:00
---


# Product design: remarquee capability scope and UX surfaces (CLI/REPL/Web)

## Executive Summary

remarquee is a **single Go binary** that unifies four proven tools into one coherent product surface:

- **Cloud document management** (rmapi / Sync15): browse, upload, download, move, delete, refresh/sync.
- **Annotation extraction and conversion** (remarks): extract highlights/typed text/ink into **Markdown/PDF** and other artifacts.
- **Publishing docs to the tablet** (remarkable_upload workflow): Markdown → PDF → upload, with docmgr-friendly preprocessing and deduping.
- **Real-time tablet screen streaming** (goMarkableStream): run on-device and view in a browser; optionally integrate/control from remarquee.

We expose this through three UX surfaces:

- **CLI verbs**: stable, scriptable, composable commands.
- **REPL (interactive shell)**: fast exploration + “cloud filesystem” navigation + autocomplete, mirroring what works in `rmapi` today.
- **Optional web UI**: a thin surface for “browse + jobs + stream view” built on the same underlying command/API layer.

This document defines **what functionality we want to cover**, the **command taxonomy** that controls it, and the initial **scope boundaries** so iteration stays disciplined.

## Problem Statement

Today, the ecosystem is powerful but fragmented:

- **Four tools, four mental models**: `rmapi` (cloud tree), `remarks` (content pipeline), `remarkable_upload.py` (publishing workflow), `goMarkableStream` (live streaming).
- **Different configuration conventions**: rmapi config + tokens, Python CLIs, env-vars on device.
- **Manual glue and format shuffling**: `.rmdoc` vs xochitl layouts vs Markdown/PDF; users must remember “which tool expects what”.
- **Hard to scale to new surfaces**: adding a REPL and a web UI means redoing the same workflows and config in yet another place.

We need a single tool that is:

- **Clear**: consistent commands and nouns; predictable outputs.
- **Effective**: supports full “download → extract → publish back → present” loops.
- **Safe**: good defaults, dry-runs, explicit destructive operations.
- **Evolvable**: can start by wrapping existing tools and incrementally absorb functionality into Go over time.

## Proposed Solution

### Product shape

remarquee is a **top-level orchestrator** with pluggable “engines” underneath:

- **Cloud engine**: rmapi API (prefer library import) for cloud-side operations and filetree navigation.
- **Extract engine**: remarks (initially via subprocess wrapper) for annotation extraction and conversion.
- **Publish engine**: Markdown → PDF conversion (pandoc/xelatex pipeline initially) + upload via cloud engine.
- **Stream engine**: goMarkableStream (runs on device) with helpers for install/start/stop/tunnel/view.

The orchestrator provides:

- **Unified configuration** (single config file + env overrides).
- **Unified identity and targets** (“which account?”, “which tablet?”, “which workspace?”).
- **A stable command taxonomy** across CLI, REPL, and web UI.
- **A job layer** (optional at first) for long-running tasks: batch extract, bulk upload, OCR, streaming session management.

### Product nouns (user-facing mental model)

- **Remote (cloud) tree**: ReMarkable cloud “filesystem-like” hierarchy (folders + documents).
- **Local workspace**: a directory containing downloaded artifacts and/or docmgr tickets.
- **Document**: a cloud object representing a PDF/EPUB/notebook/collection, possibly backed by a `.rmdoc`.
- **Artifacts**: files produced by processing (e.g., extracted Markdown, annotated PDF, JSON manifests, thumbnails).
- **Jobs**: long-running operations that produce artifacts (extract, OCR, batch uploads).

### Capability buckets (what we intend to cover)

#### 1) Cloud management (rmapi-derived)

- Browse remote tree (`ls`, `cd`, `pwd`, `find`)
- CRUD for documents/folders (`get`, `put`, `mv`, `rm`, `mkdir`)
- Bulk operations (`mget`, `mput`)
- Account/session utilities (`account`, `refresh`, `stat`, `version`)

#### 2) Extraction & conversion (remarks-derived)

- Inputs: `.rmdoc`, xochitl directory, or “download then extract from cloud object”
- Outputs:
  - Obsidian-friendly **Markdown** (highlights + metadata/frontmatter)
  - Annotated **PDF**
  - Optional per-page images/SVG/JSON for downstream processing

#### 3) Publishing docs to tablet (remarkable_upload workflow)

- Inputs: Markdown (possibly with docmgr/Obsidian YAML frontmatter)
- Conversion: Markdown → PDF using a deterministic typography profile
- Upload:
  - Deduplicate by remote existence checks
  - Default foldering strategy (date/ticket-based)
  - Batch operation over a ticket directory

#### 4) Live streaming & device-side tooling (goMarkableStream-derived)

- Start/stop the streamer on the device
- Provide a “view URL” and optional tunneling story
 (Optional) expose a local reverse proxy that embeds the streamer into remarquee’s web UI

#### 5) Intelligence (future: OCR/LLM via geppetto)

- OCR handwriting and/or rendered page images
- Summarize highlights and notes; create structured outputs (e.g., “key points”, “action items”)
- Build searchable indices across extracted artifacts

### Command taxonomy (CLI verbs)

We want the top-level verbs to remain stable even if internals change.

#### Core verbs

- **`remarquee status`**: show config/account connectivity, cached state, recent jobs.
- **`remarquee config`**: print/validate config and discovered paths (tokens, rmapi config, doc roots).

#### Cloud verbs (rmapi family)

We can either expose these directly at top-level or group under `cloud`. Grouping keeps the top-level clean while still allowing an rmapi-like REPL.

- **`remarquee cloud ls [path]`**
- **`remarquee cloud cd [path]`** (REPL only; CLI uses explicit `--path`)
- **`remarquee cloud get <remote> [--out DIR]`**
- **`remarquee cloud put <local> [--to PATH]`**
- **`remarquee cloud mv <src> <dst>`**
- **`remarquee cloud rm <path>`**
- **`remarquee cloud mkdir <path>`**
- **`remarquee cloud find <query> [--path PATH]`**
- **`remarquee cloud refresh`** (refresh/sync index)
- **`remarquee cloud account`**, **`remarquee cloud version`**, **`remarquee cloud stat <path>`**

#### Extract verbs (remarks family)

- **`remarquee extract <input>`** (input can be `.rmdoc`, xochitl dir, or remote path with `--from-cloud`)
  - **`--format md|pdf|all`**
  - **`--out DIR`**
  - **`--include-drawings`**, **`--include-text`** (scoping knobs)
  - **`--from-cloud <remote>`** (download then extract)

#### Publish/upload verbs (remarkable_upload family)

- **`remarquee upload <file-or-dir>`**
  - **`--to PATH`** (default: `ai/YYYY/MM/DD/` or ticket-based)
  - **`--pdf-only`**, **`--dry-run`**, **`--force`**
  - **`--ticket RMQ-0001`** (docmgr ticket-aware batch upload)

#### Stream verbs (goMarkableStream family)

- **`remarquee stream start`** (on device)
- **`remarquee stream stop`**
- **`remarquee stream status`**
- **`remarquee stream view`** (open browser / print URL)

#### Workflow verbs (composed experiences)

- **`remarquee workflow loop <ticket-or-dir>`**:
  - Convert & upload docs
  - Later: detect updates, download annotated docs, extract, and append back into Markdown
- **`remarquee workflow backup`**:
  - Cloud sync + local archival of `.rmdoc` and derived artifacts

### REPL design (interactive shell)

We explicitly want a REPL because rmapi’s shell UX is a proven fit for cloud browsing.

- Entry: **`remarquee shell`** (or `remarquee repl`)
- Command set:
  - Cloud navigation commands mirroring rmapi’s shell: `ls`, `pwd`, `cd`, `get`, `put`, `mv`, `rm`, `mkdir`, `find`, `refresh`, `account`, `version`, `stat`.
  - remarquee-native commands: `extract`, `upload`, `stream`, `status`, `config`, `jobs`.
- Behavior:
  - Maintains a **current remote working directory** (like rmapi `ShellCtxt.path`).
  - Autocomplete for commands and remote paths.
  - Scriptability: `remarquee shell <args...>` runs a single command (rmapi supports `Process(args...)`; we should too).

### Web UI (optional)

Web UI is explicitly secondary to CLI/REPL and should only exist if it is thin and high-leverage.

Minimum useful web UI slice:

- **Document browser**: view remote tree, download artifacts, trigger extract.
- **Jobs**: queue, status, logs, retry.
- **Stream**: embed or link to live streamer view; show device connectivity.
- **Upload**: drop a Markdown/PDF, choose folder, upload.

The web UI should call the same underlying application layer as the CLI/REPL (i.e., no “second implementation” of workflows).

## Design Decisions

### 1) Stable taxonomy, modular engines

We keep top-level verbs stable (cloud/extract/upload/stream/workflow/ocr) and allow internals to change:

- Initially: wrap existing tools and libraries.
- Later: migrate features into native Go packages behind the same interfaces.

### 2) REPL mirrors rmapi shell (don’t reinvent the UX)

rmapi already provides a strong interaction model (path-aware prompt, `ls/cd/get/put/refresh`, custom completer). We adopt the same “remote tree as a shell” model so users can transfer muscle memory.

### 3) One config story

We unify:

- Account credentials / token discovery (leveraging rmapi’s established storage)
- Output directories and naming conventions
- Defaults for conversion and extraction
- Device connection details for streaming

### 4) First-class “artifacts” outputs

Every processing command writes a predictable output structure (directory + manifest). This reduces glue code, helps the web UI, and makes downstream OCR/LLM feasible.

### 5) Safe-by-default destructive operations

Operations like delete/nuke must be explicit:

- default to dry-run where it makes sense
- require confirmation flags for destructive actions in non-interactive contexts

## Alternatives Considered

### A) Keep four tools separate + write glue scripts

- Pros: minimal engineering
- Cons: doesn’t solve the core UX/config fragmentation; hard to add REPL/web UI without duplicating logic

### B) Rewrite everything in Go immediately

- Pros: “one language”, simpler distribution long-term
- Cons: high risk and time; duplicates years of reverse engineering (Sync15 and `.rm` parsing)

### C) Web UI first

- Pros: approachable for non-CLI users
- Cons: ignores the strongest existing surface (rmapi shell) and delays scriptability; likely becomes the “second implementation” problem

## Implementation Plan

This is a phased rollout plan to keep iteration tight.

### Phase 0: skeleton + config + verb scaffolding

- Create `remarquee` Go binary with cobra verbs and shared config loading.
- Implement `status` and `config` output.

### Phase 1: cloud module (rmapi-backed)

- Implement `cloud ls/get/put/mv/rm/find/refresh` via rmapi API (prefer library integration).
- Implement `shell` with rmapi-like path navigation and completion.

### Phase 2: extract module (remarks-backed)

- Implement `extract` via subprocess wrapper that shells out to remarks (initially).
- Standardize artifact output structure.

### Phase 3: upload module (publish workflow)

- Port the markdown preprocessing rules + conversion pipeline behavior (initially by invoking pandoc/xelatex).
- Integrate remote existence checks and date/ticket-based foldering.

### Phase 4: stream module (device workflows)

- Add “helpful glue” around goMarkableStream deployment and viewing.

### Phase 5: intelligence (OCR/LLM)

- Add OCR on derived page images / rendered PDFs.
- Add summarization outputs that feed back into Markdown artifacts.

### Phase 6: web UI

- Add minimal browser for jobs + stream + document tree on top of the same APIs.

## Open Questions

- **Command grouping**: do we prefer top-level rmapi-like verbs (`remarquee ls/get/put`) or a `cloud` namespace (`remarquee cloud ls/get/put`)? (REPL can still use the short names.)
- **Python bridging**: do we standardize on subprocess execution for remarks initially, or do we create a small RPC boundary for better structured results?
- **Artifact naming**: what is the canonical output folder layout for extracted artifacts (especially when driven from remote paths)?
- **Device management**: how should remarquee detect/target a tablet for streaming (SSH, mDNS, manual host config)?
- **Sync semantics**: what is “refresh” vs “sync” in remarquee terminology (rmapi’s “refresh” maps to rebuilding the filetree from cloud state)?

## References

- Ticket index: `../index.md`
- rmapi analysis: `../reference/01-rmapi-api-overview-architecture-auth-transport-shell-commands.md`
- remarks analysis: `../reference/02-remarks-package-analysis-parsing-conversion-output-formats.md`
- remarkable_upload analysis: `../reference/03-remarkable-upload-py-script-analysis-markdown-to-pdf-conversion-and-upload.md`
- goMarkableStream analysis: `../reference/04-gomarkablestream-package-analysis-screen-streaming-event-handling-websocket-api.md`
