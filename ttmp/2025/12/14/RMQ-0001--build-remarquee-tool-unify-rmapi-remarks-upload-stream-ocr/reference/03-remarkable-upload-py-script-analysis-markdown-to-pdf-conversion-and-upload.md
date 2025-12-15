---
Title: remarkable_upload.py script analysis (markdown to PDF conversion and upload)
Ticket: RMQ-0001
Status: active
Topics:
    - backend
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarquee/remarkable_upload.py
      Note: Main script - markdown to PDF conversion and upload to reMarkable via rmapi
ExternalSources: []
Summary: 'Comprehensive analysis of remarkable_upload.py: markdown to PDF conversion using pandoc/xelatex, upload to reMarkable via rmapi, workflow, architecture, and usage guide'
LastUpdated: 2025-12-14T17:48:44.407629506-05:00
---


# remarkable_upload.py script analysis (markdown to PDF conversion and upload)

## Goal

This document provides a comprehensive technical analysis of the `remarkable_upload.py` script, covering:
- Purpose and workflow
- Architecture and design patterns
- Key functions and data structures
- Command-line interface
- Markdown preprocessing (YAML frontmatter stripping, list normalization)
- PDF conversion pipeline (pandoc + xelatex)
- Integration with rmapi
- Developer usage guide

### remarkable_upload.py in the remarquee Ecosystem: Closing the Documentation Loop

In the remarquee ecosystem, remarkable_upload.py serves a unique and essential role: it's the **"write path"** for documentation workflows. While rmapi downloads and remarks extracts, remarkable_upload.py enables the reverse flow—taking documentation you've created on your computer and making it available on your tablet for reading, review, and annotation.

This might seem like a simple utility script ("just convert Markdown to PDF and upload"), but its design reflects deep understanding of documentation workflows in software development. Let me explain why this script exists and why it's designed the way it is.

**The Documentation Workflow Problem:**

Modern software development involves extensive documentation:
- Design documents explaining architectural decisions
- Meeting notes capturing discussions
- Research findings and analysis
- Technical specifications and API documentation
- Investigation notes and bug reports

These documents live in your codebase as Markdown files (often managed by tools like docmgr). They contain valuable context that you might want to review:
- During commute (read design doc on tablet)
- In meetings (annotate discussion points)
- While coding (reference API specs with tablet next to laptop)
- During code review (annotate architecture diagrams)

But there's a friction: how do you get dozens of Markdown files onto your tablet efficiently, organized properly, without manual conversion and upload steps?

**Why not just use ReMarkable's apps?**

ReMarkable's official desktop app can upload PDFs, but:
- Doesn't support Markdown (you must convert first)
- No batch operations (upload files one by one)
- No organization scheme (files go wherever)
- No duplicate detection (easy to create duplicates)
- No automation (must click through GUI)

**Why not just use rmapi directly?**

You could script it manually:
```bash
pandoc doc.md -o doc.pdf
rmapi put doc.pdf
```

But this has problems:
- YAML frontmatter breaks pandoc (common in docmgr/Obsidian)
- List formatting issues (pandoc quirks)
- Poor PDF typography (default fonts, spacing)
- No duplicate detection
- No organization (where do files go?)
- No date-based grouping

**remarkable_upload.py's Solution:**

This script automates the entire workflow with careful attention to real-world issues:

1. **Preprocessing**: Strips YAML frontmatter (docmgr compatibility), normalizes lists (pandoc compatibility)
2. **Typography**: Uses xelatex + DejaVu (Unicode support, professional quality)
3. **Organization**: Date-based folders (`ai/YYYY/MM/DD/`) for chronological browsing
4. **Safety**: Duplicate detection (won't overwrite unless `--force`)
5. **Integration**: Ticket-aware (finds docmgr tickets automatically)
6. **Flexibility**: Dry-run, PDF-only modes for testing and archival

For the remarquee tool, remarkable_upload.py demonstrates how to build user-friendly automation on top of lower-level tools (rmapi). It's not just glue code—it's thoughtful workflow design that solves real problems developers face daily.

## Context

### What is remarkable_upload.py?

`remarkable_upload.py` is a convenience script that bridges the gap between writing documentation in Markdown and reading it on your ReMarkable tablet. It automates the tedious process of:
1. Converting Markdown to PDF (with proper formatting)
2. Uploading PDF to tablet (organized by date)
3. Avoiding duplicate uploads

### Why does it exist?

**The problem:**

You write documentation in Markdown (design docs, meeting notes, research). You want to read/annotate these on your ReMarkable tablet, but:
- Markdown isn't natively supported by ReMarkable
- Manual conversion is tedious: `pandoc → PDF → rmapi upload → ...`
- Easy to accidentally overwrite documents
- Hard to organize uploads (where do they go?)
- YAML frontmatter (from docmgr, Obsidian) breaks pandoc

**The solution:**

This script automates the entire workflow:
```
Write Markdown → Run script → PDF on tablet (organized, no duplicates)
```

**Real-world use case:**

```
You're working on ticket RMQ-0001 with docmgr:
- Created design doc: ttmp/.../design/01-architecture.md
- Has YAML frontmatter (docmgr metadata)
- Want to read on tablet during commute

Run:
$ python remarkable_upload.py --ticket RMQ-0001

Script does:
1. Finds ticket directory
2. Strips YAML frontmatter (would break pandoc)
3. Converts to PDF with good fonts
4. Checks tablet (doesn't already exist?)
5. Uploads to ai/2025/12/14/ (organized by date)

Now on tablet:
- Open: ai/2025/12/14/01-architecture.pdf
- Read, annotate, highlight
```

### Why these specific tools?

**pandoc + xelatex:**
- `pandoc`: Universal document converter (Markdown → anything)
- `xelatex`: Modern LaTeX engine with Unicode support
- Alternative would be: `wkhtmltopdf` (worse typography), `pdfkit` (requires wkhtmltopdf), `weasyprint` (CSS-based, different look)

**DejaVu fonts:**
- Full Unicode support (CJK, Cyrillic, symbols, emojis)
- Preinstalled on most Linux systems
- Good rendering quality
- Free/open-source

**rmapi:**
- Already installed (needed for download anyway)
- Handles authentication (script doesn't need to)
- Reliable upload mechanism

### Key characteristics

- **Converts `.md` → `.pdf` using pandoc**: High-quality PDF with proper typography
- **xelatex + DejaVu fonts**: Unicode-safe (handles special characters, emojis, math)
- **YAML frontmatter stripping**: Prevents pandoc errors, keeps PDFs clean
- **List spacing normalization**: Ensures pandoc recognizes lists correctly
- **Existence checking**: Won't overwrite unless `--force`
- **Date-based organization**: Uploads to `ai/YYYY/MM/DD/` for easy navigation
- **Ticket-aware**: Can find tickets by ID, infer dates from ticket paths
- **Flexible modes**: Dry-run (preview), PDF-only (no upload), force (overwrite)

**Dependencies:**
- `pandoc`: Universal document converter
- `xelatex`: LaTeX PDF engine (part of TeX Live distribution)
- `rmapi`: ReMarkable API CLI
- Python 3.6+: Standard library only (no pip dependencies!)

## Architecture Overview

The script follows a linear pipeline architecture:

```
┌─────────────────────────────────────────┐
│      Input Layer                        │
│  - Markdown file(s)                     │
│  - Ticket directory resolution          │
│  - Date inference/validation            │
├─────────────────────────────────────────┤
│      Preprocessing Layer                │
│  ┌──────────────────────────────────┐   │
│  │ _strip_yaml_frontmatter()        │   │
│  │ - Remove YAML frontmatter        │   │
│  └──────────────────────────────────┘   │
│  ┌──────────────────────────────────┐   │
│  │ _normalize_list_spacing()        │   │
│  │ - Ensure proper list formatting  │   │
│  └──────────────────────────────────┘   │
├─────────────────────────────────────────┤
│      PDF Conversion Layer               │
│  ┌──────────────────────────────────┐   │
│  │ _pandoc_to_pdf()                 │   │
│  │ - Generate LaTeX header          │   │
│  │ - Run pandoc with xelatex        │   │
│  │ - Use DejaVu fonts               │   │
│  └──────────────────────────────────┘   │
├─────────────────────────────────────────┤
│      Upload Layer                      │
│  ┌──────────────────────────────────┐   │
│  │ _remote_file_exists()            │   │
│  │ - Check if PDF exists on device  │   │
│  └──────────────────────────────────┘   │
│  ┌──────────────────────────────────┐   │
│  │ _upload_pdf()                    │   │
│  │ - Upload via rmapi put           │   │
│  └──────────────────────────────────┘   │
└─────────────────────────────────────────┘
```

### Workflow (per file)

1. **Convert `.md` → `.pdf`** using pandoc with xelatex + DejaVu fonts (Unicode-safe)
2. **Check if destination PDF exists** on device (via `rmapi ls` or `rmapi get`)
3. **Upload via `rmapi put`** to `ai/YYYY/MM/DD/` directory (use `--force` to overwrite)
4. **Cleanup** temporary directory

### The Value of Documentation on Tablet: A Deeper Context

You might question whether uploading documentation to a tablet is worth the effort—after all, you can read Markdown on your computer. But there's significant value in having technical documentation on a focused reading device, and this value multiplies when the workflow is frictionless (which is what remarkable_upload.py provides).

**Why read documentation on a ReMarkable tablet?**

1. **Focused reading environment**: No notifications, no browser tabs, no distractions. When reviewing a complex design document, this focus is invaluable. You can think deeply about architecture decisions without context-switching to Slack, email, or code.

2. **Active reading and annotation**: Reading on screen encourages passive consumption. Reading on tablet encourages active engagement—you naturally highlight key points, add marginal notes, draw diagrams. These annotations become valuable context when you return to implementation.

3. **Mobility and convenience**: Review design docs during commute, read meeting notes while away from desk, annotate research papers on the couch. The tablet enables reading in contexts where a laptop is impractical.

4. **Better retention**: The act of writing (even stylus on tablet) engages different cognitive pathways than typing. Annotating a design doc by hand often leads to better understanding and retention than just reading.

5. **Asynchronous review**: Your team creates a design doc. You download to tablet, review thoroughly with annotations, then share feedback. The tablet enables thoughtful review rather than quick skim-and-comment.

**The Workflow Integration Story:**

remarkable_upload.py emerged from real workflow needs. Consider this scenario (why this script was created):

You're working on a complex ticket (RMQ-0001: Build remarquee tool). Over several days, you create:
- Design documents explaining architecture (10 pages)
- Analysis documents reviewing existing code (20 pages)
- Reference documents for APIs (30 pages)
- Meeting notes and decisions (5 pages)

That's 65 pages of Markdown documentation. You want to review this on your tablet because:
- It's easier to see the big picture when reading sequentially
- You can annotate connections between documents
- You can spot inconsistencies that weren't obvious while writing
- The focused environment helps you think strategically (not just tactically)

Without automation, you'd need to:
1. Open each Markdown file
2. Convert to PDF (remember pandoc command)
3. Upload via rmapi (remember syntax)
4. Repeat 65 times
5. Hope you didn't create duplicates
6. Somehow organize the files

With remarkable_upload.py:
```bash
$ python remarkable_upload.py --ticket RMQ-0001
```

Done. All documents converted, uploaded, organized by date, no duplicates. This transforms documentation review from a tedious chore into a simple step in your workflow.

**Integration with docmgr:**

The script's ticket-awareness isn't accidental—it's designed for docmgr workflows specifically. docmgr creates structured documentation with:
- YAML frontmatter (metadata for docmgr's knowledge base)
- Ticket organization (hierarchical directory structure)
- Date-based folders (temporal organization)

remarkable_upload.py respects this structure:
- Handles docmgr's YAML frontmatter gracefully (strips it)
- Understands ticket directory layout (finds tickets automatically)
- Infers dates from ticket paths (maintains temporal context)

This tight integration means documentation created in docmgr flows seamlessly to tablet, where it can be reviewed, annotated, and those annotations can flow back (via remarks) into your knowledge base. This is the vision: **frictionless bidirectional flow between development documentation and focused reading/annotation**.

For the remarquee unified tool, remarkable_upload.py provides a template: understand user workflows deeply, remove friction points, integrate with existing tools, provide sensible defaults while allowing customization. These principles should guide remarquee's design.

## Key Functions and Classes

### Data Structures

#### `UploadTarget` (dataclass)
```python
@dataclass(frozen=True)
class UploadTarget:
    md_path: Path      # Path to source markdown file
    pdf_name: str      # Name of PDF file (with .pdf extension)
```

### Core Functions

#### `main(argv: Optional[list[str]] = None) -> int`
Main entry point:
- Parses command-line arguments
- Resolves ticket directory (via `--ticket-dir`, `--ticket/--root`, or script location)
- Infers or validates date for remote directory
- Builds list of upload targets
- Processes each target:
  - Validates markdown file exists
  - Checks if PDF exists on device (unless `--force` or `--pdf-only`)
  - Converts markdown to PDF
  - Uploads PDF (unless `--pdf-only`)
- Returns exit code (0 = success, 2 = error)

#### `_run(argv, *, check=True, capture=False, cwd=None) -> CompletedProcess`
Wrapper around `subprocess.run()`:
- Executes shell commands
- Handles `FileNotFoundError` → `RuntimeError` conversion
- Supports capture mode for output
- Supports custom working directory

#### `_pandoc_to_pdf(md_path: Path, out_pdf: Path) -> None`
Converts markdown to PDF:
1. Reads markdown file
2. Strips YAML frontmatter
3. Normalizes list spacing
4. Writes preprocessed markdown to temp file
5. Creates LaTeX header file with:
   - `enumitem` package for better list formatting
   - Custom margins and spacing
6. Runs pandoc with:
   - `--pdf-engine=xelatex`
   - `--standalone`
   - Custom header file (`-H`)
   - DejaVu Sans fonts (`-V mainfont=DejaVu Sans`)
   - Geometry settings (`-V geometry:margin=1in`)
7. Cleans up temp files

#### `_strip_yaml_frontmatter(md_text: str) -> str`
Removes YAML frontmatter block:
- Looks for `---\n` at start
- Finds closing `---\n`
- Returns body text (everything after closing delimiter)
- Returns input unchanged if no frontmatter found

**Rationale:**
- Prevents pandoc from failing on invalid YAML (e.g., unquoted `:` in scalars)
- Prevents frontmatter metadata from appearing in rendered PDF
- Matches docmgr's strict delimiter logic

#### `_normalize_list_spacing(md_text: str) -> str`
Ensures proper spacing before lists:
- Detects bullet lists (`-`, `*`, `+` followed by space)
- Detects numbered lists (digits followed by `.` and space)
- Inserts blank line before list items that aren't preceded by blank line
- Preserves existing list structure

**Rationale:**
- pandoc requires blank lines before lists for proper recognition
- Ensures consistent formatting across different markdown sources

#### `_infer_ticket_date_from_path(ticket_dir: Path) -> Optional[str]`
Infers date from ticket directory path:
- Looks for pattern: `.../ttmp/YYYY/MM/DD/<ticket>`
- Returns `"YYYY/MM/DD"` if found, `None` otherwise
- Validates that YYYY, MM, DD are 4, 2, 2 digits respectively

#### `_default_date(ticket_dir: Path) -> str`
Determines default date:
- Tries to infer from ticket directory path
- Falls back to today's date (`YYYY/MM/DD`)

#### `_normalize_rm_dir(date_ymd: str) -> str`
Normalizes date format for remote directory:
- Accepts `YYYY/MM/DD` or `YYYY-MM-DD`
- Converts to `YYYY/MM/DD` format
- Validates format (4-digit year, 2-digit month/day)
- Returns `"ai/YYYY/MM/DD/"` (with trailing slash)

#### `_targets_from_args(md_files: list[str], *, ticket_dir: Path) -> list[UploadTarget]`
Builds list of upload targets:
- If `md_files` provided: creates targets from file paths
- If empty: uses default documents from ticket:
  - `reference/01-bug-report-doc-relate-fails-on-non-docmgr-markdown-files.md`
  - `analysis/02-code-flow-analysis-frontmatter-validation-failure.md`
- Resolves relative paths to absolute
- Generates PDF names from markdown filenames

#### `_rm_ls(remote_dir: str) -> tuple[int, str]`
Runs `rmapi ls` command:
- Returns `(exit_code, output)`
- Captures stdout/stderr

#### `_rm_get(remote_path: str) -> int`
Runs `rmapi get` command:
- Returns exit code
- Used as existence probe (may download file)

#### `_remote_file_exists(remote_dir: str, pdf_name: str) -> bool`
Checks if PDF exists on device:
1. Tries `rmapi ls` (cheap, doesn't download)
   - Checks if `pdf_name` appears in output
2. Falls back to `rmapi get` (may download)
   - Checks if exit code is 0

**Note:** Conservative matching - requires exact filename match in `ls` output.

#### `_upload_pdf(local_pdf: Path, remote_dir: str, *, force: bool) -> None`
Uploads PDF via rmapi:
- Runs `rmapi put <local_pdf> <remote_dir>`
- Adds `--force` flag if `force=True`

#### `_ensure_file_exists(p: Path) -> None`
Validates file exists:
- Raises `FileNotFoundError` if path doesn't exist or isn't a file

## Command-Line Interface

### Arguments

| Argument | Type | Description |
|----------|------|-------------|
| `md` | `*` (positional) | Markdown files to upload. If omitted, uploads ticket's default documents. |
| `--ticket-dir` | path | Ticket directory to use for default documents. |
| `--ticket` | string | Ticket ID to locate under `--root` (best-effort name match). |
| `--root` | path | Docs root directory to search for tickets (default: `ttmp`). |
| `--date` | string | Destination date folder under `ai/` (`YYYY/MM/DD` or `YYYY-MM-DD`). Defaults to ticket date if inferable, else today. |
| `--force` | flag | Overwrite existing PDFs on device (`rmapi put --force`). |
| `--dry-run` | flag | Print what would be done, but don't run pandoc/rmapi. |
| `--pdf-only` | flag | Only generate PDF, don't upload to reMarkable. PDF saved to current directory or `--output-dir`. |
| `--output-dir` | path | Output directory for PDF when using `--pdf-only` (default: current directory). |

### Ticket Directory Resolution

The script resolves ticket directory in this order:

1. **`--ticket-dir`**: Explicit path (resolved relative to CWD if not absolute)
2. **`--ticket` + `--root`**: Searches `--root` directory for matching ticket folder
   - Looks for directories containing ticket ID in name
   - Requires `index.md` file to exist
3. **Script location**: If script lives under `<ticket>/scripts/`, uses parent directory
   - Falls back if script is installed in PATH

### Date Resolution

1. **`--date`**: Explicit date (validated and normalized)
2. **Ticket directory inference**: Extracts date from path `.../ttmp/YYYY/MM/DD/<ticket>`
3. **Today's date**: Falls back to current date (`YYYY/MM/DD`)

## Markdown Preprocessing (Fixing Common Issues)

### The YAML Frontmatter Problem

**What is YAML frontmatter?**

Many Markdown tools (docmgr, Obsidian, Jekyll) use YAML frontmatter for metadata:

```markdown
---
Title: My Document
Status: active
Topics:
  - backend
RelatedFiles: []
---
# Document Content

Body text here...
```

**Why is it a problem for PDF conversion?**

1. **Pandoc tries to parse it**: Expects strict YAML syntax
2. **Common YAML errors in docmgr**:
   - Unquoted colons: `Title: API Design: Key Decisions` (breaks YAML)
   - Unquoted special chars: `Note: Don't use this: it breaks!`
   - Complex nested structures (easy to get wrong)
3. **Unwanted in PDF**: Frontmatter is metadata, not content
4. **Rendering unpredictable**: May appear as title page, or cause errors

**Example of broken YAML:**
```markdown
---
Title: API Design: Key Decisions
Note: Don't use this: it breaks
RelatedFiles: [file.go:Description with: colons]
---
```

Pandoc error: `YAML parse error: mapping values are not allowed here`

**Solution: Strip it entirely before pandoc**

Pseudocode:
```
FUNCTION _strip_yaml_frontmatter(md_text):
    IF NOT md_text.startswith("---"):
        RETURN md_text  // No frontmatter
    
    lines = md_text.splitlines()
    
    // Find closing delimiter (second "---")
    FOR i FROM 1 TO len(lines):
        IF lines[i].strip() == "---":
            body_lines = lines[i+1:]
            RETURN "\n".join(body_lines).lstrip("\n")
    
    // No closing delimiter found (malformed)
    RETURN md_text
```

**Example transformation:**
```markdown
Before:
---
Title: Design Doc
Topics:
  - backend
  - api
---
# Architecture

This is the content.

After:
# Architecture

This is the content.
```

**Why this approach?**

- **Simple**: No YAML parsing (avoids complexity)
- **Fast**: Single pass through lines
- **Robust**: Handles malformed YAML gracefully
- **Clean PDFs**: No metadata visible

### The List Spacing Problem

**What pandoc expects:**

Markdown spec (original) requires **blank line** before lists:

```markdown
Correct:
Paragraph text.

- List item 1
- List item 2

Incorrect (for pandoc):
Paragraph text.
- List item 1
- List item 2
```

**Why this matters:**

Without blank line, pandoc treats `-` as part of previous paragraph!

Result: `Paragraph text. - List item 1 - List item 2` (all one line)

**Solution: Auto-insert blank lines**

Pseudocode:
```
FUNCTION _normalize_list_spacing(md_text):
    lines = md_text.splitlines()
    result = []
    
    FOR i, line IN ENUMERATE(lines):
        stripped = line.lstrip()
        
        // Detect list items
        is_list_item = FALSE
        IF stripped starts with ("- ", "* ", "+ "):
            is_list_item = TRUE  // Bullet list
        ELSE IF stripped matches "\\d+\\. ":
            is_list_item = TRUE  // Numbered list (e.g., "1. ", "2. ")
        
        // Insert blank line if needed
        IF is_list_item AND i > 0:
            prev_line = lines[i-1].strip()
            prev_is_list = CheckIfListItem(prev_line)
            
            IF prev_line AND NOT prev_is_list:
                result.append("")  // Blank line
        
        result.append(line)
    
    RETURN "\n".join(result)
```

**Example transformation:**
```markdown
Before:
Here's my list:
- Item 1
- Item 2
Another paragraph.
1. First
2. Second

After:
Here's my list:

- Item 1
- Item 2
Another paragraph.

1. First
2. Second
```

**Edge cases handled:**
- Consecutive lists: No extra blank lines
- Already-spaced lists: No change
- Nested lists: Preserves indentation
- Mixed types: Each list type handled separately

## PDF Conversion Pipeline (The Details)

### Why pandoc + xelatex?

**The problem**: Converting Markdown to PDF is harder than it seems:
- Need to handle Unicode (accented chars, symbols, emojis)
- Need good typography (proper fonts, spacing, margins)
- Need to support Markdown extensions (tables, footnotes, math)
- Need reliable rendering (no broken layouts)

**The solution**: pandoc + xelatex is the gold standard:
- **pandoc**: Universal document converter, understands many Markdown flavors
- **xelatex**: Modern LaTeX engine with native Unicode support
- **LaTeX**: Professional typesetting system (publications-quality output)

**Alternatives (why not use these?):**
- `wkhtmltopdf`: Uses WebKit, but deprecated, buggy font handling
- `weasyprint`: CSS-based, but different aesthetic, limited font control
- `pdfkit`: Wrapper around wkhtmltopdf (same issues)
- `reportlab`: Python library, but requires manual layout code

### LaTeX Header Generation (Customizing Typography)

The script generates a custom LaTeX header to improve formatting:

```latex
\usepackage{enumitem}
\setlist[itemize]{leftmargin=*,topsep=0.5em,itemsep=0.3em,parsep=0.2em}
\setlist[enumerate]{leftmargin=*,topsep=0.5em,itemsep=0.3em,parsep=0.2em}
\usepackage{geometry}
\geometry{margin=1in}
```

**What each line does:**

- `\usepackage{enumitem}`: Load list customization package
- `\setlist[itemize]{...}`: Configure bullet lists:
  - `leftmargin=*`: Auto-calculate margin (aligns with text)
  - `topsep=0.5em`: Space above/below list (0.5× font size)
  - `itemsep=0.3em`: Space between items
  - `parsep=0.2em`: Space between paragraphs in items
- `\setlist[enumerate]{...}`: Same for numbered lists
- `\geometry{margin=1in}`: Set page margins to 1 inch (all sides)

**Why customize lists?**

Default LaTeX list spacing is generous (designed for print). For tablets:
- Tighter spacing = more content per page
- Better readability on smaller screens
- Consistent with digital documents

**Visual comparison:**

```
Default LaTeX:       Custom spacing:
┌─────────────┐      ┌─────────────┐
│ Text        │      │ Text        │
│             │      │             │
│  • Item 1   │      │  • Item 1   │
│             │      │  • Item 2   │
│  • Item 2   │      │  • Item 3   │
│             │      │             │
│  • Item 3   │      │ More text   │
│             │      │             │
│ More text   │      └─────────────┘
└─────────────┘      (saves ~20% space)
```

### Pandoc Command (Complete Explanation)

```bash
pandoc preprocessed.md \
  -o output.pdf \
  --pdf-engine=xelatex \
  --standalone \
  -H custom-header.tex \
  -V mainfont="DejaVu Sans" \
  -V monofont="DejaVu Sans Mono" \
  -V geometry:margin=1in
```

**Option-by-option breakdown:**

- **`--pdf-engine=xelatex`**: Which LaTeX engine to use
  - **xelatex**: Modern, Unicode-native, supports TTF/OTF fonts
  - **pdflatex**: Legacy, requires font packages, no Unicode
  - **lualatex**: Alternative modern engine, but slower
  - **Why xelatex?** Best Unicode support, widely available

- **`--standalone`**: Generate complete document
  - **With `--standalone`**: Full LaTeX document with preamble, \begin{document}, etc.
  - **Without**: Just fragments (would fail for PDF)
  - **Always use for PDF output**

- **`-H custom-header.tex`**: Include custom LaTeX header
  - Inserted into document preamble (before \begin{document})
  - Used for package imports, formatting customization
  - Alternative: `-B` (before body), `-A` (after body)

- **`-V mainfont="DejaVu Sans"`**: Set main text font
  - **DejaVu Sans**: Sans-serif, Unicode-complete, free
  - **Why sans-serif?** Better readability on tablets
  - **Alternative fonts**: "Liberation Sans", "Noto Sans", "Arial"

- **`-V monofont="DejaVu Sans Mono"`**: Set code font
  - Used for inline code and code blocks
  - Monospace fonts ensure alignment
  - **DejaVu Sans Mono**: Pairs with DejaVu Sans

- **`-V geometry:margin=1in`**: Page margins
  - Sets all four margins (top, bottom, left, right)
  - `1in` = 2.54cm (comfortable reading margin)
  - **Alternative**: `margin=0.75in` (more content), `margin=1.5in` (more whitespace)

**Why DejaVu fonts specifically?**

- **Unicode coverage**: Supports 3,000+ glyphs (Latin, Greek, Cyrillic, symbols)
- **Free**: No licensing issues
- **Preinstalled**: Available on most Linux systems
- **Quality**: Derived from Vera fonts (professionally designed)
- **Tablet-friendly**: Clear rendering at tablet resolution

**Handling Unicode in PDFs:**

```
Without proper fonts:           With DejaVu:
┌────────────────────┐        ┌────────────────────┐
│ Café → Caf�        │        │ Café → Café        │
│ ½ → ?              │        │ ½ → ½              │
│ → → ?              │        │ → → →              │
│ 中文 → ??          │        │ 中文 → 中文        │
└────────────────────┘        └────────────────────┘
```

DejaVu handles Latin extended, but for CJK (Chinese/Japanese/Korean), you'd need:
```bash
-V CJKmainfont="Noto Sans CJK"
```

### Conversion Process (Complete Flow)

Pseudocode:
```
FUNCTION _pandoc_to_pdf(md_path, out_pdf):
    // 1. Read and preprocess markdown
    md_text = ReadFile(md_path)
    body_text = _strip_yaml_frontmatter(md_text)
    body_text = _normalize_list_spacing(body_text)
    
    // 2. Write preprocessed markdown to temp file
    temp_md = out_pdf.with_suffix(".input.md")
    WriteFile(temp_md, body_text)
    
    // 3. Generate LaTeX header
    header_content = """
    \\usepackage{enumitem}
    \\setlist[itemize]{leftmargin=*,topsep=0.5em,...}
    \\setlist[enumerate]{leftmargin=*,topsep=0.5em,...}
    \\usepackage{geometry}
    \\geometry{margin=1in}
    """
    header_file = out_pdf.with_suffix(".header.tex")
    WriteFile(header_file, header_content)
    
    // 4. Run pandoc
    RunCommand([
        "pandoc",
        temp_md,
        "-o", out_pdf,
        "--pdf-engine=xelatex",
        "--standalone",
        "-H", header_file,
        "-V", "mainfont=DejaVu Sans",
        "-V", "monofont=DejaVu Sans Mono",
        "-V", "geometry:margin=1in"
    ])
    
    // 5. Cleanup temp files
    DeleteFile(temp_md)
    DeleteFile(header_file)
```

**What pandoc does internally:**

```
1. Parse Markdown → Abstract Syntax Tree (AST)
2. Convert AST → LaTeX code
3. Insert custom header (-H)
4. Write LaTeX file (temporary)
5. Run: xelatex latex-file.tex
6. xelatex generates: output.pdf
7. pandoc cleans up LaTeX temp files
```

**Typical conversion time:**
- Small document (< 10 pages): 2-5 seconds
- Large document (50+ pages): 10-30 seconds
- Bottleneck: xelatex compilation (CPU-intensive)

**Troubleshooting:**

If pandoc fails, check:
1. `pandoc --version` (need 2.0+)
2. `xelatex --version` (need TeX Live)
3. Font availability: `fc-list | grep DejaVu`
4. Temp directory writable
5. Markdown syntax (run `pandoc -t latex` to see LaTeX output)

## Integration with rmapi (How Upload Works)

### Understanding rmapi Integration

The script uses `rmapi` as a command-line tool (not as a library). This means:
- **Pros**: Don't need to implement auth, sync, upload logic
- **Cons**: Must shell out to rmapi binary, parse its output

**Design decision**: Favor simplicity over performance. For occasional uploads, this is fine.

### File Existence Check (Avoiding Duplicates)

**Why check existence?**

Without checking:
```bash
$ python remarkable_upload.py doc.md
OK: uploaded
$ python remarkable_upload.py doc.md  # Oops, ran again
OK: uploaded  # Created duplicate!
```

With checking:
```bash
$ python remarkable_upload.py doc.md
OK: uploaded
$ python remarkable_upload.py doc.md
SKIP: Already exists. Use --force to overwrite.
```

**Implementation strategy:**

Pseudocode:
```
FUNCTION _remote_file_exists(remote_dir, pdf_name):
    // Strategy 1: Try ls (cheap, doesn't download)
    exit_code, output = RunCommand(["rmapi", "ls", remote_dir], capture=True)
    
    IF exit_code == 0:
        // ls succeeded, check output
        IF pdf_name IN output:
            RETURN True  // File exists
        ELSE:
            RETURN False  // File doesn't exist
    ELSE:
        // ls failed (directory might not exist)
        // Fall back to direct existence probe
        
        // Strategy 2: Try get (may download)
        remote_path = remote_dir + "/" + pdf_name
        exit_code = RunCommand(["rmapi", "get", remote_path], capture=True)
        
        IF exit_code == 0:
            RETURN True  // File exists (and was downloaded)
        ELSE:
            RETURN False  // File doesn't exist
```

**Trade-offs:**

- **`rmapi ls`**:
  - Pros: Fast, doesn't download
  - Cons: Directory must exist, output format varies
  
- **`rmapi get`**:
  - Pros: Definitive existence check
  - Cons: Downloads file (slow, uses bandwidth)

**When fallback triggers:**

```bash
# Directory doesn't exist yet
$ rmapi ls ai/2025/12/14/
ERROR: directory doesn't exist

# Script falls back to:
$ rmapi get ai/2025/12/14/doc.pdf
ERROR: entry 'ai/2025/12/14/doc.pdf' doesn't exist
# Exit code: 1 → File doesn't exist
```

**Limitation**: `rmapi ls` output parsing is brittle. If rmapi changes output format, detection might break.

### Upload Process (The Final Step)

Pseudocode:
```
FUNCTION _upload_pdf(local_pdf, remote_dir, force):
    command = ["rmapi", "put", str(local_pdf), remote_dir]
    IF force:
        command.append("--force")
    
    RunCommand(command, check=True)
    // Raises exception if fails
```

**What `rmapi put` does:**

```bash
$ rmapi put document.pdf ai/2025/12/14/

# rmapi internally:
1. Reads document.pdf
2. Computes SHA256 hash
3. Uploads blob to cloud
4. Creates document entry
5. Updates root index
6. Triggers tablet sync
```

**Remote directory organization:**

The script uploads to `ai/YYYY/MM/DD/` (with trailing slash):

```
Tablet directory structure:
/
├── Books/
├── Papers/
└── ai/                  ← Script's upload location
    ├── 2025/
    │   ├── 12/
    │   │   ├── 13/
    │   │   │   ├── doc1.pdf
    │   │   │   └── doc2.pdf
    │   │   └── 14/
    │   │       ├── design-doc.pdf
    │   │       └── analysis.pdf
    │   └── ...
    └── ...
```

**Why `ai/` prefix?**

- **ai** = "artificial intelligence" or "automated input"
- Separates automated uploads from manual documents
- Easy to find (top of alphabetical sort)
- Can bulk-delete if needed

**Why date-based organization?**

- **Chronological**: Natural browsing (recent documents first)
- **Context**: Know when document was uploaded
- **Grouping**: Related documents (same day) together
- **Cleanup**: Easy to delete old uploads (entire day folder)

**Force flag behavior:**

```bash
# Without --force (default):
$ python remarkable_upload.py doc.md
SKIP: ai/2025/12/14/doc.pdf already exists

# With --force:
$ python remarkable_upload.py doc.md --force
OK: uploaded (replaced existing)
```

**What --force does in rmapi:**

```
rmapi put --force:
1. Check if document exists
2. If exists: DELETE old document (completely, including annotations!)
3. Upload new document (fresh UUID)
4. Update root index
```

**Warning**: `--force` destroys annotations! If you annotated the PDF on tablet, those are lost.

## Developer Usage Guide

### Basic Usage (Common Scenarios)

**Scenario 1: Upload a single markdown file**

```bash
# Simplest usage
$ python remarkable_upload.py meeting-notes.md

# What happens:
# 1. Strips YAML frontmatter (if any)
# 2. Normalizes list spacing
# 3. Converts to PDF: meeting-notes.pdf
# 4. Checks tablet: ai/2025/12/14/meeting-notes.pdf exists?
# 5. If not, uploads to ai/2025/12/14/
# 6. Cleanup temp files

Output:
Ticket dir: /home/user/current-dir
Remote dir: ai/2025/12/14/
Force: False
Dry-run: False

=== meeting-notes.md ===
PDF name: meeting-notes.pdf
OK: uploaded meeting-notes.pdf -> ai/2025/12/14/
```

**Scenario 2: Upload multiple files at once**

```bash
$ python remarkable_upload.py doc1.md doc2.md doc3.md

# Processes each file sequentially:
# - Converts doc1.md → doc1.pdf → uploads
# - Converts doc2.md → doc2.pdf → uploads
# - Converts doc3.md → doc3.pdf → uploads
```

**Scenario 3: Organize uploads by custom date**

```bash
# Upload to specific date folder
$ python remarkable_upload.py notes.md --date 2025/12/01

# Now on tablet:
# ai/2025/12/01/notes.pdf

# Why? Maybe you're organizing by project dates, not upload dates
```

**Scenario 4: Overwrite existing PDF (dangerous!)**

```bash
# First upload
$ python remarkable_upload.py doc.md
OK: uploaded

# Fix typo, upload again
$ python remarkable_upload.py doc.md
SKIP: ai/2025/12/14/doc.pdf already exists. Use --force to overwrite.

# Force overwrite (loses annotations!)
$ python remarkable_upload.py doc.md --force
OK: uploaded (replaced existing)
```

**Warning**: `--force` deletes the old document completely, including any annotations you made on the tablet!

**Scenario 5: Preview without uploading (dry-run)**

```bash
$ python remarkable_upload.py doc.md --dry-run

=== doc.md ===
PDF name: doc.pdf
DRY: pandoc /path/to/doc.md -> /tmp/.../doc.pdf (xelatex, DejaVu fonts)
DRY: rmapi put /tmp/.../doc.pdf ai/2025/12/14/

# Nothing actually executed, just shows what would happen
# Useful for testing, debugging
```

**Scenario 6: Generate PDF only (no upload)**

```bash
# Generate PDF to current directory
$ python remarkable_upload.py doc.md --pdf-only
OK: generated doc.pdf

# Generate to specific directory
$ python remarkable_upload.py doc.md --pdf-only --output-dir ~/pdfs/
OK: generated /home/user/pdfs/doc.pdf

# Use cases:
# - Preview PDF before uploading
# - Share PDF with others (email, print)
# - Archive documentation as PDF
```

### Ticket-Based Usage (docmgr Integration)

**Scenario 1: Upload from ticket directory**

```bash
# When script is in ticket's scripts/ folder
$ cd ttmp/2025/12/14/RMQ-0001--ticket-name/scripts/
$ python remarkable_upload.py

# Uses default documents:
# - reference/01-bug-report.md
# - analysis/02-code-flow.md
# (These are hardcoded defaults for a specific ticket)
```

**Scenario 2: Specify ticket directory explicitly**

```bash
$ python remarkable_upload.py \
  --ticket-dir ttmp/2025/12/14/RMQ-0001--ticket-name/

# Uses default documents from this ticket
```

**Scenario 3: Find ticket by ID**

```bash
$ python remarkable_upload.py --ticket RMQ-0001 --root ttmp

# Script searches:
# 1. Look in ttmp/ recursively
# 2. Find directories containing "rmq-0001" (case-insensitive)
# 3. Check for index.md (validates it's a ticket)
# 4. Use first match

# Found: ttmp/2025/12/14/RMQ-0001--ticket-name/
# Infers date: 2025/12/14 (from path)
# Uploads to: ai/2025/12/14/
```

**Scenario 4: Upload specific files from ticket**

```bash
$ python remarkable_upload.py \
  --ticket RMQ-0001 \
  reference/01-design.md \
  analysis/02-implementation.md

# Uploads these specific files (not defaults)
# Still uses ticket date (2025/12/14)
```

### Practical Workflow Examples

**Daily documentation workflow:**

```bash
#!/bin/bash
# upload-docs.sh - Run after updating documentation

# Find all modified markdown files in ticket
cd ttmp/2025/12/14/RMQ-0001--ticket-name/

# Upload all markdown files in reference/ and design/
python remarkable_upload.py \
  reference/*.md \
  design/*.md \
  --force  # Always update (docs change frequently)

echo "Documentation synced to tablet!"
```

**Git hook integration:**

```bash
#!/bin/bash
# .git/hooks/post-commit

# Upload documentation when committed
CHANGED_MD=$(git diff --name-only HEAD~1 HEAD | grep '\.md$')

if [ -n "$CHANGED_MD" ]; then
  echo "Uploading changed markdown files..."
  python remarkable_upload.py $CHANGED_MD --date $(date +%Y/%m/%d)
fi
```

**Cron job for nightly sync:**

```bash
# Upload all ticket docs nightly
0 2 * * * cd /path/to/repo && \
  find ttmp -name "*.md" -mtime -1 | \
  xargs python remarkable_upload.py --force
```

### Programmatic Usage

```python
from pathlib import Path
from remarkable_upload import main

# Run with custom arguments
exit_code = main([
    "document.md",
    "--date", "2025/12/14",
    "--force",
    "--pdf-only",
    "--output-dir", "/tmp/pdfs"
])
```

### Integration Examples

**As a git hook:**
```bash
#!/bin/bash
# .git/hooks/post-commit

# Upload documentation when committed
python remarkable_upload.py \
  --ticket-dir "$(git rev-parse --show-toplevel)" \
  --date "$(date +%Y/%m/%d)"
```

**As a docmgr plugin:**
```python
# After creating/updating document
import subprocess
subprocess.run([
    "python", "remarkable_upload.py",
    str(document_path),
    "--ticket-dir", str(ticket_dir),
    "--date", date_str
])
```

## Error Handling

### Common Errors

**FileNotFoundError:**
- Markdown file doesn't exist
- Ticket directory not found
- Command not found (pandoc, rmapi, xelatex)

**RuntimeError:**
- Command execution failed (wrapped FileNotFoundError)

**ValueError:**
- Invalid date format
- Invalid ticket directory structure

**Exit Codes:**
- `0`: Success
- `2`: Error (file not found, ticket not found, etc.)

### Error Messages

The script provides clear error messages:
- `ERROR: markdown file not found: <path>`
- `ERROR: docs root not found: <path>`
- `ERROR: ticket not found under <root> matching <ticket>`
- `SKIP: <path> already exists on reMarkable. Re-run with --force to overwrite.`

## File Organization

### Script Structure

```
remarkable_upload.py
├── Imports and dependencies
├── UploadTarget dataclass
├── Helper functions
│   ├── _run()                    # Subprocess wrapper
│   ├── _infer_ticket_date_from_path()
│   ├── _default_date()
│   ├── _normalize_rm_dir()
│   ├── _strip_yaml_frontmatter()
│   ├── _normalize_list_spacing()
│   ├── _pandoc_to_pdf()
│   ├── _rm_ls()
│   ├── _rm_get()
│   ├── _remote_file_exists()
│   ├── _upload_pdf()
│   ├── _targets_from_args()
│   └── _ensure_file_exists()
└── main()                        # Entry point
```

### Temporary Files

**During PDF conversion:**
- `{pdf_name}.input.md`: Preprocessed markdown (YAML stripped, lists normalized)
- `{pdf_name}.header.tex`: Generated LaTeX header

**During upload:**
- Temporary directory: `{tempdir}/{pdf_name}.pdf`
- Cleaned up after upload (unless `--pdf-only`)

## Quick Reference

### Key Functions

**Main Processing:**
- `main(argv)`: Entry point, argument parsing, workflow orchestration

**Preprocessing:**
- `_strip_yaml_frontmatter(md_text)`: Remove YAML frontmatter
- `_normalize_list_spacing(md_text)`: Normalize list spacing

**PDF Conversion:**
- `_pandoc_to_pdf(md_path, out_pdf)`: Convert markdown to PDF

**rmapi Integration:**
- `_remote_file_exists(remote_dir, pdf_name)`: Check if PDF exists
- `_upload_pdf(local_pdf, remote_dir, force)`: Upload PDF

**Utilities:**
- `_run(argv, check, capture, cwd)`: Subprocess wrapper
- `_infer_ticket_date_from_path(ticket_dir)`: Extract date from path
- `_normalize_rm_dir(date_ymd)`: Normalize date format
- `_targets_from_args(md_files, ticket_dir)`: Build upload targets

### Key Data Structures

- `UploadTarget`: `(md_path: Path, pdf_name: str)`

### Command-Line Options

**Input:**
- `md`: Markdown files (positional, optional)
- `--ticket-dir`: Explicit ticket directory
- `--ticket` + `--root`: Find ticket by ID

**Output:**
- `--date`: Remote directory date
- `--output-dir`: PDF output directory (`--pdf-only` mode)

**Behavior:**
- `--force`: Overwrite existing PDFs
- `--dry-run`: Preview without executing
- `--pdf-only`: Generate PDF only, don't upload

### Dependencies

**Required:**
- `pandoc`: Markdown converter
- `xelatex`: PDF engine (TeX Live)
- `rmapi`: ReMarkable Cloud API CLI

**Python:**
- Standard library only (`argparse`, `subprocess`, `pathlib`, `tempfile`, `shutil`, `datetime`)

## Related

- **rmapi**: See `01-rmapi-api-overview-architecture-auth-transport-shell-commands.md`
- **pandoc**: https://pandoc.org/
- **XeLaTeX**: Part of TeX Live distribution
- **DejaVu Fonts**: https://dejavu-fonts.github.io/
