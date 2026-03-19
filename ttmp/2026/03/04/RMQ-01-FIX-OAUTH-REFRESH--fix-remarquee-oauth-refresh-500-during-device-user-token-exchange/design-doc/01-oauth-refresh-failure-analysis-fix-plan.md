---
Title: OAuth refresh failure analysis + fix plan
Ticket: RMQ-01-FIX-OAUTH-REFRESH
Status: active
Topics:
    - backend
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarquee/cmd/remarquee/cmds/cloud/account.go
      Note: Command entrypoint calling rmcloud.CreateApiCtx
    - Path: remarquee/pkg/rmcloud/auth.go
      Note: Remarquee wrapper; retries + backoff and token parsing
    - Path: rmapi/api/auth.go
      Note: Auth bootstrap; previously hard-exited on user token refresh failure
    - Path: rmapi/config/config.go
      Note: Token cache path selection + read/write
    - Path: rmapi/config/url.go
      Note: Auth/sync/document endpoint construction
    - Path: rmapi/main.go
      Note: rmapi binary auth retry loop + backoff
    - Path: rmapi/transport/transport.go
      Note: Authorization header + HTTP status error handling
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-04T10:05:02.399359848-05:00
WhatFor: ""
WhenToUse: ""
---


# OAuth refresh failure analysis + fix plan

## Executive Summary

### Who this is for

This document is written for a new intern joining the project who has never seen `remarquee`, `rmapi`, or the reMarkable cloud APIs before.

It explains:

- What the tools are and how they fit together
- What “OAuth refresh” means in this codebase (device token → user token)
- Why `remarquee cloud account --reauth` could fail with `request failed with status 500`
- Why that backend failure looked like a broken CLI (hard exit)
- The fix strategy, what was changed, and how to validate/debug it

### What broke (symptom)

When running:

```bash
remarquee cloud account --reauth
```

the tool could fail with:

```
failed to create user token from device token request failed with status 500
```

### Root cause (what actually happened)

`remarquee` relies on `rmapi` for authentication bootstrapping. The failing call path is:

- `remarquee` → `rmapi` → **POST user-token endpoint using device token**
- The reMarkable auth service returned HTTP 500 for that request
- The `rmapi` code path treated this as a fatal error and **terminated the entire process**

So even though the 500 may be transient, `remarquee` could not retry and could not print actionable remediation.

### Fix (design in one paragraph)

Change the auth bootstrap API so that `rmapi` returns errors instead of calling `log.Fatal*`, and add bounded retries with backoff in both `rmapi` and `remarquee` so transient 5xx errors don’t appear as “OAuth refresh is broken”.

## Problem Statement

### What is “OAuth refresh” in this project?

Even though we say “OAuth refresh” casually, in this repo it mostly refers to the `rmapi` bootstrap sequence:

- You register a device once using a one-time code → you get a **device token**.
- You exchange that device token for a **user token** (and periodically re-fetch the user token).
- The cached tokens live on disk (by default under `~/.config/rmapi/rmapi.conf`).

The CLI flag `--reauth` means: “ignore the cached user token and fetch a new one using the cached device token”.

### Why is the current behavior unacceptable?

Because a transient backend error (HTTP 500) becomes a hard-stop:

- No retry
- No backoff (if you try again manually, you may spam the endpoint)
- Poor diagnostics (hard exit before the caller can wrap/annotate)
- In `remarquee`, it looks like “the CLI is broken” rather than “the backend returned 500; try again”

### Constraints

- We cannot fix reMarkable’s backend returning 500.
- We *can* fix our local behavior:
  - avoid hard exits
  - retry safely
  - provide actionable guidance and logging hooks

## Current-State Architecture (how auth works today)

### Components

- **`remarquee`**: A CLI toolkit (Go) that wraps multiple workflows: cloud operations, uploads, rendering, etc.
- **`rmapi`**: A Go client/CLI used by `remarquee` for token management + API context construction.

In code terms:

- `remarquee` uses `rmapi/api.AuthHttpCtx(...)` to load/create tokens and build an HTTP client context.
- `remarquee` then parses the user token and builds an API context for cloud operations.

### Token cache (where credentials are stored)

By default, rmapi resolves its config path as:

1. `$RMAPI_CONFIG` if set
2. `$HOME/.rmapi` if it exists
3. `$XDG_CONFIG_HOME/rmapi/rmapi.conf` (typically `~/.config/rmapi/rmapi.conf`)

Code: `rmapi/config/config.go:ConfigPath()` and `rmapi/config/config.go:LoadTokens()`.

Token schema:

```yaml
devicetoken: <device token>
usertoken: <user token>
```

Code: `rmapi/model/auth.go:AuthTokens`.

### Network endpoints involved

These URLs are assembled in `rmapi/config/url.go:init()`:

- Device token creation:
  - `POST https://webapp-prod.cloud.remarkable.engineering/token/json/2/device/new`
- User token creation (device token → user token):
  - `POST https://webapp-prod.cloud.remarkable.engineering/token/json/2/user/new`
- Sync APIs (used after auth):
  - `https://internal.cloud.remarkable.com/...`
- Document storage APIs:
  - `https://document-storage-production-dot-remarkable-production.appspot.com/...`

Important note for debugging:

- The user-token endpoint uses **Authorization: Bearer <device token>**.
  - See `rmapi/transport/transport.go:addAuthorization()` and `rmapi/api/auth.go:newUserToken()`.

### Call graph (happy path)

ASCII diagram of the “cloud account” command:

```
remarquee cloud account --reauth
  |
  v
remarquee/cmd/.../cloud/account.go:Run()
  |
  v
remarquee/pkg/rmcloud/auth.go:CreateApiCtx(authSettings)
  |
  v
rmapi/api/auth.go:AuthHttpCtx(reAuth=true, nonInteractive=?)
  |
  +--> rmapi/config.LoadTokens(configPath)
  |
  +--> if devicetoken missing: prompt for one-time code -> POST /device/new
  |
  +--> if usertoken missing OR reAuth: POST /user/new (Authorization: Bearer devicetoken)
  |
  v
rmapi/api.ParseToken(usertoken) -> userInfo(syncVersion, user, ...)
  |
  v
rmapi/api.CreateApiCtx(httpCtx, syncVersion)
  |
  v
remarquee prints: user=<email> sync_version=<...>
```

### Where it failed (error path)

The failure happened at the “POST /user/new” call:

```
POST /token/json/2/user/new  (Authorization: Bearer <device token>)
  -> HTTP 500
```

In the pre-fix behavior, this error was handled by a fatal exit in rmapi, which prevented callers from retrying.

## Proposed Solution

### Goals

- **Never hard-exit** the process during auth bootstrap from library-style code paths
- Add bounded retry + backoff for transient failures
- Preserve the existing user-facing behavior where possible (same commands/flags)
- Provide an intern-friendly debugging story: “here is how to capture traces safely”

### Non-goals

- We do not attempt to reverse engineer or “fix” the backend HTTP 500 itself.
- We do not redesign authentication (no new token formats, no new storage).

### Design changes

1. **Make auth bootstrap return errors, not exit.**
   - `rmapi/api.AuthHttpCtx(...)` returns `(*transport.HttpClientCtx, error)`.
   - Callers (rmapi main loop and remarquee) can now handle failures and retry.

2. **Add exponential backoff between retries.**
   - Both `rmapi/main.go` and `remarquee/pkg/rmcloud/auth.go` sleep between retries:
     - `250ms`, `500ms`, `1000ms` (for 3 attempts)

3. **Interactive recovery on 401 device token rejection.**
   - If the device token is invalid (`401 Unauthorized`), we clear cached tokens and:
     - in non-interactive mode: return a clear error (`rmapi reset` + `rmapi account`)
     - in interactive mode: prompt for a new one-time code, obtain a new device token, and retry once

### Pseudocode (intern-friendly)

Auth bootstrap (simplified):

```pseudo
function AuthHttpCtx(reAuth, nonInteractive) -> (httpCtx, err):
  tokens = LoadTokens(configPath)
  httpCtx = CreateHttpClientCtx(tokens)

  if tokens.deviceToken is empty:
    if nonInteractive: return error("missing device token")
    code = promptUserForOneTimeCode()
    tokens.deviceToken = POST /device/new with code
    SaveTokens(tokens)

  if tokens.userToken is empty OR reAuth:
    userToken = POST /user/new with Authorization: Bearer tokens.deviceToken
    if userToken request returns 401:
      clear tokens, SaveTokens(tokens)
      if nonInteractive: return error("device token rejected; reset + re-register")
      code = promptUserForOneTimeCode()
      tokens.deviceToken = POST /device/new with code
      SaveTokens(tokens)
      userToken = POST /user/new again
    if request errors: return error("failed to create user token")
    tokens.userToken = userToken
    SaveTokens(tokens)

  return httpCtx, nil
```

Retry + backoff in the caller (simplified):

```pseudo
for attempt in 0..2:
  reauth = userPassedReauth OR attempt>0
  httpCtx, err = AuthHttpCtx(reauth, nonInteractive)
  if err:
    sleep(backoff(attempt))
    continue
  userInfo = ParseToken(httpCtx.tokens.userToken)
  if err:
    sleep(backoff(attempt))
    continue
  apiCtx = CreateApiCtx(httpCtx, userInfo.syncVersion)
  if err:
    sleep(backoff(attempt))
    continue
  return success
return last error
```

## Design Decisions

### Why change rmapi instead of only changing remarquee?

Because the failure mode was inside `rmapi` in a way that callers could not recover from:

- `log.Fatal*` aborts the whole process
- no retry logic above can run after that

So the correct place to fix “don’t hard-exit” is in `rmapi` itself.

### Why add retries in both rmapi and remarquee?

There are two consumers:

- The `rmapi` binary itself (needs resilient auth)
- `remarquee` (uses rmapi packages; should also be resilient and user-friendly)

Keeping retries at the consumer layer also helps when additional steps (token parse, API ctx creation) fail for transient reasons.

### Why exponential backoff?

To reduce load on the auth service and to improve chances of success for transient 5xx errors without requiring the user to re-run commands manually.

## Alternatives Considered

1. **Do nothing; tell users to re-run the command.**
   - Rejected: leaves a “broken tool” experience and makes debugging hard.

2. **Retry only inside rmapi/api.AuthHttpCtx.**
   - Not chosen for now: retries already exist at the outer layers; adding too many nested retries can multiply request volume in surprising ways.

3. **Extend rmapi transport to return typed errors with HTTP status + body.**
   - Worth doing later. It’s useful for debugging, but it’s a larger change with higher chance of leaking sensitive data into errors/logs.

## Implementation Plan

### Step 1: Fix the fatal-exit behavior

Files:

- `rmapi/api/auth.go`
  - Make `AuthHttpCtx` return `error`
  - Remove `log.Fatal*` from the “create user token” path
  - Ensure 401 invalid device token is recoverable and saved back to config

### Step 2: Update callers

Files:

- `rmapi/main.go`
  - Handle `AuthHttpCtx` returning an error
- `remarquee/pkg/rmcloud/auth.go`
  - Handle `AuthHttpCtx` returning an error

### Step 3: Add retry/backoff

Files:

- `rmapi/main.go`
- `remarquee/pkg/rmcloud/auth.go`

### Step 4: Validate behavior locally

Suggested commands:

```bash
(cd rmapi && go test ./...)
(cd remarquee && go test ./cmd/remarquee/... ./pkg/...)

# Basic auth validation (non-destructive):
remarquee cloud account
remarquee cloud account --reauth
```

If you need HTTP traces:

```bash
RMAPI_TRACE=1 remarquee cloud account --reauth --non-interactive > /tmp/rmapi-trace.txt 2>&1
```

Important: trace output includes credentials unless redacted.

### Step 5: Documentation + delivery

- Update the ticket diary with exact commands and errors encountered
- Upload the bundle to reMarkable via `remarquee upload bundle`

## Open Questions

1. Is the HTTP 500 reproducible on demand, or was it transient?
2. Should we implement a typed HTTP error in `rmapi/transport` that includes:
   - status code
   - request id / trace headers
   - safe truncated body
   with redaction guardrails?
3. Should `remarquee` surface a friendlier message specifically for 5xx during `/token/json/2/user/new`?

## References (files to read first)

- `remarquee/cmd/remarquee/cmds/cloud/account.go` (command entrypoint)
- `remarquee/pkg/rmcloud/auth.go` (`CreateApiCtx`, retry/backoff)
- `rmapi/api/auth.go` (`AuthHttpCtx`, token bootstrap)
- `rmapi/config/url.go` (endpoint construction)
- `rmapi/config/config.go` (token cache path and file format)
- `rmapi/transport/transport.go` (Authorization header selection; status handling)

## Problem Statement

<!-- Describe the problem this design addresses -->

## Proposed Solution

<!-- Describe the proposed solution in detail -->

## Design Decisions

<!-- Document key design decisions and rationale -->

## Alternatives Considered

<!-- List alternative approaches that were considered and why they were rejected -->

## Implementation Plan

<!-- Outline the steps to implement this design -->

## Open Questions

<!-- List any unresolved questions or concerns -->

## References

<!-- Link to related documents, RFCs, or external resources -->
