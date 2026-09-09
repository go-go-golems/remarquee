---
Title: Tree synchronization progress and cancellation
Ticket: RMQ-HANG-001
Status: active
Topics:
    - remarquee
    - cloud
    - rmcloud
    - cli
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/remarquee/main.go
      Note: Signal cancellation and forced-exit fallback
    - Path: repo://pkg/rmcloud/auth.go
      Note: Bootstrap instrumentation boundary
    - Path: repo://pkg/rmcloud/logtransport.go
      Note: Streaming and request context lifetime
ExternalSources: []
Summary: Ship honest phase and HTTP-activity progress now; reserve exact document totals for an rmapi progress API.
LastUpdated: 2026-09-09T14:00:00-04:00
WhatFor: Specify the first implementation increment and its boundaries.
WhenToUse: When implementing or reviewing cloud tree progress and cancellation.
---


# Tree synchronization progress and cancellation

## User experience

Tree synchronization must not look like a hung process, especially with a cold cache. Progress is normal CLI feedback, independent of `--log-level debug`, and goes exclusively to stderr. Example first increment:

```text
No cached tree found; initial sync may take a while.
Synchronizing cloud tree… 0s | 0 HTTP responses, 0 active requests
Synchronizing cloud tree… 5s | 42 HTTP responses, 3 active requests
Cloud tree synchronized. 8s | 64 HTTP responses
```

An existing cache only justifies “Cached tree found; checking for remote changes.” It does NOT establish cache validity. A cache stat failure is reported without suggesting destructive reset. During stalled headers or bodies, elapsed-time updates continue even without new traffic. No ETA or document percentage is invented.

## Dependency boundary and staged delivery

The pinned rmapi fork exposes `api.CreateApiCtx` and an HTTP client, but its `sync15.CreateCtx` owns unexported cache-load/save functions and invokes `HashTree.Mirror` without callbacks. Exact completed/total *documents* and cache-validation reasons require an upstream/fork API change. Parsing dependency log strings, unsafe reflection for progress, copying sync15, or committing a machine-local module replacement are not acceptable implementations.

**This increment implements phase/elapsed/HTTP activity progress using the exposed HTTP client.** It does not count requests as documents. It does not claim to observe cache validation, local cache load/save subphases, or metadata worker completion. The existing exact-document progress task remains open pending a publishable rmapi callback API.

Future callback contract: immutable events for cache result, root discovery, known total of changed/new documents, successful document completion, cache save, and terminal outcome. The producer owns counts; the CLI owns presentation. Emit updates safely from concurrent workers, count only successfully mirrored documents, and distinguish cached/unchanged documents from network work. Callback delivery must not serialize network work on terminal writes.

## Implementation architecture

### Per-operation reporter

- Add optional `Progress io.Writer` to `rmcloud.AuthSettings`; nil keeps library calls silent. CLI entry points pass stderr (Cobra commands use their configured error writer where available).
- Start the reporter immediately before tree initialization, after authentication. Cache preflight only stats `os.UserCacheDir()/rmapi/tree.cache`; it never reads, deletes, or validates the cache.
- Use a ticker and atomic request counters. Render every second on a TTY with carriage-return line replacement; render newline milestones every five seconds otherwise. Initial and final lines are unconditional. Stop and join the reporter before returning to avoid late writes and leaked goroutines.
- HTTP response count means headers received, not completed body or completed document. Active count includes body consumption until EOF/error/Close. This keeps stalled-body work visible without buffering it.
- Terminal outcomes: success, failure, cancellation, deadline. No success message on a failed attempt. Every retry starts a distinct reporter; debug events identify its attempt.

### Transport and debug logging

Install instrumentation on `httpCtx.Client.Transport` BEFORE `api.CreateApiCtx`. Replace the existing eager `io.ReadAll` logging wrapper with metadata-only logging and a body wrapper that delegates Read/Close. Do not log URLs, headers, bodies, document names, tokens, or raw transport error strings (which can contain signed URLs). Log method, outcome category, response status, and elapsed time. Do not enable rmapi trace.

Bind each request to both its original request context and the command context. Release the command cancellation callback on request failure, body EOF/error, or Close, not on receipt of headers. Check command cancellation before entering the base transport. Leave the request's existing deadline intact.

Keep that context-bound transport for later API work too, but freeze/detach sync activity counters when initialization ends. No process-global transport or log mutation.

### Cancellation and retries

Change `CreateApiCtx` and `WithAuthRetry` to accept context explicitly; update all callers rather than add compatibility overloads. Check cancellation before auth, after auth, before retry, and after tree initialization (rmapi sometimes formats errors with `%v`, losing error identity). Do not retry cancellation/deadline errors.

At the CLI root, deliver SIGINT to a cancellable command context. Restore termination semantics with an explicit second-interrupt / two-second forced-exit fallback, because dependency authentication prompting and some local work remain uncancellable. Stop signal handling and timers after ordinary completion. A cancellation exits nonzero (130 for SIGINT) without printing a usage screen.

**Boundary:** this does not make rmapi's token bootstrap or CPU/local-file work context-aware, add a total-operation deadline, or diagnose why the user's installed binary ignored Ctrl-C. Existing five-minute per-request timeout remains. Account-only authentication separation and full dependency context propagation remain ticket follow-ups.

## Validation

- Reporter: cold/existing/inaccessible cache preflight, periodic heartbeat with unchanged counts, TTY vs non-TTY formatting, success/error/cancel, deterministic ticker tests, stop/join behavior.
- Transport: no eager reads at any log level, original Close ownership and read errors, concurrent counters, stalled headers/body cancellation through a real local HTTP server, original request cancellation, no secret-bearing log fields.
- Sync integration: invoke actual rmapi `CreateApiCtx` using isolated HOME/cache and a fake HTTP transport/server, never production credentials. Cover cold/warm/corrupt cache, empty/changed root, failed request, and canceled initialization with no retry.
- CLI signal handling: subprocess tests sending actual SIGINT for cooperative cancellation, uncooperative timeout fallback, and second interrupt. No real cloud commands needed.
- Run focused tests, race tests, broader repository tests/build, docmgr doctor, and whitespace checks. Record failures and environmental blockers in the diary. Commit design, implementation checkpoints, and validation documentation separately.
