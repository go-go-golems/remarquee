#!/usr/bin/env bash
set -euo pipefail

# Render Test.rmdoc pages 1-2 to PNGs via remarquee->PDF->pdftoppm.
#
# Usage:
#   bash .../12-render-test-rmdoc-pages1-2-pngs.sh
#
# Outputs:
#   Prints WORKDIR and A1_PNG/A2_PNG paths.

ROOT="/home/manuel/workspaces/2025-12-14/build-remarquee-tool"
REMARQUEE="$ROOT/remarquee"
FIXTURE="$REMARQUEE/cmd/remarquee-ui/testdata/Test.rmdoc"

if [[ ! -f "$FIXTURE" ]]; then
  echo "fixture missing: $FIXTURE" >&2
  exit 1
fi

if ! command -v pdftoppm >/dev/null 2>&1; then
  echo "pdftoppm missing on PATH" >&2
  exit 1
fi

WORKDIR="$(mktemp -d)"
echo "WORKDIR=$WORKDIR"

cd "$REMARQUEE"
go run ./cmd/remarquee rmdoc render-v6 "$FIXTURE" --out "$WORKDIR/A-remarquee.pdf" --force

pdftoppm -png -r 200 -f 1 -l 2 "$WORKDIR/A-remarquee.pdf" "$WORKDIR/A" >/dev/null 2>&1

echo "A1_PNG=$WORKDIR/A-1.png"
echo "A2_PNG=$WORKDIR/A-2.png"


