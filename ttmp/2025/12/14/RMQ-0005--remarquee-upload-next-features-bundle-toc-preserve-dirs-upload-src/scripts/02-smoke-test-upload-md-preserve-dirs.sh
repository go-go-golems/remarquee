#!/usr/bin/env bash
set -euo pipefail

# Smoke test for `remarquee upload md --preserve-dirs`.
#
# Creates a small nested markdown fixture directory (including basename collisions),
# uploads it with --preserve-dirs, and verifies visibility by listing remote subfolders.
#
# Requirements:
# - run from the `remarquee/` repo root (so `go run ./cmd/remarquee ...` works)
# - rmapi tokens present (use `--non-interactive` to avoid prompts)
# - pandoc + xelatex installed for PDF conversion
#
# Usage:
#   ./ttmp/.../scripts/02-smoke-test-upload-md-preserve-dirs.sh
#   REMOTE_DIR="/ai/test/my-folder" ./ttmp/.../scripts/02-smoke-test-upload-md-preserve-dirs.sh
#   KEEP_TEMP=1 ./ttmp/.../scripts/02-smoke-test-upload-md-preserve-dirs.sh

REMOTE_DIR="${REMOTE_DIR:-/ai/test/remarquee-upload-md-preserve-dirs-$(date +%Y%m%d-%H%M%S)}"
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

mkdir -p "${testdir}/a" "${testdir}/b/sub"

cat > "${testdir}/a/readme.md" <<'EOF'
# A/readme

Hello from A.
EOF

cat > "${testdir}/b/readme.md" <<'EOF'
# B/readme

Hello from B (same basename, different dir).
EOF

cat > "${testdir}/b/sub/note.md" <<'EOF'
# B/sub/note

Nested.
EOF

echo "=== Local testdir ==="
echo "${testdir}"
find "${testdir}" -type f -maxdepth 4 -print | sort

echo
echo "=== Remote test folder ==="
echo "${REMOTE_DIR}"

echo
echo "=== Dry-run ==="
go run ./cmd/remarquee upload md --dry-run --preserve-dirs --remote-dir "${REMOTE_DIR}" "${testdir}"

echo
echo "=== Upload ==="
go run ./cmd/remarquee upload md --non-interactive --preserve-dirs --remote-dir "${REMOTE_DIR}" "${testdir}"

echo
echo "=== Cloud ls (verify remote dirs exist) ==="
go run ./cmd/remarquee cloud ls "${REMOTE_DIR}" --with-glaze-output --output json
go run ./cmd/remarquee cloud ls "${REMOTE_DIR}/a" --with-glaze-output --output json
go run ./cmd/remarquee cloud ls "${REMOTE_DIR}/b" --with-glaze-output --output json
go run ./cmd/remarquee cloud ls "${REMOTE_DIR}/b/sub" --with-glaze-output --output json


