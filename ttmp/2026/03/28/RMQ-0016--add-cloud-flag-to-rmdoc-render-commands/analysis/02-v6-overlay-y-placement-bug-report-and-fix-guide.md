---
Title: V6 overlay y-placement bug report and fix guide
Ticket: RMQ-0016
Status: complete
Topics:
    - cli
    - cloud
    - rendering
    - rmdoc
    - remarkable
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/remarquee/cmds/rmdoc/render_v6.go
      Note: |-
        CLI entrypoint that calls the V6 merge renderer after resolving local or cloud input
        CLI entrypoint discussed in the guide
    - Path: pkg/rmdoc/open.go
      Note: |-
        Archive-opening layer that turns a .rmdoc zip into a parsed Document
        Archive opening layer explained in the guide
    - Path: pkg/rmdoc/render/background.go
      Note: |-
        Background-PDF assembly used before the overlay merge
        Background assembly stage explained in the guide
    - Path: pkg/rmdoc/render/v6_merge_background.go
      Note: |-
        Main V6 merge implementation and the location of the Y-placement bug and fix
        Main renderer bug and fix location
    - Path: pkg/rmdoc/render/v6_merge_background_test.go
      Note: |-
        Focused regression coverage for the coordinate conversion helper
        Regression test referenced by the guide
ExternalSources: []
Summary: Detailed bug report and fix guide for the V6 PDF-backed overlay Y-placement regression, including rendering architecture, coordinate-system background, reproduction, root cause, patch explanation, and validation guidance for a new intern.
LastUpdated: 2026-03-28T11:24:17-04:00
WhatFor: Help a new engineer understand how V6 rendering works, why the overlay drifted downward on later pages, and how the final fix addresses the bug without changing cloud-download behavior.
WhenToUse: Use when debugging V6 render placement issues, onboarding to the RMQ-0016 renderer changes, or reviewing why commit 24eeb98 was necessary.
---


# V6 overlay y-placement bug report and fix guide

## Executive Summary

This document explains a rendering regression that appeared when running `remarquee rmdoc render-v6` on a PDF-backed V6 `.rmdoc` document. The user observed that handwritten annotations on later pages appeared too low on the page. In the concrete report, a note that should have landed over the phrase "programming language" instead appeared over text farther down the page, near "When you see it that way".

The important conclusion is that this was **not** a cloud-download bug. The same broken output happened once the `.rmdoc` archive was local. The real problem lived inside the V6 merge algorithm in [`pkg/rmdoc/render/v6_merge_background.go`](../../../../../pkg/rmdoc/render/v6_merge_background.go): we were using a top-origin Y-placement formula copied from the Python `remarks` implementation, but we were applying it inside a bottom-origin PDF coordinate system without converting it properly.

The fix was to make the coordinate conversion explicit. We now convert "top-origin placement" into "bottom-origin PDF placement" before composing the background page and the V6 overlay. That restored the expected page-relative position of annotations on PDF-backed pages.

## Why This Matters

For an intern, this bug is a good learning exercise because it touches several core concepts at once:

- how the CLI command reaches the rendering engine
- how `.rmdoc` archives combine structured metadata, page plans, `.rm` annotation files, and sometimes a payload PDF
- why rendering systems often need to translate between multiple coordinate spaces
- how a formula that is mathematically correct in one coordinate system can become wrong when copied into another
- how to debug rendering regressions by isolating the pipeline into stages

This is also a good example of a class of bug that is common in graphics and layout work:

- nothing crashes
- most pages look "roughly" correct
- the output seems plausible
- but one axis is shifted because the origin convention changed

## The Bug Report

### User-reported symptom

The user ran:

```bash
go run ./cmd/remarquee rmdoc render-v6 --cloud /Articles/claude-code-v3.md
```

They reported that, starting around page 3, the notes were no longer aligned with the intended lines of text.

The concrete example they gave was:

- expected behavior:
  - the phrase "programming language" should be crossed out and replaced with "problem"
- observed behavior:
  - the handwritten "problem" note appeared on top of later text, near "When you see it that way"

### What that symptom tells us

This symptom strongly suggests a **page-local positioning problem** rather than a parsing failure:

- the annotations still rendered
- their shapes still looked like the original handwriting
- the issue was not random noise or a missing page
- the overlay was shifted downward, not completely missing

That means the likely bug classes were:

- wrong page selection
- wrong page-to-page ordering
- wrong Y-coordinate conversion
- wrong vertical alignment when combining overlay and background

The symptom did **not** initially suggest:

- broken cloud download
- corrupted `.rm` parsing
- corrupted payload PDF
- missing annotation files

## Important Scope Clarification: This Was Not A `--cloud` Bug

The user triggered the issue through `--cloud`, so it was reasonable to start there. But the investigation showed that `--cloud` only changed **how the archive arrived locally**, not how the renderer interpreted it.

The actual flow is:

1. `render-v6` resolves the input.
2. If `--cloud` is present, it downloads the `.rmdoc` archive into a temporary local file.
3. The renderer then runs on that local archive path exactly as if the user had provided a local `.rmdoc` directly.

The relevant CLI call path is in [`cmd/remarquee/cmds/rmdoc/render_v6.go`](../../../../../cmd/remarquee/cmds/rmdoc/render_v6.go):

- `execute(...)` resolves input with `ResolveRMDocInput(...)` at lines 118-123
- it opens the archive with `pkg_rmdoc.OpenFile(...)` at lines 125-128
- it calls `MergeRMDocV6OntoBackgroundPDFWithInfo(...)` at lines 144-146

That means:

- if the rendered output is wrong after the archive has been downloaded
- and the same wrong output appears when using the local `.rmdoc` directly
- then the bug is inside the rendering code, not the cloud-input code

## System Overview For A New Intern

This section explains the parts of the system you need to know before the bug itself makes sense.

## Mental Model Of The Rendering Pipeline

Think of the V6 renderer as a pipeline with four major stages:

1. **Input resolution**
   - turn either a local path or a remote cloud path into a local `.rmdoc` archive path
2. **Archive opening**
   - parse the `.rmdoc` zip archive into an internal `Document`
3. **Background construction**
   - build the page background PDF in UI page order
4. **Overlay merge**
   - render V6 strokes and smart highlights and place them on top of the background pages

ASCII diagram:

```text
User CLI
  |
  v
render-v6 command
  |
  +--> ResolveRMDocInput(...)
  |       |
  |       +--> local file path
  |       +--> or cloud download to temp file
  |
  v
OpenFile(...)
  |
  v
Document
  |
  +--> page plan from .content / .pagedata
  +--> payload PDF bytes (if PDF-backed)
  +--> page IDs for .rm annotation files
  |
  v
BuildBackgroundPDF(...)
  |
  v
background PDF in UI order
  |
  v
MergeRMDocV6OntoBackgroundPDFWithInfo(...)
  |
  +--> parse per-page .rm scene tree
  +--> extract strokes and highlights
  +--> compute bbox and placement
  +--> compose background + overlay
  |
  v
final annotated PDF
```

## What A `.rmdoc` Archive Contains

A `.rmdoc` file is a zip archive. The parts relevant to this bug are:

- `.content`
  - contains the document structure and page plan
- `.pagedata`
  - stores template/background hints for certain page styles
- `.pdf`
  - the original payload PDF for PDF-backed documents
- `<docUUID>/<pageID>.rm`
  - annotation scene files for each page

The archive-open path is implemented in [`pkg/rmdoc/open.go`](../../../../../pkg/rmdoc/open.go):

- `OpenFile(...)` at lines 15-28 opens the archive
- `OpenReaderAt(...)` at lines 30-109 reads the zip contents
- it loads `.content`, optional `.pagedata`, optional `.pdf`, and builds a `Document`

Conceptually:

```text
.rmdoc zip
  |
  +--> .content     -> page ordering and metadata
  +--> .pdf         -> original page visuals
  +--> page .rm     -> user pen strokes / scene graph
  +--> .pagedata    -> template hints
```

## What "PDF-backed" Means

There are two broad rendering cases:

- notebook-style documents
  - there is no payload PDF
  - the renderer effectively draws onto blank pages
- PDF-backed documents
  - there is a real source PDF
  - the renderer must merge annotations onto those existing PDF pages

This bug only affected the **PDF-backed V6 merge path**.

Why that matters:

- notebook pages already behave more like overlay-only output
- PDF-backed pages require us to position two things relative to each other:
  - the original PDF background
  - the rendered annotation overlay

That alignment step is where the bug lived.

## Background Construction

Before any handwritten marks are merged, the renderer constructs the background PDF in UI page order. That happens in [`pkg/rmdoc/render/background.go`](../../../../../pkg/rmdoc/render/background.go).

The key function is `BuildBackgroundPDFForPages(...)` at lines 50-143.

For PDF-backed docs:

- if a UI page refers to a real source PDF page, the code duplicates that PDF page
- if a UI page is an inserted page, the code creates a blank page of the appropriate size

This stage is important because the overlay merge stage assumes it is placing marks on top of these background pages.

## V6 Overlay Merge

The main merge logic is in [`pkg/rmdoc/render/v6_merge_background.go`](../../../../../pkg/rmdoc/render/v6_merge_background.go).

At a high level, for each page it does:

1. load the background page from the assembled background PDF
2. load the matching `.rm` annotation file for that page
3. parse the V6 scene tree
4. extract strokes and glyph ranges
5. compute a bounding box for the annotation content
6. compute how the overlay and background should sit inside a merged page canvas
7. create the merged PDF page

In pseudocode:

```text
for each ui page:
    bgPage = backgroundPdf[page]
    rmData = read matching .rm

    sceneTree = parse rmData
    strokes = extract strokes
    glyphRanges = extract smart highlights
    bbox = compute annotation bbox

    wBg, hBg = background display dimensions
    wSvg, hSvg = overlay dimensions derived from bbox

    width = max(wSvg, wBg)
    height = max(hSvg, hBg)

    compute x/y placement for background and overlay
    compose merged page
```

## Coordinate Systems: The Most Important Concept In This Bug

This is the part a new intern must understand carefully.

### Coordinate system 1: reMarkable screen space

The reMarkable scene uses a screen-like coordinate space:

- X is centered around 0
- Y starts near the top and increases downward

Roughly:

```text
top of device
Y = 0
|
|
v
larger Y values lower on the screen
```

### Coordinate system 2: remarks / MuPDF placement space

The Python `remarks` implementation uses placement logic that behaves like a **top-origin layout system** when thinking about where the overlay should go.

That means:

- `topY = 0` means "touch the top of the page/canvas"
- larger `topY` means "move lower"

### Coordinate system 3: PDF content stream space

PDF content streams use a **bottom-origin** coordinate system:

- `y = 0` is the bottom edge
- larger `y` means "move upward"

That means the same intuitive placement statement has to be converted:

- "place this object 50 points from the top"
- cannot be written directly as `y = 50`
- it must become:
  - `y = pageHeight - topOffset - objectHeight`

ASCII comparison:

```text
Top-origin system                    Bottom-origin PDF system

(0,0) top-left-ish                   y=max at top
  +------------------+                 +------------------+
  | object           |                 | object           |
  |                  |                 |                  |
  |                  |                 |                  |
  +------------------+                 +------------------+
  topY = 0                            y = pageHeight - objectHeight

y grows downward                      y grows upward
```

## The Exact Failure Mechanism

The broken code path was in the "background-present behavior" branch in [`pkg/rmdoc/render/v6_merge_background.go`](../../../../../pkg/rmdoc/render/v6_merge_background.go).

The relevant lines after the fix are 449-481.

Before the fix, the code conceptually did this:

```text
if overlay is shorter than background:
    ySvg = yShift
```

That logic is reasonable in a top-origin system, because it says:

- if the overlay box is shorter than the page
- place it `yShift` down from the top

But we were later feeding `ySvg` directly into PDF content-stream placement, where Y means **distance from the bottom**, not from the top.

So the overlay ended up being effectively bottom-aligned relative to the canvas when it should have been top-aligned.

### Why the downward shift had the size it did

The bug was especially visible on letter-sized pages.

Example:

- background page height:
  - `792 pt`
- overlay page height derived from reMarkable screen:
  - about `596 pt`

If you incorrectly treat `topY = 0` as `pdfY = 0`, then the overlay starts at the bottom.

The correct bottom-origin Y position should instead be:

```text
pdfBottomY = pageHeight - topY - overlayHeight
           = 792 - 0 - 596
           = 196
```

That `196 pt` discrepancy is exactly why notes drawn near the top of the page drifted down into the middle of the page.

## Why The Cloud Path Was Innocent

The user reproduced through:

```bash
go run ./cmd/remarquee rmdoc render-v6 --cloud /Articles/claude-code-v3.md
```

But the renderer eventually ran on a local archive path either way. After downloading the archive, reproducing with the local file showed the same problem. That ruled out:

- authentication bugs
- cloud path resolution bugs
- temp-file cleanup bugs

The cloud path only acted as an input transport step. The merge bug was already present in the local renderer.

## Investigation Walkthrough

This is the debugging sequence that proved the root cause.

### Step 1: Reframe the report

The first job was to avoid assuming that `--cloud` caused the bug.

We asked:

- Does the background text render correctly?
- Do the annotations render with the right shape?
- Is the problem page selection, or position within the page?

Observation:

- the background text looked correct
- the annotation strokes still looked like the right handwriting
- only the vertical placement was wrong

That pointed to placement math.

### Step 2: Inspect the generated PDF directly

We inspected the already-generated `claude-code-v3.md-v6.pdf` and rasterized page 3 into an image. That confirmed:

- the background PDF was fine
- the overlay was present
- the overlay was too low

### Step 3: Download the underlying `.rmdoc`

We downloaded:

```bash
remarquee cloud get "/Articles/claude-code-v3.md" --out-dir /tmp/claude-code-v3-cloud --non-interactive
```

Then inspected it:

```bash
go run ./cmd/remarquee rmdoc inspect /tmp/claude-code-v3-cloud/claude-code-v3.md.rmdoc
```

This confirmed:

- the document is `schema=cPages`
- the document is `type=pdf`
- it has `7` pages

That matters because the bug only affects the PDF-backed merge branch.

### Step 4: Inspect the `.rm` scene data

We dumped page-level information and found something important:

- these pages had no `RootText`
- these pages had no glyph-range highlight rectangles
- they contained only stroke annotations

That simplified the problem:

- typed-text anchor handling was not the immediate issue
- highlight annotation placement was not the immediate issue
- the visible bug was stroke overlay placement

### Step 5: Compare our assumptions against `remarks`

The V6 merge implementation intentionally mirrors `remarks` placement math. That was the right instinct, but there was a subtle trap:

- `remarks` uses PyMuPDF placement semantics
- our Go implementation builds PDF content streams directly

Those are not interchangeable coordinate systems.

The copied vertical logic was semantically correct in the `remarks` world, but incomplete in the raw PDF world.

### Step 6: Identify the exact missing conversion

Once we isolated the mismatch, the missing formula became clear:

```text
top-origin placement  ->  bottom-origin placement

pdfBottomY = pageHeight - topY - objectHeight
```

That became the helper:

```go
func remarksTopYToPDFBottomY(pageHeight, topY, objectHeight float64) float64 {
    return pageHeight - topY - objectHeight
}
```

Implemented at lines 980-983 in [`pkg/rmdoc/render/v6_merge_background.go`](../../../../../pkg/rmdoc/render/v6_merge_background.go).

## The Fix

## Code-Level Summary

The fix changed the PDF-backed merge branch in two places:

- the full-document merge path at lines 449-481
- the selected-pages merge path at lines 676-706

Before:

- vertical placement variables (`ySvg`, `yBg`) were used directly
- those values were effectively treated as if PDF Y were top-origin

After:

- we keep the intermediate values as `ySvgTop` and `yBgTop`
- we explicitly convert them with `remarksTopYToPDFBottomY(...)`
- the resulting `ySvg` and `yBg` are safe to use in PDF space

### Before/after pseudocode

Before:

```text
if overlay shorter than background:
    ySvg = yShift

if background shorter than overlay:
    yBg = -yShift

place overlay at ySvg in PDF
place background at yBg in PDF
```

After:

```text
if overlay shorter than background:
    ySvgTop = yShift

if background shorter than overlay:
    yBgTop = -yShift

ySvg = pageHeight - ySvgTop - overlayHeight
yBg  = pageHeight - yBgTop  - backgroundHeight

place overlay at ySvg in PDF
place background at yBg in PDF
```

## Why This Fix Is Correct

The fix is correct because it preserves the original layout intent:

- "where should this object sit relative to the top of the merged canvas?"

while translating it into the coordinate system actually used by the PDF writer:

- "what bottom-origin Y value produces that same visual top alignment?"

It does **not** change:

- page ordering
- cloud download logic
- stroke parsing
- glyph extraction
- annotation colors
- rotation handling

It changes only the Y-coordinate convention at the seam where top-origin layout is translated into PDF placement.

## Files You Should Read In Order

If you are new to this codebase, read these files in this order.

### 1. CLI entrypoint

[`cmd/remarquee/cmds/rmdoc/render_v6.go`](../../../../../cmd/remarquee/cmds/rmdoc/render_v6.go)

Why:

- this shows how the renderer is called from the command layer
- it proves that `--cloud` only affects input resolution
- it shows where the merge function is invoked

Key area:

- lines 118-149

### 2. Archive opening

[`pkg/rmdoc/open.go`](../../../../../pkg/rmdoc/open.go)

Why:

- this explains what information comes out of a `.rmdoc`
- it shows how `.content`, `.pagedata`, and `.pdf` are loaded

Key area:

- lines 30-109

### 3. Background creation

[`pkg/rmdoc/render/background.go`](../../../../../pkg/rmdoc/render/background.go)

Why:

- this shows how the UI-ordered background PDF is assembled
- it clarifies the difference between PDF-backed pages and blank inserted pages

Key area:

- lines 50-143

### 4. The main V6 merge logic

[`pkg/rmdoc/render/v6_merge_background.go`](../../../../../pkg/rmdoc/render/v6_merge_background.go)

Why:

- this is the core rendering algorithm
- this is where the bug and fix live

Key areas:

- lines 449-481
- lines 676-706
- lines 980-983

### 5. The regression test

[`pkg/rmdoc/render/v6_merge_background_test.go`](../../../../../pkg/rmdoc/render/v6_merge_background_test.go)

Why:

- this documents the coordinate conversion expectation in a minimal form

Key area:

- lines 155-170

## API Reference For This Bug

These are the most relevant functions for understanding the bug.

### `ResolveRMDocInput(...)`

Location:

- [`cmd/remarquee/cmds/rmdoc/input_resolver.go`](../../../../../cmd/remarquee/cmds/rmdoc/input_resolver.go)

Purpose:

- normalize local or cloud input into a local `.rmdoc` path

Why it matters:

- proves that cloud mode is not the rendering bug itself

### `OpenFile(...)`

Location:

- [`pkg/rmdoc/open.go`](../../../../../pkg/rmdoc/open.go)

Purpose:

- parse a `.rmdoc` archive into a `Document`

Why it matters:

- gives the renderer the page plan and payload PDF

### `BuildBackgroundPDFForPages(...)`

Location:

- [`pkg/rmdoc/render/background.go`](../../../../../pkg/rmdoc/render/background.go)

Purpose:

- build the background PDF in UI page order

Why it matters:

- the overlay must be positioned relative to this output

### `MergeRMDocV6OntoBackgroundPDFWithInfo(...)`

Location:

- [`pkg/rmdoc/render/v6_merge_background.go`](../../../../../pkg/rmdoc/render/v6_merge_background.go)

Purpose:

- merge V6 strokes/highlights onto the background PDF

Why it matters:

- the bug existed in this function’s PDF-backed merge branch

### `remarksTopYToPDFBottomY(...)`

Location:

- [`pkg/rmdoc/render/v6_merge_background.go`](../../../../../pkg/rmdoc/render/v6_merge_background.go)

Purpose:

- convert top-origin placement intent into bottom-origin PDF coordinates

Why it matters:

- this is the exact conceptual repair for the bug

## Concrete Numerical Example

This example is worth working through manually.

Given:

- merged page height = `792`
- overlay height = `596`
- intended top offset = `0`

If you use top-origin placement directly inside PDF:

```text
y = 0
```

the overlay starts at the bottom edge of the page, which is wrong.

The correct PDF bottom-origin placement is:

```text
y = 792 - 0 - 596 = 196
```

That means:

- overlay bottom edge is at `196`
- overlay top edge is at `196 + 596 = 792`
- so the overlay touches the top of the page exactly as intended

This is the key mental translation for the whole bug.

## Reproduction Guide

### Exact user-facing reproduction

```bash
go run ./cmd/remarquee rmdoc render-v6 --cloud "/Articles/claude-code-v3.md"
```

Expected bug before the fix:

- page 3 annotations appear too low

### Local-archive reproduction

```bash
remarquee cloud get "/Articles/claude-code-v3.md" --out-dir /tmp/claude-code-v3-cloud --non-interactive
go run ./cmd/remarquee rmdoc render-v6 /tmp/claude-code-v3-cloud/claude-code-v3.md.rmdoc --out /tmp/claude-code-v3-buggy.pdf --force
```

Why this version is useful:

- removes cloud transport from the equation
- lets you focus only on rendering

### Page rasterization for visual inspection

```bash
mkdir -p /tmp/claude-code-v3-pages
pdftoppm -f 3 -l 4 -png /tmp/claude-code-v3-buggy.pdf /tmp/claude-code-v3-pages/page
```

Open:

- `/tmp/claude-code-v3-pages/page-3.png`
- `/tmp/claude-code-v3-pages/page-4.png`

## Fix Verification Guide

After the patch:

1. run focused tests
2. rerender the document
3. compare the same pages again

### Focused tests

```bash
go test ./pkg/rmdoc/render ./cmd/remarquee/cmds/rmdoc
```

### Local rerender

```bash
go run ./cmd/remarquee rmdoc render-v6 /tmp/claude-code-v3-cloud/claude-code-v3.md.rmdoc --out /tmp/claude-code-v3-fixed.pdf --force
```

### Cloud-path rerender

```bash
go run ./cmd/remarquee rmdoc render-v6 --cloud "/Articles/claude-code-v3.md" --out /tmp/claude-code-v3-cloud-fixed.pdf --force --non-interactive
```

Expected outcome after the fix:

- page-3 annotation near the top of the page lands back over the "programming language" line instead of a later paragraph

## Debugging Checklist For Similar Future Bugs

When an annotation-placement report comes in, use this checklist.

### First questions

- Is the problem local-only, cloud-only, or both?
- Is the background page correct?
- Are the annotations missing, malformed, or just shifted?
- Is the shift horizontal, vertical, or both?
- Does it affect all pages or only certain pages?
- Does it affect PDF-backed docs, notebook docs, or both?

### Isolation steps

- rerun the render against a local `.rmdoc`
- inspect the archive with `rmdoc inspect`
- rasterize the bad page to PNG
- determine whether the bug is:
  - page selection
  - page ordering
  - stroke extraction
  - bbox computation
  - coordinate conversion
  - final merge placement

### Things to suspect immediately in layout bugs

- top-origin vs bottom-origin confusion
- unscaled vs scaled dimensions
- width/height mismatch after rotation
- using a page-space coordinate in an overlay-space calculation
- using a background-space coordinate in a PDF-space calculation

## Risks And Follow-Up Work

This fix is intentionally narrow and correct for the reported bug, but there are still adjacent risks worth noting.

### Residual risks

- other placement formulas copied from `remarks` may also need explicit coordinate conversion audits
- rotation-heavy documents should be checked carefully because coordinate transforms become more subtle there
- smart highlight placement is a different path from stroke placement and should continue to be spot-checked visually

### Useful follow-up tests

- add a fixture-based visual regression for a PDF-backed V6 doc with obvious top-of-page annotations
- add a test that explicitly exercises the `hSvg < hBg` branch in a more integrated way
- add a note in the renderer code explaining that the horizontal math was copied semantically from `remarks`, but the vertical math must be converted for PDF space

## Short Version For Reviewers

If you only need the essence:

- the bug was a Y-origin mismatch
- `remarks`-style top-origin placement was used directly in bottom-origin PDF coordinates
- that pushed overlays down on PDF-backed pages
- the fix converts top-origin Y placement into bottom-origin PDF Y placement with:

```go
pageHeight - topY - objectHeight
```

- the fix was applied in both full-document and selected-pages V6 merge paths
- focused tests and rerenders confirmed the placement is now correct

## References

- CLI entrypoint: [`cmd/remarquee/cmds/rmdoc/render_v6.go`](../../../../../cmd/remarquee/cmds/rmdoc/render_v6.go)
- Archive opening: [`pkg/rmdoc/open.go`](../../../../../pkg/rmdoc/open.go)
- Background assembly: [`pkg/rmdoc/render/background.go`](../../../../../pkg/rmdoc/render/background.go)
- V6 merge algorithm: [`pkg/rmdoc/render/v6_merge_background.go`](../../../../../pkg/rmdoc/render/v6_merge_background.go)
- Regression test: [`pkg/rmdoc/render/v6_merge_background_test.go`](../../../../../pkg/rmdoc/render/v6_merge_background_test.go)
