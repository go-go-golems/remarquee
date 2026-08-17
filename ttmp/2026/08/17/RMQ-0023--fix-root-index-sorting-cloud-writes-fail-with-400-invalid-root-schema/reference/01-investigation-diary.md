---
Title: Investigation diary
Ticket: RMQ-0023
Status: active
Topics:
    - backend
    - rmcloud
    - upload
    - dependency
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go.mod
      Note: rmapi replace pin — the fix target
    - Path: pkg/rmcloud/auth.go
      Note: CreateApiCtx + forceSchemaV4 (sibling bug workaround)
    - Path: pkg/rmcloud/logtransport.go
      Note: debug HTTP logging transport
ExternalSources:
    - https://github.com/go-go-golems/remarquee/issues/23
    - https://github.com/ddvk/rmapi/pull/77
Summary: 'Chronological investigation of issue #23: all cloud writes fail with 400 ''invalid root schema'' because the pinned rmapi fork serializes the root index in slice order; the cloud now requires entries sorted by document ID.'
LastUpdated: 2026-08-17T14:00:00-04:00
WhatFor: Resume point for RMQ-0023; records what was tried, what failed, and the verified fix.
WhenToUse: Read before resuming work on the root-index sorting bug or the rmapi dependency pin.
---




# Investigation diary

## Goal

Record the investigation and fix for remarquee issue #23: every reMarkable cloud *write* fails with HTTP 400 `{"message":"invalid root schema"}` since 2026-08-17, while reads keep working.

## Step 1: Scope clarification (DAG content was a mis-paste)

**User prompt (verbatim):** "Create a detailed analysis / design / implementation guide that is for a new intern ... Store in the ticket and then upload to remarkable." (followed by a long treatise on an offline-transaction DAG-IR DSL, then later: "Sorry that was a miscopy paste on my part. Forget everything about the DAG.")

**Assistant interpretation:** Produce an intern-level guide for issue #23 in docmgr, then upload to reMarkable; the DAG content is unrelated and should be ignored.

**Inferred user intent:** A grounded, file-anchored design doc for the actual issue #23 bug, delivered through the standard docmgr + reMarkable workflow.

### What I did
- Read issue #23 via `gh issue view 23 --repo go-go-golems/remarquee`.
- Explored the repo: `pkg/rmcloud/*`, `cmd/remarquee/cmds/{cloud,upload}/*`, `pkg/rmdsl/*` (confirmed the remarquee DSL is a reMarkable v6 *document-rendering* DSL, unrelated to the pasted distributed-transaction DSL).
- Confirmed the pasted treatise has no connection to remarquee and discarded it.

### Why
The deliverable must be evidence-based; the only relevant subject is the issue #23 cloud-write bug.

### What worked
- Identified the mismatch before writing a large wrong-topic deliverable.

### What didn't work
- `gh issue list --repo ... --json` piped to python failed with a GraphQL deprecation warning breaking JSON; switched to `gh issue view --json` for a single issue, which worked.

### What I learned
- remarquee has two distinct "DSL" notions: `pkg/rmdsl` (rendering fixtures → `.rm` archives) and the *sync root-index schema* (the actual subject). "Schema" in this ticket always means the latter.

### What was tricky to build
- Distinguishing the two meanings of "DSL"/"schema" in the repo to avoid documenting the wrong subsystem.

### What warrants a second pair of eyes
- None for this step; it was pure scoping.

### What should be done in the future
- N/A

### Code review instructions
- N/A (no code changed in this step).

### Technical details
- Issue #23 summary: since 2026-08-17, `upload md`, `upload bundle`, `cloud mkdir` (and `mv`/`rm`) fail with 400 `invalid root schema`; reads unaffected.

---

## Step 2: Verify the bug is still live (not already fixed in main)

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Before writing the doc, confirm the bug is real at the tip of main.

**Inferred user intent:** Avoid documenting a solved bug.

### What I did
- `git fetch origin`; compared `HEAD`, local `main`, `origin/main`.
- Checked `go.mod` rmapi pin at HEAD vs origin/main.
- Checked ddvk/rmapi#77 merge state and the marcobarcelos fork's latest commit.

### Why
The user hypothesized the bug might be fixed in main; verifying protects against wasted work.

### What worked
- `HEAD == origin/main == 183a9d3` exactly. The branch `task/remarquee-fix-root-schema` is at origin/main's tip.
- `go.mod:22` still pins `marcobarcelos/rmapi v0.0.0-20260518211546-a0d079936d46` (May 18, pre-fix).
- ddvk/rmapi#77 is still `OPEN` (`mergedAt: null`); marcobarcelos fork not updated past May 18.
- Confirmed in the pinned fork's `tree.go`: only `sort.Slice` at line 345, inside `Mirror()` (read path); `IndexReader()` (write path) has no sort.

### What didn't work
- Nothing; the verification was clean.

### What I learned
- The bug is genuinely unfixed at main's tip. The 1-commit difference between local `main` and `origin/main` was local `main` being *stale* (missing the PR#22 merge), not a fix.

### What was tricky to build
- Interpreting `git rev-list --left-right --count origin/main...HEAD` returning `0 0` correctly (means no divergence, not "nothing happened").

### What warrants a second pair of eyes
- The claim "HEAD == origin/main" — re-run `git rev-parse HEAD origin/main` to confirm identical SHAs.

### What should be done in the future
- N/A

### Code review instructions
- `git rev-parse HEAD origin/main` (must match).

### Technical details
- Local main was 1 behind origin/main (`183a9d3` Merge PR #22).

---

## Step 3: Create the ticket and write the intern design doc

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Create docmgr ticket RMQ-0023 and author the full intern-level guide.

### What I did
- `docmgr ticket create-ticket --ticket RMQ-0023 --title "Fix root index sorting: cloud writes fail with 400 'invalid root schema'" --topics backend,rmcloud,upload,dependency`.
- `docmgr doc add --ticket RMQ-0023 --doc-type design-doc --title "Root index sorting bug: analysis, design, and implementation guide"`.
- `docmgr doc add --ticket RMQ-0023 --doc-type reference --title "Investigation diary"`.
- Read the full rmapi fork write path: `api/sync15/{tree,blobdoc,common,apictx,blobstorage,entry}.go`.
- Read the remarquee write surface: `cmd/remarquee/cmds/cloud/{mkdir,rm}.go`, `cmd/remarquee/cmds/upload/{md,upload_helpers}.go`, `pkg/rmcloud/{auth,logtransport,dirs,download}.go`.
- Fetched the ddvk#77 diff via `gh pr diff 77 --repo ddvk/rmapi`.
- Wrote the ~47KB design doc with prose, ASCII diagrams, pseudocode, line-anchored file/API references, 3 fix options, a 7-step implementation guide, risks, glossary, references.

### Why
The doc must let a new intern understand the whole cloud-sync stack and ship the fix without external context.

### What worked
- The doc grounded every claim in real files; the root-cause section maps each claim to evidence.

### What didn't work
- (Token budget hit before diary/relate/doctor/upload in the first attempt; resumed this session.)

### What I learned
- The subtle asymmetry: `HashEntries()` (common.go) *sorts* document entries before hashing (document indices are canonical), but `HashTree.IndexReader()` serializes root entries in slice order and `Rehash()` hashes *that* — so the root path lacks the canonicalization the document path has. That one-line asymmetry is the whole bug.

### What was tricky to build
- Explaining *why* "sort in the mutators" is the wrong fix (fragile to future mutators; risks hash/body desync) vs "sort at the serializer" (correct by construction).

### What warrants a second pair of eyes
- The claim that `Rehash()` and `IndexReader()` read the same bytes — verify `Rehash()` calls `IndexReader()` (it does, fork `tree.go`).

### What should be done in the future
- Add a remarquee-level guard test once the fork pin stabilizes.

### Code review instructions
- Read `design-doc/01-...md` §3.3 (write path), §3.5 (the asymmetry), §5 (why not mutators), §6 (fix options).

### Technical details
- Design doc path: `ttmp/2026/08/17/RMQ-0023--.../design-doc/01-root-index-sorting-bug-analysis-design-and-implementation-guide.md`.

---

## Step 4: Apply the fix (dependency bump)

**User prompt (verbatim):** "ok, do it." (after I recommended pinning the ddvk#77 PR head)

**Assistant interpretation:** Apply the go.mod replace change, tidy, build, smoke-test, commit.

### What I did
- `go mod edit -replace=...=github.com/ddvk/rmapi@f295d5466978954edabaae4c730d50d03740cfac` → failed: Go needs a pseudo-version, not a raw SHA.
- Fetched commit date via `gh api repos/ddvk/rmapi/commits/<sha>` → `2026-08-17T15:47:36Z` → pseudo-version `v0.0.0-20260817154736-f295d5466978`.
- Set that pseudo-version → `go mod tidy` failed: "unknown revision" — the commit lives in the contributor's fork `FNStudios-NI/rmapi`, not `ddvk/rmapi` (cross-repo PR).
- Verified `FNStudios-NI/rmapi` declares `module github.com/juruen/rmapi` (drop-in compatible), has branch `fix/root-index-sort` at that SHA, and `tree.go` contains `sort.Slice(t.Docs, ...)`.
- Repointed replace to `github.com/FNStudios-NI/rmapi v0.0.0-20260817154736-f295d5466978`.
- `go mod tidy` → success. `go build -o /tmp/remarquee-fixed ./cmd/remarquee` → success (75MB binary).
- Verified sort present in pinned source: `tree.go:178` (`sort.Slice(t.Docs, ...)`), `blobdoc.go:117` (`sort.Slice(d.Files, ...)`); v4-always behavior preserved at `tree.go:157`.

### Why
A one-line dependency bump with zero application code change is the minimal, correct fix; it lives where the bug lives.

### What worked
- Live smoke tests (exact issue #23 repro):
  - `cloud mkdir /ai/2026/08/17 --non-interactive` — was 400, now `exit 0` (dir created, confirmed by read).
  - Second `Add` in same session: `cloud mkdir /ai/2026/08/17/rmq-0023-probe` — `exit 0`.
  - `Remove` (swap-delete path): `cloud rm .../rmq-0023-probe --yes` — `exit 0`.
  - Cleanup `cloud rm /ai/2026/08/17 --yes` — `exit 0`; read confirms no trace.
- `go vet ./pkg/rmcloud/... ./cmd/remarquee/cmds/{cloud,upload}/...` — clean.
- `go test ./pkg/rmcloud/... ./cmd/remarquee/cmds/upload/...` — PASS.
- Committed as `d08e2e9` (only `go.mod`, `go.sum`, `.git-commit-message.yaml`).

### What didn't work
- `go build ./...` fails on `cmd/remarquee-ui/embed.go` (`frontend/dist` missing) — **pre-existing, unrelated** (web UI needs a pnpm build not present in this checkout). The `cmd/remarquee` CLI builds fine.
- `cloud mkdir /ai/2026/08/17/rmq-0023-probe` as the *first* test failed with "directory doesn't exist" because parent `/ai/2026/08/17` didn't exist yet and `cloud mkdir` is non-recursive; switched to creating `/ai/2026/08/17` first (parent `/ai/2026/08` exists).

### What I learned
- Cross-repo PRs: the PR head commit is in the contributor's fork, not the base repo. Go modules fetch from the repo named in the `replace`, so for a cross-repo PR you must pin the contributor's fork (or wait for merge). proxy.golang.org caches the pseudo-version permanently, so an unmerged PR head pinned by SHA is still reproducible for others.
- `go mod edit -replace=...=@<40-char-sha>` is rejected; you need the `v0.0.0-YYYYMMDDHHMMSS-short12` pseudo-version form. Get the UTC timestamp via `gh api repos/.../commits/<sha>`.

### What was tricky to build
- Discovering the commit was in `FNStudios-NI/rmapi`, not `ddvk/rmapi` — the `gh pr view` `headRepositoryOwner` field revealed it. Naively pinning `ddvk/rmapi` looked correct (the PR is *against* ddvk) but the commit isn't fetchable from there.

### What warrants a second pair of eyes
- The pseudo-version timestamp: `2026-08-17T15:47:36Z` → `20260817154736`. An off-by-one digit breaks the pin.
- That `FNStudios-NI/rmapi` doesn't diverge from `marcobarcelos/rmapi` in ways we depend on beyond the v4 behavior (confirmed v4-always present).

### What should be done in the future
- TODO(#23): once ddvk/rmapi#77 merges to ddvk master, re-point the `replace` at the merge commit (or a ddvk tag) to track upstream and drop the contributor-fork dependency.

### Code review instructions
- `grep -n "replace.*rmapi" go.mod` → must show `FNStudios-NI/rmapi v0.0.0-20260817154736-f295d5466978`.
- `git show d08e2e9 --stat` → only `go.mod`, `go.sum`, `.git-commit-message.yaml`.
- Reproduce the live test: `cloud mkdir /ai/<existing-parent>/<new>` then `cloud rm`.

### Technical details
- Pseudo-version: `v0.0.0-20260817154736-f295d5466978`.
- Commit SHA: `f295d5466978954edabaae4c730d50d03740cfac` (head of ddvk/rmapi#77, branch `fix/root-index-sort` in `FNStudios-NI/rmapi`).
- Commit `d08e2e9` on branch `task/remarquee-fix-root-schema`.
