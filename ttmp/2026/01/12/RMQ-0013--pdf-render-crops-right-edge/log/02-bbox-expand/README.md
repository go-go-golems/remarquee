# Case 02: BBox-based blank-page sizing

Switch blank-background rendering to use bbox-derived page size (instead of fixed 1404x1872) for notebooks. This attempt aimed to stop clipping by expanding the canvas to include stroke extents.

## Commands

```bash
go run ./cmd/remarquee rmdoc render-v6-png /tmp/rmq-0012-journal/Journal.rmdoc --pages 65 --out-dir ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/02-bbox-expand --force
go run ./cmd/remarquee rmdoc vlm-validate \
  --image-a ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/02-bbox-expand/Journal-v6-page-065.png \
  --image-b ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/02-bbox-expand/device-page-065.png \
  --out-dir ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/02-bbox-expand/vlm \
  --prompt "Compare right-edge alignment. Is any content clipped on the right? Describe where."
```

## VLM output

```
In the provided images, the right-edge alignment appears to differ between the two. 

In the first image, some text near the right edge is clipped, particularly the last few words in the lines. For example, phrases like "so i can" and "the code to do X" are cut off, indicating that the text extends beyond the visible area.

In the second image, the right edge seems to be more contained, with no visible clipping of content. The words appear fully within the margins.

Overall, the first image has content clipping on the right edge, while the second image does not.
```

## Files

- `Journal-v6-page-065.png`
- `Journal-v6.pdf`
- `device-page-065.png`
