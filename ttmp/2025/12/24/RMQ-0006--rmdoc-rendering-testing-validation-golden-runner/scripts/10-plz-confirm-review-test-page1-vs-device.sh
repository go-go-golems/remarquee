#!/usr/bin/env bash
set -euo pipefail

# Human-in-the-loop review using plz-confirm's image widget.
#
# Shows:
# - remarquee-rendered Test.rmdoc page 1 PNG
# - reMarkable device screenshot PNG
#
# Asks a structured question about highlighter alignment.
#
# Usage:
#   bash .../10-plz-confirm-review-test-page1-vs-device.sh

ROOT="/home/manuel/workspaces/2025-12-14/build-remarquee-tool"
REMARQUEE="$ROOT/remarquee"
TICKET="$REMARQUEE/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner"
REF="$TICKET/reference"
SCRIPTS="$TICKET/scripts"

DEVICE_PNG="$REF/test-rmdoc-page1-remarkable-device.png"

if ! command -v plz-confirm >/dev/null 2>&1; then
  echo "plz-confirm missing on PATH" >&2
  exit 1
fi

if [[ ! -f "$DEVICE_PNG" ]]; then
  echo "missing device png: $DEVICE_PNG" >&2
  exit 1
fi

GEN_LOG="$(mktemp -t rmq0006-render-page1-XXXXXX.log)"
bash "$SCRIPTS/07-render-test-rmdoc-page1-png.sh" | tee "$GEN_LOG"
A_PNG="$(grep '^A_PNG=' "$GEN_LOG" | tail -n1 | cut -d= -f2-)"

plz-confirm image \
  --title "RMQ-0006: Test.rmdoc page 1 vs device (highlighter alignment)" \
  --message "Compare the thick translucent highlighter strokes. Are they aligned between the rendered PNG and the device screenshot? Pick the best description." \
  --image "$A_PNG" --image-label "A: remarquee render (page 1)" \
  --image "$DEVICE_PNG" --image-label "B: reMarkable device photo (page 1)" \
  --option "Aligned / no obvious offset" \
  --option "Slight offset (<= 2mm), acceptable" \
  --option "Noticeable offset (> 2mm), needs fix" \
  --option "Can't tell (needs better screenshot / crop)" \
  --output json


