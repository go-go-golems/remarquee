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
    - Path: remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/scripts/extract_rmc_coords.py
      Note: Analysis script to extract coordinate transform constants from rmc library
    - Path: remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/scripts/analyze_remarks_merge.py
      Note: Analysis script documenting the PDF merge algorithm from remarks
    - Path: remarks/remarks/remarks.py
      Note: Main remarks processing logic (process_document, run_remarks)
    - Path: remarks/remarks/utils.py
      Note: Page ordering and redirection map algorithms
    - Path: remarks/remarks/conversion/parsing.py
      Note: V6 .rm file parsing
ExternalSources: []
Summary: "Step-by-step diary of analyzing the .rmdoc format and remarks algorithms to prepare for Go reimplementation"
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

## Related

- Analysis doc: `analysis/01-deep-dive-rmdoc-format-container-layout-parsing-png-rendering.md`
- Earlier remarks reference: `reference/02-remarks-package-analysis-parsing-conversion-output-formats.md`
- Scripts: `scripts/extract_rmc_coords.py`, `scripts/analyze_remarks_merge.py`
