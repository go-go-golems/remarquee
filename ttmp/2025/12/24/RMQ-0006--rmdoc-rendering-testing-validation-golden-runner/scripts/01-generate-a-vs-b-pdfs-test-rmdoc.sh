#!/usr/bin/env bash
set -euo pipefail

# Generates A (remarquee) and B (remarks reference) PDFs for Test.rmdoc.
#
# Usage:
#   bash .../01-generate-a-vs-b-pdfs-test-rmdoc.sh
#
# Outputs:
#   Prints A_PDF and B_PDF paths and the workspace directory.

ROOT="/home/manuel/workspaces/2025-12-14/build-remarquee-tool"
REMARQUEE="$ROOT/remarquee"
FIXTURE="$REMARQUEE/cmd/remarquee-ui/testdata/Test.rmdoc"
REMARKS_BIN="$ROOT/remarks/.venv/bin/remarks"

if [[ ! -f "$FIXTURE" ]]; then
  echo "fixture missing: $FIXTURE" >&2
  exit 1
fi

if [[ ! -x "$REMARKS_BIN" ]]; then
  echo "remarks not found at $REMARKS_BIN" >&2
  echo "hint: install remarks via poetry and ensure .venv exists" >&2
  exit 1
fi

TMPDIR="$(mktemp -d)"
echo "WORKDIR=$TMPDIR"

cd "$REMARQUEE"

go run ./cmd/remarquee rmdoc render-v6 "$FIXTURE" --out "$TMPDIR/A-remarquee.pdf" --force

"$REMARKS_BIN" "$FIXTURE" "$TMPDIR/remarks-out" --log_level ERROR
B_PDF="$(find "$TMPDIR/remarks-out" -type f -name "* _remarks.pdf" | head -n 1 || true)"
if [[ -z "$B_PDF" ]]; then
  echo "no remarks PDF found under $TMPDIR/remarks-out" >&2
  exit 1
fi

echo "A_PDF=$TMPDIR/A-remarquee.pdf"
echo "B_PDF=$B_PDF"


