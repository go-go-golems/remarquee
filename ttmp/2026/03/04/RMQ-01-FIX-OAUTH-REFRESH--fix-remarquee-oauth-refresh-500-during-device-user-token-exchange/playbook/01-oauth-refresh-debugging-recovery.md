---
Title: OAuth refresh debugging + recovery
Ticket: RMQ-01-FIX-OAUTH-REFRESH
Status: active
Topics:
    - backend
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles:
    - Path: rmapi/api/auth.go
      Note: Auth bootstrap; key for recovery steps
    - Path: rmapi/config/config.go
      Note: Token cache path and file format referenced by playbook
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-04T10:09:18.582839098-05:00
WhatFor: ""
WhenToUse: ""
---


# OAuth refresh debugging + recovery

## Purpose

Provide a repeatable command sequence to:

- diagnose failures during device token → user token exchange (`--reauth`)
- capture safe traces for debugging
- recover from invalid tokens (401) or transient backend errors (5xx)

## Environment Assumptions

- `remarquee` is installed and on `$PATH`
- `rmapi` is available (either as a binary or via `remarquee` dependency)
- You have either:
  - a valid cached device token (typical), or
  - access to the one-time code flow (interactive mode)
- You can access `https://my.remarkable.com/device/browser/connect` if you need to re-register

## Commands

### 1) Basic sanity: is remarquee healthy?

```bash
remarquee status
remarquee cloud version
```

## Exit Criteria

- `remarquee cloud account --reauth` prints `user=<...> sync_version=<...>`
- If debugging is needed, you have a *redacted* trace file that can be shared safely

## Notes (copy/paste cookbook)

### Check account without reauth (cached user token)

```bash
remarquee cloud account --non-interactive
```

Expected output:

```
user=<email> sync_version=<...>
```

If it fails with missing tokens:

```bash
remarquee cloud account
```

### Force reauth (device token → new user token)

```bash
remarquee cloud account --reauth --non-interactive
```

If it fails with HTTP 500:

- treat as backend/transient
- re-run once or twice (the tool now includes bounded retries + backoff)
- if it persists, capture a trace

### Capture an rmapi HTTP trace (SENSITIVE; redact before sharing)

This writes a full HTTP request/response dump to stdout, which typically includes an `Authorization: Bearer ...` header.

```bash
RMAPI_TRACE=1 remarquee cloud account --reauth --non-interactive > /tmp/rmapi-trace.txt 2>&1
```

Redact before attaching to a ticket or sending to anyone:

```bash
python3 - <<'PY'\nimport re\nsrc=open('/tmp/rmapi-trace.txt','r',errors='replace').read().splitlines(True)\npatterns=[\n  (re.compile(r'(Authorization: Bearer )([^\\r\\n]+)'), r'\\1<REDACTED>'),\n  (re.compile(r'\\beyJ[a-zA-Z0-9_-]{10,}\\.[a-zA-Z0-9_-]{10,}\\.[a-zA-Z0-9_-]{10,}\\b'), '<REDACTED_JWT>'),\n]\nout=[]\nfor line in src:\n  red=line\n  for pat,repl in patterns:\n    red=pat.sub(repl, red)\n  out.append(red)\nopen('/tmp/rmapi-trace.redacted.txt','w').write(''.join(out))\nprint('wrote /tmp/rmapi-trace.redacted.txt')\nPY\n```

### Recovery when device token is invalid (401)

```bash
rmapi reset
rmapi account   # interactive: will ask for one-time code
remarquee cloud account --reauth
```

### Persistent failures (triage guidance)

- Persistent `request failed with status 500`:
  - backend issue or outage
  - wait 10–30 minutes and retry
  - keep a redacted trace for escalation
- Persistent 401:
  - device token invalid/stale; reset + re-register
