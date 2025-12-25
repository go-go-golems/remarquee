---
Title: Remarquee code walkthrough (inspector notes)
Ticket: 001-REVIEW-CODE
Status: active
Topics:
    - remarquee
    - go
    - architecture
    - review
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarquee/cmd/remarquee/cmds/cloud/find.go
      Note: cloud find (walk filetree + optional regexp)
    - Path: remarquee/cmd/remarquee/cmds/cloud/get.go
      Note: cloud get (download .rmdoc)
    - Path: remarquee/cmd/remarquee/cmds/cloud/ls.go
      Note: buildPathFromParents helper + ls output formatting
    - Path: remarquee/cmd/remarquee/cmds/cloud/put.go
      Note: cloud put (upload/replace content)
    - Path: remarquee/cmd/remarquee/cmds/cloud/refresh.go
      Note: cloud refresh (filetree refresh)
    - Path: remarquee/cmd/remarquee/cmds/cloud/rm.go
      Note: cloud rm (pattern delete + --yes safety)
    - Path: remarquee/cmd/remarquee/cmds/cloud/rmapi.go
      Note: AuthSettings + createApiCtx wrapper around pkg/rmcloud
    - Path: remarquee/cmd/remarquee/cmds/rmdoc/build_background.go
      Note: Referenced in background PDF notes
    - Path: remarquee/cmd/remarquee/cmds/rmdoc/inspect.go
      Note: Referenced in Glazed dual-mode command notes
    - Path: remarquee/cmd/remarquee/cmds/rmdoc/render_legacy.go
      Note: Referenced in schema gating + rmapi notes
    - Path: remarquee/cmd/remarquee/cmds/upload/bundle.go
      Note: upload bundle implementation (multi-md -> one pdf with ToC -> upload)
    - Path: remarquee/cmd/remarquee/cmds/upload/md.go
      Note: upload md implementation (markdown -> pdf -> upload)
    - Path: remarquee/cmd/remarquee/cmds/upload/root.go
      Note: Upload command group registration
    - Path: remarquee/cmd/remarquee/cmds/upload/src.go
      Note: upload src implementation (source -> highlighted pdf(s) -> upload)
    - Path: remarquee/cmd/remarquee/main.go
      Note: Referenced in CLI architecture section
    - Path: remarquee/pkg/rmdoc/content.go
      Note: Schema/type detection + page plan derivation
    - Path: remarquee/pkg/rmdoc/open.go
      Note: |-
        Core OpenFile/OpenReaderAt implementation
        Core OpenFile/OpenReaderAt implementation (rmdoc core)
    - Path: remarquee/pkg/rmdoc/pagedata.go
      Note: Template fill-in logic from .pagedata
    - Path: remarquee/pkg/rmdoc/render/background.go
      Note: Background PDF builder used by CLI+UI
    - Path: remarquee/pkg/rmdoc/rmv6_tagged_block_reader.go
      Note: V6 .rm structural reader (future decoding groundwork)
    - Path: remarquee/pkg/rmdoc/types.go
      Note: Document/PageRef core data model
ExternalSources: []
Summary: 'Inspector-style walkthrough of the `remarquee/` Go code: components, responsibilities, key symbols, and the main execution flows.'
LastUpdated: 2025-12-24T08:23:35.673252448-05:00
WhatFor: 'Understand and review `remarquee` architecture quickly: where to start, how pieces connect, and what invariants/contracts to watch.'
WhenToUse: Use when onboarding, debugging cross-package behavior, adding features, or reviewing for correctness/regressions.
---






# Remarquee code walkthrough (inspector notes)

## Executive summary

`remarquee/` is a Go module that provides a **unified toolkit for reMarkable workflows**. It has two main entrypoints:

- A **CLI** (`remarquee/cmd/remarquee`) built on [Cobra](https://github.com/spf13/cobra), with most complex subcommands implemented using [Glazed](https://github.com/go-go-golems/glazed) for parameter layers and optional structured output.
- A **developer UI server** (`remarquee/cmd/remarquee-ui`) built on `net/http`, with an embedded React/Vite frontend. It exposes JSON APIs to inspect and render `.rmdoc` artifacts and to persist validation “review sessions” into ticket folders.

The core “domain library” here is `remarquee/pkg/rmdoc`: it opens `.rmdoc` zip archives, detects schema (`legacy` vs `cPages`), and turns `.content` into a **deterministic UI page plan** (`[]PageRef`). That stable intermediate representation is what both the CLI and the UI build on.

This document is meant to onboard a new contributor quickly: where to start, how the parts fit, and which invariants matter when you modify or extend the system.

## How to read this document

If you’re new, read the next sections in order. If you’re here for a specific task, jump straight to the component or flow you care about.

- **New contributor path (recommended)**:
  - Start with **Architecture overview** (diagram + components)
  - Then read **Key end-to-end flows** (what happens when you “inspect” or “render”)
  - Then go into **Architecture walkthrough (deep dive)** sections for the specific area you’ll touch.
- **If you’re debugging**:
  - Skim **Key “watchouts” / sharp edges** inside the relevant component section, then trace the flow from entrypoint → package.
- **If you’re reviewing correctness**:
  - Focus on the schema/type detection rules in `pkg/rmdoc` and on any rmapi filetree mutation logic in `cloud`/`upload`.

## Quickstart for developers (practical)

This is intentionally “workflow-oriented”: it’s not meant to be perfect for every environment, but it should get a new developer moving fast.

### Prerequisites

- **Go**: `remarquee/go.mod` declares `go 1.24.3`.
- **For PDF generation** (upload commands): you’ll need `pandoc` and a LaTeX engine (default is `xelatex`) available on PATH.
- **For `remarquee-ui` frontend dev**: Node + npm (the UI Makefile runs `npm install` and `npm run dev`).

### Common commands (from repo root)

- **Run all Go tests for remarquee**:

```bash
cd /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee && go test ./...
```

- **Run the CLI**:

```bash
cd /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee && go run ./cmd/remarquee --help
```

- **Run the UI backend in dev mode** (serves API; frontend is separate in dev):

```bash
cd /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee-ui && go run . --dev
```

- **Run the UI frontend dev server** (from `cmd/remarquee-ui/Makefile`):

```bash
cd /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee-ui && make dev-frontend
```

### What “success” looks like

- CLI prints usage/help and `remarquee status` prints `remarquee: ok`.
- UI backend logs “expecting Vite dev server on :5173” in `--dev` mode and “server listening” on port `8080` by default.

## Architecture overview (mental model)

At a high level, you can think of the module as “two apps + a set of libraries”. The apps are thin shells that turn user intent (flags, HTTP requests) into calls to a small number of library functions.

### Component map

```text
                         +---------------------------+
                         |         remarquee         |
                         |   Go module + UI assets   |
                         +-------------+-------------+
                                       |
              +------------------------+------------------------+
              |                                                 |
     +--------v---------+                              +--------v---------+
     | CLI (Cobra)      |                              | UI server        |
     | cmd/remarquee    |                              | cmd/remarquee-ui |
     +--------+---------+                              +--------+---------+
              |                                                 |
              | command groups                                  | HTTP JSON APIs
              | (cloud / rmdoc / upload / ocr)                  | (/inspect, /render, ...)
              |                                                 |
              +-------------------+-----------------------------+
                                  |
                         +--------v---------+
                         |   pkg/* libs     |
                         | rmdoc, rmcloud,  |
                         | mdpdf, doc       |
                         +---+---+---+---+--+
                             |   |   |   |
                             |   |   |   +--> embedded CLI help docs
                             |   |   +------> pandoc/xelatex (PDF generation)
                             |   +----------> rmapi (cloud sync + upload + annotations)
                             +--------------> unipdf (background PDF assembly)
```

### Design philosophy (why it’s structured this way)

The architecture aims to keep “hard logic” in libraries and keep entrypoints shallow:

- `pkg/rmdoc` is a **stable intermediate representation**: if `Document` + `[]PageRef` are correct, multiple tools can be built on top.
- The CLI favors **good UX under partial failure** (subcommands can be registered even if initialization fails).
- The UI server exists as a **human-in-the-loop validation harness**: it makes it easy to inspect internal structure and to save review sessions to a ticket folder.

## Key end-to-end flows (the things users actually do)

These flows are the best way to understand how the pieces connect. Each flow starts from an entrypoint (CLI or UI), then converges quickly into a few core package calls.

### Flow A: Inspect a `.rmdoc` (page plan + schema/type)

This is the “hello world” for `pkg/rmdoc`. If this is wrong, many other tools become confusing because everything downstream depends on schema/type detection and page ordering.

**CLI path** (`remarquee rmdoc inspect <file.rmdoc>`):
- `cmd/remarquee/cmds/rmdoc/inspect.go`
- calls `pkg/rmdoc.OpenFile(...)`
- prints `uuid/schema/type/pages` (or emits Glaze rows)

**UI path** (`GET /api/document/{id}/inspect`):
- `cmd/remarquee-ui/api/inspect.go`
- resolves file path from `testdata/test-documents.json`
- calls `pkg/rmdoc.OpenFile(...)`
- returns JSON with `pages[]` and `hasPayloadPDF`

**Shared data flow (the common core both paths hit)**:

```text
<something>.rmdoc (zip)
  -> pkg/rmdoc.OpenFile
     -> OpenReaderAt(zip reader)
        -> read .content (required)
        -> read .metadata/.pagedata/.pdf (optional)
        -> ParseContent(.content)
           -> detect schema (cPages vs legacy)
           -> detect doc type (pdf/notebook/epub)
           -> derive []PageRef in UI order
        -> ApplyPagedataTemplates([]PageRef, .pagedata)
     -> Document{UUID, Schema, Type, Pages, ...}
  -> output (stdout rows or JSON)
```

**What can go wrong / what to watch**
- Schema detection is currently “key-based” (`"cPages"` presence), so weird `.content` JSON can produce surprising results.
- Many call sites currently use `context.Background()`; cancellation won’t propagate (not usually critical for inspect, but it sets a pattern).

### Flow B: Render a UI-ordered background PDF

This flow is a good example of how `Document.Pages` becomes a real artifact. The reMarkable UI order can differ from the source PDF order (due to inserted pages, redirection maps, etc.), so the tool assembles a new PDF in “UI page order”.

**CLI path** (`remarquee rmdoc build-background <file.rmdoc>`):
- `cmd/remarquee/cmds/rmdoc/build_background.go`
- `pkg/rmdoc.OpenFile(...)` → `pkg/rmdoc/render.BuildBackgroundPDF(...)` → `os.WriteFile(out, pdfBytes, ...)`

**UI path** (`POST /api/render/background`):
- `cmd/remarquee-ui/api/render.go`
- `pkg/rmdoc.OpenFile(...)` → `pkg/rmdoc/render.BuildBackgroundPDF(...)` → writes `outputs/*.pdf`

**What can go wrong / what to watch**
- The payload PDF may be encrypted; the builder tries to decrypt with an empty password.
- `unipdf` page objects must be duplicated before re-adding, or output can silently have fewer pages.

### Flow C: Render legacy annotations to PDF (V3/V5)

This flow is intentionally schema-gated: it only makes sense for legacy archives. It delegates most of the heavy lifting to rmapi’s `annotations` module.

- `pkg/rmdoc.OpenFile(...)` must detect `SchemaLegacy`
- legacy PDF generation happens via `rmapi/annotations.CreatePdfGenerator(...).Generate()`

### Flow D: Upload markdown/source content to the cloud

The upload commands are where `pkg/mdpdf` and `pkg/rmcloud` come together:

- Collect inputs deterministically (directories, ordering, collision detection)
- Convert to PDF:
  - preprocess markdown (`StripYAMLFrontmatter`, `NormalizeListSpacing`)
  - run `pandoc` via `exec.CommandContext`
- Upload to cloud:
  - `rmcloud.CreateApiCtx(...)`
  - ensure remote directory exists (`rmcloud.MkdirAll(...)`)
  - upload via `apiCtx.UploadDocument(...)`

### Flow E: Cloud operations (refresh, get, put, rm)

Cloud commands largely operate on rmapi’s in-memory filetree:

- bootstrap auth and API context (`rmcloud.CreateApiCtx`)
- refresh the filetree (`apiCtx.Refresh`) when needed
- resolve nodes by path or pattern (`NodeByPath`, `NodesByPath`)
- perform operations (fetch/upload/delete) and mutate the local filetree state to keep it consistent

## Glossary (terms you’ll see everywhere)

- **`.rmdoc`**: a zip archive that packages `.content`, optional `.pdf` payload, optional `.pagedata`, and per-page `.rm` annotation files.
- **`.content`**: JSON describing document/page structure. There are two main schemas:
  - **legacy** (older notebooks/PDFs)
  - **cPages** (newer “V6-ish” representation of pages with redirection and templates)
- **UI page plan**: the `[]PageRef` list in `pkg/rmdoc` describing UI order and mapping back to source PDF pages.
- **Inserted page**: a UI page that has no source PDF page; represented as `InsertedPage = -1`.
- **rmapi**: the Go library/tooling used to talk to the reMarkable cloud and to render legacy annotations.
- **Glazed**: a command framework used by many `remarquee` subcommands to support parameter layers and optional structured output.

## Where to start in the code (recommended reading order)

If you’re new and you want a “guided tour” through actual Go files, this order tends to build the right mental model quickly.

### Start with the entrypoints

- `remarquee/cmd/remarquee/main.go`
  - CLI root command wiring; shows how logging/help and command groups are assembled.
- `remarquee/cmd/remarquee-ui/main.go`
  - UI server wiring; shows the API surface and the dev/prod differences.

### Then learn the core domain model

- `remarquee/pkg/rmdoc/types.go`
  - `Document` + `PageRef` are the core “portable model” used by both CLI and UI.
- `remarquee/pkg/rmdoc/open.go`
  - `OpenFile`/`OpenReaderAt`: zip opening + blob reading + parse pipeline.
- `remarquee/pkg/rmdoc/content.go`
  - schema/type detection and how `.content` becomes a deterministic UI page plan.

### Then follow a real artifact path (PDF output)

- `remarquee/pkg/rmdoc/render/background.go`
  - background PDF assembly in UI order (and the subtle `unipdf` page duplication behavior).
- `remarquee/cmd/remarquee/cmds/rmdoc/build_background.go`
  - a thin CLI wrapper around the same render function (helps you see how it’s used).

### Finally, look at cloud + upload “integration glue”

- `remarquee/pkg/rmcloud/auth.go` and `remarquee/pkg/rmcloud/dirs.go`
  - token bootstrap + “mkdir -p” semantics against rmapi’s filetree.
- `remarquee/cmd/remarquee/cmds/cloud/refresh.go` and `remarquee/cmd/remarquee/cmds/cloud/put.go`
  - shows rmapi filetree refresh and mutation patterns.
- `remarquee/cmd/remarquee/cmds/upload/md.go`
  - shows how `pkg/mdpdf` + `rmcloud` + rmapi combine into a practical workflow.

## References and resources (for deeper learning)

These are the external dependencies that most influence behavior and debugging.

- **CLI framework**
  - [Cobra](https://github.com/spf13/cobra): command tree, flags, help UX.
  - [Glazed](https://github.com/go-go-golems/glazed): parameter layers and “dual mode” output (human + structured rows).
- **reMarkable cloud + legacy annotation rendering**
  - [rmapi](https://github.com/juruen/rmapi): cloud sync/filetree model and the legacy annotation PDF generator.
- **PDF and Markdown tooling**
  - [Pandoc](https://pandoc.org/): markdown → PDF conversion (via LaTeX).
  - [UniPDF](https://github.com/unidoc/unipdf): PDF parsing/assembly used by background PDF generation.

## Writing note (meta)

This document’s structure was updated using technical writing guidelines from:
`/home/manuel/workspaces/2025-12-21/echo-base-documentation/echo-base--openai-realtime-embedded-sdk/ttmp/2025/12/21/001-ANALYZE-ECHO-BASE--analyze-echo-base-openai-realtime-embedded-sdk/reference/03-technical-writing-guidelines.md`.

## Scope

This document is a **code-walk** of the `remarquee/` module, written like an inspector: we map **structures**, **subsystems**, and **flows** using concrete **file paths** and **symbols**.

Out of scope: detailed behavior of sibling modules in this monorepo (`rmapi/`, `remarks/`, etc.), except where `remarquee` imports them.

## High-level map (initial inventory)

- **CLI app**: `remarquee/cmd/remarquee/`
  - Cobra entrypoint and command tree for interacting with reMarkable-related workflows (cloud, rmdoc, upload, etc.).
- **UI/tooling app**: `remarquee/cmd/remarquee-ui/`
  - A Go server + embedded frontend (`frontend/`), exposing APIs to inspect/render/validate `rmdoc` artifacts.
- **Reusable packages**: `remarquee/pkg/`
  - `pkg/rmdoc/`: parsing/opening + types + rendering helpers for `.rmdoc` (and legacy zip?), with tests.
  - `pkg/rmcloud/`: cloud/auth/dirs helpers (used by CLI cloud commands).
  - `pkg/mdpdf/`: markdown → pdf pipeline (pandoc + preprocess), plus bundling helpers.
  - `pkg/doc/`: end-user docs/tutorials (not code, but “product documentation” for the CLI).

## Inspector checklist (what we’re looking for)

- **Entrypoints**: where execution starts (CLI / UI) and how flags/config are wired.
- **Boundaries**: which packages are “library-like” vs “app glue”.
- **Core domain types**: `rmdoc` document/page structures, rendering primitives, and stable interfaces.
- **Flow tracing**: parse/open → normalize → render → output (file, PDF, API response).
- **Sharp edges**: versioning (v3/v5/v6), legacy formats, and places likely to break with upstream changes.

## Architecture walkthrough (deep dive)

The sections below are the “inspector notebook” part of this document: they call out file paths and symbols and explain how each component works internally. Each component section is written to answer:

- **What is this component for?**
- **Which files/symbols define it?**
- **How does it get used (by CLI/UI/users)?**
- **What should a reviewer be nervous about?** (versioning, contracts, footguns)

### Component: CLI (`cmd/remarquee`)

The CLI is structured as a thin Cobra root command that wires in subcommand groups. Most “real work” is implemented as **Glazed commands** (from `github.com/go-go-golems/glazed`) wrapped into Cobra.

- **Entrypoint**: `remarquee/cmd/remarquee/main.go`
  - **`rootCmd`**: `var rootCmd = &cobra.Command{ ... }`
    - `PersistentPreRunE`: calls `logging.InitLoggerFromCobra(cmd)` after flags are parsed, so logging config is derived from Cobra flags.
  - **Help system integration**:
    - Builds a `helpSystem := help.NewHelpSystem()`
    - Calls `doc.AddDocToHelpSystem(helpSystem)` (from `remarquee/pkg/doc`)
    - Regardless of error, runs `help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)`
      - Design intent: docs may fail to load, but the CLI still works and `--help` stays usable.
  - **Subcommand wiring**:
    - `cmds.NewStatusCommand()` (a simple sanity-check command)
    - `cloud.NewCloudCommand()` (rmapi-backed cloud operations)
    - `ocr_cmd.NewOCRCommand()` (OCR workflow; not inspected yet)
    - `rmdoc_cmd.NewRmdocCommand()` (inspect/render `.rmdoc`)
    - `upload.NewUploadCommand()` (upload workflows)

#### Pattern: “init errors surface at runtime”

Multiple command groups follow a consistent pattern:

- The group command (`cloud`, `rmdoc`, …) attempts to build each subcommand via a `NewXxxCobraCommand() (*cobra.Command, error)`.
- If construction fails, it **still registers** a placeholder Cobra command whose `RunE` returns that init error.
- This keeps `remarquee <group> --help` functional and makes failures appear only when the user tries to execute the affected command.

You can see this pattern in:
- `remarquee/cmd/remarquee/cmds/rmdoc/root.go` (`NewRmdocCommand`)
- `remarquee/cmd/remarquee/cmds/cloud/root.go` (`NewCloudCommand`)

#### Command framework: Cobra + Glazed dual-mode output

Several subcommands are implemented as Glazed commands and then wrapped into Cobra:

- `remarquee/cmd/remarquee/cmds/rmdoc/inspect.go`
  - **Type**: `type InspectCommand struct { *glazecmds.CommandDescription }`
  - **Interfaces**:
    - `var _ glazecmds.BareCommand = &InspectCommand{}`
    - `var _ glazecmds.GlazeCommand = &InspectCommand{}`
  - **Dual mode**:
    - `Run(...)` prints human-oriented output
    - `RunIntoGlazeProcessor(...)` emits structured rows (`types.NewRow(...)`)
  - **Uses**: `pkg_rmdoc.OpenFile(ctx, s.File)` to parse a `.rmdoc` and print a deterministic “page plan”

- `remarquee/cmd/remarquee/cmds/rmdoc/render_legacy.go`
  - Validates schema/type via `pkg_rmdoc.OpenFile(...)`, then delegates PDF generation to rmapi annotations:
    - `rmapi_annotations.CreatePdfGenerator(s.File, out, opts).Generate()`
  - **Important contract**: only supports `SchemaLegacy` and rejects EPUB.

- `remarquee/cmd/remarquee/cmds/rmdoc/build_background.go`
  - Opens via `pkg_rmdoc.OpenFile(...)` and then calls:
    - `rmdocrender.BuildBackgroundPDF(..., doc, rmdocrender.BackgroundOptions{})`
  - Writes raw bytes to disk via `os.WriteFile`.

#### Notable “review watchouts” (early)

- **Context usage**: some commands use `context.Background()` instead of the passed `ctx` (`inspect.go`, `build_background.go`). If cancellation/timeouts are important (e.g. in UI/server contexts), this is a place to revisit.
- **Schema gating**: `render-legacy` relies on correct schema detection in `pkg/rmdoc.OpenFile`—any detection bug would lead to wrong behavior or confusing errors.

#### Subcommand group: `upload` (pandoc → pdf → rmapi upload workflows)

The `upload` command group is implemented as plain Cobra commands (not Glazed) and builds on:
- `pkg/mdpdf` for markdown preprocessing and pandoc invocation
- `pkg/rmcloud` for rmapi auth/bootstrap + remote directory creation
- `rmapi` for the actual upload/delete operations

Entry points:
- `remarquee/cmd/remarquee/cmds/upload/root.go`: registers the group and subcommands:
  - `upload md`
  - `upload bundle`
  - `upload src`

##### `upload md` — convert markdown files to PDFs and upload (or generate PDFs only)

`remarquee/cmd/remarquee/cmds/upload/md.go`:

- **Inputs**:
  - accepts one or more file/dir paths; directories are walked recursively for `*.md`
  - `--preserve-dirs` recreates the relative directory structure remotely (and in `--pdf-only` output)
- **Destination**:
  - default remote directory is `/ai/YYYY/MM/DD/` (today) unless overridden by:
    - `--date` (YYYY/MM/DD or YYYY-MM-DD)
    - `--remote-dir` (full override)
- **Safety**:
  - existence checks via `apiCtx.Filetree().NodeByPath(docName, dstNode)`
  - `--force` deletes the existing node via `apiCtx.DeleteEntry(...)` and updates the local filetree (`DeleteNode`)
- **Flow (upload mode)**:
  - convert markdown to PDF via `mdpdf.ConvertMarkdownFileToPDF(ctx, mdPath, outPDF, pandocOpts)`
  - ensure remote directory exists with `rmcloud.MkdirAll(apiCtx, dst)`
  - upload via `apiCtx.UploadDocument(dstNode.Id(), outPDF, true, nil)` and then `apiCtx.Filetree().AddDocument(document)`

##### `upload bundle` — bundle many markdown inputs into one PDF with ToC, then upload

`remarquee/cmd/remarquee/cmds/upload/bundle.go`:

- Collects markdown files with stable ordering:
  - explicit files preserve order
  - directory contents sorted by relative path (case-insensitive)
- Builds a combined markdown “bundle” via `mdpdf.BuildBundleMarkdown(...)` (stable `# <title>` headings + `\newpage` page breaks)
- Enables `pandocOpts.TOC = true` and sets `TOCDepth` for a clickable table of contents.

##### `upload src` — render source files as syntax-highlighted PDFs and upload

`remarquee/cmd/remarquee/cmds/upload/src.go`:

- Collects non-markdown files from file/dir inputs; optional `--include-ext` filter.
- Builds markdown wrappers:
  - per-file mode: `# <title>` + fenced code block
  - bundle mode: multiple `# <title>` sections + page breaks
  - selects the code fence length dynamically (`fenceForCode`) to avoid collisions with backticks in source.
- Pandoc highlighting knobs:
  - `--theme` → `pandocOpts.HighlightStyle`
  - `--listings` → `pandocOpts.Listings`

#### Subcommand group: `cloud` (rmapi-backed remote filetree operations)

The `cloud` command group is largely implemented as Glazed commands wrapped into Cobra, and relies on `pkg/rmcloud` for token bootstrap + API context creation.

Core wiring:
- `remarquee/cmd/remarquee/cmds/cloud/root.go`: registers subcommands and uses the “init error placeholder” pattern.
- `remarquee/cmd/remarquee/cmds/cloud/rmapi.go`: defines an `AuthSettings` struct (with Glazed tags) and `createApiCtx(...)` that calls `rmcloud.CreateApiCtx(...)`.

Representative commands (patterns repeat across the group):

- **`cloud refresh`** (`refresh.go`)
  - “connectivity smoke test”: creates `apiCtx`, then calls `apiCtx.Refresh()` to refresh the Sync15 filetree.
  - Dual-mode output (human vs Glaze rows).

- **`cloud get <remote> [--out-dir]`** (`get.go`)
  - Resolves a single node via `apiCtx.Filetree().NodeByPath(remote, nil)`
  - Downloads via `apiCtx.FetchDocument(node.Document.ID, outPath)` into `<name>.rmdoc`.

- **`cloud put <local> <remote-dir> [--force|--content-only] [--coverpage]`** (`put.go`)
  - Uploads documents using rmapi semantics:
    - `--content-only` (PDF only): replaces the file content via `apiCtx.ReplaceDocumentFile(...)`, preserving metadata.
    - `--force`: deletes existing file via `apiCtx.DeleteEntry(...)` and recreates.
  - Uses `util.DocPathToName(local)` to derive the document name.

- **`cloud find [start] [pattern]`** (`find.go`)
  - Walks the filetree recursively from a start node using `filetree.WalkTree`.
  - Optional regexp match against the formatted full path.

- **`cloud rm <targets...> --yes [--recursive]`** (`rm.go`)
  - Requires an explicit `--yes` confirmation.
  - Resolves targets via `apiCtx.Filetree().NodesByPath(...)` (supports patterns).
  - Deletes via `apiCtx.DeleteEntry(node, recursive, true)` and then updates local filetree state via `DeleteNode`.
  - Calls `apiCtx.SyncComplete()` at the end.

Shared helper:
- `buildPathFromParents(*model.Node) string` (defined in `ls.go` and used by multiple commands) formats a node’s absolute path by walking `.Parent` pointers up to root.

### Component: UI (`cmd/remarquee-ui`)

`cmd/remarquee-ui` is a small developer-facing web app: a Go HTTP server exposing JSON APIs that wrap `pkg/rmdoc`, plus a React/Vite frontend.

#### Entrypoint and modes

`remarquee/cmd/remarquee-ui/main.go`:

- **Flags**:
  - `-dev` (bool): dev mode expects a Vite dev server (logged as `:5173`) and does not mount embedded assets.
  - `-port` (string): listen port (default `"8080"`).
- **Logging**:
  - uses `zerolog` with console writer; debug level in `-dev`.
- **Key directories (current code)**:
  - `testDocsPath := "testdata"`: local test docs manifest + sample inputs.
  - `outputsDir := "outputs"`: generated PDFs are written here and served back.
  - `ticketDir := "../../ttmp/2025/12/15/RMQ-RMDOC-WEB-001--build-remarquee-ui-web-validation-tool-for-rmdoc-rendering"`
    - used by the validation endpoint to persist “review sessions” as JSON/Markdown into an existing ticket workspace.

#### Static frontend embedding

`remarquee/cmd/remarquee-ui/embed.go`:

- Embeds `frontend/dist` via `//go:embed frontend/dist`
- `GetFrontendFS()` returns `fs.Sub(frontendDist, "frontend/dist")`
- In **prod mode**, `main.go` mounts `http.FileServer(http.FS(frontendFS))` at `/`.

#### HTTP surface (routes + responsibilities)

Routes are wired in `main.go` onto a `http.ServeMux`:

- **Health**
  - `GET /api/health` ⇒ `{ "status": "ok" }`

- **Test documents**
  - `GET /api/test-documents`
    - returns `testdata/test-documents.json` (manifest the UI uses to list available test docs)

- **Document inspection**
  - `GET /api/document/{id}/inspect` ⇒ `api.HandleInspect(testDocsPath)`
    - `remarquee/cmd/remarquee-ui/api/inspect.go`
    - loads doc path via `findDocumentPath(...)`
    - opens via `rmdoc.OpenFile(context.Background(), docPath)` (note: this discards `r.Context()` cancellation today; see `analysis/02-flow-a-context-propagation-audit-context-background-usage-unification-plan.md`)
    - returns `uuid/schema/docType/pages[]/hasPayloadPDF`

  - `GET /api/document/{id}/structure` ⇒ `api.HandleInternalStructure(testDocsPath)`
    - `remarquee/cmd/remarquee-ui/api/internal_structure.go`
    - opens via `rmdoc.OpenFile(...)` to get raw `.content`/`.metadata`
    - also opens the zip directly to list:
      - `AllFiles[]` (all entry names)
      - `RMFiles[]` with (pageId, filename, size, version) detected by reading the first 43 bytes of `.rm`

- **Rendering**
  - `POST /api/render/background` ⇒ `api.HandleRenderBackground(testDocsPath, outputsDir)`
    - `remarquee/cmd/remarquee-ui/api/render.go`
    - uses `rmdoc.OpenFile(...)` then `render.BuildBackgroundPDF(...)`
    - writes `outputs/<doc>-background-<ts>.pdf`
  - `POST /api/render/legacy` ⇒ `api.HandleRenderLegacy(testDocsPath, outputsDir)`
    - schema-gates with `doc.Schema == rmdoc.SchemaLegacy`
    - delegates to `rmapi/annotations` `PdfGenerator`

- **Outputs hosting**
  - `GET|HEAD /api/outputs/{filename}` ⇒ `api.HandleOutputs(outputsDir)`
    - `remarquee/cmd/remarquee-ui/api/outputs.go`
    - simple path traversal guard (`..` and `/`)
    - serves generated PDFs with `Content-Type: application/pdf`

- **Validation session persistence**
  - `POST /api/validation` ⇒ `api.HandleValidation(ticketDir)`
    - `remarquee/cmd/remarquee-ui/api/validation.go`
    - writes a “validation session” as:
      - `ticketDir/reference/validation/<session>.json`
      - `ticketDir/reference/validation/<session>.md`

#### Notable “review watchouts” (early)

- **Hard-coded ticket path**: `ticketDir` is currently a hard-coded relative path to a specific ticket under `remarquee/ttmp/...`. If you move/rename tickets, validation persistence may silently write to the wrong place.
- **Context usage**: API handlers use `context.Background()` rather than `r.Context()`. For long-running render work, request cancellation currently won’t propagate.

### Component: `pkg/rmdoc`

`pkg/rmdoc` is the “front door” library for `.rmdoc` archives: open the zip, detect schema/type, and derive a deterministic UI page plan.

#### File format (what `.rmdoc` is)

`remarquee/pkg/rmdoc/doc.go` documents the expected archive contents:
- `<docUUID>.content` (JSON; legacy or cPages schema)
- `<docUUID>.metadata` (JSON; optional)
- `<docUUID>.pagedata` (text; template name per line; optional)
- `<docUUID>.pdf` (payload PDF; optional, present for PDF documents)
- `<docUUID>/<pageID>.rm` (annotation files)

#### Key types (the “portable model”)

Defined in `remarquee/pkg/rmdoc/types.go`:

- **`type Document struct`**
  - **`UUID string`**: derived from the `.content` filename prefix.
  - **`Schema ArchiveSchema`**: one of `SchemaLegacy`, `SchemaCPages`, `SchemaUnknown`.
  - **`Type DocumentType`**: `DocTypeNotebook`, `DocTypePDF`, `DocTypeEPUB`, `DocTypeUnknown`.
  - **Raw payload** retained for debugging/forward-compat:
    - `ContentJSON []byte`, `MetadataJSON []byte`
    - `Pagedata string`
    - `PayloadPDF []byte`
  - **`Pages []PageRef`**: deterministic UI page plan.

- **`type PageRef struct`**
  - `Index int`: UI order index.
  - `PageID string`: identifier used by `.rm` filenames.
  - `SourcePDFPage int`: 0-based payload PDF page index; or `InsertedPage` (`-1`) for inserted/blank pages.
  - `Template string`: template background name (from cPages or pagedata).
  - `Deleted bool`: currently set/used primarily during parsing (cPages deleted pages are filtered out).

#### Core API: open + parse + page plan

Implemented in `remarquee/pkg/rmdoc/open.go`:

- **`OpenFile(ctx, path) (*Document, error)`**:
  - opens a file and delegates to `OpenReaderAt`.
- **`OpenReaderAt(ctx, r io.ReaderAt, size int64) (*Document, error)`**:
  - currently ignores `ctx` (`_ = ctx // reserved for future cancellation / IO abstraction`)
  - opens the zip and reads:
    - exactly one `.content` (fails if missing or multiple via `findUniqueByExt`)
    - optional `.metadata`, `.pagedata`, `.pdf`
  - parses `.content` via `ParseContent(contentJSON)`
  - post-processes templates via `ApplyPagedataTemplates(pages, pagedata)`
  - returns a `Document` holding raw blobs + computed `Pages`.

#### Schema detection + content parsing

Implemented in `remarquee/pkg/rmdoc/content.go`:

- **`ParseContent(contentJSON)`**:
  - unmarshals into `map[string]json.RawMessage` to detect keys without committing to a full schema.
  - **Document type detection**:
    - reads legacy key `"fileType"` (lowercased); `""` ⇒ `DocTypeNotebook`, `"pdf"` ⇒ `DocTypePDF`, `"epub"` ⇒ `DocTypeEPUB`.
  - **Schema detection**:
    - if `raw["cPages"]` exists ⇒ `SchemaCPages` and `parseCPages(...)`
    - else ⇒ `SchemaLegacy` and `parseLegacyPages(...)`

- **cPages parsing (`parseCPages`)**:
  - iterates `cPages.pages[]` and:
    - filters out pages with `deleted.value == 1`
    - maps `redir.value` to `SourcePDFPage` (else `InsertedPage`)
    - maps `template.value` to `Template` (else empty)
    - reassigns `Index` as `len(out)` while appending (i.e. indexes are dense after filtering).

- **Legacy parsing (`parseLegacyPages`)**:
  - unmarshals a small envelope containing:
    - `pageCount`, `pages[]`, `redirectionPageMap[]`
  - chooses `pageCount` as the max of those lengths
  - derives:
    - `PageID`: from `pages[i]` if present; else numeric string `strconv.Itoa(i)`
    - `SourcePDFPage`: defaults to `i`, but uses `redirectionPageMap[i]` if present
    - `Template`: initially empty (later filled from `.pagedata`).

#### Pagedata/template application

Implemented in `remarquee/pkg/rmdoc/pagedata.go`:

- **`ApplyPagedataTemplates(pages, pagedata) []PageRef`**
  - splits `.pagedata` by newline, trims whitespace
  - sets `pages[i].Template` **only if** it is currently empty (so cPages templates win)
  - stops when it runs out of pagedata lines.

#### Rendering helper: build “UI-ordered” background PDF

Implemented in `remarquee/pkg/rmdoc/render/background.go`:

- **`render.BuildBackgroundPDF(ctx, doc, opts) ([]byte, error)`**
  - currently ignores `ctx` (`_ = ctx // reserved for future cancellation`)
  - uses `github.com/unidoc/unipdf/v3` to:
    - open `doc.PayloadPDF` (if present), decrypting with empty password if encrypted
    - infer a page size (prefer a referenced payload page, else fallback to payload page 1, else default constant `{445,594}`)
    - iterate `doc.Pages` in UI order:
      - if `SourcePDFPage >= 0`: copy that payload page (important detail: duplicates the `*PdfPage` before adding)
      - if `InsertedPage` or no payload: insert a blank page
  - returns PDF bytes (in-memory buffer).

#### Versioning note: v6 `.rm` groundwork

`remarquee/pkg/rmdoc/rmv6_tagged_block_reader.go` contains a minimal port of `rmscene`’s tagged-block reader for v6 `.rm` files.

- This reader focuses on **file structure** (headers, block boundaries, subblocks) and intentionally defers semantic decoding (scene tree / CRDT).
- It appears to be a foundation for later tasks (the file comments reference “RMQ-0004 task 33”).

### Component: `pkg/rmcloud` (rmapi-backed cloud helpers)

`pkg/rmcloud` is a small adapter around `github.com/juruen/rmapi` primitives, intended to be imported by CLI command implementations.

- `remarquee/pkg/rmcloud/auth.go`
  - **`type AuthSettings`**: a tag-free struct for common auth knobs (non-interactive, reauth).
  - **`CreateApiCtx(auth) (userInfo, apiCtx, error)`**:
    - uses `rmapi/api.AuthHttpCtx(reauth, nonInteractive)` for token bootstrapping
    - parses the user token (`api.ParseToken(...)`) and creates the API context (`api.CreateApiCtx(...)`)
    - wraps errors with `github.com/pkg/errors` for call-site context.

- `remarquee/pkg/rmcloud/dirs.go`
  - **`MkdirAll(apiCtx, dirPath) (*model.Node, error)`**:
    - remote equivalent of `mkdir -p`: ensures intermediate directories exist.
    - key behavior:
      - normalizes `dirPath` (`""` ⇒ `/`, enforces leading `/`, trims trailing `/` except root)
      - checks for existing node via `apiCtx.Filetree().NodeByPath(...)`
      - creates missing segments via `apiCtx.CreateDir(parentId, name, true)` and then `apiCtx.Filetree().AddDocument(doc)`
      - re-resolves via `FindByName` to get the correct node instance.

### Component: `pkg/mdpdf` (markdown → pdf pipeline)

`pkg/mdpdf` is the Markdown/PDF tooling library used by upload/bundling workflows.

- `remarquee/pkg/mdpdf/preprocess.go`
  - **`StripYAMLFrontmatter(md)`**:
    - removes a leading `---` … `---` YAML block
    - explicitly aligns with docmgr’s delimiter convention; prevents pandoc failures on invalid YAML and prevents frontmatter from appearing in PDFs.
  - **`NormalizeListSpacing(md)`**:
    - inserts blank lines before list items to increase pandoc’s likelihood of recognizing lists correctly.

- `remarquee/pkg/mdpdf/bundle.go`
  - **`BuildBundleMarkdown(inputs)`**:
    - reads multiple markdown files, strips frontmatter, normalizes list spacing
    - concatenates into a single markdown document with stable `# <title>` headings
    - inserts a LaTeX page break between documents (`\newpage`) for readability.

- `remarquee/pkg/mdpdf/pandoc.go`
  - **`ConvertMarkdownFileToPDF(ctx, mdPath, outPDF, opts)`**:
    - runs `pandoc` via `exec.CommandContext` with:
      - `--pdf-engine` (default `xelatex`)
      - fonts (`mainfont`, `monofont`)
      - geometry
      - optional `--toc` and `--toc-depth`
      - optional highlight/listings options
    - writes preprocessed markdown and (optional) a LaTeX header file into a temp directory.

### Component: `pkg/doc` (embedded help docs)

`pkg/doc` is how the CLI ships user documentation inside the binary:

- `remarquee/pkg/doc/doc.go`
  - embeds markdown under `tutorials/*.md`, `cloud/*.md`, `upload/*.md`
  - `AddDocToHelpSystem(helpSystem)` loads the embedded sections into Glazed’s help system.

