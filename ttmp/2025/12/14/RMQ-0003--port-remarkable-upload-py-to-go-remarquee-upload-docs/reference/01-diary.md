---
Title: Diary
Ticket: RMQ-0003
Status: active
Topics:
    - backend
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/remarquee/cmds/cloud/put.go
      Note: Existing upload semantics to mirror
    - Path: cmd/remarquee/cmds/cloud/rmapi.go
      Note: Current rmapi bootstrap helper to refactor
    - Path: cmd/remarquee/cmds/upload/md.go
      Note: Implementation of 'remarquee upload md' (commit d70169e...)
    - Path: cmd/remarquee/cmds/upload/root.go
      Note: Upload command group wiring (commit d70169e...)
    - Path: cmd/remarquee/main.go
      Note: Wiring point for new upload command group
    - Path: pkg/mdpdf/pandoc.go
      Note: Pandoc/xelatex runner (commit d70169e...)
    - Path: pkg/mdpdf/preprocess.go
      Note: Frontmatter stripping + list normalization (commit d70169e...)
    - Path: pkg/rmcloud/auth.go
      Note: Shared rmapi CreateApiCtx helper (commit d70169e...)
    - Path: pkg/rmcloud/dirs.go
      Note: Recursive remote mkdir helper MkdirAll (commit d70169e...)
    - Path: ttmp/2025/12/14/RMQ-0003--port-remarkable-upload-py-to-go-remarquee-upload-docs/design-doc/01-port-remarkable-upload-py-to-go-remarquee-upload-docs.md
      Note: Spec/design for the implementation steps
ExternalSources: []
Summary: Implementation diary for RMQ-0003 (port remarkable_upload.py into remarquee as a general-purpose markdown uploader).
LastUpdated: 2025-12-14T20:55:51.275801633-05:00
---



# Diary

## Goal

Track RMQ-0003 implementation as a narrative (what changed, why, what worked, what failed, and what to do next), with enough detail that a new contributor can continue mid-stream.

## Step 1: Add `remarquee upload md` (general-purpose markdown uploader) + factor rmapi bootstrap

This step made the “upload docs to reMarkable” workflow first-class in the `remarquee` Go CLI. The main outcome is a new command, `remarquee upload md`, that takes **only markdown files and/or directories** (directories are recursively scanned for `*.md`), converts markdown to PDF via **pandoc + xelatex**, and uploads via rmapi **in-process** (no shelling out to `rmapi`).

To keep auth handling consistent across commands, we also factored rmapi bootstrap (`CreateApiCtx`) into a small reusable package (`pkg/rmcloud`) and made the existing cloud commands delegate to it.

**Commit (code):** d70169eee359fe706bbcb955819d4fdc89167374 — "remarquee: add upload md (pandoc + rmapi)"

### What I did
- Added rmapi bootstrap helper:
  - `pkg/rmcloud/auth.go` (`CreateApiCtx`)
- Added recursive remote mkdir helper:
  - `pkg/rmcloud/dirs.go` (`MkdirAll`)
- Added markdown → PDF conversion helpers:
  - `pkg/mdpdf/preprocess.go` (frontmatter strip + list spacing normalization)
  - `pkg/mdpdf/pandoc.go` (pandoc/xelatex runner with DejaVu defaults)
- Added new command group + command:
  - `cmd/remarquee/cmds/upload/root.go` (`remarquee upload`)
  - `cmd/remarquee/cmds/upload/md.go` (`remarquee upload md`)
- Wired upload into the CLI:
  - `cmd/remarquee/main.go`
- Refactored existing cloud bootstrap to use shared helper:
  - `cmd/remarquee/cmds/cloud/rmapi.go` now delegates to `pkg/rmcloud`
- Validated build:
  - `gofmt -w ...`
  - `go test ./... -count=1`

### Why
- Make the “write path” workflow (Markdown → PDF → upload) part of the `remarquee` binary, with consistent UX and auth.
- Keep PDF output quality identical to the proven Python script by continuing to use pandoc/xelatex.
- Avoid brittle stdout parsing by using rmapi as a Go library.

### What worked
- `go test ./...` succeeded after adding the new packages/command wiring.
- `remarquee upload md` compiles and is wired into the root command.

### What didn't work
- Initial build failed due to an unused variable in `cmd/remarquee/cmds/upload/md.go`:
  - `declared and not used: _ext`
  - Fixed by assigning to `_` and re-running `go test`.

### What I learned
- In this repo layout, git lives under `remarquee/.git` (not at the workspace root), so commits must be made from the `remarquee/` directory.
- It’s worth failing fast on duplicate markdown basenames when uploading from directories, because rmapi document naming is basename-stem-based and collisions are otherwise surprising.

### What was tricky to build
- Getting rmapi “directory creation” semantics right: root uses an empty parent id, but non-root uses `node.Id()`.
- Keeping path normalization consistent: user-friendly inputs may include or omit leading/trailing slashes, while rmapi path resolution behaves best with normalized absolute-like paths.
- Preserving behavioral parity with the Python script’s preprocessing without over-engineering (simple delimiter-based frontmatter removal is a feature).

### What warrants a second pair of eyes
- `pkg/rmcloud/MkdirAll` correctness and edge cases:
  - behavior when intermediate path segment exists as a file
  - behavior with `..` and `.` segments
- `remarquee upload md --force` safety wording: it deletes the existing document (and annotations) before upload; confirm the UX is appropriately explicit.

### What should be done in the future
- Add unit tests for:
  - `StripYAMLFrontmatter` and `NormalizeListSpacing`
  - directory recursion behavior
  - date parsing/formatting and remote dir normalization
- Add embedded help docs for the new `upload` command group (`pkg/doc/upload/*`) and extend `pkg/doc/doc.go` embed patterns.

### Code review instructions
- Start at:
  - `cmd/remarquee/cmds/upload/md.go` (CLI UX + recursion + upload loop)
  - `pkg/mdpdf/pandoc.go` and `pkg/mdpdf/preprocess.go` (conversion + preprocessing)
  - `pkg/rmcloud/auth.go` and `pkg/rmcloud/dirs.go` (rmapi plumbing)
- Validate by running:
  - `go test ./... -count=1`
  - `go run ./cmd/remarquee upload md --dry-run <some.md>`

### Technical details
- Default remote upload destination: `/ai/YYYY/MM/DD/` (today) unless overridden by `--date` or `--remote-dir`.
- Inputs:
  - `remarquee upload md file.md`
  - `remarquee upload md ./some-directory` (recursive scan for `*.md`)

### What I'd do differently next time
- Add a tiny smoke test that only builds the CLI packages (fast) so we catch simple compile issues before running the full `go test ./...`.
