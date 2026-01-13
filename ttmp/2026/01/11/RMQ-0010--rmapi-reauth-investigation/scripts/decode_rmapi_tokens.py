#!/usr/bin/env python3
"""Decode rmapi.conf JWT payloads for quick inspection.

Usage:
  python decode_rmapi_tokens.py [path/to/rmapi.conf]
"""

import base64
import json
import sys
import time
from pathlib import Path


def decode_payload(token: str):
    parts = token.split(".")
    if len(parts) < 2:
        return None
    payload = parts[1]
    padding = "=" * ((4 - len(payload) % 4) % 4)
    try:
        raw = base64.urlsafe_b64decode(payload + padding)
        return json.loads(raw)
    except Exception as exc:  # noqa: BLE001 - utility script
        return {"error": str(exc)}


def main() -> int:
    conf_path = Path(sys.argv[1]) if len(sys.argv) > 1 else Path.home() / ".config" / "rmapi" / "rmapi.conf"
    if not conf_path.exists():
        print(f"missing config: {conf_path}")
        return 1

    tokens = {}
    for line in conf_path.read_text().splitlines():
        if ":" not in line:
            continue
        key, value = line.split(":", 1)
        tokens[key.strip()] = value.strip()

    now = int(time.time())
    print(f"now_unix={now}")
    for key in ("devicetoken", "usertoken"):
        payload = decode_payload(tokens.get(key, ""))
        print(f"{key}={json.dumps(payload, sort_keys=True)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
