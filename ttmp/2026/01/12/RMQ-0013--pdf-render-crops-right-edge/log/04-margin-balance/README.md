# Case 04: Margin balancing for notebook pages

Balance left/right margins when the stroke bbox overflows the default screen width by extending the bbox on the tighter side. This keeps the left margin and right margin symmetric for notebook pages.

## Commands

```bash
go run ./cmd/remarquee rmdoc render-v6-png /tmp/rmq-0012-journal/Journal.rmdoc --pages 65 --out-dir ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/04-margin-balance --force
go run ./cmd/remarquee rmdoc vlm-validate \
  --image-a ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/04-margin-balance/Journal-v6-page-065.png \
  --image-b ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/04-margin-balance/device-page-065.png \
  --out-dir ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/04-margin-balance/vlm \
  --prompt "Compare right-edge alignment. Is any content clipped on the right? Describe where."
```

## VLM output

```
In the provided images, the right-edge alignment appears consistent, but there is some content clipping on the right side of the first image. Specifically:

1. The text "how does the skill creation skill work?" is partially clipped, as the last few letters may not be fully visible.
2. The text "i kinda stroke my brain hul" also shows clipping, particularly the final letters in the word "hul."

In the second image, the text is more completely visible, indicating that the content is not clipped. The alignment and layout differences might be due to the image capture or display method.
```

## Files

- `Journal-v6-page-065.png`
- `Journal-v6.pdf`
- `device-page-065.png`
