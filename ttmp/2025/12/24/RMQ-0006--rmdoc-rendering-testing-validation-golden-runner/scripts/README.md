---
Title: RMQ-0006 scripts (golden testing + validation)
Ticket: RMQ-0006
Status: active
DocType: reference
Intent: long-term
Owners: []
Summary: "Reusable scripts for diagnosing and running the RMQ-0006 golden test + validation pipeline."
---

# RMQ-0006 scripts (golden testing + validation)

This folder contains small, reusable scripts we use while building RMQ-0006’s golden testing pipeline. The goal is to **avoid brittle one-off commands** in chat history and make it easy for the next developer to reproduce investigations.

## Scripts

### `01-generate-a-vs-b-pdfs-test-rmdoc.sh`

Generates:

- **A**: remarquee `render-v6` output for `Test.rmdoc`
- **B**: `remarks` reference output for the same `Test.rmdoc`

It prints the resulting file paths so you can feed them into other scripts.

### `02-measure-pdf-page-and-raster-dims.sh`

Given two PDF paths (A and B), prints:

- `pdfinfo` (if installed): page size metadata
- `pdftoppm` rasterization output size for page 1 (PNG width/height)

This is useful when golden tests show `maxDiffRatio=1.0`, which often means the rasterized images differ in dimensions.

### `03-pdf-box-dump.go`

`go run` helper to dump MediaBox/CropBox sizes and rotation for each page (UniDoc-based, no external deps).

### `04-debug-golden-size-mismatch-test-rmdoc.sh`

One-shot “first suspicion” debug script that:

- generates A/B PDFs for `Test.rmdoc`
- dumps PDF boxes (MediaBox/CropBox/rotation)
- measures raster PNG dimensions for page 1 (via `pdftoppm`)

### `05-generate-a-vs-b-pdfs-cpage-pdf.sh`

Same as `01-...` but for the PDF-backed fixture `cpage-pdf.rmdoc`.

### `06-debug-golden-size-mismatch-cpage-pdf.sh`

Same as `04-...` but for `cpage-pdf.rmdoc`.

### `07-render-test-rmdoc-page1-png.sh`

Renders `Test.rmdoc` page 1 to a PNG via `remarquee rmdoc render-v6` + `pdftoppm`. Useful as a stable input for VLM comparisons.

### `08-vlm-compare-test-page1-vs-device-screenshot.sh`

Runs a VLM comparison between:

- `Test.rmdoc` page 1 rendered by remarquee (PNG)
- a real-device screenshot stored in `reference/test-rmdoc-page1-remarkable-device.jpg`

### `09-convert-device-screenshot-to-png.sh`

Converts `reference/test-rmdoc-page1-remarkable-device.jpg` → PNG (some vision backends/tools are picky about JPEG).

### `10-plz-confirm-review-test-page1-vs-device.sh`

Uses `plz-confirm image` to put a **human** in the loop for image review. It shows:

- A: remarquee-rendered `Test.rmdoc` page 1 (PNG)
- B: reMarkable device screenshot (PNG)

and asks a structured multiple-choice question about highlighter alignment.

## Prerequisites

- `pdftoppm` (Poppler) must be installed and on PATH.
- `remarks` should be installed and on PATH, typically via Poetry:

```bash
export PATH="/home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarks/.venv/bin:$PATH"
command -v remarks
```

`pdfinfo` is optional (helpful if present).


