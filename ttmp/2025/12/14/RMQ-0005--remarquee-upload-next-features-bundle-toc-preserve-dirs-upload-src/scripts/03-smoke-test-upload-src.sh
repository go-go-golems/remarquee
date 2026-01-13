#!/usr/bin/env bash
set -euo pipefail

# Smoke test for `remarquee upload src`.
#
# Creates a small set of source code fixtures (with a subdir), uploads them as PDFs
# with syntax highlighting, and verifies visibility via cloud ls.
#
# Requirements:
# - run from the `remarquee/` repo root (so `go run ./cmd/remarquee ...` works)
# - rmapi tokens present (use `--non-interactive` to avoid prompts)
# - pandoc + xelatex installed for PDF conversion
#
# Usage:
#   ./ttmp/.../scripts/03-smoke-test-upload-src.sh
#   REMOTE_DIR="/ai/test/my-folder" ./ttmp/.../scripts/03-smoke-test-upload-src.sh
#   KEEP_TEMP=1 ./ttmp/.../scripts/03-smoke-test-upload-src.sh

REMOTE_DIR="${REMOTE_DIR:-/ai/test/remarquee-upload-src-$(date +%Y%m%d-%H%M%S)}"
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

mkdir -p "${testdir}/sub"

cat > "${testdir}/hello.go" <<'EOF'
package main

import "fmt"

func main() {
	fmt.Println("hello")
}
EOF

cat > "${testdir}/sub/example.c" <<'EOF'
#include <stdio.h>

int main(void) {
    printf("hi\n");
    return 0;
}
EOF

cat > "${testdir}/config.yaml" <<'EOF'
foo: bar
items:
  - one
  - two
EOF

echo "=== Local testdir ==="
echo "${testdir}"
find "${testdir}" -type f -maxdepth 3 -print | sort

echo
echo "=== Remote test folder ==="
echo "${REMOTE_DIR}"

echo
echo "=== Dry-run ==="
go run ./cmd/remarquee upload src --dry-run --remote-dir "${REMOTE_DIR}" "${testdir}"

echo
echo "=== Upload ==="
go run ./cmd/remarquee upload src --non-interactive --remote-dir "${REMOTE_DIR}" "${testdir}"

echo
echo "=== Cloud ls (verify uploaded docs exist) ==="
go run ./cmd/remarquee cloud ls "${REMOTE_DIR}" --with-glaze-output --output json


