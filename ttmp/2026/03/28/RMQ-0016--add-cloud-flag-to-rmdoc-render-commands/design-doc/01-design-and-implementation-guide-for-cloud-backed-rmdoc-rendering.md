---
Title: Design and implementation guide for cloud-backed rmdoc rendering
Ticket: RMQ-0016
Status: active
Topics:
    - cli
    - cloud
    - rendering
    - rmdoc
    - remarkable
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/remarquee/cmds/cloud/get.go
      Note: |-
        Existing cloud download behavior that should be reused instead of reimplemented
        Existing cloud download flow to reuse
    - Path: cmd/remarquee/cmds/rmdoc/render_legacy.go
      Note: |-
        Current legacy render command and rmapi-backed PDF generation
        Current legacy render command and path-oriented rmapi generator
    - Path: cmd/remarquee/cmds/rmdoc/render_v6.go
      Note: |-
        Current V6 render command and its local-path assumptions
        Current V6 render command and local-path assumption
    - Path: pkg/rmcloud/auth.go
      Note: |-
        Shared rmapi authentication bootstrap
        Shared rmapi auth bootstrap
    - Path: pkg/rmdoc/open.go
      Note: |-
        Local archive opening API and package boundary
        Local archive opening boundary
    - Path: pkg/rmdoc/render/v6_merge_background.go
      Note: |-
        V6 renderer entrypoint that currently takes a local archive path
        V6 renderer reopening archive by path
    - Path: ttmp/2025/12/14/RMQ-0004--port-rmdoc-parsing-rendering-to-go-v3-v5-v6/tasks.md
      Note: Earlier ticket already called out the need to support local input or cloud download
    - Path: ttmp/2026/01/17/MO-001-CLEANUP-CLI--consolidate-and-improve-the-remarquee-cli/analysis/01-remarquee-cli-inventory-and-analysis.md
      Note: Prior CLI inventory and command taxonomy
ExternalSources: []
Summary: Add a shared cloud input resolution layer so `rmdoc render-v6` and `rmdoc render-legacy` can accept remote cloud paths via `--cloud` and render from a temporary local download without changing the renderer package boundaries.
LastUpdated: 2026-03-28T10:35:12.568156281-04:00
WhatFor: Intern-facing design and implementation guide for adding cloud-backed input support to the render commands without mixing network concerns into `pkg/rmdoc`.
WhenToUse: Use when implementing or reviewing `--cloud` support for `rmdoc` commands, or when onboarding someone new to the remarquee render stack.
---


# Design and implementation guide for cloud-backed rmdoc rendering

## Executive Summary

Today, `remarquee rmdoc render-v6` and `remarquee rmdoc render-legacy` only accept local `.rmdoc` paths. The implementation is explicit about that: both commands declare a `file` argument, call `pkg/rmdoc.OpenFile(ctx, s.File)`, and then pass the same local path deeper into the render pipeline (`cmd/remarquee/cmds/rmdoc/render_v6.go`, `cmd/remarquee/cmds/rmdoc/render_legacy.go`). In practice, that forces users to run `remarquee cloud get ...` first, then run a second command against the downloaded file.

The right fix is not to teach `pkg/rmdoc` how to talk to the network. The right fix is to add a small command-layer input resolver that can:

1. Keep the current local-path behavior unchanged.
2. When `--cloud` is set, authenticate with rmapi through `pkg/rmcloud`.
3. Resolve a remote cloud path to a node.
4. Download that document into a temporary directory as a local `.rmdoc`.
5. Hand the resulting local path to the existing render code.
6. Remove the temporary download after the command finishes.

That design matches the existing package boundaries, works for both legacy and V6 renderers, preserves current output naming, and is easy to extend later to `inspect`, `build-background`, or `render-v6-png`.

## Problem Statement And Scope

The user-visible problem is simple:

- Current workflow:
  1. `remarquee cloud get /Books/MyDoc --out-dir /tmp`
  2. `remarquee rmdoc render-v6 /tmp/MyDoc.rmdoc`
- Desired workflow:
  1. `remarquee rmdoc render-v6 /Books/MyDoc --cloud`

The friction is especially visible because an earlier rendering ticket already captured the intended end state. RMQ-0004's task list explicitly described a user-facing command that "takes a local `.rmdoc` (or downloads via `remarquee cloud get`)" and emits a rendered PDF (`ttmp/2025/12/14/RMQ-0004--port-rmdoc-parsing-rendering-to-go-v3-v5-v6/tasks.md:87-90`).

### In scope

- Add `--cloud` support to:
  - `remarquee rmdoc render-v6`
  - `remarquee rmdoc render-legacy`
- Reuse the existing rmapi authentication flow.
- Reuse the existing cloud download semantics.
- Keep renderer packages local-file-oriented.
- Preserve current local-path behavior and default output naming.
- Document the design and a clean implementation path.

### Out of scope

- Streaming render input directly from the cloud without a temp file.
- Adding cloud support to every `rmdoc` command in this ticket.
- Redesigning the entire CLI taxonomy in this ticket.
- Refactoring V6 or legacy rendering fidelity.
- Adding a new remote document cache.

## Current-State Architecture

This section is written for an intern who has never seen the codebase before.

### 1. CLI entrypoints

The root CLI wires seven top-level verbs: `status`, `cloud`, `device`, `ocr`, `rmdsl`, `rmdoc`, and `upload` (`cmd/remarquee/main.go:17-37`). The `rmdoc` subtree currently registers six leaf commands: `inspect`, `build-background`, `render-legacy`, `render-v6`, `render-v6-png`, and `vlm-validate` (`cmd/remarquee/cmds/rmdoc/root.go:5-88`).

That means the feature we are adding is not a standalone top-level concern. It is a refinement of the `rmdoc` subcommands.

### 2. The local-file assumption is hard-coded today

Each relevant command takes a `file` argument and describes it as a local path:

- `render-v6`: "Path to the V6 .rmdoc file" (`cmd/remarquee/cmds/rmdoc/render_v6.go:73-79`)
- `render-legacy`: "Path to the legacy .rmdoc or .zip file" (`cmd/remarquee/cmds/rmdoc/render_legacy.go:91-97`)
- `inspect`: "Path to the .rmdoc file" (`cmd/remarquee/cmds/rmdoc/inspect.go:50-57`)
- `build-background`: "Path to the .rmdoc file" (`cmd/remarquee/cmds/rmdoc/build_background.go:64-70`)
- `render-v6-png`: "Path to the V6 .rmdoc file" (`cmd/remarquee/cmds/rmdoc/render_v6_png.go:63-70`)

The commands then immediately open the path locally with `pkg/rmdoc.OpenFile(ctx, s.File)`:

- `render-v6.go:93`
- `render-v6.go:141`
- `render-legacy.go:111`
- `render-legacy.go:161`
- `inspect.go:71`
- `inspect.go:94`
- `build_background.go:84`
- `render_v6_png.go:130`

This is the main seam we need to change.

### 3. `pkg/rmdoc` is intentionally a local archive package

The `pkg/rmdoc` package models an opened `.rmdoc` archive as:

- `Document{UUID, Schema, Type, ContentJSON, MetadataJSON, Pagedata, PayloadPDF, Pages}`
- `PageRef{Index, PageID, SourcePDFPage, Template, Deleted}`

See `pkg/rmdoc/types.go:52-86`.

`pkg/rmdoc.OpenFile` opens a local file path with `os.Open`, stats it, and passes the file handle into `OpenReaderAt` (`pkg/rmdoc/open.go:15-28`). `OpenReaderAt` then:

1. Creates a ZIP reader.
2. Reads `.content`, optional `.metadata`, optional `.pagedata`, and optional `.pdf`.
3. Parses schema and document type through `ParseContent`.
4. Applies `.pagedata` templates.
5. Returns a `Document`.

See `pkg/rmdoc/open.go:30-109` and `pkg/rmdoc/content.go:11-35`.

This package has no cloud authentication, no rmapi dependency, and no notion of remote paths. That is a good boundary and should remain intact.

### 4. The render pipelines still depend on archive paths

The render commands are not merely parsing an archive once and carrying the entire document in memory.

#### V6 path

`render-v6` validates the schema with `pkg_rmdoc.OpenFile`, computes a default output path from the input basename, and then calls `rmdocrender.MergeRMDocV6OntoBackgroundPDFWithInfo(ctx, s.File, ...)` (`cmd/remarquee/cmds/rmdoc/render_v6.go:93-127`).

Inside `MergeRMDocV6OntoBackgroundPDFWithInfo`, the renderer again opens the archive by path (`pkg/rmdoc/render/v6_merge_background.go:272-281`). This matters: the renderer is built around reopening and traversing archive contents from a filesystem path, not around receiving a pre-loaded `Document` plus all `.rm` payload bytes.

#### Legacy path

`render-legacy` validates the schema locally, then constructs `rmapi_annotations.CreatePdfGenerator(s.File, out, opts)` and runs `g.Generate()` (`cmd/remarquee/cmds/rmdoc/render_legacy.go:136-145`).

That legacy generator also wants a file path, not raw archive bytes.

### 5. Cloud access already exists

The repository already has a cloud command surface. `cloud get` is especially important here:

1. It creates an rmapi API context via `createApiCtx`.
2. Resolves a remote node with `apiCtx.Filetree().NodeByPath(s.Remote, nil)`.
3. Computes an output filename from the remote node name.
4. Downloads the document using `apiCtx.FetchDocument(node.Document.ID, outPath)`.

See `cmd/remarquee/cmds/cloud/get.go:86-108`.

The common authentication bootstrap already lives below the command layer in `pkg/rmcloud.CreateApiCtx`, which retries auth, handles expired tokens, and returns `api.ApiCtx` (`pkg/rmcloud/auth.go:20-52`).

### 6. Historical product intent

The existing ticket history already points in this direction:

- RMQ-0004 states that remarquee should be able to download `.rmdoc` via `remarquee cloud get` and then render it to an annotated PDF (`ttmp/2025/12/14/RMQ-0004--port-rmdoc-parsing-rendering-to-go-v3-v5-v6/index.md:56-66`).
- The same ticket's task list explicitly leaves room for commands that either take a local `.rmdoc` or download via `cloud get` (`ttmp/2025/12/14/RMQ-0004--port-rmdoc-parsing-rendering-to-go-v3-v5-v6/tasks.md:87-90`).
- RMQ-0002 records the introduction of rmapi-backed `cloud get` as a first-class verb, so we already have the raw capability (`ttmp/2025/12/14/RMQ-0002--implement-remarquee-cloud-cli/changelog.md:79-86`).

## Gap Analysis

The current system has four gaps relevant to this feature.

### Gap 1: The render UX is two-step when it should be one-step

Users must manually:

1. Resolve a cloud path.
2. Download the document.
3. Remember where it landed.
4. Re-run a second command against the downloaded path.

That is unnecessary operator overhead for a common workflow.

### Gap 2: The cloud/download logic is not reusable outside `cloud get`

Today the download logic is embedded in `cmd/remarquee/cmds/cloud/get.go`. The `rmdoc` commands cannot reuse `cmd/...` packages cleanly because that would invert package boundaries and couple one command tree to another.

### Gap 3: The render commands duplicate their execution logic

Both `render-v6` and `render-legacy` effectively duplicate:

- parse settings
- open file and validate schema
- compute default output path
- enforce `--force`
- render
- print or emit structured output

You can see the duplication in:

- `render_v6.go:87-129` and `render_v6.go:131-183`
- `render_legacy.go:105-149` and `render_legacy.go:151-204`

If we add cloud resolution naively, we will duplicate that too.

### Gap 4: There is no common "input source" abstraction

The CLI can currently accept:

- local file path for `rmdoc` commands
- remote cloud path for `cloud get`
- `.rmdoc` path or PDF/PNG path for `vlm-validate`

But there is no shared type that says, "this command resolved user input into a local working file plus cleanup behavior." That is exactly what this feature needs.

## Proposed Solution

### High-level idea

Add a shared resolver that turns a user-supplied `file` argument into a local working `.rmdoc` path.

There are two modes:

1. Local mode:
   - default
   - `file` is already a local filesystem path
   - no network access
   - no cleanup needed

2. Cloud mode:
   - activated by `--cloud`
   - `file` is treated as a remote reMarkable cloud path
   - authenticate through `pkg/rmcloud`
   - resolve the remote node
   - download to a temp directory
   - return the downloaded local path plus a cleanup function

The renderer remains oblivious to the origin of the input.

### CLI contract

Add these flags to both render commands:

```text
--cloud            Treat <file> as a remote reMarkable cloud path
--non-interactive  Do not prompt for one-time code; fail if tokens are missing
--reauth           Force re-authentication (re-fetch user token)
```

Examples:

```bash
remarquee rmdoc render-v6 /Books/MyDoc --cloud
remarquee rmdoc render-v6 /Books/MyDoc --cloud --non-interactive --out /tmp/mydoc.pdf
remarquee rmdoc render-legacy /Archive/OldPDF --cloud --reauth
```

### Package boundary decision

Put the cloud download primitive in `pkg/rmcloud`, not in `pkg/rmdoc`, and not as a cross-import from `cmd/remarquee/cmds/cloud`.

Why:

- `pkg/rmdoc` should stay focused on archive parsing/rendering, not auth/network.
- `cmd/...` packages should not become shared libraries for other commands.
- `pkg/rmcloud` already owns rmapi bootstrap and remote-directory helpers.

### Recommended new pieces

#### A. `pkg/rmcloud` download helper

Add a new helper file, for example:

```text
pkg/rmcloud/download.go
```

Recommended API:

```go
type DownloadedDocument struct {
    RemotePath string
    LocalPath  string
    NodeID     string
    Name       string
}

func DownloadDocumentByPath(
    ctx context.Context,
    auth AuthSettings,
    remotePath string,
    outDir string,
) (*DownloadedDocument, error)
```

Responsibilities:

- authenticate with `CreateApiCtx`
- resolve node by remote path
- reject directories
- compute stable local `.rmdoc` filename
- call `FetchDocument`
- return metadata the caller may want to log or expose

This should internally absorb the important logic that currently lives only in `cloud get`.

#### B. `rmdoc` command-side input resolver

Add a small command helper, for example:

```text
cmd/remarquee/cmds/rmdoc/input_resolver.go
```

Recommended API:

```go
type CloudInputSettings struct {
    Cloud          bool `glazed.parameter:"cloud"`
    NonInteractive bool `glazed.parameter:"non-interactive"`
    Reauth         bool `glazed.parameter:"reauth"`
}

type ResolvedRMDocInput struct {
    RequestedPath string
    LocalPath     string
    Source        string // "local" or "cloud"
    RemotePath    string
    Cleanup       func() error
}

func ResolveRMDocInput(
    ctx context.Context,
    file string,
    cloudSettings CloudInputSettings,
) (*ResolvedRMDocInput, error)
```

Responsibilities:

- trim and validate the user input
- in local mode, return `LocalPath=file`
- in cloud mode:
  - create temp dir
  - call `rmcloud.DownloadDocumentByPath`
  - return local path and cleanup callback

This is the right place for temp-dir lifecycle management because it is a command concern, not a transport concern.

#### C. Shared execution helpers inside each render command

Refactor each render command so `Run` and `RunIntoGlazeProcessor` share a single execution helper.

Example:

```go
type renderV6Result struct {
    InputSource string
    InputPath   string
    OutputPath  string
    Schema      string
    Type        string
    Pages       int
}

func (c *RenderV6Command) execute(ctx context.Context, s *RenderV6Settings) (*renderV6Result, error)
```

Why this matters:

- `--cloud` logic should only be implemented once.
- output naming should only be implemented once.
- schema validation should only be implemented once.
- structured output should consume the same execution result as human output.

## Detailed Flow

### Runtime flow for `render-v6 --cloud`

```text
User
  |
  | remarquee rmdoc render-v6 /Books/MyDoc --cloud
  v
RenderV6Command
  |
  | parse flags/settings
  v
ResolveRMDocInput
  |
  | if --cloud:
  |   rmcloud.CreateApiCtx
  |   Filetree().NodeByPath("/Books/MyDoc")
  |   FetchDocument(node.ID, /tmp/.../MyDoc.rmdoc)
  v
local temp .rmdoc path
  |
  | pkg/rmdoc.OpenFile(localPath)
  | validate schema == cPages
  | default out = MyDoc-v6.pdf
  v
MergeRMDocV6OntoBackgroundPDFWithInfo(localPath)
  |
  | pkg/rmdoc.OpenFile(localPath)
  | BuildBackgroundPDF(...)
  | merge strokes/highlights/text
  v
write output PDF
  |
  | cleanup temp dir
  v
user-visible PDF in cwd or --out path
```

### Runtime flow for `render-legacy --cloud`

```text
User
  |
  | remarquee rmdoc render-legacy /Archive/OldDoc --cloud
  v
RenderLegacyCommand
  |
  | ResolveRMDocInput(...)
  v
local temp .rmdoc path
  |
  | pkg/rmdoc.OpenFile(localPath)
  | validate schema == legacy
  v
rmapi_annotations.CreatePdfGenerator(localPath, out, opts)
  |
  | Generate()
  v
write output PDF
  |
  | cleanup temp dir
  v
done
```

## API Sketches

### Proposed settings changes

#### `render-v6`

```go
type RenderV6Settings struct {
    File string `glazed.parameter:"file"`
    Out  string `glazed.parameter:"out"`

    Force bool `glazed.parameter:"force"`

    CloudInputSettings
}
```

#### `render-legacy`

```go
type RenderLegacySettings struct {
    File string `glazed.parameter:"file"`
    Out  string `glazed.parameter:"out"`

    Force           bool `glazed.parameter:"force"`
    AddPageNumbers  bool `glazed.parameter:"add-page-numbers"`
    AllPages        bool `glazed.parameter:"all-pages"`
    AnnotationsOnly bool `glazed.parameter:"annotations-only"`

    CloudInputSettings
}
```

### Recommended `pkg/rmcloud` helper pseudocode

```go
func DownloadDocumentByPath(ctx context.Context, auth AuthSettings, remotePath, outDir string) (*DownloadedDocument, error) {
    if err := ctx.Err(); err != nil {
        return nil, err
    }
    remotePath = strings.TrimSpace(remotePath)
    if remotePath == "" {
        return nil, errors.New("remote path is empty")
    }

    _, apiCtx, err := CreateApiCtx(auth)
    if err != nil {
        return nil, err
    }

    node, err := apiCtx.Filetree().NodeByPath(remotePath, nil)
    if err != nil || node.IsDirectory() {
        return nil, errors.New("file doesn't exist")
    }

    if err := os.MkdirAll(outDir, 0o755); err != nil {
        return nil, errors.Wrap(err, "ensure output dir")
    }

    localPath := filepath.Join(outDir, fmt.Sprintf("%s.%s", node.Name(), util.RMDOC))
    if err := apiCtx.FetchDocument(node.Document.ID, localPath); err != nil {
        return nil, errors.Wrapf(err, "failed to download %s", remotePath)
    }

    return &DownloadedDocument{
        RemotePath: remotePath,
        LocalPath:  localPath,
        NodeID:     node.Document.ID,
        Name:       node.Name(),
    }, nil
}
```

### Recommended command resolver pseudocode

```go
func ResolveRMDocInput(ctx context.Context, file string, s CloudInputSettings) (*ResolvedRMDocInput, error) {
    file = strings.TrimSpace(file)
    if file == "" {
        return nil, errors.New("file is empty")
    }

    if !s.Cloud {
        return &ResolvedRMDocInput{
            RequestedPath: file,
            LocalPath:     file,
            Source:        "local",
            Cleanup:       func() error { return nil },
        }, nil
    }

    tmpDir, err := os.MkdirTemp("", "remarquee-rmdoc-cloud-*")
    if err != nil {
        return nil, errors.Wrap(err, "create temp dir")
    }

    downloaded, err := rmcloud.DownloadDocumentByPath(ctx, rmcloud.AuthSettings{
        NonInteractive: s.NonInteractive,
        Reauth:         s.Reauth,
    }, file, tmpDir)
    if err != nil {
        _ = os.RemoveAll(tmpDir)
        return nil, err
    }

    return &ResolvedRMDocInput{
        RequestedPath: file,
        LocalPath:     downloaded.LocalPath,
        Source:        "cloud",
        RemotePath:    downloaded.RemotePath,
        Cleanup:       func() error { return os.RemoveAll(tmpDir) },
    }, nil
}
```

### Recommended `render-v6` execution pseudocode

```go
func (c *RenderV6Command) execute(ctx context.Context, s *RenderV6Settings) (*renderV6Result, error) {
    input, err := ResolveRMDocInput(ctx, s.File, s.CloudInputSettings)
    if err != nil {
        return nil, err
    }
    defer func() { _ = input.Cleanup() }()

    doc, err := pkg_rmdoc.OpenFile(ctx, input.LocalPath)
    if err != nil {
        return nil, err
    }
    if doc.Schema != pkg_rmdoc.SchemaCPages {
        return nil, errors.Errorf("render-v6 only supports cPages/V6 archives; detected schema=%s", schemaString(doc.Schema))
    }
    if doc.Type == pkg_rmdoc.DocTypeEPUB {
        return nil, errors.New("render-v6: epub not supported")
    }

    out := s.Out
    if out == "" {
        out = defaultRenderOutputPath(input.LocalPath, "-v6.pdf")
    }
    if err := ensureOutputWritable(out, s.Force); err != nil {
        return nil, err
    }

    res, err := rmdocrender.MergeRMDocV6OntoBackgroundPDFWithInfo(ctx, input.LocalPath, rmdocrender.V6MergeOptions{})
    if err != nil {
        return nil, err
    }
    if err := os.WriteFile(out, res.PDF, 0o644); err != nil {
        return nil, errors.Wrap(err, "write output pdf")
    }

    return &renderV6Result{
        InputSource: input.Source,
        InputPath:   input.RequestedPath,
        OutputPath:  out,
        Schema:      schemaString(doc.Schema),
        Type:        docTypeString(doc.Type),
        Pages:       len(doc.Pages),
    }, nil
}
```

## Design Decisions

### Decision 1: Use temp downloads, not in-memory streaming

This is the most important design choice.

Why temp downloads are correct:

- `render-legacy` ultimately hands a path to rmapi's PDF generator (`render_legacy.go:142`).
- `render-v6` hands a path to `MergeRMDocV6OntoBackgroundPDFWithInfo`, which reopens the archive (`render_v6.go:118`, `v6_merge_background.go:272-281`).
- Other archive helpers, such as `ReadRMFileFromArchive`, also operate on archive paths (`pkg/rmdoc/rm_archive_rmfiles.go:36-105`).

If we tried to "stream from cloud" into memory, we would still need to re-plumb the render stack to pass raw bytes or reader interfaces everywhere. That is much larger than the requested feature.

### Decision 2: Keep cloud logic out of `pkg/rmdoc`

`pkg/rmdoc` is an archive parser/renderer package. Mixing cloud authentication into it would:

- add network/auth coupling to a local parser,
- make tests more complex,
- blur the distinction between "open archive bytes" and "find remote node and download archive".

That would be a bad package boundary.

### Decision 3: Add `--cloud` as an explicit flag, not implicit path guessing

Do not guess based on whether the input starts with `/`.

Why:

- local absolute paths also start with `/`
- implicit heuristics create ambiguous behavior
- an explicit flag is easier to explain and test

### Decision 4: Reuse `pkg/rmcloud`, not `cloud get`

The reusable layer should be a package helper, not a function hidden inside a command implementation.

Why:

- command packages should not become dependency hubs
- `pkg/rmcloud` already contains the shared auth abstraction (`pkg/rmcloud/auth.go:11-21`)
- the new helper is cloud infrastructure, not a CLI surface

### Decision 5: Preserve current output naming behavior

When `--out` is omitted:

- local `foo.rmdoc` still yields `foo-v6.pdf` or `foo-annotations.pdf`
- cloud `/Books/Foo` should yield the same names after download because the temp file will be `Foo.rmdoc`

This keeps user expectations consistent.

## Alternatives Considered

### Alternative A: Add remote support directly to `pkg/rmdoc.OpenFile`

Rejected.

Problems:

- requires auth flags to reach a low-level package
- breaks the clean parser/transport boundary
- would not help the legacy generator, which still wants a path

### Alternative B: Shell out to `remarquee cloud get` from `rmdoc render-*`

Rejected.

Problems:

- command shells are a poor internal API
- harder error handling
- harder tests
- duplicates CLI parsing and output handling

### Alternative C: Introduce a new command such as `rmdoc render-cloud`

Rejected for now.

Problems:

- explodes the verb surface further
- duplicates existing render verbs
- worsens the CLI cleanup problem already identified in MO-001

### Alternative D: Unify immediately into `rmdoc render`

Technically attractive, but out of scope for this ticket.

This repo likely should move toward:

```text
remarquee rmdoc render <path> [--cloud] [--format pdf|png]
```

But that is a broader CLI migration. The small, safe step is to add `--cloud` to the two existing render commands first.

## Phased Implementation Plan

### Phase 1: Extract reusable cloud download helper

Files:

- `pkg/rmcloud/download.go` (new)
- `cmd/remarquee/cmds/cloud/get.go`

Tasks:

1. Move the "resolve node and fetch document" logic behind a reusable `pkg/rmcloud` helper.
2. Update `cloud get` to call the helper.
3. Keep CLI behavior identical.

Expected outcome:

- `cloud get` still works.
- The new code path becomes reusable by `rmdoc`.

### Phase 2: Add shared `rmdoc` input resolver

Files:

- `cmd/remarquee/cmds/rmdoc/input_resolver.go` (new)

Tasks:

1. Define shared cloud settings struct.
2. Define resolved input struct.
3. Implement temp-dir lifecycle and cleanup.
4. Return local path for both local and cloud modes.

Expected outcome:

- `rmdoc` commands can ask for "give me a usable local archive path" without caring where it came from.

### Phase 3: Wire `render-v6`

Files:

- `cmd/remarquee/cmds/rmdoc/render_v6.go`
- `cmd/remarquee/cmds/rmdoc/render_v6_test.go`

Tasks:

1. Add `--cloud`, `--non-interactive`, `--reauth` flags.
2. Refactor `Run` and `RunIntoGlazeProcessor` through one execution helper.
3. Resolve input before validation/rendering.
4. Add tests for output naming and resolved input behavior.

Expected outcome:

- `render-v6` accepts remote paths through `--cloud`.

### Phase 4: Wire `render-legacy`

Files:

- `cmd/remarquee/cmds/rmdoc/render_legacy.go`
- new tests if needed

Tasks:

1. Add the same flags/settings.
2. Use the shared input resolver.
3. Refactor duplicated execution paths.
4. Preserve legacy-specific options.

Expected outcome:

- `render-legacy` can render cloud-backed legacy documents after temp download.

### Phase 5: Manual validation and docs

Files:

- ticket docs
- optional CLI help docs if present elsewhere

Tasks:

1. Verify local mode still works.
2. Verify cloud mode for one V6 and one legacy fixture/document.
3. Confirm temp cleanup occurs.
4. Confirm structured glaze output still includes the expected fields.

## Testing And Validation Strategy

### Automated tests

Add tests at two levels.

#### 1. Resolver-focused unit tests

Prefer designing the resolver so its download step can be injected or wrapped for tests.

What to test:

- local mode returns the original path
- cloud mode creates a temp dir and returns a downloaded path
- cleanup removes temp content
- empty file input errors cleanly

#### 2. Command smoke tests

There is already a local smoke test for `render-v6` that writes a PDF from `cpage-pdf.rmdoc` (`cmd/remarquee/cmds/rmdoc/render_v6_test.go:24-58`).

Extend coverage to include:

- `render-v6` local mode still passes
- cloud-mode execution path can be exercised with an injected resolver or a fake downloader
- legacy render path at least verifies setting parsing and path resolution behavior

### Manual validation

Use a real cloud-backed document for each renderer:

```bash
remarquee status
remarquee cloud account --non-interactive

remarquee rmdoc render-v6 /Some/V6/Document --cloud --non-interactive --force
remarquee rmdoc render-legacy /Some/Legacy/Document --cloud --non-interactive --force
```

Validation checklist:

1. The command succeeds without a manual `cloud get`.
2. Output PDF lands in the expected directory.
3. Output naming matches local behavior.
4. Temporary download directory is removed afterward.
5. Structured output mode still reports input/output/schema/type fields.

### Regression checks

Because `remarks` automatically extracts `.rmdoc` archives internally, it remains a good conceptual reference for "remote input should become local working state before render" (`ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/analysis/01-using-remarks-for-golden-testing-and-pdf-comparison.md:47-77`).

The critical regression risk is not rendering math. It is command orchestration:

- wrong temp cleanup timing
- wrong output naming
- remote path resolution failures
- local behavior accidentally changing when `--cloud` is not set

## Risks, Tradeoffs, And Open Questions

### Risk 1: Temp file lifecycle bugs

If cleanup runs too early, the renderer may fail mid-run. If cleanup does not run, temp files will accumulate.

Mitigation:

- resolver returns explicit `Cleanup`
- caller defers cleanup immediately after successful resolution
- tests assert cleanup behavior

### Risk 2: Repeated flag definitions across commands

This feature adds auth flags to more commands. The repo already repeats these flags across `cloud` and `upload` (`rg` shows the same `non-interactive` and `reauth` strings across many command files).

Mitigation:

- at minimum, share the settings struct
- longer-term, centralize cloud-auth flag registration

### Risk 3: Structured output drift

`render-v6` and `render-legacy` support dual-mode output. Refactoring must not accidentally remove or rename existing columns.

Mitigation:

- make `RunIntoGlazeProcessor` consume the same execution result object as `Run`

### Open question 1: Should `inspect` and `build-background` also gain `--cloud` immediately?

My recommendation: not in this ticket.

Reason:

- the user request is specifically about `render` and `render-v6`
- shared resolver should be written so those commands can adopt it later
- keeping the first implementation narrow reduces churn

### Open question 2: Should `render-v6-png` adopt the same resolver right away?

Recommendation: document it as an immediate follow-up, not part of the minimum scope.

Reason:

- it is user-adjacent, but it is also more debug-oriented
- the main user pain is PDF rendering from cloud paths

### Open question 3: Should the long-term CLI become `rmdoc render`?

Probably yes, but not now.

That proposal belongs in the CLI cleanup track because it affects aliases, docs, help text, and migration messaging.

## Suggested File-Level Change Map

```text
pkg/rmcloud/auth.go
pkg/rmcloud/download.go                # new reusable remote-download helper

cmd/remarquee/cmds/cloud/get.go        # call pkg/rmcloud helper

cmd/remarquee/cmds/rmdoc/input_resolver.go   # new shared local/cloud input resolver
cmd/remarquee/cmds/rmdoc/render_v6.go        # add flags + shared execute path
cmd/remarquee/cmds/rmdoc/render_legacy.go    # add flags + shared execute path
cmd/remarquee/cmds/rmdoc/render_v6_test.go   # extend smoke coverage
cmd/remarquee/cmds/rmdoc/...                 # optional follow-up adopters later
```

## References

- `cmd/remarquee/main.go:17-37`
- `cmd/remarquee/cmds/rmdoc/root.go:5-88`
- `cmd/remarquee/cmds/rmdoc/render_v6.go:27-183`
- `cmd/remarquee/cmds/rmdoc/render_legacy.go:27-204`
- `cmd/remarquee/cmds/rmdoc/render_v6_png.go:29-198`
- `cmd/remarquee/cmds/rmdoc/inspect.go:23-119`
- `cmd/remarquee/cmds/cloud/get.go:22-108`
- `pkg/rmcloud/auth.go:11-52`
- `pkg/rmdoc/open.go:15-174`
- `pkg/rmdoc/types.go:52-86`
- `pkg/rmdoc/content.go:11-167`
- `pkg/rmdoc/render/v6_merge_background.go:256-303`
- `pkg/rmdoc/rm_archive_rmfiles.go:36-105`
- `cmd/remarquee/cmds/rmdoc/render_v6_test.go:24-58`
- `ttmp/2025/12/14/RMQ-0004--port-rmdoc-parsing-rendering-to-go-v3-v5-v6/index.md:56-66`
- `ttmp/2025/12/14/RMQ-0004--port-rmdoc-parsing-rendering-to-go-v3-v5-v6/tasks.md:87-90`
- `ttmp/2025/12/14/RMQ-0002--implement-remarquee-cloud-cli/changelog.md:79-86`
- `ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/analysis/01-using-remarks-for-golden-testing-and-pdf-comparison.md:47-77`
