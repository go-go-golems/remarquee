---
Title: 'Debug strategy: right-edge crop in PDF render'
Ticket: RMQ-0013
Status: active
Topics:
    - rendering
    - remarkable
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarquee/pkg/rmdoc/bbox.go
      Note: Stroke bbox width
    - Path: remarquee/pkg/rmdoc/render/background.go
      Note: Background PDF sizing
    - Path: remarquee/pkg/rmdoc/render/v6_merge_background.go
      Note: Merge math + overlay ops
    - Path: remarquee/pkg/rmdoc/render/v6_strokes_pdf.go
      Note: Scale constants
ExternalSources: []
Summary: Debug strategy for right-edge cropping in rendered PDFs, with VLM and human validation loops.
LastUpdated: 2026-01-12T13:19:54-05:00
WhatFor: Plan a focused debug loop to locate and fix right-edge cropping in rendered PDFs.
WhenToUse: When diagnosing PDF output alignment/cropping issues and validating fixes with human + VLM checks.
---


# Debug strategy: right-edge crop in PDF render

This strategy focuses on narrowing the right-edge cropping issue to a specific stage in the V6 merge pipeline, then validating fixes using automated VLM checks plus human review via `plz-confirm`.

## Rendering overview + crop injection points

High-level flow (rendered PNGs and PDFs share this pipeline):

```
.rmdoc -> OpenFile -> Page plan + payload PDF
  -> ReadRMFileFromArchive (per page)
  -> ParseRMV6SceneTree
  -> ExtractRMV6StrokesWithAnchors + ExtractRMV6GlyphRangesWithAnchors
  -> BuildBackgroundPDFForPages
  -> MergeRMDocV6OntoBackgroundPDFWithInfoForPages
      - buildMergedPage (background transform + overlay ops)
      - buildOverlayOps (stroke path mapping)
      - applySmartHighlightsScaled (highlight rects)
  -> PDF bytes
  -> pdftoppm rasterize (PNG)
```

Pseudocode (cropping-sensitive operations):

```
for each page:
  bbox = union(stroke points)           // ignores stroke width
  wSvg = scale(bbox width)
  wBg  = background page width
  width = max(wSvg, wBg)
  xBg/xSvg = shifts based on bbox
  backgroundTransform(rotation, xBg, yBg)
  overlayOps = map stroke points to PDF coords
  compose (background + overlay)
```

Likely crop points:

- `BBoxForStroke` in `remarquee/pkg/rmdoc/bbox.go` (bbox width too small).
- `buildOverlayOps` in `remarquee/pkg/rmdoc/render/v6_merge_background.go` (x-shift math).
- `backgroundTransform` in `remarquee/pkg/rmdoc/render/v6_merge_background.go` (translation for rotated pages).
- `inferPayloadPageSize` in `remarquee/pkg/rmdoc/render/background.go` (background page width mismatch).

## Where to look first (likely sources)

The crop is most likely introduced in the layout math that merges annotation overlays onto the background PDF. Start with:

- `remarquee/pkg/rmdoc/render/v6_merge_background.go`
  - `buildMergedPage` (background form placement and content stream composition)
  - `buildOverlayOps` (stroke coordinate transform from rmv6 -> PDF)
  - `backgroundTransform` (rotation + translation of the background)
  - `displayDims` + `pageBoxDims` (page box sizing and rotation)
  - `highlightsXTranslation` (used to offset highlight rectangles)
- `remarquee/pkg/rmdoc/render/background.go`
  - `BuildBackgroundPDFForPages` and `inferPayloadPageSize`
  - Check for size mismatches between payload PDF pages and computed overlay canvas.
- `remarquee/pkg/rmdoc/bbox.go`
  - `BBoxForStroke` (bbox uses points only; ignores stroke width)
- `remarquee/pkg/rmdoc/render/v6_strokes_pdf.go`
  - `rmv6Scale` and screen width/height constants (screen size mismatch could compress or shift).

## Hypotheses to test

- **BBox shrinkage**: stroke bbox ignores width, reducing overlay canvas width and clipping the right edge.
- **Rotation transform**: `backgroundTransform` may translate background content incorrectly for rotated PDFs.
- **Payload page size mismatch**: `inferPayloadPageSize` may choose a smaller page size than the background.
- **X shift miscalculation**: `xShift` (based on bbox min/max) may offset overlays leftward, cropping the right edge.
- **Scale mismatch**: `rmv6Scale` / `cairoSVGScale` may be applied incorrectly for notebook vs PDF-backed docs.

## Debug loop (step-by-step)

1) **Baseline (VLM-first)**
   - Render PNGs from the `.rmdoc` and run a VLM prompt focused on right-edge clipping.
   - This avoids PDF vs PNG comparisons (same source) and gets a quick signal on cropping.

2) **Log dimensions**
   - Add temporary logging (or debug prints) in:
     - `buildMergedPage`: log `width`, `height`, `xBg`, `yBg`, `xSvg`, `ySvg`, `wSvg`, `hSvg`, `wBg`, `hBg`.
     - `pageBoxDims`: log `MediaBox/CropBox` dimensions and rotation.
   - Confirm if the overlay canvas is smaller than the background page width.

3) **Clamp or expand bbox**
   - Temporarily expand `BBoxForStroke` or the default bbox to include a fixed padding.
   - Re-render and check if the right-edge crop disappears.

4) **Isolate background**
   - Render background-only and overlay-only to see which layer is cropped:
     - Use `buildOverlayOnlyPage` with a full screen bbox.
     - Compare against `BuildBackgroundPDF` output.

5) **Fix and validate**
   - Apply the minimal fix (bbox padding, width computation adjustment, or transform correction).
   - Re-run the same pages; once the VLM indicates improvement, ask the human to open the page
     (via plz-confirm) before taking the device screenshot.

## VLM-assisted validation (pinocchio)

`remarquee rmdoc vlm-validate` now accepts PNG, PDF, or `.rmdoc` inputs. VLM prompts should focus on clipping in the rendered PNG first, then compare against a device screenshot once a human confirms the page is visible.

Example commands:

```bash
# Rendered PNG only (VLM-first)
remarquee rmdoc vlm-validate \
  --image-a /tmp/rmq-0012-journal/png/Journal-v6-page-076.png \
  --prompt "Inspect the right edge. Is any handwriting clipped?"

# Rendered PNG vs device screenshot (after plz-confirm gating)
remarquee rmdoc vlm-validate \
  --image-a /tmp/rmq-0012-journal/png/Journal-v6-page-076.png \
  --image-b /tmp/rmq-0013-crop/device-page-076.png \
  --prompt "Compare right-edge alignment. Is any content clipped on the right?"
```

## Human-in-the-loop validation (plz-confirm)

Use `plz-confirm` to ensure the device screenshot is captured only after the correct page is visible, and to collect subjective feedback.

1) Ask the user to open the target page:

```bash
plz-confirm confirm \
  --title "Open page for crop check" \
  --message "Please open /Journal page 76 on the device and keep it visible." \
  --approve-text "Ready"
```

2) Optional form to confirm doc/page:

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

3) Present the comparison to the user:

```bash
plz-confirm image \
  --title "Right-edge crop review" \
  --message "Compare the render (left) vs device screenshot (right). Is the right edge clipped?" \
  --image /tmp/rmq-0012-journal/png/Journal-v6-page-076.png \
  --image /tmp/device-screenshot-page-076.png \
  --mode confirm
```

4) Collect textual notes:

```bash
cat > /tmp/rmq-0013-crop-notes.json <<'EOF'
{
  "type": "object",
  "properties": {
    "crop_severity": {"type": "string", "title": "Crop severity", "enum": ["none", "minor", "moderate", "severe"]},
    "notes": {"type": "string", "title": "Notes"}
  },
  "required": ["crop_severity"]
}
EOF

plz-confirm form --title "Right-edge crop notes" --schema /tmp/rmq-0013-crop-notes.json
```

## Concrete artifacts to collect

- Rendered PNG for the page (e.g., `/tmp/rmq-0012-journal/png/Journal-v6-page-076.png`).
- Device screenshot for that page (captured after plz-confirm gating).
- VLM output text describing the crop.
- User notes from `plz-confirm`.

## Stop conditions

- The right-edge crop no longer appears in side-by-side comparison (human + VLM).
- Overlay and background layers align without visible drift after merging.
