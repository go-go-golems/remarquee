---
Title: Diary
Ticket: RMQ-0022
Status: active
Topics:
    - backend
    - upload
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/remarquee/cmds/upload/layout.go
      Note: shared helper wiring
    - Path: repo://pkg/mdpdf/pandoc.go
      Note: DefaultFromFormat + FromFormat fallback (commit 882cb3d)
    - Path: repo://pkg/mdpdf/pandoc_args_test.go
      Note: custom/empty FromFormat tests
ExternalSources: []
Summary: Chronological implementation diary for the --pandoc-from flag.
LastUpdated: 2026-08-10T00:00:00Z
WhatFor: Track implementation of configurable pandoc from-format.
WhenToUse: Read before resuming work on this ticket.
---


# Diary

## Goal

Record the implementation of a configurable pandoc `--from` format string (`--pandoc-from` flag) across remarquee's upload commands, fixing the silent override of custom pandoc wrapper scripts.

## Step 1: Design doc and implementation

Root cause (confirmed earlier the same day during a failed transcript upload): `buildPandocArgs` in `pkg/mdpdf/pandoc.go` hardcoded `--from=markdown-yaml_metadata_block`. Pandoc's "last `--from` wins" rule means this silently discards any `-f` set by a custom pandoc wrapper passed via `--pandoc` — the chatgpt-transcript-archiving skill's `pandoc-math` wrapper was exactly such a casualty.

The fix is a full-string `FromFormat` option (default unchanged) rather than an extension-append flag: simpler to implement, subsumes the append use case, and makes any override explicit. Implementation touched `pkg/mdpdf/pandoc.go`, the shared `configureMarkdownPandocOptions` helper, and all four upload subcommands.

### Prompt Context

**User prompt (verbatim):** "Ok, do that in remarquee. Add to the current remarquee ticket if you have one, otherwise create a new one"

**Assistant interpretation:** Implement the previously proposed remarquee fix (configurable `--from` / `--pandoc-from` flag) under a docmgr ticket — reuse an existing active ticket if one fits, else create one.

**Inferred user intent:** Close out the second half of the morning's root-cause analysis so custom pandoc wrappers (and math-heavy transcripts) work without surf-cli-side normalization.

**Commit (code):** 882cb3d — "feat(upload): add --pandoc-from flag for configurable pandoc input format"

### What I did
- Checked existing active tickets (RMQ-0014..0017, RMQ-0021, RMQ-001, REMARQUEE-IMPROVE) — none fit; created RMQ-0022
- `pkg/mdpdf/pandoc.go`: `DefaultFromFormat` const, `PandocOptions.FromFormat`, default + empty-fallback in `buildPandocArgs`
- `cmd/remarquee/cmds/upload/layout.go`: `configureMarkdownPandocOptions` takes `pandocFrom`, applied when `--pandoc-from` flag changed
- `md.go`, `bundle.go`, `sync.go`: `PandocFrom` setting + `--pandoc-from` flag (via perl edits over the repeated struct/flag/call patterns) + helper arg
- `src.go`: field + flag + direct `pandocOpts.FromFormat` assignment (doesn't use the shared helper)
- Tests: `TestBuildPandocArgsCustomFromFormat`, `TestBuildPandocArgsEmptyFromFormatFallsBackToDefault`
- Validation: built `/tmp/remarquee-test`, converted the RAW (un-normalized, 90 `\(` lines) "Semantics and Equalizers" transcript with `--pandoc-from "markdown-yaml_metadata_block+tex_math_single_backslash"` → `OK: generated transcript-raw.pdf` (121 KB)
- lefthook pre-commit: golangci-lint 0 issues, full test suite all ok

### Why
- Full format string over `--pandoc-ext` append flag: pandoc extensions are toggled only inside the `--from` string; exposing it directly is the least-magic option.
- Empty-fallback in `buildPandocArgs`: keeps programmatic callers (which never set the new field) on the old default.

### What worked
- First build compiled; perl multi-file edits hit all 12 intended sites (verified with grep before building).
- End-to-end proof: the same file that failed this morning with `! Missing $ inserted` now converts with one flag.

### What didn't work
- N/A — no failures this step.

### What I learned
- remarquee already had a `pandoc_args_test.go` asserting the hardcoded `--from` is first in argv — a good seam to extend rather than add a new test file.
- `src.go` bypasses `configureMarkdownPandocOptions`; flag wiring there is manual.

### What was tricky to build
- The struct-field/flag/call-site pattern repeats across three commands with near-identical text; did it with anchored perl regexes and verified each site with grep rather than risking divergent manual edits.

### What warrants a second pair of eyes
- `configureMarkdownPandocOptions` signature grew a positional param (now 10) — check the four call sites pass args in the right order.
- `--pandoc-from ""` (explicitly empty, changed) would set `FromFormat=""` and fall back to default — benign but slightly surprising.

### What should be done in the future
- Update the chatgpt-transcript-archiving skill: prefer `--pandoc-from` (or surf-cli's normalized output) over the `pandoc-math` wrapper.

### Code review instructions
- Start: `pkg/mdpdf/pandoc.go` (`DefaultFromFormat`, `buildPandocArgs`), then `cmd/remarquee/cmds/upload/layout.go` and one representative command (`md.go`).
- Validate: `go test ./pkg/mdpdf/ -run BuildPandocArgs -count=1`; end-to-end: `remarquee upload md <raw-chatgpt-transcript>.md --pdf-only --output-dir /tmp/x --pandoc-from "markdown-yaml_metadata_block+tex_math_single_backslash"`.

### Technical details
- Default format: `markdown-yaml_metadata_block` (unchanged; YAML metadata blocks stay disabled because frontmatter is stripped before pandoc runs).
- Usage example: `--pandoc-from "markdown-yaml_metadata_block+tex_math_single_backslash"` for ChatGPT `\(...\)` / `\[...\]` math.
