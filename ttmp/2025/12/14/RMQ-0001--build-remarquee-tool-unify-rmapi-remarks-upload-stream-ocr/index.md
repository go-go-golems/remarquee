---
Title: Build remarquee tool (unify rmapi/remarks/upload/stream/OCR)
Ticket: RMQ-0001
Status: complete
Topics:
    - backend
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: 'Analysis phase for building unified remarquee toolkit: comprehensive documentation of rmapi (cloud sync), remarks (annotation extraction), remarkable_upload.py (markdown→PDF pipeline), and goMarkableStream (real-time streaming). Includes architecture, protocols, APIs, workflows, and integration strategy.'
LastUpdated: 2026-01-12T16:16:51.511588058-05:00
WhatFor: ""
WhenToUse: ""
---



# Build remarquee tool (unify rmapi/remarks/upload/stream/OCR)

## Overview

### Vision: A Unified Toolkit for ReMarkable Tablets

The remarquee project aims to create a comprehensive, open-source toolkit for working with ReMarkable tablets. Currently, the ecosystem consists of several excellent but independent tools, each solving a specific problem. The goal of remarquee is to unify these tools into a cohesive system that's more powerful than the sum of its parts.

### The Current Landscape (Four Independent Tools)

**1. rmapi - Cloud Synchronization**  
A Go application that implements ReMarkable's Cloud API (Sync15 protocol). It's the data access layer—handling upload, download, and management of documents in ReMarkable's cloud storage. Think of it as "rsync for ReMarkable" or "git for documents."

**Key capabilities:**
- Upload/download documents programmatically
- Organize documents in folders
- Handle document metadata and versions
- Sync across multiple devices
- Non-interactive mode for automation

**2. remarks - Annotation Extraction**  
A Python package that parses ReMarkable's proprietary `.rm` annotation format and converts it to standard formats (PDF, Markdown). It's the content intelligence layer—understanding what's *inside* documents.

**Key capabilities:**
- Parse binary `.rm` files (drawings, highlights, typed text)
- Extract highlights as searchable text
- Generate Obsidian-compatible Markdown
- Create annotated PDFs with highlights
- Handle coordinate transformations

**3. remarkable_upload.py - Documentation Pipeline**  
A Python script that converts Markdown documentation to PDF and uploads it to tablets. It's the "write path"—taking content you create on your computer and making it tablet-readable.

**Key capabilities:**
- Convert Markdown to professional PDFs
- Handle docmgr/Obsidian YAML frontmatter
- Organize uploads by date
- Avoid duplicates
- Integrate with ticket workflows

**4. goMarkableStream - Real-Time Streaming**  
A Go application that runs on the tablet and streams the screen to web browsers. It's the live access layer—providing real-time visibility into tablet activity.

**Key capabilities:**
- Stream tablet screen to browsers
- Forward pen/touch events
- Support presentations (overlay mode)
- Enable remote collaboration
- Compress for bandwidth efficiency

### Why Unification Matters: The Gaps in the Current Approach

While each tool is excellent individually, using them together reveals friction points:

**Gap 1: Workflow Complexity**  
Currently, a complete workflow requires:
```bash
# Download from cloud
$ rmapi get document.pdf

# Extract annotations
$ remarks document.rmdoc output/

# Process results
$ cat output/document_obsidian.md

# Upload new document
$ pandoc notes.md -o notes.pdf
$ rmapi put notes.pdf

# Stream for presentation
$ ssh root@remarkable "./goMarkableStream"
```

Each step requires different commands, different syntax, different mental models. Users must understand four separate tools.

**Gap 2: Data Format Conversions**  
Each tool has its own data formats:
- rmapi works with `.rmdoc` files
- remarks expects xochitl directory structure
- remarkable_upload.py expects Markdown
- goMarkableStream works with raw framebuffer

Converting between these formats is manual and error-prone.

**Gap 3: No Shared Configuration**  
Each tool has different configuration mechanisms:
- rmapi: `~/.rmapi` (YAML)
- remarks: Command-line arguments
- remarkable_upload.py: Command-line arguments
- goMarkableStream: Environment variables

No unified configuration for credentials, preferences, paths.

**Gap 4: Limited Intelligence**  
None of the tools understand content semantically:
- Can't search handwritten notes (need OCR)
- Can't summarize annotations (need LLM)
- Can't extract structured data (need parsing + AI)
- Can't provide contextual suggestions (need understanding)

**Gap 5: No Bidirectional Workflow**  
The ideal workflow is bidirectional:

```
Create docs (Markdown) → Upload to tablet → Annotate → Download → Extract annotations → Integrate back into docs
```

Currently, this loop is manual and lossy (annotations don't flow back into source documents automatically).

### The remarquee Vision: Unified, Intelligent, Bidirectional

remarquee will unify these tools into a single system with:

**1. Unified CLI/API**  
```bash
$ remarquee sync            # Uses rmapi internally
$ remarquee extract doc.pdf # Uses remarks internally
$ remarquee upload notes.md # Uses remarkable_upload.py logic
$ remarquee stream          # Uses goMarkableStream patterns
$ remarquee ocr handwriting.pdf  # New: OCR via geppetto
```

**2. Shared Configuration**  
```yaml
# ~/.remarquee/config.yaml
cloud:
  credentials: ~/.rmapi
  auto_sync: true
extraction:
  output_format: markdown
  include_drawings: true
upload:
  default_folder: ai/{date}
  convert_markdown: true
streaming:
  compression: rle
  frame_rate: 200
ocr:
  provider: geppetto
  model: claude-3-opus
```

**3. Intelligent Processing**  
```bash
$ remarquee analyze document.pdf
# Uses remarks to extract annotations
# Uses geppetto (LLM) to summarize highlights
# Generates structured output (key points, action items, questions)
```

**4. Bidirectional Workflow**  
```bash
$ remarquee workflow create project-design.md
# 1. Converts to PDF
# 2. Uploads to tablet
# 3. Monitors for changes (via sync)
# 4. Downloads when annotated
# 5. Extracts annotations
# 6. Appends to original Markdown
# 7. Commits to git
```

**5. Observability**  
```bash
$ remarquee status
Cloud: Connected (5 documents pending sync)
Tablet: Online (streaming available)
Queue: 3 documents processing
Recent: Uploaded design-doc.pdf (2 min ago)
```

### Current Status: Analysis Phase Complete

This ticket is currently in the **analysis phase**. We've created comprehensive documentation for each tool:

1. **rmapi analysis**: Architecture, sync protocol, API reference (1,600+ lines)
2. **remarks analysis**: Parsing pipeline, conversion process (1,000+ lines)
3. **remarkable_upload.py analysis**: Markdown-to-PDF workflow (800+ lines)
4. **goMarkableStream analysis**: Streaming architecture, real-time processing (1,800+ lines)

These documents provide the foundation for integration. Next steps:
- Design unified API/CLI interface
- Identify integration points between tools
- Plan data flow through unified system
- Design geppetto integration for OCR/LLM features
- Prototype core workflows

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Design doc**: `design-doc/01-product-design-remarquee-capability-scope-and-ux-surfaces-cli-repl-web.md` (top-level product scope + CLI/REPL/Web taxonomy)
- **Implementation guide (CLI + cloud)**: `playbook/01-implementation-guide-remarquee-cli-cloud-module-glazed-cobra-rmapi.md`
- **Diary**: `reference/05-diary.md`

## Status

Current status: **complete**

## Topics

- backend

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbook/ - Command sequences, implementation guides, and operational procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
