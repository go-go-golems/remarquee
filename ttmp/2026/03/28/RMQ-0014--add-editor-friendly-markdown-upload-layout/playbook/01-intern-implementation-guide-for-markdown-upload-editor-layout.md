---
Title: Intern implementation guide for markdown upload editor layout
Ticket: RMQ-0014
Status: complete
Topics:
    - remarkable
    - upload
    - markdown
    - pdf
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/remarquee/cmds/upload/bundle.go
      Note: Bundled markdown upload path shares the same preset logic
    - Path: cmd/remarquee/cmds/upload/md.go
      Note: Primary markdown upload command interns should inspect first
    - Path: pkg/doc/upload/02-remarquee-upload-reference.md
      Note: User-facing docs must stay in sync with CLI behavior
    - Path: pkg/mdpdf/layout.go
      Note: Preset catalog to extend for new layouts
ExternalSources: []
Summary: Step-by-step intern guide for understanding, validating, and extending the markdown upload editor layout feature without breaking the default rendering path.
LastUpdated: 2026-03-28T09:28:20.148840528-04:00
WhatFor: Provide a repeatable implementation and validation guide for future contributors extending markdown upload layouts.
WhenToUse: Use this playbook when adding new markdown layout presets, changing pandoc option precedence, or validating annotation-oriented PDF output.
---


# Intern implementation guide for markdown upload editor layout

## Purpose

This playbook explains how to reason about the editor-friendly markdown upload layout feature, where to make changes, how to validate them, and what not to accidentally break. It is written for an intern who is new to the repo but needs enough detail to extend this area safely.

## Environment Assumptions

1. You are in the `remarquee` repository root.
2. Go tooling is installed.
3. For local PDF smoke tests, `pandoc` and `xelatex` are available on `PATH`.
4. For real reMarkable uploads, cloud auth is already configured or you are prepared to authenticate.

Key repo areas:

1. `cmd/remarquee/cmds/upload/` contains the CLI commands.
2. `pkg/mdpdf/` contains markdown preprocessing and pandoc execution.
3. `pkg/doc/upload/` contains embedded user-facing documentation.

## Commands

### 1. Rebuild your mental model of the feature

Read these files in order:

1. `pkg/mdpdf/layout.go`
2. `pkg/mdpdf/pandoc.go`
3. `cmd/remarquee/cmds/upload/layout.go`
4. `cmd/remarquee/cmds/upload/md.go`
5. `cmd/remarquee/cmds/upload/bundle.go`
6. `cmd/remarquee/cmds/upload/src.go`

What to notice:

1. Layout definitions live in `pkg/mdpdf`, not the Cobra commands.
2. The upload helper applies a preset first, then explicit overrides.
3. `upload src` is intentionally separate.

### 2. Run the focused tests before you touch anything

```bash
go test ./pkg/mdpdf ./cmd/remarquee/cmds/upload
```

If this is already failing, do not start implementing new work until you know why.

### 3. Inspect the current CLI behavior

```bash
go run ./cmd/remarquee upload md --help
go run ./cmd/remarquee upload bundle --help
go run ./cmd/remarquee upload md --dry-run --layout editor --pdf-only /abs/path/to/doc.md
go run ./cmd/remarquee upload bundle --dry-run --layout editor --pdf-only /abs/path/to/dir
```

Expected behavior:

1. `--layout` is visible in help for markdown upload commands.
2. Dry-run prints `DRY: layout=editor`.
3. No actual pandoc or upload work happens in dry-run mode.

### 4. If you are adding a new preset

Do the work in this order:

1. Add a new constant in `pkg/mdpdf/layout.go`.
2. Add a case in `ApplyMarkdownLayoutPreset`.
3. Decide whether the preset needs:
   - geometry only,
   - extra LaTeX header only,
   - or both.
4. Add or update tests in `pkg/mdpdf/layout_test.go`.
5. If the preset should be user-facing, update the `--layout` flag description only if the new value materially changes usage expectations.
6. Update the embedded docs in `pkg/doc/upload/`.

### 5. If you are changing precedence rules

Be careful here. The current contract is:

1. Start with `DefaultPandocOptions()`.
2. Apply the named preset.
3. Reapply explicit `--geometry` if the user changed it.
4. Reapply explicit `--latex-header-file` if the user changed it.

That behavior is implemented in `cmd/remarquee/cmds/upload/layout.go`. If you change it, you must update:

1. tests,
2. docs,
3. the design doc for this ticket.

### 6. Run local smoke tests for actual PDF generation

```bash
mkdir -p /tmp/rmq-layout-smoke
go run ./cmd/remarquee upload md \
  --pdf-only \
  --layout editor \
  --output-dir /tmp/rmq-layout-smoke \
  /abs/path/to/doc.md
```

Then visually inspect the PDF and confirm:

1. the text block is narrower than the default,
2. there is visibly more space on the right side for comments,
3. paragraph spacing is looser,
4. the document still looks readable.

### 7. Update docs and bookkeeping

After code changes, update:

```bash
docmgr doc relate --doc /abs/path/to/design-doc.md --file-note "/abs/path/to/file.go:reason"
docmgr doctor --ticket RMQ-0014 --stale-after 30
```

If you add a new layout that changes the user-facing behavior, also update the upload docs and re-run the dry-run commands shown above.

## Exit Criteria

You are done only when all of the following are true:

1. `go test ./pkg/mdpdf ./cmd/remarquee/cmds/upload` passes.
2. The new or modified preset is visible and understandable from `--help` or the docs.
3. Dry-run output clearly shows which layout is being used.
4. Manual PDF output looks intentional on-device or in a local PDF viewer.
5. `docmgr doctor` passes for the ticket you updated.

## Notes

1. Do not add layout-specific branches independently to `upload md` and `upload bundle`; that creates drift.
2. Do not silently change the default layout unless the ticket explicitly asks for a global visual change.
3. Be cautious with aggressive geometry changes. The goal is annotation room, not squeezing the prose into an unreadable column.
4. If you are tempted to touch `upload src`, stop and verify that the request is actually about prose review rather than code rendering.
