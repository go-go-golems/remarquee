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

CASES=(
  "01-empty-1page.yaml"
  "02-single-line.yaml"
  "03-rect.yaml"
  "04-ellipse.yaml"
  "05-two-pages-empty.yaml"
  "06-two-pages-mixed.yaml"
  "07-two-layers.yaml"
  "08-tools-colors.yaml"
)

mkdir -p "$OUTDIR"

cd "$REMARQUEE"

# Ensure remote folder exists (ignore error if it already exists).
go run ./cmd/remarquee cloud mkdir --non-interactive "$REMOTE_DIR" || true

for f in "${CASES[@]}"; do
  in="$CASE_DIR/$f"
  base="${f%.yaml}"
  out="$OUTDIR/$base.rmdoc"

  if [[ ! -f "$in" ]]; then
    echo "missing case: $in" >&2
    exit 1
  fi

  echo "compile: $in -> $out"
  go run ./cmd/remarquee rmdsl compile "$in" --out "$out"

  echo "upload: $out -> $REMOTE_DIR"
  go run ./cmd/remarquee cloud put --non-interactive --force "$out" "$REMOTE_DIR"

done

echo "ok: uploaded core 8 notebooks to $REMOTE_DIR"
go run ./cmd/remarquee cloud ls --non-interactive "$REMOTE_DIR"
