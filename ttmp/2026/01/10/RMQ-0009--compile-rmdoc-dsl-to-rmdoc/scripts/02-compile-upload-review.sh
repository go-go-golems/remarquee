#!/usr/bin/env bash
set -euo pipefail

# RMQ-0009: compile a DSL case to .rmdoc, upload it, and ask for device validation.
#
# Usage:
#   bash .../02-compile-upload-review.sh

ROOT="/home/manuel/workspaces/2025-12-14/build-remarquee-tool"
REMARQUEE="$ROOT/remarquee"

CASE_JS="$REMARQUEE/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/cases/03-ellipse-sweep.js"
OUTDIR="$REMARQUEE/rendering/rmq-0009-ellipse"
OUTDOC="$OUTDIR/ellipse-sweep.rmdoc"
REMOTE_DIR="/remarquee/rendering/rmq-0009-ellipse"

if [[ ! -f "$CASE_JS" ]]; then
  echo "missing case: $CASE_JS" >&2
  exit 1
fi

mkdir -p "$OUTDIR"

cd "$REMARQUEE"

echo "compile: JS case -> .rmdoc"
go run ./cmd/remarquee rmdsl compile "$CASE_JS" --out "$OUTDOC"

echo "upload: .rmdoc -> $REMOTE_DIR"
go run ./cmd/remarquee cloud put --non-interactive --force "$OUTDOC" "$REMOTE_DIR"

plz-confirm confirm \
  --title "RMQ-0009: .rmdoc uploaded" \
  --message "Open 'ellipse-sweep.rmdoc' on the tablet in $REMOTE_DIR. Confirm: is it an editable notebook (can you add strokes) and do the red dash page markers appear?" \
  --approve-text "Yes, editable" \
  --reject-text "Not now" \
  --wait-timeout 300 \
  --output json
