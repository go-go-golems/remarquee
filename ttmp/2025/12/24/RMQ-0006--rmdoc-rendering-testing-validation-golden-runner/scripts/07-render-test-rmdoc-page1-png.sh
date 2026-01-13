#!/usr/bin/env bash
set -euo pipefail

# Render page 1 of Test.rmdoc to a PNG via remarquee->PDF->pdftoppm.
#
# Usage:
#   bash .../07-render-test-rmdoc-page1-png.sh
#
# Outputs:
#   Prints WORKDIR and A_PNG path.

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

pdftoppm -png -r 200 -f 1 -l 1 -singlefile "$WORKDIR/A-remarquee.pdf" "$WORKDIR/A-page-001" >/dev/null 2>&1

echo "A_PNG=$WORKDIR/A-page-001.png"


