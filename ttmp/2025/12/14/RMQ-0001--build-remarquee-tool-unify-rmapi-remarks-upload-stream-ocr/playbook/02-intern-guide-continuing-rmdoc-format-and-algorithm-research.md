---
Title: 'Intern guide: continuing .rmdoc format and algorithm research'
Ticket: RMQ-0001
Status: active
Topics:
    - backend
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Step-by-step guide for continuing the .rmdoc format research, exploring rmc/rmscene source code, and preparing for Go reimplementation"
LastUpdated: 2025-12-15T00:50:00-05:00
---

# Intern guide: continuing .rmdoc format and algorithm research

## Purpose

This playbook guides you through continuing the `.rmdoc` format and algorithm research that was started for RMQ-0001. You'll explore the `rmc` and `rmscene` Python libraries to understand how they parse and render reMarkable annotation files, with the goal of preparing for a Go reimplementation.

By the end of this playbook, you'll understand:
- How V6 `.rm` files are parsed into scene trees
- How coordinate transformations work
- How strokes are rendered to PDF/SVG
- What needs to be ported to Go

## Environment Assumptions

You're working in the RMQ-0001 workspace with:

- Python 3.11+ with `pyenv`
- `poetry` for dependency management
- Access to the cloned repositories:
  - `~/code/others/rmc` (converter/renderer)
  - `~/code/others/rmscene` (V6 parser)
  - This workspace has `remarks/`, `rmapi/`, etc.

## Prerequisites

Before starting, read these documents in order:

1. **Analysis doc** (read on reMarkable or locally):
   ```bash
   cat remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/analysis/01-deep-dive-rmdoc-format-container-layout-parsing-png-rendering.md
   ```
   This gives you the big picture of `.rmdoc` layout and the two parsing pipelines.

2. **Earlier remarks reference** (for context):
   ```bash
   cat remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/reference/02-remarks-package-analysis-parsing-conversion-output-formats.md
   ```

3. **Diary** (to see what was already explored):
   ```bash
   cat remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/reference/06-diary-rmdoc-format-analysis-and-go-reimplementation-prep.md
   ```

## Phase 1: Explore the rmc library (coordinate transforms and rendering)

### Step 1.1: Understand the coordinate transform constants

The `rmc` library defines the coordinate transformation from reMarkable device space to PDF space.

**Key file**: `~/code/others/rmc/src/rmc/exporters/svg.py`

```bash
# Read the constants section
cd ~/code/others/rmc
head -60 src/rmc/exporters/svg.py
```

**What to look for:**
- `SCALE = 72.0 / SCREEN_DPI` (line ~26)
- `SCREEN_WIDTH = 1404` and `SCREEN_HEIGHT = 1872`
- `X_SHIFT = PAGE_WIDTH_PT // 2`
- `def scale(screen_unit)` and how `xx` and `yy` are aliases

**Exercise**: Calculate what `SCALE` evaluates to and verify it matches the documented value (0.3185840707964602).

```python
# Run this in the rmc directory
cd ~/code/others/rmc
poetry install
poetry run python -c "from rmc.exporters.svg import SCALE, SCREEN_WIDTH, SCREEN_HEIGHT, X_SHIFT; print(f'SCALE={SCALE}'); print(f'SCREEN_WIDTH={SCREEN_WIDTH}'); print(f'SCREEN_HEIGHT={SCREEN_HEIGHT}'); print(f'X_SHIFT={X_SHIFT}')"
```

### Step 1.2: Trace the bounding box calculation

**Key function**: `get_bounding_box()` in `svg.py`

```bash
# Read the bounding box function
cd ~/code/others/rmc
grep -A 30 "^def get_bounding_box" src/rmc/exporters/svg.py
```

**What to understand:**
- It walks the scene tree recursively
- For `Group` items: applies anchor offsets and recurses
- For `Line` items: tracks min/max X/Y across all points
- Returns `(x_min, x_max, y_min, y_max)` in device coordinates (not yet scaled)

**Exercise**: Draw a diagram showing how anchor offsets propagate through nested groups.

### Step 1.3: Study the PDF rendering pipeline

**Key file**: `~/code/others/rmc/src/rmc/exporters/pdf.py`

```bash
# Read the rm_to_pdf function
cd ~/code/others/rmc
head -100 src/rmc/exporters/pdf.py
```

**What to look for:**
- How it uses `cairosvg` to convert SVG → PDF
- The intermediate step: `.rm` → SVG → PDF (not direct)
- Why this approach (leverage Cairo's mature rendering)

**Key insight**: `rmc` doesn't render strokes directly to PDF — it generates SVG first, then uses CairoSVG to convert to PDF. This is important for Go reimplementation: you could follow the same pattern or render directly.

## Phase 2: Explore the rmscene library (V6 binary parser)

### Step 2.1: Understand the scene tree structure

**Key file**: `~/code/others/rmscene/src/rmscene/scene_tree.py`

```bash
# Read the SceneTree class
cd ~/code/others/rmscene
cat src/rmscene/scene_tree.py
```

**What to understand:**
- `SceneTree` has `root` (a Group) and `root_text` (optional Text)
- `build_tree(tree, blocks)` constructs the tree from parsed blocks
- The tree is hierarchical: Root → Layers (Groups) → Lines/Text/Highlights

**Exercise**: Sketch the tree structure for a simple annotated page (one layer, two strokes, one highlight).

### Step 2.2: Study the scene item types

**Key file**: `~/code/others/rmscene/src/rmscene/scene_items.py`

```bash
# Read the scene item definitions
cd ~/code/others/rmscene
head -150 src/rmscene/scene_items.py
```

**What to look for:**
- `@dataclass` definitions for `Group`, `Line`, `GlyphRange`, `Text`
- `PenColor` enum (lines 58-104) — this is the color mapping you'll need in Go
- `HARDCODED_COLORMAP` (lines 107-121) — RGB values for each PenColor
- `Point` structure (has x, y, pressure, tilt, width)

**Exercise**: Copy the `HARDCODED_COLORMAP` into a new analysis script and format it as a Go map.

### Step 2.3: Understand the binary block format

**Key files**: 
- `~/code/others/rmscene/src/rmscene/tagged_block_reader.py` (reads blocks)
- `~/code/others/rmscene/src/rmscene/scene_stream.py` (defines block types)

```bash
# Read the block reader
cd ~/code/others/rmscene
head -200 src/rmscene/tagged_block_reader.py
```

**What to understand:**
- `.rm` files are sequences of "tagged blocks"
- Each block has a type tag (byte) and length-prefixed data
- Block types include: `SceneLineItemBlock`, `SceneGlyphItemRangeBlock`, `RootTextBlock`, etc.
- The reader uses a state machine to parse nested structures

**Warning**: This is the hardest part to port to Go. The binary format is complex with variable-length encoding, nested sub-blocks, and CRDT sequences.

### Step 2.4: Trace a complete parse

**Exercise**: Use the analysis script to trace parsing of a real `.rm` file.

Create `scripts/trace_rm_parse.py`:

```python
#!/usr/bin/env python3
"""Trace the parsing of a .rm file to understand the block structure."""

import sys
from rmscene import read_blocks, SceneTree, build_tree
from rmscene.scene_items import Line, GlyphRange, Text

def main(rm_file_path):
    print(f"Parsing: {rm_file_path}")
    print("=" * 80)
    
    with open(rm_file_path, "rb") as f:
        # Read all blocks
        blocks = list(read_blocks(f))
        print(f"\nTotal blocks: {len(blocks)}")
        
        # Show block types
        print("\nBlock types:")
        for i, block in enumerate(blocks[:20]):  # First 20 blocks
            print(f"  {i}: {type(block).__name__}")
        
        # Build tree
        tree = SceneTree()
        build_tree(tree, blocks)
        
        print(f"\nScene tree built:")
        print(f"  Root children: {len(tree.root.children)}")
        print(f"  Has root text: {tree.root_text is not None}")
        
        # Walk tree and count elements
        lines = []
        highlights = []
        for item in tree.walk():
            if isinstance(item, Line):
                lines.append(item)
            elif isinstance(item, GlyphRange):
                highlights.append(item)
        
        print(f"\nExtracted elements:")
        print(f"  Lines (strokes): {len(lines)}")
        print(f"  Highlights: {len(highlights)}")
        
        # Show first line details
        if lines:
            line = lines[0]
            print(f"\nFirst line details:")
            print(f"  Color: {line.color}")
            print(f"  Tool: {line.tool}")
            print(f"  Thickness: {line.thickness_scale}")
            print(f"  Points: {len(line.points)}")
            if line.points:
                p = line.points[0]
                print(f"  First point: x={p.x:.2f}, y={p.y:.2f}, pressure={p.pressure:.2f}")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python trace_rm_parse.py <path-to-.rm-file>")
        sys.exit(1)
    main(sys.argv[1])
```

Run it:

```bash
cd /home/manuel/workspaces/2025-12-14/build-remarquee-tool
python scripts/trace_rm_parse.py remarks/tests/in/copies\ of\ different\ pages.rmdoc
# (You'll need to unzip first and point to a specific .rm file)
```

## Phase 3: Analyze the PDF merge algorithm in detail

### Step 3.1: Run the existing analysis scripts

We already created two analysis scripts. Run them to see the output:

```bash
cd /home/manuel/workspaces/2025-12-14/build-remarquee-tool

# Extract coordinate constants
cd remarks && poetry run python ../remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/scripts/extract_rmc_coords.py

# Analyze merge algorithm
poetry run python ../remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/scripts/analyze_remarks_merge.py
```

### Step 3.2: Trace the merge with a real document

**Exercise**: Process a test document and observe the intermediate files.

```bash
cd /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarks

# Process a test document
testdir=$(mktemp -d)
unzip -q "tests/in/copies of different pages.rmdoc" -d "$testdir"
outdir=$(mktemp -d)

# Run remarks with debug logging
poetry run python -c "
import logging
logging.basicConfig(level=logging.DEBUG)
from remarks import run_remarks
import pathlib
run_remarks(pathlib.Path('$testdir'), pathlib.Path('$outdir'))
"

# Examine outputs
ls -lh "$outdir"
echo "Temp dir: $testdir (inspect the extracted files)"
echo "Output dir: $outdir (inspect the generated PDFs)"

# Don't cleanup yet - explore the files first
# rm -rf "$testdir" "$outdir"
```

### Step 3.3: Understand the positioning formulas

The trickiest part is the `xTranslation` calculation. Study this code:

```bash
cd ~/code/others
grep -A 20 "w_svg > w_bg" remarks/remarks/remarks.py
```

**What to understand:**
- When annotations are wider than background: background gets shifted right
- When background is wider than annotations: annotations get centered
- `xTranslation` tells smart highlights where the "center" is
- The formulas account for the center-top origin

**Exercise**: Work through the math with concrete numbers:
- Background: 500pt wide
- Annotations: 300pt wide
- What are `xSvg`, `xBg`, and `xTranslation`?

## Phase 4: Map the Go reimplementation requirements

### Step 4.1: Identify what exists in rmapi already

```bash
cd /home/manuel/workspaces/2025-12-14/build-remarquee-tool

# See what rmapi can already do
ls rmapi/encoding/rm/    # V3/V5 parser
ls rmapi/annotations/    # PDF rendering
ls rmapi/archive/        # .rmdoc ZIP handling
```

**Key findings from earlier analysis:**
- `rmapi/encoding/rm` parses V3/V5 `.rm` files (not V6)
- `rmapi/annotations/pdf.go` renders strokes to PDF using UniDoc
- `rmapi/archive` handles `.rmdoc` ZIP but doesn't understand `cPages`

### Step 4.2: Create a gap analysis document

Create `scripts/go_reimplementation_gaps.md`:

```markdown
# Go Reimplementation Gap Analysis

## What we have (in rmapi)

- ✅ ZIP extraction (`archive.Unpack`)
- ✅ V3/V5 .rm parser (`encoding/rm`)
- ✅ PDF stroke rendering (`annotations.PdfGenerator`)
- ✅ PDF library (UniDoc)
- ✅ Legacy .content parsing (`archive.Content`)

## What we need (from remarks/rmc/rmscene)

- ❌ V6 .rm parser (scene tree format)
- ❌ cPages .content parsing
- ❌ Coordinate transform constants (SCALE, etc.)
- ❌ Bounding box calculation with anchors
- ❌ PDF merge algorithm (positioning logic)
- ❌ Smart highlight application
- ❌ GlyphRange extraction (highlight rectangles + text)
- ❌ Typed text extraction (RootTextBlock)

## Porting strategy options

### Option 1: Port rmscene to Go
- **Effort**: High (~2-3 weeks)
- **Pros**: Full control, no Python dependency
- **Cons**: Complex binary format, ongoing maintenance

### Option 2: Call Python from Go
- **Effort**: Low (~2-3 days)
- **Pros**: Reuse existing code, faster to market
- **Cons**: Python runtime dependency, subprocess overhead

### Option 3: Hybrid approach
- **Effort**: Medium (~1 week)
- **Pros**: Port simple parts (coords, merge), call Python for parsing
- **Cons**: Still has Python dependency

## Recommended: Start with Option 2, migrate to Option 1

1. Build Go wrapper around `remarks` (subprocess)
2. Test end-to-end with real documents
3. Identify performance bottlenecks
4. Port critical path to Go incrementally
```

Save this to the ticket:

```bash
cd /home/manuel/workspaces/2025-12-14/build-remarquee-tool
cat > remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/scripts/go_reimplementation_gaps.md << 'EOF'
[paste the content above]
EOF
```

### Step 4.3: Extract the color mapping

The `HARDCODED_COLORMAP` in `rmscene` is essential for rendering highlights correctly.

Create `scripts/extract_color_map.py`:

```python
#!/usr/bin/env python3
"""Extract the PenColor → RGBA mapping for Go reimplementation."""

import sys
sys.path.insert(0, '/home/manuel/workspaces/2025-12-14/build-remarquee-tool/rmscene/src')

from rmscene.scene_items import PenColor, HARDCODED_COLORMAP

print("// PenColor enum for Go")
print("type PenColor int")
print()
print("const (")
for color in PenColor:
    print(f"    {color.name} PenColor = {color.value}")
print(")")
print()

print("// RGBA color map")
print("var HardcodedColorMap = map[PenColor][4]uint8{")
for rgba, pen_color in HARDCODED_COLORMAP.items():
    r, g, b, a = rgba
    print(f"    {pen_color.name}: {{{r}, {g}, {b}, {a}}},")
print("}")
print()

print("// RGB normalized (for PDF annotations)")
print("var HighlightColors = map[PenColor][3]float64{")
for rgba, pen_color in HARDCODED_COLORMAP.items():
    r, g, b, a = rgba
    print(f"    {pen_color.name}: {{{r/255:.4f}, {g/255:.4f}, {b/255:.4f}}},")
print("}")
```

Run it:

```bash
cd /home/manuel/workspaces/2025-12-14/build-remarquee-tool
python remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/scripts/extract_color_map.py > remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/scripts/color_map.go
```

## Phase 5: Test with real documents

### Step 5.1: Process a document and inspect intermediate files

```bash
cd /home/manuel/workspaces/2025-12-14/build-remarquee-tool

# Extract a test .rmdoc
testdir=$(mktemp -d)
unzip -q "remarks/tests/in/copies of different pages.rmdoc" -d "$testdir"

# List what's inside
echo "=== Document structure ==="
ls -lh "$testdir"

# Read the .content file
echo -e "\n=== .content (first 50 lines) ==="
find "$testdir" -name "*.content" -exec python -m json.tool {} \; | head -50

# Read the .metadata file
echo -e "\n=== .metadata ==="
find "$testdir" -name "*.metadata" -exec python -m json.tool {} \;

# List .rm files
echo -e "\n=== .rm files ==="
find "$testdir" -name "*.rm" -ls

# Process with remarks
outdir=$(mktemp -d)
cd remarks
poetry run remarks "$testdir" "$outdir"

# Examine outputs
echo -e "\n=== Generated files ==="
ls -lh "$outdir"

# Keep these directories for inspection
echo -e "\nTest dir: $testdir"
echo "Output dir: $outdir"
echo "(Inspect these before cleanup)"
```

### Step 5.2: Compare rmapi vs remarks output

```bash
cd /home/manuel/workspaces/2025-12-14/build-remarquee-tool

# Try processing with rmapi's annotation generator
# (This will likely fail on modern .rmdoc due to cPages)
cd rmapi
# Build rmapi if needed
go build -o rmapi-bin ./cmd/rmapi

# Try to generate annotated PDF
# ./rmapi-bin ... (document the command and result)
```

**Expected result**: rmapi's annotation generator may fail on modern `.rmdoc` files. Document exactly how it fails — this is the gap we need to fill.

## Phase 6: Document your findings

### Step 6.1: Update the diary

```bash
cd /home/manuel/workspaces/2025-12-14/build-remarquee-tool

# Edit the diary
docmgr doc list --ticket RMQ-0001 --doc-type reference | grep -i diary
# Then manually add a new step to the diary document
```

**What to include in your diary entry:**
- What you explored (which files, which functions)
- What you learned (key insights about the format/algorithms)
- What surprised you (unexpected complexity, clever solutions)
- What questions remain (ambiguities, edge cases)

### Step 6.2: Create analysis scripts for anything new you discover

Follow the pattern of existing scripts:

```bash
ls remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/scripts/
# extract_rmc_coords.py
# analyze_remarks_merge.py
```

If you discover something not yet documented (e.g., how text anchors work, how pressure affects stroke width), create a new script that extracts and demonstrates it.

### Step 6.3: Update the deep-dive analysis doc

If you find gaps or corrections:

```bash
cd /home/manuel/workspaces/2025-12-14/build-remarquee-tool

# Read the current analysis
cat remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/analysis/01-deep-dive-rmdoc-format-container-layout-parsing-png-rendering.md

# Add new sections or correct existing ones
# Then relate any new files and update changelog
```

## Phase 7: Prepare for Go implementation

### Step 7.1: Choose a porting strategy

Review the gap analysis (Step 4.2) and make a recommendation:

- **If time is critical**: Start with subprocess wrapper (Option 2)
- **If performance matters**: Plan to port rmscene (Option 1)
- **If unsure**: Prototype both and measure

### Step 7.2: Create a Go implementation plan

Create `playbook/03-go-implementation-plan.md` with:

1. **Phase 1**: Extend `rmapi/archive` to parse `cPages`
2. **Phase 2**: Add coordinate transform constants
3. **Phase 3**: Implement PDF merge algorithm
4. **Phase 4**: Either port V6 parser OR wrap Python
5. **Phase 5**: Integration testing

Use `docmgr doc add` to create this document.

## Key Files Reference

### In this workspace

```
remarks/remarks/
├── remarks.py              # Main processing loop
├── Document.py             # Document model
├── utils.py                # Page ordering, cPages parsing
├── conversion/
│   └── parsing.py          # V6 .rm parsing (uses rmscene)
└── output/
    └── PdfFile.py          # Smart highlight application

rmapi/
├── archive/
│   ├── reader.go           # .rmdoc ZIP reader
│   ├── file.go             # Content/Metadata structs
│   └── zipdoc.go           # ZIP creation
├── encoding/rm/
│   └── rm.go               # V3/V5 .rm parser
└── annotations/
    └── pdf.go              # PDF stroke rendering
```

### In ~/code/others

```
rmc/src/rmc/exporters/
├── svg.py                  # Coordinate transforms, bounding box
├── pdf.py                  # rm_to_pdf function
└── writing_tools.py        # Pen/brush definitions

rmscene/src/rmscene/
├── scene_tree.py           # SceneTree, build_tree
├── scene_items.py          # Line, GlyphRange, Text, PenColor
├── scene_stream.py         # Block type definitions
├── tagged_block_reader.py  # Binary parser
└── text.py                 # Text document handling
```

## Success Criteria

You've successfully continued the research when you can:

1. ✅ Explain how V6 `.rm` files are structured (blocks → tree → items)
2. ✅ Trace coordinate transformations from device space to PDF space
3. ✅ Describe the PDF merge positioning algorithm in your own words
4. ✅ Identify which parts of rmscene are hardest to port to Go
5. ✅ Recommend a porting strategy with rationale
6. ✅ Have created at least 2 new analysis scripts
7. ✅ Updated the diary with your findings

## Common Pitfalls

- **Don't skip reading the existing analysis docs first** — they contain context you'll need
- **Don't try to understand everything at once** — focus on one algorithm at a time
- **Don't assume the Python code is optimal** — it evolved over time, some parts are hacky
- **Don't forget to document as you go** — future you (or the next person) will thank you

## Getting Help

If you get stuck:

1. Check the existing analysis docs (they answer most questions)
2. Run the analysis scripts to see concrete examples
3. Look at the test files in `remarks/tests/` and `rmscene/tests/`
4. Read the commit history in rmc/rmscene to understand why things changed
5. Ask specific questions with concrete examples (file paths, error messages, expected vs actual output)

## Next Steps After This Playbook

Once you've completed this research:

1. Present your findings (gap analysis, porting recommendation)
2. Get approval on the porting strategy
3. Create detailed implementation tasks in docmgr
4. Begin implementation (likely starting with `cPages` support in Go)

## Related Documents

- **Analysis**: `analysis/01-deep-dive-rmdoc-format-container-layout-parsing-png-rendering.md`
- **Reference**: `reference/02-remarks-package-analysis-parsing-conversion-output-formats.md`
- **Diary**: `reference/06-diary-rmdoc-format-analysis-and-go-reimplementation-prep.md`
- **Scripts**: `scripts/extract_rmc_coords.py`, `scripts/analyze_remarks_merge.py`
