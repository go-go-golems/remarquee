#!/usr/bin/env bash
set -euo pipefail

# RMQ-0006 helper: generate A/B PDFs for cpage-pdf.rmdoc and print size diagnostics
# (PDF boxes + pdftoppm raster dims).
#
# Usage:
#   bash .../06-debug-golden-size-mismatch-cpage-pdf.sh

ROOT="/home/manuel/workspaces/2025-12-14/build-remarquee-tool"
REMARQUEE="$ROOT/remarquee"
SCRIPTS="$REMARQUEE/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts"

OUT_LOG="$(mktemp -t rmq0006-ab-cpage-XXXXXX.log)"
bash "$SCRIPTS/05-generate-a-vs-b-pdfs-cpage-pdf.sh" | tee "$OUT_LOG"

A_PDF="$(grep '^A_PDF=' "$OUT_LOG" | tail -n1 | cut -d= -f2-)"
B_PDF="$(grep '^B_PDF=' "$OUT_LOG" | tail -n1 | cut -d= -f2-)"

echo
echo "== boxes (unidoc) =="
cd "$REMARQUEE"
go run "$SCRIPTS/03-pdf-box-dump.go" "$A_PDF"
echo
go run "$SCRIPTS/03-pdf-box-dump.go" "$B_PDF"

echo
echo "== raster dims (pdftoppm) =="
bash "$SCRIPTS/02-measure-pdf-page-and-raster-dims.sh" "$A_PDF" "$B_PDF"

echo
echo "LOG=$OUT_LOG"


