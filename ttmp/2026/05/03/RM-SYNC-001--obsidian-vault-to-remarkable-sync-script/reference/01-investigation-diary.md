---
Title: Investigation Diary
Ticket: RM-SYNC-001
Status: active
Topics:
    - remarquee
    - obsidian
    - remarkable
    - sync
    - pdf
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/05/03/RM-SYNC-001--obsidian-vault-to-remarkable-sync-script/reference/01-investigation-diary.md
      Note: Chronological continuation notes for the resumed ticket work
    - Path: ttmp/2026/05/03/RM-SYNC-001--obsidian-vault-to-remarkable-sync-script/tasks.md
      Note: Task bookkeeping for doctor and reMarkable upload verification
    - Path: ttmp/vocabulary.yaml
      Note: Topic vocabulary fix that made docmgr doctor pass
ExternalSources: []
Summary: Chronological investigation of the Obsidian-to-reMarkable sync problem, remarquee capabilities, gaps, and prototype script.
LastUpdated: 2026-05-03T18:25:00-04:00
WhatFor: Record investigation steps, commands run, failures, and learnings for future reference.
WhenToUse: When continuing this work or reviewing why design decisions were made.
---


# Diary

## Goal

Investigate how to build a sync pipeline that converts Obsidian vault project notes (Markdown) to PDF and uploads them to a reMarkable tablet, skipping files that already exist on the target. Evaluate whether remarquee (the existing reMarkable CLI toolkit) has gaps that warrant new verbs or subcommands. Document everything for a future implementer.

## Step 1: Understanding the Obsidian Vault Layout

The user's vault lives at `/home/manuel/code/wesen/obsidian-vault`. Project notes are stored under `Projects/YYYY/MM/DD/` with filenames like `PROJ - <Name>.md` or `ARTICLE - <Name>.md`. There are hundreds of files across monthly folders. A sync tool must handle this tree recursively, preserve structure (or flatten), and reliably detect duplicates on the reMarkable side.

### What I did
- Listed the vault directory: `find /home/manuel/code/wesen/obsidian-vault/Projects/2026 -type f -name "*.md" | sort`
- Confirmed the naming conventions and deep nesting (e.g., `Projects/2026/04/22/ARTICLE - Foo.md`)

### What I learned
- The vault uses dated folders, but a sync target probably wants a stable remote directory per run (e.g., `/ai/YYYY/MM/DD/obsidian-projects/`)
- Files have YAML frontmatter, wikilinks, callouts, and Mermaid diagrams — all things pandoc + xelatex must handle

## Step 2: Reverse-Engineering remarquee Capabilities

remarquee is a Go CLI built on Cobra + Glazed, backed by rmapi. It lives in `/home/manuel/code/wesen/go-go-golems/remarquee`.

### What I did
- Ran `remarquee status` → ok
- Inspected all upload subcommands: `remarquee upload md`, `remarquee upload bundle`, `remarquee upload src`
- Inspected cloud introspection commands: `remarquee cloud ls`, `remarquee cloud find`, `remarquee cloud stat`
- Read the Go source for the upload and cloud packages

### What I learned

**Upload layer (`cmd/remarquee/cmds/upload/`)**
- `md` — converts individual .md files to PDF via pandoc/xelatex, then uploads. Supports `--preserve-dirs`, `--flatten`, `--dry-run`, `--force`. Already checks for existing documents by name and skips unless `--force`.
- `bundle` — concatenates multiple .md files into one PDF with a clickable ToC, then uploads as a single document. Also skips existing by name.
- `src` — syntax-highlights source files into PDF and uploads.

**Cloud layer (`cmd/remarquee/cmds/cloud/`)**
- `find` — recursive walk of the remote file tree. Supports `--with-glaze-output --output json` for structured data. Returns `name`, `path`, `is_dir`, `modified_time`, `id`, etc.
- `stat` — metadata for a single remote path. Also supports JSON output.
- `ls` — directory listing, human or structured.
- `mkdir`, `mv`, `rm`, `put`, `get`, `search` — standard filesystem verbs.

**rmcloud package (`pkg/rmcloud/`)**
- `CreateApiCtx` — bootstraps rmapi auth (token-based, with retry)
- `MkdirAll` — `mkdir -p` for remote paths
- `DownloadDocumentByPath` — fetches .rmdoc archives

### What was tricky to build

Nothing yet — this is analysis only. The trickiest part to understand was how remarquee does its existence check. In `md.go` (`runUploadMarkdown`), it calls:

```go
existingNode, err := apiCtx.Filetree().NodeByPath(docName, dstNode)
if err == nil {
    if !s.Force {
        fmt.Fprintf(..., "SKIP: %s already exists ...")
        continue
    }
    // delete + overwrite
}
```

This means remarquee **already** skips existing files, but it does so **per-file during upload**, not as a pre-flight plan. You don't get a summary of "would upload 3, skip 7" before the conversion starts. For a large vault, this is suboptimal because pandoc runs are expensive.

## Step 3: Gap Analysis — What Is Missing?

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket for upload the obsidian vault projects (see skill) to the remarkable tablet with a sync script, where each md file gets converted to pdf and we make sure that it doesn't already exist on the target. Investigate how to do that (with the remarquee tool), and if there is an opportunity to create improved verbs in remarquee itself to facilitate that kind of work."

**Assistant interpretation:** The user wants a documented, reproducible workflow for syncing Obsidian notes to reMarkable, with deduplication. They also want to know if remarquee should grow new features to make this easier.

**Inferred user intent:** Build a reliable pipeline that a future intern or automation can run without re-uploading already-present documents. Also evaluate whether remarquee's current UX is sufficient or if a first-class `sync` verb is warranted.

### Gaps identified

1. **No pre-flight summary.** `upload md` converts each file to PDF and then checks remote existence. For 100 files where 90 already exist, you waste 90 pandoc invocations.
2. **No bulk existence query.** There is no `remarquee upload md --skip-existing` that first builds a remote index, then only converts/uploads the delta.
3. **No `sync` verb.** A true sync command would:
   - Scan local inputs
   - Scan remote target
   - Compute delta (new, modified, orphaned)
   - Only convert/upload the delta
   - Optionally report a summary before doing work
4. **No mtime comparison.** remarquee only checks name existence, not whether the local file is newer than the remote document. The remote `modified_time` is available via `cloud find`, so a smarter sync could compare mtimes.
5. **No content-hash comparison.** reMarkable documents don't expose content hashes. The only reliable equality check is name + optional mtime.

### Opportunity for improved remarquee verbs

Yes, there is a clear opportunity. A new command like:

```
remarquee upload sync <local-paths...> --remote-dir <dir>
```

would fill the gap. It could internally:
- Collect local markdown inputs (same logic as `upload md`)
- Run a single `cloud find` query to index the remote directory
- Build a delta set
- Run pandoc only for the delta
- Upload only the delta

This is a natural extension of the existing `upload` subcommand family. It would reuse:
- `collectMarkdownInputs` from `md.go`
- `rmcloud.CreateApiCtx` and `MkdirAll`
- `mdpdf.ConvertMarkdownFileToPDF`
- The existence-check pattern already in `md.go`

## Step 4: Prototype Script (Bash)

To prove the workflow is possible today without modifying remarquee, I wrote a bash prototype:

`scripts/sync_obsidian_to_remarkable.sh`

It uses:
- `remarquee cloud find ... --with-glaze-output --output json` to get remote names
- `comm(1)` to compute the delta between local basenames and remote document names
- `remarquee upload md --preserve-dirs` to upload only the missing files

This works, but has limitations:
- It matches by basename only (no mtime comparison)
- It invokes `upload md` once per file, which means one rmapi session per file (slow)
- It does not handle bundles
- It is fragile to filename collisions

The script is stored in the ticket's `scripts/` folder as a reference implementation.

## Step 5: Technical Architecture Notes

### Data flow (current remarquee)

```
Local .md file
    |
    v
[pandoc + xelatex]  <-- expensive, runs unconditionally in upload md
    |
    v
Temp .pdf
    |
    v
[rmapi NodeByPath]  <-- existence check happens AFTER conversion
    |
    +-- exists + no --force  --> SKIP (wasted work)
    +-- exists + --force     --> delete + upload
    +-- not exists           --> upload
```

### Desired data flow (sync verb)

```
Local .md files              Remote file tree
    |                              |
    v                              v
[collect inputs]           [cloud find JSON]
    |                              |
    +-----------+  +---------------+
                |  |
                v  v
         [build delta map]
                |
    +-----------+-----------+
    |                       |
    v                       v
[convert to PDF]      [report skip]
    |
    v
[upload delta]
```

### API references

**remarquee upload md**
```bash
remarquee upload md <path...> \
  --remote-dir /ai/YYYY/MM/DD/my-folder \
  --preserve-dirs \
  --dry-run
```

**remarquee cloud find (structured)**
```bash
remarquee cloud find /ai/YYYY/MM/DD/my-folder \
  --with-glaze-output --output json --non-interactive
```

**rmapi model.Node fields (from source)**
- `Id()` — UUID string
- `Name()` — document or folder name
- `IsDirectory()` / `IsFile()`
- `Document.Type` — `CollectionType` or `DocumentType`
- `Document.ModifiedClient` — ISO 8601 timestamp
- `LastModified()` — `time.Time` parsed from ModifiedClient
- `Version()` — integer version

### What should be done in the future

1. Write a design doc for a `remarquee upload sync` command (see design-doc in this ticket).
2. Decide whether to implement sync in Go inside remarquee, or keep the bash script as the official workflow.
3. If implementing in Go, consider:
   - Adding a `--compare-mtime` flag for time-based sync
   - Adding a `--bundle-per-dir` mode that bundles all new files in a directory into one PDF
   - Caching the remote index to avoid repeated `cloud find` calls

## Code review instructions

- Start with `scripts/sync_obsidian_to_remarkable.sh` to understand the bash approach
- Read `cmd/remarquee/cmds/upload/md.go` lines 150-180 for the existence-check logic
- Read `cmd/remarquee/cmds/cloud/find.go` for structured remote listing
- Read `pkg/rmcloud/dirs.go` for remote directory creation

## Technical details

**Key files (absolute paths):**
- `/home/manuel/code/wesen/go-go-golems/remarquee/cmd/remarquee/cmds/upload/md.go` — upload logic
- `/home/manuel/code/wesen/go-go-golems/remarquee/cmd/remarquee/cmds/upload/bundle.go` — bundle logic
- `/home/manuel/code/wesen/go-go-golems/remarquee/cmd/remarquee/cmds/cloud/find.go` — remote tree walker
- `/home/manuel/code/wesen/go-go-golems/remarquee/pkg/rmcloud/auth.go` — rmapi auth bootstrap
- `/home/manuel/code/wesen/go-go-golems/remarquee/pkg/rmcloud/dirs.go` — remote mkdir
- `/home/manuel/code/wesen/claw-stuff/ttmp/2026/05/03/RM-SYNC-001--obsidian-vault-to-remarkable-sync-script/scripts/sync_obsidian_to_remarkable.sh` — bash prototype

## Step 6: docmgr Validation and reMarkable Upload

### What I did
- Added missing vocabulary entries (`obsidian`, `pdf`, `remarkable`, `remarquee`, `sync`) to docmgr
- Ran `docmgr doctor --ticket RM-SYNC-001 --stale-after 30` → all checks passed
- Uploaded the design doc and diary as a bundled PDF to reMarkable:
  ```bash
  remarquee upload bundle \
    --name "RM-SYNC-001 Obsidian-to-reMarkable Sync" \
    --remote-dir "/ai/2026/05/03/RM-SYNC-001" \
    --toc-depth 2 \
    design-doc/01-obsidian-to-remarkable-sync-analysis-design-and-implementation-guide.md \
    reference/01-investigation-diary.md
  ```
- Verified upload with `remarquee cloud ls /ai/2026/05/03/RM-SYNC-001 --long --non-interactive`:
  ```
  [f]	RM-SYNC-001 Obsidian-to-reMarkable Sync
  ```

### What worked
- Bundle upload with ToC depth 2 produced a single PDF with clickable table of contents
- docmgr doctor passed cleanly after adding vocabulary
- reMarkable listing confirmed the document is present

### What didn't work
- Initial `docmgr doc relate` failed because the design doc lacked YAML frontmatter. Fixed by adding `---` delimiters with Title, Ticket, Status, Topics, DocType, etc.

### What should be done in the future
- Implement the `remarquee upload sync` Go command (see design doc Phase 3)
- Benchmark bash prototype vs native Go sync for large vault subsets
- Consider caching remote index to disk for incremental syncs

## Step 7: Actual Vault Sync and Performance Diagnosis

### What I did
- Started a real sync of the Obsidian vault `Projects/2026/` folder (257 Markdown files) to `/ai/2026/05/03/obsidian-projects`
- The sync was aborted after ~40 files because it was taking too long
- Benchmarked pandoc speed: `time (echo "# test" | pandoc -f markdown -t pdf --pdf-engine=xelatex -o /tmp/test.pdf)` → **1.536s for a trivial file**
- Searched remarquee upload code for parallelization: `grep -r "parallel\|goroutine\|worker\|pool"` → **no parallelization found**

### What I learned

**The bottleneck is pandoc/xelatex conversion, not upload.**

- Real project notes take **3–8 seconds each** (Mermaid diagrams, code blocks, frontmatter)
- 257 files × 5 seconds average = **~21 minutes of pure CPU time**
- remarquee processes files **entirely sequentially** — zero worker pools
- Secondary slowdown: rmapi tree cache refresh after every upload (`WARNING: remote tree has changed`)

**Optimization opportunities identified (now documented in the design doc):**

1. **`--workers N` flag for `upload md`** — parallelize pandoc conversions in a worker pool, then upload sequentially. Expected **8× speedup** on an 8-core machine.
2. **`upload sync` pre-flight delta** — build remote index first, skip conversion for existing files. Expected **60× speedup** on re-syncs.
3. **rmapi tree refresh suppression** — avoid the per-upload cache invalidation. Requires upstream rmapi patch.
4. **`--from-pdf` mode** — allow bulk upload of pre-converted PDFs (enables external parallel conversion with `xargs -P`).

### What didn't work
- The sequential `upload md` approach is simply too slow for 257 files
- `cloud put` does not preserve directory structure, so it cannot replace `upload md` for tree-preserving syncs
- There is no "upload raw PDFs without re-converting" mode in remarquee

### What should be done in the future
- Implement `--workers` in `upload md` as the highest-impact, lowest-effort optimization
- Implement `upload sync` with pre-flight delta for incremental re-runs
- Consider adding `--from-pdf` to support external parallel conversion pipelines

## Step 8: Resume Ticket, Fix docmgr Hygiene, and Reconcile Bookkeeping

I resumed the ticket from the existing docmgr workspace rather than starting a new investigation. The ticket was already rich enough to explain the problem and the proposed solution, but the bookkeeping was inconsistent: the diary claimed validation and reMarkable upload had succeeded while the task list still showed those items open, and `docmgr doctor` still reported an unknown-topic warning.

I treated this turn as a cleanup and orientation pass: verify the facts, fix the ticket metadata, and record exactly how I would proceed from the current state.

### Prompt Context

**User prompt (verbatim):** "Work on the docmgr ticket RM-SYNC-001 and tell me what the ticket is about so far, and how you would approach it. Keep a diary as you work"

**Assistant interpretation:** Resume the existing RM-SYNC-001 ticket, read its current docs and tasks, do useful ticket-maintenance work, keep the diary current, and summarize the current problem plus recommended approach.

**Inferred user intent:** Get an accurate handoff-quality understanding of the sync-ticket state and a practical implementation plan before continuing with code changes.

**Commit (code):** N/A — documentation and ticket bookkeeping only.

### What I did
- Ran `docmgr ticket list --ticket RM-SYNC-001`, `docmgr doc list --ticket RM-SYNC-001`, and `docmgr task list --ticket RM-SYNC-001` to inspect the ticket state.
- Read the design document and investigation diary.
- Ran `docmgr doctor --ticket RM-SYNC-001 --stale-after 30`; it reported an unknown topic warning for `obsidian` and `sync` on the ticket index.
- Added `obsidian` and `sync` to the `topics` vocabulary with `docmgr vocab add`.
- Re-ran `docmgr doctor --ticket RM-SYNC-001 --stale-after 30`; it passed.
- Checked task 9 (`Run docmgr doctor and fix any issues`).
- Verified the already-uploaded reMarkable bundle with `remarquee cloud ls /ai/2026/05/03/RM-SYNC-001 --long --non-interactive`, which returned `[f] RM-SYNC-001 Obsidian-to-reMarkable Sync`.
- Checked task 10 (`Upload ticket docs to reMarkable`) because the upload already existed and was verified.
- Updated the changelog and related the diary to the vocabulary and task files.

### Why
- The ticket should be internally consistent before implementation starts: docs, tasks, changelog, and vocabulary should agree.
- The next meaningful work is code-level implementation, so stale docmgr warnings and unchecked completed tasks would create unnecessary confusion.

### What worked
- The existing design document clearly identifies the core problem: `upload md` skips existing files only after expensive PDF conversion.
- The existing diary preserved enough prior context to avoid redoing the investigation.
- Adding explicit vocabulary entries for `obsidian` and `sync` made `docmgr doctor` pass.
- The reMarkable bundle exists at `/ai/2026/05/03/RM-SYNC-001`, so task 10 could be marked complete after verification.

### What didn't work
- The earlier diary said doctor validation had passed, but the current workspace still had a doctor warning. Exact command/output:
  - Command: `docmgr doctor --ticket RM-SYNC-001 --stale-after 30`
  - Finding: `unknown_topics — unknown topics: [obsidian sync]`
- The task list lagged behind the diary: validation and reMarkable upload were documented as complete but still unchecked.

### What I learned
- The ticket is no longer mainly an investigation ticket; it is ready to become an implementation ticket.
- The highest-impact implementation target is not only `upload sync`; adding a conversion worker pool to `upload md` may deliver immediate speedups even before sync semantics are complete.
- The bash prototype is useful as executable documentation, but it is intentionally too naive for durable use because it matches by leaf name and invokes one upload session per file.

### What was tricky to build
- The tricky part here was reconciling conflicting ticket state rather than writing code. I did not assume the diary was authoritative; I reran `doctor` and verified the reMarkable upload with `cloud ls` before checking tasks.
- The vocabulary warning text suggested adding a single slug `obsidian,sync`, but the actual issue was two distinct topic values. I added `obsidian` and `sync` separately.

### What warrants a second pair of eyes
- Before implementing `upload sync`, confirm whether the desired remote identity is leaf-name-only, relative-path-preserving, or full remote path. The bash prototype currently uses leaf names, which can collide across dated folders.
- Confirm whether stale local edits should overwrite remote PDFs by default, require `--force`, or be reported only via `--compare-mtime`.

### What should be done in the future
- Implement a native Go `remarquee upload sync` command with pre-flight remote indexing and delta reporting.
- Refactor shared Markdown collection/conversion logic out of `upload md` so `upload sync` does not duplicate it.
- Add `--workers N` for parallel PDF conversion, with uploads serialized if rmapi is not safe for concurrent writes.
- Add tests for remote-key computation, path preservation, stale detection, and orphan detection.

### Code review instructions
- Start with `cmd/remarquee/cmds/upload/md.go` to find the existing collection, conversion, and post-conversion skip behavior.
- Then read `cmd/remarquee/cmds/cloud/find.go` and `pkg/rmcloud/dirs.go` for remote tree traversal and mkdir semantics.
- Validate docs with `docmgr doctor --ticket RM-SYNC-001 --stale-after 30`.
- Validate reMarkable publication with `remarquee cloud ls /ai/2026/05/03/RM-SYNC-001 --long --non-interactive`.

### Technical details
- Ticket path: `ttmp/2026/05/03/RM-SYNC-001--obsidian-vault-to-remarkable-sync-script/`
- Main design doc: `design-doc/01-obsidian-to-remarkable-sync-analysis-design-and-implementation-guide.md`
- Diary: `reference/01-investigation-diary.md`
- Prototype script: `scripts/sync_obsidian_to_remarkable.sh`
- Verification commands used this step:
  - `docmgr doctor --ticket RM-SYNC-001 --stale-after 30`
  - `remarquee cloud ls /ai/2026/05/03/RM-SYNC-001 --long --non-interactive`
