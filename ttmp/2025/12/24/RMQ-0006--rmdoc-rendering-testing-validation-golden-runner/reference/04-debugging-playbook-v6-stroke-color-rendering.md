---
Title: 'Debugging playbook: V6 stroke color rendering'
Ticket: RMQ-0006
Status: active
Topics:
    - go
    - remarkable
    - testing
    - validation
    - rmdoc
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../rmc/src/rmc/exporters/writing_tools.py
      Note: Reference palette (RM_PALETTE) used by remarks/rmc
    - Path: cmd/remarquee/cmds/rmdoc/vlm_validate.go
      Note: VLM helper (pdftoppm + pinocchio)
    - Path: pkg/rmdoc/pen_color.go
      Note: PenColor enum + RGBA/pen palette mapping
    - Path: pkg/rmdoc/render/v6_merge_background.go
      Note: Overlay renderer applies per-stroke RG
    - Path: pkg/rmdoc/rmv6_line_decode.go
      Note: Decodes stroke color_id and optional trailing RGBA marker
    - Path: pkg/rmdoc/rmv6_stroke_color_test.go
      Note: Parsing assertion for stroke colors on Test.rmdoc
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-24T16:34:04.69650344-05:00
WhatFor: ""
WhenToUse: ""
---


# Debugging playbook: V6 stroke color rendering

Stroke color bugs are one of the easiest ways for remarquee to “look wrong” even when all geometry is correct. They’re also surprisingly easy to mis-diagnose because V6 stores multiple “kinds” of color information depending on tool type (pen vs highlighter/shader).

This playbook gives you a fast, repeatable loop to answer:

- “Are we *parsing* the right colors?”
- “Are we *rendering* those colors into the PDF content stream?”
- “Do the results match `remarks` on the same fixture?”

## Goal

Help a developer debug and fix V6 stroke color rendering quickly, without getting stuck in brittle setup issues.

## Context

### The two color paths in V6

For most “pen” strokes, V6 line items contain a `color_id` that matches `rmscene.scene_items.PenColor` (the same enum used by `rmc`).

For highlighter/shader strokes, the `color_id` may be a generic value (often `PenColorHighlight`), and the *actual* highlighter color is stored in an **optional trailing RGBA marker** at the end of the line payload.

In our Go reimplementation this matters because:

- If we only use the header `color_id`, all highlighter strokes can collapse to a single color.
- If we consume the RGBA marker but throw it away, we can’t recover the real highlight color later.

### Where the implementation lives

- **Decode V6 line item (stroke + color)**:
  - `pkg/rmdoc/rmv6_line_decode.go`
  - The `color_id` comes from tag 2
  - Optional trailing marker uses prefix `0x84 0x01` then bytes `(b,g,r,a)`
  - Mapping of RGBA → `PenColor` is in `pkg/rmdoc/pen_color.go` (`HardcodedColorMap`)
- **Stroke primitive**:
  - `pkg/rmdoc/strokes.go` (`Stroke.Color uint32`)
- **Extract strokes from scene tree**:
  - `pkg/rmdoc/rmv6_strokes_extract.go`
- **Render strokes into PDF**:
  - `pkg/rmdoc/render/v6_merge_background.go` (`buildOverlayOps`)
  - `pkg/rmdoc/render/v6_strokes_pdf.go` (strokes-only PDF renderer)

## Quick Reference

This section is copy/paste-first; use it to build a tight loop.

### 0) Preconditions checklist

- `pdftoppm` is available (for VLM image generation):

```bash
command -v pdftoppm && pdftoppm -v | head -n 2
```

- `pinocchio` is available:

```bash
command -v pinocchio
```

- `remarks` is installed (Poetry):

```bash
export PATH="/home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarks/.venv/bin:$PATH"
command -v remarks && remarks --version
```

### 1) Verify parsing sees multiple stroke colors (fast unit test)

```bash
cd /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee && \
go test ./pkg/rmdoc -run TestParseRMV6SceneTree_StrokeColorsPresent_TestRmdoc -count=1 -v
```

What you’re looking for:

- `seen stroke colors: [...]` includes **non-zero** values
- it includes at least one of the **concrete highlight colors** (`PenColorHighlightYellow..PenColorHighlightGray`)

If this fails, start in `pkg/rmdoc/rmv6_line_decode.go` (RGBA marker parsing).

### 2) Render + VLM validate stroke colors (A vs B)

This is the fastest “does it look right?” loop without opening a PDF viewer.

```bash
cd /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee && \
export PATH="/home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarks/.venv/bin:$PATH" && \
TMPDIR="$(mktemp -d)" && echo "TMPDIR=$TMPDIR" && \
go run ./cmd/remarquee rmdoc render-v6 ./cmd/remarquee-ui/testdata/Test.rmdoc --out "$TMPDIR/A.pdf" --force && \
remarks ./cmd/remarquee-ui/testdata/Test.rmdoc "$TMPDIR/remarks-out" --log_level ERROR && \
REF_PDF="$(find "$TMPDIR/remarks-out" -type f -name "* _remarks.pdf" | head -n 1)" && echo "REF_PDF=$REF_PDF" && \
go run ./cmd/remarquee rmdoc vlm-validate \
  --rasterizer poppler --dpi 200 \
  --pdf-b "$REF_PDF" \
  --pages 1 \
  --out-dir "$TMPDIR/vlm" \
  --prompt "You will receive TWO images, in order: (1) A-page-001 from remarquee, (2) B-page-001 from remarks. Compare ONLY stroke colors. List each visible stroke color in A and whether it matches B. Ignore typed text and highlight annotations." \
  "$TMPDIR/A.pdf"
```

### 3) If it’s still wrong: inspect the PDF content stream

The merged PDF pages include a marker string in the overlay content:

- `rmv6-overlay`

You can extract a page’s content stream and look for repeated `RG` operations (stroke color commands). If everything is `0 0 0 RG`, we’re still rendering black.

(Exact extraction varies; easiest is to use UniDoc tooling or a PDF inspection tool.)

## Usage Examples

### Example: “All strokes are black”

1. Run the parsing test:
   - If only `0` appears, the issue is in decoding/extraction (we’re losing `color_id`).
2. If parsing looks good, the issue is rendering:
   - check `buildOverlayOps` uses per-stroke `RG` (not a single global `RG`)
   - ensure we map `Stroke.Color` → `PenColor` → RGB using the same palette as `rmc`

### Example: “Highlighter strokes are the wrong color”

This often means we used `PenColorHighlight` (9) directly, instead of decoding the trailing RGBA marker.

Checklist:
- Does `DecodeRMV6Line` parse the marker and map `(b,g,r,a)` to `HardcodedColorMap`?
- Does the parsing test show concrete highlight color ids (14..19)?
- Does the renderer map those ids to the pastel highlight RGB values?

## Related

- Golden system guide (usage + implementation): `reference/03-golden-testing-validation-system-how-to-use-how-it-works.md`
- VLM helper: `cmd/remarquee/cmds/rmdoc/vlm_validate.go`
- UniDoc rasterizer failure bug report: `bug-report-vlm-validate-unidoc-render-type-check-error.md`
- Reference palette (Python rmc): `rmc/src/rmc/exporters/writing_tools.py` (`RM_PALETTE`)
