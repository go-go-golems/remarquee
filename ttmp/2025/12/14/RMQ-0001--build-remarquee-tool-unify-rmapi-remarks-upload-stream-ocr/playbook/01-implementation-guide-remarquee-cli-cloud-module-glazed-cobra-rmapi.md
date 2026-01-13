---
Title: 'Implementation guide: remarquee CLI + cloud module (Glazed/Cobra + rmapi)'
Ticket: RMQ-0001
Status: active
Topics:
    - backend
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Intern-friendly implementation guide for the remarquee CLI: Glazed+Cobra wiring, one-file-per-command layout, and a detailed plan for the rmapi-backed cloud module. REPL explicitly deferred."
LastUpdated: 2025-12-14T19:08:31.815350514-05:00
---

# Implementation guide: remarquee CLI + cloud module (Glazed/Cobra + rmapi)

## Purpose

Give a new developer (including an intern) **full, end-to-end context** and **step-by-step implementation guidance** to build the **remarquee CLI surface**, with a deeper plan for the **cloud module** (rmapi-backed).

This is intentionally written as a “how to build it here” guide:

- What to read first (context)
- Which patterns we use in this repo (Glazed + Cobra + help system)
- How to shape the CLI command taxonomy (from the product design doc)
- How to implement and test the cloud commands and an interactive REPL

This guide is not an API contract; it is a practical playbook for building the CLI.

## Environment Assumptions

- **Go**: go.work pins Go to `1.24.3` (see repo root `go.work`).
- **docmgr**: ticket docs live under `remarquee/ttmp/` (already set up for RMQ-0001).
- **rmapi credentials**: you can authenticate with rmapi (tokens/config) on your machine.
  - See `rmapi/config/config.go` and RMQ-0001 `reference/01-rmapi-...md` for details.
- **Local binaries**:
  - For the cloud module, you need access to the ReMarkable Cloud via rmapi’s token mechanism.
  - For the broader tool, you may eventually need `pandoc`/`xelatex` (upload path) and Python dependencies (remarks), but those are *not required* for the first “cloud CLI” milestone.

## Orientation (read this first)

### What this project is (1-minute version)

remarquee is a planned **single Go binary** that unifies:

- rmapi (cloud operations, Sync15)
- remarks (extract/convert annotations)
- remarkable_upload workflow (publish Markdown→PDF back to tablet)
- goMarkableStream (live streaming)

The initial implementation focus for CLI is:

1. **Stable CLI verbs** and wiring (Glazed+Cobra)
2. A **cloud module** that exposes rmapi capabilities as `remarquee cloud ...`
3. REPL **later** (explicitly out of scope for the initial cloud CLI implementation)

### Key RMQ-0001 docs to read (in order)

- **Product design (what we’re building and why)**:
  - `../design-doc/01-product-design-remarquee-capability-scope-and-ux-surfaces-cli-repl-web.md`
- **rmapi deep dive (cloud API + shell command mapping)**:
  - `../reference/01-rmapi-api-overview-architecture-auth-transport-shell-commands.md`
- **remarks deep dive (extraction pipeline)**:
  - `../reference/02-remarks-package-analysis-parsing-conversion-output-formats.md`
- **remarkable_upload deep dive (publish pipeline)**:
  - `../reference/03-remarkable-upload-py-script-analysis-markdown-to-pdf-conversion-and-upload.md`
- **goMarkableStream deep dive (streaming architecture)**:
  - `../reference/04-gomarkablestream-package-analysis-screen-streaming-event-handling-websocket-api.md`

### The “how we build CLIs here” (Glazed tutorial + exemplar app)

This repo already has a strong CLI pattern:

- **Glazed tutorial**: `glazed/pkg/doc/tutorials/05-build-first-command.md`
  - Core patterns:
    - command struct embedding `*cmds.CommandDescription`
    - settings struct with `glazed.parameter` tags
    - parse flags via `parsedLayers.InitializeStruct(...)`
    - output structured data via `types.Row` + a glaze processor
    - build cobra commands via `cli.BuildCobraCommand(...)`
    - dual-mode commands (human + structured) via `cli.WithDualMode(true)`
- **Working exemplar**: `pinocchio/cmd/pinocchio/main.go`
  - Shows:
    - root command setup
    - help system wiring (`help.NewHelpSystem`, `help_cmd.SetupCobraRootCommand`)
    - logging initialization (`logging.InitLoggerFromViper`)
    - use of Clay/Viper for config

When building remarquee, prefer copying patterns from `pinocchio` over inventing new ones.

## Implementation scope for the CLI milestone

### Milestone A: “Cloud CLI MVP” (REPL later)

Deliverables:

- `remarquee --help` works and shows:
  - `remarquee status`
  - `remarquee cloud ls|get|put|mv|rm|mkdir|find|refresh|stat|version|account`
- `--with-glaze-output` on selected commands yields structured output (JSON/YAML/CSV).
- Works with existing rmapi auth/token mechanism.

Non-goals for this milestone:

- Integrate remarks (Python) into the CLI
- Implement upload pipeline
- Web UI
- REPL / interactive shell

## Decisions to lock down early (ask Manuel if unclear)

### Decision 1 (locked): Where the remarquee binary and module live

We will implement remarquee as its own Go module (submodule) and binary:

- **Go module (submodule)**: `remarquee/go.mod`
  - **Module name**: `github.com/go-go-golems/remarquee`
- **Binary entrypoint**: `remarquee/cmd/remarquee/`
  - Rename the current placeholder path `remarquee/cmd/XXX/` to `remarquee/cmd/remarquee/`
  - Rename any remaining `XXX` references accordingly

Also required:

- Add `./remarquee` to the root `go.work` `use (...)` list.

### Decision 2: `remarquee cloud ...` namespace vs flat verbs

The product design doc lists both possibilities. For implementation, we recommend:

- CLI: **namespace** (`remarquee cloud ls`) for clarity
- REPL: **later** (not part of the cloud CLI implementation ticket)

## CLI architecture (recommended)

### Package layout (concrete, one file per command)

We want **one file per command**. Use a directory-per-command-group, and a file-per-command inside it:

- `remarquee/cmd/remarquee/main.go`
  - cobra root command init + help system wiring + logging init
- `remarquee/cmd/remarquee/cmds/status.go`
  - `remarquee status`
- `remarquee/cmd/remarquee/cmds/cloud/root.go`
  - `remarquee cloud` command group
- `remarquee/cmd/remarquee/cmds/cloud/ls.go`
- `remarquee/cmd/remarquee/cmds/cloud/get.go`
- `remarquee/cmd/remarquee/cmds/cloud/put.go`
- `remarquee/cmd/remarquee/cmds/cloud/mv.go`
- `remarquee/cmd/remarquee/cmds/cloud/rm.go`
- `remarquee/cmd/remarquee/cmds/cloud/mkdir.go`
- `remarquee/cmd/remarquee/cmds/cloud/find.go`
- `remarquee/cmd/remarquee/cmds/cloud/refresh.go`
- `remarquee/cmd/remarquee/cmds/cloud/stat.go`
- `remarquee/cmd/remarquee/cmds/cloud/version.go`
- `remarquee/cmd/remarquee/cmds/cloud/account.go`

Implementation code can live in `remarquee/pkg/...` as needed, but command *entrypoints* stay one-file-per-command.

### Interfaces to keep boundaries clean

The cloud module should not let rmapi types bleed everywhere. Create a minimal interface boundary:

- `CloudClient` (our interface) wraps rmapi’s `api.ApiCtx` and adds:
  - `Resolve(path string) (*model.Node, error)`
  - `List(path string, opts ...) ([]Entry, error)`
  - `Get(remote string, dst string) error`
  - ...

Under the hood, the implementation can use rmapi `api.ApiCtx`:

- See rmapi’s interface: `rmapi/api/api.go` (`type ApiCtx interface { ... }`)
- rmapi shell command inventory (for parity): `rmapi/shell/shell.go`

## Cloud module plan (rmapi-backed) — detailed

### Goal

Implement `remarquee cloud ...` by leveraging rmapi’s proven implementation while adding:

- consistent CLI help + structured output
- consistent error messages
- predictable local output paths
- (later) shared config across modules

### Authentication + ApiCtx construction (rmapi reuse)

rmapi’s `main.go` shows the current boot sequence:

- build auth HTTP context (`api.AuthHttpCtx(...)`)
- parse user token (`api.ParseToken(...)`)
- create API context (`api.CreateApiCtx(...)`)

Relevant files:

- `rmapi/main.go`
- `rmapi/api/api.go`

Recommendation:

- Wrap this in `pkg/cloud/rmapiadapter` with a constructor:
  - `NewCloudClient(ctx context.Context, opts ...Option) (CloudClient, error)`
- Add an explicit “non-interactive” option (like rmapi’s `-ni`) so scripts never prompt.

### Remote path semantics

We need a clear convention for remote paths so both CLI and REPL behave predictably:

- Paths are **remote** by default under `remarquee cloud ...`
- Support:
  - absolute paths: `/Books/foo.pdf`
  - relative paths: `foo.pdf` resolved relative to `--cwd` (CLI) or the REPL’s current remote directory

Implementation detail:

- rmapi already has a filetree abstraction (`apiCtx.Filetree()`) with root node and path resolution logic.
- Use the filetree for:
  - browsing (`ls`)
  - resolving target directories for `put`
  - mapping user-friendly paths to document IDs

### Command mapping (first pass)

Implement commands in this order (each should have strong `--help`, sane defaults, and good errors):

1. `remarquee cloud refresh`
2. `remarquee cloud ls [path]`
3. `remarquee cloud stat <path>`
4. `remarquee cloud get <remote> --out <dir|file>`
5. `remarquee cloud put <local> --to <remote-dir>`
6. `remarquee cloud mkdir <remote-dir>`
7. `remarquee cloud mv <src> <dst>`
8. `remarquee cloud rm <path> [--recursive] [--yes]`
9. `remarquee cloud find <query> [--path <root>]`
10. `remarquee cloud account` / `version`

### Structured output (Glazed) guidance

Follow the Glazed tutorial patterns:

- Each command is a struct embedding `*cmds.CommandDescription`
- Each command defines a settings struct with `glazed.parameter` tags
- Parse via `parsedLayers.InitializeStruct(layers.DefaultSlug, &Settings{})`

Output guidance:

- For “list-like” commands (`ls`, `find`):
  - implement as `GlazeCommand` and yield rows
  - fields: `id`, `name`, `path`, `type`, `modified`, `size`, `parent_id`, etc.
- For “action-like” commands (`put`, `rm`, `mv`):
  - consider **dual-mode**:
    - default: human-friendly confirmations
    - `--with-glaze-output`: structured rows for automation

### Error handling conventions

- Wrap errors with context (prefer `github.com/pkg/errors` in this repo).
- Normalize rmapi errors into user-facing messages:
  - include which remote path was being resolved
  - include remediation hints (“run `remarquee cloud refresh`”, “check tokens”, etc.)

### Caching and refresh strategy

rmapi maintains an in-memory filetree built on startup and supports `ApiCtx.Refresh()`.

For the CLI:

- In single-shot commands, simplest is:
  - create ApiCtx
  - run operation
  - exit
- For REPL:
  - create ApiCtx once
  - keep filetree in memory
  - implement `refresh` to rebuild tree

Future (out of scope for MVP):

- local persisted cache of the filetree to speed startup

## REPL plan

REPL is explicitly **deferred**. When we get to it, `rmapi/shell/shell.go` is the primary reference for UX parity.

## Commands (recommended local workflow)

These are the “do this in order” commands an intern can use while implementing.

```bash
# 1) Read the core docs (product scope + rmapi + Glazed patterns)
sed -n '1,120p' remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/design-doc/01-product-design-remarquee-capability-scope-and-ux-surfaces-cli-repl-web.md
sed -n '1,120p' glazed/pkg/doc/tutorials/05-build-first-command.md
sed -n '1,220p' rmapi/main.go
sed -n '1,120p' rmapi/api/api.go

# 2) Use pinocchio as the reference for wiring a Glazed+Cobra app
sed -n '1,200p' pinocchio/cmd/pinocchio/main.go

# 3) Build/test cycles while implementing (example)
# go test ./... -count=1
# go run ./remarquee/cmd/remarquee --help
# go run ./remarquee/cmd/remarquee cloud ls --with-glaze-output --output json
```

## Exit Criteria

- You can run `remarquee --help` and see a coherent command tree (root + `cloud`).
- You can run at least `remarquee cloud ls` and `remarquee cloud refresh` successfully against a real account.
- `remarquee cloud ls --with-glaze-output --output json` prints machine-parseable structured output.
- REPL is not part of this milestone.

## Notes

### Security note (tokens)

Avoid printing access tokens in logs or structured output. If you need debug logging, log only:

- token expiry timestamps
- user email (if already present)
- last 4 characters of token (optional)

### Don’t reinvent the help system

The Glazed help system and “doc to help system” patterns in `pinocchio` are proven. Prefer using those directly rather than hand-maintaining large help strings in Cobra.

