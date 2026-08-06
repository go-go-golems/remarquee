# remarquee

A practical toolkit for **getting content into, out of, and around your reMarkable** — from scriptable cloud file operations, to Markdown/source-code PDF upload, to `.rmdoc` rendering, to on-device framebuffer capture.

**The core problem it solves:** reMarkable is wonderful as a reading and thinking device, but the workflows around it are often manual: click through cloud folders, export notebooks one by one, prepare PDFs by hand, or inspect opaque `.rmdoc` archives with little feedback. remarquee turns those chores into **repeatable CLI workflows** you can run, script, diff, and automate.

---

## What it does

remarquee is a unified Go CLI for reMarkable power users, automation scripts, and tool builders.

It currently focuses on five areas:

- **Cloud filesystem workflows** — browse, search, upload, download, move, delete, and inspect reMarkable cloud documents using rmapi-backed Sync15 primitives.
- **Markdown and source upload** — convert Markdown or source trees into reMarkable-friendly PDFs with `pandoc`/`xelatex`, then upload safely. Supports local image embedding and Mermaid diagram rendering.
- **`.rmdoc` inspection and rendering** — inspect local `.rmdoc` archives and render legacy/V6 notebook content to PDF or PNG.
- **On-device capture** — run a small server on the tablet to capture screenshots, raw framebuffer frames, pen events, and gesture summaries.
- **RMDoc-DSL fixtures** — author small YAML/JavaScript document fixtures for renderer testing and debugging.

remarquee builds on established tools and libraries — especially [`rmapi`](https://github.com/juruen/rmapi), Cobra, and Glazed — while providing a more cohesive command surface for document automation.

---

## Installation

```bash
# Homebrew (recommended for normal use)
brew tap go-go-golems/go-go-go
brew install remarquee

# Go install
go install github.com/go-go-golems/remarquee/cmd/remarquee@latest

# From source
git clone https://github.com/go-go-golems/remarquee.git
cd remarquee
go build ./cmd/remarquee
```

From a checkout, you can also run commands directly:

```bash
go run ./cmd/remarquee --help
go run ./cmd/remarquee status
```

---

## Quick start

### Browse your reMarkable cloud like a filesystem

```bash
# Authenticate/refresh the rmapi-backed local cloud tree
remarquee cloud refresh

# List folders and documents
remarquee cloud ls /
remarquee cloud ls /Books --compact --long --time

# Search by name or path
remarquee cloud search "meeting notes"

# Download a document as an .rmdoc archive
remarquee cloud get "/Notebooks/Project Plan" --output ./project-plan.rmdoc
```

### Upload Markdown to your tablet

```bash
# Always start with a dry run when automating
remarquee upload md --dry-run ./notes.md

# Convert Markdown to PDF and upload to /ai/YYYY/MM/DD/
remarquee upload md ./notes.md

# Upload a whole directory of Markdown files
remarquee upload md ./docs/

# Generate PDFs locally without uploading
remarquee upload md --pdf-only --output-dir ./out ./docs/

# Include local images and Mermaid diagrams
remarquee upload md ./design-doc.md

# Customize Mermaid rendering
remarquee upload md --mermaid-scale 3 --mermaid-width 1200 --mermaid-theme dark ./design.md

# Disable Mermaid or image resolution
remarquee upload md --mermaid=false --resolve-images=false ./notes.md
```

### Bundle docs into one reMarkable PDF

```bash
# Make a single PDF with a clickable table of contents
remarquee upload bundle ./design.md ./api.md ./runbook.md --name "Project Handbook"

# Use a layout with more annotation-friendly margins
remarquee upload bundle ./docs/ --layout editor --name "Docs Review"
```

### Turn source code into reviewable PDFs

```bash
# Render source files with syntax highlighting and upload them
remarquee upload src ./cmd ./pkg --remote-dir /CodeReviews/remarquee

# Bundle a source tree into one document
remarquee upload src ./pkg/rmdoc --bundle --name "rmdoc package review"
```

### Render local `.rmdoc` archives

```bash
# Legacy notebooks/documents
remarquee rmdoc render-legacy ./notebook.rmdoc --out ./notebook.pdf
remarquee rmdoc render-legacy ./notebook.rmdoc --pages 1,3-5 --out ./notebook-pages.pdf

# V6/cPages notebooks
remarquee rmdoc render-v6 ./notebook.rmdoc --out ./notebook.pdf
remarquee rmdoc render-v6 ./notebook.rmdoc --pages 2-4 --out ./notebook-pages.pdf

# V6 annotations only (handwriting on blank pages, no background PDF);
# pages without annotations are skipped unless explicitly selected via --pages
remarquee rmdoc render-v6 ./notebook.rmdoc --annotations-only --out ./notebook-annotations.pdf
remarquee rmdoc render-v6 ./notebook.rmdoc --annotations-only --pages 2-4 --out ./notebook-annotations-pages.pdf

# Debug V6 rendering as page PNGs
remarquee rmdoc render-v6-png ./notebook.rmdoc --out-dir ./pages
remarquee rmdoc render-v6-png ./notebook.rmdoc --pages 2-4 --annotations-only --out-dir ./annotation-pages
```

### Capture the tablet framebuffer

```bash
# On the device, typically over ssh root@remarkable
/home/root/remarquee device serve --bind :2718 --username admin --password password

# From your workstation
remarquee device info --url http://remarkable:2718 --username admin --password password
remarquee device screenshot --url http://remarkable:2718 --out screenshot.png
remarquee device events --url http://remarkable:2718
```

---

## Why this exists

The reMarkable ecosystem has several different “surfaces”:

```text
local Markdown/source files  →  PDF conversion  →  reMarkable cloud upload
                                      │
                                      ▼
                              tablet reading/review
                                      │
                                      ▼
remote cloud documents       →  .rmdoc archives  →  PDF/PNG rendering
                                      │
                                      ▼
on-device framebuffer/events →  screenshots, streams, automation signals
```

Without a tool like remarquee, these surfaces tend to stay separate. You might use one tool for cloud operations, another script for PDF conversion, manual export for notebooks, and ad-hoc SSH commands for device capture.

remarquee’s goal is to make those surfaces feel like one coherent workflow:

1. **Prepare** content from Markdown, source code, or generated documents.
2. **Upload** it into predictable cloud folders with safe overwrite behavior.
3. **Browse and script** the remote tree with structured output where useful.
4. **Download and render** `.rmdoc` archives for backup, review, or regression testing.
5. **Capture** what the tablet is actually showing when cloud exports are not enough.

The result is a CLI you can put in shell scripts, CI jobs, documentation workflows, and research notebooks — not just a one-off helper command.

---

## Command map

| Area | Commands | Use when you want to... |
| --- | --- | --- |
| Cloud | `remarquee cloud refresh`, `ls`, `find`, `search`, `stat`, `get`, `put`, `mkdir`, `mv`, `rm`, `account` | Treat the reMarkable cloud as a scriptable document tree. |
| Upload | `remarquee upload md`, `bundle`, `src`, `sync` | Convert local Markdown/source content to PDFs and upload it safely. |
| RMDoc | `remarquee rmdoc inspect`, `render-legacy`, `render-v6`, `render-v6-png`, `build-background`, `vlm-validate` | Inspect and render local `.rmdoc` archives. |
| Device | `remarquee device serve`, `info`, `screenshot`, `raw`, `stream`, `events`, `gestures` | Capture tablet frames and input signals from a device-side server. |
| RMDoc-DSL | `remarquee rmdsl compile` | Generate controlled `.rmdoc` fixtures from YAML/JavaScript descriptions. |
| OCR | `remarquee ocr` | Run image OCR through an LLM vision model. |

Most commands have detailed Cobra help:

```bash
remarquee --help
remarquee cloud --help
remarquee upload md --help
remarquee rmdoc render-v6 --help
```

remarquee also embeds richer Glazed help pages:

```bash
remarquee help
remarquee help remarquee-cloud-getting-started
remarquee help remarquee-upload-getting-started
remarquee help device-capture
```

---

## Documentation

The README is the landing page. The real documentation lives in [`pkg/doc/`](pkg/doc/) and is embedded into the binary help system.

Start here:

### Cloud workflows

- [Getting Started with `remarquee cloud`](pkg/doc/cloud/01-getting-started-remarquee-cloud.md)
- [`remarquee cloud` Reference](pkg/doc/cloud/02-remarquee-cloud-reference.md)
- [`remarquee cloud` Usage Examples](pkg/doc/cloud/03-remarquee-cloud-usage-examples.md)
- [rmapi Filetree and Sync15 Model](pkg/doc/topics/filetree-and-sync15-model.md)

### Upload workflows

- [`remarquee upload` Getting Started](pkg/doc/upload/01-remarquee-upload-getting-started.md)
- [`remarquee upload md` Reference](pkg/doc/upload/02-remarquee-upload-reference.md)
- [`remarquee upload bundle` Reference](pkg/doc/upload/03-remarquee-upload-bundle.md)
- [`remarquee upload src` Reference](pkg/doc/upload/04-remarquee-upload-src.md)

### Rendering, fixtures, and device capture

- [Device Capture: Framebuffer API + CLI](pkg/doc/topics/device-capture.md)
- [RMDoc-DSL Getting Started](pkg/doc/topics/rmdsl-getting-started.md)

### Development docs

- [Adding a Glazed CLI Command to remarquee](pkg/doc/tutorials/01-adding-a-glazed-command-to-remarquee.md)
- [Build a React + RTK Query app with Vite and Dagger go:generate](pkg/doc/topics/how-to-create-a-web-app-with-react-rtk-vite-dagger-gen.md)

---

## Safety notes

remarquee tries to make destructive workflows explicit:

- Upload commands skip existing remote documents by default.
- `--force` can overwrite remote documents, but doing so may delete existing annotations.
- `upload md --dry-run` and `upload bundle --dry-run` are recommended before bulk uploads.
- Cloud delete/move operations should be tested on disposable folders before scripting against important documents.
- Device capture requires running code on the tablet and may need root access; use basic auth and trusted networks.

---

## Dependencies

For cloud operations, remarquee uses rmapi and the same general device-token flow. You authenticate through the official reMarkable connect page; remarquee/rmapi store local tokens for later use.

For upload/rendering workflows, install the external tools required by the command you use:

- `pandoc` for Markdown/source-to-PDF conversion
- `xelatex` / TeX Live for PDF generation
- DejaVu fonts, or pass custom font flags
- Docker/container runtime if you use repository build steps that invoke Dagger

---

## Project status

remarquee is actively evolving. Some areas are production-style daily workflow tools (cloud browsing, Markdown upload); others are lower-level research/development surfaces for renderer work, V6 `.rmdoc` support, and on-device capture.

If you are building automation around it, prefer:

- `--dry-run` before writes
- explicit `--remote-dir` / `--date` destinations
- structured output where commands support Glazed output
- checked-in scripts for recurring workflows

---

## License

See [LICENSE](LICENSE).
