#!/usr/bin/env bash
set -euo pipefail

# RMQ-0006 inverse protocol:
# - take a device-authored rmdoc (ellipses-test)
# - parse strokes
# - emit RMDoc-DSL YAML that replays those strokes
# - render both:
#   A) remarquee render-v6 on the original .rmdoc
#   B) DSL->PDF on the regenerated YAML
# - rasterize a few pages and ask the user if A and B match
#
# Usage:
#   bash .../21-ellipses-test-parse-regenerate-compare.sh

ROOT="/home/manuel/workspaces/2025-12-14/build-remarquee-tool"
REMARQUEE="$ROOT/remarquee"

IN_RMDOC="$REMARQUEE/rendering/rmq-0006-ellipse/ellipses-test.rmdoc"
OUTDIR="$REMARQUEE/rendering/rmq-0006-ellipse"

OUT_A_PDF="$OUTDIR/ellipses-test.remarquee-v6.pdf"
OUT_REGEN_YAML="$OUTDIR/ellipses-test.regen.yaml"
OUT_B_PDF="$OUTDIR/ellipses-test.regen.pdf"

if [[ ! -f "$IN_RMDOC" ]]; then
  echo "missing input rmdoc: $IN_RMDOC" >&2
  exit 1
fi
if [[ ! -d "$OUTDIR" ]]; then
  echo "missing output dir: $OUTDIR" >&2
  exit 1
fi
if ! command -v pdftoppm >/dev/null 2>&1; then
  echo "pdftoppm missing on PATH" >&2
  exit 1
fi

cd "$REMARQUEE"

echo "A) render original rmdoc via remarquee render-v6"
go run ./cmd/remarquee rmdoc render-v6 "$IN_RMDOC" --out "$OUT_A_PDF" --force

echo "B1) parse rmdoc strokes -> regenerate DSL YAML"
go run ./ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/21-rmdoc-v6-to-dsl-yaml/main.go \
  --in "$IN_RMDOC" \
  --out "$OUT_REGEN_YAML"

echo "B2) render regenerated DSL YAML -> PDF"
go run ./ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/19-rmdsl-render-to-pdf/main.go \
  --in "$OUT_REGEN_YAML" \
  --out "$OUT_B_PDF"

echo "Rasterize a few pages for review (first/middle/last)"

# Determine page count by counting .rm files in the rmdoc zip.
# This is the simplest robust signal for "how many pages exist" for V6 notebooks.
PAGECOUNT="$(unzip -l "$IN_RMDOC" | awk '$4 ~ /\.rm$/ {c++} END{print c+0}')"
if [[ "$PAGECOUNT" -le 0 ]]; then
  echo "could not determine pagecount from $IN_RMDOC (no .rm entries?)" >&2
  exit 1
fi

FIRST=1
MID=$(( (PAGECOUNT + 1) / 2 ))
LAST="$PAGECOUNT"

# Choose unique pages to show (1, mid, last).
PAGES=("$FIRST")
if [[ "$MID" != "$FIRST" ]]; then
  PAGES+=("$MID")
fi
if [[ "$LAST" != "$FIRST" && "$LAST" != "$MID" ]]; then
  PAGES+=("$LAST")
fi

pp() {
  local pdf="$1"
  local page="$2"
  local prefix="$3"
  pdftoppm -png -r 200 -f "$page" -l "$page" -singlefile "$pdf" "$OUTDIR/${prefix}-p${page}" >/dev/null 2>&1
}

IMG_ARGS=()
OPT_ARGS=()

add_img() {
  IMG_ARGS+=(--image "$1" --image-label "$2")
}
add_opt() {
  OPT_ARGS+=(--option "$1")
}

for p in "${PAGES[@]}"; do
  pp "$OUT_A_PDF" "$p" "A"
  pp "$OUT_B_PDF" "$p" "B"
  add_img "$OUTDIR/A-p${p}.png" "A page ${p}"
  add_img "$OUTDIR/B-p${p}.png" "B page ${p}"
  add_opt "Mismatch on page ${p}"
done

add_opt "Looks identical enough (regen matches)"
add_opt "Hard to tell / need higher DPI"

plz-confirm image \
  --title "RMQ-0006: ellipses-test inverse regen (A vs B)" \
  --message "A = remarquee render-v6 of ellipses-test.rmdoc. B = regenerated DSL (parsed strokes -> YAML -> PDF). Review the selected pages: do the ellipses match?" \
  "${IMG_ARGS[@]}" \
  --multi \
  "${OPT_ARGS[@]}" \
  --wait-timeout 300 \
  --output json


