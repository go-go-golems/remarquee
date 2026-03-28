---
Title: Analysis and design for editor-friendly markdown upload layout
Ticket: RMQ-0014
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
    - Path: cmd/remarquee/cmds/upload/bundle.go
      Note: Bundled markdown upload wiring for --layout
    - Path: cmd/remarquee/cmds/upload/layout.go
      Note: Shared CLI option precedence for markdown layouts
    - Path: cmd/remarquee/cmds/upload/md.go
      Note: Markdown upload command wiring for --layout
    - Path: pkg/mdpdf/layout.go
      Note: Named markdown layout presets live here
    - Path: pkg/mdpdf/pandoc.go
      Note: Pandoc option layering and extra header support
ExternalSources: []
Summary: Evidence-backed analysis of how remarquee's markdown upload path works, why a named annotation-friendly layout preset is the right extension point, and how to extend it safely.
LastUpdated: 2026-03-28T09:28:20.108740378-04:00
WhatFor: Explain the architecture, tradeoffs, and implementation plan behind the editor-friendly markdown upload layout feature.
WhenToUse: Use this document before changing markdown upload presentation, pandoc option layering, or future layout presets.
---


# Analysis and design for editor-friendly markdown upload layout

## Executive Summary

`remarquee` already had a solid markdown upload path, but it exposed only low-level typography controls such as `--geometry` and `--latex-header-file`. That was workable for power users and poor for a repeatable editorial workflow. The feature request was not “let me hand-author TeX,” it was “give me a mode where uploaded markdown leaves room for editing and margin comments.”

The implemented change adds a named `--layout editor` preset for `remarquee upload md` and `remarquee upload bundle`. The preset is defined once in [`pkg/mdpdf/layout.go`](../../../../../../pkg/mdpdf/layout.go) and applied through a shared helper in [`cmd/remarquee/cmds/upload/layout.go`](../../../../../../cmd/remarquee/cmds/upload/layout.go). The preset widens margins and loosens paragraph spacing, while explicit `--geometry` and `--latex-header-file` flags still override the defaults if a caller needs to go lower-level.

This is the right shape for the feature because it:

1. Preserves backward compatibility for existing upload workflows.
2. Keeps layout policy out of the command handlers themselves.
3. Reuses the existing pandoc/xelatex pipeline rather than forking a separate rendering path.
4. Gives future contributors an obvious place to add more named layouts without duplicating CLI wiring.

## Problem Statement

The user requirement was editor-oriented rather than purely typographic: uploaded markdown should provide visual room to rephrase text and write comments in the margins on reMarkable. Before this change, markdown uploads exposed `--geometry` and `--latex-header-file` as raw escape hatches, but there was no high-level flag expressing “annotation-friendly review layout.”

Observed current-state constraints:

1. `remarquee upload md` already exposes raw pandoc/LaTeX knobs, but no named presentation preset. The flag surface before the change was centered on `--pandoc`, `--pdf-engine`, `--mainfont`, `--monofont`, `--geometry`, and `--latex-header-file` in [`cmd/remarquee/cmds/upload/md.go:83-90`](../../../../../../cmd/remarquee/cmds/upload/md.go).
2. `remarquee upload bundle` had the same problem for the single-PDF markdown workflow, with no higher-level layout abstraction in [`cmd/remarquee/cmds/upload/bundle.go:100-107`](../../../../../../cmd/remarquee/cmds/upload/bundle.go).
3. The actual markdown-to-PDF execution is centralized in [`pkg/mdpdf/pandoc.go:47-132`](../../../../../../pkg/mdpdf/pandoc.go), which preprocesses markdown, writes temporary header files, and shells out to pandoc.
4. `upload src` is a separate workflow with syntax-highlighting concerns and should not automatically inherit an editorial markdown layout. It still configures pandoc directly in [`cmd/remarquee/cmds/upload/src.go:117-166`](../../../../../../cmd/remarquee/cmds/upload/src.go).

The gap was therefore not “we need more rendering machinery.” The gap was “we need a first-class preset layered over the existing machinery.”

## Current-State Analysis

### Architecture of the markdown upload path

The markdown upload path is already fairly modular.

1. `upload md` collects markdown inputs, resolves remote destinations, and constructs `PandocOptions` before generating PDFs or uploading them in [`cmd/remarquee/cmds/upload/md.go:95-220`](../../../../../../cmd/remarquee/cmds/upload/md.go).
2. `upload bundle` collects markdown files, builds a temporary wrapper markdown document, and uses the same PDF conversion function in [`cmd/remarquee/cmds/upload/bundle.go:112-208`](../../../../../../cmd/remarquee/cmds/upload/bundle.go).
3. The markdown renderer itself lives in [`pkg/mdpdf/pandoc.go:14-132`](../../../../../../pkg/mdpdf/pandoc.go). It:
   - normalizes defaults,
   - strips docmgr-style frontmatter,
   - normalizes list spacing,
   - writes temporary markdown and header files,
   - shells out to pandoc.

This matters because the best extension point is above the pandoc execution core but below the two markdown-oriented commands. A preset should be defined once, then shared.

### Where the final implementation inserts itself

The implemented design introduces three clean seams:

1. A named layout catalog in [`pkg/mdpdf/layout.go:9-41`](../../../../../../pkg/mdpdf/layout.go).
2. Shared upload-side option construction in [`cmd/remarquee/cmds/upload/layout.go:9-38`](../../../../../../cmd/remarquee/cmds/upload/layout.go).
3. Support for layering an extra temporary LaTeX header after the default or custom header in [`pkg/mdpdf/pandoc.go:82-110`](../../../../../../pkg/mdpdf/pandoc.go).

This is narrower and safer than scattering `if layout == "editor"` branches across both commands.

## Gap Analysis

Before the change:

1. Users had to know raw LaTeX geometry syntax to get wider margins.
2. There was no stable name for an editorial layout, so docs and habits would drift.
3. `upload md` and `upload bundle` could have diverged if layout logic were copied into each command.
4. There was no validation layer to reject invalid preset names early.

After the change:

1. `--layout editor` gives a discoverable, documented preset in [`cmd/remarquee/cmds/upload/md.go:84-90`](../../../../../../cmd/remarquee/cmds/upload/md.go) and [`cmd/remarquee/cmds/upload/bundle.go:100-107`](../../../../../../cmd/remarquee/cmds/upload/bundle.go).
2. Unknown layouts fail immediately via [`pkg/mdpdf/layout.go:31-40`](../../../../../../pkg/mdpdf/layout.go).
3. The dry-run output echoes the resolved layout so users can verify intent before rendering in [`cmd/remarquee/cmds/upload/md.go:140-165`](../../../../../../cmd/remarquee/cmds/upload/md.go) and [`cmd/remarquee/cmds/upload/bundle.go:148-165`](../../../../../../cmd/remarquee/cmds/upload/bundle.go).
4. The embedded docs now mention the preset in [`pkg/doc/upload/02-remarquee-upload-reference.md:38-45`](../../../../../../pkg/doc/upload/02-remarquee-upload-reference.md) and [`pkg/doc/upload/03-remarquee-upload-bundle.md:39-45`](../../../../../../pkg/doc/upload/03-remarquee-upload-bundle.md).

## Proposed Solution

The implemented solution is:

1. Add `MarkdownLayoutDefault` and `MarkdownLayoutEditor` constants.
2. Add `ApplyMarkdownLayoutPreset(opts, layout)` to mutate `PandocOptions` based on a named layout.
3. Encode the editor preset as:
   - `top=1in,bottom=1.15in,left=1.1in,right=1.9in`
   - `\setstretch{1.18}`
   - `\parskip` increase
   - `\parindent` removal
4. Extend `PandocOptions` with `ExtraLatexHeader` so presets can layer additional layout behavior without replacing the default header.
5. Add a shared `configureMarkdownPandocOptions(...)` helper in the upload package that:
   - starts from `DefaultPandocOptions()`,
   - applies the named preset,
   - then respects explicit `--geometry` and `--latex-header-file` overrides if the caller changed those flags.
6. Wire the helper into both markdown upload commands.

### API Sketch

```go
const (
    MarkdownLayoutDefault = "default"
    MarkdownLayoutEditor  = "editor"
)

func ApplyMarkdownLayoutPreset(opts *PandocOptions, layout string) error

func configureMarkdownPandocOptions(
    flags *pflag.FlagSet,
    layout string,
    pandoc string,
    pdfEngine string,
    mainFont string,
    monoFont string,
    geometry string,
    latexHeaderFile string,
) (mdpdf.PandocOptions, error)
```

### Pseudocode

```text
runUploadMarkdown / runUploadBundle:
  collect inputs
  resolve remote dir
  opts = DefaultPandocOptions()
  apply named layout preset
  if user explicitly set --geometry:
    override preset geometry
  if user explicitly set --latex-header-file:
    use that header file
  if bundle:
    enable TOC
  if dry-run:
    print layout + planned outputs
  else:
    render pdf via ConvertMarkdownFileToPDF
    optionally upload
```

## Design Decisions

### Decision 1: use a named preset instead of another boolean

The feature is exposed as `--layout default|editor`, not `--editor-mode`.

Reasoning:

1. The codebase can grow more than one layout later without adding another boolean explosion.
2. A named preset is easier to document and validate.
3. The meaning of `default` is explicit in tests and dry-run output.

### Decision 2: keep the implementation in `pkg/mdpdf`

The actual layout definitions live in [`pkg/mdpdf/layout.go`](../../../../../../pkg/mdpdf/layout.go), not the Cobra command files.

Reasoning:

1. Layouts are renderer concerns, not argument-parsing concerns.
2. Both markdown upload commands need the same behavior.
3. Future non-CLI callers can reuse the same presets.

### Decision 3: add an extra header layer instead of replacing the main header

`ConvertMarkdownFileToPDF` now supports `ExtraLatexHeader` in addition to `LatexHeaderFile`.

Reasoning:

1. The existing default header still handles list formatting.
2. Presets can layer new behavior without losing the built-in defaults.
3. A caller can still provide `--latex-header-file`, and the preset can be appended after it if desired.

### Decision 4: explicit flag changes outrank presets

The upload helper checks `flags.Changed("geometry")` and `flags.Changed("latex-header-file")` in [`cmd/remarquee/cmds/upload/layout.go:28-36`](../../../../../../cmd/remarquee/cmds/upload/layout.go).

Reasoning:

1. Presets should provide good defaults, not trap users.
2. This avoids surprising behavior where `--layout editor --geometry ...` silently ignores the more specific request.
3. It preserves the existing power-user escape hatch.

## Alternatives Considered

### Alternative 1: tell users to pass `--geometry` manually

Rejected because it does not solve discoverability, repeatability, or docs quality. It also pushes low-level LaTeX knowledge onto the editorial workflow.

### Alternative 2: add a custom header file checked into the repo and tell users to pass `--latex-header-file`

Rejected because it would still be an implicit workflow rather than a first-class CLI feature. It also leaves validation and dry-run visibility out of the product surface.

### Alternative 3: fork a separate markdown-to-PDF rendering path for editorial review

Rejected because the existing pandoc pipeline is already the right renderer. The problem was configuration policy, not missing rendering capability.

### Alternative 4: apply the preset to `upload src` too

Rejected for now because source upload is optimized for code readability and syntax highlighting, not margin-heavy prose review. The separation is visible in [`cmd/remarquee/cmds/upload/src.go:111-166`](../../../../../../cmd/remarquee/cmds/upload/src.go).

## Implementation Plan

### Phase 1: define the preset catalog

1. Add layout constants and normalization.
2. Add editor geometry and extra header content.
3. Validate unknown preset names with a clear error.

Implemented in [`pkg/mdpdf/layout.go:9-41`](../../../../../../pkg/mdpdf/layout.go).

### Phase 2: make pandoc option layering support presets

1. Extend `PandocOptions` with `ExtraLatexHeader`.
2. Write the extra header to a temp file only when present.
3. Append all header files to pandoc in order.

Implemented in [`pkg/mdpdf/pandoc.go:14-28`](../../../../../../pkg/mdpdf/pandoc.go) and [`pkg/mdpdf/pandoc.go:82-110`](../../../../../../pkg/mdpdf/pandoc.go).

### Phase 3: share markdown command wiring

1. Add a helper that applies presets and then re-applies explicit overrides.
2. Use it from both `upload md` and `upload bundle`.
3. Echo the resolved layout in dry-run mode.

Implemented in [`cmd/remarquee/cmds/upload/layout.go:9-38`](../../../../../../cmd/remarquee/cmds/upload/layout.go), [`cmd/remarquee/cmds/upload/md.go:126-165`](../../../../../../cmd/remarquee/cmds/upload/md.go), and [`cmd/remarquee/cmds/upload/bundle.go:132-165`](../../../../../../cmd/remarquee/cmds/upload/bundle.go).

### Phase 4: validate and document

1. Add unit tests for preset normalization and validation.
2. Add dry-run-oriented command tests so the new surface is visible and asserted.
3. Update the embedded upload reference docs.

Implemented in [`pkg/mdpdf/layout_test.go`](../../../../../../pkg/mdpdf/layout_test.go), [`cmd/remarquee/cmds/upload/md_test.go:97-136`](../../../../../../cmd/remarquee/cmds/upload/md_test.go), [`cmd/remarquee/cmds/upload/bundle_test.go:51-70`](../../../../../../cmd/remarquee/cmds/upload/bundle_test.go), [`pkg/doc/upload/02-remarquee-upload-reference.md:38-45`](../../../../../../pkg/doc/upload/02-remarquee-upload-reference.md), and [`pkg/doc/upload/03-remarquee-upload-bundle.md:39-45`](../../../../../../pkg/doc/upload/03-remarquee-upload-bundle.md).

## Testing and Validation Strategy

Primary command:

```bash
go test ./pkg/mdpdf ./cmd/remarquee/cmds/upload
```

What this covers:

1. Preset normalization and unknown-layout rejection in `pkg/mdpdf`.
2. Markdown upload dry-run output includes `DRY: layout=editor`.
3. Bundle dry-run output includes `DRY: layout=editor`.

Manual smoke tests for a contributor:

```bash
go run ./cmd/remarquee upload md --dry-run --layout editor --pdf-only /abs/path/to/doc.md
go run ./cmd/remarquee upload bundle --dry-run --layout editor --pdf-only /abs/path/to/dir
go run ./cmd/remarquee upload md --pdf-only --layout editor --output-dir /tmp/rmq-layout /abs/path/to/doc.md
```

Review points:

1. Confirm the default layout remains unchanged when `--layout` is omitted.
2. Confirm `--layout editor --geometry <custom>` honors the explicit geometry.
3. Confirm a custom `--latex-header-file` still works.

## Risks, Alternatives, and Open Questions

### Risks

1. The chosen editor geometry may still feel too narrow or too wide on-device for some documents.
2. A user-provided header file might deliberately fight the preset; explicit override precedence is intentional, but visual results may vary.
3. `upload src` now differs more clearly from markdown upload, which is correct but should remain documented.

### Open Questions

1. Should a future preset support symmetric wide margins instead of a larger right margin?
2. Should we add a `review` or `proof` layout later with different fonts or section spacing?
3. Do we want a screenshot-based golden test for PDF layout eventually, or are option-level tests sufficient?

## References

### Key Files

1. [`pkg/mdpdf/layout.go`](../../../../../../pkg/mdpdf/layout.go)
2. [`pkg/mdpdf/pandoc.go`](../../../../../../pkg/mdpdf/pandoc.go)
3. [`cmd/remarquee/cmds/upload/layout.go`](../../../../../../cmd/remarquee/cmds/upload/layout.go)
4. [`cmd/remarquee/cmds/upload/md.go`](../../../../../../cmd/remarquee/cmds/upload/md.go)
5. [`cmd/remarquee/cmds/upload/bundle.go`](../../../../../../cmd/remarquee/cmds/upload/bundle.go)
6. [`cmd/remarquee/cmds/upload/src.go`](../../../../../../cmd/remarquee/cmds/upload/src.go)
7. [`cmd/remarquee/cmds/upload/md_test.go`](../../../../../../cmd/remarquee/cmds/upload/md_test.go)
8. [`cmd/remarquee/cmds/upload/bundle_test.go`](../../../../../../cmd/remarquee/cmds/upload/bundle_test.go)
9. [`pkg/doc/upload/02-remarquee-upload-reference.md`](../../../../../../pkg/doc/upload/02-remarquee-upload-reference.md)
10. [`pkg/doc/upload/03-remarquee-upload-bundle.md`](../../../../../../pkg/doc/upload/03-remarquee-upload-bundle.md)
