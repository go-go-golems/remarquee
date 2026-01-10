#!/usr/bin/env bash
set -euo pipefail

# RMQ-0006: generate an ellipse sweep fixture (JS → DSL), render to PDF, upload to device, then ask user questions.
#
# This script follows the “device vs export” protocol:
# - generate a controlled fixture family (multiple pages, known Y positions)
# - render locally (PNG optional, PDF for tablet viewing)
# - upload the PDF to a stable folder on device
# - ask structured questions via plz-confirm
#
# Usage:
#   bash .../20-ellipse-sweep-generate-upload-review.sh

ROOT="/home/manuel/workspaces/2025-12-14/build-remarquee-tool"
REMARQUEE="$ROOT/remarquee"

CASE_JS="$REMARQUEE/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/cases/03-ellipse-sweep.js"
OUTDIR="$REMARQUEE/rendering/rmq-0006-ellipse"
OUTPDF="$OUTDIR/ellipse-sweep.pdf"

if [[ ! -f "$CASE_JS" ]]; then
  echo "missing case: $CASE_JS" >&2
  exit 1
fi
if [[ ! -d "$OUTDIR" ]]; then
  echo "missing output dir (create it once): $OUTDIR" >&2
  exit 1
fi

cd "$REMARQUEE"

echo "render: JS case -> PDF"
go run ./ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/19-rmdsl-render-to-pdf/main.go \
  --in "$CASE_JS" \
  --out "$OUTPDF"

echo "upload: PDF -> /remarquee/rendering/rmq-0006-ellipse"
go run ./cmd/remarquee cloud put --non-interactive --force "$OUTPDF" /remarquee/rendering/rmq-0006-ellipse

echo "ask: confirm device review"
plz-confirm confirm \
  --title "RMQ-0006: Ellipse sweep uploaded" \
  --message "Open 'ellipse-sweep.pdf' on the tablet in /remarquee/rendering/rmq-0006-ellipse. Each page has N red dashes near the top-left (page marker) and one ellipse at a different Y position. Ready to answer a quick question about which pages look wrong?" \
  --approve-text "Ready" \
  --reject-text "Not now" \
  --wait-timeout 300 \
  --output json

plz-confirm select \
  --title "RMQ-0006: Which ellipse pages look WRONG on device?" \
  --option "All pages look correct (ellipse moves top->bottom as expected)" \
  --option "Page 1 (1 red dash): ellipse Y seems wrong" \
  --option "Page 2 (2 red dashes): ellipse Y seems wrong" \
  --option "Page 3 (3 red dashes): ellipse Y seems wrong" \
  --option "Page 4 (4 red dashes): ellipse Y seems wrong" \
  --option "Page 5 (5 red dashes): ellipse Y seems wrong" \
  --option "Hard to tell / need a better marker" \
  --multi \
  --wait-timeout 300 \
  --output json


