---
Title: Cloud hang investigation and remediation plan
Ticket: RMQ-HANG-001
Status: active
Topics:
    - remarquee
    - cloud
    - rmcloud
    - cli
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/remarquee/cmds/cloud/account.go
      Note: Account lookup enters full API initialization and ignores context for bootstrap
    - Path: repo://cmd/remarquee/main.go
      Note: Root execution and logger setup; no signal-backed command context
    - Path: repo://go.mod
      Note: Pinned rmapi fork used for cache and mirror source inspection
    - Path: repo://pkg/rmcloud/auth.go
      Note: Shared authentication retries and late transport instrumentation
    - Path: repo://pkg/rmcloud/logtransport.go
      Note: Eager body buffering and logging privacy and ownership concerns
ExternalSources: []
Summary: Initial source evidence and investigation plan for silent cloud initialization and ineffective cancellation.
LastUpdated: 2026-09-09T14:00:00-04:00
WhatFor: Identify the blocking stage before choosing a fix.
WhenToUse: When reproducing cloud hangs or implementing diagnostics and cancellation.
---


# Cloud hang investigation and remediation plan

> Initial investigation snapshot. The first implementation increment is now in `c9f621e`; see the [design and upstream verification](../design-doc/01-tree-synchronization-progress-and-cancellation.md) and [diary](../reference/01-diary.md). Findings below describe the pre-change code; incident attribution and full dependency-level progress/cancellation remain open.

## Report and scope

User reports that pretty much every operation hangs. Specifically, `remarquee cloud account` prompted for a one-time code, accepted input, then stopped making visible progress. Ctrl-C also appeared ineffective. Cache involvement is a hypothesis, not an established cause.

This ticket covers reproduction, authentication and tree-cache initialization, safe debug logging, bounded waits, and interrupt handling across cloud operations. This initial pass is source inspection only: no live login, cache deletion, credential inspection, or reproduction was performed. No runtime fix has been made.

## Initial evidence

1. **Account info performs full cloud initialization.** `cmd/remarquee/cmds/cloud/account.go:Run` accepts a context but calls the context-free `createApiCtx`. That delegates to `pkg/rmcloud/auth.go:CreateApiCtx`, which calls `api.AuthHttpCtx`, parses the token, then calls `api.CreateApiCtx`. Account info does not need a document tree, so authentication-only initialization is a promising separation.
2. **The important startup work is outside existing HTTP logging.** `WrapTransportWithLogging` runs only after `api.CreateApiCtx` returns. Token exchange and initial tree synchronization therefore occur before the wrapper is installed. There are no local start/end/duration events around those calls or the three outer attempts.
3. **Cache initialization includes remote synchronization.** The pinned rmapi fork's `api/sync15/apictx.go:CreateCtx` calls `loadTree`, `cacheTree.Mirror`, `saveTree`, and `DocumentsFileTree`. Its `common.go` stores `tree.cache` under `os.UserCacheDir()/rmapi`. Missing, corrupt, or wrong-version cache can lead to a resync; deleting it can make startup slower. The searched cache code does not establish a lock deadlock.
4. **Network waits can be long.** The fork's `transport/transport.go:CreateHttpClientCtx` sets a five-minute HTTP client timeout. This is not a total command deadline: multiple requests, synchronization work, and up to three local initialization attempts can accumulate. Determine the actual blocking request before shortening arbitrary timeouts.
5. **Command cancellation is disconnected.** `cmd/remarquee/main.go` calls `rootCmd.Execute()`, not a signal-backed `ExecuteContext`. `CreateApiCtx` has no context parameter. The fork's `HashTree.Mirror` uses `errgroup.WithContext(context.TODO())`; worker calls do not receive the command context. Cancellation needs to reach actual I/O and workers, not merely return from a wrapper goroutine.
6. **This does not yet explain ignored SIGINT.** Default Go SIGINT behavior normally terminates a process even without context plumbing. Source searches found no signal registration in remarquee or the pinned rmapi fork. Investigate other dependencies, the launched executable/version, shell or wrapper, terminal foreground process group, and signal delivery. Distinguish graceful cancellation from failure to terminate at all.
7. **The existing transport wrapper itself needs repair.** `pkg/rmcloud/logtransport.go:RoundTrip` unconditionally reads the entire response body before returning, even when debug is disabled. The 4096-byte cap limits only the logged preview, not the read. It replaces the original body without closing it and does not propagate body-read errors. A stalled body delays the response log; large bodies allocate unnecessarily. This wrapper is installed after initialization, so it cannot alone explain a first-login stall inside initialization.
8. **Do not enable raw dependency trace indiscriminately.** The fork's `api/auth.go` logs device and user tokens at trace level and uses fatal exits for several auth failures. The local wrapper logs full URLs, non-Authorization headers, and response previews. Bootstrap logging must not expose one-time codes, tokens, signed URLs, cookies, or document contents.

Dependency examined from the local Go module cache: `github.com/FNStudios-NI/rmapi@v0.0.0-20260817154736-f295d5466978`, selected by the `go.mod` replacement. These findings must be checked against the user's installed executable before attributing its behavior to this checkout.

## Reproduction plan

1. Record binary path, build/module revision, OS, launch method, exact commands, elapsed time, and whether all operations includes local/help commands. Do not record credentials.
2. Reproduce in a controlled terminal/tmux session with explicit time bounds. Compare help/local commands, cached non-interactive account lookup, cloud listing, and fresh registration only when necessary and authorized. Avoid routine `--reauth` or `rmapi reset` as diagnostic defaults.
3. Add safe phase events before auth configuration load, code input, device exchange, user exchange, token parse, cache load, remote root lookup, tree mirror, cache save, and tree construction. Include attempt, elapsed time, completion/error, and cancellation. Keep diagnostics on stderr.
4. Capture the last completed phase and a private goroutine/process sample while stuck. A SIGQUIT dump can terminate the process and expose sensitive data; use only on a controlled reproduction, sanitize before storing ticket artifacts.
5. Check cache metadata and source-resolved location without dumping cache contents. Compare backed-up/isolated warm, empty, corrupt, and stale caches with a mocked cloud. Avoid deleting the live cache; a cold cache increases remote work.
6. Test terminal Ctrl-C and directly delivered SIGINT separately. Identify the foreground child process and any dependency signal hooks. Exercise code-input wait, token exchange, headers wait, body stall, mirror workers, and retry delay.

## Remediation direction

- Split authentication/account introspection from eager tree synchronization so `cloud account` does not need the entire library.
- Thread a signal-backed command context through authentication, HTTP requests, sync workers, and retries. Prefer fixes in the pinned fork where APIs cannot accept contexts. Cancellation/deadline errors must not trigger reauthentication or more retries.
- Define documented operation/request timeouts and bounded shutdown; first Ctrl-C cancels, with a reliable hard-exit fallback for uncooperative work. Avoid swallowing SIGINT while uncancellable dependency calls keep running.
- Install safe instrumentation before the first network call. Log start/end and elapsed time, retry reason/count, cache outcome, and sync progress. For genuinely long work, provide periodic concise stderr progress rather than silent waits; preserve machine-readable stdout.
- Replace eager body buffering with streaming-safe metadata logging. Debug-disabled operation must not consume bodies; preserve Close and read-error behavior. Prefer no body content in cloud logs and sanitize URL query values and headers.
- Improve auth errors rather than exiting inside library code. Explain whether a timeout occurred during token exchange, tree sync, or cache access. Never recommend credential reset solely because synchronization is slow.

## Acceptance criteria

- A reproduced hang has an identified blocking stage backed by sanitized logs or stack evidence; cache conclusions distinguish local I/O from remote resync.
- `cloud account` completes after authentication without loading/mirroring the document tree.
- `--log-level debug` shows bootstrap stages, durations, retry reasons, and final errors before the first potentially blocking cloud call; secrets and document data are absent.
- Long work has progress visibility, an explicit timeout policy, and prompt cancellation. Controlled stalled-I/O tests exit within two seconds of SIGINT with nonzero status, no retry after cancellation, and no lingering child/workers.
- Tests cover slow headers, stalled bodies, auth failure, corrupt/missing cache, mirror worker failure, cancellation during prompting and initialization, and response-body ownership/read errors.
- Normal non-debug execution retains streaming behavior; structured command stdout remains clean.

## Open questions

- Which installed build hung, and was it launched directly, via `go run`, or through another wrapper?
- Is time spent in token exchange, cache loading, remote mirror, cache writing, or later work?
- Is Ctrl-C not delivered, intercepted by a dependency, or delivered only to a wrapper?
- How large is the document tree, and did the cache become cold or invalid after a version change?
- What fork changes are required to support cancellation throughout bootstrap and synchronization?
