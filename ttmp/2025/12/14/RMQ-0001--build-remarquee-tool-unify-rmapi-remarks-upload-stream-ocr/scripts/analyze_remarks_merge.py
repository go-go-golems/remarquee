#!/usr/bin/env python3
"""
analyze_remarks_merge.py - Analyze the PDF merge algorithm in remarks.

This script extracts and documents the merge algorithm used in remarks.py
to overlay annotation PDFs onto background PDFs. Essential for Go reimplementation.

Run from the remarks project directory:
    cd /path/to/remarks && poetry run python /path/to/this/script.py
"""

import inspect
import ast

def main():
    print("=" * 80)
    print("REMARKS PDF MERGE ALGORITHM ANALYSIS")
    print("=" * 80)
    print()
    
    # Read the remarks.py file directly
    import pathlib
    remarks_path = pathlib.Path(__file__).parent.parent.parent.parent.parent.parent / "remarks" / "remarks" / "remarks.py"
    
    if not remarks_path.exists():
        # Try relative path from CWD
        remarks_path = pathlib.Path("remarks/remarks.py")
    
    print(f"Reading: {remarks_path}")
    print()
    
    source = remarks_path.read_text()
    
    # Find the key section: the merge logic in process_document
    print("# 1. IMPORTS AND DEPENDENCIES")
    print("-" * 40)
    print("""
from rmc.exporters.pdf import rm_to_pdf
from rmc.exporters.svg import build_anchor_pos, get_bounding_box
from rmc.exporters.svg import xx, yy
""")
    
    print()
    print("# 2. CORE MERGE ALGORITHM (from process_document)")
    print("-" * 40)
    print("""
The merge algorithm in process_document() works as follows:

STEP 1: Convert .rm to PDF
--------------------------
    rm_to_pdf(rm_annotation_file, temp_pdf.name)
    svg_pdf = fitz.open(temp_pdf.name)

This uses rmc to render all strokes/paths from the .rm file into a temporary PDF.
The resulting PDF has coordinates already scaled by SCALE (see extract_rmc_coords.py).

STEP 2: Check if background page has content
--------------------------------------------
    if page.get_contents():
        # Has background content (PDF/EPUB) - need to merge
    else:
        # Empty page (notebook) - just insert annotations

STEP 3: For pages WITH background content - compute positioning
---------------------------------------------------------------
    # Get page rotation and temporarily zero it
    page_rotation = page.rotation
    page.set_rotation(0)
    
    # Get background page dimensions
    w_bg, h_bg = page.cropbox.width, page.cropbox.height
    if page_rotation in [90, 270]:
        w_bg, h_bg = h_bg, w_bg  # Swap for landscape
    
    # Compute annotation bounding box
    anchor_pos = build_anchor_pos(ann_data["scene_tree"].root_text)
    x_min, x_max, y_min, y_max = get_bounding_box(ann_data["scene_tree"].root, anchor_pos)
    
    # Convert to scaled coordinates
    x_shift = xx(x_min)      # Left edge of annotation content
    y_shift = yy(y_min)      # Top edge of annotation content  
    w_svg = xx(x_max - x_min + 1)  # Width of annotation content
    h_svg = yy(y_max - y_min + 1)  # Height of annotation content

STEP 4: Compute merge canvas and positions
------------------------------------------
    # Canvas must fit both background and annotations
    width = max(w_svg, w_bg)
    height = max(h_svg, h_bg)
    
    # Initialize positions
    x_svg, y_svg = 0, 0  # Where to place annotation PDF
    x_bg, y_bg = 0, 0    # Where to place background PDF
    highlights_x_translation = 0  # For smart highlight positioning
    
    # Case 1: Annotations wider than background
    if w_svg > w_bg:
        x_bg = width/2 - w_bg/2 - (w_svg/2 + x_shift)
        highlights_x_translation = x_bg + w_bg/2
    
    # Case 2: Background wider than annotations  
    elif w_svg < w_bg:
        x_svg = width/2 - w_svg/2 + (w_svg/2 + x_shift)
        highlights_x_translation = w_bg/2
    
    # Vertical positioning
    if h_svg > h_bg:
        y_bg = -y_shift
    elif h_svg < h_bg:
        y_svg = y_shift

STEP 5: Create merged page
--------------------------
    # Create a new document with a blank page
    doc = fitz.open()
    page = doc.new_page(-1, width=width, height=height)
    
    # Draw background first
    page.show_pdf_page(
        fitz.Rect(x_bg, y_bg, x_bg + w_bg, y_bg + h_bg),
        rmc_pdf_src,      # Source background PDF
        page_idx,         # Page index in source
        rotate=-page_rotation
    )
    
    # Draw annotations on top
    page.show_pdf_page(
        fitz.Rect(x_svg, y_svg, x_svg + w_svg, y_svg + h_svg),
        svg_pdf,          # Annotation PDF from rm_to_pdf
        0                 # Always page 0 (single-page temp PDF)
    )
    
    # Replace the page in the output document
    rmc_pdf_src.insert_pdf(doc, start_at=page_idx)
    rmc_pdf_src.delete_page(page_idx + 1)

STEP 6: For pages WITHOUT background content (notebooks)
--------------------------------------------------------
    # Just insert the annotation PDF directly
    rmc_pdf_src.insert_pdf(svg_pdf, start_at=page_idx)
    rmc_pdf_src.delete_page(page_idx + 1)

STEP 7: Apply smart highlights as PDF annotations
-------------------------------------------------
    for highlight in ann_data["highlights"]:
        apply_smart_highlight(rmc_pdf_src[page_idx], highlight, highlights_x_translation)
""")
    
    print()
    print("# 3. KEY CONSTANTS AND FORMULAS")
    print("-" * 40)
    print("""
SCALE = 0.3185840707964602  # = 445/1404 (PDF export width / device width)
SCREEN_WIDTH = 1404         # Device width in pixels
SCREEN_HEIGHT = 1872        # Device height in pixels
X_SHIFT = 223.0             # ≈ SCREEN_WIDTH * SCALE / 2

# Coordinate transforms (device -> PDF units)
xx(device_x) = device_x * SCALE
yy(device_y) = device_y * SCALE

# ReMarkable uses center-top origin:
#   X: -702 to +702 (center = 0)
#   Y: 0 to 1872 (top = 0)
#
# After scaling:
#   X: -223.65 to +223.65
#   Y: 0 to 596.39
""")
    
    print()
    print("# 4. SMART HIGHLIGHT ALGORITHM")
    print("-" * 40)
    print("""
Smart highlights are PDF highlight annotations placed over text.

def apply_smart_highlight(page, highlight, x_translation):
    # Get color from PenColor enum
    highlight_color = get_highlight_color(highlight.color)
    
    for rectangle in highlight.rectangles:
        x, y, w, h = rectangle.x, rectangle.y, rectangle.w, rectangle.h
        
        # Rectangles are already in PDF coordinates (scaled via xx/yy)
        # x_translation positions them relative to reMarkable's (0,0) at center-top
        rect = Rect(
            (x + x_translation, y),
            (x + x_translation + w, y + h)
        )
        
        # Add as PDF highlight annotation
        annot = page.add_highlight_annot(quads=rect)
        annot.set_colors(stroke=highlight_color)
        annot.set_opacity(0.3)
        annot.update()
""")
    
    print()
    print("# 5. GO REIMPLEMENTATION CHECKLIST")
    print("-" * 40)
    print("""
To reimplement in Go, you need:

1. V6 .rm parser:
   - Parse scene tree blocks (use rmscene as reference)
   - Extract Line elements (points with x,y,pressure,etc)
   - Extract GlyphRange elements (highlights with rectangles)
   - Extract RootTextBlock (typed text)

2. Coordinate transformation:
   - Scale all coordinates by SCALE = 0.3185840707964602
   - Handle center-top origin (X can be negative)

3. PDF rendering:
   - Use a PDF library that can:
     * Create new PDFs with strokes/paths
     * Merge PDFs (overlay one on another)
     * Add highlight annotations
   - Consider: pdfcpu, unipdf, or gofpdf

4. Bounding box calculation:
   - Walk scene tree recursively
   - Track min/max X/Y across all Line points
   - Handle anchor offsets for text

5. Page positioning:
   - Compute canvas size as max of annotation and background
   - Center smaller content within larger canvas
   - Track x_translation for highlight positioning
""")

if __name__ == "__main__":
    main()

