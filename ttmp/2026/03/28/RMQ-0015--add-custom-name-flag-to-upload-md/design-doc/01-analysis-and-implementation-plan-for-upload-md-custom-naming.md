---
Title: Analysis and implementation plan for upload md custom naming
Ticket: RMQ-0015
Status: complete
Topics:
    - remarkable
    - upload
    - markdown
    - pdf
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/remarquee/cmds/upload/md.go
      Note: Primary implementation of upload-md custom naming
    - Path: cmd/remarquee/cmds/upload/md_test.go
      Note: Focused tests for dry-run naming and multi-file rejection
    - Path: pkg/doc/upload/02-remarquee-upload-reference.md
      Note: Embedded docs updated for the new flag
ExternalSources: []
Summary: Evidence-backed analysis of how `upload md` derives names today, why `--name` should be constrained to single-document use, and how the implementation is validated.
LastUpdated: 2026-03-28T10:01:52.712142775-04:00
WhatFor: Explain the rationale and implementation plan for adding a custom output-name override to `remarquee upload md`.
WhenToUse: Use this document before changing upload-md naming behavior, collision detection, or CLI semantics around multi-file uploads.
---


# Analysis and implementation plan for upload md custom naming

## Executive Summary

`remarquee upload md` previously always derived its output document name from the source markdown filename. That made it impossible to upload a single markdown file under a temporary or editorial name without renaming the local file first. The new `--name` flag fixes that workflow gap.

The important design constraint is scope: `--name` only works when the expanded input set contains exactly one markdown file. That keeps the flag comprehensible in all three execution modes: dry-run, `--pdf-only`, and actual upload. The final implementation threads the override through the existing naming path, validates the multi-file case early, updates the embedded docs, and adds focused command tests.

## Problem Statement

Observed behavior before the change:

1. `upload md` accepts files or directories, recursively expands directories to markdown files, and derives each output PDF name from the input filename in [`cmd/remarquee/cmds/upload/md.go:97-130`](../../../../../../cmd/remarquee/cmds/upload/md.go).
2. The generated PDF filename is then reused for dry-run output, local `--pdf-only` rendering, and remote upload naming in [`cmd/remarquee/cmds/upload/md.go:146-287`](../../../../../../cmd/remarquee/cmds/upload/md.go).
3. The embedded user-facing docs mention many flags for `upload md`, but there was no documented way to override the output document name in [`pkg/doc/upload/02-remarquee-upload-reference.md:38-45`](../../../../../../pkg/doc/upload/02-remarquee-upload-reference.md).

That meant the only way to choose a different remote name was to rename or copy the markdown file locally before running the command. The user request was to make the naming override first-class.

## Proposed Solution

Add a `--name` flag to `remarquee upload md` and apply it as an output PDF/document-name override when exactly one markdown file is selected after input expansion.

Final semantics:

1. `--name` is accepted by `upload md`.
2. The value may be passed as `foo` or `foo.pdf`; the implementation normalizes it with `ensurePDFSuffix(...)`.
3. If the expanded markdown input set contains more than one file, the command returns:
   - `--name is only valid when exactly one markdown file is selected`
4. The override affects:
   - dry-run output,
   - `--pdf-only` output filename,
   - uploaded remote document name.

### API Sketch

```go
type uploadMarkdownSettings struct {
    ...
    Name string
}

func markdownPDFName(in markdownInput, override string, total int) (string, error)
```

### Pseudocode

```text
collect markdown inputs
if no inputs:
  error

for each markdown input:
  pdfName = markdownPDFName(input, --name, totalInputs)
  if totalInputs > 1 and --name != "":
    error

use pdfName consistently for:
  collision detection
  dry-run output
  pdf-only output path
  temp upload path -> util.DocPathToName(outPDF)
```

## Design Decisions

### Decision 1: `--name` is single-document only

This is the key design choice. A single string cannot map cleanly onto multiple PDFs discovered from a directory or passed as separate files. Instead of inventing partial or positional behavior, the command rejects the multi-file case early.

Reasoning:

1. It keeps the CLI behavior obvious.
2. It avoids hidden precedence rules with `--preserve-dirs`.
3. It matches the user’s real use case: “pick the name for this upload.”

### Decision 2: reuse the existing PDF filename as the naming source of truth

The implementation introduces `markdownPDFName(...)` in [`cmd/remarquee/cmds/upload/md.go:290-299`](../../../../../../cmd/remarquee/cmds/upload/md.go) and uses it in every path that previously derived `foo.pdf` ad hoc.

Reasoning:

1. Dry-run, local PDF generation, and upload mode stay aligned.
2. Collision detection uses the same resolved name as upload mode.
3. The feature remains a small change instead of a structural refactor.

### Decision 3: use the existing bundle naming convention

The implementation reuses `ensurePDFSuffix(...)`, which already normalizes bundle names in the same package.

Reasoning:

1. Users can pass either `draft-copy` or `draft-copy.pdf`.
2. The naming model stays consistent between `upload bundle` and `upload md`.

## Alternatives Considered

### Alternative 1: allow `--name` with multiple files and apply it only to the first

Rejected because it is surprising and hard to justify.

### Alternative 2: allow `--name` with multiple files by prefixing or templating

Rejected because it turns a simple naming override into a templating feature. That is a separate product decision and not needed for this request.

### Alternative 3: document “rename the local file first” and add no flag

Rejected because it preserves the workflow friction the user explicitly asked to remove.

## Implementation Plan

1. Add `Name string` to `uploadMarkdownSettings`.
2. Register `--name` in `NewUploadMarkdownCommand()`.
3. Add a helper that resolves the output PDF name from:
   - the input filename, or
   - the custom override when exactly one markdown file is selected.
4. Replace the four duplicated “base filename + .pdf” call sites with the shared helper:
   - collision detection,
   - dry-run,
   - `--pdf-only`,
   - upload mode.
5. Add tests for:
   - single-file dry-run with a custom name,
   - rejection when `--name` is used with multiple markdown files.
6. Update the embedded `upload md` reference docs.

## Validation Strategy

Focused validation used for this ticket:

```bash
go test ./pkg/mdpdf ./cmd/remarquee/cmds/upload
tmpdir=$(mktemp -d) && printf '# Draft\n' > "$tmpdir/note.md" && go run ./cmd/remarquee upload md --dry-run --pdf-only --name "editor-copy" "$tmpdir/note.md"
```

Observed dry-run output:

```text
DRY: layout=default
DRY: remote-dir=/ai/2026/03/28
DRY: pandoc /tmp/.../note.md -> editor-copy.pdf
```

## Open Questions

1. Do we eventually want a multi-file naming template for `upload md`, or is single-file override enough?
2. Should `upload src` grow a similar `--name` override for its one-file mode, or should that remain a separate ticket?

## References

1. [`cmd/remarquee/cmds/upload/md.go`](../../../../../../cmd/remarquee/cmds/upload/md.go)
2. [`cmd/remarquee/cmds/upload/md_test.go`](../../../../../../cmd/remarquee/cmds/upload/md_test.go)
3. [`pkg/doc/upload/02-remarquee-upload-reference.md`](../../../../../../pkg/doc/upload/02-remarquee-upload-reference.md)
