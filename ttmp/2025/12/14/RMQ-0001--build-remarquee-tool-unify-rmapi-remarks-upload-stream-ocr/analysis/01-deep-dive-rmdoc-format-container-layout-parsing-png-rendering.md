---
Title: 'Deep dive: .rmdoc format (container layout, parsing, PNG rendering)'
Ticket: RMQ-0001
Status: active
Topics:
    - backend
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: 2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/scripts/analyze_remarks_merge.py
      Note: Documents the 7-step PDF merge algorithm from remarks
    - Path: 2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/scripts/extract_rmc_coords.py
      Note: Extracts coordinate transform constants (SCALE
ExternalSources: []
Summary: In-depth explanation of the .rmdoc container format, the two observed .content schema variants (legacy vs cPages), how this repo parses .rmdoc/.rm data (rmapi vs remarks), and how to reliably derive PNGs by rasterizing the merged PDF output.
LastUpdated: 2025-12-14T19:27:35.948556732-05:00
---


# Deep dive: .rmdoc format (container layout, parsing, PNG rendering)

## 1. Overview

An `.rmdoc` file is a **ZIP container** that bundles a reMarkable document’s *metadata*, *page ordering*, an optional *source payload* (PDF/EPUB), and the per-page **annotation files** (`.rm`). The hard part is not “unzipping” — it’s correctly mapping **page order** and **coordinates** so you can render handwriting (“paths”) and typed/highlight information (“notes”) into outputs like PDF and PNG.

This document explains (a) what’s inside `.rmdoc`, (b) how to parse it robustly across format variants, and (c) how this repo currently converts annotations into a merged PDF and then into PNGs.

## 2. What's inside a `.rmdoc` (ZIP layout)

The `.rmdoc` container is a standard ZIP archive whose internal structure follows a consistent naming convention centered around a document UUID. Understanding this layout is essential because each component plays a specific role in document reconstruction: metadata tells you *what* the document is called and where it lives, content tells you *how pages are ordered*, pagedata provides *visual styling*, and the `.rm` files contain the *actual annotations*. Without this mental model, it's easy to misinterpret which file controls page sequencing or where coordinate data lives.

To see the structure, list the ZIP entries of a modern fixture included in this repo:

```bash
unzip -l "remarks/tests/in/copies of different pages.rmdoc"
```

Expected output (abridged):

```text
d4e3ce4f-....content
d4e3ce4f-....metadata
d4e3ce4f-....pagedata
d4e3ce4f-....pdf
d4e3ce4f-..../<page-uuid-1>.rm
d4e3ce4f-..../<page-uuid-2>.rm
...
```

### File roles explained

- **Document UUID** (`d4e3ce4f-...`): The shared prefix that binds all files together. This UUID comes from the reMarkable device and is stable across syncs.
  
- **`<docUUID>.metadata`**: High-level document metadata — the user-visible title (`visibleName`), parent collection UUID (`parent`), and timestamps (`lastModified`, `createdTime`). This is where the document "lives" in the folder hierarchy.

- **`<docUUID>.content`**: The *page ordering* authority. It defines which pages exist, in what order, and how they map to source PDF pages (via redirection). Modern exports use a `cPages` object; older exports use simpler arrays. **This is the most version-sensitive file.**

- **`<docUUID>.pagedata`**: Background templates per page — one line per page, naming templates like `"Blank"`, `"Lined"`, `"Grid"`. Parsers typically use this for visual rendering but it's optional for extracting annotations.

- **`<docUUID>.pdf`** (optional): The original PDF file for PDF-backed documents. Notebooks don't have this; instead they're rendered from scratch using the dimensions of annotation content.

- **`<docUUID>/<pageUUID>.rm`**: Per-page annotation files containing handwriting strokes (V3/V5/V6 formats), and in V6 also typed text blocks and highlight geometry. These are binary files, not JSON.

- **`<docUUID>.highlights/...`** (optional): Additional highlight metadata in some exports. Note that `rmapi/archive` explicitly ignores this directory, relying instead on `.rm` file content.

## 3. The `.content` file: page ordering, redirection, and schema variants

The `.content` file is the single source of truth for page ordering and the primary compatibility challenge when parsing `.rmdoc` files. This JSON file defines which page UUIDs exist, in what sequence, and how they map back to the original PDF (via "redirection"). The challenge is that reMarkable has evolved this format over time: **older archives** use a simple `pages` array or rely on `pageCount` + numeric filenames, while **newer exports** introduce a `cPages` object with per-page insertion/duplication tracking. A parser that expects one schema will silently fail or produce incorrectly ordered output when encountering the other. This section explains both variants and shows how `remarks` and `rmapi` each handle them.

### 3.1. Legacy `.content` (rmapi/archive model)

`rmapi/archive` models `.content` as the `archive.Content` struct and uses:

- `pageCount` to size the page list when there is no explicit mapping.
- `pages` (if present) to map **UUID-named `.rm` files** to indices.
- `redirectionPageMap` to map “logical page index” → “source document page index”.

See `rmapi/archive/file.go` (`type Content`) and `rmapi/archive/reader.go` (`readContent`).

The rmapi test fixture shows the legacy-style container:

```bash
unzip -l rmapi/archive/test.zip
```

Expected entries (abridged):

```text
<docUUID>.content
<docUUID>.pagedata
<docUUID>.thumbnails/0.jpg
<docUUID>/0.rm
<docUUID>/0-metadata.json
```

In this legacy layout, page files are named `0.rm`, `1.rm`, ... so `rmapi/archive` can index pages by `atoi("0")` without needing any UUID→index mapping.

### 3.2. Modern `.content` (`cPages`) and why it matters

The modern fixture (`copies of different pages.rmdoc`) has a `.content` with a `cPages.pages[]` array:

- `id`: the **page UUID** (this matches the `.rm` filename stem).
- `redir.value`: where this page originates from in the source PDF (if applicable).
- `deleted.value`: whether this page should be considered removed (observed by `remarks`).
- `template.value`: background template per page.

This is the key compatibility point:

- **Modern `.rmdoc` uses UUID-named page `.rm` files** (e.g. `<pageUUID>.rm`).
- If your parser does not build a **page UUID → page index** mapping from `cPages.pages[].id`, you cannot order pages correctly.

In this repo, `remarks` handles `cPages` explicitly (`remarks/remarks/utils.py:get_pages_data` + `construct_redirection_map`), while `rmapi/archive` does not currently model `cPages` in `archive.Content`. That means:

- `remarks` can map UUID `.rm` files to page positions reliably for modern exports.
- `rmapi/archive.Zip.Read` can parse legacy numeric page archives, but will fail to map modern UUID page files unless `pages[]`/`redirectionPageMap` are present (or `archive.Content` is extended to understand `cPages`).

## 4. The `.metadata` file: title, parent, and “UI path”

The `<docUUID>.metadata` JSON describes the document at the library level:

- `visibleName`: user-facing title.
- `parent`: the UUID of the containing collection/folder.
- timestamps like `createdTime`, `lastModified`, etc.

`remarks` uses the `parent` chain to reconstruct the document’s “UI path” (folder hierarchy) by repeatedly loading parent `.metadata` files:

- `remarks/remarks/utils.py:get_ui_path`
- `remarks/remarks/utils.py:read_meta_file`

This is why `remarks` can output files into nested directories that match the on-device folder structure.

## 5. The per-page `.rm` files: “paths” and “notes”

Each page’s `<pageUUID>.rm` is the annotation payload. It is the source of:

- **Paths**: handwriting strokes as sequences of points grouped into lines and layers.
- **Notes**: typed text blocks, highlight geometry, and other scene elements (in newer formats).

### 5.1. `.rm` versions and parser split (Go vs Python)

This repo contains two different `.rm` parsers with different version support:

- **Go (`rmapi/encoding/rm`)**: models V3/V5 “lines” format (strokes, brush type/color/size, points). See `rmapi/encoding/rm/rm.go` (`HeaderV3`, `HeaderV5`, `type Rm`, `type Line`, `type Point`).
- **Python (`remarks`)**: currently supports **V6** and uses `rmscene` to build a scene graph. See `remarks/remarks/conversion/parsing.py:parse_rm_file` (V6 supported; V3/V5 rejected).

Practically:

- If you want “just strokes” and your `.rm` is V3/V5, the Go pipeline is a fit.
- If you want modern typed text + highlight rectangles (and your `.rm` is V6), the `remarks` pipeline is a fit.

## 6. How parsing works in this repo (end-to-end)

This repo contains two separate parsing pipelines that handle `.rmdoc` files with different capabilities and constraints. The Python `remarks` pipeline is modern and feature-complete (supports V6 `.rm`, handles `cPages`, produces rich output), while the Go `rmapi/archive` pipeline is older and more limited (V3/V5 `.rm` only, legacy `.content` schemas, focused on PDF export). Neither pipeline currently handles *all* format variants, which creates a gap that RMQ-0001 must address. Understanding what each pipeline *can* and *cannot* do is essential for designing a unified tool. This section traces each pipeline from ZIP extraction through to final output, highlighting the decision points and capabilities at each stage.

### 6.1. Python: `remarks` (modern `.rmdoc` / V6 `.rm` oriented)

`remarks` treats `.rmdoc` as “xochitl-like files in a folder (or zip)”, then:

- **Extract**: if input ends with `.rmn` or `.rmdoc`, unzip to a temp directory (`remarks/remarks/remarks.py:run_remarks`).
- **Discover documents**: iterate `*.metadata`, filter to `"type": "DocumentType"` (`remarks/remarks/utils.py:is_document`).
- **Load model**: build a `Document` object which:
  - reads page order from `.content` (including `cPages` redirection logic) (`remarks/remarks/utils.py:get_pages_data`)
  - finds page `.rm` files under `<docUUID>/*.rm` (`remarks/remarks/utils.py:list_ann_rm_files`)
- **Build background PDF**:
  - for PDF-backed docs: open `<docUUID>.pdf` (via `<docUUID>.metadata` stem) and insert blank/duplicate pages as required (`remarks/remarks/Document.py:open_source_pdf`)
  - for notebooks: build a blank PDF and infer page sizes from V6 path bounds (`remarks/remarks/Document.py:open_source_pdf` + `determine_document_dimensions`)
- **Parse “notes”** (typed text + highlight rectangles):
  - read `.rm` header version (`read_rm_file_version`)
  - for V6: `parse_rm_file` builds a scene tree and extracts text/highlight geometry (`remarks/remarks/conversion/parsing.py`)
- **Render “paths”**:
  - use `rmc.exporters.pdf.rm_to_pdf(rm_file, temp_pdf)` to produce a PDF containing the drawn strokes
  - merge that PDF page over the background PDF page using PyMuPDF `page.show_pdf_page(...)`
- **Apply smart highlights**:
  - turn highlight rectangles into PDF highlight annotations (`remarks/remarks/output/PdfFile.py:apply_smart_highlight`)
- **Write outputs**:
  - merged PDF: `<name> _remarks.pdf`
  - Obsidian markdown: `<name>.md`

Key files: `remarks/remarks/remarks.py`, `remarks/remarks/Document.py`, `remarks/remarks/utils.py`, `remarks/remarks/conversion/parsing.py`, `remarks/remarks/output/PdfFile.py`.

### 6.2. Go: `rmapi/archive` + `rmapi/annotations` (legacy `.content` / V3-V5 `.rm` oriented)

`rmapi` can treat `.rmdoc` as a ZIP and parse it into an `archive.Zip` object:

- `archive.Zip.Read(...)` reads `.content` first, then payload, then pagedata, then `.rm` data, and thumbnails (`rmapi/archive/reader.go`).
- `.rm` data is decoded into `rm.Rm` (V3/V5 model) via `UnmarshalBinary` (not shown in the snippet above, but used in `readData`).
- `annotations.PdfGenerator.Generate()` renders strokes into a PDF by iterating `Rm.Layers[].Lines[].Points[]` and drawing paths / highlight lines (`rmapi/annotations/pdf.go`).

This pipeline is great for “export annotations as PDF”, but it has two important constraints for RMQ-0001:

- **Modern `.content`**: if `.rm` filenames are UUIDs and `.content` only provides `cPages`, `rmapi/archive` currently cannot build the UUID→index map it needs.
- **V6 `.rm`**: Go’s `rmapi/encoding/rm` describes V3/V5; it does not parse V6 scene-based `.rm` files.

## 7. Converting “paths and notes” into PNGs (what to do in practice)

PNG output is almost always easiest as a second-stage rasterization step: first produce a correct, merged PDF page (background + paths + highlights/text), then rasterize that PDF page into PNG(s).

### 7.1. Recommended: rasterize the merged PDF produced by `remarks`

Once `remarks` produces `... _remarks.pdf`, you can turn each page into a PNG with PyMuPDF:

```python
import fitz

doc = fitz.open("My Document _remarks.pdf")
for i, page in enumerate(doc):
    pix = page.get_pixmap(dpi=200)  # tweak DPI for quality/size tradeoff
    pix.save(f"My Document - page {i+1}.png")
```

This approach automatically includes:

- background PDF content (when present),
- drawn paths (via `rm_to_pdf` + merge),
- highlight annotations (via `apply_smart_highlight`),
- page rotations/translations as handled by `remarks`.

### 7.2. Go option: render the generated PDF to images (similar to rmapi thumbnails)

`rmapi` already renders thumbnails from PDF bytes using UniDoc’s renderer (`rmapi/archive/zipdoc.go:makeThumbnail`). The same idea can be extended:

- generate a PDF (either via `rmapi/annotations` or by other means),
- render each page to an image buffer with UniDoc’s `render.NewImageDevice()`,
- encode to PNG via Go’s standard `image/png`.

### 7.3. Direct `.rm` → PNG (possible, but not what this repo does today)

Direct rasterization from `.rm` paths requires choosing a graphics backend (Cairo, Skia, etc.) and implementing stroke rendering consistent with reMarkable brushes. In this repo, the practical implementation is:

- `.rm` → vector (PDF) via `rmc.exporters.pdf.rm_to_pdf`
- vector + background composition via PyMuPDF
- then PDF → PNG via `get_pixmap`

## 8. Common gotchas (the “why is my output shifted?” section)

- **Page ordering vs filenames**: UUID-named `.rm` files require a mapping from `.content` (`cPages.pages[].id`) to page indices.
- **Inserted/duplicate pages**: PDF-backed documents can include pages not present in the original PDF (blank inserts) or duplicated pages; `remarks` accounts for this when constructing the background PDF.
- **Coordinate systems**: reMarkable uses a “center-top” origin model for some annotation coordinate spaces; `remarks` computes translations for correct overlay (`remarks/remarks/remarks.py` merge logic).
- **Rotation**: background PDF pages can be rotated; `remarks` temporarily zeroes rotation to compute correct placement.
- **Version skew**: `.rm` V6 parsing is not compatible with V3/V5 parsers, and vice versa.

## 9. Practical debugging checklist

When parsing/rendering fails, follow this sequence:

1. List ZIP entries: `unzip -l document.rmdoc` and verify you have `.content`, `.metadata`, and `<docUUID>/<pageUUID>.rm`.
2. Inspect `.content`:
   - if it contains `cPages`, make sure your page-order logic uses it.
   - if it contains `pages`, make sure you build a UUID→index map.
3. Inspect `.rm` header version (first line): V3/V5 vs V6 determines which parser to use.
4. Produce a merged PDF first, then rasterize to PNG.

---

# Part II: Algorithms for Go Reimplementation

Part I explained *what* is in a `.rmdoc` and *why* parsers need to handle format variants. Part II shifts to *how* — the precise algorithms, constants, and formulas needed to reimplement `remarks` functionality in Go. Each section provides executable pseudocode with comments explaining design decisions, not just syntax. The goal is to give you enough detail that you can write a Go implementation without needing to reverse-engineer the Python code yourself. We extracted these algorithms by running analysis scripts (in `scripts/`) against the live `remarks` and `rmc` libraries to capture exact constants and logic flow.

## 10. Coordinate Transformation Constants

ReMarkable's device coordinates don't map 1:1 to PDF coordinates — they need scaling and origin translation. The core challenge is that the reMarkable uses a **center-top origin** (X=0 is the horizontal center, Y=0 is the top edge), while PDFs typically use bottom-left origins. Additionally, the device coordinate space (1404×1872 pixels) must be scaled down to fit standard PDF export dimensions (445×596 points). This section documents the exact constants and formulas used by `rmc` to perform these transformations. All values were extracted via `scripts/extract_rmc_coords.py` from the live `rmc.exporters.svg` module.

```python
# Core scaling factor: PDF export width / device width
SCALE = 0.3185840707964602   # = 445 / 1404

# Device dimensions in pixels
SCREEN_WIDTH = 1404
SCREEN_HEIGHT = 1872

# Horizontal shift for centering (≈ SCREEN_WIDTH * SCALE / 2)
X_SHIFT = 223.0

# Text positioning constants
TEXT_TOP_Y = -88
LINE_HEIGHTS = {
    PLAIN: 70,
    BULLET: 35,
    BULLET2: 35,
    BOLD: 70,
    HEADING: 150,
    CHECKBOX: 35,
    CHECKBOX_CHECKED: 35,
}
```

### 10.1. Coordinate Transform Functions

The transformation from device coordinates to PDF coordinates is surprisingly simple: both `xx()` and `yy()` are identical functions that multiply by the `SCALE` constant. There's no Y-axis flip here (that happens elsewhere in the pipeline during PDF composition). The simplicity is deceptive — the real complexity is in understanding the center-top origin model and applying the correct offsets during the merge step (see Section 13.1).

```go
// Go implementation
const Scale = 0.3185840707964602  // 445 / 1404

// xx converts device X coordinate to PDF coordinate
func xx(deviceX float64) float64 {
    return deviceX * Scale
}

// yy converts device Y coordinate to PDF coordinate  
func yy(deviceY float64) float64 {
    return deviceY * Scale
}
```

### 10.2. ReMarkable Coordinate System Explained

The reMarkable device uses a **center-top origin**, which is unusual compared to most graphics systems. This means X=0 is not the left edge but the *horizontal center*, and Y=0 is the top edge (as expected). This design choice makes sense for a pen-input device where users naturally orient content around the center, but it complicates PDF overlay because PDF coordinate systems typically use bottom-left origins (though `rmc` handles this by producing PDFs that already account for the offset).

**Device coordinate space** (before scaling):

- **X axis**: -702 to +702 (center = 0, full width = 1404 pixels)
- **Y axis**: 0 (top) to 1872 (bottom)

**PDF coordinate space** (after applying `xx()` and `yy()`):

- **X axis**: -223.65 to +223.65
- **Y axis**: 0 to 596.39

Example coordinate transformations showing how device coordinates map to PDF space:

| Device (X, Y) | PDF (X, Y) | Location | Notes |
|---------------|------------|----------|-------|
| (0, 0) | (0.00, 0.00) | center-top | Origin point |
| (-702, 0) | (-223.65, 0.00) | left-top | Left edge |
| (702, 0) | (223.65, 0.00) | right-top | Right edge |
| (0, 1872) | (0.00, 596.39) | center-bottom | Bottom center |
| (-702, 1872) | (-223.65, 596.39) | left-bottom | Bottom-left corner |
| (702, 1872) | (223.65, 596.39) | right-bottom | Bottom-right corner |

**Key insight**: Negative X values are normal and expected — they represent content left of center. When merging annotations with background PDFs, you'll need to translate these negative coordinates into positive canvas positions (see Section 13.1's `xTranslation` calculation).

## 11. Page Ordering Algorithm (`get_pages_data`)

Page ordering is deceptively critical: if you get the page sequence wrong, annotations end up on the wrong pages, and inserted/duplicated pages appear in incorrect positions or not at all. This algorithm extracts two pieces of information from `.content`: (1) an ordered list of page UUIDs (these match `.rm` filenames), and (2) a "redirection map" that tells you which source PDF page each logical page corresponds to. The algorithm must handle both modern `cPages` format (with explicit per-page metadata) and legacy `pages` arrays (which assume numeric ordering). A robust implementation checks for `cPages` first, falls back to `pages`, and skips soft-deleted pages (marked with `deleted.value == 1`).

```go
// Pseudocode for get_pages_data
func GetPagesData(contentPath string) (pagesList []string, redirectionMap []int, err error) {
    content := readJSON(contentPath)  // Read .content file
    
    // Build redirection map from cPages if present
    // (Must do this first to handle inserted/duplicated pages)
    redirectionMap = constructRedirectionMap(content)
    
    if content.CPages != nil {
        // Modern format: use cPages.pages[].id
        for _, page := range content.CPages.Pages {
            // Skip deleted pages (soft-delete flag)
            if page.Deleted != nil && page.Deleted.Value == 1 {
                continue
            }
            pagesList = append(pagesList, page.ID)
        }
    } else {
        // Legacy format: use top-level pages array
        // (This assumes numeric page filenames or an explicit pages list)
        pagesList = content.Pages
    }
    
    return pagesList, redirectionMap, nil
}
```

### 11.1. Redirection Map Construction

The redirection map tells you which source PDF page each logical page corresponds to:

```go
const InsertedPage = -1  // Sentinel for blank inserted pages

// Pseudocode for construct_redirection_map
func ConstructRedirectionMap(content Content) []int {
    var redirectionMap []int
    
    if content.CPages == nil {
        return redirectionMap  // Empty for legacy format
    }
    
    for _, page := range content.CPages.Pages {
        if page.Redir != nil {
            // This page maps to source PDF page at index redir.value
            redirectionMap = append(redirectionMap, page.Redir.Value)
        } else {
            // No redir = inserted blank page
            redirectionMap = append(redirectionMap, InsertedPage)
        }
    }
    
    return redirectionMap
}
```

**Interpretation of redirection map values:**

- `redir.value >= 0`: Page shows content from source PDF at that index
- `InsertedPage (-1)`: User-inserted blank page (for notes)
- Same `redir.value` on multiple pages: Duplicated page

## 12. Background PDF Construction Algorithm

Before you can overlay annotations, you need the correct background PDF — and "correct" means handling three distinct scenarios: (1) PDF-backed documents where the source PDF exists but may need inserted blank pages or duplicated pages spliced in, (2) notebook documents that have no source PDF and must be synthesized from scratch with pages sized to fit the annotation content, and (3) EPUB documents (treated similarly to PDFs after conversion). The redirection map (from Section 11.1) drives this process: when you see `InsertedPage (-1)`, you create a blank page; when you see a repeated index, you duplicate an existing page. Notebooks are trickier because page dimensions must be inferred from the bounding box of strokes in each `.rm` file (V6 introduced variable page sizes). This algorithm shows how `remarks` builds the foundation document that annotations will be merged onto.

```go
// Pseudocode for open_source_pdf
func OpenSourcePDF(doc *Document) (*PDFDocument, error) {
    switch doc.DocType {
    case "pdf", "epub":
        return openPDFBackedDocument(doc)
    case "notebook":
        return createNotebookPDF(doc)
    default:
        return nil, fmt.Errorf("unsupported doc type: %s", doc.DocType)
    }
}

func openPDFBackedDocument(doc *Document) (*PDFDocument, error) {
    // Open the source PDF
    pdfPath := doc.MetadataPath.WithSuffix(".pdf")
    pdf := openPDF(pdfPath)
    
    sourcePDFPageCount := pdf.PageCount()
    
    // Process each page according to redirection map
    for i, redirectIdx := range doc.RedirectionMap {
        if isInsertedPage(redirectIdx) {
            // Insert a blank page at position i
            pdf.InsertBlankPage(i, REMARKABLE_DOCUMENT_WIDTH, REMARKABLE_DOCUMENT_HEIGHT)
        } else if isDuplicatePage(redirectIdx) && i >= sourcePDFPageCount {
            // Duplicate: copy from source page redirectIdx
            pdf.CopyPage(redirectIdx, i)
        }
        // else: page already exists at correct position
    }
    
    return pdf, nil
}

func createNotebookPDF(doc *Document) (*PDFDocument, error) {
    pdf := newEmptyPDF()
    
    // For each page, determine dimensions from .rm file bounds
    for i, pageUUID := range doc.PagesList {
        rmFile := findRMFile(doc, pageUUID)
        
        var dims Dimensions
        if rmFile != nil {
            dims = determineDocumentDimensions(rmFile)
        } else {
            dims = REMARKABLE_DOCUMENT  // Default: 1404 x 1872
        }
        
        // Convert to PDF units
        muDims := dims.ToMM().ToMU()
        pdf.NewPage(i, muDims.Width, muDims.Height)
    }
    
    return pdf, nil
}
```

### 12.1. Dimension Calculation for Notebooks

For notebooks, page dimensions are computed from the bounding box of all strokes:

```go
func DetermineDocumentDimensions(rmFilePath string) Dimensions {
    // Default boundaries (standard page size)
    dims := BoundingBox{
        XMin: -SCREEN_WIDTH / 2,  // -702
        XMax: SCREEN_WIDTH/2 - 1, // 701
        YMin: 0,
        YMax: SCREEN_HEIGHT - 1,  // 1871
    }
    
    // Parse .rm file and walk scene tree
    sceneTree := parseRMFile(rmFilePath)
    
    for element := range sceneTree.Walk() {
        if line, ok := element.(*Line); ok {
            for _, point := range line.Points {
                dims.XMin = min(dims.XMin, point.X)
                dims.XMax = max(dims.XMax, point.X)
                dims.YMin = min(dims.YMin, point.Y)
                dims.YMax = max(dims.YMax, point.Y)
            }
        }
    }
    
    return ReMarkableDimensions{
        Width:  dims.XMax - dims.XMin,
        Height: dims.YMax - dims.YMin,
    }
}
```

## 13. PDF Merge Algorithm (Core Rendering Loop)

This is where everything comes together: parsing `.rm` files, rendering strokes to PDF, computing bounding boxes, positioning annotation PDFs over background PDFs, and applying smart highlights. The algorithm operates in two passes — first collecting page tags (for markdown output), then processing each page's annotations individually. The core challenge is the merge step (Section 13.1): annotations and background may have different dimensions, rotations, and origins, so you must compute a canvas large enough to fit both, then position each PDF fragment correctly while tracking an `xTranslation` offset for smart highlights. The algorithm handles both PDF-backed pages (complex merge) and notebook pages (simple insertion) with different code paths. This pseudocode captures the structure of `remarks/remarks/remarks.py:process_document()` — see `scripts/analyze_remarks_merge.py` for a detailed walkthrough of the positioning formulas.

```go
func ProcessDocument(doc *Document, outputDir string) error {
    pdfSrc := doc.OpenSourcePDF()
    markdown := NewObsidianMarkdownFile(doc)
    
    // First pass: collect page tags
    for pageIdx, pageUUID := range doc.PagesList {
        if tags := doc.GetPageTags(pageUUID); len(tags) > 0 {
            markdown.AddPageTags(pageIdx, tags)
        }
    }
    
    // Second pass: process annotations
    for pageUUID, pageIdx, rmFile := range doc.Pages() {
        if rmFile == nil {
            continue  // No annotations on this page
        }
        
        // Check .rm version
        version := readRMFileVersion(rmFile)
        if version != V6 {
            addWarningAnnotation(pdfSrc[pageIdx], "Only V6 supported")
            continue
        }
        
        // Parse .rm file
        annData := parseRMFile(rmFile)
        
        // STEP 1: Convert .rm to PDF (strokes/paths)
        tempPDF := rmToPDF(rmFile)
        svgPDF := openPDF(tempPDF)
        defer removeTempFile(tempPDF)
        
        // STEP 2: Get current page
        page := pdfSrc[pageIdx]
        
        var highlightsXTranslation float64
        
        if page.HasContent() {
            // STEP 3: Merge with background
            highlightsXTranslation = mergeWithBackground(
                pdfSrc, pageIdx, svgPDF, annData,
            )
        } else {
            // STEP 6: Just insert annotations (notebook page)
            pdfSrc.InsertPDF(svgPDF, pageIdx)
            pdfSrc.DeletePage(pageIdx + 1)
        }
        
        // STEP 7: Apply smart highlights
        for _, highlight := range annData.Highlights {
            applySmartHighlight(pdfSrc[pageIdx], highlight, highlightsXTranslation)
        }
        
        // Collect for markdown
        if annData.Text != nil {
            markdown.AddText(pageIdx, annData.Text)
        }
        if len(annData.GlyphRanges) > 0 {
            markdown.AddHighlights(pageIdx, annData.GlyphRanges)
        }
    }
    
    // Save outputs
    pdfSrc.Save(outputDir + "/" + doc.Name + " _remarks.pdf")
    markdown.Save(outputDir + "/" + doc.Name)
    
    return nil
}
```

### 13.1. Merge With Background Algorithm

This is the most intricate part of the rendering pipeline: overlaying an annotation PDF (produced by `rm_to_pdf`) onto a background PDF page while preserving correct positioning, handling rotation, and computing the `xTranslation` offset needed for smart highlights. The algorithm computes bounding boxes to find the smallest rectangle containing all annotations, creates a canvas large enough to hold both the background and annotations (using `max(widths, heights)`), then positions each PDF fragment within that canvas. The trick is handling the center-top origin: when annotations are wider than the background, the background must be shifted right; when the background is wider, the annotations must be centered. The `xTranslation` value returned tells the highlight application step (Section 14) where the "center" is so highlight rectangles align with text. Page rotation adds another wrinkle: rotated pages have swapped width/height, so dimensions are temporarily normalized before positioning calculations.

```go
func mergeWithBackground(pdfSrc *PDF, pageIdx int, svgPDF *PDF, annData *AnnotationData) float64 {
    page := pdfSrc[pageIdx]
    
    // Get page rotation and temporarily zero it
    pageRotation := page.Rotation()
    page.SetRotation(0)
    
    // Get background dimensions
    wBg, hBg := page.CropBox().Width, page.CropBox().Height
    if pageRotation == 90 || pageRotation == 270 {
        wBg, hBg = hBg, wBg  // Swap for landscape
    }
    
    // Compute annotation bounding box
    anchorPos := buildAnchorPos(annData.SceneTree.RootText)
    xMin, xMax, yMin, yMax := getBoundingBox(annData.SceneTree.Root, anchorPos)
    
    // Convert to scaled PDF coordinates
    xShift := xx(xMin)
    yShift := yy(yMin)
    wSvg := xx(xMax - xMin + 1)
    hSvg := yy(yMax - yMin + 1)
    
    // Compute canvas size (must fit both)
    width := max(wSvg, wBg)
    height := max(hSvg, hBg)
    
    // Compute positions
    var xSvg, ySvg, xBg, yBg, highlightsXTranslation float64
    
    if wSvg > wBg {
        // Annotations wider: center background
        xBg = width/2 - wBg/2 - (wSvg/2 + xShift)
        highlightsXTranslation = xBg + wBg/2
    } else if wSvg < wBg {
        // Background wider: center annotations
        xSvg = width/2 - wSvg/2 + (wSvg/2 + xShift)
        highlightsXTranslation = wBg / 2
    }
    
    if hSvg > hBg {
        yBg = -yShift
    } else if hSvg < hBg {
        ySvg = yShift
    }
    
    // Create merged page
    mergedDoc := newEmptyPDF()
    mergedPage := mergedDoc.NewPage(width, height)
    
    // Draw background first
    mergedPage.ShowPDFPage(
        Rect{xBg, yBg, xBg + wBg, yBg + hBg},
        pdfSrc, pageIdx,
        -pageRotation,
    )
    
    // Draw annotations on top
    mergedPage.ShowPDFPage(
        Rect{xSvg, ySvg, xSvg + wSvg, ySvg + hSvg},
        svgPDF, 0,
        0,
    )
    
    // Replace page in source
    pdfSrc.InsertPDF(mergedDoc, pageIdx)
    pdfSrc.DeletePage(pageIdx + 1)
    
    return highlightsXTranslation
}
```

## 14. Smart Highlight Application

Smart highlights are "smart" because they're not just yellow rectangles drawn as vector paths — they're actual PDF highlight annotations that readers like Adobe Acrobat and Preview recognize as interactive elements (you can click them, extract their text, change their color). This distinction matters for PDF portability and searchability. The algorithm takes highlight rectangles extracted from the `.rm` scene tree (already scaled to PDF coordinates via `xx()`/`yy()` during parsing) and applies the `xTranslation` offset computed during the merge step (Section 13.1) to position them correctly relative to the merged canvas. Each highlight has a `color` field (a PenColor enum value) that maps to an RGB color via a lookup table. The default is yellow (`#FFED75`), but reMarkable supports multiple highlight colors that should be preserved in the output.

```go
func applySmartHighlight(page *PDFPage, highlight *RemarksRectangle, xTranslation float64) {
    // Get color from PenColor enum
    color := getHighlightColor(highlight.Color)
    
    for _, rect := range highlight.Rectangles {
        // Rectangles are already in PDF coordinates (scaled via xx/yy during parsing)
        // xTranslation positions them relative to reMarkable's (0,0) at center-top
        pdfRect := Rect{
            X1: rect.X + xTranslation,
            Y1: rect.Y,
            X2: rect.X + xTranslation + rect.W,
            Y2: rect.Y + rect.H,
        }
        
        annot := page.AddHighlightAnnotation(pdfRect)
        annot.SetStrokeColor(color)
        annot.SetOpacity(0.3)
        annot.Update()
    }
}

// Color lookup from PenColor enum
func getHighlightColor(penColor int) RGB {
    colorMap := map[int]RGB{
        // These come from rmscene.scene_items.HARDCODED_COLORMAP
        // Default: yellow highlight
    }
    
    if rgb, ok := colorMap[penColor]; ok {
        return rgb
    }
    return RGB{1.0, 0.93, 0.46}  // Default yellow: #FFED75
}
```

## 15. Go Reimplementation Checklist

Reimplementing `remarks` in Go is a substantial undertaking — you're essentially building a V6 `.rm` parser, a PDF composition engine, and a coordinate transformation pipeline. This checklist breaks the work into discrete components you can tackle independently, prioritized by dependency order (you need the parser before you can test rendering, you need coordinate transforms before you can test merging). The analysis scripts in `scripts/` provide executable references for the constants and algorithms, and the earlier sections document the exact formulas. The biggest unknowns are (1) whether to port or rewrite the V6 scene tree parser (`rmscene` is ~2000 lines of Python), and (2) which Go PDF library provides the best balance of capabilities (stroke rendering, PDF merging, annotation support) versus complexity.

### Components needed for a complete reimplementation

### 15.1. V6 `.rm` Parser

- Parse scene tree blocks (reference: `rmscene` Python library)
- Extract `Line` elements (points with x, y, pressure, tilt, width)
- Extract `GlyphRange` elements (highlights with rectangles and text)
- Extract `RootTextBlock` (typed text with paragraphs and styles)

### 15.2. PDF Library

Use a Go PDF library that supports:

- Creating new PDFs with vector paths/strokes
- Opening and reading existing PDFs
- Merging PDFs (overlay one on another)
- Adding highlight annotations
- Page manipulation (insert, delete, copy)

Options: `unipdf` (already used by rmapi), `pdfcpu`, `gofpdf`

### 15.3. Core Algorithms

1. **Page ordering**: Parse `.content` (`cPages` vs legacy `pages`)
2. **Redirection map**: Handle inserted/duplicated pages
3. **Coordinate transforms**: Apply `SCALE = 0.3185840707964602`
4. **Bounding box**: Walk scene tree, track min/max across Line points
5. **PDF merge**: Position background and annotation PDFs correctly
6. **Highlight positioning**: Apply `xTranslation` for center-top origin

### 15.4. Analysis Scripts

Reference scripts are stored in the ticket:

- `scripts/extract_rmc_coords.py` - Coordinate constants and transforms
- `scripts/analyze_remarks_merge.py` - PDF merge algorithm details

