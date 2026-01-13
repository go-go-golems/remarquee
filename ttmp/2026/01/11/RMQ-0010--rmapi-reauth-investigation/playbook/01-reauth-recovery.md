---
Title: Reauth Recovery
Ticket: RMQ-0010
Status: active
Topics:
    - backend
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarquee/cmd/remarquee/cmds/cloud/account.go
      Note: Help text mentions reauth/reset steps
    - Path: remarquee/pkg/rmcloud/auth.go
      Note: Auth retry and guidance
    - Path: remarquee/ttmp/2026/01/11/RMQ-0010--rmapi-reauth-investigation/scripts/decode_rmapi_tokens.py
      Note: Token inspection helper
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-11T18:44:34-05:00
WhatFor: Recover from rmapi token expiration or invalid device tokens
WhenToUse: When remarquee reports token expiration or cannot parse a user token
---


# Reauth Recovery

## Purpose

Restore working rmapi authentication for `remarquee` when cached tokens are expired or invalid.

## Environment Assumptions

- You have `remarquee` and `rmapi` available in the shell.
- You can open the reMarkable device registration page to obtain a one-time code if needed.

## Commands

```bash
# 1) Confirm auth status
cd /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee
go run ./cmd/remarquee cloud account

# 2) If auth failed with token expiration, retry with reauth
go run ./cmd/remarquee cloud account --reauth

# 3) If auth still fails and guidance says the device token is invalid, reset rmapi
rmapi reset

# 4) Re-register the device (follow the prompt for the one-time code)
rmapi account

# 5) Confirm remarquee auth is working again
go run ./cmd/remarquee cloud account
```

## Exit Criteria

- `remarquee cloud account` prints `user=<email> sync_version=1.5`.
- No `token Expired` or missing-user-token errors.

## Notes

- `rmapi reset` removes the local token cache; only use when reauth fails.
- If you need to inspect token payloads, use:
  `python /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/11/RMQ-0010--rmapi-reauth-investigation/scripts/decode_rmapi_tokens.py`
