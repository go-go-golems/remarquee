---
Title: Stroke, highlight, and color rendering analysis
Ticket: RMQ-0012
Status: active
Topics:
    - backend
    - rendering
    - remarkable
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarquee/pkg/rmdoc/bbox.go
      Note: BBox math used for merge alignment
    - Path: remarquee/pkg/rmdoc/pen_color.go
      Note: Color mapping and RGBA map
    - Path: remarquee/pkg/rmdoc/render/v6_merge_background.go
      Note: Stroke style + highlight annotation rendering
    - Path: remarquee/pkg/rmdoc/render/v6_strokes_pdf.go
      Note: Strokes-only renderer and rmv6 scale constants
    - Path: remarquee/pkg/rmdoc/rmv6_glyph_decode.go
      Note: Glyph range decoding for smart highlights
    - Path: remarquee/pkg/rmdoc/rmv6_line_decode.go
      Note: Stroke decoding (tool
    - Path: remarquee/pkg/rmdoc/strokes.go
      Note: Stroke/StrokePoint structure
ExternalSources: []
Summary: Analysis of current V6 stroke/highlight/color rendering path and likely fidelity gaps vs device output.
LastUpdated: 2026-01-11T21:31:16-05:00
WhatFor: Understand where stroke thickness, highlight opacity, and color mapping are simplified in the current V6 renderer.
WhenToUse: Before adjusting renderer math or investigating differences between device screenshots and remarquee renders.
---


# Stroke, highlight, and color rendering analysis

This document reviews how V6 annotations are decoded and rendered today, with an emphasis on stroke width/thickness, highlight behavior, and color mapping. It is grounded in the current renderer implementation and highlights where the pipeline intentionally simplifies or drops data that exists in the `.rmdoc` or `.rm` payloads.

The goal is to locate the exact code paths and parameters that influence the observed differences vs the reMarkable device (stroke thickness, highlight opacity, and color fidelity), so we can iterate with targeted fixes.

## Code map (files and symbols)

- `remarquee/pkg/rmdoc/rmv6_line_decode.go`: `DecodeRMV6Line`, `rmv6PointFromStream`
- `remarquee/pkg/rmdoc/strokes.go`: `Stroke`, `StrokePoint`
- `remarquee/pkg/rmdoc/pen_color.go`: `PenColor`, `HardcodedColorMap`, `PenColorToRGB`, `PenColorToRGBForStroke`
- `remarquee/pkg/rmdoc/rmv6_glyph_decode.go`: `DecodeRMV6GlyphRange`, `RMV6GlyphRange`
- `remarquee/pkg/rmdoc/render/v6_merge_background.go`: `MergeRMDocV6OntoBackgroundPDFWithInfo`, `strokeStyleForTool`, `buildOverlayOps`, `applySmartHighlightsScaled`
- `remarquee/pkg/rmdoc/render/v6_strokes_pdf.go`: `RenderRMV6StrokesToPDF`, `rmv6Scale`
- `remarquee/pkg/rmdoc/bbox.go`: `BBoxForStroke`, `BBoxForStrokes`

## Rendering pipeline overview

At a high level, the renderer:

- Opens the `.rmdoc` container and builds a UI page plan (`OpenFile` + `.content` parsing).
- Reads the per-page `.rm` annotation file (V6 scene tree).
- Decodes line items into normalized strokes and glyph ranges.
- Merges a background PDF with overlay content (strokes + typed text), then adds smart highlight annotations.

ASCII pipeline diagram:

```
.rmdoc (zip)
  ├─ .content          -> UI page plan (page IDs)
  ├─ <page>.rm         -> RMV6 scene tree (lines, glyphs, text)
  └─ payload.pdf       -> background PDF (optional)
        |
        v
Parse scene tree
  ├─ DecodeRMV6Line -> []Stroke
  ├─ DecodeRMV6GlyphRange -> []GlyphRange
  └─ BuildRMV6TextDocument -> typed text
        |
        v
MergeRMDocV6OntoBackgroundPDFWithInfo
  ├─ buildOverlayOps (strokes + typed text)
  └─ applySmartHighlightsScaled (glyph ranges)
```

## Stroke decoding (input data)

Strokes are decoded in `DecodeRMV6Line`:

- Tool, color ID, thickness scale, and starting length are read from the line header.
- Points are decoded with per-point fields (speed, direction, width, pressure).
- Optional trailing RGBA marker can override the color for highlights/shaders.

Key snippet (simplified):

```go
tool, _ := vr.readUint32(1)
color, _ := vr.readUint32(2)
thicknessScale, _ := vr.readFloat64(3)
points := readPoints(...)
// Optional RGBA marker (0x84 0x01) -> HardcodedColorMap override
return &Stroke{Tool: tool, Color: color, ThicknessScale: thicknessScale, Points: points}
```

Important: the `StrokePoint` structure captures `Width` and `Pressure`, but the renderer does not yet use them.

## Stroke rendering (current behavior)

Two renderers exist:

- `RenderRMV6StrokesToPDF` (strokes-only test renderer)
- `MergeRMDocV6OntoBackgroundPDFWithInfo` (full overlay path)

The production path is the merge renderer. It applies coarse tool-based styles in `strokeStyleForTool` and then renders polylines in `buildOverlayOps` / `buildOverlayOpsScreen`.

Current stroke style mapping:

```go
func strokeStyleForTool(tool, thicknessScale, color) strokeStyle {
    switch tool {
    case 5, 18: // highlighter
        return {WidthScreenUnits: 25, Opacity: 0.3, LineCap: 2}
    case 23: // shader
        opacity := alphaFromColorID(color) // from HardcodedColorMap alpha
        return {WidthScreenUnits: 12, Opacity: opacity, LineCap: 1}
    case 4, 17: // fineliner
        return {WidthScreenUnits: thicknessScale * 1.8, Opacity: 1.0, LineCap: 1}
    case 7, 13: // mechanical pencil
        return {WidthScreenUnits: thicknessScale * thicknessScale, Opacity: 0.7, LineCap: 1}
    default:
        return {WidthScreenUnits: thicknessScale, Opacity: 1.0, LineCap: 1}
    }
}
```

Observations:

- Stroke widths are derived from a single `thicknessScale` value per stroke, not per-point width/pressure.
- The highlighter uses a fixed width (25 screen units) regardless of input pressure or width.
- The shader tool uses color-derived alpha, but other tools are fully opaque.
- Line caps are fixed (round or square) per tool; there is no join style, taper, or brush texture.

PDF coordinate conversion uses the `rmv6Scale` (72/226) and inverts Y.

## Highlight rendering (two mechanisms)

Highlights can appear in two distinct ways:

1) **Highlight strokes (tool 5/18)**:
   - Rendered as thick, semi-transparent strokes (width 25, opacity 0.3).
   - These are line items, so they follow the stroke renderer path.

2) **Smart highlights (glyph ranges)**:
   - Decoded from `SceneGlyphItemBlock` via `DecodeRMV6GlyphRange`.
   - Rendered as PDF highlight annotations (`/Subtype /Highlight`) in `applySmartHighlightsScaled`.

Key snippet (smart highlights):

```go
hl := pdf.NewPdfAnnotationHighlight()
hl.C = core.MakeArrayFromFloats([]float64{r, g, b})
hl.CA = core.MakeFloat(0.3)
hl.Rect = ... // union rect
hl.QuadPoints = quadArr
```

Observations:

- Highlight annotation alpha is hard-coded to 0.3 for all glyph ranges.
- Color for glyph ranges comes from `PenColorToRGB`, which defaults to yellow if unknown.
- Smart highlight positioning uses `highlightsXTranslation` computed by merge math; this can drift if bounding boxes are off.

## Color rendering (current mapping)

Color uses a small map of reMarkable `PenColor` enums to RGB values.

Relevant functions:

- `PenColorToRGB` for highlight annotations and general color.
- `PenColorToRGBForStroke` for strokes (unknown -> black).
- `HardcodedColorMap` for RGBA marker decoding.

Potential color pitfalls:

- `PenColorToRGB` defaults unknown colors to yellow highlight, which can tint highlights incorrectly if the color ID is not mapped.
- Shader colors are interpreted via RGBA alpha values, but the renderer does not combine tool opacity with RGBA alpha for non-shader tools.
- Highlight colors can appear as `PenColorHighlight` or via RGBA marker; only the latter gives the specific hue (yellow/blue/pink/etc).

## Bounding boxes and alignment side effects

Bounding boxes are computed in `BBoxForStroke` using point positions only:

```go
// NOTE: At this stage we only consider point coordinates; brush width/thickness
// is not accounted for.
```

This affects:

- Overlay placement relative to background pages (bbox determines shifts).
- Smart highlight X translation (`highlightsXTranslation`) for glyph ranges.
- Potential clipping if thick strokes extend beyond the bbox.

## Practical mismatches vs device output (likely)

- **Stroke thickness**: per-stroke `ThicknessScale` is a coarse substitute for per-point width/pressure.
- **Pressure/taper**: per-point `Width` and `Pressure` are ignored, so tapering and brush dynamics are lost.
- **Highlighter opacity**: hard-coded 0.3 may not match device alpha curves or tool-specific blending.
- **Shader opacity**: uses RGBA alpha but not adjusted by tool behavior or point pressure.
- **Color hues**: `PenColorToRGB` is a static palette and may not match device tone/opacity for different tools.
- **BBox-driven alignment**: missing stroke widths in bbox math can shift or clip overlays, impacting highlights and text alignment.

## Pseudocode summary (current renderer)

```
for page in doc.pages:
  bg = background page (or blank screen)
  tree = parse scene (.rm)
  strokes = decode line items
  glyphs = decode glyph ranges

  bbox = union(point-only bboxes of strokes) or default full screen
  (xSvg, ySvg, shifts) = merge math using bbox + background size

  for stroke in strokes:
     style = strokeStyleForTool(tool, thicknessScale, color)
     widthPt = scale(style.WidthScreenUnits)
     alpha = style.Opacity
     rgb = PenColorToRGBForStroke(color)
     draw polyline (no per-point width)

  add smart highlight annotations for glyph ranges (CA=0.3)
```

## Notes on what is not yet modeled

- Per-point width, pressure, or speed curves for brush dynamics.
- Brush-specific textures (pencil grain, calligraphy, marker bleed).
- Join styles and tapering (caps are set, but joins are not modeled).
- Tool-specific opacity curves beyond the coarse constants.
- Color space conversion to match device rendering (current mapping is a static palette).

## Suggested investigation checkpoints

- Validate whether `thicknessScale` correlates with `StrokePoint.Width` on real data.
- Compare `PenColorToRGB` values against device screenshot samples for highlight hues.
- Measure how bbox shifts influence `highlightsXTranslation` on pages with thick strokes.
- Trace whether highlighter tool strokes should be rendered as highlights (blend mode) instead of plain strokes.
