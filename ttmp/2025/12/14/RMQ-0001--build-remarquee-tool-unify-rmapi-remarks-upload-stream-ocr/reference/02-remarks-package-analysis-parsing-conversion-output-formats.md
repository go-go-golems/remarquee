---
Title: remarks package analysis (parsing, conversion, output formats)
Ticket: RMQ-0001
Status: active
Topics:
    - backend
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarks/pyproject.toml
      Note: Package configuration - dependencies
    - Path: remarks/remarks/Document.py
      Note: Document model class - represents ReMarkable document with pages and metadata
    - Path: remarks/remarks/__main__.py
      Note: CLI entry point - argument parsing and main() function
    - Path: remarks/remarks/conversion/parsing.py
      Note: Parse .rm files - parse_rm_file()
    - Path: remarks/remarks/conversion/text.py
      Note: Text extraction from PDF - highlight extraction
    - Path: remarks/remarks/dimensions.py
      Note: Coordinate system - ReMarkable dimensions and conversions
    - Path: remarks/remarks/metadata.py
      Note: Metadata constants - file version and device type constants
    - Path: remarks/remarks/output/ObsidianMarkdownFile.py
      Note: Markdown output - Obsidian-compatible markdown generation with frontmatter
    - Path: remarks/remarks/output/PdfFile.py
      Note: PDF output - smart highlight application
    - Path: remarks/remarks/remarks.py
      Note: Main processing logic - run_remarks() and process_document() functions
    - Path: remarks/remarks/server.py
      Note: HTTP server - Flask API for processing documents
    - Path: remarks/remarks/utils.py
      Note: Utility functions - metadata reading
    - Path: remarks/remarks/warnings.py
      Note: Warning system - ScrybbleWarning class for PDF annotations
ExternalSources: []
Summary: 'Comprehensive analysis of remarks Python package: parsing ReMarkable .rm files, converting to PDF/Markdown, architecture, and developer usage guide'
LastUpdated: 2025-12-14T17:47:30.479035546-05:00
---


# remarks package analysis (parsing, conversion, output formats)

## Goal

This document provides a comprehensive technical analysis of the `remarks` Python package, covering:
- Package structure and organization
- Architecture and data flow
- Key classes, functions, and data structures
- ReMarkable file format parsing (.rm, .metadata, .content)
- Conversion pipeline (annotations → PDF/Markdown)
- Output format generation
- Developer usage guide

### remarks in the remarquee Ecosystem: The Content Intelligence Layer

While rmapi handles cloud communication (upload/download), remarks solves a fundamentally different problem: **understanding what's inside** those documents. This distinction is crucial for appreciating remarks' role in the remarquee ecosystem.

When you download a document from ReMarkable using rmapi, you get a `.rmdoc` file—essentially a ZIP archive containing binary `.rm` files. These files contain your annotations (highlights, handwriting, typed notes), but they're in a proprietary binary format that nothing else can read. This is where remarks becomes essential.

**The Challenge remarks Addresses:**

ReMarkable has brilliant hardware and software, but it operates in a closed ecosystem. Your annotations—the intellectual value you've created—are trapped in a format that only ReMarkable's software understands. You can't:
- Search your highlights in Obsidian or Notion
- Extract key insights for research papers
- Share annotated PDFs with colleagues (they see overlay images, not searchable text)
- Build automation around your notes (like generating summaries, flashcards, or knowledge graphs)

remarks unlocks this value by reverse-engineering the binary format and providing clean Python APIs to access annotation data. This makes remarks the **content intelligence layer** of remarquee—it transforms opaque binary data into structured, queryable, processable information.

**How remarks Complements rmapi:**

Think of the data flow:

```
Cloud (ReMarkable servers)
    ↓ rmapi downloads
Your Computer: document.rmdoc
    ↓ remarks extracts
Structured Data: {highlights: [...], text: [...], drawings: [...]}
    ↓ remarks converts
Usable Outputs: document.pdf (annotated), document.md (searchable)
    ↓ remarquee will add
AI Processing: OCR, summarization, knowledge extraction
```

remarks sits at the critical conversion point—from proprietary to open, from binary to structured, from locked to liberated. For the remarquee project, remarks provides the foundation for the planned OCR/LLM integration (via geppetto). You can't run OCR on binary `.rm` files, but you can run it on remarks-extracted text and highlights.

**Why Python for remarks (when rmapi is Go)?**

This is intentional—different tools for different jobs:
- **rmapi (Go)**: Network I/O, concurrency, protocol implementation → Go's strengths
- **remarks (Python)**: PDF manipulation, scientific computing, ML libraries → Python's strengths

The remarquee unified tool will need to bridge these (via subprocess calls or language bindings), but that's acceptable—each tool uses the right language for its problem domain.

## Context

### What is remarks?

`remarks` is a Python package that extracts annotations from ReMarkable notebook files and converts them to standard formats (PDF, Markdown, PNG, SVG). Think of it as a "translator" that takes ReMarkable's proprietary annotation format and converts it to something you can use in other tools.

### Why does it exist?

When you annotate a PDF or write notes on your ReMarkable tablet, your markings are stored in binary `.rm` files using a proprietary format. These files contain:
- **Pen strokes**: Your drawings and handwriting
- **Highlights**: Text you've highlighted
- **Typed text**: Text entered with the on-screen keyboard

Without `remarks`, these annotations are locked in ReMarkable's ecosystem. You can't:
- Search highlighted text in your note-taking app (Obsidian, Notion, etc.)
- Share annotated PDFs with proper highlights (they're overlay images)
- Extract notes for further processing

### What problem does it solve?

`remarks` solves three main problems:

1. **Format conversion**: `.rm` binary → PDF/Markdown/SVG
2. **Intelligent highlight extraction**: Recognizes highlighted text, not just rectangles
3. **Integration**: Generates Obsidian-compatible Markdown with metadata

**Real-world use case:**

```
You read a research paper on ReMarkable:
- Highlight key passages
- Add typed notes
- Draw diagrams

Download with rmapi:
- get paper.pdf → paper.rmdoc (ZIP with .rm files)

Process with remarks:
- paper.rmdoc → paper_remarks.pdf (PDF with rendered highlights)
- paper.rmdoc → paper_obsidian.md (Markdown with extracted highlights)

Now in Obsidian:
- Full-text search your highlights
- Link notes to other documents
- Export to other formats
```

### Key characteristics

- **Parses ReMarkable `.rm` files (V6)**: Only current format supported (pre-V6 deprecated)
- **Extracts multiple annotation types**: Highlights, typed text, drawings
- **Smart highlight positioning**: Handles coordinate transformation (center-top to bottom-left)
- **Obsidian-compatible output**: YAML frontmatter, proper formatting
- **Supports PDF, EPUB, notebooks**: All tablet document types
- **Uses external libraries**: `rmscene` (parsing), `rmc` (rendering), PyMuPDF (PDF)

### How it works (high-level)

1. **Discovery**: Find all `.metadata` files (one per document)
2. **Document creation**: Create `Document` object with pages, metadata
3. **Page iteration**: For each page with `.rm` file:
   - Parse `.rm` binary file
   - Extract highlights, text, drawings
   - Convert to PDF (merge with source PDF)
   - Collect highlights for Markdown
4. **Output generation**: Save PDF and Markdown files

**Dependencies explained:**

- **PyMuPDF (fitz)**: Industry-standard PDF library for Python (fast, feature-rich)
- **rmscene**: Specialized library for parsing ReMarkable's binary format (handles V6 scene tree)
- **rmc**: Companion library to rmscene (converts scene tree to PDF/SVG)
- **PyYAML**: Standard YAML library (for Obsidian frontmatter)
- **jinja2**: Templating engine (generates Markdown from template)
- **numpy**: Used by rmscene for coordinate calculations
- **parsita**: Parser combinator library (used by rmscene internally)

## Architecture Overview

### The Challenge of Parsing Proprietary Binary Formats

Before diving into the architecture, it's worth understanding the fundamental challenge remarks solves: parsing and interpreting proprietary binary formats. This isn't like parsing JSON or XML where the format is documented and stable—it's reverse engineering that requires deep technical detective work.

**What makes .rm files challenging?**

ReMarkable's `.rm` format is a binary format optimized for tablet performance, not interoperability. It contains:
- **Compact encoding**: Coordinates stored as 16-bit fixed-point numbers
- **Hierarchical structure**: Nested groups, layers, and elements (like a scene graph)
- **Multiple data types**: Lines, rectangles, text, highlights, all with different structures
- **Version evolution**: Format changes across firmware versions (V3, V5, V6)
- **No documentation**: Reverse-engineered by community through hex dumps and experimentation

The rmscene library (which remarks depends on) represents years of community effort:
- Analyzing memory dumps from tablets
- Comparing binary files across firmware versions
- Testing edge cases (what happens with 1000 layers? rotated text? overlapping highlights?)
- Maintaining compatibility as ReMarkable updates firmware

**Why remarks doesn't parse .rm files directly:**

You might wonder: why depend on external libraries (rmscene, rmc) instead of implementing parsing directly? Several reasons:

1. **Expertise**: The maintainers of rmscene have deep knowledge of the format from years of reverse engineering
2. **Maintenance**: When firmware updates change the format, rmscene is updated by the community
3. **Complexity**: The binary format is ~2000 lines of parsing code with subtle edge cases
4. **Testing**: rmscene has extensive test suites with real-world files
5. **Features**: rmscene handles many annotation types remarks might not even need yet

For the remarquee project, this dependency architecture is beneficial: remarks provides a stable Python API, and rmscene handles the messy binary parsing. When we integrate remarks into remarquee, we inherit this stability.

### The Pipeline Architecture

remarks follows a pipeline architecture where data flows through distinct processing stages, each with a specific transformation:

```
┌─────────────────────────────────────────┐
│      Input Layer                        │
│  - .rmn/.rmdoc (ZIP archives)          │
│  - xochitl directory structure          │
│  - .metadata, .content, .rm files       │
├─────────────────────────────────────────┤
│      Parsing Layer                      │
│  ┌──────────────────────────────────┐   │
│  │ conversion/parsing.py            │   │
│  │ - parse_rm_file()                │   │
│  │ - read_rm_file_version()         │   │
│  │ - determine_document_dimensions()│   │
│  └──────────────────────────────────┘   │
│  ┌──────────────────────────────────┐   │
│  │ utils.py                          │   │
│  │ - Metadata/content reading        │   │
│  │ - Page redirection mapping        │   │
│  │ - Tag extraction                  │   │
│  └──────────────────────────────────┘   │
├─────────────────────────────────────────┤
│      Document Model                      │
│  ┌──────────────────────────────────┐   │
│  │ Document.py                       │   │
│  │ - Document class                   │   │
│  │ - Page iteration                  │   │
│  │ - Source PDF handling             │   │
│  └──────────────────────────────────┘   │
├─────────────────────────────────────────┤
│      Conversion Layer                   │
│  ┌──────────────────────────────────┐   │
│  │ conversion/text.py                │   │
│  │ - Highlight extraction            │   │
│  │ - Text extraction                 │   │
│  │ - Markdown formatting             │   │
│  └──────────────────────────────────┘   │
│  ┌──────────────────────────────────┐   │
│  │ remarks.py                        │   │
│  │ - run_remarks()                    │   │
│  │ - process_document()              │   │
│  │ - PDF/SVG merging                 │   │
│  └──────────────────────────────────┘   │
├─────────────────────────────────────────┤
│      Output Layer                       │
│  ┌──────────────────────────────────┐   │
│  │ output/PdfFile.py                 │   │
│  │ - Smart highlight application     │   │
│  │ - PDF annotation rendering        │   │
│  └──────────────────────────────────┘   │
│  ┌──────────────────────────────────┐   │
│  │ output/ObsidianMarkdownFile.py    │   │
│  │ - Markdown generation             │   │
│  │ - Frontmatter creation            │   │
│  │ - Template rendering              │   │
│  └──────────────────────────────────┘   │
└─────────────────────────────────────────┘
```

### Data Flow Example: Process Notebook

1. **Input** (`remarks.py`): Load `.rmn` ZIP or xochitl directory
2. **Discovery** (`utils.py`): Find all `.metadata` files
3. **Document Creation** (`Document.py`): Create `Document` object from metadata
4. **Page Iteration** (`Document.pages()`): Iterate over pages with `.rm` files
5. **Parsing** (`conversion/parsing.py`): Parse `.rm` file using `rmscene`
6. **PDF Conversion** (`rmc.exporters.pdf`): Convert `.rm` to PDF
7. **Merging** (`remarks.py`): Merge annotation PDF with source PDF
8. **Highlight Extraction** (`conversion/text.py`): Extract highlights and text
9. **Output Generation** (`output/`): Generate PDF and Markdown files

## Package Structure

### Core Modules

#### `remarks/remarks.py` - Main Processing Logic
- **`run_remarks(input_dir, output_dir)`**: Main entry point
  - Handles `.rmn`/`.rmdoc` ZIP extraction
  - Discovers documents via `.metadata` files
  - Processes each document
- **`process_document(metadata_path, relative_doc_path, output_dir)`**: Process single document
  - Creates `Document` object
  - Opens source PDF
  - Iterates pages, parses `.rm` files
  - Merges annotations with PDF
  - Generates output files

#### `remarks/Document.py` - Document Model
- **`Document` class**: Represents a ReMarkable document
  - `metadata_path`: Path to `.metadata` file
  - `pages_list`: List of page UUIDs
  - `pages_map`: Redirection map (for inserted/duplicated pages)
  - `doc_type`: "pdf", "epub", or "notebook"
  - `name`: Document name
  - `rm_tags`: Document-level tags
  - `rm_annotation_files`: List of `.rm` files
- **`open_source_pdf()`**: Opens or creates source PDF
  - For PDF/EPUB: Opens existing PDF, handles inserted/duplicated pages
  - For notebooks: Creates blank PDF with correct page dimensions
- **`pages()`**: Generator yielding `(page_uuid, page_idx, rm_annotation_file)`
- **`get_page_tags_for_page(page_uuid)`**: Get tags for specific page

#### `remarks/conversion/parsing.py` - .rm File Parsing
- **`parse_rm_file(file_path)`**: Parse `.rm` file
  - Returns `(TMetaData, has_ann_hl)`, version string
  - Uses `rmscene` to read blocks and build scene tree
- **`parse_v6(file_path)`**: Parse V6 format specifically
  - Extracts `GlyphRange` (highlights), `Line` (drawings), `RootTextBlock` (typed text)
  - Returns `TMetaData` dict with:
    - `glyph_ranges`: List of `GlyphRange` objects
    - `highlights`: List of `RemarksRectangle` objects
    - `text`: `TTextBlock` with typed text
    - `scene_tree`: `SceneTree` root
- **`read_rm_file_version(file_path)`**: Read version from file header
- **`check_rm_file_version(file_path)`**: Validate file format
- **`determine_document_dimensions(file_path)`**: Compute page dimensions from drawing bounds

**Data Structures:**
- **`TMetaData`** (TypedDict): Parsed annotation data
- **`TTextBlock`** (TypedDict): Typed text block with position
- **`RemarksRectangle`**: Highlight rectangle with color

#### `remarks/conversion/text.py` - Text Extraction
- **`check_if_text_extractable(page)`**: Check if PDF page has extractable text
- **`get_highlight_rects(page)`**: Get highlight annotation rectangles from PDF
- **`get_page_text_tuples(page, option, flags, sort)`**: Extract words/blocks from PDF
- **`extract_groups_from_pdf_ann_hl(page, malformed)`**: Extract highlighted text groups from PDF annotations
- **`extract_groups_from_smart_hl(hl_data)`**: Extract highlights from ReMarkable smart highlights
- **`prepare_md_from_hl_groups(page, ann_hl_groups, smart_hl_groups, presentation)`**: Format highlights as Markdown

#### `remarks/output/PdfFile.py` - PDF Output
- **`apply_smart_highlight(page, highlight, x_translation)`**: Apply ReMarkable highlight to PDF page
  - Converts ReMarkable coordinates to PDF coordinates
  - Accounts for reMarkable's center-top origin (0,0)
  - Applies color from `PenColor` enum
- **`get_highlight_color(pen_color)`**: Convert `PenColor` to RGB tuple
- **`add_error_annotation(page, more_info)`**: Add error annotation to PDF page

#### `remarks/output/ObsidianMarkdownFile.py` - Markdown Output
- **`ObsidianMarkdownFile` class**: Generates Obsidian-compatible Markdown
  - `pages`: Dict mapping page index to `RMPage` objects
  - `document`: Associated `Document` object
- **`RMPage` class**: Represents a single page
  - `highlights`: List of `GlyphRange` objects
  - `tags`: List of page tags
  - `text`: List of `Paragraph` objects (typed text)
- **`add_highlights(page_idx, highlights)`**: Add highlights to page
  - Merges nearby highlights using `merge_highlights()`
- **`add_text(page_idx, text)`**: Add typed text to page
- **`add_page_tags(page_idx, tags)`**: Add tags to page
- **`save(location)`**: Render and save Markdown file
  - Uses Jinja2 template (`obsidian_markdown.md.jinja`)
  - Generates YAML frontmatter with tags and metadata

**Helper Functions:**
- **`render_paragraph(paragraph)`**: Convert `Paragraph` to Markdown
  - Handles bold, italic, headings, bullets, checkboxes
- **`merge_highlights(highlights)`**: Merge nearby highlights (gap ≤ 3 characters)
- **`merge_highlight_texts(h1, h2, distance)`**: Merge text from two highlights
- **`calculate_highlight_distance(h1, h2)`**: Calculate distance between highlights

#### `remarks/utils.py` - Utility Functions
- **`read_meta_file(path, suffix)`**: Read `.metadata`, `.content`, etc. files (cached)
- **`is_document(path)`**: Check if path is a document (not collection)
- **`get_document_filetype(path)`**: Get document type ("pdf", "epub", "notebook")
- **`get_visible_name(path)`**: Get document display name
- **`get_ui_path(path)`**: Build UI path from parent hierarchy
- **`construct_redirection_map(content)`**: Build page redirection map
  - Handles inserted pages (`INSERTED_PAGE = -1`)
  - Handles duplicated pages (positive index)
- **`is_inserted_page(idx)`**: Check if page is inserted (blank/notebook)
- **`is_duplicate_page(idx)`**: Check if page is duplicated from source PDF
- **`get_document_tags(path)`**: Extract document-level tags
- **`get_page_tags(path, page_id)`**: Extract page-level tags
- **`sanitize_obsidian_tag(tag)`**: Sanitize tag for Obsidian compatibility
- **`get_pages_data(path)`**: Get pages list and redirection map
- **`list_ann_rm_files(path)`**: List all `.rm` annotation files

#### `remarks/dimensions.py` - Coordinate System
- **`LengthUnit` enum**: Unit types (rmpts, mupts, mm, pt)
- **`Dimensions` dataclass**: Base dimensions class
- **`ReMarkableDimensions`**: ReMarkable point dimensions (1404×1872 default)
- **`PaperDimensions`**: Millimeter dimensions (A4: 210×297)
- **`PyMuPDFDimensions`**: PyMuPDF point dimensions (A4: 595×842)
- **`TypographicDimensions`**: Typographic point dimensions

**Constants:**
- `REMARKABLE_DOCUMENT`: Default notebook size (1404×1872 rmpts)
- `REMARKABLE_PHYSICAL_SCREEN`: Physical screen size (188×246 mm)
- `REMARKABLE_PDF_EXPORT`: Export size (445×594 typographic pts)

**Conversion Methods:**
- `to_mm()`: Convert to millimeters
- `to_mu()`: Convert to PyMuPDF points

#### `remarks/metadata.py` - Metadata Constants
- **`ReMarkableAnnotationsFileHeaderVersion`**: File version constants
  - `V3`, `V5`, `V6`, `UNKNOWN`
- **`ReMarkableDevice`**: Device type constants
  - `reMarkable`, `reMarkable2`, `reMarkablePaperPro`

#### `remarks/warnings.py` - Warning System
- **`ScrybbleWarning` dataclass**: Warning message with PDF annotation rendering
- **`scrybble_warning_only_v6_supported`**: Warning for non-V6 files
- **`scrybble_warning_typed_text_highlighting_not_supported`**: Warning for typed text highlights

#### `remarks/server.py` - HTTP Server (Optional)
- **Flask application**: HTTP API for processing documents
- **`/process` endpoint**: POST with `in_path` and `out_path`
- **`/health` endpoint**: Health check
- **`main_prod()`**: Gunicorn production entry point

#### `remarks/__main__.py` - CLI Entry Point
- **`main()`**: Command-line interface
  - Arguments: `input_dir`, `output_dir`, `--log_level`
  - Calls `run_remarks(input_dir, output_dir)`

## Key Types and Data Structures

### The Object Model: Representing ReMarkable's Document Hierarchy in Python

Before diving into specific classes, it's important to understand the conceptual model remarks uses to represent documents. ReMarkable's document format is multi-file and hierarchical (documents contain pages, pages contain annotations, annotations contain strokes/text). remarks needs to model this hierarchy while remaining practical to use.

**Design challenge:**

The raw files downloaded from ReMarkable are:
- Scattered across multiple files (`.metadata`, `.content`, `.rm` per page)
- Referenced by UUIDs (not human-friendly)
- Using different formats (JSON, binary)
- With complex relationships (pages can be inserted, duplicated, reordered)

remarks' object model solves this by providing:
1. **High-level abstractions**: Work with `Document` objects, not raw files
2. **Iterator pattern**: Process pages sequentially without loading all at once
3. **Lazy loading**: Only parse `.rm` files when needed (performance)
4. **Clean separation**: Parsing logic separate from data model

**The three-tier model:**

**Tier 1: Document (high-level)**  
Represents entire document, manages pages, handles file discovery.  
*For remarquee*: This is the main entry point—create a Document, iterate its pages.

**Tier 2: Page (mid-level)**  
Represents single page with annotations (RMPage class).  
*For remarquee*: Collect highlights, text, tags per page for processing.

**Tier 3: Annotation Data (low-level)**  
Raw parsed data from `.rm` files (GlyphRange, Line, Rectangle).  
*For remarquee*: Usually don't need to work at this level unless building custom features.

Understanding this hierarchy helps you decide which level to work at when building remarquee features. Need to batch-process all highlights across documents? Work at Document level. Need to analyze stroke patterns? Work at Annotation Data level.

### Core Classes

#### `Document`

The Document class is your primary interface to ReMarkable documents in Python. It's designed as a facade that hides the complexity of ReMarkable's multi-file format, presenting a clean, intuitive API.

**Design philosophy:**

Document is lazy and lightweight—it doesn't load all data upfront. Instead:
- Reads metadata/content files (small, JSON)
- Discovers available `.rm` files (just filenames)
- Provides iterator for processing pages (one at a time)

This design enables processing large documents (hundreds of pages) without excessive memory usage. When you iterate pages, only the current page's `.rm` file is loaded and parsed.

**Why this class exists:**

Without Document class, you'd need to:
```python
# Manually read metadata
with open("abc123.metadata") as f:
    metadata = json.load(f)
    name = metadata["visibleName"]

# Manually read content
with open("abc123.content") as f:
    content = json.load(f)
    pages = content["cPages"]["pages"]

# Manually handle redirection
for i, page in enumerate(pages):
    if "redir" in page:
        # Copy from source PDF...
    else:
        # Inserted page...
```

With Document class:
```python
doc = Document(Path("abc123.metadata"))
pdf = doc.open_source_pdf()  # Handles everything!
for uuid, idx, rm_file in doc.pages():
    # Process annotations...
```

Much cleaner! This is good API design—complexity hidden behind simple interface.
```python
class Document:
    metadata_path: pathlib.Path
    pages_list: List[str]  # Page UUIDs
    pages_map: List[int]   # Redirection map
    doc_type: str          # "pdf", "epub", "notebook"
    name: str              # Document name
    rm_tags: List[str]     # Document tags
    rm_annotation_files: List[pathlib.Path]  # .rm files
    
    def open_source_pdf() -> fitz.Document
    def pages() -> Generator[Tuple[str, int, pathlib.Path]]
    def get_page_tags_for_page(page_uuid: str) -> List[str]
```

#### `ObsidianMarkdownFile`
```python
class ObsidianMarkdownFile:
    pages: Dict[int, RMPage]
    document: Document
    
    def add_highlights(page_idx: int, highlights: List[GlyphRange])
    def add_text(page_idx: int, text: Dict)
    def add_page_tags(page_idx: int, tags: List[str])
    def save(location: pathlib.Path)
```

#### `RMPage`
```python
class RMPage:
    highlights: List[GlyphRange]
    tags: List[str]
    text: List[Paragraph] | None
```

### Data Structures

#### `TMetaData` (TypedDict)
```python
{
    "glyph_ranges": List[GlyphRange],      # Highlight ranges
    "highlights": List[RemarksRectangle],  # Highlight rectangles
    "text": TTextBlock | None,              # Typed text block
    "scene_tree": SceneTree | None          # Scene tree root
}
```

#### `TTextBlock` (TypedDict)
```python
{
    "pos_x": float,
    "pos_y": float,
    "width": float,
    "text": TextDocument
}
```

#### `RemarksRectangle` (dataclass)
```python
@dataclass
class RemarksRectangle:
    color: int              # PenColor enum value
    rectangles: List[Rectangle]  # Rectangle coordinates
```

## ReMarkable File Format (Understanding the Structure)

### Why understand the file format?

ReMarkable stores documents in a complex multi-file structure. Understanding this helps you:
- Debug parsing issues
- Know what files to look for
- Understand why certain operations work/fail
- Extend remarks for custom use cases

### Directory Structure

When you download a document from ReMarkable (using `rmapi get` or directly from device), you get a **`.rmdoc`** or **`.rmn`** file. These are just ZIP archives containing:

```
document-uuid.rmdoc  (this is a ZIP file)
└── When extracted:
    ├── document-uuid.metadata    # Document metadata (JSON)
    ├── document-uuid.content     # Document content structure (JSON)
    ├── document-uuid.pdf         # Source PDF (if PDF/EPUB document)
    ├── document-uuid.pagedata    # Page metadata
    └── document-uuid/            # Directory with annotations
        ├── page-uuid-1.rm        # Page 1 annotations (binary)
        ├── page-uuid-2.rm        # Page 2 annotations
        ├── page-uuid-1-metadata.json  # Page 1 layer metadata
        └── ...
```

**Key insight**: One document = one UUID, but each page has its own UUID.

### .metadata File Format (Document-Level Info)

This JSON file contains high-level document information:

```json
{
  "type": "DocumentType",              // or "CollectionType" (folder)
  "visibleName": "Research Paper",     // What user sees
  "parent": "parent-folder-uuid",      // Parent directory (or "" for root)
  "lastModified": "1702598400123",     // Unix timestamp (milliseconds)
  "version": 3,                        // Document version (increments on change)
  "deleted": false,                    // Soft-delete flag
  "pinned": false,                     // Pinned in UI
  "lastOpened": "1702598400123"        // Last opened timestamp
}
```

**What remarks uses this for:**
- `visibleName`: Document display name
- `parent`: Build directory hierarchy
- `type`: Filter out folders (only process documents)

### .content File Format (Page Structure)

This JSON file is more complex—it describes the document's pages and how they relate to the source PDF:

```json
{
  "fileType": "pdf",    // "pdf", "epub", or "notebook"
  "pages": ["uuid-1", "uuid-2", "uuid-3"],  // Legacy format
  
  "cPages": {           // Current format (more detailed)
    "pages": [
      {
        "id": "page-uuid-1",
        "redir": {"value": 0}  // Maps to page 0 of source PDF
      },
      {
        "id": "page-uuid-2"    // No redir = INSERTED page (blank)
      },
      {
        "id": "page-uuid-1",   // Duplicate of page 1!
        "redir": {"value": 0}  // Points to same source page
      },
      {
        "id": "page-uuid-3",
        "redir": {"value": 1}  // Maps to page 1 of source PDF
      }
    ]
  },
  
  "tags": [                    // Document-level tags
    {"name": "important"},
    {"name": "work"}
  ],
  
  "pageTags": [                // Page-level tags (new feature)
    {"pageId": "page-uuid-1", "name": "key-concept"},
    {"pageId": "page-uuid-3", "name": "summary"}
  ]
}
```

**Understanding page redirection:**

The `redir` field is tricky but important:

- **With `redir`**: Page shows content from source PDF page `redir.value`
- **Without `redir`**: Inserted blank page (for notes, sketches)

**Example scenario:**

```
You have a 10-page PDF.
You insert a blank page after page 1 to add notes.
You duplicate page 5 to annotate differently.

Result:
- cPages.pages[0]: {redir: 0}     → Source page 1
- cPages.pages[1]: (no redir)     → INSERTED blank page
- cPages.pages[2]: {redir: 1}     → Source page 2
- ...
- cPages.pages[5]: {redir: 4}     → Source page 5
- cPages.pages[6]: {redir: 4}     → DUPLICATE of source page 5
- cPages.pages[7]: {redir: 5}     → Source page 6
- ...
```

**Why remarks needs this:**

When generating PDF output:
- Inserted pages: Create blank PDF page
- Redirected pages: Copy from source PDF
- Duplicated pages: Copy again from source PDF

### .rm File Format (V6) - The Tricky Part

This is a **binary format** (not JSON!) that contains all annotations for one page.

**File structure:**
```
Bytes 0-47:   "reMarkable .lines file, version=6          " (48 bytes, ASCII)
Bytes 48-51:  Number of layers (4 bytes, little-endian uint32)
Bytes 52+:    Block data (variable length, complex binary format)
```

**What's in the blocks?**

The `.rm` file uses a "scene tree" format (like a scenegraph in 3D graphics):

```
SceneTree
├── Root
│   ├── Layer 1
│   │   ├── Line (pen stroke)
│   │   │   └── Points: [(x,y,pressure), ...]
│   │   ├── Line (another stroke)
│   │   └── ...
│   ├── Layer 2
│   │   └── ...
│   └── RootTextBlock (typed text)
│       └── TextDocument
│           ├── Paragraph (style: heading)
│           │   └── StyledText ("Title", bold)
│           ├── Paragraph (style: bullet)
│           │   └── StyledText ("Point 1", normal)
│           └── ...
└── Highlights
    └── GlyphRange (highlight)
        ├── start: 150 (character index in PDF)
        ├── length: 45 (characters)
        ├── text: "highlighted phrase"
        ├── color: 2 (PenColor enum)
        └── rectangles: [(x,y,w,h), ...] (bounding boxes)
```

**Why scene tree?**

ReMarkable organizes annotations hierarchically:
- **Layers**: Like Photoshop layers (can show/hide, reorder)
- **Groups**: Collections of strokes
- **Elements**: Individual lines, text blocks, highlights

**Parsing complexity:**

The binary format is complex because it:
- Supports nested groups and layers
- Stores stroke points efficiently (16-bit fixed-point)
- Includes color, pressure, tool type for each stroke
- Handles text with formatting (bold, italic, styles)
- Links highlights to source text (character positions)

**This is why we use `rmscene`:**

Writing a parser from scratch would require:
- Reverse-engineering the binary format
- Handling format version changes
- Supporting all annotation types
- Maintaining as ReMarkable updates firmware

The `rmscene` library does this for us (community-maintained, up-to-date).

### Dependencies Explained (Why Each is Needed)

- **PyMuPDF (fitz)**: PDF manipulation is hard. PyMuPDF handles:
  - Reading PDF files
  - Creating new PDF pages
  - Merging PDFs
  - Adding annotations (highlights)
  - Extracting text with coordinates
  
- **rmscene**: Specialized for ReMarkable binary format:
  - Parses `.rm` V6 format
  - Builds scene tree
  - Extracts annotations as Python objects
  
- **rmc**: Companion to rmscene:
  - Renders scene tree to PDF
  - Renders scene tree to SVG
  - Handles coordinate transformations
  
- **PyYAML**: Standard YAML library (Obsidian uses YAML frontmatter)
- **jinja2**: Templating (generates Markdown from structured data)
- **numpy**: Used by rmscene for matrix operations
- **flask** (optional): HTTP server for cloud processing

## Conversion Pipeline (Step-by-Step Explanation)

Understanding the conversion pipeline is key to working with remarks. Let's walk through the entire process with detailed explanations.

### Step 1: Document Discovery (Find What to Process)

Pseudocode:
```
FUNCTION run_remarks(input_dir, output_dir):
    // Input can be:
    // 1. A .rmn or .rmdoc ZIP file → extract to temp directory
    // 2. An xochitl directory → use directly
    
    IF input_dir ends with ".rmn" or ".rmdoc":
        temp_dir = CreateTempDir()
        ExtractZip(input_dir, temp_dir)
        input_dir = temp_dir
    
    // Find all documents (each has a .metadata file)
    metadata_files = FindFiles(input_dir, "*.metadata")
    
    FOR EACH metadata_path IN metadata_files:
        metadata = ReadJSON(metadata_path)
        
        // Skip folders
        IF metadata["type"] != "DocumentType":
            CONTINUE
        
        // Check document type
        content = ReadJSON(metadata_path.with_suffix(".content"))
        doc_type = content["fileType"]  // "pdf", "epub", or "notebook"
        
        IF doc_type IN ["pdf", "epub", "notebook"]:
            PROCESS document
        ELSE:
            SKIP (unsupported type)
```

**Why this matters:**

- ReMarkable stores both documents and folders as separate entities
- Each has a `.metadata` file, but folders don't have content
- We need to filter out folders and unsupported types

### Step 2: Document Initialization (Build Document Object)

Pseudocode:
```
CLASS Document:
    FUNCTION __init__(metadata_path):
        // Read metadata
        self.metadata_path = metadata_path
        self.name = ReadJSON(metadata_path)["visibleName"]
        
        // Read content structure
        content = ReadJSON(metadata_path.with_suffix(".content"))
        self.doc_type = content["fileType"]
        
        // Get pages list and redirection map
        self.pages_list, self.pages_map = construct_page_data(content)
        // pages_list = ["uuid-1", "uuid-2", ...]
        // pages_map = [0, -1, 1, ...]  (-1 = inserted, >=0 = source page index)
        
        // Get tags
        self.rm_tags = [tag["name"] for tag in content.get("tags", [])]
        
        // Find all .rm annotation files
        annotation_dir = metadata_path.parent / metadata_path.stem
        self.rm_annotation_files = FindFiles(annotation_dir, "*.rm")
```

**What's a redirection map?**

Imagine you have a 3-page PDF and you insert a blank page after page 1:

```
Display order → Source PDF mapping:
Page 0 → PDF page 0 (redir: 0)
Page 1 → INSERTED (-1)
Page 2 → PDF page 1 (redir: 1)
Page 3 → PDF page 2 (redir: 2)
```

The redirection map is: `[0, -1, 1, 2]`

This tells remarks:
- Which source PDF page to use for each display page
- Which pages are inserted blanks
- Which pages are duplicates

### Step 3: Source PDF Handling (Prepare Canvas)

This step prepares the base PDF that annotations will be added to.

**For PDF/EPUB documents:**

Pseudocode:
```
FUNCTION open_source_pdf():
    // Open the original PDF
    pdf_src = fitz.open(uuid + ".pdf")
    source_page_count = pdf_src.page_count
    
    // Process pages according to redirection map
    FOR i, redir_value IN ENUMERATE(pages_map):
        IF redir_value == -1:
            // Inserted blank page
            pdf_src.insert_page(
                index: i,
                width: REMARKABLE_WIDTH,
                height: REMARKABLE_HEIGHT
            )
        ELSE IF redir_value >= 0 AND i >= source_page_count:
            // Duplicated page (copy from source)
            pdf_src.copy_page(
                source_page: redir_value,
                destination: i
            )
        ELSE:
            // Regular page from source (already exists)
            PASS
    
    RETURN pdf_src
```

**What's happening:**

Imagine original PDF has 2 pages, but display has 4 pages (with insertions):

```
Before:  [Page 0, Page 1]  (source PDF)
After:   [Page 0, BLANK, Page 1, Page 1 copy]  (processed PDF)
         ↑        ↑       ↑       ↑
         source   new     source  new
```

**For Notebook documents:**

Pseudocode:
```
FUNCTION open_source_pdf():
    // Notebooks have no source PDF, create from scratch
    pdf_src = fitz.open()  // Empty PDF
    
    // Each page might have different dimensions (V6 feature!)
    page_sizes = []
    FOR page_uuid IN pages_list:
        rm_file = Find(f"{page_uuid}.rm")
        IF rm_file EXISTS:
            dims = determine_document_dimensions(rm_file)
            // Scans all pen strokes to find bounding box
            page_sizes.append(dims)
        ELSE:
            // No annotations, use default size
            page_sizes.append(REMARKABLE_DOCUMENT)
    
    // Create blank pages with computed dimensions
    FOR dims IN page_sizes:
        pdf_src.new_page(
            width: dims.width,
            height: dims.height
        )
    
    RETURN pdf_src
```

**Why dynamic dimensions?**

ReMarkable V6 introduced "infinite canvas"—pages can be different sizes depending on where you drew. remarks computes the minimum size needed to fit all annotations.

### Step 4: Page Processing (The Core Loop)

This is where annotations are parsed and applied to the PDF.

Pseudocode:
```
FUNCTION process_document(metadata_path):
    document = Document(metadata_path)
    pdf_src = document.open_source_pdf()
    obsidian_markdown = ObsidianMarkdownFile(document)
    
    // First pass: Add page tags to markdown
    FOR page_idx, page_uuid IN ENUMERATE(document.pages_list):
        page_tags = document.get_page_tags_for_page(page_uuid)
        IF page_tags:
            obsidian_markdown.add_page_tags(page_idx, page_tags)
    
    // Second pass: Process annotations
    FOR page_uuid, page_idx, rm_file IN document.pages():
        IF rm_file is NULL:
            CONTINUE  // No annotations on this page
        
        // Check version
        version = read_rm_file_version(rm_file)
        IF version != V6:
            ADD warning annotation to PDF
            CONTINUE
        
        // Parse .rm file
        ann_data = parse_rm_file(rm_file)
        // ann_data = {
        //   "glyph_ranges": [...],    # Highlights
        //   "highlights": [...],       # Highlight rectangles
        //   "text": {...},             # Typed text
        //   "scene_tree": SceneTree    # Full scene tree
        // }
        
        // Convert .rm to PDF (render pen strokes)
        temp_pdf = CreateTempFile(suffix=".pdf")
        rm_to_pdf(rm_file, temp_pdf)  // Uses rmc library
        annotation_pdf = fitz.open(temp_pdf)
        
        // Merge annotation PDF with source page
        source_page = pdf_src[page_idx]
        
        IF source_page has content:
            // Source page has text/images (PDF/EPUB)
            // Need to carefully position annotation overlay
            
            // Get bounding box of annotations
            bbox = get_bounding_box(ann_data["scene_tree"])
            // bbox = {x_min, x_max, y_min, y_max}
            
            // Calculate positions for merging
            x_translation = calculate_x_translation(
                source_page.width,
                annotation_pdf.width,
                bbox
            )
            
            // Merge: show both source page and annotations
            merged_page = CREATE new page(
                width: max(source.width, annotation.width),
                height: max(source.height, annotation.height)
            )
            merged_page.show_pdf_page(source_page, at: (x_bg, y_bg))
            merged_page.show_pdf_page(annotation_pdf, at: (x_svg, y_svg))
            
            // Replace in source PDF
            pdf_src.replace_page(page_idx, merged_page)
        ELSE:
            // Empty page (notebook), just insert annotations
            pdf_src.insert_pdf(annotation_pdf, at: page_idx)
            pdf_src.delete_page(page_idx + 1)  // Remove old page
        
        // Apply smart highlights (add as PDF annotations)
        FOR highlight IN ann_data["highlights"]:
            add_highlight_annotation(
                source_page,
                highlight.rectangles,
                highlight.color,
                x_translation
            )
        
        // Extract for Markdown
        IF ann_data["text"]:
            obsidian_markdown.add_text(page_idx, ann_data["text"])
        
        IF ann_data["glyph_ranges"]:
            obsidian_markdown.add_highlights(page_idx, ann_data["glyph_ranges"])
    
    // Save outputs
    pdf_src.save(output_dir / f"{document.name}_remarks.pdf")
    obsidian_markdown.save(output_dir / f"{document.name}")
```

**Why two passes?**

1. First pass (page tags): Need to collect all tags before processing annotations
2. Second pass (annotations): Process each page's `.rm` file

**Why merge PDFs instead of rendering directly?**

ReMarkable's pen strokes are complex (pressure, tool type, color). Rather than re-implement rendering:
1. Use `rmc.rm_to_pdf()` to render annotations (it's battle-tested)
2. Merge rendered PDF with source PDF (leverage PyMuPDF)

### Step 5: Coordinate Transformation (The Math)

This is one of the trickiest parts—coordinate systems don't match.

**ReMarkable coordinate system:**
```
        (0, 0) ← Center-top
          │
    ──────┼──────
    │     │     │
    │     │     │
    │     ↓     │
    
X: -702 to +702  (center is 0)
Y: 0 to 1872     (top is 0)
```

**PDF coordinate system:**
```
    │              
    │              
    └──────────→
  (0,0)         
  Bottom-left
  
X: 0 to page_width
Y: 0 to page_height
```

**Transformation:**

Pseudocode:
```
FUNCTION xx(remarkable_x):
    // Convert ReMarkable X to PDF X
    // ReMarkable: -702 to +702 (center = 0)
    // PDF: 0 to page_width
    
    RETURN remarkable_x + (RM_WIDTH / 2)
    // Example: -702 → 0, 0 → 702, +702 → 1404

FUNCTION yy(remarkable_y):
    // Convert ReMarkable Y to PDF Y
    // ReMarkable: 0 to 1872 (top = 0)
    // PDF: 0 to page_height (bottom = 0)
    
    RETURN RM_HEIGHT - remarkable_y
    // Example: 0 → 1872, 1872 → 0 (flipped)
```

**Smart highlight positioning:**

When merging annotation PDF with source PDF, we need to calculate `x_translation`:

Pseudocode:
```
FUNCTION calculate_x_translation(page_width, svg_width, bbox):
    IF svg_width > page_width:
        // Annotations wider than source
        // Position source PDF at center of annotation canvas
        x_bg = (svg_width / 2) - (page_width / 2)
        x_translation = x_bg + (page_width / 2)
    ELSE IF svg_width < page_width:
        // Source wider than annotations
        // Position annotations at center of source canvas
        x_svg = (page_width / 2) - (svg_width / 2)
        x_translation = page_width / 2
    ELSE:
        // Same width
        x_translation = page_width / 2
    
    RETURN x_translation
```

**Why is this complex?**

- ReMarkable center-top origin is unusual
- Source PDF might be any size
- Annotations might extend beyond source page
- Need to position highlights correctly relative to text

### Step 6: Output Generation

**PDF Output:**

```python
output_pdf_path = output_dir / f"{relative_doc_path} _remarks.pdf"
pdf_src.save(output_pdf_path)
```

**What you get:**
- Original PDF content preserved
- Pen strokes rendered as vector graphics (high quality)
- Highlights added as PDF annotations (selectable, searchable)
- Typed text rendered as text

**Markdown Output:**

```python
obsidian_markdown.save(output_dir / f"{relative_doc_path}")
# Creates: document_obsidian.md
```

**What you get:**
```markdown
---
scrybble_timestamp: 1702598400
scrybble_filename: Research Paper
tags:
  - #remarkable/important
  - #remarkable/work
---

# Research Paper

> [!WARNING] **Do not modify** this file
> This file is automatically generated by Scrybble and will be overwritten.

## Pages

### [[Research Paper.pdf#page=1|Research Paper, page 1]]
#key-concept

#### Highlights
> The key finding is that distributed systems require consensus algorithms.

> Paxos and Raft are two popular approaches to solving this problem.

#### Typed text
## Meeting Notes
- Review the paper
- Discuss implications
```

**Features:**
- YAML frontmatter with metadata
- Page links (Obsidian can jump to PDF page)
- Extracted highlights (searchable text)
- Typed text preserved (with formatting)
- Tags at document and page level

## Developer Usage Guide

### Getting Started (For New Developers)

**Prerequisites:**
- Python 3.12+
- Poetry or Nix (for dependency management)
- ReMarkable files downloaded (via `rmapi get` or USB transfer)

**Installation:**

Using Nix (recommended):
```bash
nix develop  # Enters development environment
nix run .#   # Runs remarks
```

Using Poetry:
```bash
poetry install
poetry run remarks <input_dir> <output_dir>
```

### Basic Usage (Complete Workflow)

**Scenario**: You downloaded a document from your tablet using rmapi.

```python
from remarks import run_remarks
import pathlib

# You have: research-paper.rmdoc (from rmapi get)
# You want: research-paper_remarks.pdf and research-paper_obsidian.md

# Method 1: Process .rmdoc file directly
input_file = pathlib.Path("research-paper.rmdoc")
output_dir = pathlib.Path("output/")

run_remarks(input_file, output_dir)
# Output:
# - output/research-paper_remarks.pdf
# - output/research-paper_obsidian.md

# Method 2: Process extracted directory
# First extract: unzip research-paper.rmdoc -d extracted/
input_dir = pathlib.Path("extracted/")
run_remarks(input_dir, output_dir)
```

**What happens:**
1. Discovers all documents in input
2. For each document:
   - Parses metadata and content
   - Opens/creates source PDF
   - Processes each page's annotations
   - Generates PDF and Markdown outputs
3. Outputs saved to output_dir

### Processing Individual Documents (Advanced)

Sometimes you want fine-grained control over the conversion process.

```python
from remarks.Document import Document
from remarks.output.ObsidianMarkdownFile import ObsidianMarkdownFile
from remarks.conversion.parsing import parse_rm_file, read_rm_file_version
from remarks.metadata import ReMarkableAnnotationsFileHeaderVersion
import pathlib
import fitz

# Step 1: Create document object
metadata_path = pathlib.Path("abc123-def456.metadata")
document = Document(metadata_path)

print(f"Document: {document.name}")
print(f"Type: {document.doc_type}")
print(f"Pages: {len(document.pages_list)}")
print(f"Tags: {document.rm_tags}")

# Step 2: Open source PDF (or create for notebooks)
pdf_src = document.open_source_pdf()
# This handles:
# - Opening existing PDF (for PDF/EPUB)
# - Creating blank pages (for notebooks)
# - Inserting blank pages (where user inserted)
# - Duplicating pages (where user duplicated)

# Step 3: Process each page
obsidian_markdown = ObsidianMarkdownFile(document)

for page_uuid, page_idx, rm_file in document.pages():
    if not rm_file:
        continue  # No annotations on this page
    
    # Check version
    version = read_rm_file_version(rm_file)
    if version != ReMarkableAnnotationsFileHeaderVersion.V6:
        print(f"Warning: Page {page_idx} is not V6 format")
        continue
    
    # Parse annotations
    (ann_data, has_ann_hl), version_str = parse_rm_file(rm_file)
    
    print(f"Page {page_idx}:")
    print(f"  - Highlights: {len(ann_data['glyph_ranges'])}")
    print(f"  - Has typed text: {ann_data['text'] is not None}")
    
    # Add to markdown
    if ann_data["glyph_ranges"]:
        obsidian_markdown.add_highlights(page_idx, ann_data["glyph_ranges"])
    
    if ann_data["text"]:
        obsidian_markdown.add_text(page_idx, ann_data["text"])
    
    # You could also:
    # - Extract highlights to your own format
    # - Process typed text differently
    # - Skip PDF generation (markdown only)

# Step 4: Save outputs
pdf_src.save("output_custom.pdf")
obsidian_markdown.save(pathlib.Path("output_custom"))
```

### Working with Annotations (Detailed Examples)

**Example 1: Extract all highlights to text file**

```python
from remarks.Document import Document
from remarks.conversion.parsing import parse_rm_file

metadata_path = pathlib.Path("document.metadata")
document = Document(metadata_path)

all_highlights = []

for page_uuid, page_idx, rm_file in document.pages():
    if not rm_file:
        continue
    
    (ann_data, _), _ = parse_rm_file(rm_file)
    
    for glyph_range in ann_data["glyph_ranges"]:
        all_highlights.append({
            "page": page_idx + 1,
            "text": glyph_range.text,
            "color": glyph_range.color,
            "position": glyph_range.start  # Character position in source
        })

# Save to JSON
import json
with open("highlights.json", "w") as f:
    json.dump(all_highlights, f, indent=2)
```

**Example 2: Generate PDF without highlights (drawings only)**

```python
from remarks.Document import Document
from remarks.conversion.parsing import parse_rm_file
from rmc.exporters.pdf import rm_to_pdf
import fitz

document = Document(pathlib.Path("document.metadata"))
pdf_src = document.open_source_pdf()

for page_uuid, page_idx, rm_file in document.pages():
    if not rm_file:
        continue
    
    # Convert .rm to PDF (has all drawings)
    temp_pdf = f"temp_{page_idx}.pdf"
    rm_to_pdf(rm_file, temp_pdf)
    
    # Merge with source
    annotation_pdf = fitz.open(temp_pdf)
    pdf_src.insert_pdf(annotation_pdf, start_at=page_idx)
    pdf_src.delete_page(page_idx + 1)
    
    # Don't apply highlights (skip apply_smart_highlight)

pdf_src.save("output_no_highlights.pdf")
```

**Example 3: Custom Markdown formatting**

```python
from remarks.output.ObsidianMarkdownFile import ObsidianMarkdownFile, RMPage

class CustomMarkdownFile(ObsidianMarkdownFile):
    def save(self, location):
        # Custom template logic
        with open(f"{location}_custom.md", "w") as f:
            f.write(f"# {self.document.name}\n\n")
            
            for page_idx, page in sorted(self.pages.items()):
                f.write(f"## Page {page_idx + 1}\n\n")
                
                if page.tags:
                    f.write(f"Tags: {', '.join(page.tags)}\n\n")
                
                if page.highlights:
                    f.write("### Highlights\n")
                    for hl in page.highlights:
                        f.write(f"- {hl.text}\n")
                
                if page.text:
                    f.write("\n### Notes\n")
                    for para in page.text:
                        # Custom paragraph rendering
                        f.write(f"{para}\n")
                
                f.write("\n")

# Use custom formatter
document = Document(metadata_path)
custom_md = CustomMarkdownFile(document)
# ... process pages ...
custom_md.save(pathlib.Path("output"))
```

### Parsing .rm Files (Understanding the Binary Format)

The `.rm` file is a binary format that's not human-readable. Here's how parsing works:

**Step-by-step parsing:**

Pseudocode:
```
FUNCTION parse_rm_file(file_path):
    // 1. Open file and read header
    file = OpenBinary(file_path)
    header = file.read(48)  // "reMarkable .lines file, version=6..."
    version = header[32]  // Version at byte 32
    num_layers = ReadUint32(file)  // Next 4 bytes
    
    // 2. Read all blocks using rmscene
    blocks = []
    WHILE not end_of_file:
        block = read_next_block(file)  // rmscene handles this
        blocks.append(block)
    
    // Block types include:
    // - RootTextBlock: Typed text
    // - SceneGroupItemBlock: Group of items
    // - SceneLineItemBlock: Pen stroke
    // - SceneGlyphItemRangeBlock: Highlight
    // - etc.
    
    // 3. Build scene tree from blocks
    scene_tree = SceneTree()
    build_tree(scene_tree, blocks)
    // This links blocks into hierarchy:
    // Root → Groups → Lines/Highlights/Text
    
    // 4. Extract specific annotation types
    output = {
        "highlights": [],
        "glyph_ranges": [],
        "text": None,
        "scene_tree": scene_tree
    }
    
    // Find typed text
    FOR block IN blocks:
        IF block is RootTextBlock:
            output["text"] = {
                "pos_x": block.pos_x,
                "pos_y": block.pos_y,
                "width": block.width,
                "text": TextDocument.from_scene_item(scene_tree.root_text)
            }
    
    // Walk scene tree to find highlights
    FOR element IN scene_tree.walk():
        IF element is GlyphRange:
            // Transform coordinates to PDF space
            translated_rectangles = [
                TransformCoordinates(rect) FOR rect IN element.rectangles
            ]
            
            output["glyph_ranges"].append(element)
            output["highlights"].append(RemarksRectangle(
                color: element.color,
                rectangles: translated_rectangles
            ))
    
    RETURN output
```

**Real-world example:**

```python
from remarks.conversion.parsing import parse_rm_file, read_rm_file_version
from remarks.metadata import ReMarkableAnnotationsFileHeaderVersion

# Check version first (V3 and V5 are deprecated)
version = read_rm_file_version("abc123-page1.rm")

if version == ReMarkableAnnotationsFileHeaderVersion.V6:
    # Parse file
    (ann_data, has_ann_hl), version_str = parse_rm_file("abc123-page1.rm")
    
    # Access highlights
    for glyph_range in ann_data["glyph_ranges"]:
        print(f"Highlight: {glyph_range.text}")
        print(f"  Position: chars {glyph_range.start} to {glyph_range.start + glyph_range.length}")
        print(f"  Color: {glyph_range.color}")
        print(f"  Rectangles: {len(glyph_range.rectangles)}")
    
    # Access typed text
    if ann_data["text"]:
        text_doc = ann_data["text"]["text"]
        print(f"Typed text at ({ann_data['text']['pos_x']}, {ann_data['text']['pos_y']}):")
        
        for paragraph in text_doc.contents:
            print(f"  Paragraph: {paragraph}")
            # Each paragraph has style (heading, bullet, plain, checkbox)
    
    # Access scene tree (for advanced processing)
    scene_tree = ann_data["scene_tree"]
    for element in scene_tree.walk():
        print(f"Element: {type(element).__name__}")
        # Line, GlyphRange, Rectangle, etc.
else:
    print(f"Unsupported version: {version}")
```

**Common mistake: Not checking version**

```python
# BAD: Assumes V6
ann_data = parse_rm_file("old-file.rm")
# CRASH: V3/V5 files will raise ValueError

# GOOD: Check version first
version = read_rm_file_version("old-file.rm")
if version == ReMarkableAnnotationsFileHeaderVersion.V6:
    ann_data = parse_rm_file("old-file.rm")
else:
    print(f"Skipping unsupported version: {version}")
```

### Processing Individual Documents (Fine-Grained Control)

When you need more control than `run_remarks()` provides:

```python
from remarks.Document import Document
from remarks.output.ObsidianMarkdownFile import ObsidianMarkdownFile
from remarks.conversion.parsing import parse_rm_file
from rmc.exporters.pdf import rm_to_pdf
import fitz
import pathlib

# Create document from metadata file
metadata_path = pathlib.Path("abc123.metadata")
document = Document(metadata_path)

print(f"Processing: {document.name}")
print(f"  Type: {document.doc_type}")
print(f"  Pages: {len(document.pages_list)}")
print(f"  Annotations: {len(document.rm_annotation_files)} .rm files")

# Open or create source PDF
pdf_src = document.open_source_pdf()
# For PDF/EPUB: Opens existing PDF
# For notebooks: Creates blank PDF

# Prepare markdown output
obsidian_markdown = ObsidianMarkdownFile(document)

# Process each page
for page_uuid, page_idx, rm_file in document.pages():
    if not rm_file:
        print(f"  Page {page_idx + 1}: No annotations")
        continue
    
    print(f"  Page {page_idx + 1}: Processing...")
    
    # Parse annotations
    (ann_data, _), _ = parse_rm_file(rm_file)
    
    # Convert .rm to PDF (renders pen strokes)
    temp_pdf_path = f"temp_page_{page_idx}.pdf"
    rm_to_pdf(rm_file, temp_pdf_path)
    annotation_pdf = fitz.open(temp_pdf_path)
    
    # Merge with source (detailed in conversion pipeline)
    # ... merging logic ...
    
    # Add highlights to markdown
    if ann_data["glyph_ranges"]:
        obsidian_markdown.add_highlights(page_idx, ann_data["glyph_ranges"])
        print(f"    - Added {len(ann_data['glyph_ranges'])} highlights")
    
    # Add typed text to markdown
    if ann_data["text"]:
        obsidian_markdown.add_text(page_idx, ann_data["text"])
        print(f"    - Added typed text")

# Save outputs
output_pdf = pathlib.Path("output") / f"{document.name}_custom.pdf"
output_pdf.parent.mkdir(parents=True, exist_ok=True)
pdf_src.save(output_pdf)

output_md = pathlib.Path("output") / f"{document.name}"
obsidian_markdown.save(output_md)

print(f"\nSaved:")
print(f"  - {output_pdf}")
print(f"  - {output_md}_obsidian.md")
```

### Common Pitfalls and Solutions

**Issue 1: "No .metadata files found"**

```python
# PROBLEM: Running remarks on wrong directory
run_remarks(pathlib.Path("~/Downloads/"), output_dir)
# ERROR: No .metadata files found

# SOLUTION: Use the extracted .rmdoc directory
# First: unzip document.rmdoc -d extracted/
# Then: run_remarks(pathlib.Path("extracted/"), output_dir)

# Or use .rmdoc directly (remarks extracts automatically):
run_remarks(pathlib.Path("document.rmdoc"), output_dir)
```

**Issue 2: Missing typed text in output**

```python
# PROBLEM: Typed text exists but not in markdown
# REASON: Not calling add_text()

# SOLUTION: Check if text exists and add it
(ann_data, _), _ = parse_rm_file(rm_file)
if ann_data["text"]:
    obsidian_markdown.add_text(page_idx, ann_data["text"])
else:
    print(f"No typed text on page {page_idx}")
```

**Issue 3: Coordinates are wrong**

```python
# PROBLEM: Highlights appear in wrong position
# REASON: Forgot x_translation parameter

# SOLUTION: Calculate x_translation when merging PDFs
x_translation = calculate_x_translation(...)  # See coordinate transformation
apply_smart_highlight(page, highlight, x_translation)
```

### Working with Dimensions

```python
from remarks.dimensions import (
    REMARKABLE_DOCUMENT,
    ReMarkableDimensions,
    PaperDimensions,
    PyMuPDFDimensions
)

# Default notebook size
notebook = REMARKABLE_DOCUMENT  # 1404×1872 rmpts

# Convert to millimeters
notebook_mm = notebook.to_mm()  # PaperDimensions

# Convert to PyMuPDF points
notebook_mu = notebook.to_mm().to_mu()  # PyMuPDFDimensions

# Create custom dimensions
custom = ReMarkableDimensions(width=2000, height=2500)
```

### Extracting Metadata

```python
from remarks.utils import (
    get_visible_name,
    get_document_filetype,
    get_document_tags,
    get_page_tags,
    get_pages_data
)

metadata_path = pathlib.Path("document-uuid.metadata")

# Document info
name = get_visible_name(metadata_path)
doc_type = get_document_filetype(metadata_path)
tags = list(get_document_tags(metadata_path))

# Page info
pages_list, pages_map = get_pages_data(metadata_path)
page_tags = get_page_tags(metadata_path, "page-uuid-1")
```

### Custom Output Formats

**Custom Markdown Template:**
```python
from remarks.output.ObsidianMarkdownFile import ObsidianMarkdownFile
from jinja2 import Environment, FileSystemLoader

class CustomMarkdownFile(ObsidianMarkdownFile):
    def save(self, location: pathlib.Path):
        # Use custom template
        env = Environment(loader=FileSystemLoader("templates/"))
        template = env.get_template("custom.md.jinja")
        content = template.render(pages=self.pages, ...)
        # Save...
```

**Custom PDF Processing:**
```python
from remarks.output.PdfFile import apply_smart_highlight
from remarks.conversion.parsing import RemarksRectangle

# Apply custom highlight
highlight = RemarksRectangle(
    color=1,  # PenColor enum value
    rectangles=[Rectangle(x=100, y=200, w=300, h=50)]
)
apply_smart_highlight(page, highlight, x_translation=0)
```

## File Organization Summary

### Entry Points
- `remarks/__main__.py`: CLI entry point (`remarks` command)
- `remarks/server.py`: HTTP server entry point (`remarks-server` command)

### Core Processing
- `remarks/remarks.py`: Main processing logic (`run_remarks()`, `process_document()`)
- `remarks/Document.py`: Document model class

### Parsing
- `remarks/conversion/parsing.py`: `.rm` file parsing
- `remarks/conversion/text.py`: Text extraction from PDF
- `remarks/conversion/__init__.py`: Conversion module exports

### Output
- `remarks/output/PdfFile.py`: PDF annotation rendering
- `remarks/output/ObsidianMarkdownFile.py`: Markdown generation
- `remarks/output/obsidian_markdown.md.jinja`: Markdown template

### Utilities
- `remarks/utils.py`: Metadata reading, path resolution, tag extraction
- `remarks/dimensions.py`: Coordinate system and conversions
- `remarks/metadata.py`: Version and device constants
- `remarks/warnings.py`: Warning system

### Configuration
- `pyproject.toml`: Package configuration, dependencies, scripts

## Quick Reference

### remarks API Cheat Sheet for remarquee Integration

This section distills the most essential parts of remarks' API—the functions and classes you'll use most frequently when building remarquee. While the full document provides deep context and explanations, this quick reference helps you write code quickly.

**When to use remarks in remarquee:**

remarks is your tool whenever you need to:
1. **Extract value from annotations**: User highlighted important passages, you need the text
2. **Generate shareable outputs**: Convert proprietary .rm files to standard formats
3. **Process batch documents**: Downloaded 50 PDFs via rmapi, extract all highlights
4. **Build knowledge systems**: Feed extracted highlights into note-taking apps, LLMs, search indices

**Common remarquee workflows using remarks:**

- **Workflow 1**: Download → Extract → Index
  ```
  rmapi downloads documents → remarks extracts highlights → remarquee indexes for search
  ```

- **Workflow 2**: Process → Analyze → Summarize
  ```
  remarks extracts text → geppetto (LLM) summarizes → remarquee generates report
  ```

- **Workflow 3**: Annotate → Integrate → Publish
  ```
  User annotates on tablet → rmapi syncs → remarks extracts → remarquee integrates into docs → publish
  ```

**Integration tips:**

- **Use `run_remarks()` for batch**: Process entire directories efficiently
- **Use `Document` class for control**: Fine-grained processing, custom outputs
- **Cache `.rm` parsing**: Parsing is expensive, cache results if processing multiple times
- **Handle errors gracefully**: Not all documents have annotations, not all .rm files are V6

### Key Functions

**Main Processing:**
- `run_remarks(input_dir, output_dir)`: Process directory of documents (highest-level, use for batch operations)
- `process_document(metadata_path, relative_doc_path, output_dir)`: Process single document (mid-level, use for custom processing)

**Parsing:**
- `parse_rm_file(file_path)`: Parse `.rm` file → `(TMetaData, has_ann_hl), version`
- `read_rm_file_version(file_path)`: Read version string
- `check_rm_file_version(file_path)`: Validate file format
- `determine_document_dimensions(file_path)`: Compute page dimensions

**Text Extraction:**
- `extract_groups_from_pdf_ann_hl(page, malformed)`: Extract PDF highlights
- `extract_groups_from_smart_hl(hl_data)`: Extract smart highlights
- `prepare_md_from_hl_groups(...)`: Format highlights as Markdown

**Utilities:**
- `get_visible_name(path)`: Get document name
- `get_document_filetype(path)`: Get document type
- `get_pages_data(path)`: Get pages list and redirection map
- `get_document_tags(path)`: Get document tags
- `get_page_tags(path, page_id)`: Get page tags

### Key Classes

- `Document`: Document model
- `ObsidianMarkdownFile`: Markdown generator
- `RMPage`: Page data container
- `ReMarkableDimensions`: Coordinate system
- `ScrybbleWarning`: Warning system

### External Dependencies

- **rmscene**: Parse `.rm` scene tree format
- **rmc**: Export `.rm` to PDF/SVG
- **PyMuPDF (fitz)**: PDF manipulation
- **jinja2**: Template rendering
- **PyYAML**: YAML frontmatter

## Related

- ReMarkable Wiki: https://remarkablewiki.com/tech/filesystem
- rmscene library: https://github.com/scrybbling-together/rmscene
- rmc library: https://github.com/scrybbling-together/rmc
- PyMuPDF documentation: https://pymupdf.readthedocs.io/
