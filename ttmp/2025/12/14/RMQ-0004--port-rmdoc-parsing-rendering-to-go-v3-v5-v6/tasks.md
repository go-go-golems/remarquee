# Tasks

## TODO

### 0) Scope + success criteria

- [x] Confirm scope + acceptance criteria:
- [x] Required outputs: PDF only vs PDF+PNG
- [x] Fidelity: pixel-perfect vs “good enough” (and what “good enough” means: strokes only? highlights? typed text?)
- [x] Supported inputs: `.rmdoc` only, or also bare folders/unpacked exports?
- [x] Performance constraints: acceptable runtime per page / per document
- [x] Validation workflow: define how we manually verify output (interactive UI vs scripts), where feedback is stored

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

- [x] Implement legacy pipeline (V3/V5):
- [x] Decide integration approach:
- [x] Option A: call rmapi renderer directly (wrap/adapter)
- [x] Option B: reimplement minimal legacy renderer in `remarquee` (less coupling)
- [x] Implement adapter so we can render V3/V5 `.rm` to PDF pages
- [x] Ensure legacy page order follows `pkg/rmdoc.Document.Pages` (not file iteration)
- [x] Handle background PDF payload merge for PDF documents

- [x] Add CLI prototype for legacy rendering (rmapi-backed):
  - [x] `remarquee rmdoc render-legacy <file>` writes `<file>-annotations.pdf` (or `--out`)
  - [x] Guardrail: refuse cPages archives (V6) for now

### 3) `cPages`-aware PDF background construction (V6 + PDFs)

- [x] Implement background PDF assembly based on `PageRef.SourcePDFPage`:
  - [x] Insert blank pages for inserted pages (`SourcePDFPage == -1`)
  - [x] Duplicate pages for “duplicate page” cases (when needed)
- [x] Define template-to-page-size/template rendering strategy (initially: blank page size constants)

### 4) V6 `.rm` parsing (PORT rmscene concepts)

- [x] Implement V6 parser (scene tree) with a minimal stroke-only milestone:
- [x] Tagged block reader (header + main blocks + subblocks)
- [x] CRDT sequence decoding needed for scene items
- [x] Build scene tree (groups + lines)
- [x] Expose strokes as normalized primitives for rendering
- [x] Expand incrementally:
- [x] Highlights (GlyphRange rectangles + PenColor)
- [x] Typed text (RootTextBlock)

### 5) V6 rendering + merge algorithm

- [x] Implement V6 stroke rendering to PDF:
- [x] Convert V6 line points to renderer stroke primitives
- [x] Apply coordinate transforms (SCALE, X_SHIFT, etc.)
- [x] Compute bounding boxes (including anchor offsets for text-linked groups)

- [x] Implement PDF merge algorithm (background + annotation overlay):
- [x] Match the `remarks` positioning logic (w_svg vs w_bg cases + `highlights_x_translation`)
- [x] Handle page rotation edge cases

### 6) Smart highlights

- [x] Implement smart highlights for V6:
- [x] Use `PenColor` mapping to RGB (from RMQ-0001 `scripts/color_map.go`)
- [x] Create PDF highlight annotations and position them using `x_translation`

### 7) Fixtures + golden tests (REAL DOCUMENTS)

- [x] Add fixtures and golden tests:
- [x] At least one **legacy PDF** `.rmdoc` from the device (V3/V5)
- [x] At least one **notebook** `.rmdoc` from the device (V6)
- [x] Add a reproducible test runner that renders to PDF and compares outputs (visual diff or raster diff)

### 8) Wire into remarquee CLI

- [x] Add a user-facing command (name TBD) that:
- [x] Takes a local `.rmdoc` (or downloads via `remarquee cloud get`)
- [x] Emits an annotated PDF (and later PNGs)
- [x] Prints clear diagnostics for unsupported formats/features

### 9) Interactive validation UI (human feedback loop)

- [x] Implement `remarquee-ui` (interactive validation tool) per design doc:
- [x] Design doc: `design-doc/02-design-interactive-rmdoc-render-validation-ui.md`
- [x] Phase 0: skeleton + choose input `.rmdoc` path
- [x] Phase 1: run Inspect / Build Background / Render Legacy actions
- [x] Phase 2: capture PASS/FAIL + notes and persist “validation sessions” in ticket

