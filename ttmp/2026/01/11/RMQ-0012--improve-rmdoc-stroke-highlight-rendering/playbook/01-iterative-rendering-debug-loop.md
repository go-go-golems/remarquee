---
Title: Iterative rendering debug loop
Ticket: RMQ-0012
Status: active
Topics:
    - backend
    - rendering
    - remarkable
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Iterative debug loop to compare remarquee V6 renders against device screenshots and capture rendering gaps.
LastUpdated: 2026-01-11T21:22:29-05:00
WhatFor: Iteratively compare remarquee V6 render output against on-device rendering to debug stroke, highlight, and color fidelity.
WhenToUse: After changing stroke/highlight rendering or when investigating fidelity gaps between remarquee output and reMarkable display.
---

# Iterative rendering debug loop

## Purpose

Establish a repeatable loop for comparing remarquee's V6 renderer against the reMarkable device output, using device screenshots plus VLM assistance to highlight discrepancies in stroke thickness, highlight opacity, and color fidelity.

## Environment Assumptions

- `remarquee device serve` is running on the tablet and reachable (HTTP screenshot capture).
- You can SSH to the device (for restarting the capture server if needed).
- `remarquee` CLI is available locally.
- `rmapi` auth is configured for `remarquee cloud get` if pulling a device doc.
- `pdftoppm` (Poppler) is installed for PDF -> PNG conversion.
- `pinocchio` CLI is available for visual comparison queries.

## Commands

### 0) Pick the test document + page

Choose a V6 `.rmdoc` and the page number you want to compare (1-based). Prefer a document that contains:

- Thin strokes and bold strokes
- A dense highlight region
- Mixed pen colors (if available)

If you need to pull the document from the device cloud:

```bash
mkdir -p /tmp/rmq-0012
remarquee cloud get /Your/Remote/Doc --out-dir /tmp/rmq-0012
```

### 1) Render the rmdoc with remarquee

```bash
RMDOC=/tmp/rmq-0012/Your-Doc.rmdoc
OUT_DIR=/tmp/rmq-0012/run-$(date +%Y%m%d-%H%M%S)
mkdir -p "$OUT_DIR"

remarquee rmdoc render-v6 "$RMDOC" --out "$OUT_DIR/render-v6.pdf" --force
```

### 2) Convert the rendered PDF page to PNG

```bash
PAGE=1
pdftoppm -r 200 -f "$PAGE" -l "$PAGE" -png "$OUT_DIR/render-v6.pdf" "$OUT_DIR/render-page"
mv "$OUT_DIR/render-page-${PAGE}.png" "$OUT_DIR/render-page.png"
```

### 3) Ask the user to open the document on-device

Use a confirmation gate so we only capture screenshots when the right doc/page is visible.

```bash
plz-confirm confirm \
  --title "Open rMDOC for comparison" \
  --message "Please open the document on the tablet: /Your/Remote/Doc, page ${PAGE}. Keep it visible, then confirm." \
  --approve-text "Ready"
```

Optional: collect a doc name/page confirmation to reduce mistakes.

```bash
cat > "$OUT_DIR/device-open-schema.json" <<'EOF'
{
  "type": "object",
  "properties": {
    "doc_title": {"type": "string", "title": "Document title on device"},
    "page": {"type": "number", "title": "Page number currently visible"},
    "ready": {"type": "boolean", "title": "Page is visible and stable"}
  },
  "required": ["doc_title", "page", "ready"]
}
EOF

plz-confirm form --title "Confirm device state" --schema "$OUT_DIR/device-open-schema.json"
```

### 4) Fetch the device screenshot over HTTP

```bash
remarquee device screenshot \
  --url http://10.11.99.1:2718 \
  --username admin \
  --password password \
  --out "$OUT_DIR/device-screenshot.png"
```

### 5) Compare the images with pinocchio (VLM)

Ask a focused question about strokes/highlights.

```bash
pinocchio code professional \
  --images "$OUT_DIR/render-page.png,$OUT_DIR/device-screenshot.png" \
  "Compare stroke thickness, highlight opacity, and color density. List any mismatches or artifacts."
```

### 6) Ask the user for subjective notes (optional)

```bash
cat > "$OUT_DIR/user-feedback-schema.json" <<'EOF'
{
  "type": "object",
  "properties": {
    "match_quality": {"type": "string", "title": "Match quality", "enum": ["perfect", "good", "okay", "bad"]},
    "artifact_notes": {"type": "string", "title": "Artifacts or mismatches observed"}
  },
  "required": ["match_quality"]
}
EOF

plz-confirm form --title "Rendering comparison feedback" --schema "$OUT_DIR/user-feedback-schema.json"
```

### 7) Iterate after code changes

After adjusting stroke/highlight rendering, repeat steps 1-6 with the same document/page so comparisons are consistent.

## Exit Criteria

- You can consistently capture a device screenshot of the target doc/page.
- The VLM comparison and user feedback highlight the same set of rendering gaps.
- Follow-up renders show measurable improvements in stroke width, highlight opacity, or color fidelity.

## Notes

- Keep the capture server running on the device to avoid losing context between iterations.
- If the device is in sleep mode or locked, screenshots will not reflect the correct page.
- Use a fixed DPI (200 recommended) for PDF rasterization so comparisons are stable.
