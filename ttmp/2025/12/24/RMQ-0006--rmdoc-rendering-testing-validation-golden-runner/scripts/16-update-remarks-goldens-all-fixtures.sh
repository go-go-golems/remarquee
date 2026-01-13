#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
TICKET_DIR="$(cd -- "${SCRIPT_DIR}/.." >/dev/null 2>&1 && pwd)"
REPO_DIR="$(cd -- "${TICKET_DIR}/../../../../.." >/dev/null 2>&1 && pwd)"

echo "Repo: ${REPO_DIR}"
echo "Ticket: ${TICKET_DIR}"

cd "${REPO_DIR}"

if ! command -v remarks >/dev/null 2>&1; then
  echo "error: remarks not found on PATH; install it (see RMQ-0006 decision doc) and re-run" >&2
  exit 1
fi

go test ./pkg/rmdoc/render -run TestUpdateRemarksGoldens -update-golden -count=1

echo
echo "Goldens updated under: cmd/remarquee-ui/testdata/golden/"

