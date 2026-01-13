---
Title: remarquee upload src — reference
DocType: reference
Slug: remarquee-upload-src
---

## remarquee upload src

Render source files as PDFs (pandoc + xelatex) with syntax highlighting, then upload to reMarkable.

### Usage

```bash
remarquee upload src <path...> [flags]
```

`<path...>` may contain:
- files (any extension; best-effort highlighting)
- directories (recursively scanned for files)

Note: `*.md` files are ignored by `upload src` (use `upload md` / `upload bundle` instead).

### Destination

- Default: `/ai/YYYY/MM/DD/` (today)
- `--date`: choose `/ai/YYYY/MM/DD/` explicitly
- `--remote-dir`: override remote directory

### Highlighting and titles

- `--theme`: maps to pandoc `--highlight-style` (default: `tango`)
- `--listings`: use LaTeX listings (pandoc `--listings`)
- `--title-mode name|path`: choose whether the PDF title uses basename or relative path (default: `path`)

### Bundle mode

You can bundle multiple source files into a single PDF with a ToC:

- `--bundle`: enable bundle mode (one PDF total)
- `--name`: bundle document name
- `--toc-depth`: ToC depth (default 1)

### Safety / overwrite

- Default: if a document with the same name already exists in the destination dir, the upload is skipped.
- `--force`: delete the existing document and upload a new one (this also deletes annotations).

### Auth flags

- `--non-interactive`: never prompt; fails if tokens are missing
- `--reauth`: re-authenticate


