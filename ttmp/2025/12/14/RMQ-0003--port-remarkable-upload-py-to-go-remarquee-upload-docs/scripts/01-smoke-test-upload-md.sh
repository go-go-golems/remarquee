#!/usr/bin/env bash
set -euo pipefail

# Smoke test for `remarquee upload md`.
#
# Creates a small set of markdown fixtures, uploads them to a new remote test folder,
# and verifies visibility by listing the remote folder.
#
# Requirements:
# - run from the `remarquee/` repo root (so `go run ./cmd/remarquee ...` works)
# - rmapi tokens present (use `--non-interactive` to avoid prompts)
# - pandoc + xelatex installed for PDF conversion
#
# Usage:
#   ./ttmp/.../scripts/01-smoke-test-upload-md.sh
#   REMOTE_DIR="/ai/test/my-folder" ./ttmp/.../scripts/01-smoke-test-upload-md.sh
#   KEEP_TEMP=1 ./ttmp/.../scripts/01-smoke-test-upload-md.sh

REMOTE_DIR="${REMOTE_DIR:-/ai/test/remarquee-upload-md-$(date +%Y%m%d-%H%M%S)}"
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
Title: Upload smoke test: frontmatter + lists
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

- bullet
EOF

cat > "${testdir}/03-code.md" <<'EOF'
# Code

```go
package main

func main() {
    println("hello")
}
```
EOF

mkdir -p "${testdir}/sub"
cat > "${testdir}/sub/04-subdir.md" <<'EOF'
# Subdir

This file lives in a subdirectory.
EOF

cat > "${testdir}/05-empty-frontmatter.md" <<'EOF'
---
---
# Empty frontmatter

Body.
EOF

echo "=== Local testdir ==="
echo "${testdir}"
find "${testdir}" -type f -maxdepth 3 -print | sort

echo
echo "=== Remote test folder ==="
echo "${REMOTE_DIR}"

echo
echo "=== Dry-run ==="
go run ./cmd/remarquee upload md --dry-run --remote-dir "${REMOTE_DIR}" "${testdir}"

echo
echo "=== Upload ==="
go run ./cmd/remarquee upload md --non-interactive --remote-dir "${REMOTE_DIR}" "${testdir}"

echo
echo "=== Cloud ls (verify uploaded docs exist) ==="
go run ./cmd/remarquee cloud ls "${REMOTE_DIR}" --with-glaze-output --output json


