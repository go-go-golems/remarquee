#!/usr/bin/env bash
set -euo pipefail

# RMQ-0009: compile the ellipse sweep DSL case into a .rmdoc notebook.
#
# Usage:
#   bash .../01-compile-ellipse-sweep-rmdoc.sh

ROOT="/home/manuel/workspaces/2025-12-14/build-remarquee-tool"
REMARQUEE="$ROOT/remarquee"

CASE_JS="$REMARQUEE/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/cases/03-ellipse-sweep.js"
OUTDIR="$REMARQUEE/rendering/rmq-0009-ellipse"
OUTDOC="$OUTDIR/ellipse-sweep.rmdoc"

if [[ ! -f "$CASE_JS" ]]; then
  echo "missing case: $CASE_JS" >&2
  exit 1
fi

mkdir -p "$OUTDIR"

cd "$REMARQUEE"

go run ./cmd/remarquee rmdsl compile "$CASE_JS" --out "$OUTDOC"

echo "ok: wrote $OUTDOC"
