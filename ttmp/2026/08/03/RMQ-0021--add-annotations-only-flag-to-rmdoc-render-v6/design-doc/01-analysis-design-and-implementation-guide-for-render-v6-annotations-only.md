---
Title: Analysis, Design and Implementation Guide for render-v6 --annotations-only
Ticket: RMQ-0021
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
    - Path: repo://cmd/remarquee-ui/testdata/generated/fake-cpages-pdf-v6-sample-rm.rmdoc
      Note: Fixture with 2 UI pages but only 1 annotated - core test case for skip/blank semantics
    - Path: repo://cmd/remarquee/cmds/rmdoc/input_resolver.go
      Note: CloudInputSettings/ResolveRMDocInput for --cloud input
    - Path: repo://cmd/remarquee/cmds/rmdoc/pages.go
      Note: parsePageSelection1Based drives includeUnannotated derivation
    - Path: repo://cmd/remarquee/cmds/rmdoc/render_legacy.go
      Note: Legacy verb whose --annotations-only semantics (rmapi PdfGeneratorOptions) we mirror
    - Path: repo://cmd/remarquee/cmds/rmdoc/render_v6.go
      Note: V6 verb receiving the --annotations-only flag (settings, flag def, execute branch)
    - Path: repo://pkg/rmdoc/render/background.go
      Note: Background PDF assembly the annotations-only renderer deliberately skips
    - Path: repo://pkg/rmdoc/render/v6_merge_background.go
      Note: Merge pipeline; overlay-only path (buildOverlayOnlyPageBBoxScaled) reused by the new renderer
ExternalSources: []
Summary: Intern-ready analysis, design, and implementation guide for adding an --annotations-only flag to remarquee's rmdoc render-v6 verb, with parity to render-legacy --annotations-only.
LastUpdated: 2026-08-03T18:06:17.429829407-04:00
WhatFor: Use when implementing, reviewing, or extending annotations-only rendering for remarquee's V6 (cPages) render pipeline.
WhenToUse: Before touching render-v6 flags, the pkg/rmdoc/render merge pipeline, or page-selection semantics.
---


# Analysis, Design and Implementation Guide for render-v6 --annotations-only

## Executive Summary

`remarquee rmdoc render-v6` renders a reMarkable V6 (cPages) document into an annotated PDF by merging handwritten strokes, smart highlights, and typed text on top of the original background PDF pages. It always produces the **full composite**: background plus annotations. There is currently no way to produce a PDF that contains **only the annotations on blank pages**, even though the sibling verb `render-legacy` has supported exactly that for a long time via its `--annotations-only` flag (`cmd/remarquee/cmds/rmdoc/render_legacy.go:36`).

This matters more than feature parity for its own sake: on modern reMarkable devices, even documents whose container still uses the legacy schema carry **V6-format annotation files**, and the legacy renderer (which delegates to rmapi's `PdfGenerator`) **hard-fails on them**. We verified this with a real document pulled from the tablet: `render-legacy --annotations-only` exits with `Error: Unknown header`, while `render-v6` renders the same pages correctly. In other words, for a large and growing class of real-world documents, annotations-only export is not merely missing — it is **impossible today through any verb**.

This guide explains every part of the system you need to understand to add `--annotations-only` to `render-v6`: the `.rmdoc` container format, the Glazed CLI framework the verbs are built with, the V6 parse pipeline (scene trees, strokes, glyph ranges, typed text), the merge/overlay render pipeline, and the legacy rmapi pipeline whose behavior we want to mirror. It then proposes a concrete design — a dedicated annotations-only renderer in `pkg/rmdoc/render` plus one new CLI flag — with pseudocode, decision records, a phased file-level implementation plan, and a test strategy built on fixtures that already exist in the repository.

## Problem Statement and Scope

### The user-visible problem

A user has a document on their reMarkable tablet — a PDF worksheet they annotated by hand. They want a PDF containing **only their handwriting** (for example to review their notes without the worksheet text, to overlay onto a revised worksheet later, or to shrink the file). Today:

- `remarquee rmdoc render-legacy --annotations-only doc.rmdoc` produces annotations-only output **only** for documents with legacy (V3/V5) `.rm` annotation files. For documents with V6 `.rm` files it fails with `Error: Unknown header` (rmapi's parser does not understand the V6 header; observed verbatim on 2026-08-03 with the workbook example below).
- `remarquee rmdoc render-v6 doc.rmdoc` handles V6 annotations correctly but **always** composites them onto the background PDF pages. There is no flag to suppress the background.

### Verified reproduction (real document, 2026-08-03)

The motivating example used throughout this guide is the document
`/ai/2026/08/03/TTC-GARDEN-UXQA-001/TTC Garden Human Calibration Workbook`
on the reMarkable cloud, downloaded with `remarquee cloud get`:

```text
$ remarquee rmdoc inspect "TTC Garden Human Calibration Workbook.rmdoc"
uuid=da144dd8-fc09-4672-a565-c2664ae4fbd2 schema=legacy type=pdf pages=44
idx  page_id                                  src_pdf  template
0    68920305-a4ae-4527-85d6-72ebdc1ce5ee    0        Blank
1    e07be4b9-3bee-4acd-bf00-5e66d52c1bc8    1
2    f956bd27-0091-4b0d-9eb1-5e27e0ef52aa    2
...  (41 more pages with no .rm files)

$ remarquee rmdoc render-v6  workbook.rmdoc --pages 1-3 --out v6.pdf --force
ok: wrote v6.pdf                          # 3 pages, 612x792pt (letter), works

$ remarquee rmdoc render-legacy workbook.rmdoc --pages 1-3 --annotations-only --out ann.pdf --force
Error: Unknown header                     # rmapi cannot parse V6 .rm files
```

Two facts in that output drive the whole design:

1. **Schema is not the same as annotation version.** The archive's `.content` file is in the *legacy* schema (plain page list, no `cPages` block), yet its three `.rm` files all begin with the header `reMarkable .lines file, version=6`. `render-v6` already anticipates exactly this situation: when the schema is not cPages it probes each page's `.rm` file and proceeds if any of them is V6 (`archiveHasV6RM`, `cmd/remarquee/cmds/rmdoc/render_v6.go:128`).
2. **Only pages 1–3 are annotated.** The 44-page archive contains exactly three `.rm` files, one for each of the first three pages. This makes the example perfect for deciding what annotations-only output should do with unannotated pages.

### In scope

- Add an `--annotations-only` boolean flag to `remarquee rmdoc render-v6`.
- A new exported renderer in `pkg/rmdoc/render` that produces a PDF of annotation content (strokes, smart highlights, typed text) on blank pages, without reading or compositing the background payload PDF.
- Correct interplay with the existing `--pages` subset selector (RMQ-0019).
- Unit and CLI smoke tests, plus manual validation against the workbook example.
- Help text and README updates.

### Out of scope

- `render-v6-png` and `vlm-validate` (they share the same library primitive and are natural follow-ups, but no CLI change is requested for them).
- The `remarquee-ui` HTTP API (`cmd/remarquee-ui/api/render.go`), which currently hard-codes `AnnotationsOnly: false` for its rmapi legacy path.
- EPUB-backed documents (already rejected by both verbs).
- Template backgrounds for notebooks (a known later milestone, see `BuildBackgroundPDF` docs at `pkg/rmdoc/render/background.go:29-36`).
- Changing `render-legacy` in any way.

## Background: The System You Are Working In

This section orients a reader who has never seen the codebase. Each subsection names the concrete files you should open while reading.

### What remarquee is

`remarquee` is a Go CLI (module `github.com/go-go-golems/remarquee`) that works with reMarkable tablet documents. Among other things it can:

- talk to the reMarkable cloud (`remarquee cloud ls|get|put|...`, backed by a fork of `rmapi`),
- upload Markdown as PDFs (`remarquee upload ...`),
- inspect and render `.rmdoc` archives (`remarquee rmdoc inspect|build-background|render-legacy|render-v6|render-v6-png|vlm-validate`).

The command tree is wired in `cmd/remarquee/cmds/rmdoc/root.go`; `render-v6` and `render-legacy` are siblings under `remarquee rmdoc`.

### What an `.rmdoc` archive is

An `.rmdoc` file is a ZIP archive. A PDF-backed document (like our example) contains:

```text
<uuid>.content     JSON: file type + the ordered UI page plan
<uuid>.metadata    JSON: bookkeeping (name, timestamps)
<uuid>.pagedata    text: template name per page (e.g. "Blank")
<uuid>.pdf         the payload PDF (the "background"; absent for notebooks)
<uuid>/<pageID>.rm one annotation file per annotated page (only present if you drew on the page)
```

The page plan in `.content` is what defines "page 1, page 2, ..." from the user's point of view. Two schema families exist:

- **legacy**: a flat page list; each page maps onto payload PDF pages in order.
- **cPages** ("V6 schema"): pages have explicit IDs, a `redir` pointer to a payload PDF page (or "inserted" for blank pages), and deletion markers.

The Go model lives in `pkg/rmdoc/types.go` (`Document`, `PageRef`, `SchemaLegacy`, `SchemaCPages`, `InsertedPage = -1` at line 50) and is produced by `pkg_rmdoc.OpenFile(ctx, path)` (`pkg/rmdoc/open.go:15`). `PageRef.SourcePDFPage` is the 0-based payload PDF page that backs a UI page, or `InsertedPage` for blank pages (`pkg/rmdoc/types.go:73-84`).

### Annotation formats: V3/V5 ("legacy") vs V6

Each `<pageID>.rm` file starts with the ASCII header `reMarkable .lines file, version=N`:

- **V3/V5** (legacy): a simple line/layer format that the vendored rmapi fork (`github.com/juruen/rmapi`, replaced by `github.com/marcobarcelos/rmapi` in `go.mod`) can parse. Its stroke decoder errors with `Unknown header` on anything else — which is exactly the failure we observed.
- **V6**: a CRDT-based scene-graph format (tagged blocks, group/scene items, glyph ranges, root text). remarquee has its own pure-Go parser for it in `pkg/rmdoc/` (`rmv6_*.go`, roughly 3,500 lines), built across tickets RMQ-0004, RMQ-0006, RMQ-0009, and RMQ-0012.

### The Glazed CLI framework (how verbs are defined)

All `rmdoc` verbs are built with the in-house `glazed` framework rather than raw cobra. The pattern, identical in `render_legacy.go` and `render_v6.go`, is:

1. A **settings struct** maps flag names to Go fields via struct tags, e.g. `AnnotationsOnly bool \`glazed:"annotations-only"\`` (`render_legacy.go:36`).
2. `NewRenderV6Command()` builds flag **definitions** with `fields.New("out", fields.TypeString, fields.WithDefault(""), fields.WithHelp(...))` and assembles a `glazecmds.CommandDescription` with a name, short/long help, flags, and standard sections (`render_v6.go:41-95`).
3. The command implements `Run(ctx, parsedValues)` for plain execution and `RunIntoGlazeProcessor(...)` for structured output (`--with-glaze-output` gives JSON/CSV/etc. for free). Both decode settings via `parsedValues.DecodeSectionInto(schema.DefaultSlug, s)` and delegate to a shared `execute(ctx, s)` method (`render_v6.go:97-213`).
4. `NewRenderV6CobraCommand()` wraps the glazed command into a cobra command (`cli.BuildCobraCommand`, `render_v6.go:232-250`), which `root.go` mounts under `rmdoc`.

When you add a flag to `render-v6` you touch exactly three places in `render_v6.go`: the settings struct, the `flags` slice in `NewRenderV6Command`, and the branching inside `execute`.

### The V6 parse pipeline (`pkg/rmdoc`)

From an `.rmdoc` path to renderable content, the pipeline is:

```text
.rmdoc (zip)
  └─ ReadRMFileFromArchive(ctx, path, pageID)     pkg/rmdoc/rm_archive_rmfiles.go:38
       └─ RMFile{Version: "V6", Bytes: [...]}
            └─ ParseRMV6SceneTree(bytes.Reader)   pkg/rmdoc/rmv6_scene_tree.go:127
                 └─ RMV6SceneTree
                      ├─ ExtractRMV6StrokesWithAnchors(tree)      pkg/rmdoc/rmv6_strokes_extract.go:7
                      │    └─ []Stroke{Tool, Color, ThicknessScale, Points[]}
                      ├─ ExtractRMV6GlyphRangesWithAnchors(tree)  pkg/rmdoc/rmv6_glyph_extract.go:7
                      │    └─ []RMV6GlyphRange{Color, Rectangles[]}   ("smart highlights")
                      └─ BuildRMV6TextDocument(tree.RootText)     pkg/rmdoc/rmv6_text_document.go:19
                           └─ []RMV6TextParagraph                  (typed text)
```

Three annotation content kinds fall out of this:

- **Strokes** — polylines in reMarkable *screen units* (the display is 1404×1872 units; constants `rmv6ScreenWidth/rmv6ScreenHeight/rmv6ScreenDPI` at `pkg/rmdoc/render/v6_strokes_pdf.go:16-20`). Tool IDs and color IDs map to pen styles via `strokeStyleForTool` (`v6_merge_background.go:97`) and `PenColorToRGBForStroke` (`pkg/rmdoc/pen_color.go:148`).
- **Glyph ranges** — rectangles over (typed or PDF) text the user highlighted; rendered as real PDF `Highlight` annotations rather than pixels ("smart highlights").
- **Root text** — typed notebook text, rendered with base-14 PDF fonts.

### The V6 render pipeline (`pkg/rmdoc/render`)

The public entry points used by the CLI are (`pkg/rmdoc/render/v6_merge_background.go`):

```go
type V6MergeOptions struct { StrokeWidthPt float64 }          // line 19
type V6MergeResult  struct { PDF []byte; HighlightsXTranslation []float64 }

func MergeRMDocV6OntoBackgroundPDF(ctx, path, opts) ([]byte, error)                       // line 264
func MergeRMDocV6OntoBackgroundPDFWithInfo(ctx, path, opts) (*V6MergeResult, error)       // line 272
func MergeRMDocV6OntoBackgroundPDFWithInfoForPages(ctx, path, opts, pageIndices []int)    // line 506
    (*V6MergeResult, error)
```

Both `WithInfo` variants implement the same per-page algorithm (the two loops are near-copies — this duplication matters later, see DR-1):

```text
for each UI page i (all pages, or the requested subset):
    1. fetch the pre-built background page i
    2. read the page's .rm file; skip to "add background as-is" if absent/empty
    3. parse scene tree -> strokes, glyphRanges, textParagraphs
    4. if all three are empty -> add background page as-is
    5. compute annotation canvas bbox:
       defaultBBox = full screen  UNION  stroke bbox (padded)
    6. branch on background content:
       a) background page HAS content  -> "merge path":
          remarks-style math computes a canvas that fits BOTH the background
          page box and the annotation bbox, positions each with x/y shifts,
          draws the background as a form XObject ("Bg"), then draws strokes
          on top (buildMergedPage, line 1010)
       b) background page is BLANK     -> "overlay-only path":
          page size is derived from the annotation bbox alone
          (buildOverlayOnlyPageBBoxScaled, line 755)
    7. applySmartHighlightsScaled(page, glyphRanges, ...)   (line 1075)
       adds PDF Highlight annotations
    8. writer.AddPage(page)
```

The background PDF itself is assembled first by `BuildBackgroundPDF` / `BuildBackgroundPDFForPages` (`pkg/rmdoc/render/background.go:39,51`): payload pages are copied (duplicated when referenced twice), inserted pages become blanks, notebooks get device-sized blank pages.

**The key observation for this ticket:** the *overlay-only path* (step 6b) already renders exactly what we want for one page — annotations on a blank canvas — but today it only triggers incidentally, when a background page happens to have no content stream (`v6_merge_background.go:399-400` and `634-635`). The feature is "force path 6b for every page, skip building the background entirely, and don't emit anything for unannotated pages."

### The legacy pipeline (rmapi) and the exact semantics to mirror

`render-legacy` delegates everything to rmapi:

```go
opts := rmapi_annotations.PdfGeneratorOptions{
    AddPageNumbers:  s.AddPageNumbers,
    AllPages:        s.AllPages || !selection.All,   // forced true when --pages is used
    AnnotationsOnly: s.AnnotationsOnly,
}
g := rmapi_annotations.CreatePdfGenerator(input.LocalPath, renderOut, opts)
```

(`cmd/remarquee/cmds/rmdoc/render_legacy.go:192-197`).

Inside the vendored fork (`github.com/marcobarcelos/rmapi@v0.0.0-20260518211546-a0d079936d46/annotations/pdf.go`), the semantics are:

- **Skip unannotated pages unless `AllPages`:** `if !p.options.AllPages && !hasContent { continue }` (line 96). So legacy annotations-only output *by default contains only the pages you actually drew on*.
- **`AnnotationsOnly` swaps the background for a blank page:** in `addBackgroundPage` (line 218), when `p.options.AnnotationsOnly` is true the generator calls `c.NewPage()` instead of copying the payload PDF page (line 226: `if !p.template && !p.options.AnnotationsOnly && pageNum > 0 {...}`). Strokes are then scaled onto that blank page.
- **`--pages` forces `AllPages`:** the CLI sets `AllPages: s.AllPages || !selection.All` because legacy subsetting works by rendering *everything* to a temp PDF and then extracting pages by index with `extractPDFPages` (`cmd/remarquee/cmds/rmdoc/pages.go:109`); indices only line up if every page exists in the intermediate file.

These three behaviors define "just like the render-legacy verb" for our purposes.

## Worked Example: TTC Garden Human Calibration Workbook, Pages 1–3

Ground-truth facts about the example document, gathered 2026-08-03 (commands in the reproduction block above):

| Fact | Value | How verified |
| --- | --- | --- |
| Archive schema / type | `legacy` / `pdf` | `rmdoc inspect` |
| UI pages | 44 | `rmdoc inspect` |
| Payload page size | 612×792 pt (US Letter) | `pdfinfo` on rendered output |
| `.rm` files present | 3, for UI pages 1–3 only | zip listing |
| `.rm` versions | all `version=6` | first 43 bytes of each file |
| `render-v6 --pages 1-3` | ✅ 3-page annotated PDF | ran it |
| `render-legacy --annotations-only` | ❌ `Error: Unknown header` | ran it |

What the current `render-v6` output contains for these pages: the letter-size worksheet pages (tables, printed questions) with the user's handwritten check marks, crosses, and short pen annotations merged on top — i.e. the composite. An annotations-only render of pages 1–3 should instead show a blank (white) page carrying only the strokes — and, because all three selected pages are annotated, should still produce exactly 3 pages.

Because the document is `schema=legacy` with V6 `.rm` files, it exercises the `archiveHasV6RM` probe in `render-v6` (`render_v6.go:128-141`, `159-166`). Any implementation must keep working for this hybrid case, not just for cPages fixtures.

## Gap Analysis

| Capability | render-legacy | render-v6 today | Needed |
| --- | --- | --- | --- |
| V6 `.rm` annotations | ❌ `Unknown header` | ✅ | ✅ (already there) |
| Annotations-only output | ✅ (`--annotations-only`) | ❌ | **this ticket** |
| Skip unannotated pages in ann-only mode | ✅ (rmapi `AllPages=false` default) | n/a | **this ticket** |
| Page subset (`--pages`) | ✅ | ✅ | unchanged |
| Smart highlights / typed text in output | n/a (legacy format has none) | ✅ in composite | ✅ in ann-only output too |
| Cloud input (`--cloud`) | ✅ | ✅ | unchanged |

No library-level gaps: every building block (scene-tree parse, stroke extraction, overlay-only page builder, highlight annotation writer) already exists and is exercised by the current merge path and its tests (`pkg/rmdoc/render/v6_merge_background_test.go`).

## Proposed Solution

### CLI contract

Add one flag to `render-v6`, named identically to the legacy verb's:

```text
--annotations-only   Export annotations only (no background PDF); unannotated pages are
                     skipped unless explicitly selected with --pages   <bool> (default false)
```

Behavior matrix:

| Invocation | Output pages |
| --- | --- |
| `render-v6 doc.rmdoc` | all pages, composite (unchanged) |
| `render-v6 doc.rmdoc --annotations-only` | one blank-background page per **annotated** page, in UI order |
| `render-v6 doc.rmdoc --annotations-only --pages 1-3` | exactly pages 1–3; any selected page without annotations is emitted as a blank page (legacy parity: subset forces `AllPages`) |
| `render-v6 doc.rmdoc --annotations-only --pages 2` (page 2 unannotated) | 1 blank page |

For the workbook example: `--annotations-only` alone yields a 3-page PDF; `--annotations-only --pages 1-3` also yields a 3-page PDF; `--annotations-only --pages 1-4` yields 4 pages (page 4 blank).

### Library API sketch

One new exported function in a new file `pkg/rmdoc/render/v6_annotations_only.go`, deliberately mirroring the shape of the existing merge API so call sites read the same way:

```go
// RenderRMDocV6AnnotationsOnlyWithInfo renders only V6 annotation content
// (strokes, smart highlights, typed text) onto blank pages, without reading or
// compositing the background payload PDF.
//
// pageIndices are 0-based indexes into doc.Pages (UI order), same convention as
// MergeRMDocV6OntoBackgroundPDFWithInfoForPages.
//
// When includeUnannotated is false, pages without any V6 annotation content are
// skipped (rmapi PdfGeneratorOptions{AnnotationsOnly: true, AllPages: false} parity).
// When true, such pages are emitted as blank device-sized pages
// (the render-legacy CLI's forced AllPages behavior when --pages is used).
func RenderRMDocV6AnnotationsOnlyWithInfo(
    ctx context.Context,
    rmdocPath string,
    opts V6MergeOptions,
    pageIndices []int,
    includeUnannotated bool,
) (*V6MergeResult, error)
```

Notes on the shape:

- **Reuse `V6MergeOptions` and `V6MergeResult`.** `StrokeWidthPt` is still the fallback stroke width, and `HighlightsXTranslation` is still meaningful (the smart-highlight writer consumes it). No new options/result types means no new API surface to learn.
- **`includeUnannotated` is a function argument, not a new CLI flag.** The CLI derives it from `--pages` being present, exactly like legacy forces `AllPages` (`render_legacy.go:194`). This keeps the flag surface at parity: one flag, same name, same meaning.
- **No background is built.** The function never calls `BuildBackgroundPDF` and never parses `doc.PayloadPDF` — faster, and robust to corrupt payloads (see DR-5).

### Pseudocode

```text
func RenderRMDocV6AnnotationsOnlyWithInfo(ctx, path, opts, pageIndices, includeUnannotated):
    opts = opts.withDefaults()
    doc  = rmdoc.OpenFile(ctx, path)
    validate every idx in pageIndices within [0, len(doc.Pages))     # same errors as ForPages
    if len(pageIndices) == 0: return error "pageIndices is empty"

    writer = new PdfWriter
    highlightsXTranslation = []

    for i, pageIdx in enumerate(pageIndices):
        check ctx.Err()

        pageID = doc.Pages[pageIdx].PageID
        rm     = ReadRMFileFromArchive(ctx, path, pageID) if pageID != ""
        if rm missing or rm.Version != "V6" or rm.Bytes empty:
            if includeUnannotated: writer.AddPage(blankDeviceSizedPage())
            continue

        tree          = ParseRMV6SceneTree(rm.Bytes)
        strokes       = ExtractRMV6StrokesWithAnchors(tree)
        glyphRanges   = ExtractRMV6GlyphRangesWithAnchors(tree)
        paragraphs    = BuildRMV6TextDocument(tree.RootText)
        hasTypedText  = tree.RootText != nil and HasNonEmptyRMV6Text(paragraphs)

        if strokes empty and glyphRanges empty and not hasTypedText:
            if includeUnannotated: writer.AddPage(blankDeviceSizedPage())
            continue

        bbox = annotationCanvasBBox(strokes)      # extracted helper, see Phase 1
        scale = rmv6Scale * cairoSVGScale         # identical to existing overlay-only path
        pageW = (bbox.MaxX - bbox.MinX + 1) * scale
        pageH = (bbox.MaxY - bbox.MinY + 1) * scale

        page = buildOverlayOnlyPageBBoxScaled(pageW, pageH, strokes,
                                              tree.RootText, paragraphs,
                                              bbox, scale, opts)
        xTrans = -bbox.MinX * scale
        applySmartHighlightsScaled(page, glyphRanges, xTrans, pageH, scale)
        writer.AddPage(page)
        highlightsXTranslation.append(xTrans)

    return V6MergeResult{PDF: writer.Bytes(), HighlightsXTranslation: highlightsXTranslation}
```

And the CLI branch in `RenderV6Command.execute` (`render_v6.go:176-190` region):

```text
selection = parsePageSelection1Based(s.Pages, len(doc.Pages))
if s.AnnotationsOnly:
    res = render.RenderRMDocV6AnnotationsOnlyWithInfo(ctx, input.LocalPath,
            render.V6MergeOptions{}, selection.Indices0, !selection.All)
else:
    # existing code path, untouched
    if selection.All: res = MergeRMDocV6OntoBackgroundPDFWithInfo(...)
    else:             res = MergeRMDocV6OntoBackgroundPDFWithInfoForPages(...)
write res.PDF to out
```

### Dataflow diagrams

Current `render-v6` (composite):

```text
             ┌──────────────────────────── cmd/remarquee/cmds/rmdoc/render_v6.go ─────────────┐
 --file ───▶ │ ResolveRMDocInput (local | --cloud)  →  OpenFile  →  schema probe (archiveHasV6RM) │
 --pages ──▶ │ parsePageSelection1Based → Indices0                                              │
             └──────────────┬───────────────────────────────────────────────────────────────────┘
                            ▼
        pkg/rmdoc/render: MergeRMDocV6OntoBackgroundPDFWithInfo[ForPages]
                            │
            ┌───────────────┴────────────────┐
            ▼                                ▼
 BuildBackgroundPDF[ForPages]        per page: ReadRMFileFromArchive
 (payload PDF → UI-ordered bg)             │        ▼
            │                      ParseRMV6SceneTree → strokes / glyphRanges / text
            ▼                                ▼
      bg page has content? ── yes ──▶ buildMergedPage (Bg XObject + overlay)
            │ no
            └───────────────────────▶ buildOverlayOnlyPageBBoxScaled (blank canvas)
                            │
                            ▼
              applySmartHighlightsScaled → writer → out.pdf
```

Proposed `--annotations-only` branch:

```text
 render_v6.go execute(..., AnnotationsOnly: true)
        │  (background stage eliminated entirely)
        ▼
 RenderRMDocV6AnnotationsOnlyWithInfo(ctx, path, opts, Indices0, includeUnannotated=!selection.All)
        │
        ▼  per selected UI page
   .rm absent / not V6 / empty ──▶ skip   (or blank page if includeUnannotated)
        │ annotated
        ▼
   scene tree → strokes/glyphRanges/text
        ▼
   annotationCanvasBBox(strokes)  ──▶ buildOverlayOnlyPageBBoxScaled
        ▼
   applySmartHighlightsScaled ──▶ writer ──▶ out.pdf   (only annotated content, blank pages)
```

Page-emission decision tree:

```text
for each page in scope:
  has V6 annotation content?
    ├─ yes ───────────────────────────▶ render overlay-only page
    └─ no ── includeUnannotated? (= --pages given)
              ├─ true  ───────────────▶ emit blank device-sized page
              └─ false ───────────────▶ skip page entirely
```

## Decision Records

### Decision: DR-1 — dedicated renderer function instead of threading a bool through the merge loops

- **Context:** The feature needs "the overlay-only path, always, without a background". The two existing merge functions (`v6_merge_background.go:272`, `:506`) already contain near-identical 200-line per-page loops; a boolean could in theory redirect each iteration to the overlay-only branch and skip background construction.
- **Options considered:** (a) add `AnnotationsOnly bool` to `V6MergeOptions` and branch inside both loops; (b) a new dedicated function in a new file.
- **Decision:** (b) — new `RenderRMDocV6AnnotationsOnlyWithInfo` in `pkg/rmdoc/render/v6_annotations_only.go`.
- **Rationale:** The merge loops are golden-test-covered rendering code (`golden_remarks_test.go`); threading a mode switch through them triples their branching complexity and risks regressions in the default path. A dedicated function reuses the same unexported helpers (`buildOverlayOnlyPageBBoxScaled`, `applySmartHighlightsScaled`, `maxStrokeWidthScreenUnits`) because it lives in the same package, keeps the default path byte-identical, and is far easier for a new engineer to read in isolation. It also lets us skip building the background PDF entirely (DR-5), which option (a) would make awkward.
- **Consequences:** A third copy of the small "read rm → parse → extract" prologue appears; mitigated by Phase 1 extracting the canvas-bbox computation into a shared helper used by all three loops. If the merge loops are ever refactored into one, the annotations-only function should join that refactor.
- **Status:** proposed

### Decision: DR-2 — page emission semantics (skip vs blank)

- **Context:** Annotations-only output must decide what to do with pages that have no annotations; the workbook example (3 of 44 pages annotated) makes this the dominant UX question.
- **Options considered:** (a) always emit every in-scope page (blank when unannotated); (b) always skip unannotated pages; (c) skip by default, but emit blanks for pages the user explicitly selected with `--pages`.
- **Decision:** (c), via the `includeUnannotated = !selection.All` derivation.
- **Rationale:** This is exactly the legacy behavior users already know: rmapi skips unannotated pages unless `AllPages` (`annotations/pdf.go:96`), and the legacy CLI forces `AllPages` when a subset is selected (`render_legacy.go:194`). Skipping by default avoids a 44-page PDF with 41 blank pages; emitting selected blanks avoids the "I asked for pages 1-4 and got 3 pages" confusion.
- **Consequences:** Output page count is not always equal to input page count — this must be documented in help text and in the Glaze output (see Phase 3 `selected_pages` row field). Legacy's `AllPages` forcing exists for a technical reason (index alignment for post-extraction) that does not apply to v6's pre-selection architecture; we adopt its *user-visible* semantics, not its mechanism.
- **Status:** proposed

### Decision: DR-3 — page geometry reuses the existing overlay-only canvas math

- **Context:** On blank pages we must pick a page size and stroke transform. Candidates: (a) the existing bbox-derived canvas used by the overlay-only branch (`v6_merge_background.go:400-449`): full-screen default bbox ∪ padded stroke bbox, scaled by `rmv6Scale * cairoSVGScale`; (b) a fixed device page (1404×1872 units × `rmv6Scale` ≈ 447×596 pt, or × `cairoSVGScale` ≈ 335×447 pt); (c) the rmapi legacy constant 445×594 pt (`rmPageSize`).
- **Decision:** (a) — reuse `buildOverlayOnlyPageBBoxScaled` with the identical bbox/scale computation.
- **Rationale:** Zero new rendering code; output is byte-for-byte consistent with what the current pipeline already produces for blank-background pages, so reviewers can eyeball annotations-only pages against existing renders of inserted/notebook pages; strokes that extend beyond the default screen area are not clipped (the bbox union handles them); smart highlights and typed text inherit their already-validated positioning.
- **Consequences:** Page sizes can vary slightly per page (stroke-extent padding), matching existing behavior rather than legacy's fixed 445×594 pt. Parity with legacy is behavioral (blank pages, annotation content), not geometric; this is acceptable because v6 already uses different geometry than rmapi everywhere.
- **Status:** proposed

### Decision: DR-4 — default output filename

- **Context:** When `--out` is omitted, `render-v6` writes `<input>-v6.pdf` (`render_v6.go:170-172` via `defaultOutputPath`, `input_resolver.go:104`). Writing annotations-only output to the same default name makes accidental overwrites of a full render likely (mitigated only by the `--force` guard).
- **Options considered:** (a) keep `-v6.pdf` regardless; (b) use `-v6-annotations.pdf` when `--annotations-only` is set; (c) reuse legacy's `-annotations.pdf`.
- **Decision:** (b).
- **Rationale:** Self-describing, stays in the `-v6*` namespace, and cannot collide with either existing default. (c) would collide with `render-legacy`'s default for the same input file.
- **Consequences:** One conditional in `execute`; must be covered by a CLI test.
- **Status:** proposed

### Decision: DR-5 — do not open the payload PDF in annotations-only mode

- **Context:** The merge path parses `doc.PayloadPDF` via `BuildBackgroundPDF*` before rendering. Annotations-only output needs nothing from the payload.
- **Options considered:** (a) still build the background and ignore it (uniform code path); (b) skip it.
- **Decision:** (b).
- **Rationale:** Real speed and memory win on large payloads (the workbook's 44-page letter PDF; the 12 MB `cpage-pdf.rmdoc` fixture); removes a failure mode (corrupt/encrypted payload would break an annotations-only export that doesn't need it); blank-page emission needs only the device size constants.
- **Consequences:** Page *geometry* can no longer be derived from the payload (DR-3 makes this moot). `doc` is still opened for the page plan and pageIDs.
- **Status:** proposed

## Implementation Phases

All work happens on branch `task/remarquee-v6-only-annotations`. Run `go test ./... -count=1` and `gofmt -w <files>` at the end of every phase; commit per phase.

### Phase 1 — extract the shared canvas-bbox helper (`pkg/rmdoc/render/v6_merge_background.go`)

The bbox + padding + margin-compensation computation is currently inline-duplicated in both merge loops (`v6_merge_background.go:388-437` and `604-653`). Extract it verbatim (behavior-preserving) into:

```go
// annotationCanvasBBox returns the annotation canvas bbox for a page:
// the full-screen default bbox unioned with the stroke bbox, expanded by
// half the maximum stroke width, with left/right margin compensation so the
// canvas stays centered over the default screen extents.
func annotationCanvasBBox(strokes []rmdoc.Stroke) (bbox rmdoc.BBox, scale float64)
```

Also extract the `blankDeviceSizedPage()` one-pager used by DR-2 (a `pdf.PdfPage` with MediaBox/CropBox of `rmv6ScreenWidth*rmv6Scale*cairoSVGScale` × `rmv6ScreenHeight*...`, empty resources). Replace both inline copies with calls. **Validation:** `go test ./pkg/rmdoc/... -count=1` must pass with zero golden changes — this phase must not alter output bytes.

### Phase 2 — the annotations-only renderer (`pkg/rmdoc/render/v6_annotations_only.go`, new)

Implement `RenderRMDocV6AnnotationsOnlyWithInfo` per the pseudocode above. Error messages should copy the style of `MergeRMDocV6OntoBackgroundPDFWithInfoForPages` (`v6_merge_background.go:511-523`): wrap with `errors.Wrapf` including page index and pageID.

Unit tests in `pkg/rmdoc/render/v6_annotations_only_test.go` (new) — fixtures already exist in `cmd/remarquee-ui/testdata/generated/`:

| Test | Fixture | Assertion |
| --- | --- | --- |
| `TestRenderV6AnnotationsOnly_SkipsUnannotatedPages` | `fake-cpages-pdf-v6-sample-rm.rmdoc` (2 UI pages, `.rm` only for page 1) | `includeUnannotated=false` → 1 page; contains `rmv6-overlay` marker |
| `TestRenderV6AnnotationsOnly_EmitsBlanksWhenIncluded` | same fixture | `includeUnannotated=true` → 2 pages; page 2 has empty content stream |
| `TestRenderV6AnnotationsOnly_NoAnnotationsAtAll` | `fake-cpages-pdf-no-annotations.rmdoc` (4 pages, no `.rm`) | `includeUnannotated=false` → 0-page PDF (valid header); `true` → 4 blank pages |
| `TestRenderV6AnnotationsOnly_EmptyRMFile` | `fake-cpages-pdf-v6-empty-rm.rmdoc` | header-only `.rm` treated as unannotated |
| `TestRenderV6AnnotationsOnly_PageIndexValidation` | sample fixture | out-of-range index → error mentioning `pageIndices` |
| `TestRenderV6AnnotationsOnly_NoBackgroundXObject` | sample fixture | content stream contains no `/Bg` Do operator (proves DR-5) |

Note the fixture layout asymmetry: `fake-cpages-pdf-v6-sample-rm.rmdoc` stores its one `.rm` at `<uuid>/p0.rm` and has a second page (`p1`) with none — purpose-built for DR-2 testing. The 12 MB real-world fixture `cpage-pdf.rmdoc` should also get a smoke test asserting page count equals its annotated-page count.

### Phase 3 — CLI wiring (`cmd/remarquee/cmds/rmdoc/render_v6.go`)

1. Add `AnnotationsOnly bool \`glazed:"annotations-only"\`` to `RenderV6Settings` (after `Force`, mirroring legacy's ordering).
2. Add the flag definition in `NewRenderV6Command` (copy legacy's help string, `render_legacy.go:85-90`, extended with the skip behavior note).
3. In `execute`, branch on `s.AnnotationsOnly` per the pseudocode; use suffix `-v6-annotations.pdf` for the default output (DR-4).
4. Add `types.MRP("annotations_only", res.AnnotationsOnly)` to the Glaze row; add the field to `renderV6Execution`.
5. Update the command's `WithLong` help: new bullet explaining annotations-only + page emission rules.

CLI tests in `cmd/remarquee/cmds/rmdoc/render_v6_test.go` (same `newDefaultParsedValues` helper as the existing smoke tests, `test_values_helpers_test.go`):

- `TestRenderV6Command_AnnotationsOnly`: sample fixture, expect valid PDF with 1 page.
- `TestRenderV6Command_AnnotationsOnlyPagesSubset`: sample fixture with `pages: "1,2"`, expect 2 pages (blank second page, DR-2).
- `TestRenderV6Command_AnnotationsOnlyDefaultOutName`: assert default output ends with `-v6-annotations.pdf` (DR-4).

### Phase 4 — docs, manual validation, handoff

- README: add one example line next to the existing `render-v6` examples.
- Manual validation on the real workbook (evidence for changelog):
  ```bash
  go run ./cmd/remarquee rmdoc render-v6 --cloud --non-interactive \
    "/ai/2026/08/03/TTC-GARDEN-UXQA-001/TTC Garden Human Calibration Workbook" \
    --annotations-only --pages 1-3 --out /tmp/workbook-ann-only.pdf --force
  pdfinfo /tmp/workbook-ann-only.pdf        # expect 3 pages
  pdftoppm -png -r 100 /tmp/workbook-ann-only.pdf /tmp/ann && # eyeball: strokes only, white background
  ```
  Also run without `--pages` and confirm the same 3 pages (skip semantics), and rasterize side-by-side with the composite render for visual review.
- Update ticket changelog/tasks/diary; `docmgr doctor --ticket RMQ-0021`.

## Testing and Validation Strategy

- **Regression fence:** the entire existing suite (`go test ./... -count=1`) must pass after Phase 1 with no golden updates; goldens live behind `pkg/rmdoc/render/golden_remarks_test.go` and friends.
- **Unit level (library):** the Phase 2 table — page-count semantics (DR-2), no-background proof (DR-5), error paths.
- **CLI level:** Phase 3 tests through the real cobra/glazed stack, including Glaze row contents.
- **Integration/manual:** Phase 4 workbook run against the cloud path (which exercises `ResolveRMDocInput`, the legacy-schema + V6-rm probe, and the new renderer end-to-end). Compare page counts and visually inspect rasterized pages; `vlm-validate` can be used for an automated eyeball if needed (`cmd/remarquee/cmds/rmdoc/vlm_validate.go`).
- **What "done" looks like:** `render-v6 --annotations-only` on the workbook produces a 3-page PDF showing only handwriting on white pages; `render-legacy --annotations-only` still fails on that file (documenting exactly why the new flag is needed).

## Risks, Alternatives, Open Questions

### Risks

- **Semantic surprise:** users expecting *every* page (like composite mode) get only annotated pages. Mitigated by help text and the Glaze `selected_pages` field; a future `--all-pages` flag could force emission (deliberately not added now — one flag at parity, DR-2).
- **Drift between the three per-page loops.** Mitigated by Phase 1 helper extraction; a deeper unification of the merge loops is a separate refactor ticket.
- **Golden sensitivity:** any accidental change to the default path fails golden tests — which is the desired early warning, but means Phase 1 must be purely mechanical.

### Alternatives considered

- **Thread `AnnotationsOnly` through `V6MergeOptions`** — rejected (DR-1).
- **Post-process the composite PDF** (render full, then strip background form XObjects) — rejected: fragile (content-stream surgery), wasteful, and unipdf gives us the clean primitive already.
- **Expose `includeUnannotated` as a CLI `--all-pages` flag on render-v6** — deferred; legacy parity is achieved without it.

### Open questions

1. Should `render-v6-png` grow the same flag (it rasterizes the merged PDF, so it would be a one-line pass-through)? Follow-up ticket.
2. Should the blank-page size for `includeUnannotated` pages equal the *payload* page size instead of the device size? That would require reading the payload (violates DR-5); revisit only if users ask.
3. Do we want outline/bookmark preservation ever? Legacy explicitly drops outlines on subset extraction; v6 has no outlines today. Non-goal.

## References

### Code (this repository)

- `cmd/remarquee/cmds/rmdoc/render_v6.go` — v6 verb: settings (line 28), schema probe `archiveHasV6RM` (line 128), `execute` (line 144), merge calls (lines 182-186), cobra wrapper (line 232).
- `cmd/remarquee/cmds/rmdoc/render_legacy.go` — legacy verb: `AnnotationsOnly` setting (line 36), flag def (lines 84-89), rmapi options (lines 192-197).
- `cmd/remarquee/cmds/rmdoc/pages.go` — `parsePageSelection1Based` (line 19), `extractPDFPages` (line 109).
- `cmd/remarquee/cmds/rmdoc/input_resolver.go` — `CloudInputSettings` (line 16), `ResolveRMDocInput` (line 61), `defaultOutputPath` (line 104), `ensureOutputWritable` (line 113).
- `cmd/remarquee/cmds/rmdoc/render_v6_test.go`, `test_values_helpers_test.go` — CLI test harness patterns.
- `pkg/rmdoc/render/v6_merge_background.go` — `V6MergeOptions` (line 19), merge entry points (lines 264/272/506), blank-background branch (lines 399-400, 634-635), `buildOverlayOnlyPageBBoxScaled` (line 755), `buildMergedPage` (line 1010), `applySmartHighlightsScaled` (line 1075).
- `pkg/rmdoc/render/background.go` — `BackgroundOptions` (line 15), `BuildBackgroundPDF`/`ForPages` (lines 39/51).
- `pkg/rmdoc/render/v6_strokes_pdf.go` — screen geometry constants (lines 16-24).
- `pkg/rmdoc/types.go`, `open.go`, `rm_archive_rmfiles.go`, `rmv6_scene_tree.go`, `rmv6_strokes_extract.go`, `rmv6_glyph_extract.go`, `rmv6_text_document.go`, `bbox.go`, `pen_color.go` — document model and V6 parse pipeline.
- `cmd/remarquee-ui/testdata/generated/` — fixtures: `fake-cpages-pdf-v6-sample-rm.rmdoc` (2 pages, 1 annotated), `fake-cpages-pdf-no-annotations.rmdoc`, `fake-cpages-pdf-v6-empty-rm.rmdoc`; generator in `testdata/gen_fakes/main.go`.

### Vendored dependency (legacy semantics)

- `github.com/marcobarcelos/rmapi@.../annotations/pdf.go` — `PdfGeneratorOptions` (lines 38-39), skip-unannotated logic (line 96), `addBackgroundPage` annotations-only branch (lines 218-244).

### Prior ticket docs

- `ttmp/2026/05/29/RMQ-0019--add-pages-to-rmdoc-render-commands/design-doc/01-...md` — the `--pages` feature this flag composes with.
- `ttmp/2026/03/28/RMQ-0016--add-cloud-flag-to-rmdoc-render-commands/design-doc/01-...md` — the `--cloud` input path.
- `ttmp/2025/12/14/RMQ-0004--port-rmdoc-parsing-rendering-to-go-v3-v5-v6/` — V6 data model design.
