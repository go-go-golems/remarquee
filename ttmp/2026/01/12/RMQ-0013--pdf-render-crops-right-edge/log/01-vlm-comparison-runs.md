---
Title: VLM comparison runs
Ticket: RMQ-0013
Status: active
Topics:
    - rendering
    - remarkable
DocType: log
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarquee/ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/Journal-v6-page-065.png
      Note: Rendered page 65 PNG (rerun)
    - Path: remarquee/ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/Journal-v6.pdf
      Note: Subset PDF for page 65
    - Path: remarquee/ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/device-page-065.png
      Note: Device screenshot page 65
    - Path: remarquee/ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/render-page-065.png
      Note: Rendered page 65 PNG
    - Path: remarquee/ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/vlm-A-page-076.png
      Note: Rendered page 76 PNG
ExternalSources: []
Summary: VLM comparison runs and outputs for right-edge crop validation.
LastUpdated: 2026-01-12T13:34:17-05:00
WhatFor: Track VLM prompts, commands, and outputs alongside the images used.
WhenToUse: Update after each OCR/VLM comparison run.
---



# VLM comparison runs

## 2026-01-12 - Page 65 render vs device screenshot (rerun, page confirmed)

Command:

```bash
go run ./cmd/remarquee rmdoc render-v6-png /tmp/rmq-0012-journal/Journal.rmdoc --pages 65 --out-dir ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log --force
go run ./cmd/remarquee device screenshot --url http://10.11.99.1:2718 --username admin --password password --out ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/device-page-065.png
go run ./cmd/remarquee rmdoc vlm-validate \
  --image-a ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/Journal-v6-page-065.png \
  --image-b ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/device-page-065.png \
  --out-dir ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/vlm-20260112-133346 \
  --prompt "Compare right-edge alignment. Is any content clipped on the right? Describe where."
```

Output:

```
In the images provided, the right-edge alignment appears to be inconsistent. 

1. **First Image**: There is content that is clipped on the right side. Specifically, the text near the bottom right, starting from "write the app to run code that does X," is cut off, making it unreadable.

2. **Second Image**: This image shows more complete text compared to the first, with no apparent clipping on the right edge. The text appears to be fully visible.

In summary, the first image has clipped content on the right, particularly towards the bottom, while the second image does not show any clipping.
```

Stored images:

- `remarquee/ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/Journal-v6-page-065.png`
- `remarquee/ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/device-page-065.png`

Stored output directory:

- `remarquee/ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/vlm-20260112-133346/`

## 2026-01-12 - Page 65 render vs device screenshot

Command:

```bash
go run ./cmd/remarquee rmdoc vlm-validate \
  --image-a /tmp/rmq-0013-crop/Journal-v6-page-065.png \
  --image-b /tmp/rmq-0013-crop/device-page-065.png \
  --prompt "Compare right-edge alignment. Is any content clipped on the right? Describe where."
```

Output:

```
In the first image, it appears that some content may be clipped on the right edge, particularly in the lower section where the text ends abruptly. Specifically, the last line contains incomplete thoughts, suggesting that additional text may be missing. 

In the second image, the right edge seems to be aligned better, but there are instances where text extends close to the margin, making it difficult to determine if any content is clipped.

Overall, the first image has more noticeable clipping issues, particularly in the lower portion.
```

Stored images:

- `remarquee/ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/render-page-065.png`
- `remarquee/ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/device-page-065.png`

## 2026-01-12 - Page 76 render (VLM-first)

Command:

```bash
go run ./cmd/remarquee rmdoc vlm-validate \
  --rmdoc-a /tmp/rmq-0012-journal/Journal.rmdoc \
  --pages 76 \
  --prompt "Inspect the right edge. Is any handwriting clipped? Describe where."
```

Output:

```
Yes, some handwriting is clipped along the right edge. Specifically, the text at the very end of the lines appears to be cut off. Phrases like "generative capacity" and "what programming literature is" are partially visible but not fully legible, indicating that the right edge of the page is affecting the readability of the handwriting.
```

Stored images:

- `remarquee/ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/vlm-A-page-076.png`
