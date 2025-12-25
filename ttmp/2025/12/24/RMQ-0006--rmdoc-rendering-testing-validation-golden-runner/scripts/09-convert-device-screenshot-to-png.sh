#!/usr/bin/env bash
set -euo pipefail

# Convert the committed device screenshot (JPG) into a PNG for tools that are picky about formats.
#
# Usage:
#   bash .../09-convert-device-screenshot-to-png.sh
#
# Outputs:
#   Prints DEVICE_PNG path.

ROOT="/home/manuel/workspaces/2025-12-14/build-remarquee-tool"
REMARQUEE="$ROOT/remarquee"
REF="$REMARQUEE/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/reference"
IN="$REF/test-rmdoc-page1-remarkable-device.jpg"
OUT="$REF/test-rmdoc-page1-remarkable-device.png"

if [[ ! -f "$IN" ]]; then
  echo "missing input: $IN" >&2
  exit 1
fi

if ! command -v convert >/dev/null 2>&1; then
  echo "ImageMagick 'convert' missing on PATH" >&2
  exit 1
fi

# Normalize orientation just in case, and write a PNG.
convert "$IN" -auto-orient "$OUT"
echo "DEVICE_PNG=$OUT"


