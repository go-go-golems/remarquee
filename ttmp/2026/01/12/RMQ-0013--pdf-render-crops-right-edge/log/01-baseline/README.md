# Case 01: Baseline (pre-fix)

Baseline render + device screenshot for page 65 before any bbox expansion or margin adjustments. This is the comparison from the rerun after confirming page 65 on-device.

## Commands

```bash
go run ./cmd/remarquee rmdoc render-v6-png /tmp/rmq-0012-journal/Journal.rmdoc --pages 65 --out-dir ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log --force
go run ./cmd/remarquee device screenshot --url http://10.11.99.1:2718 --username admin --password password --out ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/device-page-065.png
go run ./cmd/remarquee rmdoc vlm-validate \
  --image-a ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/Journal-v6-page-065.png \
  --image-b ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/device-page-065.png \
  --out-dir ttmp/2026/01/12/RMQ-0013--pdf-render-crops-right-edge/log/vlm-20260112-133346 \
  --prompt "Compare right-edge alignment. Is any content clipped on the right? Describe where."
```

## VLM output

```
In the images provided, the right-edge alignment appears to be inconsistent. 

1. **First Image**: There is content that is clipped on the right side. Specifically, the text near the bottom right, starting from "write the app to run code that does X," is cut off, making it unreadable.

2. **Second Image**: This image shows more complete text compared to the first, with no apparent clipping on the right edge. The text appears to be fully visible.

In summary, the first image has clipped content on the right, particularly towards the bottom, while the second image does not show any clipping.
```

## Files

- `Journal-v6-page-065.png`
- `Journal-v6.pdf`
- `device-page-065.png`
