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
OUTPDF="$OUTDIR/ellipse-sweep-pdf.pdf"
OUTREMARKS="$OUTDIR/ellipse-sweep-remarks.pdf"
REMOTE_DIR="/remarquee/rendering/rmq-0009-ellipse"
REMARKS_BIN="$ROOT/remarks/.venv/bin/remarks"

if [[ ! -f "$CASE_JS" ]]; then
  echo "missing case: $CASE_JS" >&2
  exit 1
fi

mkdir -p "$OUTDIR"

cd "$REMARQUEE"

if [[ ! -x "$REMARKS_BIN" ]]; then
  if command -v remarks >/dev/null 2>&1; then
    REMARKS_BIN="$(command -v remarks)"
  else
    echo "remarks not found at $REMARKS_BIN and not on PATH" >&2
    echo "hint: install remarks via poetry and ensure .venv exists" >&2
    exit 1
  fi
fi

echo "compile: JS case -> .rmdoc"
go run ./cmd/remarquee rmdsl compile "$CASE_JS" --out "$OUTDOC"

echo "render: .rmdoc -> PDF"
go run ./cmd/remarquee rmdoc render-v6 "$OUTDOC" --out "$OUTPDF" --force

TMPREMARKS="$(mktemp -d)"
"$REMARKS_BIN" "$OUTDOC" "$TMPREMARKS" --log_level ERROR
REMARKS_PDF="$(find "$TMPREMARKS" -type f -name "* _remarks.pdf" | head -n 1 || true)"
if [[ -z "$REMARKS_PDF" ]]; then
  echo "no remarks PDF found under $TMPREMARKS" >&2
  exit 1
fi
cp "$REMARKS_PDF" "$OUTREMARKS"
rm -rf "$TMPREMARKS"

echo "upload: .rmdoc -> $REMOTE_DIR"
go run ./cmd/remarquee cloud put --non-interactive --force "$OUTDOC" "$REMOTE_DIR"

echo "upload: render-v6 PDF -> $REMOTE_DIR"
go run ./cmd/remarquee cloud put --non-interactive --force "$OUTPDF" "$REMOTE_DIR"

echo "upload: remarks PDF -> $REMOTE_DIR"
go run ./cmd/remarquee cloud put --non-interactive --force "$OUTREMARKS" "$REMOTE_DIR"

plz-confirm confirm \
  --title "RMQ-0009: .rmdoc uploaded" \
  --message "Open 'ellipse-sweep.rmdoc' and the PDFs in $REMOTE_DIR. Confirm: is it an editable notebook (can you add strokes) and do the red dash page markers appear vs the PDFs?" \
  --approve-text "Yes, editable" \
  --reject-text "Not now" \
  --wait-timeout 300 \
  --output json
