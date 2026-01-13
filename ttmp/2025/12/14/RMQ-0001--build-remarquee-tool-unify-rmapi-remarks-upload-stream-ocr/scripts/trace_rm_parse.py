#!/usr/bin/env python3
"""
trace_rm_parse.py - Trace the parsing of a .rm file to understand the block structure.

This script demonstrates how rmscene parses V6 .rm files into a scene tree,
showing block types, element counts, and structure.

Usage:
    poetry run python trace_rm_parse.py <path-to-.rm-file>
"""

import sys
import pathlib

def main(rm_file_path):
    # Add rmscene to path if needed
    rmscene_path = pathlib.Path.home() / "workspaces/2025-12-14/build-remarquee-tool/rmscene/src"
    if rmscene_path.exists():
        sys.path.insert(0, str(rmscene_path))
    
    from rmscene import read_blocks, SceneTree, build_tree
    from rmscene.scene_items import Line, GlyphRange, Text, Group
    
    print(f"Parsing: {rm_file_path}")
    print("=" * 80)
    
    with open(rm_file_path, "rb") as f:
        # Read all blocks
        blocks = list(read_blocks(f))
        print(f"\nTotal blocks: {len(blocks)}")
        
        # Show block types
        print("\nBlock types (first 20):")
        for i, block in enumerate(blocks[:20]):
            print(f"  {i:3d}: {type(block).__name__}")
        
        if len(blocks) > 20:
            print(f"  ... and {len(blocks) - 20} more blocks")
        
        # Build tree
        tree = SceneTree()
        build_tree(tree, blocks)
        
        print(f"\nScene tree built:")
        root_children_count = sum(1 for _ in tree.root.children.values())
        print(f"  Root children: {root_children_count}")
        print(f"  Has root text: {tree.root_text is not None}")
        
        # Walk tree and count elements
        lines = []
        highlights = []
        groups = []
        for item in tree.walk():
            if isinstance(item, Line):
                lines.append(item)
            elif isinstance(item, GlyphRange):
                highlights.append(item)
            elif isinstance(item, Group):
                groups.append(item)
        
        print(f"\nExtracted elements:")
        print(f"  Groups (layers): {len(groups)}")
        print(f"  Lines (strokes): {len(lines)}")
        print(f"  Highlights: {len(highlights)}")
        
        # Show first line details
        if lines:
            line = lines[0]
            print(f"\nFirst line details:")
            print(f"  Color: {line.color} ({line.color.name if hasattr(line.color, 'name') else 'N/A'})")
            print(f"  Tool: {line.tool} ({line.tool.name if hasattr(line.tool, 'name') else 'N/A'})")
            print(f"  Thickness scale: {line.thickness_scale}")
            print(f"  Points: {len(line.points)}")
            if line.points:
                p = line.points[0]
                print(f"  First point: x={p.x:.2f}, y={p.y:.2f}, pressure={p.pressure:.2f}")
                print(f"  Coordinate range:")
                xs = [p.x for p in line.points]
                ys = [p.y for p in line.points]
                print(f"    X: {min(xs):.2f} to {max(xs):.2f}")
                print(f"    Y: {min(ys):.2f} to {max(ys):.2f}")
        
        # Show first highlight details
        if highlights:
            hl = highlights[0]
            print(f"\nFirst highlight details:")
            print(f"  Color: {hl.color} ({hl.color.name if hasattr(hl.color, 'name') else 'N/A'})")
            print(f"  Text: {hl.text[:50]}..." if len(hl.text) > 50 else f"  Text: {hl.text}")
            print(f"  Start: {hl.start}, Length: {hl.length}")
            print(f"  Rectangles: {len(hl.rectangles)}")
            if hl.rectangles:
                r = hl.rectangles[0]
                print(f"  First rect: x={r.x:.2f}, y={r.y:.2f}, w={r.w:.2f}, h={r.h:.2f}")
        
        # Show text details
        if tree.root_text is not None:
            from rmscene.text import TextDocument
            doc = TextDocument.from_scene_item(tree.root_text)
            print(f"\nTyped text:")
            print(f"  Position: x={tree.root_text.pos_x:.2f}, y={tree.root_text.pos_y:.2f}")
            print(f"  Width: {tree.root_text.width:.2f}")
            print(f"  Paragraphs: {len(doc.contents)}")
            for i, para in enumerate(doc.contents[:3]):  # First 3 paragraphs
                text = str(para)[:60]
                print(f"  Para {i}: {para.style.name} - {text}...")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python trace_rm_parse.py <path-to-.rm-file>")
        print()
        print("Example:")
        print("  # Extract a .rmdoc first")
        print("  unzip -q 'remarks/tests/in/copies of different pages.rmdoc' -d /tmp/test")
        print("  # Then trace a specific .rm file")
        print("  python trace_rm_parse.py /tmp/test/<docUUID>/<pageUUID>.rm")
        sys.exit(1)
    
    main(sys.argv[1])

