#!/usr/bin/env bash
set -euo pipefail

ROOT="/home/manuel/workspaces/2025-12-14/build-remarquee-tool"
REMARQUEE="$ROOT/remarquee"
CASES_DIR="$REMARQUEE/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/cases"
OUTDIR="$REMARQUEE/rendering/rmq-0009-complicated"
REMOTE_DIR="/remarquee/rendering/rmq-0009-complicated"
REMARKS_BIN="$ROOT/remarks/.venv/bin/remarks"

CASES=(
  "09-complex-spiral-grid"
  "10-layered-highlighter"
  "11-two-pages-dense"
  "12-tools-showcase-advanced"
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

go run ./cmd/remarquee cloud mkdir --non-interactive "$REMOTE_DIR" || true

for name in "${CASES[@]}"; do
  case_file="$CASES_DIR/${name}.yaml"
  outdoc="$OUTDIR/${name}.rmdoc"
  outpdf="$OUTDIR/${name}-pdf.pdf"
  outremarks="$OUTDIR/${name}-remarks.pdf"
  tmpremarks="$OUTDIR/tmp-${name}-remarks"

  if [[ ! -f "$case_file" ]]; then
    echo "missing case: $case_file" >&2
    exit 1
  fi

  echo "compile: $case_file -> $outdoc"
  go run ./cmd/remarquee rmdsl compile "$case_file" --out "$outdoc"

  echo "render: $outdoc -> $outpdf"
  go run ./cmd/remarquee rmdoc render-v6 "$outdoc" --out "$outpdf" --force

  rm -rf "$tmpremarks"
  mkdir -p "$tmpremarks"
  "$REMARKS_BIN" "$outdoc" "$tmpremarks" --log_level ERROR
  remarks_pdf="$(find "$tmpremarks" -type f -name "* _remarks.pdf" | head -n 1 || true)"
  if [[ -z "$remarks_pdf" ]]; then
    echo "no remarks PDF found under $tmpremarks" >&2
    exit 1
  fi
  cp "$remarks_pdf" "$outremarks"
  rm -rf "$tmpremarks"

  echo "upload: $outdoc -> $REMOTE_DIR"
  go run ./cmd/remarquee cloud put --non-interactive --force "$outdoc" "$REMOTE_DIR"

  echo "upload: $outpdf -> $REMOTE_DIR"
  go run ./cmd/remarquee cloud put --non-interactive --force "$outpdf" "$REMOTE_DIR"

  echo "upload: $outremarks -> $REMOTE_DIR"
  go run ./cmd/remarquee cloud put --non-interactive --force "$outremarks" "$REMOTE_DIR"

done

echo "ok: uploaded .rmdoc + PDFs to $REMOTE_DIR"
