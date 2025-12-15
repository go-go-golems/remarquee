---
Title: 'Upload next features: bundle PDFs w/ ToC, mirror dirs, syntax-highlight source'
Ticket: RMQ-0003
Status: active
Topics:
    - backend
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/remarquee/cmds/upload/md.go
      Note: Current upload implementation to extend (bundle + preserve-dirs)
    - Path: pkg/mdpdf/pandoc.go
      Note: Pandoc runner; needs ToC/highlight knobs for new features
    - Path: pkg/rmcloud/dirs.go
      Note: MkdirAll used by preserve-dirs/mirror mode
ExternalSources: []
Summary: 'Next feature batch for remarquee upload: bundle multiple inputs into one PDF with ToC, mirror directory structure on upload, and upload source files as syntax-highlighted PDFs.'
LastUpdated: 2025-12-14T21:17:52.248602214-05:00
---


# Upload next features: bundle PDFs w/ ToC, mirror dirs, syntax-highlight source

## Executive Summary

This document proposes the next batch of features for `remarquee upload`:

- **Bundle mode**: collate multiple markdown inputs into **one** PDF with a clickable **Table of Contents** (ToC) that jumps to each included document/section.
- **Directory mirroring**: when uploading a directory, optionally preserve the local relative directory structure as remote folders (a “sync/mirror” mode).
- **Source upload**: accept source code files (e.g. `foo.c`, `main.go`) and render them as PDFs with **syntax highlighting** before uploading.

The core constraints remain unchanged:
- PDF generation stays **pandoc + xelatex** (quality + Unicode).
- Cloud operations stay rmapi **in-process** (no shelling out).

## Problem Statement

The current `remarquee upload md` command works well for “upload a bunch of markdown files to a folder”, but it has three common workflow gaps:

1. **Reading many docs is cumbersome** on-device: users often want a single “packet” PDF with ToC for fast navigation (e.g. “all notes for a meeting”, “all docs for a PR”, “a directory of research notes”).
2. **Uploading a directory flattens structure**: today all documents go into one remote directory. In practice, users often organize notes by subfolder and want that structure preserved remotely (and created if missing).
3. **Source code isn’t supported**: users want to upload code files directly, with readable typography and syntax highlighting. Right now they must wrap code into markdown manually.

## Proposed Solution

### Command surface (proposed)

Keep a single `upload` group, add two new subcommands (and extend existing ones):

- **`remarquee upload md` (extend)**
  - Add `--preserve-dirs` (or `--mirror-dirs`) so directory traversal recreates relative subfolders remotely.
  - Add `--bundle` mode (optional) to collate into a single PDF instead of one PDF per markdown file.

- **`remarquee upload bundle` (new, explicit)**
  - Collates multiple markdown inputs (files/dirs) into one PDF with ToC.
  - More discoverable than a flag overload.

- **`remarquee upload src` (new)**
  - Accepts source files and/or directories.
  - Converts each source file to a PDF with syntax highlighting (one per file), then uploads.
  - Optional `--bundle` to collate multiple source files into one PDF (useful for small patches).

All commands keep:
- `--dry-run`, `--pdf-only`, `--output-dir`, `--force`, `--remote-dir`, `--date`
- rmapi auth flags `--non-interactive`, `--reauth`

### Feature 1: Bundle multiple docs into one PDF with ToC

#### Input ordering

Bundling must be deterministic:
- For directories: lexicographic ordering by relative path (case-insensitive), then by filename.
- For explicit file lists: preserve the user-given order.

#### How bundling works (pandoc-first)

Use pandoc’s native multi-input support and ToC generation.

We also need “section boundaries” so ToC entries are meaningful. Two options:
- **Option A (recommended)**: Generate a temporary wrapper markdown that includes each document as a section with a stable heading and anchor.
  - Pros: consistent section titles, even if docs start with `##` or no title.
  - Cons: requires concatenation/preprocessing step.
- **Option B**: Pass files directly; rely on each file’s headings.
  - Pros: simpler.
  - Cons: inconsistent if docs have different heading styles.

Recommendation: **Option A** for predictable ToC.

#### Output naming

In bundle mode:
- default output name:
  - if inputs are a single directory: `<dirname>.pdf`
  - else: `bundle.pdf`
- allow `--name <name>` to override (applies to the uploaded doc name too).

### Feature 2: Mirror directory structure on remote (recursive dir creation)

Add a flag (name TBD):
- `--preserve-dirs`: upload to `remoteDir/<relative-subdir>/...`

Behavior:
- For each local file selected under a root directory, compute `rel = path.Rel(rootDir, file)`.
- Remote directory = `remoteDir + "/" + filepath.Dir(rel)`.
- Ensure the remote directory exists via `pkg/rmcloud.MkdirAll(...)`.
- Upload into that directory.

Default remains **flat** upload (current behavior) to avoid accidental folder sprawl.

### Feature 3: Upload source code as syntax-highlighted PDFs

#### Approach (generate markdown, then reuse mdpdf + pandoc)

Keep the existing pipeline by transforming source files into markdown:

- For each source file:
  - Create markdown:
    - H1 title: file name (or relative path)
    - fenced code block annotated with language (based on extension)
- Convert via the existing `pkg/mdpdf` pandoc conversion.

This leverages pandoc syntax highlighting (`--highlight-style` and/or `--listings`) without adding a new renderer.

Proposed flags for `upload src`:
- `--theme <name>`: maps to pandoc `--highlight-style`.
- `--line-numbers`: enable line numbers (implementation depends on pandoc/LaTeX settings).
- `--title-mode name|path`: use basename vs relative path as the visible title.

### Package changes (proposed)

Extend existing packages:
- `pkg/mdpdf`:
  - add “bundle inputs” wrapper generation
  - add pandoc knobs for ToC (`--toc`, `--toc-depth`)
  - add pandoc knobs for highlighting (`--highlight-style`, maybe `--listings`)
- `pkg/rmcloud`:
  - keep `MkdirAll` as the single remote mkdir primitive (used by mirror mode).

## Design Decisions

### Decision: keep pandoc/xelatex as the only PDF generator

Bundling and syntax highlighting both fit naturally into pandoc:
- bundling: multiple inputs + ToC
- highlighting: pandoc syntax highlighting pipeline

Adding another renderer (HTML→PDF, chroma→PDF) adds complexity and inconsistent typography.

### Decision: “mirror dirs” is opt-in

Directory mirroring can create many remote folders; default behavior stays simple (flat upload).

### Decision: source upload implemented as “code → markdown → pdf”

This minimizes new surface area and reuses the established conversion pipeline.

## Alternatives Considered

### Alternative: implement bundling by merging PDFs post-hoc

Rejected (for now):
- PDF merging is harder to keep correct (ToC/bookmarks, page sizes, metadata).
- pandoc already knows how to build a ToC in a single PDF.

### Alternative: use Chroma (Go) to generate highlighted HTML and convert to PDF

Rejected (for now):
- introduces an HTML-to-PDF toolchain (wkhtmltopdf/weasyprint/etc.)
- diverges typography from the pandoc pipeline

## Implementation Plan

### Phase 1: Bundle mode (single PDF)
- Add `upload bundle` subcommand (preferred) or `--bundle` flag on `upload md`.
- Implement wrapper markdown generation with stable section headings.
- Wire pandoc flags for ToC.

### Phase 2: Mirror dirs
- Add `--preserve-dirs` to `upload md`.
- Extend file collection to retain (rootDir, relativePath) pairs.
- Compute per-file remote directory and call `rmcloud.MkdirAll` as needed.

### Phase 3: Source upload (syntax highlighting)
- Add `upload src` subcommand.
- Implement extension→language mapping and markdown wrapper generation.
- Add pandoc highlight options and theme flag.

### Phase 4: Docs + tests
- Add embedded help docs:
  - `pkg/doc/upload/03-remarquee-upload-bundle.md`
  - `pkg/doc/upload/04-remarquee-upload-src.md`
- Add unit tests:
  - wrapper markdown generation (bundle)
  - relative path mirroring logic
  - extension→language mapping

## Open Questions

- Do we want bundling for both `md` and `src` via a shared `upload bundle` command, or per-subcommand `--bundle` flags?
- How much control should users have over ToC depth, numbering, and title formatting?
- Which pandoc highlighting style should be the default (and do we want a stable “house theme”)?
- For mirroring: do we want ignore patterns (`.git/`, `node_modules/`, `ttmp/`, etc.)?

## References

 - Current implementation:
   - `remarquee/cmd/remarquee/cmds/upload/md.go`
   - `remarquee/pkg/mdpdf/*`
   - `remarquee/pkg/rmcloud/*`
 - Smoke test script (real upload + cloud ls verification):
   - `remarquee/ttmp/2025/12/14/RMQ-0003--port-remarkable-upload-py-to-go-remarquee-upload-docs/scripts/01-smoke-test-upload-md.sh`
