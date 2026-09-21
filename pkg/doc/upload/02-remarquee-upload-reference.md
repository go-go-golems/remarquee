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
- Pandoc's `yaml_metadata_block` extension is disabled so ordinary `---` thematic breaks are not parsed as YAML
- list spacing is normalized so pandoc recognizes lists reliably
- local images are resolved and copied; inline `<svg>` blocks and HTML `<img src=*.svg>` tags are converted
- Mermaid code blocks are rendered to diagrams when `mmdc` is available

For a full list of what renders (and what does not), see
`remarquee help markdown-format-support`.

### Flags (high-signal)

- `--dry-run`: prints planned conversions/uploads; does not run pandoc or upload
- `--pdf-only --output-dir`: generate PDFs locally only
- `--preserve-dirs`: when uploading directories, recreate the local relative directory structure remotely
- `--name`: custom output document name; only valid when exactly one markdown file is selected
- `--layout editor`: use a more annotation-friendly reading layout with wider margins and looser paragraph spacing
- `--pandoc`, `--pdf-engine`: control conversion tooling
- `--mainfont`, `--monofont`, `--geometry`, `--latex-header-file`: typography customization
- `--resolve-images` (default true): resolve and embed local image references

### Images, SVG and Mermaid

- `--svg` (default true): render inline `<svg>` and HTML `<img src=*.svg>` as vector images
- `--svg-converter PATH`: use a specific SVG converter (default: `rsvg-convert`, then `inkscape`)
- `--svg-default-width W`: width for extracted inline SVGs, e.g. `70%` or `12cm`
- `--mermaid` (default true): render ` ```mermaid ` blocks (requires `mmdc`)
- `--mermaid-scale`, `--mermaid-theme`, `--mermaid-bg`, `--mermaid-width`, `--mermaid-pdf-width`, `--mmdc-path`: Mermaid rendering controls

### Auth flags

- `--non-interactive`: never prompt; fails if tokens are missing
- `--reauth`: re-authenticate


## remarquee upload bundle

See `pkg/doc/upload/03-remarquee-upload-bundle.md`.

## remarquee upload src

See `pkg/doc/upload/04-remarquee-upload-src.md`.
