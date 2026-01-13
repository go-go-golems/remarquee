#!/usr/bin/env python3
"""
extract_rmc_coords.py - Extract coordinate transformation constants and functions from rmc library.

This script documents the exact algorithms used by remarks/rmc to convert reMarkable 
device coordinates to PDF coordinates. Essential for Go reimplementation.

Run from the remarks project directory:
    cd /path/to/remarks && poetry run python /path/to/this/script.py
"""

import inspect

def main():
    from rmc.exporters import svg
    
    print("=" * 80)
    print("RMC COORDINATE TRANSFORMATION ANALYSIS")
    print("=" * 80)
    print()
    
    # Key constants
    print("# 1. CONSTANTS")
    print("-" * 40)
    constants = ['SCALE', 'X_SHIFT', 'SCREEN_WIDTH', 'SCREEN_HEIGHT', 'TEXT_TOP_Y', 'LINE_HEIGHTS']
    for name in constants:
        if hasattr(svg, name):
            val = getattr(svg, name)
            print(f"{name} = {val}")
        else:
            print(f"{name} = <not found>")
    print()
    
    # Coordinate transform functions
    print("# 2. COORDINATE TRANSFORM FUNCTIONS")
    print("-" * 40)
    
    print("\n## xx (horizontal coordinate transform):")
    print(inspect.getsource(svg.xx))
    
    print("\n## yy (vertical coordinate transform):")
    print(inspect.getsource(svg.yy))
    
    print("\n## scale (underlying scale function):")
    if hasattr(svg, 'scale'):
        print(inspect.getsource(svg.scale))
    else:
        # xx and yy might BE the scale function
        print("# Note: xx and yy ARE the scale function (same implementation)")
    print()
    
    # Anchor and bounding box
    print("# 3. ANCHOR AND BOUNDING BOX FUNCTIONS")
    print("-" * 40)
    
    print("\n## get_anchor:")
    print(inspect.getsource(svg.get_anchor))
    
    print("\n## build_anchor_pos:")
    print(inspect.getsource(svg.build_anchor_pos))
    
    print("\n## get_bounding_box:")
    print(inspect.getsource(svg.get_bounding_box))
    print()
    
    # Show what the transforms actually do with examples
    print("# 4. EXAMPLE COORDINATE TRANSFORMATIONS")
    print("-" * 40)
    
    # ReMarkable coordinate system:
    # - X: center is 0, range roughly -702 to +702 (total width 1404)
    # - Y: top is 0, range 0 to 1872
    
    test_coords = [
        (0, 0, "center-top"),
        (-702, 0, "left-top"),
        (702, 0, "right-top"),
        (0, 1872, "center-bottom"),
        (-702, 1872, "left-bottom"),
        (702, 1872, "right-bottom"),
    ]
    
    print("\nReMarkable coord -> PDF coord (using xx/yy):")
    print(f"{'RM_X':>8} {'RM_Y':>8} {'PDF_X':>10} {'PDF_Y':>10}  Description")
    for rm_x, rm_y, desc in test_coords:
        pdf_x = svg.xx(rm_x)
        pdf_y = svg.yy(rm_y)
        print(f"{rm_x:>8} {rm_y:>8} {pdf_x:>10.2f} {pdf_y:>10.2f}  {desc}")
    
    print()
    print("# 5. INTERPRETATION FOR GO REIMPLEMENTATION")
    print("-" * 40)
    print("""
The coordinate system works as follows:

1. ReMarkable Device Coordinates:
   - Origin: CENTER-TOP of the page
   - X range: approximately -702 to +702 (center = 0)
   - Y range: 0 (top) to 1872 (bottom)
   - Device dimensions: 1404 x 1872 pixels

2. PDF/SVG Output Coordinates:
   - The xx() and yy() functions scale device units to output units
   - SCALE factor converts device pixels to output units
   - X_SHIFT may be used to translate the origin

3. To reimplement in Go:
   - xx(device_x) = device_x * SCALE
   - yy(device_y) = device_y * SCALE
   - The SCALE factor is the key: it converts from device units to PDF points

4. Bounding box calculation:
   - Walk the scene tree to find all Line points
   - Track min/max X and Y across all children
   - Use anchor positions for text positioning
""")

if __name__ == "__main__":
    main()

