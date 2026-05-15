---
Title: Analysis: Pandoc errors abort sync
Ticket: RMQ-0015
Status: active
Topics:
    - remarquee
    - upload
    - pandoc
    - resilience
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - /home/manuel/code/wesen/corporate-headquarters/remarquee/cmd/remarquee/cmds/upload/sync.go:sync executeSyncPlan returns on first pandoc error
    - /home/manuel/code/wesen/corporate-headquarters/remarquee/cmd/remarquee/cmds/upload/conversion_workers.go:convertMarkdownJobs cancels all workers on first error
    - /home/manuel/code/wesen/corporate-headquarters/remarquee/cmd/remarquee/cmds/upload/md.go:runUploadMarkdown returns on first pandoc/upload error
ExternalSources: []
Summary: "Pandoc conversion failures (e.g. LaTeX 'Too deeply nested') abort the entire upload sync instead of skipping broken files and continuing."
LastUpdated: 2026-05-15
WhatFor: "Understanding the root cause and designing a resilient error-handling strategy for batch uploads"
WhenToUse: "Reference when implementing error resilience in upload commands"
---

# Analysis: Pandoc Errors Abort Upload Sync

## Problem Statement

When running `remarquee upload sync go-go-parc --remote-dir /PARC`, a single pandoc conversion failure (e.g. LaTeX "Too deeply nested" error, exit status 43) aborts the entire sync. The user must manually identify which file failed, remove it, and re-run. For large directory trees with hundreds of markdown files, this is painful and time-consuming.

### Example Error

```
Error: pandoc failed: Error producing PDF.
! LaTeX Error: Too deeply nested.
...
l.95         \tightlist
: exit status 43
```

## Root Cause Analysis

### 1. `sync.go` — `executeSyncPlan()`

The critical line is:

```go
if err := mdpdf.ConvertMarkdownFileToPDF(ctx, item.Local.Input.AbsPath, outPDF, pandocOpts); err != nil {
    return err  // ← aborts the entire sync
}
```

When any single file's pandoc conversion fails, the error propagates up and the loop exits. All remaining UPLOAD items are abandoned.

Similarly, upload errors also abort:

```go
if err := uploadPDFToRemote(cmd, apiCtx, dstNodeCache, item.Local.RemoteDir, outPDF, item.Local.PDFName); err != nil {
    return err  // ← also aborts the entire sync
}
```

### 2. `conversion_workers.go` — `convertMarkdownJobs()`

In parallel (multi-worker) mode, the first pandoc error cancels the context and all workers stop:

```go
once.Do(func() {
    errCh <- err
    cancel()  // ← all workers stop
})
```

### 3. `md.go` — `runUploadMarkdown()`

Same pattern in the single-worker upload loop:

```go
if err := mdpdf.ConvertMarkdownFileToPDF(ctx, job.Input.AbsPath, job.OutPDF, pandocOpts); err != nil {
    return err  // ← aborts
}
```

And upload errors:

```go
if err := uploadPDFToRemote(cmd, apiCtx, dstNodeCache, dst, job.OutPDF, job.PDFName); err != nil {
    return err  // ← aborts
}
```

## Design: Resilient Error Handling

### Strategy

- **Continue on error**: Pandoc and upload errors should be logged but should not abort the batch.
- **Collect errors**: Track which files failed and why.
- **Report at end**: Print a summary of all failures after processing completes.
- **Exit code**: Return a non-zero exit code if any files failed (even if some succeeded), so CI/scripts can detect partial failures.

### Changes Per File

#### `sync_plan.go` — Add `syncActionError`

Add a new `syncActionError` constant to represent files that failed during execution. This isn't a plan-time action — it's an execution-time state — but using the same type lets `printSyncPlan` and `Count` work for the final summary.

#### `sync.go` — `executeSyncPlan()`

- Replace `return err` with error collection in the pandoc conversion and upload steps.
- Log `ERROR-CONVERT: <path> — <error>` for pandoc failures.
- Log `ERROR-UPLOAD: <path> — <error>` for upload failures.
- After the loop, print a summary: `ERRORS: convert-failed=<N> upload-failed=<M>`.
- Return a combined error if any failures occurred.

#### `conversion_workers.go` — `convertMarkdownJobs()`

- Instead of cancel-on-first-error, collect errors per file in a slice.
- Each worker sends errors to a collected errors slice (mutex-protected or channel-based).
- After all workers finish, return a combined error if any failed.
- Add a `ConversionResult` type to track per-file success/failure.

#### `md.go` — `runUploadMarkdown()`

- Same pattern: collect errors, log, continue.
- Print summary at end.

### Error Reporting Format

```
ERROR-CONVERT: /path/to/broken.md — pandoc failed: ... (exit status 43)
ERROR-UPLOAD: /path/to/ok.pdf — failed to upload: connection reset
...
ERRORS: convert-failed=2 upload-failed=0
```

### Exit Code Behavior

- 0 = all files succeeded
- 1 = some files failed (partial success)
