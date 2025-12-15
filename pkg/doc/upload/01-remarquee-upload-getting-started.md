---
Title: remarquee upload — getting started
DocType: reference
Slug: remarquee-upload-getting-started
---

## Goal

Upload Markdown to a reMarkable device as a PDF.

`remarquee upload md` is the **write path** for documentation workflows:

- Markdown → PDF via `pandoc` + `xelatex` (DejaVu fonts)
- Upload via rmapi (in-process, no shelling out)
- Default destination: `/ai/YYYY/MM/DD/` (today)
- Safe by default: skips existing documents unless `--force`

## Quick start

### 1) Dry-run (recommended)

```bash
go run ./cmd/remarquee upload md --dry-run /abs/path/to/doc.md
```

### 2) Upload (no overwrite)

```bash
go run ./cmd/remarquee upload md /abs/path/to/doc.md
```

### 3) Upload a directory (recursive)

```bash
go run ./cmd/remarquee upload md --dry-run /abs/path/to/dir/
go run ./cmd/remarquee upload md /abs/path/to/dir/
```

## Common flags

- `--date YYYY/MM/DD` / `YYYY-MM-DD`: uploads to `/ai/YYYY/MM/DD/`
- `--remote-dir /Some/Folder`: override destination directory entirely
- `--pdf-only --output-dir ./out`: generate PDFs locally, do not upload
- `--force`: overwrite existing documents (**deletes existing document and annotations**)

## Dependencies

- `pandoc` must be installed and on PATH (or pass `--pandoc`)
- `xelatex` must be available (TeX Live)
- fonts: DejaVu Sans / DejaVu Sans Mono (defaults; override with flags)


