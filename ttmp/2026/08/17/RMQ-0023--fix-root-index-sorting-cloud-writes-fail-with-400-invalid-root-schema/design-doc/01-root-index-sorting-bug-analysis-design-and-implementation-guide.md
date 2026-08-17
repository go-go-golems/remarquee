---
Title: 'Root index sorting bug: analysis, design, and implementation guide'
Ticket: RMQ-0023
Status: active
Topics:
    - backend
    - rmcloud
    - upload
    - dependency
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: abs:///home/manuel/go/pkg/mod/github.com/marcobarcelos/rmapi@v0.0.0-20260518211546-a0d079936d46/api/sync15/apictx.go
      Note: CreateDir/UploadDocument/DeleteEntry route through Sync() which uploads the root index
    - Path: abs:///home/manuel/go/pkg/mod/github.com/marcobarcelos/rmapi@v0.0.0-20260518211546-a0d079936d46/api/sync15/common.go
      Note: HashEntries() sorts document entries before hashing — the asymmetry that makes document indices canonical but root not
    - Path: abs:///home/manuel/go/pkg/mod/github.com/marcobarcelos/rmapi@v0.0.0-20260518211546-a0d079936d46/api/sync15/tree.go
      Note: HashTree.IndexReader() (bug site, no sort) + Mirror() (read path sorts) + Remove() (swap-delete)
    - Path: cmd/remarquee/cmds/cloud/mkdir.go
      Note: cloud mkdir command — repro/verification entrypoint
    - Path: cmd/remarquee/cmds/cloud/rm.go
      Note: cloud rm command — exercises the Remove() swap-delete write path
    - Path: cmd/remarquee/cmds/upload/md.go
      Note: upload md command — converts markdown to PDF then uploads (write path)
    - Path: cmd/remarquee/cmds/upload/upload_helpers.go
      Note: uploadPDFToRemoteWithAuthRetry — MkdirAll + UploadDocument write path
    - Path: go.mod
      Note: rmapi module + replace fork pin (line 22) — the fix lives here
    - Path: pkg/rmcloud/auth.go
      Note: CreateApiCtx, forceSchemaV4 (different bug's workaround), WithAuthRetry
    - Path: pkg/rmcloud/dirs.go
      Note: MkdirAll — first write path that fails (calls CreateDir per segment)
    - Path: pkg/rmcloud/download.go
      Note: DownloadDocumentByPath — read path, unaffected by the bug
    - Path: pkg/rmcloud/logtransport.go
      Note: loggingRoundTripper — how --log-level debug reveals the root.docSchema 400 body
ExternalSources:
    - https://github.com/go-go-golems/remarquee/issues/23
    - https://github.com/ddvk/rmapi/issues/75
    - https://github.com/ddvk/rmapi/issues/76
    - https://github.com/ddvk/rmapi/pull/77
Summary: All reMarkable cloud writes fail with HTTP 400 'invalid root schema' because the pinned rmapi fork serializes the root index in slice order; the cloud now requires entries sorted by document ID. Intern-level analysis, design, and implementation guide.
LastUpdated: 2026-08-17T00:00:00-04:00
WhatFor: 'Onboarding an intern to the remarquee cloud-sync stack and the root-index sorting bug (issue #23).'
WhenToUse: Read before touching pkg/rmcloud, the rmapi dependency pin, or any cloud write path.
---













# Root index sorting bug — analysis, design & implementation guide

> **Audience:** a new intern joining the `remarquee` project. This document assumes you can read Go and use `git`/`gh`, but assumes **nothing** about reMarkable, `rmapi`, or cloud sync. Everything you need to understand the bug and ship the fix is here, anchored to real files and line numbers.

---

## 0. Executive summary (read this first)

Since **2026-08-17**, every **write** to the reMarkable cloud through `remarquee` fails with HTTP 400 `{"message":"invalid root schema"}`. All **reads** keep working. The cause is one missing `sort.Slice` in a dependency we don't own.

- **Where the bug is not:** It is *not* in `remarquee` application code, *not* an auth problem, *not* a schema-version problem, *not* a cache problem. Those were all ruled out in the issue.
- **Where the bug is:** In the pinned `rmapi` fork (`github.com/marcobarcelos/rmapi`), in `api/sync15/tree.go`, `HashTree.IndexReader()`. That function serializes the **root index** in raw slice order. The cloud now requires root-index entries to be **strictly sorted by document ID**. Two mutators (`Add()` appends; `Remove()` swap-deletes) break the order, then only re-hash — so the next write uploads an unsorted root and gets a 400.
- **Why reads work:** the read path (`Mirror()`) *does* sort `t.Docs` before assigning it. So a freshly-mirrored tree is sorted; only a write mutates it back out of order.
- **The fix:** sort `t.Docs` by `DocumentID` at the single serialization point (`IndexReader()`), and the same for `BlobDoc.IndexReaderWithSchema()`. This keeps the uploaded body and its hash consistent **by construction** because `Rehash()` hashes exactly what `IndexReader()` emits. Upstream fix is ddvk/rmapi#77 (open at time of writing); we either bump our fork to a commit containing it, or apply the equivalent one-liner ourselves.
- **Scope of change in `remarquee` itself:** essentially zero lines of application code — a dependency bump in `go.mod` plus a rebuild. Optionally, surface the server's error body so future 400s are diagnosable without `--log-level debug`.

The rest of this document explains *why* all of that is true, with the full system context an intern needs.

---

## 1. Problem statement & scope

### 1.1 The symptom (from issue #23)

Every cloud write command fails the same way:

```
$ remarquee cloud mkdir /ai/2026/08/17 --non-interactive
Error: failed to create directory: request failed with status 400

$ remarquee upload bundle a.md b.md --name "X" --remote-dir /ai/2026/08/17/ESP-55 --toc-depth 2 --non-interactive
ERR rmcloud mkdir-all: create remote directory failed error="request failed with status 400" entry=17 parent_id=0f1a7d42-...

$ remarquee upload md README.md --name "probe" --remote-dir /ai/2026/08/16 --non-interactive
ERROR-UPLOAD: ... failed to upload file [...probe.pdf]: request failed with status 400
```

Reads are unaffected:

```
$ remarquee cloud ls /ai/2026/08       # works
```

With `--log-level debug`, the failure is localized to exactly one HTTP call — the **final root PUT**. Every prior blob PUT (`*.metadata`, `*.content`, `*.docSchema`) returns 200/202; only the root upload is rejected:

```
DBG rmapi HTTP request ... header_rm-filename=["root.docSchema"] method=PUT url=https://internal.cloud.remarkable.com/sync/v3/files/<hash>
DBG rmapi HTTP response response_body="{\"message\":\"invalid root schema\"}\n" status=400 status_text="400 Bad Request"
```

### 1.2 Scope

- **Affected:** all cloud *mutations* — `cloud mkdir`, `upload md`, `upload bundle`, and by extension `cloud rm` / `cloud mv` (they mutate the same root index).
- **Not affected:** all cloud *reads* — `cloud ls`, `cloud get`, `cloud stat`, `cloud find`, `cloud search`, `cloud account`, `cloud version`.
- **Not the cause:** auth (reads succeed; `--reauth` doesn't help), the local tree cache (rebuilding it doesn't help), schema version (the fork already emits v4; v3 roots were rejected earlier with a *different* message, "Software must be updated").

### 1.3 Provenance

| Field | Value |
|---|---|
| GitHub issue | [go-go-golems/remarquee#23](https://github.com/go-go-golems/remarquee/issues/23) |
| Upstream bug | [ddvk/rmapi#75](https://github.com/ddvk/rmapi/issues/75), [#76](https://github.com/ddvk/rmapi/issues/76) |
| Upstream fix PR | [ddvk/rmapi#77](https://github.com/ddvk/rmapi/pull/77) (open at time of writing) |
| First reported | 2026-08-17 |
| Branch | `task/remarquee-fix-root-schema` (currently == `origin/main` tip `183a9d3`; the fix is not yet in main) |

> [!info] Branch status (verified this session)
> `HEAD == origin/main == 183a9d3`. The bug is **still live at the tip of main**: `go.mod:22` still pins the pre-fix fork. ddvk/rmapi#77 is still `OPEN` (`mergedAt: null`). So this ticket documents a real, unfixed bug — not a solved one.

---

## 2. Background: the system an intern needs to know

Before the bug, you need a mental model of three layers: **reMarkable cloud**, **rmapi**, and **remarquee**. They nest like this:

```
┌─────────────────────────────────────────────────────────────┐
│ remarquee (this repo, Go CLI)                               │
│  cmd/remarquee/cmds/{cloud,upload,...}  +  pkg/rmcloud      │
│        │                                                     │
│        │ uses (via go.mod replace pin)                       │
│        ▼                                                     │
│ rmapi  (github.com/juruen/rmapi, forked to marcobarcelos)    │
│  api/sync15/{tree,blobdoc,blobstorage,apictx,common}.go     │
│        │                                                     │
│        │ HTTPS to                                            │
│        ▼                                                     │
│ reMarkable cloud  (sync/v3 REST + blob storage)             │
└─────────────────────────────────────────────────────────────┘
```

### 2.1 What `remarquee` is

`remarquee` is a Go CLI ("a practical toolkit for getting content into, out of, and around your reMarkable"). Five focus areas (per `README.md`):

- **Cloud filesystem workflows** — browse/upload/download/move/delete cloud documents. ← *this bug lives here*
- Markdown & source upload (pandoc/xelatex → PDF → upload).
- `.rmdoc` inspection and rendering (legacy/V6 notebooks → PDF/PNG).
- On-device capture (framebuffer/pen/gesture capture server).
- RMDoc-DSL fixtures (YAML/JS document fixtures for renderer testing — unrelated to cloud sync).

It is a **Cobra** + **Glazed** CLI. Cloud commands live under `cmd/remarquee/cmds/cloud/`; upload commands under `cmd/remarquee/cmds/upload/`. The thin Go glue to `rmapi` lives in `pkg/rmcloud/`.

> [!note] Don't confuse the two DSLs
> `remarquee` has a `pkg/rmdsl` (RMDoc-DSL) that compiles *document fixtures* into reMarkable v6 `.rm` page archives (pages/layers/strokes/glyphs). That is a **rendering** DSL and has nothing to do with cloud sync or this bug. When we say "schema" in this doc, we mean the **sync root-index schema**, not the RMDoc-DSL.

### 2.2 What `rmapi` is

`rmapi` (`github.com/juruen/rmapi`, actively maintained as `ddvk/rmapi`) is a third-party Go client for the reMarkable cloud sync API. `remarquee` does **not** talk to reMarkable directly; it talks through `rmapi`'s `api/sync15` package.

`remarquee` does **not** depend on upstream `juruen/rmapi` directly at a useful version. Look at `go.mod`:

```
# go.mod (excerpt)
 13:   github.com/juruen/rmapi v0.0.25
 ...
 22:   replace github.com/juruen/rmapi => github.com/marcobarcelos/rmapi v0.0.0-20260518211546-a0d079936d46
```

- **Line 13** declares the module path `github.com/juruen/rmapi` (so all `import "github.com/juruen/rmapi/..."` in remarquee resolves to that path).
- **Line 22** is a Go `replace` directive that redirects that import path to a **fork**: `github.com/marcobarcelos/rmapi` at the May 18, 2026 commit `a0d079936d46`.

> [!important] The bug is in the fork, not in remarquee
> The pinned fork predates the fix. `remarquee` code only *wraps* the fork (`pkg/rmcloud/logtransport.go` adds debug logging; `pkg/rmcloud/auth.go` adds a reflection workaround for a *different* bug). No `remarquee` source file serializes the root index. So the fix is primarily a **dependency change**, not an application change.

You can find the pinned fork on disk in the module cache:

```
$(go env GOMODCACHE)/github.com/marcobarcelos/rmapi@v0.0.0-20260518211546-a0d079936d46
```

All fork file references below (e.g. `api/sync15/tree.go`) resolve under that path.

### 2.3 The reMarkable cloud sync model (what an intern must understand)

This is the core conceptual model. Read it carefully; the bug is a violation of an invariant in this model.

reMarkable cloud is a **content-addressed blob store** with a **root index**. There are three levels of "index":

```
                     ┌──────────────────────────────────────┐
                     │ 1. ROOT INDEX  (root.docSchema)       │
                     │    one entry per DOCUMENT in the cloud │
                     │    <hash>:0:<DocumentID>:<subfiles>:<size>   │
                     └───────────────┬──────────────────────┘
                                     │ points at, per doc
                                     ▼
            ┌────────────────────────────────────────────────┐
            │ 2. DOCUMENT INDEX  (<DocumentID>.docSchema)     │
            │    one entry per FILE inside a document         │
            │    <hash>:0:<fileID>:<subfiles>:<size>          │
            └───────────────┬────────────────────────────────┘
                            │ points at, per file
                            ▼
            ┌────────────────────────────────────────────────┐
            │ 3. BLOBS  (the actual bytes: .metadata,        │
            │    .content, .pdf, .rm pages, ...)              │
            └────────────────────────────────────────────────┘
```

- **Blobs** are uploaded by content hash (`PUT /sync/v3/files/{hash}` with header `rm-filename: {name}`). Uploading the same bytes twice is a no-op because the URL is the hash.
- **A document** is a small set of files: a `.metadata` (JSON: name, parent, type, timestamps), a `.content` (JSON: tags, page info), and zero or more page/content files (e.g. `.pdf`, `.rm`). Each document has its own **document index** (`<DocumentID>.docSchema`) listing those files with their hashes.
- **The root index** (`root.docSchema`) is the top-level table of contents: one line per document, giving that document's hash, ID, subfile count, and total size.
- **The root hash** is the SHA-256 of the serialized root index *body*. The cloud tracks the current root hash plus a monotonically increasing **Generation** number. A "write" = upload any new blobs + upload the new root index blob + `PUT /sync/v3/.../root` (the `RootPut` endpoint) with `{hash, generation}` to advance the generation.

#### The exact root-index wire format (schema v4)

The root index is a tiny text file. Schema 4 layout (serialized by `HashTree.IndexReader()`, fork `api/sync15/tree.go`):

```
4                                     <- line 1: schema version
0:.<count>:<totalSize>                <- line 2 (v4 only): summary line
<hash>:0:<DocumentID>:<subfiles>:<size>   <- one line per document
<hash>:0:<DocumentID>:<subfiles>:<size>
...
```

- Line 1 is literally the string `4`.
- Line 2 (v4 only) is `0:.<N>:<S>` where `N` = number of docs, `S` = sum of all doc sizes.
- Each subsequent line is colon-separated: `<sha256-hex>` `:` `0` (file type) `:` `<DocumentID>` `:` `<subfile count>` `:` `<total size in bytes>`.
- The **root hash** = `SHA-256` of this entire file's bytes (including the `4\n` and the summary line).

> [!important] The invariant the cloud now enforces
> As of 2026-08-17, the cloud **rejects** a `root.docSchema` whose document lines are not in **strictly ascending order by `DocumentID`**. The rejection is `400 {"message":"invalid root schema"}` on the root PUT. This is the invariant our fork violates.

---

## 3. Current-state architecture (evidence-based)

Let's walk the actual code path from a CLI command to the failing HTTP call. Every reference is real.

### 3.1 The `remarquee` cloud write commands

All cloud commands share an auth entry point. Example: `cloud mkdir`.

**`cmd/remarquee/cmds/cloud/mkdir.go`** — `MkdirCommand.Run`:
```go
_, apiCtx, err := createApiCtx(s.AuthSettings)   // pkg/rmcloud: builds rmapi ApiCtx
...
document, err := apiCtx.CreateDir(parentId, newDir, true)   // <- rmapi fork
apiCtx.Filetree().AddDocument(document)
```

`createApiCtx` is the remarquee wrapper that:
1. Bootstraps rmapi auth (`api.AuthHttpCtx`).
2. Builds the rmapi context (`api.CreateApiCtx`) — this **mirrors** the remote tree into a local cache.
3. Calls `forceSchemaV4(apiCtx)` (a reflection workaround — see §3.4).
4. Wraps the HTTP transport with debug logging (`WrapTransportWithLogging`).

**`cmd/remarquee/cmds/cloud/rm.go`** — `RmCommand.Run` calls `apiCtx.DeleteEntry(node, s.Recursive, true)`.

**`cmd/remarquee/cmds/upload/md.go`** — `runUploadMarkdown` converts markdown → PDF, then per file calls `uploadPDFToRemoteWithAuthRetry(...)`.

**`cmd/remarquee/cmds/upload/upload_helpers.go`** — the helper that actually mutates the cloud:
```go
func uploadPDFToRemoteWithAuthRetry(...) (api.ApiCtx, error) {
    return rmcloud.WithAuthRetry(authSettings, apiCtx, func(currentCtx api.ApiCtx) (api.ApiCtx, error) {
        dstNode, err := rmcloud.MkdirAll(currentCtx, dst)      // may create dirs (writes!)
        ...
        document, err := currentCtx.UploadDocument(dstNode.Id(), outPDF, true, ...)  // write
        currentCtx.Filetree().AddDocument(document)
        ...
    })
}
```

So every upload first **`MkdirAll`** (which creates directories = writes), then `UploadDocument` (another write). Either can hit the 400. That's why the issue's `mkdir` and `upload` examples both fail at the *first* write.

### 3.2 `pkg/rmcloud` — the thin remarquee glue

- **`pkg/rmcloud/dirs.go`** — `MkdirAll(apiCtx, dirPath)`: walks the path, and for each missing segment calls `apiCtx.CreateDir(parentId, entry, true)`. `CreateDir` is the rmapi-fork function that performs a root write.
- **`pkg/rmcloud/logtransport.go`** — `loggingRoundTripper`: wraps the HTTP client so every request/response is logged at debug level. This is why `--log-level debug` reveals the `root.docSchema` PUT and its `400` body. It re-reads the response body fully (so downstream rmapi code still sees it) and logs the first 4096 bytes.
- **`pkg/rmcloud/download.go`** — `DownloadDocumentByPath`: a read path (`apiCtx.FetchDocument`). Reads never touch `Sync`/`IndexReader`, so they are unaffected.
- **`pkg/rmcloud/auth.go`** — `CreateApiCtx`, `WithAuthRetry`, `IsAuthError`, and `forceSchemaV4`. See §3.4.

### 3.3 The rmapi fork write path (where the bug lives)

This is the heart of the document. We follow a single `CreateDir` to the failing PUT. All paths below are in the **fork** at `$(go env GOMODCACHE)/github.com/marcobarcelos/rmapi@v0.0.0-20260518211546-a0d079936d46`.

#### 3.3.1 `ApiCtx.CreateDir` — `api/sync15/apictx.go`

```go
func (ctx *ApiCtx) CreateDir(parentId, name string, notify bool) (*model.Document, error) {
    id := uuid.New().String()
    // ... build .metadata + .content files, upload each as a blob ...
    doc := NewBlobDoc(name, id, model.DirectoryType, parentId)
    for _, f := range files.Files {
        // ... upload blob ...
        doc.AddFile(fileEntry)   // appends to doc.Files, then doc.Rehash()
    }
    // upload the document index (<id>.docSchema)
    indexReader, _ := doc.IndexReader()
    ctx.blobStorage.UploadBlob(doc.Hash, addExt(doc.DocumentID, archive.DocSchemaExt), indexReader)

    // *** the root write ***
    err = Sync(ctx.blobStorage, ctx.hashTree, func(t *HashTree) error {
        return t.Add(doc)         // <-- mutator #1: APPEND
    }, notify)
    ...
}
```

`UploadDocument` is structurally identical: build a `BlobDoc`, upload its files + its doc index, then `Sync(..., t.Add(doc), ...)`.

`DeleteEntry` calls `Sync(..., t.Remove(node.Document.ID), ...)` — mutator #2.

`MoveEntry` calls `Sync(..., <rehash the moved doc + the tree>, ...)` without Add/Remove.

#### 3.3.2 `Sync` — `api/sync15/apictx.go`

`Sync` is the generic "apply a mutation to the local tree, then push the new root to the cloud" routine. Simplified:

```go
func Sync(b *BlobStorage, tree *HashTree, operation func(*HashTree) error, notify bool) error {
    for syncTry := 1; syncTry <= 10; syncTry++ {
        err := operation(tree)            // (A) mutate: Add / Remove / rehash
        if err != nil { return err }

        indexReader, _ := tree.IndexReader()                       // (B) serialize root body
        b.UploadBlob(tree.Hash, "root.docSchema", indexReader)     // (C) PUT root blob by hash

        newGeneration, err := b.WriteRootIndex(tree.Hash, tree.Generation, notify)  // (D) advance generation
        if err == nil { tree.Generation = newGeneration; break }
        if err != transport.ErrWrongGeneration { return err }     // real error -> bail
        tree.Mirror(b, concurrent)                                // (E) someone else wrote; re-sync and retry
    }
    return saveTree(tree)
}
```

Step (C) is the `PUT /sync/v3/files/<hash>` with `rm-filename: root.docSchema` that the issue shows returning 400. Step (D) (`WriteRootIndex`) is the `PUT` to `RootPut` with `{hash, generation}` — it would also fail, but we never get there because (C) already returned 400.

> [!note] `tree.Hash` must equal `SHA-256(body from step B)`
> `Rehash()` (called inside the mutators) computes the root hash by reading `IndexReader()` and SHA-256-ing it. So the hash and the uploaded body are always *consistent with each other*. The bug is not a hash/body mismatch — it's that the *body itself* is invalid (unsorted), even though it's correctly hashed.

#### 3.3.3 The root serializer — `api/sync15/tree.go`, `HashTree.IndexReader()`

This is the buggy function. Verbatim, the relevant part:

```go
func (t *HashTree) IndexReader() (io.Reader, error) {
    var w bytes.Buffer
    schemaVersion := SchemaVersionV4            // always v4 for writes
    if envSchema := os.Getenv("RMAPI_FORCE_SCHEMA_VERSION"); envSchema != "" {
        schemaVersion = envSchema
    }
    w.WriteString(schemaVersion); w.WriteString("\n")

    if schemaVersion == SchemaVersionV4 {
        totalSize := int64(0)
        for _, d := range t.Docs { totalSize += d.Size }
        w.WriteString("0:.")
        w.WriteString(strconv.Itoa(len(t.Docs)))
        w.WriteString(":")
        w.WriteString(strconv.FormatInt(totalSize, 10))
        w.WriteString("\n")
    }

    for _, d := range t.Docs {                   // <-- BUG: iterates in SLICE order, no sort
        w.WriteString(d.LineWithSchema(schemaVersion))
        w.WriteString("\n")
    }
    return bytes.NewReader(w.Bytes()), nil
}
```

There is **no `sort.Slice`** here. `t.Docs` is emitted in whatever order the slice happens to be in.

#### 3.3.4 The two mutators that unsort the slice

**`HashTree.Add`** — `api/sync15/blobdoc.go`:
```go
func (t *HashTree) Add(d *BlobDoc) error {
    if len(d.Files) == 0 { return errors.New("no files") }
    t.Docs = append(t.Docs, d)      // <-- APPEND: new doc goes to the end
    return t.Rehash()               // re-hash only; does not sort
}
```
If the new document's ID doesn't sort *last*, the slice is now unsorted. (A freshly-mirrored tree is sorted; `Add` breaks that.)

**`HashTree.Remove`** — `api/sync15/tree.go`:
```go
func (t *HashTree) Remove(id string) error {
    docIndex := -1
    for index, d := range t.Docs {
        if d.DocumentID == id { docIndex = index; break }
    }
    if docIndex > -1 {
        length := len(t.Docs) - 1
        t.Docs[docIndex] = t.Docs[length]   // swap last into the removed slot
        t.Docs = t.Docs[:length]            // truncate
        t.Rehash()                          // re-hash only; does not sort
        return nil
    }
    return fmt.Errorf("%s not found", id)
}
```
This is a classic swap-delete: `aaa,bbb,ccc,ddd` minus `bbb` becomes `aaa,ddd,ccc` — unsorted.

#### 3.3.5 Why reads are fine — `HashTree.Mirror()` *does* sort

`api/sync15/tree.go`, end of `Mirror()`:
```go
    sort.Slice(head, func(i, j int) bool { return head[i].DocumentID < head[j].DocumentID })
    t.Docs = head
    t.Generation = gen
    t.Hash = rootHash
    return nil
```

So whenever the tree is *mirrored from the server*, `t.Docs` is sorted. That's why a tree that has only ever been read from (never locally mutated) happens to be sorted — and why the bug is **intermittent across users**: a user whose document IDs happen to already be in sorted order, or who only ever appends IDs that sort last, won't hit it. The moment an `Add` or `Remove` reorders the slice, the next write 400s.

### 3.4 An adjacent workaround that is *not* this bug (don't be fooled)

`pkg/rmcloud/auth.go` contains `forceSchemaV4(apiCtx)`, called from `CreateApiCtx`. It uses reflection + `unsafe.Pointer` to set the unexported `hashTree.SchemaVersion` to `"4"` when empty. The comment explains why:

```go
// forceSchemaV4 ... works around the rmapi bug where an empty SchemaVersion
// defaults to V3, causing 400 "invalid hash" from the reMarkable cloud.
```

> [!warning] Different bug, different message
> That workaround addresses a *different* rmapi defect (empty `SchemaVersion` → v3 body → **400 "invalid hash"**). Our bug is a v4 body that is *unsorted* → **400 "invalid root schema"**. `forceSchemaV4` is already in place and does not fix issue #23. Do not confuse the two: one is about *which schema version* is emitted; the other is about the *ordering of entries within* the (already-v4) body.

The reflection/`unsafe` pattern in `forceSchemaV4` is also a useful precedent: it shows that `remarquee` has been willing to patch around rmapi bugs *without* modifying the fork, by reaching into unexported state. That matters for one of the fix options (§6.3).

### 3.5 The asymmetry that makes the bug subtle: document index *is* sorted

Here is the subtle part that an intern should not miss. The **document** index (`BlobDoc`) is canonicalized — but the **root** index is not. Compare:

`api/sync15/common.go`, `HashEntries()` (used by `BlobDoc.Rehash()`):
```go
func HashEntries(entries []*Entry) (string, error) {
    sort.Slice(entries, func(i, j int) bool { return entries[i].DocumentID < entries[j].DocumentID })  // <-- SORTS
    hasher := sha256.New()
    for _, d := range entries { ... }
    return hashStr, nil
}
```

`BlobDoc.AddFile()` calls `d.Rehash()` → `HashEntries()`, which sorts `d.Files` **in place as a side effect**. So by the time a document index is serialized by `BlobDoc.IndexReaderWithSchema()`, `d.Files` is usually already sorted — *as a side effect of hashing*.

But the root path has **no equivalent**: `HashTree.Rehash()` → `IndexReader()`, and `IndexReader()` does **not** sort. The root's "hash" function is inline SHA-256 of the unsorted body, not a `HashEntries`-style canonicalization. That is the whole bug, in one sentence:

> [!important] One-line root cause
> `HashEntries()` sorts document entries before hashing (so document indices are canonical), but `HashTree.IndexReader()` serializes root entries in raw slice order with no sort, and `Rehash()` hashes that unsorted body — so after any `Add`/`Remove` the root body is unsorted and the cloud rejects it.

---

## 4. Root cause analysis (line-anchored)

| Claim | Evidence |
|---|---|
| Bug is in the fork, not remarquee | `go.mod:13,22` replace pin; no `remarquee` file serializes the root index |
| Writes fail at the root PUT | issue #23 debug log: `root.docSchema` PUT → 400; `pkg/rmcloud/logtransport.go` logs it |
| The root serializer emits slice order | fork `api/sync15/tree.go` `IndexReader()`: `for _, d := range t.Docs` with no `sort.Slice` |
| `Add` unsorts (append) | fork `api/sync15/blobdoc.go` `Add`: `t.Docs = append(t.Docs, d)` + `Rehash()` only |
| `Remove` unsorts (swap-delete) | fork `api/sync15/tree.go` `Remove`: `t.Docs[docIndex] = t.Docs[length]` + `Rehash()` only |
| Reads sort, so reads work | fork `api/sync15/tree.go` `Mirror()`: `sort.Slice(head, ...)` before `t.Docs = head` |
| Hash & body stay consistent | `Rehash()` reads `IndexReader()` and SHA-256s it; mismatch is not the failure mode |
| Document indices are already canonical | fork `api/sync15/common.go` `HashEntries()` sorts in place before hashing |
| This is not the v3/v4 schema bug | `forceSchemaV4` in `pkg/rmcloud/auth.go` already handles that; message differs ("invalid hash" vs "invalid root schema") |
| Upstream fix exists but is unmerged | ddvk/rmapi#77 `state: OPEN`, `mergedAt: null`; patches `IndexReader()` + `IndexReaderWithSchema()` |

---

## 5. Why the obvious "just sort in the mutators" fix is the wrong instinct

A naive reading says: "Add appends, Remove swap-deletes — so sort in Add and Remove." Resist that. Two reasons:

1. **It fixes the symptom, not the invariant.** Any *future* mutator (a new `Move` that reorders, a batch import, a cache-deserialization path that doesn't sort) reintroduces the bug. The cloud's requirement is "the *emitted body* is sorted," not "the in-memory slice is sorted." The right place to enforce an output invariant is at the **output boundary** — the single serialization point, `IndexReader()`.
2. **It risks a hash/body desync.** If you sort in a mutator but `Rehash()` is ever called on a slice that a later code path reorders before serialization, the hash (computed from one order) and the body (emitted in another) disagree, and the cloud rejects the *hash* as well. Sorting at `IndexReader()` makes `Rehash()` and the upload read the **same bytes by construction**, because `Rehash()` literally calls `IndexReader()`.

This is exactly the reasoning in upstream PR ddvk/rmapi#77, and it is correct.

> [!tip] Generalize the lesson
> Whenever an invariant is about a *serialized representation* (canonical ordering, deterministic encoding, no trailing whitespace, etc.), enforce it at the **serializer**, not at each caller. Callers will always outnumber serializers, and new callers will forget.

---

## 6. The fix: options & decision

### 6.1 Option A — Bump the fork to a commit containing ddvk/rmapi#77 (preferred)

**What:** change the `replace` in `go.mod` to point at a fork/commit that includes the one-line sort in `IndexReader()` (and the matching guard in `IndexReaderWithSchema()`). Once ddvk/rmapi#77 merges to `ddvk/rmapi` master, point at `ddvk/rmapi`; until then, either the PR head or our own fork with the identical change.

**The upstream patch (verbatim, from ddvk/rmapi#77):**

`api/sync15/tree.go`, at the top of `IndexReader()`, before writing anything:
```go
// Canonical order: the cloud validates that root index entries are sorted
// by document ID and rejects unsorted uploads with
// 400 {"message":"invalid root schema"} (ddvk/rmapi#75, #76 — 2026-08-17).
sort.Slice(t.Docs, func(i, j int) bool { return t.Docs[i].DocumentID < t.Docs[j].DocumentID })
```

`api/sync15/blobdoc.go`, at the top of `IndexReaderWithSchema()`, before writing anything:
```go
// Same canonical-order requirement as the root index (see HashTree.IndexReader).
// Normally d.Files is already sorted (Rehash()->HashEntries() sorts in place),
// but AddFile() appends and not every write path rehashes first.
sort.Slice(d.Files, func(i, j int) bool { return d.Files[i].DocumentID < d.Files[j].DocumentID })
```

Both sorts are in-place and idempotent (already-sorted input is a no-op), so they cannot disagree with `HashEntries()`'s ordering.

**Why preferred:** zero application code; the fix lives where the bug lives; we inherit upstream regression tests (`TestRootIndexIsSortedByDocumentID`, `TestRootIndexSortedAfterRemove`); future upstream maintenance flows to us for free.

**Cost:** we depend on a fork commit being available. Today that means either (a) waiting for ddvk#77 to merge and pointing at `ddvk/rmapi`, or (b) pushing the same patch to `marcobarcelos/rmapi` (or our own fork) and pointing the `replace` there.

### 6.2 Option B — Apply the patch ourselves in a vendored/our own fork

If we don't want to wait on ddvk#77 and don't want to depend on `marcobarcelos` cutting a new commit, we fork the fork: push `marcobarcelos/rmapi`'s pinned tree + the two `sort.Slice` lines to a `go-go-golems/rmapi` fork, tag it, and `replace` at our tag. Same code change as Option A; the only difference is who owns the fork.

### 6.3 Option C — Patch around it in `remarquee` without touching the fork (escape hatch)

If for some reason we cannot change the dependency *today*, we can reach into the unexported `hashTree` exactly like `forceSchemaV4` does — but instead of setting `SchemaVersion`, we sort `t.Docs` before each write. Concretely, in `pkg/rmcloud/auth.go` add a helper invoked from `CreateApiCtx` (after `forceSchemaV4`):

```go
// sortHashTreeDocs sorts the unexported hashTree.Docs by DocumentID in place,
// mirroring the missing canonicalization in rmapi's HashTree.IndexReader().
// This is a targeted workaround for issue #23 (ddvk/rmapi#75/#76); remove once
// the fork pin in go.mod includes ddvk/rmapi#77.
func sortHashTreeDocs(apiCtx api.ApiCtx) {
    v := reflect.ValueOf(apiCtx)
    if v.Kind() != reflect.Pointer || v.IsNil() { return }
    v = v.Elem()
    hf := v.FieldByName("hashTree")
    if !hf.IsValid() || hf.IsNil() { return }
    docsField := hf.Elem().FieldByName("Docs")
    if !docsField.IsValid() || docsField.Len() == 0 { return }

    // Build a settable slice view over the unexported []*BlobDoc.
    settable := reflect.NewAt(docsField.Type(), unsafe.Pointer(docsField.UnsafeAddr())).Elem()
    docs := settable.Interface().([]*sync15.BlobDoc)

    sort.Slice(docs, func(i, j int) bool {
        return docs[i].DocumentID < docs[j].DocumentID
    })
    // NOTE: this only fixes the in-memory slice ONCE at context creation.
    // It does NOT fix subsequent Add()/Remove() within the same session,
    // so it is insufficient on its own for multi-write sessions. See §6.4.
}
```

> [!warning] Option C is incomplete
> Sorting once at context creation fixes the *first* write of a session, but `Add`/`Remove` re-unsort within the same `ApiCtx`, so the *second* write 400s again. To make C actually correct you'd also have to hook *every* mutator, which means you're re-implementing the fix in the wrong layer. Option C is only acceptable as a same-day stopgap for single-write scripts, and must carry a `TODO(issue #23)` pointing at the dependency bump.

### 6.4 Decision

> [!check] Go with Option A (Option B as fallback)
> - **Primary:** bump the `replace` to a commit containing ddvk/rmapi#77 (the PR head, ddvk master once merged, or a `go-go-golems/rmapi` fork carrying the same two-line patch). Run `go mod tidy`, rebuild, verify with `cloud mkdir` + `cloud rm`.
> - **Fallback (Option B):** if no upstream commit is available this week, push the patch to `go-go-golems/rmapi` and pin our tag.
> - **Do not** ship Option C alone; it's a known-incomplete stopgap and adds `unsafe` surface area for no long-term benefit.
> - **Bonus (application-level, optional):** surface the server error body in the user-facing error so future 400s are diagnosable without `--log-level debug`. See §8.

---

## 7. Implementation guide (step-by-step)

> [!note] Prerequisites
> - Go 1.26.3 (per `go.mod`).
> - `gh` authenticated to `go-go-golems` and `ddvk/rmapi`.
> - A reMarkable account with a valid device token (`remarquee cloud account` works today because reads work).
> - **A throwaway test folder** under `/ai/2026/08/17/` to mutate; you'll create and delete it.

### Step 0 — Reproduce (confirm you can see the bug)

```bash
# From the repo root (branch task/remarquee-fix-root-schema == origin/main today).
remarquee cloud mkdir /ai/2026/08/17/rmq-0023-probe --non-interactive --log-level debug 2>&1 \
  | grep -E "root.docSchema|status=400|invalid root schema"
```

Expected: a `PUT .../files/<hash>` with `rm-filename=[root.docSchema]` returning `status=400` with body `{"message":"invalid root schema"}`. If you don't see this, stop — the bug may have been fixed upstream; re-check `go.mod` and §1.3.

### Step 1 — Get a fixed rmapi commit

Pick whichever is available, in this order:

```bash
# 1a. Is ddvk/rmapi#77 merged to ddvk master?
gh pr view 77 --repo ddvk/rmapi --json state,mergedAt
# If merged: pin ddvk/rmapi@<merge-commit-sha>.

# 1b. Otherwise, use the PR head (unmerged but functional, verified live by upstream).
gh pr view 77 --repo ddvk/rmapi --json headRefOid
# Use ddvk/rmapi@<headRefOid>.

# 1c. Or push the same two-line patch to go-go-golems/rmapi and tag it (Option B).
```

### Step 2 — Update `go.mod`

Edit `go.mod` line 22. Replace:

```
replace github.com/juruen/rmapi => github.com/marcobarcelos/rmapi v0.0.0-20260518211546-a0d079936d46
```

with (example for Option A using ddvk master after merge):

```
replace github.com/juruen/rmapi => github.com/ddvk/rmapi v0.0.0-<YYYYMMDDhhmmss>-<short12>
```

Then:

```bash
go mod tidy
grep -n "rmapi" go.mod          # confirm the replace now points at the fixed commit
go build ./...                  # must compile cleanly
```

> [!important] Pseudocode for the dependency change
> ```
> # go.mod replace directive:
> - replace juruen/rmapi => marcobarcelos/rmapi @ <pre-fix>
> + replace juruen/rmapi => <fixed-fork>            @ <post-fix-contains-#77>
>
> # then:
> go mod tidy
> go build ./...
> # the fork at <post-fix> must contain, in api/sync15/tree.go IndexReader():
> #   sort.Slice(t.Docs, func(i,j int) bool { return t.Docs[i].DocumentID < t.Docs[j].DocumentID })
> # and in api/sync15/blobdoc.go IndexReaderWithSchema():
> #   sort.Slice(d.Files, func(i,j int) bool { return d.Files[i].DocumentID < d.Files[j].DocumentID })
> ```

### Step 3 — Verify the fix is actually present in the pinned source

Don't trust the version string; trust the code. After `go mod tidy`:

```bash
FORK=$(go env GOMODCACHE)/$(grep '^replace' go.mod | sed -E 's/.*=>\s*([^ ]+) (.*)/\1@\2/' | tr -d '\n')
# simpler: just find the new fork dir
ls "$(go env GOMODCACHE)"/github.com/* rmapi* -d 2>/dev/null
NEWFORK=$(go env GOMODCACHE)/$(go mod download -json github.com/juruen/rmapi 2>/dev/null | jq -r .Dir | sed "s|$(go env GOMODCACHE)/||" | head -1)
echo "New fork dir: $NEWFORK"

# Confirm the sort exists in IndexReader:
grep -n "sort.Slice(t.Docs" "$NEWFORK/api/sync15/tree.go"
grep -n "sort.Slice(d.Files" "$NEWFORK/api/sync15/blobdoc.go"
```

Both greps must return a line. If not, you pinned the wrong commit.

### Step 4 — Rebuild and install

```bash
go build -o remarquee ./cmd/remarquee
# or, to match the issue's deployment:
cp remarquee ~/.local/bin/remarquee
remarquee --version
```

### Step 5 — Smoke-test against the live cloud (the real proof)

```bash
# A single mkdir is the minimal write.
remarquee cloud mkdir /ai/2026/08/17/rmq-0023-probe --non-interactive
# Expect: success (no error). Previously: "failed to create directory: ... status 400"

# A second write in the same session proves Add()-after-Add() stays sorted.
remarquee cloud mkdir /ai/2026/08/17/rmq-0023-probe2 --non-interactive

# A delete proves the Remove() swap-delete path.
remarquee cloud rm /ai/2026/08/17/rmq-0023-probe --yes
remarquee cloud rm /ai/2026/08/17/rmq-0023-probe2 --yes

# An upload proves the full MkdirAll + UploadDocument path.
echo "# probe" > /tmp/rmq-0023-probe.md
remarquee upload md /tmp/rmq-0023-probe.md --name "rmq-0023-probe" \
  --remote-dir /ai/2026/08/17 --non-interactive
# Expect: OK: uploaded rmq-0023-probe.pdf -> /ai/2026/08/17
remarquee cloud rm "/ai/2026/08/17/rmq-0023-probe" --yes   # cleanup
```

### Step 6 — Run the existing test suite + upstream regression tests

```bash
go test ./...                       # remarquee's own tests (should be unaffected)
# If you pinned ddvk/rmapi#77, its regression tests ship in the fork:
go test ./...  # within the fork dir, or via:
go test github.com/juruen/rmapi/api/sync15 -run 'TestRootIndex(IsSortedByDocumentID|SortedAfterRemove)' -v
```

Expected: both `TestRootIndexIsSortedByDocumentID` (the `Add`/append path) and `TestRootIndexSortedAfterRemove` (the `Remove` swap-delete path, which additionally asserts `tree.Hash == sha256(body)`) pass.

### Step 7 — Commit

Follow the repo's `gitmoji` convention (see `AGENT.md` git guidelines). Example:

```yaml
# .git-commit-message.yaml
title: ":wrench: fix(rmcloud): bump rmapi fork to sort root index by document ID (fixes #23)"
description: |
  All cloud writes failed with 400 "invalid root schema" since 2026-08-17
  because the pinned rmapi fork serialized the root index in slice order;
  the cloud now requires entries sorted by document ID (ddvk/rmapi#75/#76).
  Bumped the go.mod replace pin to a commit containing ddvk/rmapi#77, which
  sorts t.Docs in HashTree.IndexReader() and d.Files in
  BlobDoc.IndexReaderWithSchema() at the single serialization point, keeping
  the uploaded body and Rehash()-derived hash consistent by construction.
  No remarquee application code changed.
tests:
  - go test ./... — PASS
  - go test .../api/sync15 -run TestRootIndexIsSortedByDocumentID — PASS
  - go test .../api/sync15 -run TestRootIndexSortedAfterRemove — PASS
  - manual: remarquee cloud mkdir + cloud rm + upload md — all succeed live
```

```bash
git add go.mod go.sum
git commit -m "$(yq '.title' .git-commit-message.yaml)"
```

---

## 8. Optional application-level improvement: surface the error body

Independent of the dependency fix, the issue notes that the user-facing error hides the cause:

```
Error: failed to create directory: request failed with status 400
```

…whereas `--log-level debug` reveals `{"message":"invalid root schema"}`. Future 400s would be far easier to diagnose if the body were included. The transport wrapper already reads the full body (`pkg/rmcloud/logtransport.go`); the rmapi fork's HTTP layer discards it before returning the generic `request failed with status 400`.

**Scope note:** this is an *optional* nicety, not required to fix #23. If pursued, it belongs in its own small change (e.g. a remarquee-level error wrapper that, on a 400 from a root/docSchema PUT, appends the body), and should be filed separately so it doesn't block the dependency bump.

---

## 9. Risks, alternatives, open questions

### Risks

- **Pinning an unmerged PR head (Option A, 1b).** If ddvk/rmapi#77 is rebased/force-pushed before merge, our pinned SHA could disappear. Mitigation: pin by exact SHA (not branch), and/or mirror the commit to `go-go-golems/rmapi` (Option B) so we control it.
- **Fork divergence.** `marcobarcelos/rmapi` may have other patches we rely on (e.g. the "always emit v4" behavior in `IndexReader()` that the comment block calls out). Moving to `ddvk/rmapi` must preserve that. Mitigation: diff `marcobarcelos/rmapi@<pin>` vs the target commit on `api/sync15/*` and confirm the v4-always behavior is present; the ddvk#77 PR itself is based on ddvk master which already writes v4.
- **Reflection/`unsafe` accumulation.** If we ever adopt Option C, we add another `unsafe.Pointer` reach into unexported state next to `forceSchemaV4`. Each one is a maintenance hazard across rmapi versions. Mitigation: treat C as a same-day stopgap only; always finish with A/B.

### Alternatives considered

- **Sort in `Add`/`Remove` only.** Rejected — fixes symptom, not invariant; fragile to future mutators; risks hash/body desync. (See §5.)
- **Re-mirror before every write.** Rejected — turns every write into a full tree sync (slow, and still races with concurrent writers). The `Sync` retry loop already re-mirrors *on `ErrWrongGeneration`*; that's the right place for it, not as an ordering fix.
- **Client-side sort + last-write-wins on a single client.** Rejected — unrelated; this isn't a conflict-resolution problem, it's a serialization-ordering requirement.

### Open questions

- Does the cloud enforce **strict** (no duplicates) or **non-strict** (duplicates allowed if adjacent) ascending order? ddvk#77 uses `<` (strict ascending). Document IDs are UUIDs and unique, so strict vs non-strict is moot in practice, but worth noting.
- Is there an analogous ordering requirement on the **document** index (`<id>.docSchema`)? ddvk#77 patches `IndexReaderWithSchema()` defensively "just in case," which suggests upstream isn't 100% sure either. Our fix inherits that defensiveness for free.
- Should we add a remarquee-level unit test that constructs an `ApiCtx` with an unsorted in-memory tree and asserts `IndexReader()` (via the fork) emits sorted output? Such a test would guard against a future fork regression, but it crosses the dependency boundary (tests the fork, not remarquee). Prefer relying on the fork's own regression tests.

---

## 10. API & file reference (quick index)

### 10.1 `remarquee` files (this repo)

| File | Role | Relevant to #23? |
|---|---|---|
| `go.mod:13,22` | `rmapi` module + `replace` fork pin | **Edit here** (the fix) |
| `pkg/rmcloud/auth.go` | `CreateApiCtx`, `WithAuthRetry`, `forceSchemaV4` | Context; `forceSchemaV4` is a *different* bug's workaround |
| `pkg/rmcloud/logtransport.go` | `loggingRoundTripper` — debug HTTP logging | How we *see* the 400 body |
| `pkg/rmcloud/dirs.go` | `MkdirAll` — walks path, calls `CreateDir` per segment | First write that fails |
| `pkg/rmcloud/download.go` | `DownloadDocumentByPath` — read path | Unaffected (reads) |
| `cmd/remarquee/cmds/cloud/mkdir.go` | `cloud mkdir` command | Repro / verification |
| `cmd/remarquee/cmds/cloud/rm.go` | `cloud rm` command (uses `DeleteEntry`→`Remove`) | Repro / verification |
| `cmd/remarquee/cmds/upload/md.go` | `upload md` command | Repro / verification |
| `cmd/remarquee/cmds/upload/upload_helpers.go` | `uploadPDFToRemoteWithAuthRetry` | The `MkdirAll`+`UploadDocument` write path |

### 10.2 rmapi fork files (pinned dependency)

All under `$(go env GOMODCACHE)/github.com/marcobarcelos/rmapi@v0.0.0-20260518211546-a0d079936d46` (pre-fix). After the bump, under the new fork dir.

| File | Symbol | Role |
|---|---|---|
| `api/sync15/tree.go` | `HashTree.IndexReader()` | **Bug site** — serializes root in slice order, no sort |
| `api/sync15/tree.go` | `HashTree.Rehash()` | SHA-256 of `IndexReader()` output (keeps hash/body consistent) |
| `api/sync15/tree.go` | `HashTree.Add()` | n/a (mutator; appends) — *upstream* `Add` is in blobdoc.go |
| `api/sync15/tree.go` | `HashTree.Remove()` | Mutator #2 — swap-delete unsorts |
| `api/sync15/tree.go` | `HashTree.Mirror()` | Read path — **does** sort `t.Docs` (why reads work) |
| `api/sync15/blobdoc.go` | `HashTree.Add()` | Mutator #1 — `append` unsorts (note: `Add` lives here, not in tree.go) |
| `api/sync15/blobdoc.go` | `BlobDoc.IndexReaderWithSchema()` | Document index serializer; defensively patched by ddvk#77 |
| `api/sync15/common.go` | `HashEntries()` | Sorts `entries` in place before hashing — *why document indices are already canonical* |
| `api/sync15/apictx.go` | `ApiCtx.CreateDir / UploadDocument / DeleteEntry / MoveEntry` | The four write entrypoints; all route through `Sync` |
| `api/sync15/apictx.go` | `Sync(...)` | Upload root blob + `WriteRootIndex`; retries on `ErrWrongGeneration` |
| `api/sync15/blobstorage.go` | `UploadBlob`, `WriteRootIndex` | The actual HTTP PUTs (`/sync/v3/files/{hash}`, `RootPut`) |
| `api/sync15/entry.go` | `Entry.Line()` | One document-index line: `<hash>:0:<DocumentID>:0:<size>` |

### 10.3 Wire format reference (schema v4 root index)

```
Line 1:   "4\n"
Line 2:   "0:.<count>:<totalSize>\n"            (v4 summary)
Line 3+:  "<sha256-hex>:0:<DocumentID>:<subfiles>:<size>\n"
Root hash = SHA-256 of the entire file above.
Cloud invariant (since 2026-08-17): Line 3+ entries strictly ascending by <DocumentID>.
```

### 10.4 HTTP endpoints (reMarkable sync/v3)

| Endpoint | Method | Purpose | Called by |
|---|---|---|---|
| `config.RootGet` | GET | Fetch current root `{hash, generation}` | `BlobStorage.GetRootIndex` |
| `config.BlobUrl + <hash>` (`/sync/v3/files/{hash}`) | PUT | Upload a blob (content, metadata, docSchema, **root.docSchema**) | `BlobStorage.UploadBlob` |
| `config.RootPut` | PUT | Advance the root generation `{hash, generation, broadcast}` | `BlobStorage.WriteRootIndex` |

---

## 11. Glossary

- **Root index / `root.docSchema`** — top-level table of contents; one line per document; its SHA-256 is the *root hash*.
- **Document index / `<id>.docSchema`** — per-document table of contents; one line per file in the document.
- **Blob** — raw content-addressed bytes uploaded by hash. `.metadata` (JSON), `.content` (JSON), `.pdf`, `.rm` pages, etc.
- **Generation** — monotonic counter the cloud advances on each accepted root write; used for optimistic concurrency (`ErrWrongGeneration` triggers a re-mirror + retry in `Sync`).
- **Schema v3 vs v4** — root index wire format. v3 has no summary line and uses `DocType` (`80000000`) in the type field; v4 has the `0:.<count>:<size>` summary line and uses `FileType` (`0`). The cloud rejected new v3 roots earlier ("Software must be updated"); our fork already emits v4, so version is *not* the bug.
- **`HashTree`** — rmapi's in-memory mirror of the root index (`Docs []*BlobDoc`, `Hash`, `Generation`, `SchemaVersion`).
- **`BlobDoc`** — rmapi's in-memory model of one document (`Files []*Entry`, `Entry`, `Metadata`, `Content`).
- **`Mirror()`** — pull the remote root into the local `HashTree` (read/sync). Sorts `Docs`.
- **`Sync()`** — apply a local mutation then push the new root (write). Does *not* sort.
- **`forceSchemaV4`** — a `remarquee` reflection workaround (`pkg/rmcloud/auth.go`) for a *different* rmapi bug (empty `SchemaVersion` → v3 → 400 "invalid hash"). Not this bug.

---

## 12. References

- Issue: [go-go-golems/remarquee#23](https://github.com/go-go-golems/remarquee/issues/23)
- Upstream bugs: [ddvk/rmapi#75](https://github.com/ddvk/rmapi/issues/75), [ddvk/rmapi#76](https://github.com/ddvk/rmapi/issues/76)
- Upstream fix PR: [ddvk/rmapi#77](https://github.com/ddvk/rmapi/pull/77)
- rmapi-js schema-4 background: [rmapi-js CHANGELOG](https://github.com/erikbrinkman/rmapi-js/blob/main/CHANGELOG.md) (v3 rejection, "Software must be updated")
- Repo guidelines: `AGENT.md`
- Last good ticket (structure reference): `ttmp/2026/08/10/RMQ-0022--configurable-pandoc-from-format-string/`
