# Case 03: Stroke-width padding (min 1 unit)

Add padding based on max stroke width (half-width, minimum 1 screen unit) to the bbox used for blank-page rendering. The goal was to prevent stroke thickness from touching the right edge.

## Commands

```bash
go run ./cmd/remarquee rmdoc render-v6-png /tmp/rmq-0012-journal/Journal.rmdoc --pages 65 --out-dir ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/03-stroke-pad --force
go run ./cmd/remarquee rmdoc vlm-validate \
  --image-a ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/03-stroke-pad/Journal-v6-page-065.png \
  --image-b ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/03-stroke-pad/device-page-065.png \
  --out-dir ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/03-stroke-pad/vlm \
  --prompt "Compare right-edge alignment. Is any content clipped on the right? Describe where."
```

## VLM output

```
In the first image, there is noticeable right-edge clipping of content. Specifically:

1. The line starting with "DSL + JS goja support..." appears to be truncated. The text ends abruptly, indicating it may not be fully visible.
2. The same applies to the line beginning with "i kinda stroke my brain, huh..." where it also seems to be cut off.

In contrast, the second image has better right-edge alignment, with no clipping observed. All content is fully visible and properly aligned.
```

## Files

- `Journal-v6-page-065.png`
- `Journal-v6.pdf`
- `device-page-065.png`
