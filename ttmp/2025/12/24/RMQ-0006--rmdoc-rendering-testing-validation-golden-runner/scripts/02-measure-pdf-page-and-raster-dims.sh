#!/usr/bin/env bash
set -euo pipefail

# Measures page size metadata (pdfinfo if available) and raster dimensions (pdftoppm PNG)
# for page 1 of two PDFs. This is useful for diagnosing "maxDiffRatio=1.0" golden failures,
# which often means the rendered images differ in pixel dimensions.
#
# Usage:
#   bash .../02-measure-pdf-page-and-raster-dims.sh /path/to/A.pdf /path/to/B.pdf

if [[ $# -ne 2 ]]; then
  echo "usage: $0 A.pdf B.pdf" >&2
  exit 2
fi

A="$1"
B="$2"

for f in "$A" "$B"; do
  if [[ ! -f "$f" ]]; then
    echo "missing: $f" >&2
    exit 1
  fi
done

echo "A=$A"
echo "B=$B"
echo

if command -v pdfinfo >/dev/null 2>&1; then
  echo "== pdfinfo A =="
  pdfinfo "$A" | sed -n '1,25p' || true
  echo
  echo "== pdfinfo B =="
  pdfinfo "$B" | sed -n '1,25p' || true
  echo
else
  echo "pdfinfo not installed; skipping pdfinfo output"
  echo
fi

if ! command -v pdftoppm >/dev/null 2>&1; then
  echo "pdftoppm not installed; cannot measure raster dimensions" >&2
  exit 1
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

render_one() {
  local in="$1"
  local prefix="$2"
  local base="$tmp/$prefix"
  pdftoppm -png -r 200 -f 1 -l 1 -singlefile "$in" "$base" >/dev/null 2>&1
  echo "$base.png"
}

png_dims() {
  # Standard library only: read PNG IHDR chunk to extract width/height.
  python3 - <<'PY' "$1"
import struct, sys
fn = sys.argv[1]
with open(fn, "rb") as f:
    sig = f.read(8)
    if sig != b"\x89PNG\r\n\x1a\n":
        raise SystemExit(f"not a PNG: {fn}")
    length = struct.unpack(">I", f.read(4))[0]
    ctype = f.read(4)
    if ctype != b"IHDR":
        raise SystemExit(f"unexpected first chunk {ctype!r} in {fn}")
    data = f.read(length)
    w, h = struct.unpack(">II", data[:8])
    print(f"{w}x{h}")
PY
}

A_PNG="$(render_one "$A" "A-page-001")"
B_PNG="$(render_one "$B" "B-page-001")"

echo "== pdftoppm raster dims (page 1 @ 200 DPI) =="
echo "A_PNG=$A_PNG"
echo "A_DIMS=$(png_dims "$A_PNG")"
echo "B_PNG=$B_PNG"
echo "B_DIMS=$(png_dims "$B_PNG")"


