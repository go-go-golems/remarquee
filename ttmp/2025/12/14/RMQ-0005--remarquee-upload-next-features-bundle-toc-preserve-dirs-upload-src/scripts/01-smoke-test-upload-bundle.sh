#!/usr/bin/env bash
set -euo pipefail

# Smoke test for `remarquee upload bundle`.
#
# Creates a small set of markdown fixtures, bundles them into a single PDF (with ToC),
# uploads it to a new remote test folder, and verifies visibility via cloud ls.
#
# Requirements:
# - run from the `remarquee/` repo root (so `go run ./cmd/remarquee ...` works)
# - rmapi tokens present (use `--non-interactive` to avoid prompts)
# - pandoc + xelatex installed for PDF conversion
#
# Usage:
#   ./ttmp/.../scripts/01-smoke-test-upload-bundle.sh
#   REMOTE_DIR="/ai/test/my-folder" ./ttmp/.../scripts/01-smoke-test-upload-bundle.sh
#   BUNDLE_NAME="my-bundle" ./ttmp/.../scripts/01-smoke-test-upload-bundle.sh
#   KEEP_TEMP=1 ./ttmp/.../scripts/01-smoke-test-upload-bundle.sh

REMOTE_DIR="${REMOTE_DIR:-/ai/test/remarquee-upload-bundle-$(date +%Y%m%d-%H%M%S)}"
BUNDLE_NAME="${BUNDLE_NAME:-bundle-smoke}"
KEEP_TEMP="${KEEP_TEMP:-}"

testdir="$(mktemp -d)"
cleanup() {
  if [[ -z "${KEEP_TEMP}" ]]; then
    rm -rf "${testdir}"
  else
    echo "KEEP_TEMP=1 set; keeping testdir: ${testdir}" >&2
  fi
}
trap cleanup EXIT

cat > "${testdir}/01-frontmatter-and-lists.md" <<'EOF'
---
Title: Bundle smoke test: frontmatter + lists
Topics:
  - backend
---
This is a paragraph.
- list item 1
- list item 2

1. ordered item 1
2. ordered item 2
EOF

cat > "${testdir}/02-unicode.md" <<'EOF'
# Unicode

Café → naïve — 中文 — русский
EOF

mkdir -p "${testdir}/sub"
cat > "${testdir}/sub/03-subdir.md" <<'EOF'
# Subdir

This file lives in a subdirectory.
EOF

echo "=== Local testdir ==="
echo "${testdir}"
find "${testdir}" -type f -maxdepth 3 -print | sort

echo
echo "=== Remote test folder ==="
echo "${REMOTE_DIR}"

echo
echo "=== Dry-run ==="
go run ./cmd/remarquee upload bundle --dry-run --name "${BUNDLE_NAME}" --remote-dir "${REMOTE_DIR}" "${testdir}"

echo
echo "=== Upload ==="
go run ./cmd/remarquee upload bundle --non-interactive --name "${BUNDLE_NAME}" --remote-dir "${REMOTE_DIR}" "${testdir}"

echo
echo "=== Cloud ls (verify bundled doc exists) ==="
go run ./cmd/remarquee cloud ls "${REMOTE_DIR}" --with-glaze-output --output json


