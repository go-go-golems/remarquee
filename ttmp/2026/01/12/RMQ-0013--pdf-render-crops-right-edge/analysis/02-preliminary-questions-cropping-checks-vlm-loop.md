---
Title: 'Preliminary questions: cropping checks + VLM loop'
Ticket: RMQ-0013
Status: active
Topics:
    - rendering
    - remarkable
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarquee/cmd/remarquee/cmds/device/screenshot.go
      Note: Device screenshot capture
    - Path: remarquee/cmd/remarquee/cmds/rmdoc/render_v6_png.go
      Note: PNG render source
    - Path: remarquee/cmd/remarquee/cmds/rmdoc/vlm_validate.go
      Note: VLM validation pipeline
ExternalSources: []
Summary: Preliminary questions and VLM/human validation loop for detecting right-edge cropping.
LastUpdated: 2026-01-12T13:19:54-05:00
WhatFor: Establish a minimal VLM-first loop that flags cropping issues before involving device screenshots.
WhenToUse: At the start of the RMQ-0013 investigation to collect quick evidence of cropping in render outputs.
---


# Preliminary questions: cropping checks + VLM loop

This document captures the first-pass questions and an incremental validation flow. The intent is to start with the rendered PNG output alone (VLM-only), then bring in device screenshots only after a human confirms the page is visible on the tablet.

## Key questions to answer early

- Can a VLM reliably detect right-edge clipping in the rendered PNG (e.g., Journal page 76)?
- Does the cropping look consistent across multiple pages and templates, or does it vary with content?
- Is the cropping visible in the merged PDF itself, or only after rasterization?
- Does the issue appear in notebook pages only, or also in PDF-backed documents?

## VLM-first loop (no device screenshots yet)

Goal: get a quick signal from the VLM using rendered PNGs alone.

1) Render PNGs from the `.rmdoc`:

```bash
remarquee rmdoc render-v6-png /tmp/rmq-0012-journal/Journal.rmdoc \
  --pages 76 \
  --out-dir /tmp/rmq-0013-crop \
  --force
```

2) Ask the VLM to look for cropping:

```bash
pinocchio code professional \
  --non-interactive \
  --output text \
  --images /tmp/rmq-0013-crop/Journal-v6-page-076.png \
  "Inspect the right edge. Is any handwriting or stroke clipped? If so, describe where."
```

3) Repeat across a few pages (e.g., 13, 14, 20, 21, 39, 76, 77) to detect patterning.

## Human-in-the-loop gating before screenshots

Before capturing a device screenshot, ensure the correct document and page are visible. Use `plz-confirm` to explicitly gate the screenshot step.

```bash
plz-confirm confirm \
  --title "Open page for screenshot" \
  --message "Please open /Journal page 76 on the tablet and keep it visible." \
  --approve-text "Ready"
```

Optional form for explicit confirmation:

```bash
cat > /tmp/rmq-0013-page-schema.json <<'EOF'
{
  "type": "object",
  "properties": {
    "doc_title": {"type": "string", "title": "Document title on device"},
    "page": {"type": "number", "title": "Page number visible"},
    "ready": {"type": "boolean", "title": "Page is visible and stable"}
  },
  "required": ["doc_title", "page", "ready"]
}
EOF

plz-confirm form --title "Confirm device state" --schema /tmp/rmq-0013-page-schema.json
```

Then capture the screenshot:

```bash
remarquee device screenshot \
  --url http://10.11.99.1:2718 \
  --username admin \
  --password password \
  --out /tmp/rmq-0013-crop/device-page-076.png
```

## How vlm-validate works today (detailed)

`remarquee rmdoc vlm-validate` renders PDF pages to PNG using Poppler’s `pdftoppm`, then sends the PNGs to `pinocchio`. It does not render strokes directly; it relies on the PDF that you pass in.

Key flow:

- Parse `--pages` into 1-based page numbers.
- Render each PDF page via `pdftoppm -png -r <dpi> -f <page> -l <page>`.
- Collect PNG paths and call:

```bash
pinocchio code professional --non-interactive --output text --images <comma-list> "<prompt>"
```

Relevant files:

- `remarquee/cmd/remarquee/cmds/rmdoc/vlm_validate.go`
  - `renderPDFPagesToPNGsWithPoppler`
  - `parsePages1Based`
  - `Run` (pinocchio invocation)

## Does vlm-validate use the new PNG render path?

Yes. `vlm-validate` now accepts PNG inputs (`--image-a/--image-b`) and `.rmdoc` inputs (`--rmdoc-a/--rmdoc-b`). For `.rmdoc`, it uses the same V6 merge pipeline as `render-v6-png` and then rasterizes the subset pages via Poppler.

## Unified flow (implemented)

Inputs are now mutually exclusive per side:

- `--image-a` / `--image-b` (use PNGs directly)
- `--pdf-a` / `--pdf-b` (rasterize PDF pages)
- `--rmdoc-a` / `--rmdoc-b` (render V6 pages to PDF and rasterize)

The `--pages` flag is used for PDF or `.rmdoc` inputs; it is ignored for direct images.

Pseudocode:

```
if imageA:
  imagesA = [imageA]
else if pdfA:
  imagesA = renderPDFPagesToPNGs(pdfA, pages)
else if rmdocA:
  imagesA = renderRMDocPagesToPNGs(rmdocA, pages)
run pinocchio --images <imagesA + imagesB> --prompt <prompt>
```

## Suggested first VLM prompts

- “Inspect the right edge for clipped handwriting or missing strokes. Describe any cutoff.”
- “Compare left/right margins for symmetry. Does the right margin appear narrower?”
- “Is any vertical stroke truncated at the rightmost edge?”

## Artifacts to keep

- Rendered PNGs per page.
- VLM output text for each page.
- Device screenshots (after plz-confirm gating).
- User confirmation records from plz-confirm.
