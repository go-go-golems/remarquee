---
Title: Bug Report
Ticket: RMQ-0010
Status: active
Topics:
    - backend
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarquee/pkg/rmcloud/auth.go
      Note: Remarquee auth entrypoint
    - Path: remarquee/ttmp/2026/01/11/RMQ-0010--rmapi-reauth-investigation/scripts/decode_rmapi_tokens.py
      Note: Token inspection script
    - Path: rmapi/api/api.go
      Note: ParseToken expiration handling
    - Path: rmapi/main.go
      Note: rmapi retry loop for auth
ExternalSources: []
Summary: remarquee cloud commands fail when rmapi user tokens are expired; fixed by retrying auth
LastUpdated: 2026-01-11T18:28:43-05:00
WhatFor: Capture reproduction, impact, and current fix for rmapi reauth
WhenToUse: Use when remarquee reports token expiration or reauth fails
---


# Bug Report

## Goal

Capture the current failure mode when `remarquee` hits an expired rmapi user token and summarize what reauth does (and does not) do today.

## Context

`remarquee` uses the rmapi token cache in `~/.config/rmapi/rmapi.conf`. The command path `remarquee cloud account` calls `rmcloud.CreateApiCtx`, which calls `rmapi/api.AuthHttpCtx` and then `rmapi/api.ParseToken`. If `ParseToken` returns `token Expired`, the command exits immediately.

## Quick Reference

Reproduction (when `usertoken` in `rmapi.conf` is expired):

```bash
# From /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee

go run ./cmd/remarquee cloud account
```

Observed error (before fix):

```
Error: failed to parse rmapi user token: token Expired
```

Control comparison: `rmapi` retries auth and typically succeeds even with an expired token.

```bash
rmapi account
```

Key code paths:

- `remarquee/pkg/rmcloud/auth.go:CreateApiCtx` (calls `api.AuthHttpCtx`, then `api.ParseToken` once)
- `rmapi/api/api.go:ParseToken` (returns `errors.New("token Expired")`)
- `rmapi/main.go` retry loop (sets `reAuth` on second attempt)

## Usage Examples

Inspect token expiry quickly:

```bash
python /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/11/RMQ-0010--rmapi-reauth-investigation/scripts/decode_rmapi_tokens.py
```

Expected behavior (goal):

- If `ParseToken` fails due to expiration, `remarquee` should retry once with `reauth=true` or clearly instruct how to refresh tokens.

Current status:

- Implemented a retry loop in `remarquee/pkg/rmcloud/auth.go` (`authRetries=3`) so expired tokens trigger a reauth attempt.

## Related

- `remarquee/ttmp/2026/01/11/RMQ-0010--rmapi-reauth-investigation/design-doc/01-analysis.md`
