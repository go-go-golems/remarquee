#!/usr/bin/env bash
set -euo pipefail

# Compare remarquee-rendered Test.rmdoc page 1 vs a real-device screenshot using pinocchio VLM.
#
# This is intended to answer: are highlighter strokes (and other strokes) positioned correctly
# relative to the dotted template and other shapes, compared to what the reMarkable device shows?
#
# Usage:
#   bash .../08-vlm-compare-test-page1-vs-device-screenshot.sh

ROOT="/home/manuel/workspaces/2025-12-14/build-remarquee-tool"
REMARQUEE="$ROOT/remarquee"
TICKET_REF="$REMARQUEE/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/reference"
SCREENSHOT="$TICKET_REF/test-rmdoc-page1-remarkable-device.jpg"
SCREENSHOT_PNG="$TICKET_REF/test-rmdoc-page1-remarkable-device.png"
SCRIPTS="$REMARQUEE/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts"

if [[ ! -f "$SCREENSHOT" ]]; then
  echo "missing screenshot: $SCREENSHOT" >&2
  exit 1
fi

# Some pinocchio backends are picky about JPEGs. Convert to PNG if needed.
if [[ ! -f "$SCREENSHOT_PNG" ]]; then
  bash "$SCRIPTS/09-convert-device-screenshot-to-png.sh" >/dev/null
fi
if [[ ! -f "$SCREENSHOT_PNG" ]]; then
  echo "failed to create screenshot png: $SCREENSHOT_PNG" >&2
  exit 1
fi

if ! command -v pinocchio >/dev/null 2>&1; then
  echo "pinocchio missing on PATH" >&2
  exit 1
fi

OUT_LOG="$(mktemp -t rmq0006-vlm-device-XXXXXX.log)"

GEN_LOG="$(mktemp -t rmq0006-render-page1-XXXXXX.log)"
bash "$SCRIPTS/07-render-test-rmdoc-page1-png.sh" | tee "$GEN_LOG"
A_PNG="$(grep '^A_PNG=' "$GEN_LOG" | tail -n1 | cut -d= -f2-)"

echo "A_PNG=$A_PNG"
echo "DEVICE_JPG=$SCREENSHOT"
echo "DEVICE_PNG=$SCREENSHOT_PNG"
echo "LOG=$OUT_LOG"
echo

VLM_PROMPT='You will receive TWO images, in order:
(1) remarquee-rendered Test.rmdoc page 1 (PNG)
(2) a photo of the actual reMarkable device showing the same page (JPG)

Task: Focus on highlighter strokes (thick translucent strokes) and their alignment.
- Identify each highlighter stroke region and describe its location relative to the dot grid and nearby shapes.
- Compare whether the highlighter stroke placement in (1) matches (2) (ignore color tone/lighting differences from the photo).
- If there is any systematic offset (e.g. shifted left/right/up/down) or scale mismatch, call it out with direction and rough magnitude.

Also quickly note whether non-highlighter colored strokes (red/green/pink) appear in the same spots.'

pinocchio code professional --images "$A_PNG,$SCREENSHOT_PNG" "$VLM_PROMPT" | tee "$OUT_LOG"


