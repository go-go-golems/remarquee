---
Title: 'Design: Go rmdoc data model and APIs'
Ticket: RMQ-0004
Status: active
Topics:
    - backend
    - go
    - remarkable
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarks/remarks/remarks.py
      Note: PDF merge algorithm reference
    - Path: remarks/remarks/utils.py
      Note: cPages parsing + redirection map reference
    - Path: remarquee/cmd/remarquee/cmds/rmdoc/inspect.go
      Note: New CLI command to inspect schema/page plan (commit c804b36)
    - Path: remarquee/cmd/remarquee/cmds/rmdoc/root.go
      Note: New CLI command group for .rmdoc (commit c804b36)
    - Path: remarquee/cmd/remarquee/main.go
      Note: Wires remarquee rmdoc into the root CLI (commit c804b36)
    - Path: remarquee/pkg/rmdoc/content.go
      Note: Implemented .content parsing (legacy + cPages) and PageRef plan (commit 49acbde)
    - Path: remarquee/pkg/rmdoc/content_test.go
      Note: Unit tests for schema detection + page plan (commit 49acbde)
    - Path: remarquee/pkg/rmdoc/open.go
      Note: Implemented .rmdoc zip open + extraction (commit 49acbde)
    - Path: remarquee/pkg/rmdoc/pagedata.go
      Note: Implemented pagedata template application (commit 49acbde)
    - Path: remarquee/pkg/rmdoc/types.go
      Note: Implemented Document/PageRef types (commit 49acbde)
    - Path: remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/scripts/color_map.go
      Note: Highlight color map extracted from rmscene
    - Path: rmapi/annotations/pdf.go
      Note: V3/V5 stroke-to-PDF renderer (unipdf)
    - Path: rmapi/archive/reader.go
      Note: Legacy .rmdoc parsing logic to reuse/adapt
    - Path: rmc/src/rmc/exporters/svg.py
      Note: Coordinate constants + bounding box logic
    - Path: rmscene/src/rmscene/tagged_block_reader.py
      Note: V6 tagged-block parser reference
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-14T20:59:20.835901819-05:00
---



# Design: Go rmdoc data model and APIs

## Executive Summary

This design proposes a Go-first parsing + rendering stack for reMarkable `.rmdoc` archives with **dual-format support**:

- **Legacy** (V3/V5 `.rm`, legacy `.content`) — commonly used by older PDF-based documents on-device.
- **Modern** (V6 `.rm`, `cPages` `.content`) — used by notebooks and many newer documents.

The core outcome is a small set of packages under `remarquee/` that expose:

- A **container API** for opening and inspecting `.rmdoc` (zip layout, `.content`, `.metadata`, `.pagedata`, payload PDFs).
- A **page plan** API that deterministically maps UI pages to background PDF pages (including inserted/duplicated pages).
- A **renderer API** that produces a merged PDF (and optionally PNGs later), reusing rmapi for V3/V5 and implementing V6 support.

I include multiple proposals for the *data model* and recommend an approach that keeps **raw format fidelity** while providing a **stable normalized surface** for the rest of remarquee.

## Problem Statement

We want to port the parsing and rendering logic that currently lives across:

- `rmapi` (Go, legacy-only: legacy `.content`, V3/V5 `.rm`)
- `remarks` (Python, modern-only: `cPages`, V6 `.rm` via `rmscene` + `rmc`)

…into a coherent Go implementation inside `remarquee/` so that:

- The remarquee toolchain can be **deployed without a Python runtime** (long term goal).
- We can **reuse** the already-good V3/V5 pipeline from rmapi.
- We can support both formats that exist on the device today (confirmed empirically).

The design must answer:

- How do we represent `.rmdoc` documents (container + pages) in Go?
- Where do we draw the boundaries between **archive parsing**, **annotation parsing**, and **rendering**?
- How do we provide APIs that are stable for callers while keeping room for format evolution?

## Proposed Solution

### High-level architecture

Split the problem into three layers:

1. **Archive/container layer**: open `.rmdoc`, read `.content`, `.metadata`, `.pagedata`, payload files, and enumerate `.rm` pages.
2. **Annotation decode layer**: decode `.rm` pages into either:
   - V3/V5 strokes (`rmapi/encoding/rm`), or
   - V6 scene tree (port of `rmscene` concepts).
3. **Rendering layer**: produce a PDF by combining:
   - background PDF/template pages, and
   - rendered annotations (strokes + optional highlights/text).

### Proposed packages (inside `remarquee/`)

#### `remarquee/pkg/rmdoc`

Responsibility: `.rmdoc` ZIP parsing and document-level metadata.

**Status (implemented):** The initial version of this layer exists as `remarquee/pkg/rmdoc` (commit `49acbde`). It provides:
- zip opening (`OpenFile`, `OpenReaderAt`)
- `.content` schema detection (`cPages` vs legacy)
- deterministic `[]PageRef` page plan generation
- `.pagedata` template application helper
- unit tests for legacy + cPages parsing

Key types:

```go
package rmdoc

type ArchiveSchema int

const (
	SchemaUnknown ArchiveSchema = iota
	SchemaLegacy            // legacy .content: pages/pageCount/redirectionPageMap
	SchemaCPages            // modern .content: cPages.pages[]
)

type DocumentType int

const (
	DocTypeUnknown DocumentType = iota
	DocTypeNotebook
	DocTypePDF
)

type Document struct {
	UUID      string
	Schema    ArchiveSchema
	DocType   DocumentType

	// Raw JSON is kept for forward-compat + debugging.
	ContentJSON  []byte
	MetadataJSON []byte

	// Optional payload (e.g. original PDF).
	PayloadPDF []byte

	// Page plan (ordered UI pages + mapping to backgrounds).
	Pages []PageRef
}

type PageRef struct {
	// UI page order (0-based).
	Index int

	// Page ID used by the archive (UUID for cPages; uuid or numeric for legacy).
	PageID string

	// Background mapping.
	// For PDFs: SourcePDFPage is either a page index, or -1 for inserted pages.
	SourcePDFPage int

	// Template name if present (especially for notebooks/inserted pages).
	Template string

	// Some schemas mark pages as deleted.
	Deleted bool
}
```

Key functions:

```go
func OpenFile(ctx context.Context, path string, opts ...OpenOption) (*Document, error)
func OpenReaderAt(ctx context.Context, r io.ReaderAt, size int64, opts ...OpenOption) (*Document, error)

// Optional low-level accessors for callers that need raw files.
func (d *Document) ReadRM(ctx context.Context, pageID string) ([]byte, error)
```

Notes:
- `Open*` should be **cheap by default** (parse JSON, build page plan, but don’t eagerly decode all `.rm` pages).
- Keep raw JSON bytes for future schema drift.

#### `remarquee/pkg/rmdoc/content`

Responsibility: parse `.content` into typed structs and compute a deterministic `[]PageRef`.

**Status (implemented):** This lives in `remarquee/pkg/rmdoc/content.go` (commit `49acbde`) and is exercised by `remarquee/pkg/rmdoc/content_test.go`.

Core idea: parse both schemas, then build a single page plan:

- **Legacy**: use `pages[]` and/or `pageCount`, plus `redirectionPageMap` if present.
- **cPages**: use `cPages.pages[]` filtered by `deleted.value == 1` and `redir.value` mapping; inserted pages are `redir` missing (or `-1` depending on interpretation).

Implementation detail for cPages “timestamp/value” objects:

```go
type Value[T any] struct {
	Timestamp string `json:"timestamp,omitempty"`
	Value     T      `json:"value,omitempty"`
}

type CPages struct {
	Pages []CPage `json:"pages"`
	// lastOpened/original/etc optional
}

type CPage struct {
	ID       string         `json:"id"`
	Redir    *Value[int]    `json:"redir,omitempty"`
	Deleted  *Value[int]    `json:"deleted,omitempty"`  // observed as 0/1
	Template *Value[string] `json:"template,omitempty"`
	Idx      *Value[string] `json:"idx,omitempty"`
}
```

#### `remarquee/pkg/rmdoc/anno`

Responsibility: represent decoded page annotations behind a versioned interface.

Minimum common surface that renderers can use:

```go
type RMVersion int

const (
	RMUnknown RMVersion = iota
	RMV3
	RMV5
	RMV6
)

type PageAnnotations interface {
	Version() RMVersion
	// Extract primitives needed for rendering.
	Strokes() ([]Stroke, error)
	Highlights() ([]Highlight, error) // optional initially
	TextBlocks() ([]TextBlock, error) // optional initially
}
```

We then provide two concrete implementations:

- `anno/v3v5`: wraps `rmapi/encoding/rm` types.
- `anno/v6`: scene-tree based implementation (Go port).

#### `remarquee/pkg/rmdoc/render`

Responsibility: render merged PDFs (and later PNGs).

Key APIs:

```go
type RenderOptions struct {
	AddPageNumbers  bool
	AnnotationsOnly bool
	SmartHighlights bool
}

type DocumentRenderer interface {
	Render(ctx context.Context, doc *rmdoc.Document, out io.Writer, opts RenderOptions) error
}
```

Implementation sketch:
- Build (or open) the background PDF:
  - for PDF docs: start from payload PDF, then insert blank pages for inserted pages (per page plan)
  - for notebooks: start from blank doc, page sizes driven by constants (445x594pt) initially
- For each UI page:
  - decode `.rm` version (V3/V5 vs V6)
  - render strokes into an “overlay” content stream or overlay PDF page
  - merge overlay onto background page (matching the known merge math from `remarks`)
  - optionally add PDF highlight annotations using the V6 `HARDCODED_COLORMAP` mapping

### Rendering strategy for V6

We should *not* replicate `rmc`’s SVG → CairoSVG pipeline in Go. Instead:

- Reuse the existing **unipdf-based stroke rendering approach** from `rmapi/annotations/pdf.go`.
- Convert V6 scene items into the same internal stroke primitives used by the V3/V5 renderer.

This keeps the “brush math” in one place and avoids dragging SVG/Pango/Cairo dependencies into Go.

## Current implementation status (as of this design doc)

### Implemented

- `remarquee/pkg/rmdoc` (commit `49acbde`)
  - `.rmdoc` open (zip) + reads `.content/.metadata/.pagedata/.pdf`
  - schema detection: legacy vs `cPages`
  - deterministic page plan (`[]PageRef`) for both schemas
  - `.pagedata` template fill helper
  - unit tests

### Not implemented yet (next milestones)

- A debug CLI command to print the schema + page plan for a local `.rmdoc`
- Legacy PDF render path wired through the new `pkg/rmdoc` Document/PageRef model
- V6 `.rm` parser (scene tree) and V6 rendering + merge/highlights

## Design Decisions

### Decision: dual-format support is required

Empirical check of actual tablet documents shows:
- Notebooks often use `cPages` + V6 `.rm`
- Older PDFs can be legacy `.content` + V3/V5 `.rm`

Therefore, the design must route per-document/per-page based on **format detection**, not assumptions.

### Decision: keep raw JSON alongside typed structs

The `.content` schema evolves. Keeping `ContentJSON` (and `MetadataJSON`) enables:
- debugging against real devices,
- forward compatibility (unknown keys don’t break parsing),
- round-tripping / future migrations if needed.

### Decision: lazy page decoding

Decoding V6 pages can be expensive. The container API should:
- parse the archive and build a page plan up front,
- only decode `.rm` bytes when the caller asks to render/extract.

### Decision: deterministic page ordering

The `remarks` Python implementation iterates `.rm` files via a `set(...)`, which is nondeterministic. We should:
- define the canonical UI order from `.content` (`pages[]` or `cPages.pages[]`),
- then look up `.rm` files for those page IDs.

## Alternatives Considered

### Proposal A: fully normalized in-memory document model (recommended only later)

Parse everything into a single “canonical” Go model (pages, layers, strokes, highlights, text).

- **Pros**: simplest for downstream features (OCR, export, search); one model to rule them all.
- **Cons**: expensive (memory + CPU); easy to lose fidelity; harder incremental rollout.

### Proposal B: format-specific parse model + thin normalized adapter (recommended)

Keep “truth” close to the wire formats:
- legacy content + V3/V5 parser (reuse rmapi),
- cPages + V6 scene model (new),
and provide a small common interface for rendering and extraction.

- **Pros**: incremental; easier correctness; reuse existing working code; supports dual-format naturally.
- **Cons**: some duplication in renderers until convergence on shared stroke primitives.

### Proposal C: call Python for V6 (transitional option only)

Use `remarks` in a subprocess for V6 pages, while Go handles V3/V5. This can be useful for:
- early prototyping, or
- validating Go output against the Python “oracle”.

But it’s not the end-state: Python runtime + external dependencies make deployment harder.

## Implementation Plan

1. **Archive + format detection**
   - Implement `.rmdoc` opening (zip reader)
   - Parse `.content` as raw JSON + detect schema (`cPages` vs legacy keys)
2. **Page plan construction**
   - Legacy page list + redirection map (reuse logic from rmapi)
   - cPages page list + redirection map (port `remarks.utils.get_pages_data`)
3. **V3/V5 rendering path**
   - Wrap rmapi’s V3/V5 parser/renderer behind the new interfaces
4. **V6 parsing**
   - Port `rmscene` block reader + scene tree builder into Go
   - Start with strokes; add highlights/text incrementally
5. **V6 rendering**
   - Convert V6 strokes into shared `Stroke` primitives
   - Render strokes to PDF via unipdf
6. **Merge and highlight**
   - Implement merge math (background + overlay positioning)
   - Apply smart highlights as PDF annotations (reuse extracted `PenColor` mapping)
7. **Testing**
   - Golden tests with fixtures from reMarkable cloud:
     - at least one legacy PDF doc (V3/V5)
     - at least one notebook (V6)
   - Compare output PDFs visually / via rasterized diffs where possible

## Open Questions

- What is the full set of V6 block types we must support to avoid “Some data has not been read” warnings?
- How close do we need to match `rmc` brush rendering (pressure/width) for V6 on day 1?
- How should we handle notebook templates (`.pagedata` / `template.value`) in the Go renderer?
- Do we need EPUB support, or can we explicitly scope it out?

## References

### Primary research (RMQ-0001)

- `remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/analysis/01-deep-dive-rmdoc-format-container-layout-parsing-png-rendering.md`
- `remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/reference/06-diary-rmdoc-format-analysis-and-go-reimplementation-prep.md`
- `remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/scripts/go_reimplementation_gaps.md`
- `remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/scripts/color_map.go`

### Code references

- `rmapi/archive/reader.go` (legacy archive parsing)
- `rmapi/annotations/pdf.go` (V3/V5 PDF rendering)
- `remarks/remarks/utils.py` (cPages parsing + redirection map)
- `rmc/src/rmc/exporters/svg.py` (coordinate constants, bbox)
- `rmscene/src/rmscene/tagged_block_reader.py` (V6 block parsing)
