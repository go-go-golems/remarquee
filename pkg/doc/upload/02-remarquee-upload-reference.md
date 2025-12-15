---
Title: remarquee upload — reference
DocType: reference
Slug: remarquee-upload-reference
---

## remarquee upload md

Convert markdown files to PDF (pandoc + xelatex) and upload to reMarkable.

### Usage

```bash
remarquee upload md <path...> [flags]
```

`<path...>` may contain:
- markdown files (`*.md`)
- directories (recursively scanned for `*.md`)

### Destination

- Default: `/ai/YYYY/MM/DD/` (today)
- `--date`: choose `/ai/YYYY/MM/DD/` explicitly
- `--remote-dir`: override remote directory

### Safety / overwrite

- Default: if a document with the same name already exists in the destination dir, the upload is skipped.
- `--force`: delete the existing document and upload a new one (this also deletes annotations).

### Conversion behavior

Before running pandoc:
- YAML frontmatter is stripped (docmgr-style `--- ... ---`)
- list spacing is normalized so pandoc recognizes lists reliably

### Flags (high-signal)

- `--dry-run`: prints planned conversions/uploads; does not run pandoc or upload
- `--pdf-only --output-dir`: generate PDFs locally only
- `--pandoc`, `--pdf-engine`: control conversion tooling
- `--mainfont`, `--monofont`, `--geometry`, `--latex-header-file`: typography customization

### Auth flags

- `--non-interactive`: never prompt; fails if tokens are missing
- `--reauth`: re-authenticate


