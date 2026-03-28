---
Title: remarquee upload bundle — reference
DocType: reference
Slug: remarquee-upload-bundle
---

## remarquee upload bundle

Bundle multiple markdown inputs into a single PDF (with a clickable ToC) and upload to reMarkable.

### Usage

```bash
remarquee upload bundle <path...> [flags]
```

`<path...>` may contain:
- markdown files (`*.md`)
- directories (recursively scanned for `*.md`)

### Ordering

- Explicit files preserve user-given order.
- Directory contents are ordered lexicographically by relative path (case-insensitive).

### Destination

- Default: `/ai/YYYY/MM/DD/` (today)
- `--date`: choose `/ai/YYYY/MM/DD/` explicitly
- `--remote-dir`: override remote directory

### Output naming

- Default:
  - single directory input: `<dirname>`
  - otherwise: `bundle`
- Override with `--name`

### Flags (high-signal)

- `--name`: output document name (also used for uploaded doc name)
- `--toc-depth`: ToC depth (pandoc `--toc-depth`, default 1)
- `--dry-run`: prints planned work; does not run pandoc or upload
- `--pdf-only --output-dir`: generate the bundle PDF locally only
- `--layout editor`: use a more annotation-friendly reading layout with wider margins and looser paragraph spacing

### Auth flags

- `--non-interactive`: never prompt; fails if tokens are missing
- `--reauth`: re-authenticate

