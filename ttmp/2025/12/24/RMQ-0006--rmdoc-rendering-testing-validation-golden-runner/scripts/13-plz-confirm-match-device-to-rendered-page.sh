#!/usr/bin/env bash
set -euo pipefail

# Human-in-the-loop: determine which rendered page matches the device screenshot.
#
# Usage:
#   bash .../13-plz-confirm-match-device-to-rendered-page.sh

ROOT="/home/manuel/workspaces/2025-12-14/build-remarquee-tool"
REMARQUEE="$ROOT/remarquee"
TICKET="$REMARQUEE/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner"
REF="$TICKET/reference"
SCRIPTS="$TICKET/scripts"

DEVICE_PNG="$REF/test-rmdoc-page1-remarkable-device.png"
if [[ ! -f "$DEVICE_PNG" ]]; then
  echo "missing device png: $DEVICE_PNG" >&2
  exit 1
fi

if ! command -v plz-confirm >/dev/null 2>&1; then
  echo "plz-confirm missing on PATH" >&2
  exit 1
fi

GEN_LOG="$(mktemp -t rmq0006-render-pages-XXXXXX.log)"
bash "$SCRIPTS/12-render-test-rmdoc-pages1-2-pngs.sh" | tee "$GEN_LOG"
A1="$(grep '^A1_PNG=' "$GEN_LOG" | tail -n1 | cut -d= -f2-)"
A2="$(grep '^A2_PNG=' "$GEN_LOG" | tail -n1 | cut -d= -f2-)"

plz-confirm image \
  --title "RMQ-0006: Which rendered page matches the device photo?" \
  --message "We need to confirm we are comparing the same page. Which remarquee-rendered page (1 or 2) matches the device screenshot content best?" \
  --mode select \
  --wait-timeout 600 \
  --image "$A1" --image-label "A1: remarquee page 1" \
  --image "$A2" --image-label "A2: remarquee page 2" \
  --image "$DEVICE_PNG" --image-label "B: device photo (page 1)" \
  --option "A1 matches B" \
  --option "A2 matches B" \
  --option "Neither matches B (page mapping bug)" \
  --output json


