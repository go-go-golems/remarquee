#!/usr/bin/env bash
set -euo pipefail

# RMQ-0009: compile + upload the core 8 validation notebooks.
#
# Usage:
#   bash .../03-batch-compile-upload-tests.sh

ROOT="/home/manuel/workspaces/2025-12-14/build-remarquee-tool"
REMARQUEE="$ROOT/remarquee"

CASE_DIR="$REMARQUEE/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/cases"
OUTDIR="$REMARQUEE/rendering/rmq-0009-tests"
REMOTE_DIR="/remarquee/rendering/rmq-0009-tests"
REMARKS_BIN="$ROOT/remarks/.venv/bin/remarks"

CASES=(
  "01-empty-1page.yaml"
  "02-single-line.yaml"
  "03-rect.yaml"
  "04-ellipse.yaml"
  "05-two-pages-empty.yaml"
  "06-two-pages-mixed.yaml"
  "07-two-layers.yaml"
  "08-tools-colors.yaml"
  "13-typed-text-basic.yaml"
  "14-glyph-highlight-basic.yaml"
)

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

# Ensure remote folder exists (ignore error if it already exists).
go run ./cmd/remarquee cloud mkdir --non-interactive "$REMOTE_DIR" || true

for f in "${CASES[@]}"; do
  in="$CASE_DIR/$f"
  base="${f%.yaml}"
  out="$OUTDIR/$base.rmdoc"
  outpdf="$OUTDIR/$base-pdf.pdf"
  outremarks="$OUTDIR/$base-remarks.pdf"
  tmpremarks="$OUTDIR/tmp-$base-remarks"

  if [[ ! -f "$in" ]]; then
    echo "missing case: $in" >&2
    exit 1
  fi

  echo "compile: $in -> $out"
  go run ./cmd/remarquee rmdsl compile "$in" --out "$out"

  echo "render: $out -> $outpdf"
  go run ./cmd/remarquee rmdoc render-v6 "$out" --out "$outpdf" --force

  rm -rf "$tmpremarks"
  mkdir -p "$tmpremarks"
  "$REMARKS_BIN" "$out" "$tmpremarks" --log_level ERROR
  remarks_pdf="$(find "$tmpremarks" -type f -name "* _remarks.pdf" | head -n 1 || true)"
  if [[ -z "$remarks_pdf" ]]; then
    echo "no remarks PDF found under $tmpremarks" >&2
    exit 1
  fi
  cp "$remarks_pdf" "$outremarks"
  rm -rf "$tmpremarks"

  echo "upload: $out -> $REMOTE_DIR"
  go run ./cmd/remarquee cloud put --non-interactive --force "$out" "$REMOTE_DIR"

  echo "upload: $outpdf -> $REMOTE_DIR"
  go run ./cmd/remarquee cloud put --non-interactive --force "$outpdf" "$REMOTE_DIR"

  echo "upload: $outremarks -> $REMOTE_DIR"
  go run ./cmd/remarquee cloud put --non-interactive --force "$outremarks" "$REMOTE_DIR"

done

echo "ok: uploaded notebooks + PDFs to $REMOTE_DIR"
go run ./cmd/remarquee cloud ls --non-interactive "$REMOTE_DIR"
