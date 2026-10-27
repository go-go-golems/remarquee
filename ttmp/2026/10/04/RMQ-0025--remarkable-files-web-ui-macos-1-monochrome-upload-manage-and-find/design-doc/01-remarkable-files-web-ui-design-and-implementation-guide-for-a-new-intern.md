---
Title: 'Remarkable files web UI: design and implementation guide for a new intern'
Ticket: RMQ-0025
Status: draft
Topics:
    - remarkable
    - cloud
    - upload
    - web
    - frontend
    - ui
    - react
    - go
    - remarquee
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/build-remarquee-ui-web/main.go
      Note: Dagger/pnpm frontend build helper to reuse or generalize
    - Path: repo://cmd/remarquee-ui/embed.go
      Note: go:embed frontend/dist packaging precedent
    - Path: repo://cmd/remarquee-ui/main.go
      Note: Go HTTP server + dev/prod SPA serving precedent
    - Path: repo://cmd/remarquee/cmds/cloud/put.go
      Note: Reference upload semantics (force vs content-only, annotation loss)
    - Path: repo://cmd/remarquee/cmds/upload/md.go
      Note: Reference Markdown-to-PDF-to-upload pipeline
    - Path: repo://pkg/rmcloud/auth.go
      Note: Auth/tree initialization, retry, and forceSchemaV4 used by the proposed service
    - Path: repo://pkg/rmcloud/dirs.go
      Note: Nested remote folder creation (MkdirAll) reused by ensureDir
ExternalSources: []
Summary: Intern-facing end-to-end design for a local, macOS-1-styled web UI that uploads, manages, and quickly finds reMarkable cloud files by reusing remarquee's rmapi/rmcloud/mdpdf stack behind a small Go HTTP API and a React SPA.
LastUpdated: 2026-10-04T00:00:00-04:00
WhatFor: Onboarding a new engineer onto the planned remarquee files web UI and giving an executable implementation plan.
WhenToUse: When starting implementation of the files web UI, reviewing its architecture, or debugging the Go API / React frontend boundary.
---

# Remarkable files web UI: design and implementation guide for a new intern

## 0. Executive summary

This document is the complete, standalone guide for building a small **local web
application** that lets one person **upload, manage, and quickly find** their
reMarkable cloud files through a browser. The application is *not* a hosted
service. It runs on your laptop as a subcommand of the existing CLI —
**`remarquee serve`** — talks to the reMarkable cloud through the same `rmapi`
library the rest of `remarquee` already uses, and serves a single-page React app
whose visual language is a modern reinterpretation of **classic Macintosh
System 1 (1984)**: 1-bit black-on-white, square corners, hairline rules, dithered
separators, and a **small accent palette used only for text highlights**. There is
deliberately **no window chrome, no title bar, and no menu bar** — just the flat
"look" of early Macintosh layout, rendered with **modern fonts**. The classic
Macintosh bitmap typeface is explicitly **not used**.

The important architectural idea is that we build almost **no new cloud logic**.
The repository already contains everything we need:

- `pkg/rmcloud` — authentication, tree synchronization, download, and
  auth-retry glue around the `rmapi` fork (`pkg/rmcloud/auth.go`,
  `pkg/rmcloud/download.go`).
- `cmd/remarquee/cmds/cloud/*` — the command implementations for `ls`, `find`,
  `search`, `get`, `put`, `rm`, `mv`, `mkdir`, `stat`, `refresh`. These are the
  reference implementations for every file operation the UI exposes.
- `pkg/mdpdf` — Markdown → PDF conversion (pandoc + XeLaTeX), Mermaid/SVG
  handling, and bundle assembly.
- `cmd/remarquee/cmds/upload/md.go` — the reference pipeline that turns Markdown
  into a PDF and uploads it into `/ai/YYYY/MM/DD/…`.
- `cmd/remarquee-ui/*` — an existing, working precedent for exactly the kind of
  Go-server-plus-embedded-React-SPA this ticket needs: `net/http` `ServeMux`,
  `//go:embed frontend/dist`, a Vite dev proxy, and a Dagger/pnpm build helper.

So the new work is mostly **packaging and interaction design**, plus one genuine
engineering task: extracting the command-layer cloud operations into a
**reusable, serialized service** that an HTTP handler can call safely. This
document explains that service, the HTTP API, the frontend, and the macOS-1
visual system in enough detail that an intern can implement it phase by phase.

> **Status note.** This guide was written at commit `02475f6`; the feature is now
> **implemented** in `pkg/rmfiles` and `cmd/remarquee/cmds/serve` (see the
> implementation diary and `tasks.md`). Two deliberate deviations from the draft
> below: (1) the UI ships as the **`remarquee serve`** subcommand, not a separate
> binary; (2) the frontend is a **no-build static app** (vanilla ES modules +
> CSS embedded with `go:embed`) rather than React/Vite, so `go build` always
> works and no node toolchain is required in CI. Where behavior is inferred
> rather than read directly, it is marked "proposal" or "open question".

---

## 1. Scope, audience, and how to read this document

**Audience.** A new engineer who knows Go and basic React/TypeScript but has
never touched `rmapi`, the reMarkable Sync15 protocol, or this repository.

**What you will be able to do after reading.**

1. Explain what reMarkable cloud "documents" actually are and how `rmapi`
   models them.
2. Trace a byte from "user picks a `.md` file in the browser" to "document
   appears in the reMarkable cloud".
3. Understand why a stateless request-per-`ApiCtx` design is wrong here, and
   what to do instead.
4. Implement the backend endpoints and the React frontend from the phased plan
   in Part 6.
5. Apply the macOS-1 design system without inventing window chrome.

**Reading order.** Read Part 1 once for orientation, then Part 2 carefully
(foundational mechanisms), then Part 3 (the problem), then Parts 4–5 (the
design and API contract). Parts 6–10 are the working plan, tests, gotchas, and
open questions. Appendices are lookup material.

**Conventions used in this document.**

- **File references** look like `pkg/rmcloud/auth.go:67` (path, optional line).
- **Symbol references** look like `rmcloud.CreateApiCtx`.
- **Pseudocode** is language-neutral and marked ```text```.
- **Diagrams** are ASCII boxes; they are approximations, not exact call graphs.
- Anything labeled **PROPOSED** does not exist yet; anything else exists in the
  repository at `02475f6`.

---

## 2. Vocabulary (read this first)

These terms are used constantly. Learn them before Part 3.

| Term | Meaning |
| --- | --- |
| **reMarkable cloud** | The vendor service that syncs a user's files across their tablet, desktop app, and (via this project) the `rmapi` library. |
| **`rmapi`** | An open-source Go library + CLI that speaks the vendor's **Sync15** protocol. remarquee depends on a fork (`github.com/FNStudios-NI/rmapi`, pinned via a `replace` in `go.mod:23`). |
| **Sync15** | The cloud sync protocol version. All remote work in this repo goes through `SyncVersion = Version15`. |
| **`ApiCtx`** | `rmapi`'s stateful handle: `api.ApiCtx` (`.../rmapi/api/api.go:16`). It owns an in-memory **file tree**, an **apiStorage**, and a cached **HashTree**. Every file operation is a method on it. |
| **File tree** | `filetree.FileTreeCtx` — a node graph mirroring the cloud folders/documents. Built once at context creation, then mutated locally and pushed via `SyncComplete()`. |
| **`model.Node` / `model.Document`** | The tree node and its cloud metadata. `Document` carries `ID`, `Name`, `Version`, `ModifiedClient`, `Type`, `CurrentPage`, `Starred`, `Parent`, `Tags`. |
| **Node type** | `CollectionType` (folder), `DocumentType` (file), `TemplateType` (hidden template; the `ls`/`find` commands filter it out by default). |
| **`.rmdoc`** | The archive format the CLI can download a cloud document into. Not needed for the first UI milestone, but relevant to "manage". |
| **`~/.rmapi`** | Default token file (`config.ConfigPath()`, overridable with `RMAPI_CONFIG`). Holds `DeviceToken` and `UserToken`. |
| **`tree.cache`** | The locally cached file tree at `os.UserCacheDir()/rmapi/tree.cache` (`api/sync15/common.go:31-41`). Used to avoid a full re-sync on every start. |
| **`rmcloud`** | remarquee's thin glue package (`pkg/rmcloud`) that wraps `rmapi` auth + tree init and adds progress reporting and auth retry. |
| **`mdpdf`** | remarquee's Markdown→PDF package (`pkg/mdpdf`), wrapping pandoc/XeLaTeX, Mermaid, and SVG. |
| **macOS 1 / System 1** | Apple's 1984 Macintosh system software: 1-bit graphics, square corners, hairline rules, its distinctive bitmap typeface. Our UI keeps the *layout and contrast language* but uses modern fonts only — the classic bitmap typeface is explicitly **not** used. |

---

## 3. Part 1 — The system end to end

### 3.1 What remarquee is

`remarquee` is a Go CLI (`cmd/remarquee`) plus a set of `pkg/` libraries. Its
README describes the core problem: reMarkable is a great reading/thinking device
but the off-device workflows (browse, upload, export, prepare PDFs) are manual.
`remarquee` turns those into scriptable commands. The five capability areas are
cloud filesystem operations, Markdown/source upload, `.rmdoc` inspection and
rendering, on-device capture, and RMDoc-DSL fixtures.

For this ticket we care about the first two. The UI is essentially a friendly
front-end over:

- **cloud filesystem operations** (`remarquee cloud …`), and
- **Markdown/source upload** (`remarquee upload md|src|bundle …`).

### 3.2 The shape of the new application

```text
+---------------------------------------------------------------------------+
|                         Your laptop (localhost)                           |
|                                                                           |
|   +--------------------------+          +-----------------------------+   |
|   |  Browser                 |  fetch   |  Go HTTP server             |   |
|   |  React SPA (macOS-1 CSS) | <======> |  remarquee serve            |   |
|   |  - file list             |  /api/*  |  (cmd/remarquee/cmds/serve) |   |
|   |  - search box            |          |  go:embed frontend/dist     |   |
|   |  - upload dropzone       |          +-------------+---------------+   |
|   |  - manage actions        |                        |                   |
|   +--------------------------+                        v                   |
|                                          +-----------------------------+   |
|                                          | pkg/rmfiles (PROPOSED svc)  |   |
|                                          | - one long-lived ApiCtx     |   |
|                                          | - mutex-serialized ops      |   |
|                                          | - search index              |   |
|                                          | - upload job runner         |   |
|                                          +----+-----------+------------+   |
|                                               |           |                |
|                              +----------------+   +--------+-----------+    |
|                              v                     v                    v    |
|                        pkg/rmcloud            pkg/mdpdf        rmapi apiCtx |
|                        (auth, tree,           (md -> pdf)      (filetree,  |
|                         download)                              upload, rm) |
+--------------------------------------+------------------------------------+
                                       | HTTPS (Sync15)
                                       v
                              reMarkable cloud
```

The **only** network dependency is the reMarkable cloud, reached through
`rmapi`. The browser never talks to the cloud; it only talks to the local Go
server, which the user starts with **`remarquee serve`**. This keeps credentials
out of the browser entirely — tokens stay in `~/.rmapi` on disk and are never
serialized into JSON responses.

**Decision (resolved): ship the UI as a CLI subcommand, not a separate binary.**
The server is a new `serve` command under `cmd/remarquee/cmds/serve/`, registered
in `cmd/remarquee/main.go` next to `cloud`, `upload`, `rmdoc`, and `device`. This
gives users one binary and a discoverable `remarquee serve --help`; it also means
the embedded SPA ships in the existing `remarquee` build. Typical usage:

```bash
remarquee serve                       # serve the UI on 127.0.0.1:8080
remarquee serve --addr 127.0.0.1:9090 # custom address
remarquee serve --dev                 # dev: expect Vite on :5173, proxy /api
remarquee serve --remote-dir /ai      # default upload destination root
```

### 3.3 Package and command map (today, at `02475f6`)

```text
remarquee/
  cmd/
    remarquee/                     # main CLI
      cmds/
        cloud/                     # ls find search get put rm mv mkdir stat refresh account
        upload/                    # md src bundle sync
        rmdoc/                     # inspect render-legacy render-v6 ...
        device/                    # capture server (on-device)
    remarquee-ui/                  # EXISTING web UI precedent (rmdoc rendering)
      main.go                      # net/http ServeMux + go:embed
      api/                         # HTTP handlers
      frontend/                    # Vite + React 19 + Redux Toolkit
      embed.go  gen.go  Makefile
    build-remarquee-ui-web/        # Dagger/pnpm frontend build helper
  pkg/
    rmcloud/                       # auth.go dirs.go download.go progress.go logtransport.go
    mdpdf/                         # pandoc.go mermaid.go svg.go bundle.go images.go preprocess.go
    rmdoc/                         # .rmdoc parse/render
    rmdsl/                         # RMDoc DSL
    devicecapture/                 # on-device capture
    pdfcmp/  refimpl/  doc/
```

**File references you will use constantly:**

- `pkg/rmcloud/auth.go` — `CreateApiCtx`, `WithAuthRetry`, `IsAuthError`.
- `pkg/rmcloud/download.go` — `DownloadDocumentByPath`.
- `cmd/remarquee/cmds/cloud/ls.go` — the structured row shape (`id`, `name`,
  `type`, `is_dir`, `path`, `parent_id`, `version`, `modified_client`,
  `modified_time`) and `buildPathFromParents`.
- `cmd/remarquee/cmds/cloud/search.go` / `find.go` — recursive traversal and
  regex matching.
- `cmd/remarquee/cmds/cloud/put.go` — upload semantics including
  `--force`/`--content-only`.
- `cmd/remarquee/cmds/cloud/rm.go` — delete semantics and the `--yes` safety gate.
- `cmd/remarquee/cmds/upload/md.go` — the Markdown→PDF→upload pipeline.
- `pkg/mdpdf/pandoc.go` — `PandocOptions`, `ConvertMarkdownFileToPDF`.
- `cmd/remarquee-ui/main.go`, `cmd/remarquee-ui/embed.go`,
  `cmd/build-remarquee-ui-web/main.go` — the web-app packaging precedent.

### 3.4 End-to-end data flow: "upload a Markdown note"

This is the single most important flow to understand. It reuses the CLI upload
pipeline almost verbatim.

```text
Browser                Go server (rmfiles)              rmapi / cloud
-------                -------------------              -------------
POST /api/upload
 (multipart: file,    1. write body -> temp .md
  remoteDir, name)    2. collectMarkdownInputs(...)          (reuse md.go logic)
                      3. mdpdf.ConvertMarkdownFileToPDF  --> pandoc + xelatex
                          (PandocOptions, DejaVu fonts)       (local process)
                      4. resolve remote dir node          --> Filetree.NodeByPath(remoteDir)
                      5. apiCtx.UploadDocument(parentId,  --> Sync15 blob upload
                            pdfPath, notify=true, ...)        + SyncComplete()
                      6. apiCtx.Filetree().AddDocument(doc)
                      7. respond 201 {id, path, name}
<-- 201 JSON
```

Two subtleties an implementer must not miss:

1. **`UploadDocument` takes `parentId`**, so the server must first resolve the
   destination *path* to a *node* (`Filetree().NodeByPath`). `rmcloud.MkdirAll`
   exists for creating the whole `/ai/YYYY/MM/DD` chain (`pkg/rmcloud/dirs.go:30`).
2. **The tree is mutated in memory** (`AddDocument`, `DeleteNode`, `MoveNode`)
   and must be kept consistent with the cache; `SyncComplete()` persists the
   change. A long-lived server must therefore *own* one `ApiCtx` and serialize
   mutations (see §4.4).

### 3.5 End-to-end data flow: "find a file"

```text
Browser                Go server                         rmapi
-------                ---------                         -----
GET /api/files?query=  1. ensure tree is loaded/fresh
  selfish&limit=50      2. WalkTree(root) -> []Row        (or reuse index)
                       3. fuzzy-score against name/path
                       4. sort desc, truncate to limit
<-- 200 JSON rows
```

For a small tree, an on-demand `WalkTree` is fine. For a large tree, build a
cached index at startup and refresh it on mutation, so search is O(index) not
O(HTTP). Part 4 specifies the index.

---

## 4. Part 2 — Foundational mechanisms you must understand

### 4.1 `ApiCtx`: the stateful heart of everything

`api.ApiCtx` is an interface (`.../rmapi/api/api.go:16`):

```go
type ApiCtx interface {
    Filetree() *filetree.FileTreeCtx
    FetchDocument(docId, dstPath string) error
    CreateDir(parentId, name string, notify bool) (*model.Document, error)
    UploadDocument(parentId string, sourceDocPath string, notify bool,
        coverpage *int, currentPage *int, pageCount *int, contrastFilter *string) (*model.Document, error)
    ReplaceDocumentFile(docId, sourceDocPath string, notify bool) error
    MoveEntry(src, dstDir *model.Node, name string) (*model.Node, error)
    DeleteEntry(node *model.Node, recursive, notify bool) error
    SyncComplete() error
    Nuke() error
    Refresh() (string, int64, error)
}
```

Everything the UI does maps to one of these methods. Note that it is **not**
context-aware; cancellation is handled by remarquee above the library (see
`pkg/rmcloud/auth.go` and the `RMQ-HANG-001` ticket for the cancellation
story).

Creating a context is expensive and side-effecting. `rmcloud.CreateApiCtx`
(`pkg/rmcloud/auth.go:67`) does, in order:

1. `api.AuthHttpCtx(reauth, nonInteractive)` — loads tokens from `~/.rmapi`,
   prompting for a one-time code only if `NonInteractive` is false.
2. `api.ParseToken(userToken)` — extracts `UserInfo` including `SyncVersion`.
3. `initializeTree(...)` → `api.CreateApiCtx(httpCtx, version)` → Sync15
   `CreateCtx`, which **loads `tree.cache`, mirrors it against remote storage,
   and writes the cache back** (`api/sync15/apictx.go:43-54`).
4. `forceSchemaV4(apiCtx)` — a reflection workaround for an rmapi bug where an
   empty `HashTree.SchemaVersion` defaults to V3 and produces HTTP 400
   "invalid hash".
5. Retries up to `authRetries = 3` times, escalating to `reauth=true`.

**Implication for the web UI:** context creation can take several seconds on the
first call of a session. Do it **once** at server start (or lazily on first
request, with a loading state in the UI), not per HTTP request. Do not run two
contexts concurrently against the same `tree.cache`.

### 4.2 The file tree API you will actually call

From `filetree.FileTreeCtx` (`.../rmapi/filetree/filetree.go`):

- `Root() *model.Node` — the root node (its `Id()` is `""`).
- `NodeByPath(path string, current *model.Node) (*model.Node, error)` — resolve
  a path to a node. Used before upload/move/delete.
- `NodesByPath(path string, current *model.Node, ignoreTrailingSlash bool)
  ([]*model.Node, error)` — used by `ls`; supports patterns.
- `NodeById(id string) *model.Node` — the in-memory index by document ID.
- `NodeToPath(node) (string, error)` — reverse lookup.
- `AddDocument(doc)`, `DeleteNode(node)`, `MoveNode(src, dst)` — local tree
  mutations to keep in sync after an operation.
- `WalkTree(node, FileTreeVisitor{Visit: func(node, path) bool})`
  (`filetree/treeutil.go:14`) — depth-first traversal used by `find`.

`model.Node` helpers (`.../rmapi/model/node.go`):

- `Name()`, `Id()`, `Version()`, `IsRoot()`, `IsDirectory()` (`Type ==
  "CollectionType"`), `IsFile()`.
- `LastModified() (time.Time, error)` — parses `Document.ModifiedClient`
  (RFC3339Nano).

`buildPathFromParents` (defined in `cmd/remarquee/cmds/cloud/ls.go:305` and
duplicated in `find.go`, `sync.go`) turns a node into a `/A/B/name` path by
walking `Parent` pointers. **This helper should be centralized** when we extract
the service (Part 4).

### 4.3 Authentication, tokens, and caches

- **Tokens:** `~/.rmapi` (or `$RMAPI_CONFIG`, else XDG `rmapi/rmapi.conf`).
  Contains `DeviceToken` and `UserToken`. Treat as secrets.
- **Tree cache:** `os.UserCacheDir()/rmapi/tree.cache` with a `CacheVersion`
  guard (`api/sync15/common.go:44`). Corrupt caches trigger a full resync.
- **Non-interactive mode:** set `AuthSettings.NonInteractive = true` so a
  missing token is an error rather than a blocking terminal prompt. The server
  must do this; a web request cannot answer a prompt. If tokens are missing, the
  API should return a specific `401`/`auth_required` payload that the UI renders
  as a "run `rmapi reset` / authenticate" page.
- **Auth retry:** `rmcloud.WithAuthRetry` (`pkg/rmcloud/auth.go:172`) runs an
  operation, and if it fails with a 401/403/expired-token error, rebuilds the
  context with `reauth=true` and retries once. The server should wrap mutating
  operations in this so a stale token doesn't require a manual restart.
- **Never log or return tokens.** `--log-level debug` in `rmcloud` deliberately
  does not log URLs, headers, or bodies (`pkg/rmcloud/logtransport.go`). Keep it
  that way in the service.

### 4.4 Concurrency and the single-context rule (critical)

`ApiCtx` is **stateful and not safe for concurrent mutation**. Two requests that
both call `DeleteEntry` + `DeleteNode` on the same context can corrupt the
in-memory tree and the cached HashTree. Meanwhile, two *independent*
`CreateApiCtx` calls race on the same `tree.cache` file.

The design therefore mandates:

```text
pkg/rmfiles.Service
  mu        sync.Mutex          // guards apiCtx + tree + index
  apiCtx    api.ApiCtx          // ONE long-lived context
  index     []Entry             // derived from the tree
  uploadSem chan struct{}       // bounded concurrent conversions/upload

  func (s *Service) withTree(fn func(api.ApiCtx) error) error {
      s.mu.Lock(); defer s.mu.Unlock()
      return rmcloud.WithAuthRetry(ctx, s.auth, s.apiCtx, func(c api.ApiCtx) (api.ApiCtx, error) {
          s.apiCtx = c
          if err := fn(c); err != nil { return c, err }
          if err := c.SyncComplete(); err != nil { return c, err }
          s.reindex(c)            // keep the search index consistent
          return c, nil
      })
  }
```

Reads (`ls`, `search`) can either take the lock briefly (walk the tree under the
mutex, then release before serializing) or read from an immutable snapshot
index. Mutations always take the lock and call `SyncComplete()`.

**Long uploads** (pandoc can take seconds per file) should **not** hold the tree
lock while converting. Split the operation: convert outside the lock, then take
the lock only for the `UploadDocument` + `AddDocument` + `SyncComplete` window.
Part 4 specifies an upload-job model for this.

### 4.5 The Markdown→PDF pipeline (`pkg/mdpdf`)

Relevant API (`pkg/mdpdf/pandoc.go`):

```go
type PandocOptions struct { /* pandoc path, pdf engine, fonts, geometry, layout, ... */ }
func DefaultPandocOptions() PandocOptions
func ConvertMarkdownFileToPDF(ctx context.Context, mdPath, outPDF string, opts PandocOptions) error
```

Bundle assembly for multi-file uploads (`pkg/mdpdf/bundle.go`):

```go
func BuildBundleMarkdown(ctx context.Context, inputs []BundleInput, tmpDir string,
    mermaidCfg *MermaidRendererConfig, svgCfg *SVGRendererConfig, resolveImages bool) (string, error)
```

The CLI's `upload md` command (`cmd/remarquee/cmds/upload/md.go:118`) is the
orchestration to copy:

1. `collectMarkdownInputs(paths)` — resolve files/dirs to absolute `.md` paths.
2. `resolveRemoteDir(date, override)` — default `/ai/YYYY/MM/DD`.
3. `configureMarkdownPandocOptions(...)` — fonts (DejaVu Sans / DejaVu Sans
   Mono), layout preset, geometry.
4. `buildMarkdownConversionJobs(...)` + `convertMarkdownJobs(...)` with a worker
   pool (`--workers`) — parallel pandoc.
5. For each job: resolve the remote dir node, `UploadDocument`, `AddDocument`.

For the UI's first milestone you can restrict to **single `.md` and single
`.pdf` uploads** and reuse this logic; multi-file directory upload and bundles
come later (see phased plan).

### 4.6 The existing web-UI precedent (`cmd/remarquee-ui`)

This is not hypothetical — the repo already ships a Go + React web app. Copy its
packaging pattern exactly.

- **Server:** `cmd/remarquee-ui/main.go` builds a `net/http.ServeMux`, registers
  `/api/...` handlers, and in production serves the SPA from an embedded FS.
  It has explicit `ReadHeaderTimeout`/`ReadTimeout`/`WriteTimeout`/`IdleTimeout`.
- **Embedding:** `cmd/remarquee-ui/embed.go`:

  ```go
  //go:embed frontend/dist
  var frontendDist embed.FS
  func GetFrontendFS() (fs.FS, error) { return fs.Sub(frontendDist, "frontend/dist") }
  ```

- **Build:** `cmd/remarquee-ui/gen.go` runs `go run ../build-remarquee-ui-web`.
  That helper (`cmd/build-remarquee-ui-web/main.go`) builds the frontend inside a
  Dagger container (`node:22`, pnpm from `packageManager`, cached pnpm store) and
  exports `dist/`; it falls back to local pnpm when Dagger is unavailable, and
  honors `BUILD_WEB_LOCAL`, `RMQ_WEB_DIR`, `RMQ_WEB_DIST`.
- **Frontend:** Vite + React 19 + TypeScript + Redux Toolkit. Store split into
  slices (`documentsSlice`, `renderSlice`, `validationSlice`). `vite.config.ts`
  proxies `/api` to `http://localhost:8080`.
- **Makefile:** `dev-backend`, `dev-frontend`, `build`, `clean`, `test`.

**Decision:** the new UI is a *different application* with a different purpose
(files management vs. rmdoc rendering), but it should **not** be a separate
binary. Do not overload `remarquee-ui`. Instead add a **`remarquee serve`**
subcommand (PROPOSED) under `cmd/remarquee/cmds/serve`, and, where sensible,
parametrize the shared frontend build helper so both `remarquee-ui` and
`remarquee serve` can use it (the helper already accepts `RMQ_WEB_DIR` and
`RMQ_WEB_DIST`). The existing `remarquee-ui` binary can remain as-is; the new
UI lives in the main CLI.

### 4.7 Serving the SPA correctly (SPA fallback)

A single-page app must serve `index.html` for client-side routes like
`/files` or `/search?q=…`. The naive `http.FileServer` returns 404. Use a small
wrapper:

```text
func spaHandler(dist fs.FS) http.Handler {
    fileServer := http.FileServer(http.FS(dist))
    return http.HandlerFunc(func(w, r) {
        if strings.HasPrefix(r.URL.Path, "/api/") { http.NotFound(w, r); return }
        p := strings.TrimPrefix(r.URL.Path, "/")
        if p == "" { p = "index.html" }
        if _, err := fs.Stat(dist, p); err != nil {
            r = cloneWithPath(r, "/index.html")   // fall back to the SPA shell
        }
        fileServer.ServeHTTP(w, r)
    })
}
```

Remember: **static assets belong under a stable prefix** (the repo's web
guidelines say `/static/`; Vite emits `/assets/` by default — either is fine as
long as `/api/` is never shadowed).

### 4.8 Deleting and moving: safety semantics to preserve

- `rm` refuses to delete without `--yes` (`cmd/remarquee/cmds/cloud/rm.go:107`).
  The UI must reproduce this with an explicit confirmation dialog; a single
  click must never delete.
- `put` without `--force` errors if the name exists; `--force` deletes the
  existing document **and its annotations** (`put.go:184`). The UI must warn
  loudly, default to "rename"/"content-only", and require a typed confirmation
  for overwrite.
- `mv` supports rename and move; moving a folder into itself must be rejected.

---

## 5. Part 3 — The problem, precisely

### 5.1 What the user asked for

> "add a simple web UI in a retro monochrome macOS 1 style (modern fonts, no
> window chrome, no menu bar, just the overall look, simple color highlights for
> font color) to allow me to upload and manage and quickly find my remarkable
> files."

Decompose this into requirements.

**Functional.**

- **Upload** a local file (Markdown and PDF at minimum) to a chosen remote
  folder; Markdown converts to PDF first.
- **Manage**: list folders/documents; create folders; rename; move; delete
  (with confirmation); download a document as `.rmdoc` (nice-to-have v1).
- **Quickly find**: search by name/path with instant, incremental results.
- **Refresh**: force a cloud tree resync.

**Visual.**

- 1-bit monochrome base (black on white, white on black for selection).
- Square corners, hairline rules, dithered/hatched separators, compact type.
- **Modern fonts only.** A clean grotesque with tight spacing. The classic Macintosh bitmap typeface is explicitly out of scope and must not be used.
- **No window chrome**: no title bar, no traffic-light buttons, no fake window.
- **No menu bar**.
- **Color only as text highlights** (accent palette), not as large fills.

**Non-functional.**

- Localhost only; no external exposure by default.
- Credentials never leave the machine / never reach the browser.
- Destructive operations are guarded.
- Single binary with embedded assets (`go build` produces a runnable app).

### 5.2 Non-goals (for v1)

- Multi-user accounts, auth, or remote hosting.
- Real-time collaboration.
- Editing document contents on the tablet.
- Full `.rmdoc` rendering inside the UI (that is `remarquee-ui`'s job).
- Offline mode / local mirror as the source of truth.
- Mobile layout polish (desktop-first; workable on tablet widths).

### 5.3 Why this is not trivial (the real risks)

1. **Statefulness of `ApiCtx`.** Covered in §4.4. This is the #1 source of
   subtle corruption bugs.
2. **Slow, blocking cloud work inside HTTP handlers.** Tree init and uploads can
   take seconds. We need jobs + polling (or SSE) instead of naive synchronous
   handlers for uploads.
3. **Path → node resolution.** The cloud has no stable user-facing path; paths
   are synthesized by walking parents. Renames invalidate paths, so the UI
   should carry document **IDs** as the primary key and paths as display data.
4. **Token prompting.** A web server cannot prompt for a one-time code.
   Non-interactive auth failures must become a clear UI state.
5. **Cross-platform build.** pandoc/XeLaTeX and the Dagger frontend build add
   toolchain assumptions; the app must degrade with a clear message when pandoc
   is missing.

---

## 6. Part 4 — Design

### 6.1 Architecture overview

Three layers, each independently testable:

```text
Layer 3  Frontend SPA        React + TS + CSS tokens, no cloud knowledge
Layer 2  HTTP API            net/http ServeMux handlers, JSON, multipart
Layer 1  rmfiles Service     one ApiCtx, serialized mutations, index, jobs
Layer 0  Remotable libs      pkg/rmcloud, pkg/mdpdf, rmapi apiCtx
```

Dependency direction is strictly downward: the frontend never imports Go; the
HTTP layer never touches `rmapi` directly (it calls the service); the service is
the only place that owns an `ApiCtx`.

### 6.2 Proposed new code layout

```text
cmd/remarquee/cmds/serve/
  command.go       # cobra command `serve`; flags --addr --dev --remote-dir --workers
  server.go        # mux wiring, SPA handler, graceful shutdown
  embed.go         # //go:embed frontend/dist   (assets ship inside the remarquee binary)
  gen.go           # //go:generate go run ../../../build-remarquee-ui-web
  frontend/        # Vite + React + TS

pkg/rmfiles/
  service.go       # Service, NewService, lock discipline, reindex
  entries.go       # Entry type, path building, tree -> []Entry
  search.go        # fuzzy scorer, index query
  upload.go        # conversion + upload jobs
  manage.go        # mkdir, rename/move, delete, download
  errors.go        # typed errors -> HTTP status mapping
  service_test.go  # httptest-free unit tests with a fake ApiCtx
```

Registration: add `rootCmd.AddCommand(serve_cmd.NewServeCommand())` in
`cmd/remarquee/main.go` alongside the existing `cloud`, `upload`, `rmdoc`,
`device`, and `ocr` commands. The command's `RunE` builds the `rmfiles.Service`,
constructs the `http.Server`, and blocks until the context is cancelled (SIGINT
is already propagated by `executeInterruptible` in `cmd/remarquee/main.go`).

`pkg/rmfiles` is deliberately HTTP-agnostic so it can be unit-tested with a fake
`api.ApiCtx` and later reused by a future CLI command.

### 6.3 The `Service` type

```go
type Config struct {
    Auth        rmcloud.AuthSettings // NonInteractive=true, Reauth, Progress
    RemoteRoot  string               // default upload root, e.g. "/ai"
    DefaultDir  string               // default remote dir, e.g. "/"
    Workers     int                  // conversion worker pool size
    Refresh     time.Duration        // background resync interval (0 = off)
}

type Service struct {
    cfg   Config
    mu    sync.Mutex
    apiCtx api.ApiCtx
    index []Entry        // immutable snapshot, replaced wholesale on reindex
    jobs  *jobStore      // in-memory upload/job registry
}

func NewService(ctx context.Context, cfg Config) (*Service, error)   // creates ApiCtx once
func (s *Service) Close() error
func (s *Service) Refresh(ctx context.Context) error                  // full resync + reindex
func (s *Service) Status() Status
```

`Status` exposes only non-secret fields for the UI: `{authenticated bool,
user string, syncVersion string, documentCount int, lastRefresh time.Time,
pandocAvailable bool}`.

### 6.4 The `Entry` model (the API's currency)

Derived from `model.Node`, carrying the ID as the stable key:

```go
type Entry struct {
    ID             string    `json:"id"`
    ParentID       string    `json:"parentId"`
    Name           string    `json:"name"`
    Path           string    `json:"path"`            // synthesized /A/B/name
    IsDir          bool      `json:"isDir"`
    Type           string    `json:"type"`            // DocumentType | CollectionType
    Version        int       `json:"version"`
    ModifiedClient string    `json:"modifiedClient"`
    ModifiedTime   time.Time `json:"modifiedTime"`
    SizeHint       string    `json:"sizeHint,omitempty"` // optional, if available
}
```

Path synthesis must match `ls.go`'s `buildPathFromParents` exactly so CLI and UI
agree. Centralize that function in `pkg/rmfiles/entries.go` and have the CLI
call it (a small refactor; see phil. note in §12).

### 6.5 The search index and scorer

At startup (and after every mutation/refresh) build `index []Entry` by walking
the tree once with `filetree.WalkTree`. Search is **in-memory** and therefore
instant.

Scoring (proposal): a simple subsequence-fuzzy score over `name`, with bonuses
for prefix and path-segment matches, is enough and avoids a dependency.

```text
func score(query, entry) (int, bool):
    q = lower(query); n = lower(entry.Name); p = lower(entry.Path)
    if q == "": return 0, true
    s, ok = fuzzySubsequence(q, n);         if ok: return 100 + s, true
    s, ok = fuzzySubsequence(q, p);         if ok: return  50 + s, true
    return 0, false

func fuzzySubsequence(q, s) (int, bool):
    // greedy left-to-right; bonus for consecutive chars and word starts
    pos = 0; score = 0; streak = 0
    for ch in s:
        if pos < len(q) and ch == q[pos]:
            score += 10 + streak*5
            if atWordStart(s, i): score += 15
            streak += 1; pos += 1
        else:
            streak = 0
    if pos == len(q): return score - len(s)/10, true
    return 0, false
```

The frontend may additionally filter the already-returned page client-side for
zero-latency typing. Keep the authoritative ranking on the server.

### 6.6 Upload jobs (why and how)

A single Markdown file can take a second or more to convert; a directory can
take a minute. Holding an HTTP connection open for that is fragile. Model
uploads as **jobs**:

```text
POST /api/uploads (multipart)         -> 202 {jobId}
GET  /api/uploads/{jobId}             -> 200 {state, progress, error, results}
POST /api/uploads/{jobId}/cancel      -> 200
```

Job state machine:

```text
queued --> converting --> uploading --> done
   |            |             |
   +------------+-------------+----> failed (error message)
   +------------------------------> cancelled
```

Pseudocode:

```text
func (s *Service) StartUpload(files []UploadFile, destDir string, opts UploadOpts) JobID:
    job := s.jobs.New(len(files))
    go func():
        for i, f := range files:
            tmp = writeTemp(f)
            var pdf string
            if isMarkdown(tmp):
                pdf = tmp + ".pdf"
                opts := mdpdf.DefaultPandocOptions()   // + font/layout config
                if err := mdpdf.ConvertMarkdownFileToPDF(ctx, tmp, pdf, opts); err != nil:
                    job.Fail(i, err); continue
            else if isPDF(tmp):
                pdf = tmp
            else:
                job.Fail(i, unsupportedType); continue
            job.Set(i, "uploading")
            s.withTree(func(c api.ApiCtx) error {
                dirNode, err := ensureDir(c, destDir)      // NodeByPath or MkdirAll
                if err != nil { return err }
                doc, err := c.UploadDocument(dirNode.Id(), pdf, true, nil, nil, nil, nil)
                if err != nil { return err }
                c.Filetree().AddDocument(doc)
                return nil
            })
            job.Complete(i, pdf)
            defer cleanup(tmp, pdf)
    return job.ID
```

Concurrency is bounded by `Config.Workers`, but only *one* upload reaches the
`withTree` critical section at a time (the mutex guarantees it). Conversion runs
in parallel; the tree lock is held for the short upload window only.

### 6.7 `ensureDir` (create nested folders)

```text
func ensureDir(c api.ApiCtx, path) (*model.Node, error):
    if node, err := c.Filetree().NodeByPath(path, nil); err == nil:
        if !node.IsDirectory(): return nil, ErrNotADir
        return node, nil
    // create missing segments left to right
    cur := c.Filetree().Root()
    for seg in split(path):
        if next, err := cur.FindByName(seg); err == nil:
            cur = next; continue
        doc, err := c.CreateDir(cur.Id(), seg, true)   // notify=true
        if err != nil: return nil, err
        c.Filetree().AddDocument(doc)                  // keep local tree consistent
        cur = c.Filetree().NodeById(doc.ID)
    return cur, nil
```

`rmcloud.MkdirAll` (`pkg/rmcloud/dirs.go:30`) already implements this idea for
the CLI; adapt it rather than inventing a second version.

### 6.8 Manage operations

| UI action | Service method | rmapi call | Safety |
| --- | --- | --- | --- |
| New folder | `CreateFolder(parentID, name)` | `CreateDir` + `AddDocument` | validate name |
| Rename | `Rename(id, newName)` | `MoveEntry(node, parent, newName)` | reject `/`, empty |
| Move | `Move(id, destDirID)` | `MoveEntry(node, dest, node.Name())` | reject move into self/subtree |
| Delete | `Delete(ids []string)` | `DeleteEntry(node, recursive, true)` | explicit confirm; reuse `rm --yes` semantics |
| Download | `Download(id, outDir)` | `rmcloud.DownloadDocumentByPath` or `FetchDocument` | path traversal guard |
| Refresh | `Refresh(ctx)` | `apiCtx.Refresh()` + reindex | serialized |

All mutations go through `withTree`, which calls `SyncComplete()` and reindexes.

### 6.9 macOS 1 visual system (the design language)

The goal is the **feel** of 1984 Macintosh without pretending to be a windowed
OS. Concretely:

**Principles.**

- **1-bit base.** Page is white; ink is black. Selection *inverts* (black
  rectangle, white text) instead of using a translucent highlight.
- **Square everything.** `border-radius: 0`. No shadows, no gradients except
  literal 2-color dithers.
- **Hairlines.** 1px solid black borders and rules define structure.
- **Dither as texture.** Separators and headers use a 50% checker via
  `repeating-conic-gradient` or a tiny data-URI pattern, evoking MacPaint.
- **Compact, dense rows.** Small type, tight leading, monospace for numeric
  metadata.
- **Modern fonts.** Use a contemporary grotesque (e.g. system UI stack,
  `Inter`, or `Space Grotesk`) and a modern mono (`JetBrains Mono`/`ui-monospace`).
  Do **not** use the classic Macintosh bitmap typeface. The retro feel comes
  from *layout and contrast*, never from a period font.
- **Color only as text highlight.** Accent color is applied to *text* (links,
  active filters, destructive labels), never as a big fill. This is the user's
  explicit instruction.

**Proposed tokens** (`frontend/src/styles/tokens.css`):

```css
:root {
  --mq-paper: #ffffff;
  --mq-ink:   #000000;
  --mq-ink-soft: #3a3a3a;        /* secondary text, still monochrome */

  /* text-highlight accents only */
  --mq-accent:   #0b5fff;        /* links, active nav, selection label */
  --mq-danger:   #c1272d;        /* destructive text */
  --mq-success:  #1a7f37;        /* success text */
  --mq-warn:     #b26a00;        /* warning text */

  --mq-rule: 1px solid var(--mq-ink);
  --mq-font: "Inter", system-ui, -apple-system, "Helvetica Neue", sans-serif;
  --mq-mono: ui-monospace, "JetBrains Mono", "SFMono-Regular", monospace;

  --mq-dither: repeating-conic-gradient(var(--mq-ink) 0% 25%, var(--mq-paper) 0% 50%)
               0 0 / 4px 4px;   /* 50% checker */
}
```

**Component sketches.**

```text
Toolbar (no window chrome, no menu bar)
+---------------------------------------------------------------------+
| REMARQUEE FILES        [ Upload ] [ New Folder ] [ Refresh ]        |  <- hairline bottom
| search: [______________________________________________]  (a)      |
+---------------------------------------------------------------------+
| / root / ai / 2026 / 10 / 04                     12 items          |  <- dithered strip
+---------------------------------------------------------------------+

File list (rows are 1px-bordered; selection inverts)
+---------------------------------------------------------------------+
| NAME                              TYPE      MODIFIED      ACTIONS   |
| Archive                           folder    ---            ...       |
| Selfish-Private-Pilgrim.pdf       document  2026-09-12     ...       |
| * Notes                           folder    2026-10-01     ...       |  <- selected row:
+---------------------------------------------------------------------+     black bg/white text

Detail pane (right)                     Danger dialog (typed confirm)
+---------------------------+           +--------------------------------+
| NAME  Notes                |           | DELETE 3 ITEMS?                |
| ID    6f2a…                |           | This cannot be undone.         |
| PATH  /ai/2026/10/04/Notes |           | type DELETE to confirm:        |
| TYPE  CollectionType       |           | [________________]             |
| MOD   2026-10-01T…         |           | [ Cancel ]  [ DELETE ]         |
+---------------------------+           +--------------------------------+
```

**CSS mechanics that sell the look.**

- Selection: `.row[aria-selected="true"] { background: var(--mq-ink); color: var(--mq-paper); }`
  and invert any accent inside selection to avoid unreadable color-on-black.
- Header strip: `.strip { background: var(--mq-dither); }`.
- Focus: never remove outlines; use `outline: 1px solid var(--mq-ink); outline-offset: 1px`.
- Buttons: flat, 1px black border, uppercase label with slight letter-spacing,
  invert on `:active`.
- No border-radius, no box-shadow, no transition longer than 80ms.

**Accessibility note.** 1-bit monochrome is high contrast by construction, but
color-highlighted text must still be distinguishable without color (add a glyph
or underline), and dithered backgrounds must never sit behind body text.

### 6.10 Frontend structure

> **Implemented as a no-build static app.** The layout below is the React/Vite
> proposal for context; the shipped frontend lives at
> `cmd/remarquee/cmds/serve/frontend/` as `index.html`, `styles.css`, and
> `app.js` (vanilla ES module), embedded with `//go:embed frontend`. It provides
> the same screens (list, breadcrumbs, search, detail pane, upload with job
> polling, new folder, rename/move, typed delete confirmation, status banner)
> without a build step.

```text
frontend/                 # colocated at cmd/remarquee/cmds/serve/frontend
  index.html
  src/
    main.tsx
    App.tsx
    api/client.ts          # typed fetch wrappers, one per endpoint
    store/
      store.ts
      filesSlice.ts        # entries, currentPath, selection, status
      searchSlice.ts       # query, results, ranking
      uploadsSlice.ts      # job polling
    components/
      FileList.tsx
      Breadcrumbs.tsx
      SearchBar.tsx
      UploadDropzone.tsx
      ManageToolbar.tsx
      DetailPane.tsx
      ConfirmDeleteDialog.tsx
      StatusBanner.tsx     # auth required / pandoc missing / errors
    styles/
      tokens.css
      base.css             # reset + macOS-1 primitives (.rule, .strip, .btn)
```

State management: Redux Toolkit is already the repo norm (`remarquee-ui` uses
it). Keep slices small; use `createAsyncThunk` for API calls and a polling thunk
for upload jobs. If the app stays small, plain React state is acceptable, but
consistency with the existing UI argues for Redux Toolkit.

**API client contract** (typed):

```ts
export type Entry = { id: string; parentId: string; name: string; path: string;
  isDir: boolean; type: string; version: number;
  modifiedClient: string; modifiedTime: string };

export async function listFiles(dir = "/"): Promise<{ path: string; entries: Entry[] }>;
export async function searchFiles(q: string, limit = 50): Promise<{ entries: Entry[] }>;
export async function createFolder(parentId: string, name: string): Promise<Entry>;
export async function uploadFiles(files: File[], destDir: string): Promise<{ jobId: string }>;
export async function getUpload(jobId: string): Promise<UploadJob>;
export async function deleteEntries(ids: string[], confirm: string): Promise<void>;
export async function renameEntry(id: string, name: string): Promise<Entry>;
```

### 6.11 Package for dev vs prod

- **Dev:** `remarquee serve --dev` on `:8080` with static caching disabled. (The
  shipped frontend is a no-build static app, so there is no separate Vite dev
  server; edit `frontend/` and reload.)
- **Prod:** the same `go build ./cmd/remarquee` embeds `frontend/` via
  `go:embed`; no node/pnpm/Dagger step is required. Users get one `remarquee`
  binary with a `serve` command.

### 6.12 Security posture

- Bind to `127.0.0.1` by default (`-addr 127.0.0.1:8080`). Adding `0.0.0.0`
  should be an explicit, warned choice.
- No CORS by default (same-origin). If dev proxy is used, the proxy is
  same-origin from the browser's perspective.
- CSRF: because the server can perform destructive cloud operations, require a
  same-origin check (verify `Origin`/`Sec-Fetch-Site`) on mutating endpoints, or
  a per-session token injected into `index.html`. Localhost-only reduces but does
  not eliminate this (a malicious web page can still POST to `localhost`).
- Path traversal: never accept a client-provided filesystem path for download;
  only document IDs.
- Body size limits: cap multipart sizes (`http.MaxBytesReader`), cap job counts.

---

## 7. Part 5 — HTTP API reference (PROPOSED)

Base: `http://127.0.0.1:8080`. All bodies JSON unless noted. Errors use:

```json
{ "error": { "code": "not_found", "message": "..." } }
```

| Method | Path | Purpose | Success |
| --- | --- | --- | --- |
| GET | `/api/health` | liveness | `200 {"status":"ok"}` |
| GET | `/api/status` | auth + capability status | `200 Status` |
| GET | `/api/files?dir=/ai` | list a directory | `200 {path, entries}` |
| GET | `/api/files/tree` | full tree snapshot (for sidebar) | `200 {entries}` |
| GET | `/api/search?q=&limit=` | ranked search | `200 {entries}` |
| POST | `/api/folders` | create folder `{parentId, name}` | `201 Entry` |
| PATCH | `/api/entries/{id}` | rename `{name}` | `200 Entry` |
| POST | `/api/entries/{id}/move` | move `{destDirId}` | `200 Entry` |
| DELETE | `/api/entries` | delete `{ids:[], confirm:"DELETE"}` | `200 {deleted:n}` |
| GET | `/api/entries/{id}/download` | stream `.rmdoc` | `200 application/octet-stream` |
| POST | `/api/uploads` | multipart upload | `202 {jobId}` |
| GET | `/api/uploads/{jobId}` | job status | `200 UploadJob` |
| POST | `/api/uploads/{jobId}/cancel` | cancel | `200 UploadJob` |
| POST | `/api/refresh` | force cloud resync | `202 {jobId}` or `200` |

Type sketches:

```go
type Status struct {
    Authenticated   bool      `json:"authenticated"`
    User            string    `json:"user,omitempty"`
    SyncVersion     string    `json:"syncVersion,omitempty"`
    DocumentCount   int       `json:"documentCount"`
    LastRefresh     time.Time `json:"lastRefresh,omitempty"`
    PandocAvailable bool      `json:"pandocAvailable"`
}

type UploadJob struct {
    ID       string        `json:"id"`
    State    string        `json:"state"`   // queued|converting|uploading|done|failed|cancelled
    Total    int           `json:"total"`
    Done     int           `json:"done"`
    Items    []UploadItem  `json:"items"`
}
type UploadItem struct {
    Name    string `json:"name"`
    State   string `json:"state"`
    Error   string `json:"error,omitempty"`
    EntryID string `json:"entryId,omitempty"`
}
```

Status → HTTP mapping: `ErrAuthRequired`→401, `ErrNotFound`→404,
`ErrConflict`→409 (name exists), `ErrInvalid`→400, everything else→500 with a
generic message and a server-side log line.

---

## 8. Part 6 — Implementation plan (phased)

Estimates assume one engineer with the codebase open and the CLI working.

### Phase 0 — Spike the service (1 day)

- Add `pkg/rmfiles/entries.go` with `Entry` + path building.
- Write a tiny Go program (or test) that creates an `ApiCtx` once and dumps
  `WalkTree` to JSON. Confirm field values against `remarquee cloud ls
  --with-glaze-output --format json`.
- **Exit criteria:** you can print every file/folder as JSON, and you understand
  the tree cache behavior on second run (fast) vs. first (slow).

### Phase 1 — Service + read endpoints (1–2 days)

- Implement `Service` with the single-context + mutex discipline.
- `GET /api/status`, `/api/files`, `/api/files/tree`, `/api/search`.
- Unit tests with a fake `ApiCtx`.
- **Exit criteria:** `curl /api/files?dir=/` returns the same entries as the CLI,
  and search ranks a known document first.

### Phase 2 — Frontend shell + macOS-1 primitives (1–2 days)

- Scaffold Vite + React + TS (copy `remarquee-ui/frontend` config).
- Implement `tokens.css`, `base.css`, `FileList`, `Breadcrumbs`, `SearchBar`,
  `StatusBanner`.
- Dev proxy `/api` → `:8080`.
- **Exit criteria:** browser shows a monochrome file list with inverted
  selection, hairline rules, and working search against the real cloud.

### Phase 3 — Upload jobs (2–3 days)

- `POST /api/uploads` multipart, job store, worker bound.
- Single `.md` and single `.pdf` first; then directory upload + bundle.
- Frontend `UploadDropzone` + job polling UI.
- **Exit criteria:** drop a `.md`, watch it become a PDF-backed cloud document at
  the chosen remote dir, visible after refresh.

### Phase 4 — Manage operations (1–2 days)

- Create folder, rename, move, delete (with confirm), download.
- Guardrails: typed-delete confirmation; overwrite warning; move-into-self
  rejection.
- **Exit criteria:** all manage actions round-trip and the search index stays
  consistent without a manual refresh.

### Phase 5 — Packaging + polish (1 day)

- Add `serve/embed.go`, `serve/gen.go`, `serve/command.go`, and wire the
  subcommand into `cmd/remarquee/main.go`; reuse/parametrize the Dagger/pnpm
  helper.
- Graceful shutdown; `--addr` flag; `127.0.0.1` default.
- Empty/error/loading states; pandoc-missing banner.
- **Exit criteria:** `go build ./cmd/remarquee` produces one `remarquee` binary
  whose `remarquee serve` command serves the full UI.

### Phase 6 — Validation + docs (0.5–1 day)

- `go test ./pkg/rmfiles/... ./cmd/remarquee/cmds/serve/...`.
- Manual smoke script (tmux) documented in the ticket.
- Update this ticket's diary and index; upload the guide to reMarkable.

---

## 9. Part 7 — Testing strategy

**Unit tests (fast, no network).** Define an interface narrow enough to fake:

```go
type treeLister interface { Filetree() *filetree.FileTreeCtx }
```

Test path building, fuzzy scoring, `ensureDir` segment logic, error→status
mapping, and job state transitions. For tree logic, build a synthetic
`model.Node` graph by hand (no cloud needed).

**Handler tests.** `net/http/httptest` against the mux, mirroring
`cmd/remarquee-ui/main_test.go`'s pattern (`newTestMux`). Assert JSON shapes and
status codes; do not assert on cloud state.

**Integration tests (opt-in).** Guard with an env var such as
`RMQ_WEB_INTEGRATION=1` plus valid tokens; these create a temp remote folder,
upload, search, rename, and delete. Never run destructive integration tests
against a real user root.

**Frontend tests.** Vitest + React Testing Library for components; snapshot the
rendered list markup to catch accidental chrome (rounded corners, shadows).

**Manual smoke (tmux).**

```bash
tmux new -d -s rmqweb 'go run ./cmd/remarquee serve --dev'
tmux send-keys -t rmqweb '' Enter
curl -s localhost:8080/api/health
curl -s 'localhost:8080/api/files?dir=/' | head
# browser: upload a .md, search for it, rename, delete
lsof-who -p 8080 -k    # kill cleanly when done
```

**What "done" looks like:** all unit tests pass; `go build ./...` succeeds;
`go build ./cmd/remarquee` produces a single `remarquee` binary whose `serve`
subcommand runs the UI; the manual smoke sequence works; the ticket diary
records the commands and results.

---

## 10. Part 8 — Edge cases and gotchas

- **Second run is fast, first is slow.** Document this in the UI ("syncing…").
- **`forceSchemaV4` must remain.** If you bypass `rmcloud.CreateApiCtx`, you will
  hit HTTP 400 invalid-hash. Always go through the package.
- **Never hold the tree mutex across pandoc.** Convert outside the lock.
- **Document IDs, not paths, are primary keys.** A rename changes paths.
- **Template documents** are hidden by CLI `ls`/`find` by default
  (`model.TemplateType`). Decide whether the UI shows them (proposal: hidden
  behind a toggle).
- **Filename sanitization.** `upload md` sanitizes PDF names
  (`sanitizePDFName`, spaces → underscores). Reuse it so UI uploads don't 400.
- **`--content-only` vs `--force`.** Default to content-only / new-name; make
  full overwrite a deliberate, confirmed action because it destroys annotations.
- **Move into own subtree** must be rejected (would detach the subtree).
- **Empty directories / trailing slashes** differ between `NodeByPath` and
  `NodesByPath(ignoreTrailingSlash)`. Normalize paths in the service.
- **Pandoc/XeLaTeX missing** → uploading `.md` fails; `/api/status` should report
  `pandocAvailable` so the UI can explain instead of erroring opaquely.
- **Concurrent browser tabs** share one server and one tree — that is correct;
  the mutex serializes them. Do not create per-tab contexts.
- **Long-lived token expiry** mid-session → rely on `WithAuthRetry`; if it fails,
  surface the `auth_required` state.
- **Cloud eventual consistency.** After an upload, the tablet may not show the
  file until its own sync; the UI should not claim "synced to device", only
  "uploaded to cloud".

---

## 11. Part 9 — Open questions / decisions to raise

1. **Command shape (RESOLVED).** The UI ships as **`remarquee serve`**, a
   subcommand under `cmd/remarquee/cmds/serve`, not a separate `remarquee-web`
   binary. Rationale: one binary, one install path, discoverable help, and the
   embedded SPA rides along in the existing release.
2. **Shared frontend build helper.** Rename/parametrize
   `cmd/build-remarquee-ui-web` into a generic `cmd/build-web` used by both UIs,
   or duplicate. *Recommendation:* parametrize (env vars already exist:
   `RMQ_WEB_DIR`, `RMQ_WEB_DIST`).
3. **Index freshness.** Periodic background `Refresh()` (e.g., every 5 min) vs.
   manual only. *Recommendation:* manual + refresh after every mutation; add a
   configurable interval later.
4. **Real-time updates.** Polling vs. SSE/WebSocket for job progress.
   *Recommendation:* polling every 500ms for v1; SSE later.
5. **Download format.** `.rmdoc` archive vs. rendered PDF. *Recommendation:*
   `.rmdoc` (matches `cloud get`); rendering is `remarquee-ui`'s domain.
6. **Auth UX.** What should the UI do when `~/.rmapi` has no valid token?
   *Recommendation:* a dedicated status banner instructing the user to run
   `rmapi reset` / device registration in a terminal; do not attempt to embed
   the interactive code prompt.
7. **State management (RESOLVED).** The shipped frontend is vanilla ES modules
   with a small `state` object, not Redux Toolkit. This drops the node build
   toolchain entirely; the trade-off is fewer batteries for larger UI growth.
8. **Mobile/tablet.** Desktop-first; confirm whether a tablet width is needed in
   v1.

---

## 12. Appendix A — File and symbol index

| File | Why it matters |
| --- | --- |
| `go.mod` | Module path; note the `rmapi` `replace` on line 23. |
| `pkg/rmcloud/auth.go` | `AuthSettings`, `CreateApiCtx:67`, `IsAuthError:152`, `WithAuthRetry:172`, `forceSchemaV4:36`. |
| `pkg/rmcloud/dirs.go` | `MkdirAll:30`, `normalizeDirPath:12` — nested folder creation. |
| `pkg/rmcloud/download.go` | `DownloadDocumentByPath:23`, `DownloadedDocument`. |
| `pkg/rmcloud/progress.go` | Tree-sync progress reporting to a writer. |
| `cmd/remarquee/cmds/cloud/ls.go` | Structured row shape; `buildPathFromParents:305`. |
| `cmd/remarquee/cmds/cloud/find.go` | `WalkTree` traversal + regex. |
| `cmd/remarquee/cmds/cloud/search.go` | Search command semantics. |
| `cmd/remarquee/cmds/cloud/put.go` | Upload semantics, `--force`/`--content-only`. |
| `cmd/remarquee/cmds/cloud/rm.go` | Delete + `--yes` safety gate. |
| `cmd/remarquee/cmds/cloud/mv.go` | Rename/move via `MoveEntry`. |
| `cmd/remarquee/cmds/cloud/rmapi.go` | `createApiCtx` helper used by all cloud commands. |
| `cmd/remarquee/cmds/upload/md.go` | Markdown→PDF→upload pipeline (copy this). |
| `pkg/mdpdf/pandoc.go` | `PandocOptions`, `ConvertMarkdownFileToPDF`. |
| `pkg/mdpdf/bundle.go` | `BuildBundleMarkdown` for multi-file bundles. |
| `pkg/mdpdf/svg.go`, `mermaid.go` | Diagram/image handling. |
| `cmd/remarquee/main.go` | Root command; register the new `serve` subcommand here (`rootCmd.AddCommand(...)`). |
| `cmd/remarquee-ui/main.go` | Go server precedent: mux, timeouts, dev/prod. |
| `cmd/remarquee-ui/embed.go` | `//go:embed frontend/dist` + `fs.Sub`. |
| `cmd/build-remarquee-ui-web/main.go` | Dagger/pnpm build helper + local fallback. |
| `cmd/remarquee-ui/frontend/vite.config.ts` | `/api` dev proxy. |
| `cmd/remarquee-ui/frontend/src/store/*` | Redux slice precedent. |
| `AGENT.md` | Repo conventions (`lsof-who -p PORT -k`, go:embed, cobra, zerolog). |
| `ttmp/.../RMQ-RMDOC-WEB-001/` | Prior web-UI ticket: tasks + diary. |
| `.../rmapi/api/api.go:16` | `ApiCtx` interface (dependency). |
| `.../rmapi/filetree/filetree.go` | `NodeByPath`, `NodesByPath`, `AddDocument`, `DeleteNode`. |
| `.../rmapi/model/node.go`, `document.go` | Node/Document fields and type constants. |
| `.../rmapi/api/sync15/common.go:31` | `tree.cache` location and version. |
| `.../rmapi/config/config.go:21` | `~/.rmapi` / `RMAPI_CONFIG` resolution. |

---

## 13. Appendix B — One-page cheat sheet

**Stack:** Go `net/http` + `go:embed` SPA + React/TS/Vite + Redux Toolkit.

**Golden rule:** one `ApiCtx`, one mutex, `SyncComplete()` + reindex after every
mutation, never hold the lock across pandoc.

**Reuse, don't rewrite:**

- Auth/tree: `rmcloud.CreateApiCtx`, `rmcloud.WithAuthRetry`.
- Paths: copy `buildPathFromParents` (and centralize it).
- Upload: copy `cmd/remarquee/cmds/upload/md.go` + `pkg/mdpdf`.
- Packaging: copy `cmd/remarquee-ui` + `cmd/build-remarquee-ui-web`.

**Endpoints:** `/api/status`, `/api/files`, `/api/files/tree`, `/api/search`,
`/api/folders`, `/api/entries/{id}`, `/api/entries/{id}/move`,
`/api/entries/{id}/download`, `/api/uploads`, `/api/uploads/{jobId}`,
`/api/refresh`.

**Style:** `--radius:0`, white paper + black ink, invert selection, dither
separators, modern grotesque + mono, color only on text.

**Safety:** confirm deletes (typed), warn on overwrite (destroys annotations),
bind `127.0.0.1`, IDs not paths, no secrets to the browser.

**Definition of done:** unit tests green, `go build ./...` green,
`go build ./cmd/remarquee` yields one binary with a working `remarquee serve`,
manual smoke works, diary updated.

---

## 14. Appendix C — Evidence collected for this guide

Commands run while writing this ticket (read-only, at `02475f6`):

```text
$ git log --oneline -5
02475f6 Merge pull request #28 from go-go-golems/task/add-svg-support
...

$ go.mod
module github.com/go-go-golems/remarquee   (go 1.26.3)
replace github.com/juruen/rmapi => github.com/FNStudios-NI/rmapi v0.0.0-20260817154736-f295d5466978

$ cat cmd/remarquee-ui/embed.go
//go:embed frontend/dist  +  fs.Sub(...)

$ sed -n '1,205p' pkg/rmcloud/auth.go
CreateApiCtx retries 3x; forceSchemaV4 reflection workaround; WithAuthRetry.

$ grep -n "^func " .../rmapi/api/api.go
type ApiCtx interface { Filetree, FetchDocument, CreateDir, UploadDocument,
  ReplaceDocumentFile, MoveEntry, DeleteEntry, SyncComplete, Nuke, Refresh }

$ grep -n "getCachedTreePath" .../rmapi/api/sync15/common.go
cacheFile := path.Join(cachedir, "rmapi", "tree.cache"); cacheVersion = 3
```

No cloud mutations were performed while writing this document. All behavior
claims about the cloud are read from the pinned dependency source or from the
repository's own commands; anything untested is marked as proposal/open
question.

---

## 15. Appendix D — Notes for the reviewer

- The single most important design constraint is §4.4 (single `ApiCtx` +
  serialized mutations). If the implementation deviates, review it carefully.
- The macOS-1 style is a *visual* contract with explicit constraints from the
  user (modern fonts, no chrome, no menu bar, color only on text). Treat
  deviations (rounded corners, shadows, colored fills) as bugs.
- The phased plan is ordered so that each phase is independently demonstrable;
  resist building the upload UI before the read path and style primitives exist.
