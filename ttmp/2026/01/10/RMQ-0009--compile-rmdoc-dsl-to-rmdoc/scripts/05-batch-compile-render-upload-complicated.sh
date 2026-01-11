#!/usr/bin/env bash
set -euo pipefail

ROOT="/home/manuel/workspaces/2025-12-14/build-remarquee-tool"
REMARQUEE="$ROOT/remarquee"
CASES_DIR="$REMARQUEE/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/cases"
OUTDIR="$REMARQUEE/rendering/rmq-0009-complicated"
REMOTE_DIR="/remarquee/rendering/rmq-0009-complicated"

CASES=(
  "09-complex-spiral-grid"
  "10-layered-highlighter"
  "11-two-pages-dense"
  "12-tools-showcase-advanced"
)

mkdir -p "$OUTDIR"
cd "$REMARQUEE"

for name in "${CASES[@]}"; do
  case_file="$CASES_DIR/${name}.yaml"
  outdoc="$OUTDIR/${name}.rmdoc"
  outpdf="$OUTDIR/${name}-pdf.pdf"

  if [[ ! -f "$case_file" ]]; then
    echo "missing case: $case_file" >&2
    exit 1
  fi

  echo "compile: $case_file -> $outdoc"
  go run ./cmd/remarquee rmdsl compile "$case_file" --out "$outdoc"

  echo "render: $outdoc -> $outpdf"
  go run ./cmd/remarquee rmdoc render-v6 "$outdoc" --out "$outpdf" --force

  echo "upload: $outdoc -> $REMOTE_DIR"
  go run ./cmd/remarquee cloud put --non-interactive --force "$outdoc" "$REMOTE_DIR"

  echo "upload: $outpdf -> $REMOTE_DIR"
  go run ./cmd/remarquee cloud put --non-interactive --force "$outpdf" "$REMOTE_DIR"

done

echo "ok: uploaded .rmdoc + .pdf to $REMOTE_DIR"
