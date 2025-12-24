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

## Prerequisites

- `pdftoppm` (Poppler) must be installed and on PATH.
- `remarks` should be installed and on PATH, typically via Poetry:

```bash
export PATH="/home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarks/.venv/bin:$PATH"
command -v remarks
```

`pdfinfo` is optional (helpful if present).


