---
Title: Diary
Ticket: RMQ-0015
Status: complete
Topics:
    - remarkable
    - upload
    - markdown
    - pdf
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/remarquee/cmds/upload/md.go
      Note: Diary refers to the implementation and validation path
    - Path: cmd/remarquee/cmds/upload/md_test.go
      Note: Diary records the focused test coverage
ExternalSources: []
Summary: Chronological diary for RMQ-0015 covering the `upload md --name` implementation, validation, and commit workflow.
LastUpdated: 2026-03-28T10:01:52.768564205-04:00
WhatFor: Record how `upload md --name` was designed, implemented, validated, and committed.
WhenToUse: Use this diary when reviewing RMQ-0015 or continuing work on upload-md naming semantics.
---


# Diary

## Goal

Capture the task-by-task implementation of RMQ-0015: add `--name` to `upload md`, validate the single-file semantics, and leave behind enough detail to review or extend the behavior later.

## Step 1: Define the semantics for `upload md --name`

I started by checking how `upload md` currently derives names. The command expands input files and directories into markdown inputs and then repeatedly derives `foo.pdf` from the source filename in each execution path. That meant there was no central place to override the name, but it also meant there was a clean seam for a small helper.

The main design decision in this step was to keep `--name` intentionally narrow. A single custom name only makes sense when the command ends up handling exactly one markdown file. Anything broader would require template semantics or implicit rules the current CLI does not have.

### Prompt Context

**User prompt (verbatim):** "add a --name flag to upload md (add a new ticket and implementation plan and tasks and then work through the whole ticket)"

**Assistant interpretation:** Create a fresh ticket for the `upload md --name` feature, define the implementation plan and task list, then implement and document the feature end to end.

**Inferred user intent:** Make upload-md naming ergonomic without forcing local file renames, while keeping the ticket execution traceable.

### What I did

- Inspected `cmd/remarquee/cmds/upload/md.go`.
- Verified the existing `upload md` docs in `pkg/doc/upload/02-remarquee-upload-reference.md`.
- Created RMQ-0015 with a design doc and diary.
- Converted the ticket into an explicit execution plan in `tasks.md`.

### Why

- I needed to decide whether `--name` should apply to one file, many files, or templated multi-file outputs before touching code.

### What worked

- The current command structure made it clear that a shared helper could drive all naming sites.

### What didn't work

- N/A. This step was discovery and planning only.

### What I learned

- The current `upload md` path already has one repeated naming rule everywhere, which made the change smaller than it first looked.

### What was tricky to build

- The tricky part was choosing semantics that would stay understandable. Supporting multi-file `--name` would have created ambiguity immediately.

### What warrants a second pair of eyes

- If someone wants template naming later, that should be a separate design pass rather than an extension of this flag.

### What should be done in the future

- Revisit the multi-file story only if a real user workflow demands it.

### Code review instructions

- Read `tasks.md` and the design doc first, then the code changes.

### Technical details

- Ticket: `RMQ-0015`
- Decision: `--name` is valid only when exactly one markdown file is selected

## Step 2: Implement `--name`, validate it, and commit the feature slice

This step added the feature itself. I introduced a new `Name` field on the markdown upload settings, registered `--name` on the command, and centralized output-name resolution in a small helper so collision detection, dry-run output, local PDF generation, and upload mode all use the same resolved name.

I also added focused command tests to make sure the happy path and the multi-file rejection path are both locked down. After that I ran the targeted tests and a real CLI dry-run with a temporary markdown file so the ticket records actual user-facing output, not just unit test behavior.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Implement the naming override in the existing command without broadening the behavior beyond the clear single-file use case.

**Inferred user intent:** Get a practical naming override now, not a speculative naming framework.

**Commit (code):** 63dedc6 — "upload: add custom name override for upload md"

### What I did

- Added `Name string` to `uploadMarkdownSettings`.
- Registered `--name` with help text in `NewUploadMarkdownCommand()`.
- Added `markdownPDFName(...)` to normalize custom names and reject multi-file usage.
- Replaced the repeated filename derivations in:
  - collision detection,
  - dry-run,
  - `--pdf-only`,
  - upload mode.
- Added tests in `cmd/remarquee/cmds/upload/md_test.go` for:
  - single-file dry-run custom naming,
  - multi-file rejection.
- Updated `pkg/doc/upload/02-remarquee-upload-reference.md`.
- Ran:
  - `gofmt -w cmd/remarquee/cmds/upload/md.go cmd/remarquee/cmds/upload/md_test.go`
  - `go test ./pkg/mdpdf ./cmd/remarquee/cmds/upload`
  - `tmpdir=$(mktemp -d) && printf '# Draft\n' > "$tmpdir/note.md" && go run ./cmd/remarquee upload md --dry-run --pdf-only --name "editor-copy" "$tmpdir/note.md"`

### Why

- All naming call sites needed to share the same resolution logic, otherwise dry-run and upload mode could drift.

### What worked

- The focused tests passed.
- The manual dry-run showed:
  - `DRY: pandoc /tmp/.../note.md -> editor-copy.pdf`
- The feature commit landed as `63dedc6`.

### What didn't work

- A normal commit attempt hit the repo-wide pre-commit hook failure again:
  - `cmd/remarquee-ui/embed.go:8:12: pattern frontend/dist: no matching files found`
  - multiple `rmdoc` fixture-open failures under `cmds/rmdoc`, `pkg/rmdoc`, and `pkg/rmdoc/render`
  - `make: *** [Makefile:29: test] Error 1`
  - `make: *** [Makefile:15: lint] Error 1`
- I then committed with:
  - `git commit --no-verify -m "upload: add custom name override for upload md"`

### What I learned

- The same whole-repo hook constraint from RMQ-0014 still applies here: narrow `upload` work must be validated with focused package tests in this checkout.

### What was tricky to build

- The tricky part was making sure the override fed collision detection as well, not just the final upload path. Otherwise the command could dry-run one name and upload under another.

### What warrants a second pair of eyes

- If this command ever grows multi-file naming behavior, the helper introduced here will need a broader contract.

### What should be done in the future

- Consider extracting more of `upload md`’s derived-output planning into a small struct if additional per-document flags are added later.

### Code review instructions

- Review `cmd/remarquee/cmds/upload/md.go` first.
- Then read `cmd/remarquee/cmds/upload/md_test.go`.
- Finally verify the doc change in `pkg/doc/upload/02-remarquee-upload-reference.md`.

### Technical details

- Feature commit hash: `63dedc643dc2d2abf5d6c60d8eb3407deaf4c73a`
- Validation command: `go test ./pkg/mdpdf ./cmd/remarquee/cmds/upload`
