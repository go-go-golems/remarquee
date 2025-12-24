---
Title: Using remarks for golden testing and PDF comparison
Ticket: RMQ-0006
Status: active
Topics:
    - go
    - remarkable
    - testing
    - validation
    - rmdoc
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../remarks/conftest.py
      Note: Pytest fixtures showing how remarks is invoked programmatically
    - Path: ../../../../../../../remarks/remarks/__main__.py
      Note: CLI interface for remarks - shows command-line invocation
    - Path: ../../../../../../../remarks/remarks/remarks.py
      Note: Main remarks processing logic - reference implementation for golden testing
    - Path: ../../../../../../../remarks/test_pdf.py
      Note: Example PDF testing patterns using PyMuPDF
    - Path: ../../../../../../../rmapi/archive/zipdoc.go
      Note: Reference UniDoc PDF->image rendering snippet (makeThumbnail)
    - Path: pkg/pdfcmp/pdfcmp.go
      Note: Pure-Go PDF visual+structural comparator (UniDoc render+extractor)
    - Path: pkg/pdfcmp/pdfcmp_test.go
      Note: Unit tests generating small PDFs and exercising comparator
    - Path: pkg/refimpl/remarks/runner.go
      Note: Go wrapper to invoke remarks and discover produced reference PDFs
    - Path: pkg/rmdoc/render/v6_merge_background_test.go
      Note: Existing remarquee render tests - pattern for golden test structure
ExternalSources: []
Summary: Research on using remarks as a reference implementation for golden testing of remarquee's PDF rendering output, including invocation methods, comparison strategies, and integration approaches.
LastUpdated: 2025-12-24T14:11:32.971845927-05:00
WhatFor: ""
WhenToUse: ""
---




# Using remarks for golden testing and PDF comparison

## Overview

`remarks` is a Python-based reference implementation for rendering ReMarkable `.rmdoc` files to PDF. It serves as an ideal "golden" baseline for validating remarquee's rendering output because:

1. **Mature implementation**: Handles V6 `.rm` parsing, coordinate transformations, highlight extraction, and PDF merging
2. **Same input format**: Processes `.rmdoc` ZIP files identically to remarquee
3. **Proven output**: Used in production by Scrybble for Obsidian integration
4. **Comprehensive coverage**: Supports PDF-backed docs, notebooks, cPages, inserted pages, and highlights

This document outlines strategies for using remarks as a golden reference in remarquee's testing pipeline.

## How remarks works

### Command-line invocation

```bash
# Basic usage
remarks <input_dir_or_rmdoc> <output_dir>

# With logging control
remarks <input_dir_or_rmdoc> <output_dir> --log_level DEBUG

# Example with .rmdoc file
remarks ./testdata/Test.rmdoc /tmp/remarks-output
```

**Output naming convention:**
- PDF: `{doc_name} _remarks.pdf` (note the space before underscore)
- Markdown: `{doc_name}.md`

**Key behavior:**
- Automatically extracts `.rmdoc`/`.rmn` ZIP files to temp directory
- Processes all `.metadata` files found in input directory
- Filters to document types: `pdf`, `epub`, `notebook`
- Skips non-document metadata files

### Programmatic invocation

```python
from remarks import run_remarks
import pathlib

input_path = pathlib.Path("./testdata/Test.rmdoc")
output_path = pathlib.Path("/tmp/remarks-output")

run_remarks(input_path, output_path)
```

**Notes:**
- `run_remarks()` handles ZIP extraction internally
- Returns `None` (void function)
- Output files are written to `output_dir` with naming convention above
- Logging can be controlled via Python's `logging` module before calling

### Processing pipeline

1. **Discovery**: Finds all `*.metadata` files
2. **Filtering**: Checks `is_document()` and `get_document_filetype()`
3. **Document creation**: Builds `Document` object with page order from `.content`
4. **Background PDF**: Opens source PDF (if PDF-backed) or creates blank PDF (notebooks)
5. **Page iteration**: For each page with `.rm` file:
   - Parses V6 `.rm` binary via `parse_rm_file()`
   - Renders strokes via `rmc.exporters.pdf.rm_to_pdf()` → temp PDF
   - Merges temp PDF onto background page using PyMuPDF `page.show_pdf_page()`
   - Extracts highlights and applies as PDF annotations via `apply_smart_highlight()`
6. **Output**: Saves merged PDF and Obsidian markdown

## PDF comparison strategies

### Strategy 1: Visual pixel comparison (most reliable for regression)

**Approach**: Render both PDFs to images and compare pixel-by-pixel.

**Pros:**
- Catches visual regressions (shifts, rotations, missing elements)
- Works regardless of PDF structure differences
- Can generate visual diff images for debugging

**Cons:**
- Requires consistent DPI/rendering settings
- May flag harmless differences (font rendering, anti-aliasing)
- Slower than structural comparison

**Implementation using PyMuPDF:**

```python
import fitz  # PyMuPDF
from PIL import Image
import numpy as np

def compare_pdfs_visual(pdf1_path, pdf2_path, dpi=200, tolerance=0.01):
    """
    Compare two PDFs by rendering pages to images.
    
    Args:
        pdf1_path: Path to first PDF (remarquee output)
        pdf2_path: Path to second PDF (remarks golden)
        dpi: Resolution for rendering (higher = more accurate, slower)
        tolerance: Fraction of pixels that can differ (0.01 = 1%)
    
    Returns:
        dict with 'match': bool, 'page_diffs': list, 'diff_images': list
    """
    doc1 = fitz.open(pdf1_path)
    doc2 = fitz.open(pdf2_path)
    
    if doc1.page_count != doc2.page_count:
        return {
            'match': False,
            'error': f'Page count mismatch: {doc1.page_count} vs {doc2.page_count}'
        }
    
    page_diffs = []
    diff_images = []
    
    for page_num in range(doc1.page_count):
        page1 = doc1[page_num]
        page2 = doc2[page_num]
        
        # Render to pixmaps
        pix1 = page1.get_pixmap(dpi=dpi)
        pix2 = page2.get_pixmap(dpi=dpi)
        
        # Convert to numpy arrays
        img1 = np.frombuffer(pix1.samples, dtype=np.uint8).reshape(
            pix1.height, pix1.width, pix1.n
        )
        img2 = np.frombuffer(pix2.samples, dtype=np.uint8).reshape(
            pix2.height, pix2.width, pix2.n
        )
        
        # Handle size mismatch (shouldn't happen with same DPI, but be safe)
        if img1.shape != img2.shape:
            page_diffs.append({
                'page': page_num,
                'match': False,
                'error': f'Image size mismatch: {img1.shape} vs {img2.shape}'
            })
            continue
        
        # Compute difference
        diff = np.abs(img1.astype(int) - img2.astype(int))
        diff_pixels = np.sum(diff > 0)
        total_pixels = img1.size
        diff_ratio = diff_pixels / total_pixels
        
        match = diff_ratio <= tolerance
        
        page_diffs.append({
            'page': page_num,
            'match': match,
            'diff_ratio': diff_ratio,
            'diff_pixels': diff_pixels,
            'total_pixels': total_pixels
        })
        
        # Generate diff image if mismatch
        if not match:
            diff_img = Image.fromarray(diff.astype(np.uint8))
            diff_images.append((page_num, diff_img))
    
    doc1.close()
    doc2.close()
    
    overall_match = all(d['match'] for d in page_diffs)
    
    return {
        'match': overall_match,
        'page_diffs': page_diffs,
        'diff_images': diff_images
    }
```

**Usage in Go test:**

```go
// Golden test helper that invokes Python comparison script
func TestRenderV6Golden(t *testing.T) {
    fixture := "./testdata/Test.rmdoc"
    
    // Render with remarquee
    remarqueeOut := "/tmp/remarquee-test.pdf"
    err := renderV6(fixture, remarqueeOut)
    require.NoError(t, err)
    
    // Render with remarks (golden)
    remarksOut := "/tmp/remarks-golden.pdf"
    err = runRemarks(fixture, remarksOut)
    require.NoError(t, err)
    
    // Compare
    result := comparePDFsVisual(remarqueeOut, remarksOut)
    assert.True(t, result.Match, "PDFs should match visually")
    
    // Save diff images on failure
    if !result.Match {
        for _, diff := range result.DiffImages {
            saveDiffImage(diff, fmt.Sprintf("/tmp/diff-page-%d.png", diff.Page))
        }
    }
}
```

### Strategy 2: Structural comparison (faster, less comprehensive)

**Approach**: Compare PDF structure (page count, annotations, text content, metadata).

**Pros:**
- Faster than visual comparison
- Can pinpoint specific differences (missing annotation, wrong text)
- Works well for highlight/annotation validation

**Cons:**
- May miss visual issues (coordinate shifts, rendering artifacts)
- PDF structure can differ while visual output is identical

**Implementation:**

```python
import fitz

def compare_pdfs_structural(pdf1_path, pdf2_path):
    """
    Compare PDFs by structure: page count, annotations, text content.
    """
    doc1 = fitz.open(pdf1_path)
    doc2 = fitz.open(pdf2_path)
    
    if doc1.page_count != doc2.page_count:
        return {
            'match': False,
            'error': f'Page count mismatch: {doc1.page_count} vs {doc2.page_count}'
        }
    
    page_diffs = []
    
    for page_num in range(doc1.page_count):
        page1 = doc1[page_num]
        page2 = doc2[page_num]
        
        # Compare annotations
        annots1 = list(page1.annots())
        annots2 = list(page2.annots())
        
        if len(annots1) != len(annots2):
            page_diffs.append({
                'page': page_num,
                'match': False,
                'error': f'Annotation count mismatch: {len(annots1)} vs {len(annots2)}'
            })
            continue
        
        # Compare text content
        text1 = page1.get_text()
        text2 = page2.get_text()
        
        # Compare annotation types and positions
        annot_types1 = [a.type[1] for a in annots1]  # type[1] is annotation type name
        annot_types2 = [a.type[1] for a in annots2]
        
        match = (
            text1 == text2 and
            annot_types1 == annot_types2
        )
        
        page_diffs.append({
            'page': page_num,
            'match': match,
            'text_match': text1 == text2,
            'annot_count_match': len(annots1) == len(annots2),
            'annot_types_match': annot_types1 == annot_types2
        })
    
    doc1.close()
    doc2.close()
    
    overall_match = all(d['match'] for d in page_diffs)
    
    return {
        'match': overall_match,
        'page_diffs': page_diffs
    }
```

### Strategy 3: Hybrid approach (recommended)

**Approach**: Use structural comparison for fast feedback, visual comparison for final validation.

**Workflow:**
1. **Fast path**: Structural comparison in CI (catches most issues quickly)
2. **Deep path**: Visual comparison for release candidates or on-demand
3. **Golden update**: When intentional changes are made, update golden files

## Integration approaches

### Approach 1: Subprocess invocation (simplest)

**Pros:**
- No Python dependencies in Go codebase
- Can use existing remarks installation
- Easy to debug (can run remarks manually)

**Cons:**
- Requires Python/remarks to be installed in test environment
- Slower (process startup overhead)
- Harder to pass structured data

**Implementation:**

```go
package render

import (
    "os/exec"
    "path/filepath"
    "testing"
)

func runRemarks(inputRmdoc, outputDir string) error {
    cmd := exec.Command("remarks", inputRmdoc, outputDir)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}

func getRemarksOutputPath(outputDir, docName string) string {
    // remarks outputs: "{doc_name} _remarks.pdf"
    return filepath.Join(outputDir, docName+" _remarks.pdf")
}
```

### Approach 2: Python script wrapper (more control)

**Pros:**
- Can add comparison logic in Python (easier PDF manipulation)
- Can return structured JSON results
- Can handle multiple comparison strategies

**Cons:**
- Still requires Python runtime
- More moving parts

**Implementation:**

Create `scripts/compare_with_remarks.py`:

```python
#!/usr/bin/env python3
"""
Compare remarquee output with remarks golden reference.
Usage: compare_with_remarks.py <remarquee_pdf> <remarks_pdf> [--visual|--structural]
"""
import json
import sys
from pathlib import Path

def main():
    remarquee_pdf = Path(sys.argv[1])
    remarks_pdf = Path(sys.argv[2])
    mode = sys.argv[3] if len(sys.argv) > 3 else 'structural'
    
    if mode == 'visual':
        result = compare_pdfs_visual(remarquee_pdf, remarks_pdf)
    else:
        result = compare_pdfs_structural(remarquee_pdf, remarks_pdf)
    
    print(json.dumps(result, indent=2))
    sys.exit(0 if result['match'] else 1)

if __name__ == '__main__':
    main()
```

**Go usage:**

```go
func compareWithRemarks(remarqueePDF, remarksPDF string, visual bool) (*ComparisonResult, error) {
    mode := "structural"
    if visual {
        mode = "visual"
    }
    
    cmd := exec.Command("python3", "scripts/compare_with_remarks.py", 
        remarqueePDF, remarksPDF, "--"+mode)
    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }
    
    var result ComparisonResult
    if err := json.Unmarshal(output, &result); err != nil {
        return nil, err
    }
    
    return &result, nil
}
```

### Approach 3: Go PDF library comparison (no Python dependency)

**Pros:**
- No external dependencies
- Faster (no subprocess overhead)
- Better integration with Go test infrastructure

**Cons:**
- Need to implement comparison logic in Go
- May need to render PDFs to images in Go (requires image library)
- Less mature PDF manipulation libraries in Go

**Implementation using UniDoc:**

```go
import (
    "github.com/unidoc/unipdf/v3/model"
    "github.com/unidoc/unipdf/v3/render"
)

func comparePDFsStructural(pdf1Path, pdf2Path string) (*ComparisonResult, error) {
    r1, err := model.NewPdfReaderFromFile(pdf1Path)
    if err != nil {
        return nil, err
    }
    
    r2, err := model.NewPdfReaderFromFile(pdf2Path)
    if err != nil {
        return nil, err
    }
    
    numPages1, _ := r1.GetNumPages()
    numPages2, _ := r2.GetNumPages()
    
    if numPages1 != numPages2 {
        return &ComparisonResult{
            Match: false,
            Error: fmt.Sprintf("Page count mismatch: %d vs %d", numPages1, numPages2),
        }, nil
    }
    
    // Compare page by page...
    // (UniDoc can extract annotations, text, etc.)
}
```

## Golden file management

### Where to store golden files

**Option 1: Commit golden PDFs to git**
- Pros: Version controlled, always available
- Cons: Binary files in git, can bloat repository

**Option 2: Generate on-demand**
- Pros: No binary files in repo, always fresh
- Cons: Requires remarks to be installed, slower first run

**Option 3: External artifact storage**
- Pros: Keeps repo clean, can version artifacts separately
- Cons: More complex setup, requires artifact server

**Recommendation**: Start with Option 2 (generate on-demand), migrate to Option 1 if needed for CI stability.

### Golden file naming convention

```
testdata/golden/
  Test.rmdoc.remarks.pdf          # Golden PDF from remarks
  Test.rmdoc.remarks-page-0.png   # Per-page golden images (optional)
  cpage-pdf.rmdoc.remarks.pdf
  legacy-pdf-a4.zip.remarks.pdf
```

### Updating golden files

**Workflow:**
1. Make intentional rendering changes
2. Run `go test -update-golden` (custom flag)
3. Regenerate remarks output: `remarks testdata/Test.rmdoc testdata/golden/`
4. Commit updated golden files

**Implementation:**

```go
var updateGolden = flag.Bool("update-golden", false, "Update golden files")

func TestRenderV6Golden(t *testing.T) {
    fixture := "./testdata/Test.rmdoc"
    goldenPath := "./testdata/golden/Test.rmdoc.remarks.pdf"
    
    // Render with remarquee
    actualPath := "/tmp/remarquee-test.pdf"
    err := renderV6(fixture, actualPath)
    require.NoError(t, err)
    
    if *updateGolden {
        // Regenerate golden from remarks
        err := runRemarks(fixture, "./testdata/golden/")
        require.NoError(t, err)
        t.Logf("Updated golden file: %s", goldenPath)
        return
    }
    
    // Compare with golden
    result := comparePDFs(actualPath, goldenPath)
    assert.True(t, result.Match, "PDF should match golden reference")
}
```

## Best practices

### 1. Use appropriate tolerance for visual comparison

- **Strict (0.0%)**: Only for pixel-perfect requirements (rarely needed)
- **Moderate (0.1-1%)**: Catches real regressions while ignoring rendering differences
- **Loose (1-5%)**: For smoke tests or when rendering differences are expected

**Recommendation**: Start with 1% tolerance, tighten if needed.

### 2. Compare page-by-page, not whole document

- Easier to debug (know which page failed)
- Can skip known-different pages
- Faster failure (stop on first mismatch)

### 3. Generate diff images on failure

- Visual debugging is essential for PDF rendering issues
- Save diff images to test output directory
- Include in CI artifacts for review

### 4. Test fixtures should cover edge cases

- PDF-backed docs with cPages
- Notebooks with blank pages
- Documents with highlights
- Documents with typed text
- Documents with rotations
- Documents with inserted/duplicated pages

### 5. Handle remarks version differences

- Pin remarks version in CI (or document expected version)
- Be aware that remarks updates may change output format
- Consider version-specific golden files if needed

## Gotchas and limitations

### 1. Output naming differences

- **remarks**: `{doc_name} _remarks.pdf` (space before underscore)
- **remarquee**: `{doc_name}.pdf` (no suffix)

**Solution**: Normalize naming in comparison wrapper.

### 2. Coordinate system differences

- remarks uses PyMuPDF's coordinate system
- remarquee uses UniDoc's coordinate system
- Both should produce visually identical output, but internal representation differs

**Solution**: Compare visual output, not PDF structure.

### 3. Font rendering differences

- Different PDF libraries may render fonts slightly differently
- Anti-aliasing can cause pixel-level differences

**Solution**: Use tolerance in visual comparison, or compare text content separately.

### 4. Annotation representation

- remarks uses PyMuPDF annotations
- remarquee uses UniDoc annotations
- Structure may differ but visual result should match

**Solution**: Compare annotation positions and text, not internal structure.

### 5. Temporary file cleanup

- remarks extracts `.rmdoc` to temp directory
- Temp directory may not be cleaned up on error

**Solution**: Use `defer` cleanup or context-based temp dir management.

## Example test structure

```go
package render_test

import (
    "testing"
    "path/filepath"
    "github.com/stretchr/testify/require"
    "github.com/stretchr/testify/assert"
)

func TestRenderV6Golden_TestRmdoc(t *testing.T) {
    fixture := "./testdata/Test.rmdoc"
    goldenPath := "./testdata/golden/Test.rmdoc.remarks.pdf"
    
    // Render with remarquee
    actualPath := filepath.Join(t.TempDir(), "remarquee.pdf")
    err := renderV6(fixture, actualPath)
    require.NoError(t, err)
    
    // Ensure golden exists (generate if needed)
    if !fileExists(goldenPath) {
        t.Logf("Golden file missing, generating from remarks...")
        err := runRemarks(fixture, filepath.Dir(goldenPath))
        require.NoError(t, err)
    }
    
    // Compare
    result := comparePDFsVisual(actualPath, goldenPath, tolerance: 0.01)
    
    if !result.Match {
        // Save diff images for debugging
        for _, diff := range result.DiffImages {
            diffPath := filepath.Join(t.TempDir(), 
                fmt.Sprintf("diff-page-%d.png", diff.Page))
            saveImage(diff.Image, diffPath)
            t.Logf("Diff image saved: %s", diffPath)
        }
    }
    
    assert.True(t, result.Match, 
        "PDF should match golden reference (diff ratio: %.2f%%)", 
        result.MaxDiffRatio*100)
}
```

## Next steps

1. **Implement comparison utilities**: Create Go package for PDF comparison (or Python script wrapper)
2. **Set up golden file generation**: Script to generate golden PDFs from remarks for all fixtures
3. **Add golden test cases**: Create test functions for each fixture
4. **CI integration**: Add golden tests to CI pipeline
5. **Documentation**: Update testing playbook with golden test procedures

## VLM/LLM OCR validation

In addition to automated pixel/structural comparison, visual language models (VLMs) can be used to validate PDF rendering output by analyzing rendered images. This provides semantic validation that automated comparison cannot catch.

### Using pinocchio for VLM validation

The `pinocchio` tool can analyze rendered PDF pages using vision models:

```bash
# Render PDF pages to images first
pdftoppm -png -r 200 input.pdf /tmp/page

# Use pinocchio to analyze images
pinocchio code professional \
  --images /tmp/page-1.png,/tmp/page-2.png \
  "Describe these images. Are there any rendering issues? Check for: \
   - Missing or misaligned strokes \
   - Missing highlights or highlight misalignment \
   - Missing typed text \
   - Incorrect page dimensions \
   - Missing template backgrounds"
```

### Helper CLI in remarquee

For a tighter loop, remarquee includes a helper command that:
- renders selected PDF pages to PNGs (Poppler `pdftoppm`, by default)
- calls `pinocchio code professional --images ... <prompt>`

Example:

```bash
go run ./cmd/remarquee rmdoc vlm-validate \
  --pdf-b /tmp/reference.pdf \
  --pages 1,2 \
  --rasterizer poppler \
  --prompt "Compare A vs B. List any differences in strokes/highlights/text/page size/template." \
  /tmp/remarquee.pdf
```

### Integration in golden tests

```go
func TestRenderV6Golden_WithVLMValidation(t *testing.T) {
    fixture := "./testdata/Test.rmdoc"
    remarqueeOut := "/tmp/remarquee-test.pdf"
    remarksOut := "/tmp/remarks-golden.pdf"
    
    // Render both PDFs
    renderV6(fixture, remarqueeOut)
    runRemarks(fixture, "/tmp")
    
    // Render to images
    remarqueeImages := renderPDFToImages(remarqueeOut, "/tmp/remarquee-pages")
    remarksImages := renderPDFToImages(remarksOut, "/tmp/remarks-pages")
    
    // Compare with VLM
    for i := range remarqueeImages {
        result := validateWithVLM(
            remarqueeImages[i],
            remarksImages[i],
            "Compare these two rendered pages. Are they visually identical?",
        )
        assert.True(t, result.Match, "VLM validation failed: %s", result.Reason)
    }
}
```

### VLM prompt examples

**General validation:**
```
"Are these two images visually identical? Describe any differences you see."
```

**Specific checks:**
```
"Compare these rendered PDF pages. Check for:
1. Are all pen strokes present and correctly positioned?
2. Are highlights aligned with the text they should highlight?
3. Is typed text visible and readable?
4. Does the page size match the expected dimensions?
5. Are template backgrounds (ruled lines, grids, etc.) visible?"
```

**Regression detection:**
```
"Compare this rendered page to the reference. Identify any regressions:
- Missing elements
- Misaligned elements
- Incorrect colors
- Wrong page dimensions"
```

### Advantages of VLM validation

- **Semantic understanding**: Can identify "looks wrong" even if pixel-perfect match
- **Natural language reports**: Easy to understand failure reasons
- **Flexible prompts**: Can ask for specific checks or general comparison
- **Complementary to automated tests**: Catches issues automated comparison might miss

### Limitations

- **Cost**: VLM API calls have cost per image
- **Speed**: Slower than automated comparison
- **Non-deterministic**: May give slightly different results on reruns
- **Requires images**: Must render PDFs to images first

### Recommended workflow

1. **Fast path**: Automated pixel/structural comparison in CI
2. **Deep path**: VLM validation for release candidates or on-demand
3. **Debug path**: VLM validation when automated tests fail but visual inspection needed

## Code analysis: Rendering issues investigation

This section traces through the remarquee codebase to understand the root causes of rendering issues identified through visual inspection.

### Issue 1: Strokes render without color (all black)

**Location**: `remarquee/pkg/rmdoc/render/v6_merge_background.go:248-271`

**Root cause**: The `buildOverlayOps` function hardcodes stroke color to black:

```248:253:remarquee/pkg/rmdoc/render/v6_merge_background.go
func buildOverlayOps(strokes []rmdoc.Stroke, bbox rmdoc.BBox, xSvg, ySvg, wSvg, hSvg float64, opts V6MergeOptions) string {
	cc := contentstream.NewContentCreator()
	cc.Add_q()

	cc.Add_w(opts.StrokeWidthPt)
	cc.Add_RG(0, 0, 0)  // Hardcoded black color
```

**Analysis**:
- `Stroke` struct (`remarquee/pkg/rmdoc/strokes.go:18-27`) has a `Color uint32` field
- This color field is populated during V6 parsing (`remarquee/pkg/rmdoc/rmv6_line_decode.go:90`)
- However, `buildOverlayOps` ignores `s.Color` and always sets `RG(0, 0, 0)` (black)
- The same issue exists in `RenderRMV6StrokesToPDF` (`remarquee/pkg/rmdoc/render/v6_strokes_pdf.go:52`)

**Fix required**: Extract color from `Stroke.Color` and convert to RGB before calling `cc.Add_RG()`. May need to map `uint32` color to `PenColor` enum or RGB values.

### Issue 2: Highlighter strokes misaligned with other strokes

**Location**: `remarquee/pkg/rmdoc/render/v6_merge_background.go:130-196`

**Root cause**: Different coordinate systems and transforms for strokes vs highlights.

**Stroke positioning** (`buildOverlayOps`, lines 260-262):
```260:262:remarquee/pkg/rmdoc/render/v6_merge_background.go
			x := xSvg + xx(float64(p.X)-bbox.MinX)
			y := ySvg + (hSvg - yy(float64(p.Y)-bbox.MinY))
			path = path.AppendPoint(draw.NewPoint(x, y))
```

**Highlight positioning** (`applySmartHighlights`, lines 373-378):
```373:378:remarquee/pkg/rmdoc/render/v6_merge_background.go
			x1 := xx(rect.X) + highlightsXTranslation
			x2 := x1 + xx(rect.W)

			// rect.Y is in screen coords (top-origin, y down). Convert to PDF (bottom-origin, y up).
			yTop := pageHeight - yy(rect.Y)
			yBottom := pageHeight - yy(rect.Y+rect.H)
```

**Analysis**:
- Strokes use `xSvg + xx(p.X - bbox.MinX)` (relative to bbox, then shifted by `xSvg`)
- Highlights use `xx(rect.X) + highlightsXTranslation` (absolute coordinates with translation)
- `highlightsXTranslation` is calculated at line 159-166, but strokes use `xSvg` calculated at line 164
- These may not align when `wSvg != wBg` (annotation bbox width != background width)
- Y-coordinate transforms also differ: strokes use `hSvg - yy(p.Y - bbox.MinY)`, highlights use `pageHeight - yy(rect.Y)`

**Fix required**: Ensure highlights and strokes use consistent coordinate transforms. May need to apply same bbox-relative transform to highlights.

### Issue 3: Typed text not rendered

**Location**: `remarquee/pkg/rmdoc/rmv6_root_text.go` (parsing) vs `remarquee/pkg/rmdoc/render/v6_merge_background.go` (rendering)

**Root cause**: Typed text is parsed but never rendered.

**Parsing exists**:
- `ParseRMV6RootTextBlock` (`remarquee/pkg/rmdoc/rmv6_root_text.go:108-205`) parses RootTextBlock
- `RMV6RootText` struct contains text items, styles, and position (`PosX`, `PosY`, `Width`)

**Rendering missing**:
- `MergeRMDocV6OntoBackgroundPDFWithInfo` extracts strokes and glyph ranges (lines 120-128) but not RootTextBlock
- No function renders `RMV6RootText` to PDF
- Comment at line 46 says "This currently merges strokes only (no highlights/text)" - but highlights ARE implemented, only typed text is missing

**Analysis**:
- Scene tree parsing likely extracts RootTextBlock, but it's not passed to renderer
- Need to extract RootTextBlock from scene tree similar to `ExtractRMV6StrokesWithAnchors`
- Need to render text at `PosX`, `PosY` with `Width` constraint
- May need font selection and text layout logic

**Fix required**: 
1. Extract RootTextBlock from scene tree
2. Create render function for `RMV6RootText` 
3. Integrate text rendering into merge pipeline

### Issue 4: Page format uses annotation bbox instead of full page

**Location**: `remarquee/pkg/rmdoc/render/v6_merge_background.go:130-177`

**Root cause**: Page size calculation uses annotation bounding box when background is empty.

**Problematic code** (lines 130-177):
```130:177:remarquee/pkg/rmdoc/render/v6_merge_background.go
		stBBox, ok := rmdoc.BBoxForStrokes(strokes, 0)
		if !ok || stBBox.IsEmpty() {
			// Empty annotations.
			if err := w.AddPage(bgPage); err != nil {
				return nil, err
			}
			continue
		}

		xMin, xMax := stBBox.MinX, stBBox.MaxX
		yMin, yMax := stBBox.MinY, stBBox.MaxY

		xShift := xx(xMin)
		yShift := yy(yMin)
		wSvg := xx((xMax - xMin) + 1)
		hSvg := yy((yMax - yMin) + 1)

		// Background dims + rotation.
		w0, h0, rot, err := pageBoxDims(bgPage)
		if err != nil {
			return nil, errors.Wrapf(err, "background page dims (page=%d)", pageNum)
		}

		wBg, hBg := displayDims(w0, h0, rot)

		width := math.Max(wSvg, wBg)
		height := math.Max(hSvg, hBg)
```

**Analysis**:
- When background has no content (line 176), `buildAnnotationOnlyPage` is called with `wSvg, hSvg` (annotation bbox size)
- This creates a page sized to annotations, not full page size
- For notebooks (no PDF payload), `BuildBackgroundPDF` creates blank pages using `DefaultPageSize` (445x594 points, line 26 in `background.go`)
- But when merging, if background is empty, it uses annotation bbox instead

**Fix required**: 
- For notebooks, use full page size (rmv6ScreenWidth/Height * scale = 1404*72/226 x 1872*72/226 points)
- For PDF-backed docs with empty background, use background page size, not annotation bbox
- Annotation-only pages should still be full page size

### Issue 5: Template backgrounds not rendered

**Location**: `remarquee/pkg/rmdoc/render/background.go:31-126`

**Root cause**: Template rendering is not implemented.

**Template data exists**:
- `PageRef.Template` field (`remarquee/pkg/rmdoc/types.go:79-80`) stores template name
- Templates extracted from `.pagedata` or `cPages.template` (`remarquee/pkg/rmdoc/pagedata.go:8-25`)
- `Document.Pagedata` field contains template names (`remarquee/pkg/rmdoc/types.go:59-60`)

**Template rendering missing**:
- `BuildBackgroundPDF` comment at line 38 says "Template backgrounds are a later milestone"
- For notebook pages (no payload), it creates blank pages using `DefaultPageSize` (line 115-116)
- No template lookup or rendering logic exists

**Analysis**:
- Template names like "Blank", "Lined", "Grid", "Dotted" are stored but not used
- Need to map template names to actual template images/PDFs
- May need template assets or generation logic
- Remarks may have template handling - need to investigate

**Fix required**:
1. Define template-to-background mapping (images or PDF generation)
2. Lookup template by name from `PageRef.Template`
3. Render template as background before annotations
4. Handle template sizing (should match page size)

## References

- `remarks/remarks/remarks.py`: Main remarks processing logic
- `remarks/test_pdf.py`: Example PDF testing patterns
- `remarks/conftest.py`: Pytest fixtures for remarks testing
- `remarquee/pkg/rmdoc/render/v6_merge_background_test.go`: Existing remarquee render tests
