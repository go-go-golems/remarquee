# Tasks

## TODO

### 0) Scope + success criteria

- [ ] Confirm scope + acceptance criteria:
  - [ ] Required outputs: PDF only vs PDF+PNG
  - [ ] Fidelity: pixel-perfect vs “good enough” (and what “good enough” means: strokes only? highlights? typed text?)
  - [ ] Supported inputs: `.rmdoc` only, or also bare folders/unpacked exports?
  - [ ] Performance constraints: acceptable runtime per page / per document

### 1) `.rmdoc` container + page plan (FOUNDATION)

- [x] Implement `.rmdoc` open + schema detection + deterministic `[]PageRef` page plan (legacy + `cPages`)
  - [x] Add `pkg/rmdoc.OpenFile/OpenReaderAt` (zip open + read `.content/.metadata/.pagedata/.pdf`)
  - [x] Add `pkg/rmdoc.ParseContent`:
    - [x] Detect `cPages` vs legacy
    - [x] Build UI-ordered `[]PageRef` and filter deleted pages
    - [x] Map `redir.value` → `PageRef.SourcePDFPage` and treat missing redir as inserted page
  - [x] Add `pkg/rmdoc.ApplyPagedataTemplates` to fill missing templates from `.pagedata`
  - [x] Unit tests for legacy + cPages parsing/page plan

- [x] Add a tiny debug CLI command to inspect a local `.rmdoc`:
  - [x] Print detected schema + doc type
  - [x] Print page plan table (Index, PageID, SourcePDFPage, Template)

### 2) Legacy (V3/V5) rendering pipeline (REUSE rmapi)

- [ ] Implement legacy pipeline (V3/V5):
  - [ ] Decide integration approach:
    - [ ] Option A: call rmapi renderer directly (wrap/adapter)
    - [ ] Option B: reimplement minimal legacy renderer in `remarquee` (less coupling)
  - [ ] Implement adapter so we can render V3/V5 `.rm` to PDF pages
  - [ ] Ensure legacy page order follows `pkg/rmdoc.Document.Pages` (not file iteration)
  - [ ] Handle background PDF payload merge for PDF documents

- [x] Add CLI prototype for legacy rendering (rmapi-backed):
  - [x] `remarquee rmdoc render-legacy <file>` writes `<file>-annotations.pdf` (or `--out`)
  - [x] Guardrail: refuse cPages archives (V6) for now

### 3) `cPages`-aware PDF background construction (V6 + PDFs)

- [ ] Implement background PDF assembly based on `PageRef.SourcePDFPage`:
  - [ ] Insert blank pages for inserted pages (`SourcePDFPage == -1`)
  - [ ] Duplicate pages for “duplicate page” cases (when needed)
  - [ ] Define template-to-page-size/template rendering strategy (initially: blank page size constants)

### 4) V6 `.rm` parsing (PORT rmscene concepts)

- [ ] Implement V6 parser (scene tree) with a minimal stroke-only milestone:
  - [ ] Tagged block reader (header + main blocks + subblocks)
  - [ ] CRDT sequence decoding needed for scene items
  - [ ] Build scene tree (groups + lines)
  - [ ] Expose strokes as normalized primitives for rendering
  - [ ] Expand incrementally:
    - [ ] Highlights (GlyphRange rectangles + PenColor)
    - [ ] Typed text (RootTextBlock)

### 5) V6 rendering + merge algorithm

- [ ] Implement V6 stroke rendering to PDF:
  - [ ] Convert V6 line points to renderer stroke primitives
  - [ ] Apply coordinate transforms (SCALE, X_SHIFT, etc.)
  - [ ] Compute bounding boxes (including anchor offsets for text-linked groups)

- [ ] Implement PDF merge algorithm (background + annotation overlay):
  - [ ] Match the `remarks` positioning logic (w_svg vs w_bg cases + `highlights_x_translation`)
  - [ ] Handle page rotation edge cases

### 6) Smart highlights

- [ ] Implement smart highlights for V6:
  - [ ] Use `PenColor` mapping to RGB (from RMQ-0001 `scripts/color_map.go`)
  - [ ] Create PDF highlight annotations and position them using `x_translation`

### 7) Fixtures + golden tests (REAL DOCUMENTS)

- [ ] Add fixtures and golden tests:
  - [ ] At least one **legacy PDF** `.rmdoc` from the device (V3/V5)
  - [ ] At least one **notebook** `.rmdoc` from the device (V6)
  - [ ] Add a reproducible test runner that renders to PDF and compares outputs (visual diff or raster diff)

### 8) Wire into remarquee CLI

- [ ] Add a user-facing command (name TBD) that:
  - [ ] Takes a local `.rmdoc` (or downloads via `remarquee cloud get`)
  - [ ] Emits an annotated PDF (and later PNGs)
  - [ ] Prints clear diagnostics for unsupported formats/features

