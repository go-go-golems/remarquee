---
Title: Analysis
Ticket: RMQ-0010
Status: active
Topics:
    - backend
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarquee/pkg/rmcloud/auth.go
      Note: CreateApiCtx entrypoint
    - Path: remarquee/ttmp/2026/01/11/RMQ-0010--rmapi-reauth-investigation/scripts/decode_rmapi_tokens.py
      Note: Payload inspection
    - Path: rmapi/api/api.go
      Note: ParseToken logic
    - Path: rmapi/api/auth.go
      Note: AuthHttpCtx and token refresh
    - Path: rmapi/main.go
      Note: AUTH_RETRIES pattern
ExternalSources: []
Summary: Analyze rmapi reauth flow and how remarquee should handle expired user tokens
LastUpdated: 2026-01-11T18:44:25-05:00
WhatFor: Guide fixes for rmapi token expiration handling in remarquee
WhenToUse: When implementing or reviewing rmapi reauth handling
---


# Analysis

## Executive Summary

`remarquee` previously called rmapi auth logic once and then parsed the cached user token. If the cached token was expired, the command failed immediately with `token Expired`. I added a retry loop in `rmcloud.CreateApiCtx` (matching rmapi's `AUTH_RETRIES`) so `reauth=true` is attempted on subsequent passes, which resolves expired-token failures in the common case.

## Problem Statement

The `remarquee` command path treats rmapi authentication as a single-shot operation. With an expired `usertoken` in `~/.config/rmapi/rmapi.conf`, `rmcloud.CreateApiCtx` returns an error instead of retrying. Users must manually run rmapi (or delete the config) to refresh tokens, which is both confusing and inconsistent with `rmapi` behavior.

## Proposed Solution

Add a small retry loop in `rmcloud.CreateApiCtx` (or wrap it in the `remarquee` command layer) to mirror rmapi's `AUTH_RETRIES` behavior. Implemented in `remarquee/pkg/rmcloud/auth.go` with `authRetries=3`:

- Attempt auth and parse the token.
- If `ParseToken` fails with expiration or parse errors, retry once with `reauth=true`.
- If reauth fails due to device token invalidation, surface a clear message ("device token invalid; re-register device with rmapi reset / one-time code") instead of a generic parse error.

## Design Decisions

- Keep using rmapi's `AuthHttpCtx` to avoid duplicating token acquisition logic.
- Match rmapi's retry count (`authRetries=3`) to keep behavior aligned with the upstream CLI.
- Preserve explicit `--reauth` behavior but also auto-reauth on parse failure to reduce user friction.

## Alternatives Considered

- Require users to always pass `--reauth` or manually run `rmapi account` before using remarquee.
- Add a new `remarquee cloud reauth` command to write tokens explicitly.
- Delete `rmapi.conf` on parse failure (too destructive without confirmation).

## Implementation Plan

1. Add retry helper in `remarquee/pkg/rmcloud/auth.go` (done in commit `6a665f4`):
   - Attempt `CreateApiCtx` with `Reauth=false` (or the requested flag).
   - On `token Expired` or parse error, call `AuthHttpCtx` again with `Reauth=true` and re-parse.
2. Thread improved error messages to CLI callers (`cmd/remarquee/cmds/cloud/...`) (done via rmcloud error hints in commit `6a665f4` and `79cd1da`).
3. Add a small test or harness using a fake expired token to validate retry behavior.
4. Document the reauth behavior in CLI help or a playbook if needed.

## Open Questions

- Should auto-reauth always run, or only when `--reauth` is passed?
- When device token is invalid, should remarquee prompt for one-time code (interactive) or fail fast?
- Should we track the last reauth time to avoid repeated refresh attempts?

## References

- `remarquee/pkg/rmcloud/auth.go:CreateApiCtx`
- `remarquee/cmd/remarquee/cmds/cloud/account.go`
- `rmapi/api/auth.go:AuthHttpCtx`
- `rmapi/api/api.go:ParseToken`
- `rmapi/main.go` retry loop
- `remarquee/ttmp/2026/01/11/RMQ-0010--rmapi-reauth-investigation/scripts/decode_rmapi_tokens.py`
- `remarquee/ttmp/2026/01/11/RMQ-0010--rmapi-reauth-investigation/playbook/01-reauth-recovery.md`
