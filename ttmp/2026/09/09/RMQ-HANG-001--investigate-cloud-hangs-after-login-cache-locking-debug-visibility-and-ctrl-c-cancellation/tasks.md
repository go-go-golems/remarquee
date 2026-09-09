# Tasks

## TODO

- [ ] Confirm installed binary/version and reproduce exact hangs with bounded runs and sanitized phase/stack evidence <!-- t:hgd4 -->
- [ ] Isolate token exchange, cache load/save, and remote tree mirror; compare safe isolated warm/cold/corrupt caches <!-- t:cm18 -->
- [ ] Diagnose SIGINT delivery and dependency handlers separately from missing context cancellation <!-- t:hw8a -->
- [ ] Split authentication-only account lookup from eager document-tree initialization <!-- t:cqeo -->
- [ ] Add secret-safe bootstrap debug events, timings, retry reasons, and long-running progress on stderr <!-- t:r0h5 -->
- [ ] Propagate signal-backed context through auth, requests, mirror workers, and retries; define deadlines and bounded shutdown <!-- t:l7rd -->
- [x] Fix transport logging to preserve streaming, body Close ownership, and read errors without leaking content <!-- t:baza -->
- [ ] Add stalled-network/cache/cancellation regression tests and document troubleshooting and timeout behavior <!-- t:yt61 -->
- [ ] Expose tree-sync progress from rmapi: announce cold/invalid-cache full sync, show discovery phase before totals are known, then completed/total documents and elapsed time; emit throttled stderr updates (TTY line/non-TTY milestones), avoid document names, and keep Ctrl-C responsive. Test cold/warm sync, unknown totals, cancellation, and clean structured stdout. <!-- t:ba1l -->
- [x] Ship first increment: stderr elapsed/HTTP-activity sync progress, context-bound sync requests, SIGINT watchdog, streaming-safe logs, and focused fake-cloud/race/signal tests <!-- t:z4en -->
