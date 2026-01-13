#!/usr/bin/env bash
set -euo pipefail

# Scenario: bundle the remarquee codebase (cmd/ + pkg/) into a single syntax-highlighted PDF.
#
# This exercises:
# - `remarquee upload src --bundle`
# - deterministic ordering for directory inputs
# - ToC generation and large-input handling
#
# Requirements:
# - run from the `remarquee/` repo root (so `go run ./cmd/remarquee ...` works)
# - pandoc + xelatex installed
# - rmapi tokens present if uploading (use --non-interactive)
#
# Usage:
#   ./ttmp/.../scripts/04-scenario-upload-src-bundle-remarquee-codebase.sh
#   REMOTE_DIR="/ai/test/my-folder" ./ttmp/.../scripts/04-scenario-upload-src-bundle-remarquee-codebase.sh
#   PDF_ONLY=1 ./ttmp/.../scripts/04-scenario-upload-src-bundle-remarquee-codebase.sh

REMOTE_DIR="${REMOTE_DIR:-/ai/test/remarquee-src-bundle-codebase-$(date +%Y%m%d-%H%M%S)}"
PDF_ONLY="${PDF_ONLY:-}"

echo "=== Remote test folder ==="
echo "${REMOTE_DIR}"

echo
echo "=== Dry-run ==="
go run ./cmd/remarquee upload src --dry-run --bundle --name remarquee-codebase --toc-depth 2 --include-ext .go --remote-dir "${REMOTE_DIR}" cmd pkg

echo
if [[ -n "${PDF_ONLY}" ]]; then
  echo "=== PDF-only (no upload) ==="
  go run ./cmd/remarquee upload src --bundle --pdf-only --name remarquee-codebase --toc-depth 2 --include-ext .go --remote-dir "${REMOTE_DIR}" cmd pkg
  exit 0
fi

echo "=== Upload ==="
go run ./cmd/remarquee upload src --non-interactive --bundle --name remarquee-codebase --toc-depth 2 --include-ext .go --remote-dir "${REMOTE_DIR}" cmd pkg

echo
echo "=== Cloud ls (verify bundled doc exists) ==="
go run ./cmd/remarquee cloud ls "${REMOTE_DIR}" --with-glaze-output --output json


