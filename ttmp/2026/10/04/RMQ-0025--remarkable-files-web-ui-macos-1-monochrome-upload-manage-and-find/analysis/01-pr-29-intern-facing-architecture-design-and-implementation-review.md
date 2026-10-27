---
Title: PR 29 intern-facing architecture design and implementation review
Ticket: RMQ-0025
Status: complete
Topics:
    - remarkable
    - cloud
    - upload
    - web
    - frontend
    - ui
    - go
    - remarquee
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/remarquee/cmds/serve/frontend/app.js
      Note: Request ordering routing and upload lifecycle findings
    - Path: repo://cmd/remarquee/cmds/serve/handlers.go
      Note: Actual HTTP contracts and browser input boundary
    - Path: repo://pkg/rmfiles/manage.go
      Note: Recursive deletion and safe download staging reviewed in F01 F05 and F11
    - Path: repo://pkg/rmfiles/service.go
      Note: Service ownership and mutation consistency reviewed in F03 F05 and F10
    - Path: repo://pkg/rmfiles/upload.go
      Note: Global admission conflicts destination and asset semantics reviewed in F03 F04 F07 F12
    - Path: repo://ttmp/2026/10/04/RMQ-0025--remarkable-files-web-ui-macos-1-monochrome-upload-manage-and-find/scripts/01-review-backend.go
      Note: Offline service mux and pinned dependency reproductions
    - Path: repo://ttmp/2026/10/04/RMQ-0025--remarkable-files-web-ui-macos-1-monochrome-upload-manage-and-find/scripts/02-review-frontend.cjs
      Note: Deterministic frontend state reproductions with stubbed DOM
ExternalSources:
    - https://github.com/go-go-golems/remarquee/pull/29
Summary: Evidence-backed walkthrough and implementation review of remarquee serve, including cloud deletion semantics, browser security, upload scheduling, frontend state, actual APIs, and an ordered remediation plan.
LastUpdated: 2026-10-04T08:00:00-04:00
WhatFor: Teach a new intern how PR 29 works and how to address its correctness and safety gaps without replacing its architecture.
WhenToUse: Before merging PR 29, implementing review fixes, or onboarding onto the files service and embedded browser UI.
---

# PR 29: architecture, design, and implementation review

## 1. Executive assessment

`remarquee serve` turns an existing command-line tool into a local browser application for browsing, searching, uploading, renaming, moving, downloading, and deleting reMarkable cloud documents. The browser does not hold cloud credentials. It sends requests to a Go process on the laptop; that process uses the existing cloud library on the user's behalf. A Markdown upload is converted into a PDF locally before the PDF is sent to the cloud. The tablet subsequently synchronizes through its own connection: successful upload is not proof of tablet synchronization.

The implementation has a good core decomposition. `pkg/rmfiles` owns the cloud context and exposes domain operations; `cmd/remarquee/cmds/serve` translates HTTP into those operations; three embedded frontend files provide the interaction layer. A single service mutex protects the stateful cloud library, while expensive Markdown conversion occurs outside that mutex. This is a sensible initial architecture for one user and a relatively small desktop UI. I would keep it rather than replace it with a framework migration.

**Recommendation: request changes before merge.** The green tests demonstrate useful happy paths, but do not establish destructive-operation correctness or the advertised safety boundary. In particular, the pinned cloud dependency does not recursively remove descendants when given `recursive=true`; the service currently hides a deleted subtree locally while leaving descendant cloud records behind. The API also accepts foreign-origin mutation requests, and its listener can be exposed outside loopback despite the PR's “loopback-only” description. These are not cosmetic follow-ups.

This review is a **documentation and investigation deliverable**, not an implementation of the fixes. Production Go, JavaScript, HTML, and CSS are unchanged. Small offline probe programs and their output are stored alongside this review to make the findings reproducible.

### 1.1 Snapshot and evidence boundaries

- PR: <https://github.com/go-go-golems/remarquee/pull/29>.
- Reviewed head: `f2dd68ce4a19467ace9891eeafdddfb1443b2b70`.
- Reviewed base: `02475f61b2cc08b3f7b812e1f620dbb20adb016e`.
- Branch: `task/rmq-0025-serve-files-ui`; fork owner: `manuel-tulip`.
- Design under review: `design-doc/01-remarkable-files-web-ui-design-and-implementation-guide-for-a-new-intern.md` in this ticket.
- Dependency: `github.com/juruen/rmapi`, replaced by `github.com/FNStudios-NI/rmapi` at `v0.0.0-20260817154736-f295d5466978` (`go.mod:23`).

Repository line references below refer to the reviewed head. Dependency paths use `rmapi/...` as shorthand for the pinned replacement module, **not** a directory in this repository. Stable code links can be constructed as `https://github.com/go-go-golems/remarquee/blob/f2dd68ce4a19467ace9891eeafdddfb1443b2b70/<path>#L<line>`.

There are three evidence levels in this report:

1. **Observed offline:** exercised the actual service, mux, dependency data structures, or frontend functions against local fakes.
2. **Source-confirmed:** traced a concrete implementation path without executing the corresponding real cloud or browser operation.
3. **Proposed:** recommended API, algorithm, or acceptance test; not implemented.

No review experiment deleted or uploaded user cloud files. Prior diary smoke tests are historical evidence, not newly repeated verification. The requested delivery of this report is the only new cloud upload in this review workflow. Browser-native drag behavior, actual CSS rendering, hostile-page exploitation, token expiry, and real cloud failure recovery were not exercised by the offline probes.

### 1.2 What to preserve

- The existing `remarquee` binary and Cobra subcommand: no second application binary to distribute.
- The service/HTTP/frontend dependency direction.
- One long-lived cloud context per server, with serialized tree and cloud access.
- Stable document IDs in mutation URLs, not mutable display paths.
- Conversion outside the service mutex.
- JSON snapshots rather than exposing mutable cloud nodes to handlers.
- Embedded assets without a mandatory Node build step.
- `textContent` for cloud names and errors, avoiding HTML insertion.
- Typed deletion confirmation and move-into-own-subtree rejection.
- Synchronous capture of dropped root entries before asynchronous traversal, and repeated directory-reader batches.

## 2. Orientation for a new intern

### 2.1 The three different things called a “file”

A local input file, a remote cloud document, and a browser list entry are different objects. Keeping them distinct prevents many mistakes.

**Local input:** a browser `File`, whose bytes become multipart data. For example, `Reading/Notes/Plan.md` has a basename `Plan.md`, a relative path `Reading/Notes/Plan.md`, and Markdown bytes. The browser cannot give the server a useful local filesystem path on the user's machine.

**Cloud document:** an object with a stable ID and metadata such as name, parent ID, type, and modification time. The cloud library also maintains blob hashes and a root index used by the synchronization protocol. A cloud folder is a document of type `CollectionType`; a PDF-backed document is `DocumentType`.

**API entry:** an immutable-looking Go value copied from a cloud node. Its `path` is synthesized from parent pointers. Renaming a folder changes the paths of all its descendants, but their IDs remain the same. The UI should use paths for navigation and IDs for actions.

Example:

```text
Local selection                 Cloud metadata             API presentation
Reading/Notes/Plan.md   ->       ID = document-uuid         name = Plan
                                Parent = notes-uuid        path = /ai/Reading/Notes/Plan
                                Type = DocumentType        isDir = false
                                content = converted PDF
```

The extension is generally absent from the cloud display name because rmapi derives that name from the staged local file stem. Spaces and non-ASCII characters can also change during the current sanitization step; this becomes important in finding F04.

### 2.2 Vocabulary needed for the rest of the review

- **`ApiCtx`:** rmapi's stateful API interface. It owns access to the cloud's hash tree and the mutable local file tree. It is not a stateless HTTP client.
- **File tree:** `filetree.FileTreeCtx`, with parent/child node pointers and an ID lookup map. This is the service's local representation of the account.
- **Hash tree:** Sync15's representation of cloud records and blob hashes. It is separate from the UI file tree; updating one does not automatically repair the other.
- **Search index:** a flat `[]Entry` snapshot derived from the local file tree. It is rebuilt at startup, after successful mutations, and after explicit refresh.
- **Mutation:** create, upload, rename, move, or delete. These may perform network writes as well as local tree updates.
- **Job:** an in-memory record tracking asynchronous conversion and upload of one or more inputs. `202 Accepted` means the job was admitted, not that the files were uploaded.
- **Origin:** the browser's scheme/host/port identity. Listening on localhost does not by itself establish that an HTTP request came from the UI.
- **Cancellation ownership:** the component responsible for stopping work and releasing resources. A request context is too short-lived for an accepted background job; an untracked background context is too detached for graceful shutdown.

### 2.3 System and ownership diagram

```text
User's browser                         Laptop Go process
+-------------------------+             +------------------------------+
| index.html / styles.css |  GET /      | embedded asset handler       |
| app.js                  |<---------->| cmd/.../serve/server.go      |
|                         |             +------------------------------+
| directory/search state  |  /api/*     | JSON + multipart handlers    |
| staged File + rel path  |<---------->| cmd/.../serve/handlers.go    |
| job polling             |             +---------------+--------------+
+-------------------------+                             |
                                              domain calls + snapshots
                                                        v
                                        +------------------------------+
                                        | pkg/rmfiles.Service          |
                                        | cfg, one ApiCtx, mutex       |
                                        | flat Entry index, job store  |
                                        +------+-----------------------+
                                               |
                           +-------------------+--------------------+
                           |                                        |
                    conversion outside lock                 cloud access under lock
                           v                                        v
                 pkg/mdpdf -> pandoc/XeLaTeX       rmapi ApiCtx + mutable file tree
                           |                                        |
                     temporary PDF                         Sync15 HTTPS transport
                                                                    |
                                                                    v
                                                           reMarkable cloud
                                                                    |
                                                        tablet's independent sync
```

There are two synchronization mechanisms here, and they solve different problems. The **Go mutex** prevents concurrent in-process access to shared state. The **cloud Sync15 protocol** publishes changes and resolves remote generations. A mutex is not a cloud transaction, and it cannot make a multi-document delete atomic.

## 3. Source map and reading order

Read the small application from the outside inward, then inspect the dependency behavior before touching destructive operations.

| File / starting symbol | Responsibility | Review focus |
| --- | --- | --- |
| `cmd/remarquee/main.go:41` | Registers `serve` beside existing commands | One binary; inherited interrupt policy |
| `cmd/remarquee/cmds/serve/command.go:27,60` | Flags, service initialization, HTTP lifecycle | Listener validation and shutdown |
| `cmd/remarquee/cmds/serve/server.go:34` | Method-aware routes and static fallback | Security wrapper belongs above mux |
| `cmd/remarquee/cmds/serve/embed.go` | `go:embed frontend` and `fs.Sub` | Assets fixed at compile time |
| `cmd/remarquee/cmds/serve/handlers.go:26,69,79` | Response/error/JSON helpers | Input boundary and media types |
| `pkg/rmfiles/service.go:41,53,182` | Service ownership and mutation wrapper | Lock discipline, recovery, consistency |
| `pkg/rmfiles/entries.go:16,31,72` | Entry projection and traversal | Identity versus display path |
| `pkg/rmfiles/search.go:17,56,94` | Ranking and subsequence matching | Byte-based scoring and result limits |
| `pkg/rmfiles/manage.go:14,40,70,99,123` | Create, rename, move, delete, download | Dependency semantics and path hygiene |
| `pkg/rmfiles/upload.go:153,171,196` | Job admission, workers, upload pipeline | Bounds, collision policy, default destination |
| `frontend/app.js:93,109,282` under `serve/` | List/search requests and post-mutation refresh | Request ordering and view mode |
| `frontend/app.js:336,378,433,468` | Staging, submit, drop traversal, polling | Lifecycle and partial failure |
| `frontend/index.html` / `styles.css` | DOM contract and visual tokens | Accessibility and color contract |
| `pkg/rmcloud/auth.go:67,120,172` | Auth initialization, transport, auth retry | Existing mechanisms, not new auth code |
| `pkg/rmcloud/dirs.go:30` | Ensure remote folder chain | Local tree updates after cloud creates |
| `pkg/mdpdf/pandoc.go:136,244` | Markdown preprocessing and subprocess | Assets, cancellation, trust boundary |
| `pkg/rmfiles/service_test.go` / `serve/server_test.go` | Unit and handler happy paths | Missing destructive and adversarial cases |

The existing `cmd/remarquee-ui` is a separate document-rendering application. This PR does **not** depend on it at runtime or use its frontend as a base. Its historical mention in the design is background, not a prerequisite for working on this feature.

## 4. Startup, service ownership, and consistency

### 4.1 Startup and flags

`runServe` authenticates before starting HTTP. It creates a service with non-interactive authentication, builds the mux, and calls `ListenAndServe`. Missing credentials therefore cause command startup failure; there is no running web authentication page in this version.

Actual flags (`command.go:50-55`):

- `--addr`: default `127.0.0.1:8080`; currently an unrestricted string passed to `http.Server`.
- `--dev`: disables static caching; **does not** read changed files from disk or start Vite.
- `--remote-dir`: default `/`; service default for uploads, with the root-selection defect described below.
- `--include-templates`: exposes template entries normally omitted from projections.
- `--workers`: default 2; concurrent conversion/upload pipelines **per job**, not globally.

HTTP timeouts are 10 seconds for headers, 120 seconds for reads, no write timeout, and 120 seconds idle. Requests can spend time waiting for the service mutex as well as doing network I/O. The absence of a write timeout is a conscious trade-off for large responses but is not a general solution for lifecycle control.

### 4.2 Why one `ApiCtx` is the right starting point

`rmcloud.CreateApiCtx` loads authentication state, initializes the tree, wraps its transport with command-context cancellation, and applies the schema-v4 workaround. Creating a context per request would repeatedly mirror/save shared cache state and make independent requests operate on competing trees. The service instead creates one context and holds it for the process lifetime.

All tree-facing operations hold `s.mu`. `Search` copies the flat index under the lock and scores outside it. Conversion also occurs outside the lock. However, upload, refresh, and download network work hold the same lock, so slow cloud I/O blocks directory reads and status calls. This is correct serialization but has a user-visible responsiveness cost. Do not “fix” that cost by concurrently calling rmapi methods unless its ownership model has first been changed and verified.

### 4.3 There are three versions of account state

```text
Cloud root index / blob metadata
              |
       rmapi cloud methods
              v
ApiCtx hash tree  <---->  local FileTreeCtx
                              |
                      reindexLocked()
                              v
                         []Entry index
                              |
                          Search()
```

Successful `mutate` does:

```text
lock service
run callback(apiCtx)     // cloud operation + local tree updates
call SyncComplete()
rebuild Entry index
unlock service
```

This looks transaction-like but is not transactional. The pinned Sync15 methods already call `Sync(...)` to publish root-index changes and save tree state. `SyncComplete()` is a notification step, not the commit of all accumulated changes. In the pinned dependency, it even logs most notification errors and returns `nil` (`rmapi/api/sync15/apictx.go:495-509`). Do not teach an intern that a failed callback automatically rolls back cloud changes or that `SyncComplete` proves tablet delivery.

The service currently returns immediately on callback error. If the callback changed the tree before failing, it skips reindexing and leaves list/search views disagreeing. Finding F05 gives a direct example.

### 4.4 Scope of the mutex

The mutex only coordinates callers within this service instance. Another `remarquee` process, the vendor application, or the tablet can change the cloud concurrently. The separate process may also use the same rmapi cache. This feature should document its assumptions about concurrent local tools; it does not introduce a cross-process cache lock.

Manual Refresh rebuilds the tree and index. There is no background refresh interval, no event stream, and no synchronization between open browser tabs' visible state. Tabs share the correct backend instance but must fetch new snapshots to see each other's changes.

## 5. The actual API, not the historical proposal

### 5.1 Read and capability endpoints

| Method and URL | Actual success body | Source |
| --- | --- | --- |
| `GET /api/health` | `200 {"status":"ok"}` | `handlers.go:94` |
| `GET /api/status` | `200 Status` | `handlers.go:98`, `service.go:32` |
| `GET /api/files?dir=/ai` | `200 {"path":"/ai","entries":[...]}` | `handlers.go:102` |
| `GET /api/files/tree` | `200 {"entries":[...]}` | `handlers.go:115` |
| `GET /api/search?q=plan&limit=100` | `200 {"query":"plan","results":[{"entry":...,"score":...}]}` | `handlers.go:119` |
| `GET /api/entries/{id}/download` | `200` binary `.rmdoc`, attachment header | `handlers.go:212` |
| `GET /api/uploads/{id}` | `200 UploadJob`, or 404 | `handlers.go:275` |

`Status` contains `authenticated`, `documentCount`, `lastRefresh`, `defaultRemoteDir`, and `pandocAvailable`. Here `authenticated` means the service has a non-nil context, not that its token was just validated. `documentCount` includes visible folders and documents, not documents alone. `lastRefresh` is updated by every reindex, including local mutations. `pandocAvailable` checks only the pandoc executable, not XeLaTeX, fonts, or required TeX packages.

`Entry` fields are `id`, `parentId`, `name`, `path`, `isDir`, `type`, `version`, `modifiedClient`, and `modifiedTime`. `modifiedTime` is a Go `time.Time` serialized as a timestamp. Invalid source timestamps leave a zero time; the browser currently treats that as a valid date rather than a missing value.

### 5.2 Mutation endpoints

| Method and URL | Request | Actual success | Source |
| --- | --- | --- | --- |
| `POST /api/folders` | `{ "parentPath":"/ai", "name":"Notes" }` | `201 Entry` | `handlers.go:130` |
| `PATCH /api/entries/{id}` | `{ "name":"New name" }` | `200 Entry` | `handlers.go:147` |
| `POST /api/entries/{id}/move` | `{ "destDir":"/Archive" }` | `200 Entry` | `handlers.go:164` |
| `DELETE /api/entries` | `{ "ids":["uuid"], "confirm":"DELETE", "recursive":true }` | `200 {"deleted":1}` | `handlers.go:181` |
| `POST /api/uploads` | Multipart: `destDir`, repeated `files`, optional aligned `paths` | `202 {"jobId":"..."}` | `handlers.go:227` |
| `POST /api/refresh` | No body required | `200 Status` after synchronous refresh | `handlers.go:285` |

The API uses **parent and destination paths**, not `parentId` or `destDirId`. Unknown JSON fields are rejected. The design's proposed cancellation endpoint is **not registered**. There is no overwrite, content-only replacement, job cancellation, or read-only endpoint/flag.

The delete handler's comment says recursion defaults to true, but for a nonempty ID list omitted/false `recursive` results in false. The UI explicitly sends true. This is a documentation/comment mismatch, not evidence that the handler silently enables recursive deletion.

Example upload:

```text
POST /api/uploads
Content-Type: multipart/form-data; boundary=...

field destDir = /ai
file  files   = Plan.md         (Markdown bytes)
field paths   = Reading/Notes/Plan.md
file  files   = Appendix.pdf    (PDF bytes)
field paths   = Reading/Appendix.pdf

202 {"jobId":"upload-..."}
```

`paths[i]` belongs to `files[i]`; interleaving the parts is convenient, but correspondence is determined by order within each field's collected values. Missing paths fall back to the multipart filename. Mismatched nonzero array lengths are not rejected, so malformed clients can mix relative paths and basenames. Use explicit validation for an unambiguous contract.

### 5.3 Errors and boundary behavior

Errors have shape:

```json
{"error":{"code":"not_found","message":"entry not found"}}
```

Mappings: not found 404; conflict 409; invalid/not-a-directory 400; unsupported 415; unclassified errors 500 with a generic message and a server-side log. Missing typed delete confirmation is 400 `confirmation_required`.

Limitations of this boundary:

- JSON is decoded from a 1 MiB `LimitReader`, but trailing values are not checked.
- JSON endpoints do not enforce `Content-Type: application/json`.
- Multipart total body is limited to 64 MiB **including multipart overhead**, with an 8 MiB parser memory threshold.
- Oversized multipart errors currently become 400, not a dedicated 413.
- Cloud auth errors are unclassified 500s; they do not become a documented 401/auth-required state.
- Nonempty-directory errors from rmapi are plain errors and also become 500.
- The list response echoes the supplied path rather than a canonical path synthesized from the resolved node.

The API is small enough that these behaviors can be made explicit with focused helper functions and tests. A new abstraction layer is not necessary.

## 6. End-to-end upload and search walkthroughs

### 6.1 Uploading a directory of supported files

```text
Browser                        HTTP                       Service / cloud
-------                        ----                       ---------------
pick directory
or capture drop roots
  |
traverse all reader batches
  |
stage [{File, relativePath}]
  |
POST multipart -------------> parse and read bytes
                               |
                               StartUpload -------------> create in-memory job
<----------------------------- 202 {jobId}                 launch job goroutine
  |                                                        |
poll GET job -------------------------------------------- snapshot item states
                                                           |
                                         parallel input pipelines (per job)
                                                           |
                                         write private temp input
                                                           |
                                         if Markdown: pandoc -> PDF
                                                           |
                                         acquire service mutex
                                                           |
                                         ensureDir(targetDir)
                                         UploadDocument(parentID,...)
                                         AddDocument(returned document)
                                         SyncComplete + reindex
                                         release mutex
                                                           |
                                         mark item done / failed
<-------------------------------------------------------- terminal snapshot
reload list or search
```

`uploadOne` normalizes backslashes in relative paths, drops empty/dot/dot-dot segments, splits directory versus basename, sanitizes the basename stem, then joins the relative directory to the requested destination. The sanitized name is used for the temporary file; rmapi derives the remote display name from it. `ensureDir` delegates to `rmcloud.MkdirAll`, which walks/creates intermediate folders and updates the local tree.

This preserves **remote hierarchy for supported documents**, not a general-purpose folder copy. Empty directories have no representation in the request. Images and other unsupported files become failed job items. A Markdown file and its adjacent image are not staged into a shared local tree, so their relative image link will not work like the CLI's local-directory conversion. See F12.

### 6.2 Job state is not the same as item state

Each item progresses through queued, converting (Markdown only), uploading, and done/failed. Aggregate state is queued initially; `setItem` then sets it to uploading while anything is pending, failed when all items are terminal and at least one failed, otherwise done. Consequently the aggregate job need not expose converting even when an item is converting.

`done` counts successful items only, not completed failures. A fully terminal job can say `failed`, `done=2`, `total=3`. This is not necessarily wrong, but it needs a clearly named progress contract such as successful/failed/pending counts. Current polling shows the item errors but does not throw on a failed job and returns normally if polling itself fails.

### 6.3 Finding a document

`Service.Search` copies the Entry index under the mutex, releases the mutex, then ranks it in memory. The scorer lowercases and trims the query. Exact name is approximately 1000 points, name prefix approximately 800, fuzzy name 400 plus bonuses, and fuzzy path 150 plus bonuses. Folders receive a small bonus for name matches. A greedy subsequence matcher rewards consecutive matches and word starts; shorter strings get a modest advantage.

```text
search(query, snapshot):
    for entry in snapshot:
        if exact name: add high-score result
        else if name prefix: add prefix result
        else if query is subsequence of name: add name result
        else if query is subsequence of path: add path result
    sort by score descending, then name/path
    truncate to requested limit
```

The implementation scans bytes after lowercasing; it is not a Unicode-aware subsequence algorithm. Scoring long queries can also exceed the nominal category ranges, so prefix-before-fuzzy is a tendency, not a formal invariant for arbitrary lengths. For ordinary short names, this is a reasonable first version. More advanced search is lower priority than correctness and safety.

If N entries match and L is average name/path length, scanning is roughly O(NL) and sorting all matches is O(N log N). Every search copies N entries first. The endpoint's positive `limit` is not capped; truncation happens after scoring/sorting. The frontend debounces input by 180 ms, but debounce does not order responses. F06 shows why this matters.

## 7. Prioritized review findings

Severity used here: **P1** means fix before merge because destructive semantics, security, or state integrity are at stake. **P2** means an observable product bug or meaningful robustness gap. Findings are grounded in the reviewed snapshot; they are not claims that proposed remedies have already been implemented.

### F01 — P1: recursive delete does not delete the cloud subtree

**Evidence: source-confirmed and exercised against actual dependency data structures.** `manage.go:99-119` calls `DeleteEntry(node, recursive, true)` once per requested root, then `DeleteNode`. In the pinned `rmapi/api/sync15/apictx.go:265-274`, `recursive` only bypasses the “directory is not empty” check. The actual operation is `HashTree.Remove(node.Document.ID)`. `rmapi/api/sync15/tree.go:224` removes exactly one record and never walks descendants.

Locally, `FileTreeCtx.DeleteNode` detaches the selected folder from its parent's children. The Entry walk no longer reaches its descendants, so the UI appears clean. Remotely, those descendant records still refer to a now-missing parent. `DocumentsFileTree` rebuilds nodes and calls `FinishAdd`, which attaches missing-parent children to root. The offline dependency probe confirms `Remove(parent)` leaves the child record and a rebuild gives path `/child`.

**Impact:** “delete folder and contents” is not implemented. Descendants can reappear after refresh/restart. A previous smoke test that only checked disappearance from the existing root listing was insufficient to establish recursive deletion. This also affects the claimed cleanup of temporary trees; the review did not inspect the live account to locate any leftovers.

**Proposed correction:** expand selected roots into a stable, deduplicated, child-before-parent ID plan under the service lock; execute a delete for every planned record; reconcile after partial failures. Count and report root selections separately from individual document records. Alternatively fix the dependency with explicit recursive semantics and tests, but do not assume its flag provides those semantics today.

**Acceptance:** a parent/nested folder/document fixture must be absent from both cloud-record representation and the rebuilt file tree; perform one opt-in real integration round trip only in an isolated throwaway tree. A missing parent in a shallow UI listing is not the assertion.

### F02 — P1: the browser-to-local-server trust boundary is missing

**Evidence: observed mux behavior and source inspection.** `Server.Handler` returns the mux directly (`server.go:32`). There is no Origin, Fetch Metadata, Host, or token validation. `decodeJSON` accepts JSON bytes with `Content-Type: text/plain`. A probe with `Origin: https://attacker.invalid`, `Sec-Fetch-Site: cross-site`, and text/plain JSON created a folder with HTTP 201.

This probe establishes handler acceptance, not a demonstrated exploit in every browser. Browser private-network protections vary. Nevertheless, no CORS headers does not prohibit all cross-origin writes: simple requests may be sent even if the caller cannot read the response. Multipart upload is another relevant boundary. Typed `DELETE` is a UI intent check, not authentication; an arbitrary client can send that constant. DNS rebinding and direct non-browser clients also require separate consideration.

Furthermore, `--addr 0.0.0.0:8080` is accepted without an exposure warning or refusal (`command.go:50,75`). The feature is **loopback by default**, not loopback-only as the help and PR wording imply.

**Proposed correction:** validate and restrict the listener to loopback; introduce a top-level request guard with a canonical accepted Host and same-origin policy. Reject explicitly foreign Origin/Fetch Metadata on mutations, require expected media types, and choose a session-token strategy for stronger protection. Define behavior for absent browser headers and trusted CLI access instead of silently accepting everything. If external hosting is desired, it is a separate authenticated design, not a flag tweak.

**Acceptance:** cross-site text/plain JSON and multipart requests must be rejected before service calls; same-origin operations still succeed; invalid Host and non-loopback configuration are rejected. Test missing-header policy explicitly. Never describe CORS or loopback as a substitute for the guard.

### F03 — P1: resource bounds are per request/job, not per service

**Evidence: source-confirmed.** `handleUpload` reads all file bytes into `[]UploadInput`; each admitted job starts its own goroutine and its own semaphore of `Workers` slots (`upload.go:153-194`). The job map never evicts records. There is no total active-job count, queued-byte budget, global conversion semaphore, or shutdown join.

For J active jobs and W workers, roughly J times W input pipelines can be active. All cloud access still serializes through `s.mu`, so jobs may accumulate converted PDFs and retained input byte slices while waiting. A 64 MiB request limit bounds neither aggregate RAM nor total queued work. The “30 minute” job timeout is only passed to conversion and checked at `uploadOne` entry: it does not bound waiting on the mutex or cloud methods using the process-owned transport. A cancelled conversion job may still progress into a cloud mutation if cancellation arrives after its entry check.

**Proposed correction:** service-owned lifecycle, global admission budgets, global conversion slots, capped queued bytes/files, temporary-file-backed inputs, terminal-record TTL, and a bounded `Close`/drain strategy. Preserve independence from the initiating request context, but derive jobs from the service lifetime. Check cancellation before cloud entry; distinguish conversion deadline from context-free cloud I/O limitations.

**Acceptance:** simultaneous jobs cannot exceed the configured global conversion bound; overload returns a documented 429/503; cancelled queued jobs do not mutate cloud state; shutdown stops/drains jobs according to a tested policy; terminal jobs and input spools are cleaned up.

### F04 — P2: uploads allow ambiguous sibling names and sanitizer collisions

**Evidence: observed with the real service and fake transport; dependency source confirms no name conflict check in `UploadDocument`.** `uploadOne` never checks the resolved parent for `safe` before uploading (`upload.go:242-257`). The pinned uploader has a `TODO: overwrite file` and creates a new document ID; it does not implement the CLI's pre-upload conflict guard.

`a b.pdf` and `a_b.pdf` both become cloud name `a_b`; the probe produced two sibling entries with that name. `Plan.md` and `Plan.pdf`, or different non-ASCII names that sanitize to `document`, are other collision cases. This is not proof of automatic overwrite or annotation deletion; the current risk is ambiguous duplicate names and unpredictable path lookup.

**Proposed correction:** define a default reject-on-conflict policy, preflight normalized destination/name keys across the batch, and recheck the live parent while holding the service lock. Return a per-item conflict with original and effective names. Offer overwrite/content-only only as a separate explicit workflow with annotation semantics, not a silent retry behavior.

**Acceptance:** duplicate effective keys within a batch and conflicts with existing siblings do not create extra documents. Same basename in distinct destination folders remains valid.

### F05 — P1: partial mutations leave stale index and deleted-ID handles

**Evidence: observed offline plus dependency source.** Deleting `[validID, missingID]` deletes the first local node then returns not found for the second. `mutate` exits before `reindexLocked` (`service.go:182-194`). The probe found `List` no longer contained the document while `Entries` still did. The HTTP layer discards the partial deletion count and returns only an error.

There is a second integrity problem: pinned `FileTreeCtx.DeleteNode` only removes the parent-child edge (`rmapi/filetree/filetree.go:108`), leaving `idToNode` entries for the node and descendants. `nodeByIDLocked` accepts those disconnected nodes (`service.go:170`). A fake-storage download of a deleted ID therefore passed service validation; real storage might reject it, so the probe is **not** evidence that a real deleted document can still be downloaded. It is evidence that service lookup no longer means “live reachable entry.”

**Proposed correction:** validate/deduplicate batch plans first; expose explicit per-ID outcomes for failures that occur after side effects; reconcile authoritative tree/index on uncertain results. Ensure ID lookup only returns reachable live nodes, either by a correct dependency removal operation or a service-owned live-ID projection. Do not “fix” stale search by only reindexing if the underlying disconnected IDs remain usable.

**Acceptance:** injected failures at every delete step leave list, search, ID lookup, and reported outcomes consistent. Repeated or parent-plus-child delete selections are handled deterministically. Retry after a partial result does not pretend the operation was atomic.

### F06 — P2: frontend responses can display the wrong directory or view mode

**Evidence: deterministic execution of actual frontend functions with deferred fetch responses.** `loadFiles` sets `state.cwd` immediately, then accepts every response (`app.js:93-107`). Start `/A`, then `/B`, deliver `/B` first and `/A` last: the observed state is `cwd=/B`, rows from A. Search has the same ordering problem; its 180 ms debounce does not prevent overlapping requests.

`loadFiles` also does not clear `state.query`. Double-click a folder in search results: directory children load, but breadcrumbs and count still say search, and later `refreshAfterMutation` runs the old global search. Search does not consistently clear selected detail state either. These are state-model defects, not a reason to adopt Redux.

**Proposed correction:** one view model with explicit directory/search mode, a shared request generation (or abort plus generation), and atomic commit of results only if still current. Folder navigation clears search text/query and cancels pending debounce. Mutation refresh uses the active view's explicit mode. Distinguish requested and successfully displayed directory state on error.

**Acceptance:** reordered folder/search responses cannot replace newer state; entering a folder from search updates rows, breadcrumbs, input, selection, count, URL, and post-mutation refresh coherently.

### F07 — P2: explicit root upload is replaced by configured default

**Evidence: observed offline.** `StartUpload` normalizes the supplied destination and replaces `/` with `DefaultRemoteDir` (`upload.go:157-160`). With default `/ai`, an explicit upload to `/` becomes `/ai`. The UI fills the destination from the current directory, so selecting root cannot override that configuration.

**Proposed correction:** apply the default only to an omitted/blank input, before normalization; preserve explicit `/`. Decide whether the dialog defaults to current directory or configured default and show that choice rather than exposing one value but executing another.

**Acceptance:** omitted destination uses `/ai`, explicit `/` uses root, explicit other path is preserved. Strengthen `TestUploadFolderPaths`: its current `destDir=/` with default `/ai` masks this bug, and it only verifies an item's input name.

### F08 — P2: hash routing does not round-trip literal percent sequences

**Evidence: observed routing function behavior.** `setHash` assigns the raw path while `currentHashPath` calls `decodeURIComponent` (`app.js:39-49`). A cloud folder literally named `literal%20name` can become `literal name` on reload because percent sequences have not been encoded as data. Similar ambiguity applies to percent-encoded slashes.

**Proposed correction:** define a single encoding/decoding contract, such as `#` plus `encodeURIComponent(canonicalPath)`, or encode each path segment and join with `/`. Decode exactly once; display/report malformed routes instead of silently switching interpretations. Make breadcrumb anchors actual encoded URLs rather than `href="#"` plus an onclick override.

**Acceptance:** names with spaces, `%20`, `%2F`, `%`, `#`, `?`, and non-ASCII text survive navigation, copy/paste, reload, and history. A literal `#` is not inherently a fragment-parser failure: the prior diary's broad warning about `#` was not a browser-verified diagnosis. Test it rather than substituting that warning for the actual percent-round-trip defect.

### F09 — P2: upload/dialog lifecycle is easy to desynchronize

**Evidence: source-confirmed, with an observed unchanged input value on reopen.** Opening Upload clears `staged` but does not reset either file input (`app.js:367-376`). Selecting the same file/folder again can produce no browser `change` event, leaving no staged files despite the old visible selection. The native event consequence needs a real-browser regression test; the stale input value itself was reproduced offline.

Drop traversal rejections are not caught in the async drop handler (`app.js:422`), so inaccessible entries can fail without a useful toast. The current fallback uses `dt.files` only if no root entries were captured; mixed supported/null roots can silently omit files. Inputs and Close remain usable during polling; reopening or dropping into the upload dialog can replace staging/progress while an earlier job still writes to the same DOM. Native Escape closes prompt/confirm dialogs without resolving their promises. Polling failure renders “job lost” and returns normally, rather than marking the view as unable to track an accepted job.

**Proposed correction:** explicit upload view state with active job ID, reset file inputs on reset, disable conflicting actions or separate staging from active-job progress, catch traversal errors, and settle dialog promises on native close/cancel. Closing the upload dialog should either keep a visible background-job tracker or explicitly explain that the job continues; it should not imply cancellation.

**Acceptance:** reselect same file/folder, native Escape, unreadable drop entry, close/reopen while uploading, lost polling response, and mixed-success jobs all have deterministic outcomes. Preserve synchronous root capture and the repeated `readEntries` loop, both of which are already correct design choices.

### F10 — P2: long-lived auth and capability status are incomplete

**Evidence: source-confirmed.** The service uses `CreateApiCtx` at startup but never `WithAuthRetry` in mutations, download, or refresh. A stale token becomes a generic error or failed job until manual intervention. `Status.Authenticated` checks only non-nil context; the missing-pandoc banner checks only one of the required tools (`service.go:120`, `app.js:70`).

**Proposed correction:** preserve non-interactive auth during recovery, atomically replace the service-owned context under the mutex, and re-resolve IDs/paths from the fresh context. Extend status to separately describe context presence, last successful cloud operation, auth-required state, and conversion prerequisites. Do not wrap an entire non-idempotent batch blindly in retry: after a partial upload/delete, repeating it can duplicate or misreport work. Track and reconcile outcomes first.

**Acceptance:** fake first-attempt auth error then recovered success; failed recovery yields a clear user action without stdin prompting; retries operate on fresh nodes; ambiguous publication does not create duplicate upload records; missing XeLaTeX gives a useful preflight message.

### F11 — P1: download uses cloud display name as a local output path

**Evidence: source-confirmed; no exploit or writes outside temporary directories were attempted.** `Download` joins `tmpDir` with `node.Name()+".rmdoc"` (`manage.go:139`). HTTP rename/create restrict slashes, but existing cloud metadata may come from other clients and is not validated on download. A path-shaped cloud name can influence the local staging destination. The attachment header only strips double quotes (`handlers.go:220`), not all control characters or non-ASCII formatting concerns.

**Proposed correction:** use a fixed temporary basename such as `document.rmdoc` (or an internally generated ID-safe basename), independent of cloud display name. Construct a separately sanitized suggested filename using proper `Content-Disposition` formatting. Treat external cloud metadata as untrusted input, even though browser requests use IDs.

**Acceptance:** names containing separators, dot-dot components, quotes, control characters, and Unicode cannot affect the output path and produce valid download headers. This guard is inexpensive and should precede any attempt to optimize download lock duration.

### F12 — P2: folder hierarchy preservation is not Markdown asset preservation

**Evidence: source-confirmed.** Every file gets a separate `rmfiles-up-*` directory. The Markdown converter resolves relative images from that directory (`upload.go:209-231`, `pkg/mdpdf/pandoc.go:181-186`), not from the original selected tree. A sibling `images/figure.png` is neither present in that conversion directory nor an accepted standalone document type.

**Impact:** “upload a Markdown project folder” can preserve document folders but lose the Markdown document's images or fail conversion. Empty folders are also omitted. This is distinct from remote file hierarchy and must be explained to users.

**Proposed correction:** either explicitly limit folder upload to independently renderable supported documents, or stage a validated job-local source tree with asset references and convert against it. If implementing the latter, enforce confinement of all local resource reads and define an allowed asset policy. `mdpdf` is a trusted local CLI pipeline: it leaves absolute/URL images unchanged and permits ordinary Markdown/TeX preprocessing. Reusing it for arbitrarily supplied web bytes is a changed trust boundary, not proof of a sandbox. No arbitrary-file-read or subprocess exploit was demonstrated here.

**Acceptance:** a Markdown file referencing a selected sibling image renders with that image under the chosen policy; outside-tree resources are rejected where confinement is promised; empty-folder semantics are documented.

## 8. Additional design, UX, and maintenance observations

### 8.1 Visual contract: two colored-fill violations remain

Most of the UI follows the requested modern-font, black/white, square-corner, no-window-chrome style. The drop overlay now has an explicit `[hidden]` rule; that specific always-visible bug is fixed in this head.

However, `.btn.danger:hover` and `.toast.error` set `background: var(--danger)` (`styles.css:63,223`). These are colored fills, not colored text highlights, and violate the user's explicit constraint. Keep their backgrounds white or black and use danger color for text/underlining. This conclusion comes from CSS source; no screenshot review was performed.

The layout is desktop-first with a fixed 300 px detail pane and 360 px minimum-width dialogs. There are no row keyboard navigation/focus semantics, `aria-selected`, or live announcements for toast/job progress. Table rows can only be selected/opened with pointer handlers. Keyboard usability and screen-reader labeling should be added without changing the visual language. Do not remove the text-selection suppression requested by the user.

### 8.2 Static serving and development semantics

The page's CSS and JS URLs are relative (`index.html:7,119`). Arbitrary nested SPA paths served by the fallback resolve assets relative to that nested path; unknown `.js` and `.css` requests also fall back to HTML. Hash routing avoids this for normal entry URLs because the fragment is not sent to the server, but the advertised generic fallback is less robust than the tests imply.

Use root-relative, stable asset URLs (prefer the repository's `/static/` convention) and return 404 for unknown assets rather than HTML. A test that `/some/client/route` returns the shell does not prove the browser can load its modules.

`--dev` disables caching on **embedded** bytes. Editing source CSS/JS and merely refreshing the already-running process does not change those bytes. Recompile/restart, or implement a clearly separate disk-backed dev asset handler. The design's current “edit frontend and reload” sentence is inaccurate.

### 8.3 Input names and other small contract defects

`CreateFolder` and `Rename` allow `.` and `..` because they only reject empty names and slash/backslash. These names have special path-resolution semantics in rmapi, making a folder's synthesized display path unsuitable for resolving it. The backend probe successfully created `..`. Reject reserved segments and control characters consistently, while preserving legitimate Unicode names where supported.

Other small improvements: reject trailing JSON after the first object; use `MaxBytesReader` for JSON rather than silently truncating; cap search query length and result limit; report oversized uploads as 413; distinguish unsupported format from corrupt PDF content; and document parser errors without exposing excessive conversion diagnostics. These are targeted boundary fixes, not a mandate for a generalized validation framework.

### 8.4 Download responsiveness and memory

Download holds `s.mu` across network fetch and reads the complete archive into RAM before responding (`manage.go:123-147`). This blocks file-list and search-snapshot acquisition and duplicates large download data in memory. A temp-file response/streaming interface would reduce memory, but removing the lock around the existing shared context would trade an observable performance problem for a correctness risk. First establish a safe snapshot/serialized cloud execution model, then optimize response transmission outside its critical section.

### 8.5 Progress costs and retained metadata

Each `setItem` rescans all items; repeated updates are O(number of items) apiece. Polling also renders every item every 700 ms, while the staging summary is capped at 200 visible paths. Large batches therefore have server and DOM work beyond just PDF conversion. Use counters maintained on state transitions and bounded/virtualized progress display when batches grow. These optimizations are lower priority than global admission control and cleanup.

## 9. Design-document review: what is accurate, stale, or deferred

The original guide is useful background on rmapi ownership, the path/ID distinction, cloud credentials, conversion, and the phased service split. It also records the deliberate no-build frontend choice in a status note. Unfortunately, it leaves many historical proposals intermixed with current-state instructions. A new intern can easily implement against the wrong API.

| Design location | Current mismatch | Recommended treatment |
| --- | --- | --- |
| Executive summary and §§3.2,6.1 | Still describes React, Vite, and `frontend/dist` | Label as historical proposal; point to three embedded files |
| §3.4 upload flow | Singular `/api/upload`, synchronous 201 response | Replace current-state example with plural uploads and 202/polling |
| §3.5 find flow | Search through `/api/files?query=` | Current API is `/api/search?q=` |
| §4.4 / §6.3 | Auth retry, service Close, upload semaphore appear in sketches | Explicitly mark absent, then link remediation design |
| §4.6 and Appendix B | “Copy packaging exactly” and React toolchain cheat sheet | Superseded by accepted no-build choice; old UI is not the base |
| §6.6 | Cancellation endpoint and cancelled state | Not implemented; deferred API, not working feature |
| §6.11 | Edit files and reload in dev | Embedded assets require recompilation/restart |
| §6.12 | Origin/CSRF and job caps are requirements | Missing, review blockers/robustness work, not completed safety |
| §7 API table | `parentId`, `destDirId`, search entries envelope | Current fields are `parentPath`, `destDir`, `results` envelope |
| §§4.4,6.6 | SyncComplete described like a commit; upload lock “short” | Cloud methods publish themselves; network lock may be long |
| Visual sections | Dither still appears in proposed sketches | Later user explicitly removed patterned backgrounds behind text |
| §9 validation | Frontend framework tests proposed but absent | Add vanilla-function and actual-browser regression tests |

The quickest documentation repair is to add a clear “implemented contract” section linking this review and label all older architecture/API sketches as historical. Do not erase the original investigation history. Longer term, split accepted architecture/current API reference from the historical proposal so each fact has one authoritative current home.

### 9.1 Decision record: keep the no-build frontend

- **Context:** the app is currently small and ships inside an existing Go binary.
- **Options considered:** keep vanilla JS; adopt React/Vite/TypeScript; reuse the older rendering UI.
- **Recommendation:** keep the static frontend and add explicit state and tests.
- **Rationale:** the observed bugs are ordering/lifecycle/contract bugs, not proof a framework is required. Reusing the older UI contradicts the chosen product boundary.
- **Consequences:** no mandatory Node build for distribution; some runtime shape discipline must be supplied by small tested adapters and clear data contracts.
- **Status:** retain accepted implementation choice; remediation proposed.

### 9.2 Decision record: cloud operations remain serialized

- **Context:** ApiCtx/file tree/cache are mutable and the dependency does not provide a concurrency-safe service API.
- **Options considered:** unlock network work immediately; retain mutex; introduce a single cloud-operation executor with immutable read snapshots.
- **Recommendation:** retain serialization now; consider an executor only if responsiveness measurements justify it.
- **Rationale:** correctness precedes parallel throughput for destructive cloud operations.
- **Consequences:** reads can still block on network operations; bounded jobs and snapshot-based reads are safer improvements than concurrent context use.
- **Status:** current mutex accepted as baseline; executor is optional future design.

### 9.3 Decision record: reject collisions by default

- **Context:** normalization can collapse multiple inputs into one visible name; annotations make replacement dangerous.
- **Options considered:** silent duplicate; automatic suffix; default reject; overwrite/content-only replacement.
- **Recommendation:** reject effective-name conflicts with a clear per-item result, then offer explicit rename or future replacement workflow.
- **Rationale:** preserves predictable navigation and does not make annotation-affecting policy implicit.
- **Consequences:** batch planning and a lock-time conflict recheck are required; no new overwrite endpoint is necessary for the first fix.
- **Status:** proposed.

## 10. Proposed algorithms and API changes

The sketches in this section are **not implemented code**. They describe invariants and ownership boundaries, not compatibility adapters.

### 10.1 Safe deletion plan and partial results

```text
planDelete(selectedIDs, recursive):
    lock service
    resolve every selected ID as a reachable live node
    reject root/trash/reserved targets
    deduplicate selections and overlapping descendants
    if not recursive and any selected folder is nonempty:
        return conflict before writing anything
    walk each selected subtree in postorder
    plan = unique IDs ordered children before parents
    record plan paths/names for reporting
    return plan under the same operation boundary

executeDelete(plan):
    for ID in plan:
        check service cancellation
        resolve current live node
        delete exactly that cloud record
        update live tree/ID bookkeeping
        append per-ID success
        if error:
            reconcile tree/index where possible
            return partial outcomes + error + reconciliation status
    rebuild index and live-ID projection
    notify if needed
    return outcomes
```

Planning eliminates invalid-target and overlap failures before side effects; it cannot eliminate network failures halfway through execution. An example proposed result is:

```json
{
  "selectedRoots": 1,
  "deletedRecords": 3,
  "results": [{"id":"uuid","state":"deleted"}],
  "partial": false,
  "reconciled": true
}
```

Choose and document the HTTP status for a partial result rather than hiding it in a generic 500. Preserve error identity and use machine-readable per-item outcomes. Cloud generation conflicts must be reconciled, not described as mutex violations.

### 10.2 Bounded service-owned jobs

```text
startUpload(inputs, requestedDestination):
    destination = default only if input omitted
    validate relative paths, names, effective collision keys
    reserve service-wide job/file/byte budget
    spool inputs into private job storage
    create tracked job derived from service lifetime
    enqueue work
    return job ID

worker():
    acquire global conversion slot
    convert outside cloud serialization
    release conversion slot
    check job cancellation
    enter serialized cloud operation
    resolve destination and recheck conflict
    upload and reconcile/publish outcome
    leave serialization
    update item counters
    delete input/output spool when safe

closeService(deadline):
    stop admission
    cancel or drain tracked jobs according to policy
    wait within deadline
    clean owned spool files
```

A proposed service interface could add `Close(ctx)`, `CancelUpload(id)`, and explicit global limits to `Config`. This is a new design contract, not an installed API. Keep conversion pluggable for tests: a fake converter with a barrier can prove the global concurrency limit without invoking pandoc.

The service command's shutdown must also account for `cmd/remarquee/interrupt.go`: its top-level watchdog can force exit after two seconds, while `runServe` asks HTTP Shutdown for five seconds. Background jobs are outside HTTP's request accounting. Align these policies; a five-second child drain is not guaranteed by the current parent watchdog.

### 10.3 Coherent frontend request state

```text
navigateDirectory(path):
    cancel pending search debounce
    set view mode = directory
    clear query, search input, selected detail
    route = encoded canonical path
    epoch = increment shared view request generation
    abort previous fetch if available
    response = await list(path)
    if epoch is not current: ignore response
    atomically commit displayed path, rows, count, breadcrumbs

search(query):
    set view mode = search
    epoch = increment same generation
    response = await rankedSearch(query)
    if epoch is not current: ignore response
    atomically commit search rows and metadata

refreshCurrentView():
    dispatch from explicit view mode, not merely nonempty old query
```

Use both abort and generation if useful: abort saves work, while generation protects correctness when a response is already completing. Do not use independent epochs for search and directory loads if those responses share the same rows state.

### 10.4 A clear security boundary

```text
requestGuard(request):
    verify canonical allowed Host
    if browser mutation:
        reject explicitly foreign Origin
        reject cross-site Fetch Metadata
        require session token under the chosen policy
        require correct Content-Type for route
    apply body/admission limits
    invoke mux
```

Header validation needs a documented absent-header policy for trusted automation, reverse proxies, and tests. A permissive absent-header policy plus an arbitrary externally exposed listener is not a secure deployment mode. Derive origin checks from configuration rather than trusting an attacker-supplied Host to define the accepted origin. A token does not replace the need to limit network exposure and protect local-resource conversion.

## 11. Ordered implementation plan for the intern

### Phase A — establish safety before UI expansion

1. Implement F01 child-before-parent deletion and dependency-faithful tests.
2. Implement F05 live-ID integrity, partial result handling, and failure reconciliation.
3. Implement F02 listener restriction and request guard, including JSON/multipart input contracts.
4. Implement F11 fixed download staging filename and correct attachment formatting.

**Exit gate:** no selected descendant remains in rebuilt cloud state; no deleted ID passes live lookup; foreign-origin requests do not call domain operations; cloud names cannot affect local output paths. Run unit/handler tests with adversarial fixtures. Do not use the real account as the first test harness.

### Phase B — make uploads bounded and predictable

1. Implement global admission/conversion limits, spool ownership, shutdown tracking, and TTL (F03).
2. Implement conflict preflight and lock-time checks (F04).
3. Correct explicit root versus omitted destination (F07).
4. Integrate auth recovery with explicit idempotency/reconciliation policy (F10).
5. Choose/document Markdown asset and empty-folder policy (F12).

**Exit gate:** multi-job tests establish global bounds; collisions are rejected; root selection round-trips; cancellation/shutdown cannot silently leave untracked conversion jobs; auth retry does not duplicate documents.

### Phase C — make browser state coherent

1. Shared request generation and explicit search/directory mode (F06).
2. Route encode/decode contract and real breadcrumb links (F08).
3. Upload input reset, active-job tracking, native dialog close/cancel handling, and drop error handling (F09).
4. Keyboard row actions, live progress announcements, monochrome danger surfaces, and stable asset URLs.

**Exit gate:** deterministic deferred-fetch tests and actual-browser interaction tests pass. Include a screenshot/computed-style assertion that the drop overlay is hidden on initial load and only shown during file drag, rather than checking for a CSS string alone.

### Phase D — qualify and update documentation

1. Update accepted API/defaults/dev instructions and label historical proposals.
2. Expand happy-path tests to assert resulting state and failure outcomes.
3. Run build, full tests, affected race tests, and frontend logic/browser tests.
4. Run one opt-in isolated real-cloud cycle: create nested tree, upload, rename/move, delete plan, refresh/restart, prove every descendant absent by ID.
5. Clean up verified smoke artifacts by known IDs, with explicit operator approval where necessary. Do not issue broad account-root cleanup commands.

**Exit gate:** a reviewable evidence receipt distinguishes unit, browser, real cloud, and physical tablet synchronization. Do not close unresolved safety tasks because a report was delivered.

## 12. Validation performed and tests still required

### 12.1 Results from this review

At reviewed revision `f2dd68ce4a19467ace9891eeafdddfb1443b2b70`, with only ticket review artifacts added:

- `go build ./...`: exit 0.
- `go test ./...`: exit 0; many packages used cached test results.
- `go test -race ./pkg/rmfiles ./cmd/remarquee/cmds/serve`: exit 0.
- `go vet ./pkg/rmfiles ./cmd/remarquee/cmds/serve`: exit 0.
- `node --check cmd/remarquee/cmds/serve/frontend/app.js`: exit 0.
- Backend offline probe: completed; foreign-origin acceptance, root-default substitution, collisions, stale index/IDs, reserved name, and trailing JSON reproduced.
- Frontend offline probe: completed; reordered response, stale search mode, percent decoding ambiguity, and unchanged picker input reproduced.

Passing `-race` only covers exercised paths; it does not establish cloud transaction semantics or frontend request ordering. Likewise, `node --check` is syntax validation, not a browser interaction or rendering test.

### 12.2 Reproduction commands

Run these from the repository root; they have no real cloud transport:

```bash
T=ttmp/2026/10/04/RMQ-0025--remarkable-files-web-ui-macos-1-monochrome-upload-manage-and-find
go run "$T/scripts/01-review-backend.go"
node "$T/scripts/02-review-frontend.cjs"
```

Representative captured output:

```text
foreign Origin + text/plain create-folder: HTTP 201
explicit root upload with default /ai: dest=/ai state=done
sanitized upload collisions: 2 sibling documents named a_b
partial delete: deleted=1 error=entry not found listContains=false indexContains=true
pinned HashTree.Remove(parent): remaining=1 childStillPresent=true
rebuild after removing parent: child path=/child
out-of-order folder responses: cwd=/B rows=A-entry
navigate from search: cwd=/Target query=proj
```

These probes intentionally assert **current defective behavior** so the review can be reproduced. They are not regression tests asserting the desired product contract. Convert them into proper tests with corrected expectations when implementing fixes.

The first backend probe fixture omitted blob hashes and emitted `missing hash for: child`. The final probe supplies placeholder hash-shaped values and has no such fixture diagnostic. Both outputs are retained; the diagnostic did not indicate a production transport failure.

### 12.3 Why existing tests missed the defects

`TestUploadFolderPaths` accepts either a done **or failed** job and only checks `job.Items[0].Name`. It does not require successful items, verify destination, inspect created parent IDs, or resolve the expected cloud tree. It can pass without a successful folder upload.

Both fake APIs model successful methods generously. They do not faithfully model nonempty-directory rejection, exact hash-tree deletion, notification behavior, expired tokens, or side-effect-then-error cases. The service tests for Delete cover a single leaf. Handler tests assert successful responses and minimal bodies; they do not send hostile Origin/Host, oversized/malformed input, or duplicate effective filenames. No frontend behavior test suite is present in the feature.

A better test matrix includes:

| Area | Required assertions |
| --- | --- |
| Delete plan | Descendants first; deduplicate overlap; root protected; absence after rebuild |
| Partial mutation | Per-ID results, consistent index, live lookup, reconciling state |
| Security | Foreign Origin, invalid Host, media type, missing-header policy, non-loopback config |
| Upload destination | Empty uses default; explicit root preserved; nested paths match created parents |
| Upload names | Intra-batch and existing conflicts; Unicode and sanitizer collisions |
| Resources | Global worker peak, queued bytes, overload, TTL, cancelled work, shutdown cleanup |
| Auth | Fresh context adopted; node re-resolution; no blind non-idempotent retry |
| Download | Safe fixed staging filename and valid attachment headers |
| Frontend requests | Deferred out-of-order folder/search responses and mode transitions |
| Browser interaction | Same-file reselect; Escape; drag reader failures; native overlay visibility |
| Markdown assets | Chosen sibling-image and resource-confinement policy |
| Packaging | Nested shell asset resolution, unknown asset 404, documented dev restart behavior |

## 13. References and evidence inventory

### 13.1 Ticket documents

- Original design: `design-doc/01-remarkable-files-web-ui-design-and-implementation-guide-for-a-new-intern.md`.
- Chronological work and historical smoke tests: `reference/01-implementation-diary.md`.
- Current follow-up inventory: `tasks.md`.
- This review is the source of truth for **review conclusions at the pinned snapshot**, not a claim that those fixes have landed.

### 13.2 Review artifacts

- `scripts/01-review-backend.go`: in-memory API/mux probes and actual pinned hash-tree/file-tree behavior.
- `scripts/02-review-frontend.cjs`: actual app declarations/listeners with a stubbed DOM, no boot fetch, and controlled response completion.
- `sources/01-offline-review-probes.txt`: initial probe output, including the incomplete-hash fixture diagnostic.
- `sources/02-offline-review-probes-final.txt`: corrected fixture and final probe output.
- `sources/03-review-validation.txt`: build/test/race/vet/syntax commands and exit codes.

### 13.3 Dependency reading that is essential for destructive changes

- `rmapi/api/api.go`: full `ApiCtx` interface.
- `rmapi/api/sync15/apictx.go:214`: `Sync` publishes root changes and persists cache.
- `rmapi/api/sync15/apictx.go:265`: `DeleteEntry` does not recurse into descendants.
- `rmapi/api/sync15/apictx.go:338`: `UploadDocument` creates a new document without sibling-name protection.
- `rmapi/api/sync15/apictx.go:470`: `DocumentsFileTree` rebuild and `FinishAdd` orphan handling.
- `rmapi/api/sync15/apictx.go:495`: notification-only `SyncComplete` and swallowed notification errors.
- `rmapi/api/sync15/tree.go:224`: single-record `HashTree.Remove`.
- `rmapi/filetree/filetree.go:67,108,211`: orphan placement, delete detachment, and path resolution.

### 13.4 API documentation for implementation work

These are reference entry points, not a claim that the review ran browser conformance tests:

- Go HTTP server, ServeMux, MaxBytesReader, and Shutdown: <https://pkg.go.dev/net/http>.
- Go multipart Form/FileHeader and RemoveAll: <https://pkg.go.dev/mime/multipart>. Standard `net/http` cleans parsed multipart temporary files after a normal request; do not report a guaranteed production parser-temp leak merely because this handler lacks an explicit RemoveAll. The significant resource risks here are retained input bytes, job records, spools, and aggregate admission.
- Go attachment header formatting: <https://pkg.go.dev/mime#FormatMediaType>.
- Browser drag entries: <https://developer.mozilla.org/en-US/docs/Web/API/DataTransferItem/webkitGetAsEntry>.
- Directory-reader batches: <https://developer.mozilla.org/en-US/docs/Web/API/FileSystemDirectoryReader/readEntries>.
- Native dialog cancellation: <https://developer.mozilla.org/en-US/docs/Web/API/HTMLDialogElement/cancel_event>.
- Browser request cancellation: <https://developer.mozilla.org/en-US/docs/Web/API/AbortController>.

## 14. Final guidance

Start with the invariant you are protecting, not the library call you hope implements it. “A recursive delete removes every selected descendant from the rebuilt cloud account,” “a stale response cannot change the current view,” and “an explicit root destination remains root” are useful invariants. “Call DeleteEntry with true,” “debounce search,” and “normalize the path” are implementation steps that do not, by themselves, prove those invariants.

This PR does not need more architectural ambition before it can be useful. It needs a firmer boundary around destructive cloud state, local browser authority, resource ownership, and frontend request state. Preserve its small service-oriented structure, add the failure-oriented tests described here, then requalify the real end-to-end path before merge.
