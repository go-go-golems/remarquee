# Go Reimplementation Gap Analysis

**Important**: remarquee needs to support **both V3/V5 and V6 formats**. Testing revealed that PDF-based documents (especially older ones) use V3/V5 format, while newer notebooks use V6 format.

## What we have (in rmapi)

- ✅ ZIP extraction (`archive.Unpack`)
- ✅ PDF library (UniDoc)
- ✅ **V3/V5 .rm parser** (`encoding/rm`) - **Can be reused for V3/V5 support**
- ✅ **PDF stroke rendering** (`annotations.PdfGenerator`) - **Works for V3/V5, needs V6 support**
- ✅ **Legacy .content parsing** (`archive.Content`) - **Works for V3/V5, needs cPages support**

## What we need (from remarks/rmc/rmscene)

- ❌ **V6 .rm parser** (scene tree format) - **CRITICAL: Required for V6 documents**
- ❌ **cPages .content parsing** - **CRITICAL: Required for V6 .rmdoc**
- ❌ **Format detection** - **Determine V3/V5 vs V6 automatically**
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

## Detailed Gap Analysis

### 1. Format Detection and Dual Parser Support

**Current state**: 
- `rmapi/encoding/rm` handles V3/V5 format (simple stroke arrays) ✅
- No V6 parser exists ❌

**What's needed**:
- **Format detection**: Check `.content` file for `cPages` (V6) vs `Pages`/`pageCount` (V3/V5)
- **V6 parser**: New implementation (see below)
- **Unified interface**: Single API that handles both formats

### 2. V6 .rm Parser

**Current state**: No V6 parser exists in rmapi.

**What's needed**:
- Tagged block reader (`tagged_block_reader.py`)
- CRDT sequence handling (`crdt_sequence.py`)
- Scene tree builder (`scene_tree.py`)
- Block type parsers (SceneLineItemBlock, SceneGlyphItemRangeBlock, RootTextBlock, etc.)

**Complexity**: Very High
- Variable-length encoding
- Nested sub-blocks
- CRDT sequences for conflict resolution
- State machine for parsing

### 2. cPages .content Parsing

**Current state**: `archive.Content` struct has:
- `Pages []string` (legacy page IDs)
- `PageCount int`
- No `cPages` field

**What's needed**:
- `cPages.pages[]` array parsing
- Page UUID → index mapping
- `redir.value` handling (source PDF page mapping)
- `deleted.value` handling (soft-delete)

**Complexity**: Low-Medium
- JSON parsing is straightforward
- Need to handle optional fields
- Need to reconstruct page ordering from UUIDs

### 3. Coordinate Transform Constants

**Current state**: Not present in rmapi

**What's needed**:
```go
const (
    ScreenDPI = 226
    Scale = 72.0 / ScreenDPI  // = 0.3185840707964602
    ScreenWidth = 1404
    ScreenHeight = 1872
    PageWidthPT = ScreenWidth * Scale  // = 447.29
    PageHeightPT = ScreenHeight * Scale // = 596.39
    XShift = PageWidthPT / 2  // = 223.0
)
```

**Complexity**: Low
- Just constants and simple functions
- Already extracted in `scripts/extract_rmc_coords.py`

### 4. Bounding Box Calculation

**Current state**: Not present in rmapi

**What's needed**:
- Recursive tree walker
- Anchor offset calculation (`build_anchor_pos`)
- Min/max tracking across points
- Group anchor handling

**Complexity**: Medium
- Algorithm is straightforward
- Need to understand anchor system
- Coordinate space transformations

### 5. PDF Merge Algorithm

**Current state**: `annotations.PdfGenerator` renders strokes directly, no merge logic

**What's needed**:
- Background PDF extraction/creation
- Annotation PDF generation (via rmc or direct)
- Bounding box calculation
- Positioning logic (center-top origin handling)
- PDF page merging (UniDoc can do this)

**Complexity**: Medium-High
- Positioning formulas are subtle
- Need to handle rotation
- Need to handle different page sizes

### 6. Smart Highlight Application

**Current state**: Not present in rmapi

**What's needed**:
- GlyphRange extraction from scene tree
- Rectangle extraction
- PDF highlight annotation creation
- Color mapping (`HARDCODED_COLORMAP`)
- X translation calculation

**Complexity**: Medium
- UniDoc supports PDF annotations
- Need color mapping (already extracted)
- Need to understand highlight positioning

### 7. GlyphRange Extraction

**Current state**: Not present in rmapi

**What's needed**:
- Parse `SceneGlyphItemRangeBlock`
- Extract rectangles
- Extract associated text
- Handle text highlighting

**Complexity**: Medium
- Depends on V6 parser
- Text extraction is straightforward once parsed

### 8. Typed Text Extraction

**Current state**: Not present in rmapi

**What's needed**:
- Parse `RootTextBlock`
- Extract text document structure
- Handle paragraph styles
- Extract text content

**Complexity**: Medium
- Depends on V6 parser
- Text document structure is well-defined

## Format Distribution Findings

**Testing results from reMarkable tablet:**
- **V6 documents**: Notebooks, newer documents (Journal, TODO, Quick sheets, etc.)
- **V3/V5 documents**: PDF-based documents, especially older ones (e.g., "Artificial Intelligence Planning Systems...")

**Conclusion**: Both formats are in active use. PDF documents with annotations often use V3/V5, while notebooks use V6.

## Implementation Priority

1. **Phase 1**: Format detection (determine V3/V5 vs V6 automatically)
2. **Phase 2**: cPages parsing (enables V6 page ordering)
3. **Phase 3**: Coordinate transforms (enables rendering)
4. **Phase 4**: PDF merge algorithm (core functionality)
5. **Phase 5**: V6 parser OR Python wrapper (hardest part)
6. **Phase 6**: Smart highlights (nice-to-have)
7. **Phase 7**: Typed text extraction (nice-to-have)

## Testing Strategy

1. Use existing test fixtures from `remarks/tests/in/`
2. Compare output PDFs pixel-by-pixel
3. Test edge cases: rotated pages, inserted pages, duplicated pages
4. Performance benchmarks: Python vs Go

## References

- Analysis doc: `analysis/01-deep-dive-rmdoc-format-container-layout-parsing-png-rendering.md`
- Coordinate constants: `scripts/extract_rmc_coords.py`
- Color mapping: `scripts/color_map.go`
- Merge algorithm: `scripts/analyze_remarks_merge.py`

