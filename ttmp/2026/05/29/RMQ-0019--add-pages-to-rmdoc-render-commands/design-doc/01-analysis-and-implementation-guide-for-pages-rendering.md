---
Title: Analysis and Implementation Guide for --pages Rendering
Ticket: RMQ-0019
Status: active
Topics:
    - cli
    - rmdoc
    - rendering
    - remarkable
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/remarquee/cmds/rmdoc/render_legacy.go
      Note: Legacy render command to receive --pages
    - Path: cmd/remarquee/cmds/rmdoc/render_v6.go
      Note: V6 render command to receive --pages
    - Path: cmd/remarquee/cmds/rmdoc/vlm_validate.go
      Note: Existing --pages parsing and V6 subset usage
    - Path: pkg/rmdoc/render/golden_legacy_rmapi_test.go
      Note: Legacy fixture and PDF page-count testing pattern
    - Path: pkg/rmdoc/render/v6_merge_background.go
      Note: Existing V6 subset render primitive
ExternalSources: []
Summary: Design and implementation guide for adding a --pages subset selector to rmdoc render-v6 and render-legacy.
LastUpdated: 2026-05-29T13:15:37.452413486-04:00
WhatFor: Use when implementing, reviewing, or extending page subset rendering for remarquee rmdoc commands.
WhenToUse: Before touching render-v6/render-legacy page selection semantics, tests, or docs.
---


# Analysis and Implementation Guide for --pages Rendering

## Executive Summary

`remarquee rmdoc render-v6` and `remarquee rmdoc render-legacy` currently render complete documents. This is inconvenient for review workflows where the user only wants a few pages from a large notebook or PDF-backed `.rmdoc`, especially after using `remarquee cloud get` or `--cloud` to pull a remote document locally.

Add a shared `--pages` parameter to both verbs. The flag should accept 1-based page selectors such as `1`, `1,3,5`, and ranges such as `2-4`. Internally, commands should normalize selectors to zero-based UI page indexes, validate them against `doc.Pages`, and render only those source document pages. V6 already has a page-subset renderer in `pkg/rmdoc/render`; the CLI needs to expose it. Legacy rendering should be implemented by generating a complete temporary PDF with the existing rmapi-backed generator and then extracting the requested PDF pages into the requested output.

## Problem Statement

The rendering verbs are useful but coarse-grained. Current behavior forces users to render an entire document even when they only need a small page subset for validation, sharing, OCR, or debugging.

Evidence from the current code:

- `RenderV6Settings` only contains `file`, `out`, `force`, and cloud input settings; no page selector is exposed at the command layer (`cmd/remarquee/cmds/rmdoc/render_v6.go:28-35`).
- `render-v6` always calls `MergeRMDocV6OntoBackgroundPDFWithInfo`, which renders the full document (`cmd/remarquee/cmds/rmdoc/render_v6.go:168`).
- `RenderLegacySettings` has `all-pages`, `annotations-only`, and `add-page-numbers`, but no source page subset selector (`cmd/remarquee/cmds/rmdoc/render_legacy.go:27-36`).
- `render-legacy` delegates directly to `rmapi_annotations.CreatePdfGenerator(...).Generate()` and writes the final output at once (`cmd/remarquee/cmds/rmdoc/render_legacy.go:161-169`).
- A V6 subset primitive already exists: `MergeRMDocV6OntoBackgroundPDFWithInfoForPages(ctx, rmdocPath, opts, pageIndices)` accepts zero-based UI indexes and validates them (`pkg/rmdoc/render/v6_merge_background.go:504-523`).
- `vlm-validate` already has a `--pages` flag and converts 1-based CLI pages to zero-based indexes before calling the V6 subset primitive (`cmd/remarquee/cmds/rmdoc/vlm_validate.go:115-120` and `cmd/remarquee/cmds/rmdoc/vlm_validate.go:291-297`).

## Scope

### In scope

- Add `--pages` to `rmdoc render-v6`.
- Add `--pages` to `rmdoc render-legacy`.
- Support comma-separated 1-based pages and inclusive page ranges.
- Keep full-document rendering as the default when `--pages` is omitted.
- Include selected page metadata in Glaze output.
- Add tests for parsing, V6 subset output page count, and legacy subset output page count.
- Update README examples.

### Out of scope

- Changing `render-v6-png`; it already has page-oriented behavior through the VLM validation path and is not part of this request.
- Rewriting rmapi's legacy `PdfGenerator`.
- Supporting EPUB rendering; both commands already reject EPUBs.
- Supporting labels/bookmarks/outlines that remap after subset extraction. Legacy extraction can intentionally drop or not preserve outlines.

## Proposed Solution

### CLI contract

Add this flag to both commands:

```text
--pages string   1-based page numbers/ranges to render, for example "1", "1,3,5", or "2-4" (default: all pages)
```

Semantics:

1. Empty/omitted `--pages` means all source UI pages, preserving today's default behavior.
2. Page numbers are 1-based at the CLI boundary.
3. Ranges are inclusive and ascending (`2-4` means pages 2, 3, and 4).
4. Duplicates are preserved unless there is a concrete reason not to. Preserving order and duplicates makes the helper simple and enables future workflows such as `--pages 3,3,1`; however, tests only need to cover ordered unique selections.
5. Bounds are validated after opening the document so errors can say `page 9 out of range (pages=4)`.
6. For Glaze rows, expose `pages` as the total source document page count and `selected_pages` as the normalized 1-based selector string/list.

### Shared parser/helper

Create a small package-local helper in `cmd/remarquee/cmds/rmdoc`, for example `pages.go`:

```go
type PageSelection struct {
    All       bool
    Pages1    []int
    Indices0  []int
}

func parsePageSelection1Based(spec string, totalPages int) (PageSelection, error)
func formatPages1Based(pages []int) string
```

The helper should be command-layer code, not renderer code, because `--pages` is CLI syntax. Renderers should receive normalized integer indexes.

### V6 implementation

`render-v6` should:

1. Add `Pages string` to `RenderV6Settings`.
2. Add the `pages` field definition.
3. Open the `.rmdoc` as it does today.
4. Parse `s.Pages` against `len(doc.Pages)`.
5. If all pages are selected, keep calling `MergeRMDocV6OntoBackgroundPDFWithInfo`.
6. If a subset is selected, call `MergeRMDocV6OntoBackgroundPDFWithInfoForPages(ctx, input.LocalPath, rmdocrender.V6MergeOptions{}, selection.Indices0)`.

This is low risk because the renderer already has a tested page-indexed API and uses UI-order page IDs internally (`pkg/rmdoc/render/v6_merge_background.go:558-578`).

### Legacy implementation

`render-legacy` has no page-indexed rmapi API. The lowest-risk approach is:

1. Generate a complete temporary legacy PDF using the existing rmapi generator.
2. Extract requested pages from that generated PDF using UniPDF.
3. Move/write the extracted PDF to the requested output path.

Implementation sketch:

```go
func extractPDFPages(inputPDF, outputPDF string, pages1 []int) error {
    b, err := os.ReadFile(inputPDF)
    if err != nil { return err }
    r, err := pdf.NewPdfReader(bytes.NewReader(b))
    if err != nil { return err }
    n, err := r.GetNumPages()
    if err != nil { return err }

    w := pdf.NewPdfWriter()
    for _, pageNum := range pages1 {
        if pageNum < 1 || pageNum > n { return errors.Errorf(...) }
        p, err := r.GetPage(pageNum)
        if err != nil { return err }
        if err := w.AddPage(p); err != nil { return err }
    }
    f, err := os.Create(outputPDF)
    if err != nil { return err }
    defer f.Close()
    return w.Write(f)
}
```

To make source-page selection predictable, `render-legacy --pages` should render the intermediate PDF with `AllPages: true` even if the user did not pass `--all-pages`. Otherwise rmapi omits unannotated pages and the output page numbers no longer align with source document pages. This is a deliberate compatibility choice for the new flag: `--pages 10` should mean source page 10, not the tenth annotated page.

`--annotations-only` should still be passed through. It generates blank/background-free pages and then extracts the requested subset. `--add-page-numbers` may number the intermediate full document before extraction; document this as source page numbering rather than renumbering the subset.

## Design Decisions

### Use source UI pages, not generated output pages

The user is asking to render a subset of document pages. The only stable interpretation across V6, legacy, cloud, annotated/unannotated pages, and notebooks is source UI page order from `doc.Pages`. For legacy, this requires forcing the temporary render to include all pages before extraction.

### Keep parser in the command package

The parser translates human CLI syntax into integer indexes. Renderer APIs should remain numeric and reusable. This mirrors the existing VLM code path, where CLI `--pages` is parsed before calling V6 render-for-pages.

### Reuse V6 subset renderer

The V6 implementation should not post-process a full PDF because `MergeRMDocV6OntoBackgroundPDFWithInfoForPages` already builds a subset background PDF and overlays only requested annotations. This saves memory and preserves existing UI-order behavior.

### Post-process legacy output

The rmapi generator is external code and does not expose a page selection option. Post-processing avoids vendoring/forking rmapi and keeps this feature isolated to the remarquee command layer.

## Alternatives Considered

### Add page selection to rmapi

This would be cleaner for legacy but would require changing an upstream module API, carrying a fork, or waiting on dependency updates. It is too much surface area for a CLI feature.

### Build a subset `.rmdoc` archive for legacy

This could preserve source-page semantics without PDF extraction, but legacy `.rmdoc` archives include payload PDFs, content metadata, and `.rm` annotations with page references. Safely rewriting all of those is higher risk than extracting pages from a generated PDF.

### Always render full PDF then extract for V6

This would produce correct page counts but waste work and bypass the existing V6 page-indexed renderer. It is only acceptable for legacy because there is no equivalent lower-level API.

## Implementation Plan

1. Add shared page-selection helper and unit tests.
   - File: `cmd/remarquee/cmds/rmdoc/pages.go`
   - File: `cmd/remarquee/cmds/rmdoc/pages_test.go`
   - Cover empty selection, commas, ranges, invalid tokens, descending ranges, zero/negative pages, and out-of-range pages.
2. Wire `render-v6`.
   - Add `Pages string` setting and field definition.
   - Parse after `doc` is opened.
   - Dispatch to full renderer or `MergeRMDocV6OntoBackgroundPDFWithInfoForPages`.
   - Add selected pages to Glaze output.
3. Wire `render-legacy`.
   - Add `Pages string` setting and field definition.
   - Generate to a temp output when a subset is requested.
   - Force `AllPages: true` for the temporary render when `--pages` is set.
   - Extract selected pages into the real output.
   - Add selected pages to Glaze output.
4. Add command tests.
   - V6: use an existing fixture such as `cmd/remarquee-ui/testdata/cpage-pdf.rmdoc`; render `--pages 1` or `--pages 1-2` and assert the output PDF page count.
   - Legacy: use `cmd/remarquee-ui/testdata/legacy-pdf-a4.zip`; render `--pages 1` or `--pages 1-2` and assert the output PDF page count.
5. Update README examples under "Render local `.rmdoc` archives" to include page subset usage.
6. Run focused tests, then a broader package test pass for the touched command/render areas.

## Testing and Validation Strategy

Minimum validation commands:

```bash
go test ./cmd/remarquee/cmds/rmdoc ./pkg/rmdoc/render -count=1

go run ./cmd/remarquee rmdoc render-v6 cmd/remarquee-ui/testdata/cpage-pdf.rmdoc \
  --pages 1 --out /tmp/remarquee-v6-page1.pdf --force

go run ./cmd/remarquee rmdoc render-legacy cmd/remarquee-ui/testdata/legacy-pdf-a4.zip \
  --pages 1 --out /tmp/remarquee-legacy-page1.pdf --force --all-pages
```

Expected results:

- The V6 command writes a one-page PDF.
- The legacy command writes a one-page PDF.
- Invalid selectors fail before writing output, with clear messages.
- Existing no-`--pages` behavior still renders a complete document.

## Risks and Review Notes

- Legacy `--add-page-numbers` numbers the full intermediate PDF before extraction. This should be reviewed and documented; it may be acceptable because the flag now means source page numbers.
- Legacy `--pages` temporarily forces `AllPages: true` to preserve source-page semantics. This is intentional but should be visible in tests or comments.
- PDF page extraction may drop outlines and document-level metadata. That is acceptable for subset output but should not surprise users who expect a structurally identical PDF.
- The parser should reject ambiguous or malformed input early. Ranges should be inclusive and simple; avoid hidden syntax like `last` until explicitly requested.

## References

- `cmd/remarquee/cmds/rmdoc/render_v6.go` — V6 CLI command and full-document render call.
- `cmd/remarquee/cmds/rmdoc/render_legacy.go` — legacy CLI command and rmapi generator delegation.
- `pkg/rmdoc/render/v6_merge_background.go` — existing V6 page-subset renderer.
- `cmd/remarquee/cmds/rmdoc/vlm_validate.go` — existing command with `--pages` and V6 subset rendering usage.
- `pkg/rmdoc/render/golden_legacy_rmapi_test.go` — legacy fixture and PDF page-count pattern.
