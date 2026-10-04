# Tasks

## Documentation

- [x] Reconnaissance of existing cloud/upload/web layers
- [x] Create ticket and vocabulary topics
- [x] Write intern-facing design & implementation guide
- [x] Write implementation diary
- [x] Apply "no classic Macintosh bitmap typeface / modern fonts only" constraint
- [x] Resolve delivery shape as `remarquee serve` subcommand (not a separate binary)
- [x] Relate key source files to the design doc
- [x] Upload the design guide to reMarkable

## Implementation — DONE

### Phase 0 — Spike the service
- [x] Add `pkg/rmfiles/entries.go` with `Entry` + path building
- [x] Dump tree and compare with `remarquee cloud ls` (live: 13661 docs)

### Phase 1 — Service + read endpoints
- [x] Implement `pkg/rmfiles.Service` (single long-lived `ApiCtx`, mutex, reindex)
- [x] `GET /api/status`, `/api/files`, `/api/files/tree`, `/api/search`
- [x] Unit tests with a fake `ApiCtx`

### Phase 2 — Frontend shell + macOS-1 primitives
- [x] Static frontend at `cmd/remarquee/cmds/serve/frontend` (no build step)
- [x] `styles.css` tokens, file list, breadcrumbs, search, detail pane, status banner

### Phase 3 — Upload jobs
- [x] `POST /api/uploads` multipart + job store + bounded workers
- [x] `.md` → PDF conversion, `.pdf`/`.epub` direct; job polling UI
- [x] Verified live: create folder, md upload job `done`, recursive delete

### Phase 4 — Manage operations
- [x] Create folder, rename, move, delete (typed confirm), download
- [x] Guardrails: move-into-self rejection, conflict detection

### Phase 5 — Packaging + polish
- [x] `serve` command, `embed.go`, registered in `cmd/remarquee/main.go`
- [x] Graceful shutdown, `--addr` flag, `127.0.0.1` default, `--dev` cache toggle

### Phase 6 — Validation
- [x] `go test ./pkg/rmfiles/... ./cmd/remarquee/cmds/serve/...`
- [x] Live smoke test (tmux + curl) with cleanup

## Follow-ups (not done)

- [ ] Wire `rmcloud.WithAuthRetry` for mutating operations (signature requires a small service change)
- [ ] Same-origin/CSRF enforcement on mutating endpoints; optional read-only mode
- [ ] Opt-in integration test guarded by an env var
- [ ] Revisit `WriteTimeout: 0` and `Download` holding the mutex across I/O
- [x] Review PR 29 against the design, document evidence-backed findings and intern implementation guidance, and upload the review to reMarkable <!-- t:zxus -->
- [ ] PR review F01/F05: implement descendant-first cloud deletion, live-ID integrity and explicit partial outcomes; qualify against rebuilt state <!-- t:mum9 -->
- [ ] PR review F03: bound jobs and queued bytes globally, own cancellation/shutdown and evict terminal job records <!-- t:983u -->
- [ ] PR review F04/F07/F12: reject effective-name upload conflicts, preserve explicit root destination and define Markdown folder asset policy <!-- t:e2as -->
- [ ] PR review F06/F08/F09: fix response ordering and view mode, route encoding, picker reset and dialog/job lifecycle <!-- t:wx8g -->
- [x] PR review F11: stage downloads at fixed internal paths and safely format suggested attachment names <!-- t:v91p -->
- [ ] PR review: enforce monochrome danger surfaces and keyboard access; repair historical API/dev docs and add browser/failure-oriented tests <!-- t:4681 -->
- [x] Design the requested move directory browser, subtle directory text color, actual file-type column, non-wrapping dates and column sorting in a second design document <!-- t:2ro4 -->
- [ ] Second design: implement an isolated directory browser for Move with breadcrumbs, root/empty-folder selection, subtree guards and keyboard/cancel handling <!-- t:baaq -->
- [ ] Second design: expose authoritative PDF/EPUB/notebook metadata through a typed dependency projection and add a distinct file-type column <!-- t:2eyk -->
- [ ] Second design: add subtle directory-name color, non-wrapping dates and accessible deterministic column sorting with search relevance reset <!-- t:22a5 -->
