---
Title: 'Diary: .rmdoc format analysis and Go reimplementation prep'
Ticket: RMQ-0001
Status: active
Topics:
    - backend
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarks/remarks/conversion/parsing.py
      Note: V6 .rm file parsing
    - Path: remarks/remarks/remarks.py
      Note: Main remarks processing logic (process_document, run_remarks)
    - Path: remarks/remarks/utils.py
      Note: Page ordering and redirection map algorithms
    - Path: remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/scripts/analyze_remarks_merge.py
      Note: Analysis script documenting the PDF merge algorithm from remarks
    - Path: remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/scripts/extract_rmc_coords.py
      Note: Analysis script to extract coordinate transform constants from rmc library
    - Path: remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/scripts/go_reimplementation_gaps.md
      Note: Updated gap analysis with dual-format support requirement (commit 76e1249)
    - Path: remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/scripts/trace_rm_parse.py
      Note: Fixed to handle CrdtSequence properly (commit 76e1249)
ExternalSources: []
Summary: Step-by-step diary of analyzing the .rmdoc format and remarks algorithms to prepare for Go reimplementation
LastUpdated: 2025-12-15T00:45:00-05:00
---


# Diary: .rmdoc format analysis and Go reimplementation prep

## Goal

Document the analysis process for understanding the `.rmdoc` container format and the `remarks` package algorithms in enough detail to reimplement them in Go for the remarquee unified tool.

## Step 1: Inspect real .rmdoc fixtures

Started by examining what's actually inside `.rmdoc` files using the test fixtures in this repo.

### What I did

- Listed ZIP entries in `remarks/tests/in/copies of different pages.rmdoc` (modern format)
- Listed ZIP entries in `rmapi/archive/test.zip` (legacy format)
- Extracted and pretty-printed `.content` and `.metadata` JSON from both

### What I learned

**Key discovery: there are two `.content` schema variants.**

1. **Legacy format** (rmapi/archive/test.zip):
   - Page files named numerically: `0.rm`, `1.rm`
   - `.content` has `pageCount` but no explicit page mapping
   - No `cPages` object

2. **Modern format** (remarks fixture):
   - Page files named by UUID: `<pageUUID>.rm`
   - `.content` has `cPages.pages[]` array with:
     - `id`: page UUID (matches .rm filename)
     - `redir.value`: source PDF page index (or absent for inserted pages)
     - `deleted.value`: soft-delete flag

**Why this matters for Go:**
- `rmapi/archive` currently models the legacy format in `archive.Content`
- Modern exports require `cPages` parsing to map UUID filenames to page order
- Without this, you can't correctly order pages or handle inserted/duplicated pages

### Technical details

Modern `.content` example (abridged):
```json
{
  "fileType": "pdf",
  "cPages": {
    "pages": [
      {"id": "ce65b731-...", "redir": {"value": 0}},
      {"id": "b6560eba-...", "redir": {"value": 0}},
      {"id": "c3666c1b-..."},  // No redir = inserted page
      {"id": "ed25fb03-..."}
    ]
  }
}
```

Legacy `.content` example:
```json
{
  "fileType": "",
  "pageCount": 1
}
```

## Step 2: Extract coordinate transformation constants from rmc

Needed the exact constants and formulas `remarks` uses to convert reMarkable device coordinates to PDF coordinates.

### What I did

- Created `scripts/extract_rmc_coords.py` to introspect the `rmc` library
- Ran it via `poetry run python` in the remarks project
- Extracted `SCALE`, `SCREEN_WIDTH`, `SCREEN_HEIGHT`, `X_SHIFT`, and the `xx()`/`yy()` functions

### What I learned

**Key constants:**
```python
SCALE = 0.3185840707964602   # = 445/1404 (PDF export width / device width)
SCREEN_WIDTH = 1404          # Device width in pixels
SCREEN_HEIGHT = 1872         # Device height in pixels
X_SHIFT = 223.0              # ≈ SCREEN_WIDTH * SCALE / 2
```

**Coordinate transform (trivially simple!):**
```python
def xx(device_x): return device_x * SCALE
def yy(device_y): return device_y * SCALE
```

**ReMarkable coordinate system:**
- Origin: CENTER-TOP (not corner!)
- X: -702 to +702 (center = 0)
- Y: 0 to 1872 (top = 0)

After scaling:
- X: -223.65 to +223.65
- Y: 0 to 596.39

### What was tricky

The functions `xx` and `yy` in `rmc` are actually aliases for the same `scale()` function. I expected them to be different (maybe Y-flip for PDF bottom-left origin), but the Y-flip happens elsewhere in the pipeline.

## Step 3: Document the PDF merge algorithm

The merge algorithm is the core of `remarks` - it overlays annotation PDFs onto background PDFs.

### What I did

- Created `scripts/analyze_remarks_merge.py` with step-by-step pseudocode
- Traced through `remarks/remarks/remarks.py:process_document()`
- Documented the positioning logic for when annotations are wider/narrower than background

### What I learned

**The 7-step merge process:**

1. **Convert .rm to PDF**: `rm_to_pdf(rmFile, tempPDF)` renders strokes
2. **Check background content**: `page.get_contents()` determines if merge needed
3. **Compute bounding box**: Walk scene tree, apply anchor offsets
4. **Compute canvas size**: `max(annotationSize, backgroundSize)`
5. **Position both PDFs**: Center the smaller one within the larger
6. **Create merged page**: `show_pdf_page()` twice (background then annotations)
7. **Apply smart highlights**: Add PDF highlight annotations with `xTranslation`

**Critical insight - the `xTranslation` calculation:**
```python
if w_svg > w_bg:
    # Annotations wider than background
    x_bg = width/2 - w_bg/2 - (w_svg/2 + x_shift)
    highlights_x_translation = x_bg + w_bg/2
elif w_svg < w_bg:
    # Background wider than annotations
    x_svg = width/2 - w_svg/2 + (w_svg/2 + x_shift)
    highlights_x_translation = w_bg/2
```

This handles the center-top origin: highlights need to know where the PDF's center is.

### What warrants a second pair of eyes

- The positioning formulas are subtle. I've documented them but would appreciate verification against actual output PDFs.
- The rotation handling (`page.set_rotation(0)`, `rotate=-page_rotation`) needs testing with rotated source PDFs.

## Step 4: Update the deep-dive analysis doc

Added all algorithmic details to the analysis doc in Part II.

### What I did

- Added Section 10: Coordinate Transformation Constants
- Added Section 11: Page Ordering Algorithm
- Added Section 12: Background PDF Construction Algorithm
- Added Section 13: PDF Merge Algorithm (with Go pseudocode)
- Added Section 14: Smart Highlight Application
- Added Section 15: Go Reimplementation Checklist

### Technical details

The deep-dive doc now contains:
- Exact constants for Go reimplementation
- Pseudocode for all key algorithms
- Example coordinate transformations
- Checklist of what needs to be built

## What should be done next

1. **V6 .rm parser in Go**: The hardest part. Consider:
   - Porting `rmscene` (Python) to Go
   - Using `rmapi/encoding/rm` as a starting point (but it's V3/V5 only)
   - The scene tree block format is complex

2. **Test with real files**: Run the analysis scripts on more `.rmdoc` files to verify edge cases

3. **PDF library evaluation**: Decide between unipdf (already in rmapi), pdfcpu, or gofpdf for the merge operations

4. **Highlight color mapping**: Extract the full `HARDCODED_COLORMAP` from rmscene

## Step 5: Create intern guide and additional analysis tools

Created a comprehensive playbook for an intern to continue this research now that we have local clones of `rmc` and `rmscene`.

### What I did

- Created `playbook/02-intern-guide-continuing-rmdoc-format-and-algorithm-research.md`
- Added `scripts/trace_rm_parse.py` to demonstrate V6 parsing
- Added `scripts/extract_color_map.py` to generate Go color constants
- Generated `scripts/color_map.go` with ready-to-use Go code

### What I learned

**Repository structure:**
- `rmc/src/rmc/exporters/` contains the rendering logic (svg.py, pdf.py)
- `rmscene/src/rmscene/` contains the V6 parser (scene_items.py, tagged_block_reader.py)
- Both use poetry for dependency management
- The color mapping is in `rmscene.scene_items.HARDCODED_COLORMAP`

**Key insight**: `rmc` doesn't render directly to PDF — it goes `.rm` → SVG → PDF via CairoSVG. This is important for Go: we could follow the same pattern (render to SVG first) or render directly with a PDF library.

### What was tricky

The `SCALE` constant calculation is split across multiple lines:
```python
SCREEN_DPI = 226
SCALE = 72.0 / SCREEN_DPI  # = 0.3185840707964602
```

This is actually `72.0 / 226` (PDF points per inch / device DPI), not the `445/1404` I initially thought. Both give the same result, but the DPI-based formula is the "why" behind the magic number.

### What warrants a second pair of eyes

The intern guide assumes familiarity with Python, poetry, and basic binary formats. If the intern is junior, they may need additional guidance on:
- How to read Python dataclasses
- What CRDT sequences are (used in rmscene)
- How to interpret binary block structures

### Technical details

**Generated Go code** (`scripts/color_map.go`):
- 28 PenColor constants (BLACK through SHADER_CYAN)
- RGBA struct and color maps
- Helper function `PenColorToRGB()` for PDF annotations
- Ready to copy into a Go package

**Analysis scripts created:**
1. `trace_rm_parse.py` - Demonstrates V6 parsing, shows block types and element counts
2. `extract_color_map.py` - Generates Go color mapping code

## Step 6: Deep dive into rmc and rmscene libraries (following intern guide)

Following the intern guide playbook, I systematically explored the `rmc` and `rmscene` libraries to understand their internals and prepare for Go reimplementation.

### What I did

**Phase 1: Explored rmc library (coordinate transforms and rendering)**

1. **Verified coordinate transform constants**:
   - Confirmed `SCALE = 72.0 / SCREEN_DPI = 72.0 / 226 = 0.3185840707964602`
   - Verified `SCREEN_WIDTH = 1404`, `SCREEN_HEIGHT = 1872`
   - Confirmed `X_SHIFT = PAGE_WIDTH_PT // 2 = 223.0`
   - The DPI-based formula (`72.0 / 226`) is the "why" behind the magic number: PDF points per inch divided by device DPI

2. **Studied bounding box calculation** (`get_bounding_box()` in `svg.py`):
   - Recursively walks the scene tree
   - For `Group` items: applies anchor offsets (`anchor_x`, `anchor_y`) and recurses
   - For `Line` items: tracks min/max X/Y across all points
   - Returns `(x_min, x_max, y_min, y_max)` in device coordinates (not yet scaled)
   - Default bounding box is `(-SCREEN_WIDTH//2, SCREEN_WIDTH//2, 0, SCREEN_HEIGHT)` = `(-702, 702, 0, 1872)`
   - Key insight: Anchor offsets propagate through nested groups, affecting the final bounding box

3. **Studied PDF rendering pipeline** (`pdf.py`):
   - **Key discovery**: `rmc` doesn't render directly to PDF!
   - Pipeline: `.rm` → SVG (via `rm_to_svg`) → PDF (via `cairosvg.svg2pdf`)
   - This is important for Go: we could follow the same pattern (render to SVG first) or render directly with a PDF library like UniDoc

**Phase 2: Explored rmscene library (V6 binary parser)**

1. **Understood scene tree structure** (`scene_tree.py`):
   - `SceneTree` has `root` (a Group) and `root_text` (optional Text)
   - `build_tree(tree, blocks)` constructs the tree from parsed blocks
   - Tree is hierarchical: Root → Layers (Groups) → Lines/Text/Highlights
   - `walk()` method iterates through all leaf items (not groups)

2. **Studied scene item types** (`scene_items.py`):
   - `Group`: Represents layers, has `anchor_id`, `anchor_origin_x` for positioning
   - `Line`: Stroke data with `color`, `tool`, `points[]`, `thickness_scale`
   - `GlyphRange`: Highlight rectangles with text
   - `Text`: Typed text blocks
   - `PenColor` enum: 28 colors (BLACK through SHADER_CYAN)
   - `HARDCODED_COLORMAP`: Maps RGBA tuples to PenColor enum values
   - `Point` structure: `x`, `y`, `speed`, `direction`, `width`, `pressure`

3. **Understood binary block format** (`tagged_block_reader.py`):
   - `.rm` files are sequences of "tagged blocks"
   - Each block has a type tag (byte) and length-prefixed data
   - Block types include: `SceneLineItemBlock`, `SceneGlyphItemRangeBlock`, `RootTextBlock`, `SceneTreeBlock`, `TreeNodeBlock`, etc.
   - Reader uses a state machine to parse nested structures
   - **Warning**: This is the hardest part to port to Go. The binary format is complex with variable-length encoding, nested sub-blocks, and CRDT sequences.

4. **Traced complete parse** (using `trace_rm_parse.py`):
   - Tested on real `.rm` file from `remarks/tests/in/copies of different pages.rmdoc`
   - Found 34 blocks total
   - Block sequence: `AuthorIdsBlock` → `MigrationInfoBlock` → `PageInfoBlock` → `SceneInfo` → `SceneTreeBlock` → `TreeNodeBlock` (×2) → `SceneGroupItemBlock` → `SceneLineItemBlock` (×26)
   - Scene tree: 1 root child, 26 lines (strokes), 0 highlights
   - First line: BLACK color, BALLPOINT_2 tool, 62 points, coordinates X: -389 to -332, Y: 618 to 776 (device space)

**Phase 3: Analyzed PDF merge algorithm in detail**

Re-examined the merge positioning logic in `remarks.py`:

- When `w_svg > w_bg` (annotations wider than background):
  - Background gets shifted right: `x_bg = width/2 - w_bg/2 - (w_svg/2 + x_shift)`
  - Highlights translation: `highlights_x_translation = x_bg + w_bg/2`
  
- When `w_svg < w_bg` (background wider than annotations):
  - Annotations get centered: `x_svg = width/2 - w_svg/2 + (w_svg/2 + x_shift)`
  - Highlights translation: `highlights_x_translation = w_bg/2`

The formulas account for reMarkable's center-top origin: highlights need to know where the PDF's center is relative to the annotation PDF.

### What I learned

**Key insights:**

1. **Coordinate system understanding**:
   - reMarkable uses center-top origin (X: -702 to +702, Y: 0 to 1872)
   - After scaling: X: -223.65 to +223.65, Y: 0 to 596.39
   - The `X_SHIFT` constant (223.0) is used to center annotations horizontally

2. **Bounding box calculation complexity**:
   - Must recursively walk scene tree
   - Must apply anchor offsets at each Group level
   - Anchor offsets come from typed text positions (`build_anchor_pos`)
   - Special anchor IDs: `0xfffffffffffe` (top of page), `0xffffffffffff` (bottom of page)

3. **PDF rendering strategy**:
   - Current Python approach: `.rm` → SVG → PDF (via CairoSVG)
   - For Go: Could follow same pattern (render to SVG, convert to PDF) OR render directly with UniDoc
   - Direct rendering might be faster but requires implementing stroke rendering logic

4. **V6 parser complexity**:
   - Binary format uses tagged blocks with variable-length encoding
   - CRDT sequences for conflict-free replicated data types
   - Nested sub-blocks require careful state management
   - This will be the hardest part to port to Go

5. **Block structure**:
   - File header → AuthorIdsBlock → MigrationInfoBlock → PageInfoBlock → SceneInfo → SceneTreeBlock → TreeNodeBlocks → SceneItemBlocks
   - Each block type has specific structure and parsing requirements

### What was tricky

- Understanding how anchor offsets propagate through nested groups
- The relationship between typed text anchors and stroke positioning
- Why `xx` and `yy` are identical (I expected Y-flip for PDF bottom-left origin, but that happens elsewhere)

### What warrants a second pair of eyes

- The anchor offset calculation (`build_anchor_pos`) needs verification with documents that have typed text
- The PDF merge positioning formulas are subtle and should be tested with edge cases (very wide annotations, rotated pages)
- The binary block format parsing logic is complex and error-prone — consider using existing Go CRDT libraries if available

### Technical details

**Fixed `trace_rm_parse.py` script**:
- Handled `CrdtSequence` properly (can't use `len()`, must iterate)
- Shows block types, element counts, coordinate ranges
- Demonstrates V6 parsing workflow

**Key constants verified**:
```python
SCREEN_DPI = 226
SCALE = 72.0 / SCREEN_DPI = 0.3185840707964602
PAGE_WIDTH_PT = SCREEN_WIDTH * SCALE = 447.29
PAGE_HEIGHT_PT = SCREEN_HEIGHT * SCALE = 596.39
X_SHIFT = PAGE_WIDTH_PT // 2 = 223.0
```

## Step 7: Gap analysis for Go reimplementation

Created a comprehensive gap analysis document mapping what exists in rmapi vs what's needed from remarks/rmc/rmscene.

### What I did

**Phase 4: Mapped Go reimplementation requirements**

1. **Identified existing capabilities in rmapi**:
   - Checked `rmapi/encoding/rm/` - V3/V5 parser only (not V6)
   - Checked `rmapi/annotations/pdf.go` - PDF stroke rendering with UniDoc
   - Checked `rmapi/archive/` - ZIP handling and legacy `.content` parsing
   - Confirmed `archive.Content` struct does NOT have `cPages` field

2. **Created gap analysis document** (`scripts/go_reimplementation_gaps.md`):
   - Listed 8 major gaps: V6 parser, cPages parsing, coordinate transforms, bounding box, PDF merge, smart highlights, GlyphRange extraction, typed text
   - Analyzed 3 porting strategy options:
     - Option 1: Port rmscene to Go (High effort, full control)
     - Option 2: Call Python from Go (Low effort, Python dependency)
     - Option 3: Hybrid approach (Medium effort, partial Python dependency)
   - Recommended: Start with Option 2, migrate to Option 1
   - Detailed complexity analysis for each gap
   - Implementation priority order
   - Testing strategy

### What I learned

**Key findings:**

1. **rmapi limitations**:
   - Only handles legacy `.rmdoc` format (V3/V5 .rm files, numeric page names)
   - Cannot handle modern format (V6 .rm files, UUID page names, cPages)
   - This explains why `remarks` exists as a separate tool

2. **Gap complexity ranking**:
   - **Very High**: V6 parser (complex binary format, CRDT sequences)
   - **Medium-High**: PDF merge algorithm (positioning formulas)
   - **Medium**: Bounding box, smart highlights, GlyphRange, typed text
   - **Low-Medium**: cPages parsing (straightforward JSON)
   - **Low**: Coordinate transforms (just constants)

3. **Porting strategy rationale**:
   - Starting with Python wrapper (Option 2) allows:
     - Quick validation of approach
     - End-to-end testing with real documents
     - Performance benchmarking
     - Incremental migration path
   - Migrating to native Go (Option 1) provides:
     - No Python runtime dependency
     - Better performance
     - Easier deployment
     - Full control over implementation

4. **Implementation priority**:
   - Phase 1: cPages parsing (foundational, enables page ordering)
   - Phase 2: Coordinate transforms (foundational, enables rendering)
   - Phase 3: PDF merge algorithm (core functionality)
   - Phase 4: V6 parser OR Python wrapper (hardest part)
   - Phase 5-6: Smart highlights and typed text (nice-to-have)

### What warrants a second pair of eyes

- The porting strategy recommendation (Option 2 → Option 1) should be validated with stakeholders
- The complexity estimates are rough - actual implementation may reveal surprises
- The testing strategy needs concrete test cases and fixtures

### Technical details

**Gap analysis document structure**:
- What we have (rmapi capabilities)
- What we need (from Python libraries)
- Porting strategy options with pros/cons
- Detailed gap analysis (8 gaps with complexity ratings)
- Implementation priority order
- Testing strategy
- References to existing analysis docs

**Key insight**: The gap analysis reveals that rmapi and remarks solve different problems:
- rmapi: Legacy format support, basic PDF rendering
- remarks: Modern format support, advanced PDF merging, smart highlights

The unified tool needs to combine both capabilities.

## Step 8: Testing with real documents and V6-only constraint

Tested processing a real V6 document and confirmed the V6-only support constraint for remarquee.

### What I did

**Phase 5: Tested with real documents**

1. **Extracted and inspected test document** (`remarks/tests/in/copies of different pages.rmdoc`):
   - Document structure: Contains `.content`, `.metadata`, `.pagedata`, `.pdf`, and a UUID-named directory with `.rm` files
   - `.content` file: Has `cPages.pages[]` array with 4 pages, each with `id`, `redir.value`, `template.value`
   - `.metadata` file: Standard document metadata (name, timestamps, parent UUID)
   - `.rm` files: 4 files named by UUID (not numeric), sizes range from 7KB to 17KB

2. **Processed document with remarks**:
   - Successfully generated PDF: `Copies of different pages _remarks.pdf` (36MB, 4 pages)
   - Generated Obsidian markdown: `Copies of different pages _obsidian.md` (292 bytes)
   - Some warnings: "Some data has not been read" for block type 13 (likely newer format features)
   - Processing order: Pages processed in order 2, 4, 3, 1 (matching `cPages.pages[]` order)

3. **Analyzed rmapi limitations**:
   - `archive/reader.go:readContent()` expects `Content.Pages []string` and `Content.PageCount`
   - For V6 documents with `cPages`, these fields are missing or empty
   - `readData()` expects `.rm` files to match UUIDs in `Pages` array, but V6 uses `cPages.pages[].id`
   - `readData()` calls `rm.New().UnmarshalBinary()` which only handles V3/V5 format
   - **Conclusion**: rmapi cannot process V6 documents at all

4. **Updated gap analysis**:
   - Added note: "remarquee will only support V6 format (modern .rmdoc files)"
   - Marked V3/V5 parser as "Not needed for V6-only support"
   - Clarified that V6 parser is CRITICAL (only format we need)
   - Clarified that cPages parsing is CRITICAL (required for modern .rmdoc)

### What I learned

**Key findings:**

1. **V6-only constraint simplifies implementation**:
   - Don't need to support legacy V3/V5 format
   - Don't need backward compatibility code
   - Can focus entirely on modern format
   - Simpler codebase, fewer edge cases

2. **rmapi is completely incompatible with V6**:
   - Content parsing fails (no `cPages` support)
   - Page ordering fails (relies on `Pages` array, not `cPages.pages[]`)
   - `.rm` file parsing fails (V3/V5 parser can't read V6)
   - **We need to build V6 support from scratch**

3. **remarks successfully processes V6**:
   - Handles `cPages` correctly
   - Processes UUID-named `.rm` files
   - Generates valid PDF output
   - Some format warnings but still works (graceful degradation)

4. **Document structure insights**:
   - V6 documents use UUID-based page naming: `<docUUID>/<pageUUID>.rm`
   - Page ordering comes from `cPages.pages[]` array order
   - `redir.value` maps logical pages to source PDF pages (for PDF documents)
   - `template.value` specifies background template (e.g., "Blank")

5. **Processing workflow**:
   - Extract `.rmdoc` ZIP
   - Parse `.content` to get `cPages.pages[]`
   - Order pages by array index
   - For each page: read `<pageUUID>.rm`, parse V6 format, render to PDF
   - Merge annotation PDFs with background PDF (if present)
   - Apply smart highlights
   - Generate final output

### What warrants a second pair of eyes

- The "Some data has not been read" warnings suggest rmscene may not fully support all V6 format features. Should we investigate what's missing?
- The 36MB output PDF seems large for 4 pages - is this expected or a compression issue?
- Need to verify that all V6 documents follow the same structure (UUID naming, cPages format)

### Technical details

**Document structure** (`copies of different pages.rmdoc`):
```
d4e3ce4f-564a-43ee-a54f-203043de37b4/
├── d4e3ce4f-564a-43ee-a54f-203043de37b4.content
├── d4e3ce4f-564a-43ee-a54f-203043de37b4.metadata
├── d4e3ce4f-564a-43ee-a54f-203043de37b4.pagedata
├── d4e3ce4f-564a-43ee-a54f-203043de37b4.pdf
└── d4e3ce4f-564a-43ee-a54f-203043de37b4/
    ├── ce65b731-9fc5-469d-9d87-0672d21c06db.rm  (page 1)
    ├── b6560eba-9d29-4c6f-a638-c1f8b03f874a.rm  (page 2)
    ├── c3666c1b-63fc-4cba-8c6d-419c41e2d58b.rm  (page 3)
    └── ed25fb03-aebc-4ae9-abb8-42924a5e44f8.rm  (page 4)
```

**cPages structure**:
```json
{
  "cPages": {
    "pages": [
      {"id": "ce65b731-...", "redir": {"value": 0}, "template": {"value": "Blank"}},
      {"id": "b6560eba-...", "redir": {"value": 0}, "template": {"value": "Blank"}},
      {"id": "c3666c1b-...", "template": {"value": "Blank"}},  // No redir = inserted page
      {"id": "ed25fb03-...", "template": {"value": "Blank"}}
    ]
  }
}
```

**rmapi incompatibility points**:
1. `readContent()`: Expects `Content.Pages []string`, V6 has `cPages.pages[]`
2. `readContent()`: Expects `Content.PageCount int`, V6 uses `len(cPages.pages)`
3. `readData()`: Expects numeric `.rm` filenames or UUIDs matching `Pages` array
4. `readData()`: Calls `rm.UnmarshalBinary()` which only handles V3/V5

## Step 9: Format compatibility check - V3/V5 vs V6

Checked actual documents on reMarkable tablet to determine if we need to support both V3/V5 and V6 formats.

**Commit:** 76e1249 — "RMQ-0001: Discover dual-format requirement (V3/V5 + V6)"

### What I did

1. **Downloaded and checked multiple documents**:
   - Used `remarquee cloud` commands to download documents
   - Checked `.content` files for format indicators
   - Examined `.rm` file naming conventions

2. **Format detection method**:
   - V6 format: Has `cPages` object in `.content`
   - V3/V5 format: Has `Pages` array and/or `pageCount` in `.content`
   - `.rm` file naming: V6 uses UUID names, V3/V5 uses numeric names (0.rm, 1.rm)

3. **Documents checked**:
   - **V6 documents**: Journal, TODO, Quick sheets, WORK TODO (all notebooks)
   - **V3/V5 documents**: "Artificial Intelligence Planning Systems..." (PDF-based document)

### What I learned

**Key findings:**

1. **Both formats are in active use**:
   - **V6**: Used by notebooks and newer documents
   - **V3/V5**: Used by PDF-based documents, especially older ones
   - The format depends on document type and age, not just device version

2. **Format indicators**:
   - **V6**: `.content` has `cPages.pages[]` array with UUID page IDs
   - **V3/V5**: `.content` has `Pages []string` array and `pageCount int`
   - Easy to detect: check for presence of `cPages` key

3. **rmapi compatibility**:
   - ✅ rmapi can handle V3/V5 documents (has parser and content struct)
   - ❌ rmapi cannot handle V6 documents (no cPages support, no V6 parser)
   - **We can reuse rmapi's V3/V5 support!**

4. **Implementation implications**:
   - Need format detection logic (check for `cPages` vs `Pages`)
   - Can reuse rmapi's V3/V5 parser and content parsing
   - Only need to build V6 support from scratch
   - Unified interface that routes to appropriate parser

### What warrants a second pair of eyes

- Should we verify if ALL PDF documents use V3/V5, or if newer PDFs might use V6?
- Need to test edge cases: documents with no annotations, documents with mixed formats
- Consider migration path: do V3/V5 documents get upgraded to V6 when edited?

### Technical details

**Format detection pseudocode**:
```go
func DetectFormat(content *Content) Format {
    if content.CPages != nil {
        return FormatV6
    }
    if len(content.Pages) > 0 || content.PageCount > 0 {
        return FormatV3V5
    }
    return FormatUnknown
}
```

**V3/V5 document example** (from "Artificial Intelligence Planning Systems..."):
```json
{
    "pageCount": 123,
    "Pages": ["uuid1", "uuid2", ...],
    "FileType": "pdf",
    ...
}
```

**V6 document example** (from Journal, TODO):
```json
{
    "cPages": {
        "pages": [
            {"id": "uuid1", "redir": {"value": 0}},
            {"id": "uuid2", "template": {"value": "Blank"}}
        ]
    },
    ...
}
```

**Updated gap analysis**:
- Changed from "V6-only support" to "Both V3/V5 and V6 support"
- Added format detection as Phase 1 priority
- Noted that rmapi's V3/V5 support can be reused
- Updated implementation priority to include format detection

## Related

- Analysis doc: `analysis/01-deep-dive-rmdoc-format-container-layout-parsing-png-rendering.md`
- Earlier remarks reference: `reference/02-remarks-package-analysis-parsing-conversion-output-formats.md`
- Intern guide: `playbook/02-intern-guide-continuing-rmdoc-format-and-algorithm-research.md`
- Gap analysis: `scripts/go_reimplementation_gaps.md` (updated with dual-format support requirement)
- Scripts: `scripts/extract_rmc_coords.py`, `scripts/analyze_remarks_merge.py`, `scripts/trace_rm_parse.py`, `scripts/extract_color_map.py`, `scripts/color_map.go`
