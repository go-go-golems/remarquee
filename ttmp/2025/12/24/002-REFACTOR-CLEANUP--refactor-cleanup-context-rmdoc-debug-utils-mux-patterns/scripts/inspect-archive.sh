#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   ./inspect-archive.sh /abs/path/to/file.rmdoc
#   ./inspect-archive.sh /abs/path/to/file.zip
#
# Output:
# - Lists archive entries (name + uncompressed size)
# - For *.rm files, prints the first 43 bytes (header) as a string
#
# Notes:
# - This is a convenience script for debugging / tests. It does not modify files.
# - Keep scripts inside the ticket `scripts/` folder so they can be reused later.

if [[ $# -ne 1 ]]; then
  echo "usage: $0 /abs/path/to/archive.(rmdoc|zip)" >&2
  exit 2
fi

ARCHIVE="$1"
if [[ ! -f "$ARCHIVE" ]]; then
  echo "error: file not found: $ARCHIVE" >&2
  exit 2
fi

python3 - "$ARCHIVE" <<'PY'
import sys
import zipfile
from pathlib import Path

p = Path(sys.argv[1])
print(f"archive: {p}")

z = zipfile.ZipFile(p)
infos = z.infolist()
print(f"entries: {len(infos)}")
for i, info in enumerate(infos):
    name = info.filename
    size = info.file_size
    print(f"[{i:03d}] {name}\t{size}")
    if name.endswith(".rm"):
        with z.open(info, "r") as f:
            header = f.read(43)
        try:
            header_s = header.decode("utf-8", errors="replace")
        except Exception:
            header_s = repr(header)
        print(f"      rm.header={header_s!r}")
PY


