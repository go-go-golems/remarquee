---
Title: Diary
Ticket: RMQ-0015
Status: active
Topics:
    - remarquee
    - upload
    - pandoc
    - resilience
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - /home/manuel/code/wesen/corporate-headquarters/remarquee/cmd/remarquee/cmds/upload/sync.go:executeSyncPlan now collects errors instead of aborting
    - /home/manuel/code/wesen/corporate-headquarters/remarquee/cmd/remarquee/cmds/upload/conversion_workers.go:convertMarkdownJobs now collects errors per file instead of canceling all workers
    - /home/manuel/code/wesen/corporate-headquarters/remarquee/cmd/remarquee/cmds/upload/md.go:runUploadMarkdown upload loop now collects errors instead of aborting
    - /home/manuel/code/wesen/corporate-headquarters/remarquee/cmd/remarquee/cmds/upload/sync_plan.go:Added syncActionError constant
ExternalSources: []
Summary: "Implementation diary for making upload sync/md resilient to per-file errors"
LastUpdated: 2026-05-15
WhatFor: "Tracking the implementation journey for error-resilient upload commands"
WhenToUse: "Resume work on this ticket by reading this diary first"
---

# Diary

## Goal

Make remarquee upload sync and upload md commands resilient to per-file pandoc and upload errors, so that one broken file doesn't abort the entire batch.

## Step 1: Analyze and implement error resilience

Analyzed three source files (sync.go, conversion_workers.go, md.go) to find all places where pandoc or upload errors would abort the entire batch. Implemented a consistent pattern across all three: catch errors per file, log them with ERROR-CONVERT/ERROR-UPLOAD/ERROR-DELETE prefixes, continue processing, and print an ERRORS summary at the end.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to address pandoc errors / errors in sync. The issue I have is that `remarquee upload sync go-go-parc --remote-dir /PARC` in ~/code/wesen/go-go-golems fails with a pandoc error, but that shouldn't interrupt the upload sync, it should just skip the broken files and move on."

**Assistant interpretation:** Make the upload sync and upload md commands collect per-file errors instead of aborting on the first failure. Pandoc failures (e.g. "Too deeply nested") and upload failures should be logged and skipped.

**Inferred user intent:** When running batch sync of hundreds of markdown files, one broken file should not prevent the rest from being uploaded. The user wants to see which files failed and still get all the good files uploaded.

**Commit (code):** c30268c — "fix(upload): make sync and md commands resilient to per-file errors"

### What I did

- Added `syncActionError` to `sync_plan.go` for consistency in the plan summary
- Rewrote `executeSyncPlan` in `sync.go` to use a `failedItem` struct, collect errors per file, log them with ERROR-CONVERT/ERROR-UPLOAD/ERROR-DELETE, continue processing, and print ERRORS summary
- Rewrote `convertMarkdownJobs` in `conversion_workers.go` to use a `conversionError` struct, collect errors from all workers (mutex-protected), and format them at the end instead of cancel-on-first-error
- Updated `runUploadMarkdown` in `md.go` with the same error collection pattern for the upload loop
- Updated `printSyncPlan` SUMMARY line to include `error=N`
- Updated `TestPrintSyncPlanSummary` for new format

### Why

The original code used `return err` on any pandoc or upload failure, which aborted the entire batch. For large directory trees (like go-go-parc with many .md files), a single LaTeX error like "Too deeply nested" would stop everything, requiring manual identification and removal of the broken file.

### What worked

- The error collection pattern is consistent across all three files
- All existing tests pass
- Linter passes (golangci-lint)
- The `conversionError` type is package-level so `formatConversionErrors` can reference it properly

### What didn't work

- Initially had a bug where `errors = append(errs, ...)` shadowed the `errors` package import in conversion_workers.go. Fixed by renaming the slice to `convErrs`.
- The `conversionError` type was initially local inside the function, which made it incompatible with `formatConversionErrors`. Fixed by promoting it to package-level.

### What I learned

- The three upload paths (sync, md single-worker, md multi-worker) all had the same fail-fast pattern but slightly different implementations, requiring three separate but consistent fixes.
- The `conversion_workers.go` multi-worker mode used a `context.WithCancel` pattern for fail-fast, which had to be replaced entirely with a mutex-protected error collection.

### What was tricky to build

- Making the `conversionError` type accessible to `formatConversionErrors` — initially the type was local inside the function, then I tried an anonymous struct parameter, both of which were awkward. The clean solution was promoting the type to package level.
- Deciding which errors should still abort (infrastructure errors like MkdirAll, directory overwrite) vs. which should be collected (per-file pandoc/upload errors). I kept infrastructure errors as fatal and only made per-file operations resilient.

### What warrants a second pair of eyes

- The multi-worker `md.go` path: when `s.Workers > 1`, `convertMarkdownJobs` already processes all files and returns a combined error. The current code returns that error immediately. An improvement would be to still attempt uploads for jobs that succeeded (the workers already wrote PDFs to disk). This is left as a future improvement.
- The `syncActionError` in `printSyncPlan` is for plan-time display; execution-time errors are logged differently. These two error paths could be unified.

### What should be done in the future

- ~~For multi-worker md.go mode, attempt uploads for jobs whose PDF was successfully generated even if some conversions failed.~~ (Fixed in commit 2350d2a)
- Consider adding a `--continue-on-error` flag to allow users to opt into the old fail-fast behavior if desired.
- Add integration tests that actually test error resilience (e.g. by creating a broken .md file and verifying other files still get uploaded).

## Step 2: Fix multi-worker upload path and pdf-only exit code

After the initial implementation, I discovered two follow-up issues:

1. **Multi-worker upload path** (commit 2350d2a): When `s.Workers > 1` and `convertMarkdownJobs` returned an error, the code immediately returned that error, abandoning the upload loop entirely. But successful conversions already wrote their PDFs to disk. Fixed by logging the error and continuing to the upload loop, skipping jobs whose PDF wasn't generated (checked via `os.Stat`).

2. **pdf-only exit code** (commit e7d1acd): The `if err := convertMarkdownJobs(...)` pattern scoped the error inside the `if` block, so the final `if err != nil` check always saw `nil`. Fixed by assigning the error to a named variable `convErr` and checking that at the end.

3. **pdf-only partial success** (commit a4bf87e): Also fixed pdf-only mode to only report OK for PDFs that were actually generated, using `os.Stat` to verify existence.

### What I learned

- Go's `if err := ...` pattern creates a new scope — the error variable isn't accessible outside the block. This is a common footgun when trying to reference the error later.
- The `os.Stat` check is a pragmatic way to determine which conversions succeeded in multi-worker mode without needing the workers to report back per-job.

### Code review instructions

- Start with `sync.go` `executeSyncPlan` — the core change is the `failedItem` struct and error collection loop
- Then `conversion_workers.go` — `convertMarkdownJobs` now uses `conversionError` + mutex instead of cancel-on-first-error
- Then `md.go` — same pattern in the upload loop, plus the `os.Stat` check for multi-worker partial success
- Validate: `go test ./cmd/remarquee/cmds/upload/... -count=1 -v`
- Smoke test: build binary and run `upload md <dir-with-broken-md> --pdf-only --output-dir /tmp/test`
- Then `md.go` — same pattern in the upload loop
- Validate: `go test ./cmd/remarquee/cmds/upload/... -count=1 -v`

### Technical details

Error output format:
```
ERROR-CONVERT: /PARC/Research/KB/file — pandoc failed: Error producing PDF... (exit status 43)
ERROR-UPLOAD: /PARC/Research/KB/file — failed to upload: connection reset
ERRORS: convert-failed=2 upload-failed=0 delete-failed=0
```

Exit code: non-zero when any file fails (partial success detected by CI/scripts).
