---
Title: Obsidian-to-reMarkable Sync - Analysis, Design, and Implementation Guide
Ticket: RM-SYNC-001
Status: active
Topics:
    - remarquee
    - obsidian
    - remarkable
    - sync
    - pdf
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../go-go-golems/remarquee/cmd/remarquee/cmds/cloud/find.go
      Note: Remote tree recursive listing with structured JSON output
    - Path: ../../../../../../../go-go-golems/remarquee/cmd/remarquee/cmds/upload/md.go
      Note: Core upload logic with existence check after conversion (inefficient)
    - Path: cmd/remarquee/cmds/upload/sync_plan.go
      Note: Implementation of sync planning model from Phase 1
    - Path: cmd/remarquee/cmds/upload/sync_plan_test.go
      Note: Tests for Phase 1 sync planning semantics
    - Path: ttmp/2026/05/03/RM-SYNC-001--obsidian-vault-to-remarkable-sync-script/scripts/sync_obsidian_to_remarkable.sh
      Note: Bash prototype for sync workflow
ExternalSources: []
Summary: Comprehensive technical guide for syncing Obsidian vault Markdown notes to reMarkable tablets as PDFs, with deduplication, gap analysis, and proposed remarquee sync verb.
LastUpdated: 2026-05-03T14:00:00-04:00
WhatFor: Onboard a new intern to the remarquee/Obsidian/reMarkable sync problem space and proposed solution.
WhenToUse: When implementing the sync script or extending remarquee with a native sync command.
---





# Obsidian-to-reMarkable Sync: Analysis, Design, and Implementation Guide

## Executive Summary

This document describes how to build a pipeline that synchronizes Markdown project notes from an Obsidian vault to a reMarkable tablet as PDFs. The core constraint is **idempotency**: files that already exist on the tablet must not be re-converted or re-uploaded. We investigate the existing toolchain (`remarquee`, `rmapi`, `pandoc`, `xelatex`), identify architectural gaps, propose a design for a first-class `sync` verb, and provide a concrete bash prototype that works today without modifying any Go code.

**Key finding:** `remarquee upload md` already skips existing documents, but it performs the expensive PDF conversion *before* checking for existence. For a vault with hundreds of notes where only a handful are new, this wastes enormous CPU time. A proper sync command should build a remote index first, compute a delta, and only convert the delta.

---

## Problem Statement and Scope

### What we are trying to solve

Manuel maintains an Obsidian vault at `/home/manuel/code/wesen/obsidian-vault`. Project notes are stored under `Projects/YYYY/MM/DD/` as Markdown files with names like `PROJ - <Project Name>.md` or `ARTICLE - <Topic>.md`. There are hundreds of these files. He wants to read them on his reMarkable Paper Pro. The reMarkable ecosystem natively reads PDFs, not Markdown.

The workflow must satisfy three invariants:

1. **Convert:** Every selected Markdown file becomes a PDF with proper typography, code blocks, diagrams, and frontmatter.
2. **Deduplicate:** If a PDF with the same name already exists in the target remote directory, do not overwrite it (unless explicitly forced).
3. **Structure:** Preserve the local directory tree on the tablet, or at least provide a predictable layout.

### What is out of scope

- Two-way sync (tablet annotations back to Obsidian). That requires `.rmdoc` → Markdown conversion, which is a separate problem domain handled by `remarquee rmdoc render-v6`.
- Content-hash deduplication. The reMarkable cloud does not expose content hashes for documents. We can only compare names and, optionally, modification times.
- Real-time sync. This is a batch pipeline, not a filesystem watcher.

---

## Current-State Architecture

### The toolchain stack

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  Obsidian Vault │     │   remarquee     │     │  reMarkable     │
│  (.md files)    │────>│  (Go CLI)       │────>│  Cloud (rmapi)  │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                               │
                               v
                        ┌─────────────────┐
                        │  pandoc +       │
                        │  xelatex        │
                        │  (DejaVu fonts) │
                        └─────────────────┘
```

**Obsidian Vault.** A directory tree of Markdown files with YAML frontmatter, wikilinks, callouts, and Mermaid diagrams. The vault root is `/home/manuel/code/wesen/obsidian-vault`. Project notes specifically live under `Projects/YYYY/MM/DD/`.

**remarquee.** A Go CLI application built with Cobra and Glazed. It is the unified toolkit for all reMarkable workflows in this ecosystem. Source: `/home/manuel/code/wesen/go-go-golems/remarquee`. It provides three major command groups:

- `remarquee upload` — convert and upload content
- `remarquee cloud` — introspect and manipulate the remote file tree
- `remarquee rmdoc` — inspect and render `.rmdoc` archives

**pandoc + xelatex.** The PDF conversion backend. remarquee shells out to `pandoc` with `--pdf-engine=xelatex` and DejaVu fonts. This is the slowest step in the pipeline. A large Markdown file with Mermaid diagrams can take 5–15 seconds to convert.

**rmapi / reMarkable Cloud.** `rmapi` is an open-source Go library (`github.com/juruen/rmapi`) that speaks the reMarkable cloud protocol (Sync 1.5). It maintains a local mirror of the remote document tree, supports token-based auth, and exposes operations like `UploadDocument`, `CreateDir`, `DeleteEntry`, and `NodeByPath`.

### remarquee upload commands in detail

#### `remarquee upload md`

Source: `cmd/remarquee/cmds/upload/md.go` (lines 1–390)

This command takes one or more Markdown files or directories, recursively collects `.md` files, converts each to PDF, and uploads them.

Key flags:
- `--preserve-dirs` (default `true`) — recreates the local relative directory structure remotely
- `--flatten` — overrides `--preserve-dirs`, uploads everything to a single flat directory
- `--dry-run` — prints what would be done without running pandoc or uploading
- `--force` — overwrites existing documents (deletes annotations)
- `--remote-dir` — overrides the default `/ai/YYYY/MM/DD/` destination

The existence check happens **inside the upload loop**, after conversion:

```go
// Inside runUploadMarkdown, after mdpdf.ConvertMarkdownFileToPDF(...)
existingNode, err := apiCtx.Filetree().NodeByPath(docName, dstNode)
if err == nil {
    if !s.Force {
        fmt.Fprintf(cmd.OutOrStdout(), "SKIP: %s already exists ...\n", docName, dst)
        continue  // <-- PDF was already generated; work is wasted
    }
    // ... delete existing, then upload
}
```

This is the critical inefficiency. For 100 files where 90 already exist, pandoc runs 100 times but only 10 uploads occur.

#### `remarquee upload bundle`

Source: `cmd/remarquee/cmds/upload/bundle.go` (lines 1–280)

This command concatenates multiple Markdown files into a single PDF with a clickable Table of Contents (ToC), then uploads it as one document. It is ideal for delivering a ticket's documentation bundle (diary + design doc + README) as a single file.

Key flags:
- `--name` — output document name
- `--toc-depth` — ToC depth (default 1)
- `--remote-dir` — destination folder

Bundle mode does not support `--preserve-dirs` because it produces a single PDF. It is not suitable for syncing an entire vault tree.

#### `remarquee upload src`

Source: `cmd/remarquee/cmds/upload/src.go`

Renders source code files as syntax-highlighted PDFs. Not relevant for this project.

### remarquee cloud commands in detail

#### `remarquee cloud find`

Source: `cmd/remarquee/cmds/cloud/find.go` (lines 1–180)

Recursively walks the remote file tree starting from a given path. Supports an optional regexp pattern filter. In Glaze mode (`--with-glaze-output --output json`), it returns structured records:

```json
{
  "id": "a28daba4-dadd-4391-9b73-d32cd930d626",
  "name": "01-system-architecture-and-recovery-guide-crib-k3s-cluster",
  "type": "DocumentType",
  "is_dir": false,
  "path": "/ai/2026/05/03/k3s-restart/01-system-architecture-and-recovery-guide-crib-k3s-cluster",
  "modified_time": "2026-05-03T12:05:19Z",
  "version": 0
}
```

This is the primitive we use to build a remote index for deduplication.

#### `remarquee cloud stat`

Source: `cmd/remarquee/cmds/cloud/stat.go`

Returns metadata for a single remote path. Also supports JSON output. Useful for checking one specific file, but too slow for bulk indexing (one API call per path).

#### `remarquee cloud ls`

Source: `cmd/remarquee/cmds/cloud/ls.go`

Lists immediate children of a directory. Supports `--long` for timestamps and `--with-glaze-output` for JSON. Does not recurse.

### The rmapi data model

From `github.com/juruen/rmapi/model`:

- `Node` — represents a file or folder in the cloud tree
  - `Id()` → UUID string
  - `Name()` → leaf name
  - `IsDirectory()` / `IsFile()`
  - `Document.Type` → `CollectionType` (folder) or `DocumentType` (file)
  - `Document.ModifiedClient` → ISO 8601 timestamp string
  - `LastModified()` → `time.Time`
  - `Version()` → integer
  - `Parent` → pointer to parent `Node`

- `ApiCtx` — the authenticated API context
  - `Filetree()` → the cached remote tree
  - `UploadDocument(parentId, localPath, create bool, metadata *model.DocumentMetadata)` → uploads a PDF
  - `DeleteEntry(node, force, onlyTrash)` → deletes a remote document
  - `CreateDir(parentId, name, addToTree bool)` → creates a folder

### The rmcloud helper package

Source: `pkg/rmcloud/`

- `auth.go` — `CreateApiCtx(auth AuthSettings) (*api.UserInfo, api.ApiCtx, error)`
  - Handles token refresh with 3 retries
  - `NonInteractive` flag fails instead of prompting for a one-time code
- `dirs.go` — `MkdirAll(apiCtx api.ApiCtx, dirPath string) (*model.Node, error)`
  - Implements `mkdir -p` semantics using `CreateDir` and `FindByName`

---

## Gap Analysis

### Gap 1: Wasteful conversion order

**Observation:** `upload md` converts Markdown to PDF **before** checking whether the remote document already exists.

**Impact:** For a vault of 200 files where 190 are already on the tablet, 190 unnecessary pandoc invocations occur. At ~5 seconds each, that is ~16 minutes of wasted work.

**Evidence:** `cmd/remarquee/cmds/upload/md.go`, lines 220–260:

```go
if err := mdpdf.ConvertMarkdownFileToPDF(ctx, mdPath, outPDF, pandocOpts); err != nil {
    return err
}
// ... existence check happens here, AFTER conversion
existingNode, err := apiCtx.Filetree().NodeByPath(docName, dstNode)
```

### Gap 2: No pre-flight delta computation

**Observation:** There is no command that answers "what would change?" before doing work. `--dry-run` in `upload md` only shows what pandoc commands would run; it does not consult the remote tree.

**Impact:** Users cannot preview which files are new, which are skipped, and which would be overwritten.

### Gap 3: No dedicated sync verb

**Observation:** The `upload` subcommand family has `md`, `bundle`, and `src`, but no `sync`. A sync verb would naturally combine:
- Local input collection
- Remote index building
- Delta computation
- Selective conversion and upload

**Impact:** Users must cobble together external scripts (bash, Python) that call `cloud find` and `upload md` separately. This is fragile and slow (multiple rmapi sessions).

### Gap 4: No mtime-aware sync

**Observation:** remarquee only checks document name existence. It does not compare local file modification time against remote `ModifiedClient`.

**Impact:** If a local Markdown file is edited after its PDF was uploaded, remarquee will skip it because the name matches. The tablet will hold a stale version.

### Gap 5: No orphaned-file cleanup

**Observation:** There is no mechanism to remove remote documents that no longer have a corresponding local Markdown file.

**Impact:** Over time, the tablet accumulates stale PDFs. This is acceptable for append-only vaults but problematic if notes are renamed or deleted.

---

## Proposed Solution

### Option A: Bash prototype (works today)

A bash script uses `remarquee cloud find` to build a remote index, computes a delta with `comm(1)`, and then calls `remarquee upload md` only for missing files.

**Pros:**
- Zero Go code changes
- Can be written and tested in minutes
- Uses stable, documented remarquee commands

**Cons:**
- One `upload md` invocation per file → one rmapi auth session per file (slow)
- No mtime comparison
- No atomic batch upload
- Fragile to filename edge cases

**Prototype location:** `scripts/sync_obsidian_to_remarkable.sh` in this ticket.

### Option B: New `remarquee upload sync` command (recommended)

Add a new subcommand under `remarquee upload`:

```bash
remarquee upload sync <local-paths...> \
  --remote-dir /ai/YYYY/MM/DD/obsidian-projects \
  --preserve-dirs \
  --compare-mtime \
  --delete-orphans \
  --dry-run
```

#### Behavior specification

1. **Collect local inputs.** Same logic as `upload md`: accept files and directories, recurse for `.md`, sort, deduplicate.
2. **Build remote index.** Call `apiCtx.Filetree().NodeByPath(remoteDir, nil)` to get the target node, then walk its subtree to build a map of `remoteDocName → modifiedTime`.
3. **Compute delta.** For each local file:
   - Compute the expected remote document name (`basename(md) → basename + ".pdf"`)
   - Compute the expected remote path (with `--preserve-dirs`, include relative subdirectories)
   - Check the remote index:
     - **New:** remote path does not exist → add to `uploadSet`
     - **Unchanged:** remote path exists, and (`--compare-mtime` is off or local mtime ≤ remote mtime) → add to `skipSet`
     - **Stale:** remote path exists, `--compare-mtime` is on, and local mtime > remote mtime → add to `overwriteSet` (require `--force` to actually overwrite)
4. **Report delta (dry-run or summary).** Print a table:
   - `UPLOAD  12 files`
   - `SKIP    188 files`
   - `STALE   5 files (use --force to overwrite)`
   - `ORPHAN  3 files (use --delete-orphans to remove)`
5. **Convert and upload.** For each file in `uploadSet`:
   - Run `mdpdf.ConvertMarkdownFileToPDF` to a temp file
   - Ensure the remote parent directory exists (`rmcloud.MkdirAll`)
   - Upload via `apiCtx.UploadDocument`
6. **Cleanup (optional).** If `--delete-orphans`, remove remote documents not present in the local input set.

#### Pseudocode

```pseudocode
function runSync(localPaths, remoteDir, settings):
    localInputs = collectMarkdownInputs(localPaths)
    remoteDirNode = rmcloud.MkdirAll(apiCtx, remoteDir)  // ensures target exists
    remoteIndex = buildRemoteIndex(remoteDirNode)

    uploadSet = []
    skipSet = []
    staleSet = []
    orphanSet = []

    for in in localInputs:
        docName = basename(in.absPath, ".md") + ".pdf"
        relDir = settings.preserveDirs ? in.relDir() : ""
        remoteKey = joinRemotePath(remoteDir, relDir, docName)

        if remoteKey not in remoteIndex:
            uploadSet.append(in)
        else if settings.compareMtime:
            localMtime = stat(in.absPath).mtime
            remoteMtime = remoteIndex[remoteKey].modifiedTime
            if localMtime > remoteMtime:
                staleSet.append(in)
            else:
                skipSet.append(in)
        else:
            skipSet.append(in)

    if settings.deleteOrphans:
        localKeys = { computeRemoteKey(in) for in in localInputs }
        for remoteKey, node in remoteIndex:
            if remoteKey not in localKeys:
                orphanSet.append(node)

    printSummary(uploadSet, skipSet, staleSet, orphanSet)

    if settings.dryRun:
        return

    dstNodeCache = {}
    for in in uploadSet:
        pdfName = basename(in.absPath, ".md") + ".pdf"
        tmpPDF = tempDir + "/" + pdfName
        mdpdf.ConvertMarkdownFileToPDF(ctx, in.absPath, tmpPDF, pandocOpts)

        dst = settings.preserveDirs
              ? joinRemoteDir(remoteDir, in.relDir())
              : remoteDir

        dstNode = dstNodeCache[dst]
        if dstNode == nil:
            dstNode = rmcloud.MkdirAll(apiCtx, dst)
            dstNodeCache[dst] = dstNode

        apiCtx.UploadDocument(dstNode.id(), tmpPDF, true, nil)
        print("UPLOADED", pdfName, "->", dst)
```

#### API sketch (Go)

```go
// cmd/remarquee/cmds/upload/sync.go

type uploadSyncSettings struct {
    NonInteractive bool
    Reauth         bool
    Force          bool
    DryRun         bool
    PreserveDirs   bool
    Flatten        bool
    Date           string
    RemoteDir      string
    CompareMtime   bool
    DeleteOrphans  bool
    // ... pandoc flags same as uploadMarkdownSettings
}

func NewUploadSyncCommand() *cobra.Command { ... }
func runUploadSync(ctx, cmd, settings, args) error { ... }
```

The new file would live alongside `md.go`, `bundle.go`, and `src.go` in `cmd/remarquee/cmds/upload/`.

#### Wiring into the command tree

In `cmd/remarquee/cmds/upload/root.go`:

```go
func NewUploadCommand() *cobra.Command {
    cmd := &cobra.Command{Use: "upload", Short: "Upload content to a reMarkable device"}
    cmd.AddCommand(NewUploadMarkdownCommand())
    cmd.AddCommand(NewUploadBundleCommand())
    cmd.AddCommand(NewUploadSourceCommand())
    cmd.AddCommand(NewUploadSyncCommand())  // <-- new
    return cmd
}
```

### Option C: Enhanced `--skip-existing` flag on `upload md`

Instead of a new command, add `--skip-existing` to `upload md` that:
- Builds the remote index once before the loop
- Skips conversion for existing files

**Pros:** Minimal change to existing command surface.
**Cons:** Does not support mtime comparison, orphaned cleanup, or a pre-flight summary. Less discoverable than a dedicated `sync` verb.

---

## Phased Implementation Plan

### Phase 1: Validate the bash prototype (1 day)

1. Run `scripts/sync_obsidian_to_remarkable.sh` against a small subset of the vault.
2. Verify that existing files are skipped and new files are uploaded.
3. Measure total runtime for 10, 50, and 100 files.
4. Document pain points (slow auth per file, no mtime check, etc.).

### Phase 2: Design the Go sync command (2 days)

1. Write a formal design doc (this document) and circulate for review.
2. Decide on flag naming (`--compare-mtime`, `--delete-orphans`, etc.).
3. Decide whether to extend `md` or add `sync`. **Recommendation:** add `sync`.

### Phase 3: Implement `remarquee upload sync` (3–5 days)

1. Create `cmd/remarquee/cmds/upload/sync.go`.
2. Extract shared helpers from `md.go` into `upload/internal/` if needed:
   - `collectMarkdownInputs`
   - `resolveRemoteDir`
   - `markdownPDFName`
   - `configureMarkdownPandocOptions`
3. Implement `buildRemoteIndex` using `apiCtx.Filetree()` traversal.
4. Implement delta logic with mtime comparison.
5. Implement batch upload with cached `MkdirAll`.
6. Add unit tests for delta computation (mock `model.Node` tree).

### Phase 4: Integration testing (2 days)

1. Dry-run against the full vault.
2. Verify remote directory structure matches local.
3. Re-run sync and confirm 100% skip rate.
4. Touch a local file, re-run with `--compare-mtime`, confirm it is flagged stale.
5. Test `--delete-orphans` on a throwaway remote folder.

### Phase 5: Documentation and release (1 day)

1. Add help text and examples for `remarquee upload sync`.
2. Update the remarquee README with a "Sync workflow" section.
3. Cut a release via GoReleaser.

---

## Testing and Validation Strategy

### Unit tests

- **Delta computation:** Given a local input list and a mocked remote tree, assert the correct upload/skip/stale/orphan sets.
- **Remote index builder:** Given a `model.Node` tree, assert the flat index map is correct.
- **Path mapping:** Given local paths with `--preserve-dirs` and `--flatten`, assert the computed remote keys.

### Integration tests

- Use `remarquee upload sync --dry-run` to preview changes without side effects.
- Use a dedicated test remote directory (e.g., `/ai/YYYY/MM/DD/test-sync/`) to avoid polluting production data.
- Validate by running `remarquee cloud find` before and after sync to confirm the expected file set.

### Performance benchmarks

- Measure total runtime for syncing 200 files (190 existing, 10 new) with:
  - Current `upload md` approach (baseline, expected ~16 min wasted)
  - Bash prototype (expected ~2 min, dominated by per-file auth)
  - Go `sync` command (expected ~30 sec, single auth + selective conversion)

---

## Risks, Alternatives, and Open Questions

### Risks

1. **rmapi token expiry mid-sync.** For large vaults, the rmapi session might expire. Mitigation: `CreateApiCtx` already retries auth 3 times. For very large syncs, consider chunked batches.
2. **Pandoc failures on malformed Markdown.** A single bad file should not abort the entire sync. Mitigation: wrap each conversion in error handling and report failures at the end.
3. **Name collisions.** Two different local files with the same basename in different subdirectories will collide in `--flatten` mode. Mitigation: `upload md` already detects this and errors early. The same check should be copied into `sync`.

### Alternatives considered

1. **Use `rclone` with a custom backend.** `rclone` supports reMarkable via a third-party backend, but it does not perform Markdown-to-PDF conversion. It is not a viable alternative.
2. **Use the legacy `remarkable_upload.py` script.** This Python script lives at `/home/manuel/.local/bin/remarkable_upload.py` and does similar pandoc-based uploads. It is deprecated in favor of remarquee and lacks cloud introspection entirely.
3. **Use Obsidian’s built-in PDF export.** Obsidian can export to PDF, but only through the GUI. There is no headless API for batch export.

### Open questions

1. **Should `--compare-mtime` be the default?** If enabled by default, re-running the sync after editing a note would correctly update the tablet. If disabled by default, behavior matches the current `upload md` semantics (pure name-based dedup).
2. **Should we support bundling per-directory?** For vaults with many small notes, users might prefer one PDF per day-folder rather than one PDF per note. This would require a hybrid `sync` + `bundle` mode.
3. **Should we cache the remote index to disk?** For repeated syncs, the remote tree rarely changes. Caching the JSON output of `cloud find` could save 1–2 seconds per run.

---

## Performance Analysis and Optimization Opportunities

### What we observed during the first vault sync

We ran `remarquee upload md --preserve-dirs` against the `Projects/2026/` folder (257 Markdown files). The upload succeeded but was **painfully slow**. The output showed repeated warnings:

```
WARNING: remote tree has changed, refresh the file tree
```

Every 2–3 files triggered an rmapi tree cache invalidation.

### Benchmarking the pipeline stages

**Pandoc conversion (the dominant bottleneck):**

```bash
$ time (echo "# test" | pandoc -f markdown -t pdf --pdf-engine=xelatex -o /tmp/test.pdf)
real  0m1.536s
```

A trivial file takes 1.5 seconds. Real project notes with Mermaid diagrams, code blocks, and YAML frontmatter take **3–8 seconds each**. At 257 files, that is **15–30 minutes of pure CPU time**.

**Upload bandwidth:** Not the bottleneck. The generated PDFs are small (tens to hundreds of KB). The reMarkable API absorbs them faster than pandoc produces them.

**rmapi tree refresh:** Adds a network round-trip after every upload. For 257 files, this compounds into noticeable delay.

**Current architecture is entirely sequential.** `cmd/remarquee/cmds/upload/md.go` has zero parallelization:

```go
for _, in := range mdInputs {
    // convert (CPU-bound)
    mdpdf.ConvertMarkdownFileToPDF(...)
    // upload (network-bound)
    apiCtx.UploadDocument(...)
}
```

### Optimization Option 1: Parallel pandoc workers in `upload md`

Add a `--workers N` flag to `remarquee upload md`. The loop becomes a two-phase pipeline:

```go
// Phase 1: convert in parallel (CPU-bound)
pdfResults := parallelConvert(ctx, mdInputs, pandocOpts, numWorkers)

// Phase 2: upload sequentially (network-bound, avoids tree refresh races)
for _, r := range pdfResults {
    // ... upload ...
}
```

**Expected speedup:** Nearly N× where N is CPU core count. On an 8-core machine, a 30-minute sync drops to ~4 minutes.

**Implementation sketch:**

```go
func parallelConvert(ctx context.Context, inputs []markdownInput, opts mdpdf.PandocOptions, workers int) []convertResult {
    var wg sync.WaitGroup
    inCh := make(chan markdownInput, len(inputs))
    outCh := make(chan convertResult, len(inputs))

    for i := 0; i < workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for in := range inCh {
                pdfPath := filepath.Join(tmpDir, in.pdfName())
                err := mdpdf.ConvertMarkdownFileToPDF(ctx, in.AbsPath, pdfPath, opts)
                outCh <- convertResult{input: in, pdfPath: pdfPath, err: err}
            }
        }()
    }

    for _, in := range inputs {
        inCh <- in
    }
    close(inCh)
    wg.Wait()
    close(outCh)

    var results []convertResult
    for r := range outCh {
        results = append(results, r)
    }
    return results
}
```

**Caveat:** Phase 2 (upload) should remain sequential or lightly parallelized because rmapi's local tree cache is not goroutine-safe. Multiple concurrent uploads to the same parent directory trigger the `remote tree has changed` refresh storm.

### Optimization Option 2: Pre-flight delta for `upload sync`

For **subsequent syncs** (not the first one), the bigger waste is converting files that already exist remotely. The proposed `sync` verb should:

1. Build remote index once with `cloud find`
2. Filter local inputs against it
3. Only convert the **delta**

This turns a 30-minute "no-op" re-sync into a **30-second operation** (if only 2–3 files changed).

### Optimization Option 3: Suppress rmapi tree refreshes during bulk upload

The warning `apictx.go:259: remote tree has changed` comes from `juruen/rmapi` inside `UploadDocument`. After every upload, rmapi invalidates its filetree cache. For bulk operations we could:

- Patch rmapi to accept a `skipRefresh` hint (upstream change)
- Or cache the destination `*model.Node` in remarquee and reuse it, avoiding path-by-name lookups

In `md.go`, the code already caches `dstNode`:

```go
dstNodeCache := map[string]*model.Node{}
```

But rmapi still refreshes internally. A deeper fix would require changes to the rmapi library itself.

### Optimization Option 4: Parallel conversion + bulk `cloud put`

If we pre-convert all Markdown files to PDFs in parallel (using `xargs -P 8` or GNU `parallel`), we could then bulk-upload them. However, `remarquee cloud put` does not support directory-preserving bulk upload, and `upload md` always re-converts. A new `--pdf-only` + `--from-pdf` mode would bridge this gap.

### Recommended priority

| Priority | Optimization | Impact | Effort |
|----------|-------------|--------|--------|
| 1 | `--workers N` in `upload md` | 8× speedup on first sync | Low (~20 lines) |
| 2 | `upload sync` with pre-flight delta | 60× speedup on re-syncs | Medium (new command) |
| 3 | rmapi refresh suppression | Removes per-upload latency | High (upstream patch) |
| 4 | `--from-pdf` bulk upload mode | 8× speedup via external parallel | Low (new flag) |

---

## Detailed Implementation Task Plan

This section turns the design into a concrete implementation checklist. The intent is to make each task independently reviewable and committable.

### Phase 1: Sync planning model and tests

1. **Add sync planning types.** Create a small internal planning model in `cmd/remarquee/cmds/upload/sync.go` or a dedicated helper file. The model should represent local inputs, expected remote keys, remote index entries, and plan actions (`upload`, `skip`, `stale`, `orphan`).
2. **Reuse existing path helpers.** Reuse `collectMarkdownInputs`, `markdownPDFName`, `joinRemoteDir`, and `remoteDocKey` from `md.go` instead of introducing a second path-normalization dialect.
3. **Build remote-key computation around full paths.** Do not use leaf-name-only matching in the native command. The bash prototype does that, but a durable implementation must key by full expected remote path so identically named notes in different folders do not collide.
4. **Unit-test planning without rmapi.** Use pure Go tests with synthetic local inputs and remote index entries. Cover: new uploads, existing skips, stale files when `--compare-mtime` is enabled, orphan detection, `--preserve-dirs`, and `--flatten`.

### Phase 2: Dry-run `upload sync`

1. **Add `NewUploadSyncCommand`.** It should mirror the relevant `upload md` flags: auth, `--force`, `--dry-run`, `--preserve-dirs`, `--flatten`, `--date`, `--remote-dir`, and pandoc/layout flags. Add sync-specific flags later.
2. **Wire command registration.** Add `cmd.AddCommand(NewUploadSyncCommand())` in `cmd/remarquee/cmds/upload/root.go`.
3. **Implement local collection and collision detection.** Start by collecting local markdown inputs and validating duplicate remote keys before any remote or pandoc work.
4. **Implement remote index construction.** In dry-run mode, authenticate once, locate/create or locate the target remote root, walk its subtree, and build a path-keyed index. If the target does not exist, the remote index should be empty.
5. **Print a stable plan summary.** Output deterministic lines for `UPLOAD`, `SKIP`, `STALE`, and `ORPHAN`. This makes manual validation and future snapshot tests easier.

### Phase 3: Execute upload delta

1. **Convert only the upload set.** After planning, run `mdpdf.ConvertMarkdownFileToPDF` only for files classified as `upload`; optionally include stale files only when overwrite behavior is explicitly requested.
2. **Create remote directories lazily.** Use `rmcloud.MkdirAll` with a destination-node cache, as `upload md` already does.
3. **Handle stale documents carefully.** Require `--force` before deleting/replacing an existing remote document. Without `--force`, report stale items but do not mutate them.
4. **Keep upload execution sequential initially.** rmapi filetree mutation and refresh behavior is a known source of complexity; correctness should come before parallel uploads.

### Phase 4: Performance follow-ups

1. **Add `--workers N` to `upload md`.** Parallelize conversion, not rmapi upload, and keep `--workers 1` behavior equivalent to today's sequential path.
2. **Optionally reuse the worker pool in `upload sync`.** Once upload sync is correct, the conversion stage can share the same bounded worker implementation.
3. **Investigate rmapi refresh suppression.** Treat this as separate because it may require upstream rmapi changes.
4. **Consider raw PDF upload mode.** A `--from-pdf` path would let users pre-convert externally and still use remarquee's directory-preserving upload semantics.

### Commit strategy

- Commit documentation/task updates first.
- Commit pure planning helpers and tests before wiring the CLI.
- Commit CLI dry-run behavior before enabling mutating uploads.
- Commit execution behavior only after dry-run output and unit tests are stable.
- Keep performance work (`--workers`) separate from sync semantics.

---

## References

### Key source files (absolute paths)

- `/home/manuel/code/wesen/go-go-golems/remarquee/cmd/remarquee/cmds/upload/md.go` — existing markdown upload logic
- `/home/manuel/code/wesen/go-go-golems/remarquee/cmd/remarquee/cmds/upload/bundle.go` — bundle upload logic
- `/home/manuel/code/wesen/go-go-golems/remarquee/cmd/remarquee/cmds/upload/root.go` — upload command registration
- `/home/manuel/code/wesen/go-go-golems/remarquee/cmd/remarquee/cmds/cloud/find.go` — remote tree recursive listing
- `/home/manuel/code/wesen/go-go-golems/remarquee/cmd/remarquee/cmds/cloud/stat.go` — remote metadata lookup
- `/home/manuel/code/wesen/go-go-golems/remarquee/pkg/rmcloud/auth.go` — rmapi authentication
- `/home/manuel/code/wesen/go-go-golems/remarquee/pkg/rmcloud/dirs.go` — remote directory creation

### Prototype script

- `/home/manuel/code/wesen/claw-stuff/ttmp/2026/05/03/RM-SYNC-001--obsidian-vault-to-remarkable-sync-script/scripts/sync_obsidian_to_remarkable.sh`

### External documentation

- `remarquee help upload` — built-in help for upload commands
- `remarquee help cloud` — built-in help for cloud commands
- `remarquee help md` — detailed `upload md` help
- `remarquee help bundle` — detailed `upload bundle` help
- rmapi source: `github.com/juruen/rmapi`

### Obsidian vault

- Root: `/home/manuel/code/wesen/obsidian-vault`
- Project notes: `/home/manuel/code/wesen/obsidian-vault/Projects/YYYY/MM/DD/`
- Exemplar project note: `/home/manuel/code/wesen/obsidian-vault/Projects/2026/03/15/PROJ - ZK Tool.md`
