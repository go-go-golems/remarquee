---
Title: P1 remediation implementation diary
Ticket: RMQ-0025
Status: active
Topics:
    - go
    - remarkable
    - remarquee
    - web
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /Users/manuel.odendahl/code/go-go-golems/remarquee/pkg/rmfiles/manage.go
      Note: F11 download staging path and attachment name sanitizer
    - Path: /Users/manuel.odendahl/code/go-go-golems/remarquee/cmd/remarquee/cmds/serve/handlers.go
      Note: F11 Content-Disposition via mime.FormatMediaType
    - Path: /Users/manuel.odendahl/code/go-go-golems/remarquee/pkg/rmfiles/service_test.go
      Note: F11 staging path and sanitizer regression tests
    - Path: /Users/manuel.odendahl/code/go-go-golems/remarquee/cmd/remarquee/cmds/serve/server_test.go
      Note: F11 attachment header parsing regression test
ExternalSources: []
Summary: Serial P1 remediation implementation diary for RMQ-0025 review findings; pass 1 covers F11 safe download staging and attachment filenames.
LastUpdated: 2026-10-04T13:05:00-04:00
WhatFor: Chronological investigation diary for P1 remediation passes; each pass appends a numbered step.
WhenToUse: Resume point for subsequent P1 passes (F01, F05, F02, F03) and for parent review of implemented fixes.
---

# P1 remediation implementation diary

## Goal

Implement the P1 review findings from
`analysis/01-pr-29-intern-facing-architecture-design-and-implementation-review.md`
in serial, exclusive component-owner passes. This diary records each pass with
the full investigation format, preserving exact failures and evidence.

## Context

Ticket RMQ-0025 (reMarkable files web UI). Parent orchestrator assigns one
component-owner pass at a time; shared seams transfer sequentially. Branch
`task/rmq-0025-serve-files-ui`, initial HEAD `22dda35`. Historical
implementation history stays in `reference/01-implementation-diary.md`; this
document is separate and appends only.

## Step 1: F11 — safe download staging filename and sanitized attachment header

Implemented the F11 correction: the download staging destination inside the
private per-download temp directory is now the fixed name `document.rmdoc`,
independent of any cloud display name, and the suggested filename returned to
the HTTP layer is produced by a new `sanitizeAttachmentName` helper that
strips control characters and replaces path separators and quotes. The serve
handler now builds `Content-Disposition` with `mime.FormatMediaType`
(RFC 2231 encoding) plus a defensive fallback, replacing the old manual
quote-stripping that left traversal sequences and control characters in the
header. Regression tests simulate cloud metadata imported from other clients
(the UI's own slash validation is irrelevant here) and prove the FetchDocument
destination base name is fixed, the staging directory is cleaned up, and the
header parses with no separators, quotes, or control characters.

### Prompt Context

**User prompt (verbatim):** "launch a lunaroute/glm-5.3-flash high thinking agent to taackle the P1 issues, telling them to keep a separate diary (and looking up the skil format),c ommit at appropriate intervals, and review their work when they are done."

Follow-up instruction (abridged excerpt from the parent's runtime message, not the full verbatim text): "implement ONLY review finding F11 (safe download staging and attachment filenames), then STOP for parent review. Do not implement F01/F05/F02/F03 yet ... Use a fixed internal filename inside private tempDir, independent of all external cloud names; build a separately sanitized suggested attachment basename and use mime.FormatMediaType for the header. ... Add focused regression tests in existing service/handler test files proving traversal-shaped cloud names cannot influence FetchDocument destination, temp files are cleaned up, suggested names are safe, and attachment headers parse correctly with quotes/control characters/Unicode. Tests must simulate names imported from other cloud clients, not rely on existing UI slash validation."

**Assistant interpretation:** F11 only: make the local download staging path
immune to cloud display metadata and produce a standards-compliant,
injection-safe `Content-Disposition` suggestion, with focused tests on both
the service and the HTTP layer.
**Inferred user intent:** Close the path-trust and header-formatting gap
before any other P1 work; hand a small, independently reviewable diff back to
the parent.
**Commit (code):** a693e5b — "🚑 fix(serve): stage downloads at a fixed internal path and sanitize attachment names (RMQ-0025 F11)"

### What I did

- `pkg/rmfiles/manage.go`: added `fixedDownloadStagingName = "document.rmdoc"`
  constant; `Download` now stages into `filepath.Join(tmpDir, fixedDownloadStagingName)`
  and returns `sanitizeAttachmentName(node.Name()) + ".rmdoc"`. Added
  `sanitizeAttachmentName`: drops `unicode.IsControl` runes, replaces `/`,
  `\`, `"` with `_`, preserves other runes (spaces, Unicode), and falls back
  to `"document"` for empty, all-whitespace, or pure dot results.
- `cmd/remarquee/cmds/serve/handlers.go`: `handleDownload` now uses
  `mime.FormatMediaType("attachment", map[string]string{"filename": name})`,
  falling back to `attachment; filename="document.rmdoc"` if formatting ever
  yields an empty string. Removed the manual `strings.ReplaceAll(name, "\"", "")`.
- `pkg/rmfiles/service_test.go`: `fakeApiCtx.FetchDocument` records
  `lastFetchPath`; new `TestDownloadStagingPathIsFixed` (adversarial name
  `..\..\evil "quoted"\x07/slash`: fixed staging base name, staging dir removed
  after the call, suggestion has no separators/quotes/control chars, normal
  name still round-trips as `Project Plan.rmdoc`) and new table-driven
  `TestSanitizeAttachmentName`.
- `cmd/remarquee/cmds/serve/server_test.go`: new
  `TestDownloadAttachmentHeaderIsSanitized` builds a server around a fake
  document with an adversarial name and asserts `mime.ParseMediaType` accepts
  the header, the type is `attachment`, the filename parameter contains no
  separators/quotes/control characters, and the raw value has no CR/LF.

### Why

Cloud names are external metadata not validated by this UI (other clients may
have created them), so they were reaching both a local filesystem path
(`filepath.Join(tmpDir, node.Name()+".rmdoc")`) and an HTTP header assembled
by naive quote-stripping. The reviewed F11 evidence was source-confirmed; the
pinned rmapi `FetchDocument` writes to whatever destination it is given, so
the destination must be constructed locally from trusted constants only.
Considered formatting the header only in the handler, but
`mime.FormatMediaType` provably passes `/`, `\`, and `..` through into the
header (see What I learned), so sanitization must happen before the parameter
is formatted; keeping it in the service also keeps the service's suggested
filename contract safe for any future caller.

### What worked

- `go test ./pkg/rmfiles ./cmd/remarquee/cmds/serve -run 'TestDownload|TestSanitizeAttachmentName' -v`:
  all four targeted tests pass (`TestDownloadEndpoint`,
  `TestDownloadStagingPathIsFixed`, `TestDownloadAttachmentHeaderIsSanitized`,
  `TestSanitizeAttachmentName`).
- Full gate: `go test ./pkg/rmfiles ./cmd/remarquee/cmds/serve` → ok (both),
  `go test -race` on both packages → ok, `go build` and `go vet` clean,
  `gofmt -l pkg cmd` empty, `git diff --check` empty.

### What didn't work

- First probe of `mime.FormatMediaType` (scratch program in /tmp) showed it
  does **not** neutralize path separators: `"../../evil"` formats to
  `attachment; filename="../../evil"` and parses back unchanged; `"ctl\x07x"`
  is RFC 2231-encoded (`filename*=utf-8''ctl%07x`) and still parses back to a
  control character. This killed the idea of relying on `FormatMediaType`
  alone and confirmed the sanitizer must run before formatting.
- First `TestDownloadStagingPathIsFixed` draft used a bare `newFake()` with
  only the evil document, so the "normal name round-trips" assertion failed
  with `Project Plan not found in fixture`; fixed by building on the standard
  `tree(t)` fixture and its returned `docID`. Initial table expected `"/"` to
  sanitize to `"document"`; actual contract yields `"_"` (a safe single
  component), so the table was corrected to match the implemented contract.

### What I learned

- `mime.FormatMediaType` applies RFC 2231 encoding for non-token characters
  but does not treat `/`, `\`, or `..` as unsafe; header-level formatting is
  not a path-safety mechanism.
- The pinned dependency (github.com/FNStudios-NI/rmapi, replaced in go.mod
  for github.com/juruen/rmapi v0.0.25) `FetchDocument(docId, dstPath)` writes
  to `dstPath` as given; destination trust is entirely the caller's job.
- The existing per-call `os.MkdirTemp("", "rmfiles-dl-")` plus
  `defer os.RemoveAll` is already private and cleaned up, so the only change
  needed inside it was the fixed base name.

### What was tricky to build

Defining the sanitizer's degenerate-name behavior: `".."` and `"."` remain
hazardous as *whole* suggested filenames even without separators, and
all-whitespace names are useless; all three now fall back to `"document"`.
Single separators sanitize to `"_"` rather than the fallback, which is a
deliberate minimal contract (safe component, no overreach). Control
characters are dropped (not replaced) because any replacement inside the
value is invisible in the header either way.

### What warrants a second pair of eyes

- Whether the fallback `attachment; filename="document.rmdoc"` path in
  `handleDownload` is reachable in practice (it should not be, given the
  sanitizer's guarantees) and whether parent prefers returning an error there.
- The sanitizer preserves Unicode and spaces by design; the review's F11
  acceptance mentions "Unicode cannot affect the output path and produce
  valid download headers" — satisfied via RFC 2231 `filename*=utf-8''...`
  encoding, which browsers are expected to decode. Confirm this is the
  desired header shape.
- `Content-Type` remains `application/octet-stream` (unchanged scope).

### What should be done in the future

- Remaining P1 passes: F01 (descendant-first recursive delete), F05 (live-ID
  integrity and partial-failure reconciliation), then F02/F03 per parent
  assignment. Reuse this diary; do not append to
  `reference/01-implementation-diary.md`.
- Optional later polish (P2, out of scope here): streaming downloads and
  shorter lock hold (review §8.4).

### Code review instructions

- Start at `pkg/rmfiles/manage.go` (`fixedDownloadStagingName`, `Download`,
  `sanitizeAttachmentName`) and `cmd/remarquee/cmds/serve/handlers.go`
  (`handleDownload`).
- Validate with:
  `go test ./pkg/rmfiles ./cmd/remarquee/cmds/serve -run 'TestDownload|TestSanitizeAttachmentName' -v`
  then `go test -race ./pkg/rmfiles ./cmd/remarquee/cmds/serve`.

### Technical details

- Staging contract: per-call private temp dir + fixed base name; cloud name
  never touches a local path.
- Suggested filename contract: `sanitizeAttachmentName(name) + ".rmdoc"`;
  control chars dropped, `/` `\` `"` → `_`, fallback `document`.
- Header contract: `mime.FormatMediaType("attachment", {"filename": ...})`;
  RFC 2231 for non-ASCII/spaces; quoted-string escaping for quotes.
- Evidence note: this step's failures are exact (FormatMediaType passthrough
  probe output; fixture-not-found test failure); no real cloud transport was
  used, all tests run against in-memory fakes.

## Step 2: F11 follow-up — review repairs (Unicode/CR-LF header table, failure cleanup, metadata hygiene)

Small validation follow-up after parent review of a693e5b. No production code
changed; the fixed staging path and sanitizer were accepted provisionally.
Added table-driven handler cases (Unicode, spaces, CR/LF, quotes) asserting the
parsed filename parameter is the intended sanitized `.rmdoc` suggestion with no
raw CR/LF, added a service test proving the private staging directory is
removed when `FetchDocument` fails, fixed diary metadata to existing vocabulary
(Status `active`, dropped non-vocabulary `security` topic), linked this diary
from the ticket index Key Links, and recorded the F11 code commit hash plus
excerpt labeling for abridged parent instructions.

### Prompt Context

**User prompt (verbatim):** see Step 1.
**Assistant interpretation:** Repair the focused handoff: expand handler header
tests to an HTTP encoding round trip across Unicode/spaces/CR-LF/quotes, prove
temp cleanup on fetch failure, and fix vocabulary/metadata hygiene, without
rewriting accepted production code.
**Inferred user intent:** Make the F11 evidence independently verifiable at the
HTTP boundary and leave clean, doctor-clean ticket metadata before further P1
passes.
**Commit (code):** see the follow-up commit recorded in the final handoff (this
step's commit hash is added to the acceptance report; diary written just
before the commit).

### What I did

- `cmd/remarquee/cmds/serve/server_test.go`: converted
  `TestDownloadAttachmentHeaderIsSanitized` to table-driven cases
  (`..\..\evil "quoted"`+BEL+`/slash`, `unicode é中`, `sp ace.rmdoc`,
  `line\r\nbreak"q"`, `a"b`); each case builds a full server around a fake
  document with that exact cloud name and asserts `mime.ParseMediaType`
  success, `attachment` type, exact expected sanitized filename parameter
  (including RFC 2231 round trip for Unicode/spaces), no CR/LF in the raw
  header, and no separators/quotes/control characters in the parsed name.
- `pkg/rmfiles/service_test.go`: fake `FetchDocument` gained a `failFetch`
  injection flag; new `TestDownloadStagingCleanupOnFetchFailure` proves
  `Download` returns the error and the recorded staging directory no longer
  exists afterwards.
- Diary frontmatter: `Status: active` (existing vocabulary), removed unknown
  `security` topic; ticket `index.md` Key Links now links this diary.
- Recorded F11 code commit `a693e5b` in Step 1 and relabeled the abridged
  parent instruction as an excerpt, not full verbatim.

### Why

The prior handler test covered only one ASCII adversarial name, which does not
prove the RFC 2231 HTTP encoding round trip the F11 directive required; the
service-level Unicode unit test alone is not an HTTP round trip. Cleanup was
only proven on the success path; the failure path must not leak temp
directories either.

### What worked

- `go test ./pkg/rmfiles ./cmd/remarquee/cmds/serve -run 'TestDownload|TestSanitizeAttachmentName' -count=1` → ok, ok.
- `go test ./pkg/rmfiles ./cmd/remarquee/cmds/serve -count=1` → ok, ok;
  `go test -race -count=1` both packages → ok, ok.
- `go vet` both packages clean; `gofmt -l pkg cmd` empty; `git diff --check`
  clean; `docmgr doctor --ticket RMQ-0025` → all checks passed.

### What didn't work

- First version of the table's traversal case used a Go raw string with
  doubled backslashes (`` `..\\..\\evil \"quoted\"` `` inside backticks), so the
  actual cloud name contained two literal backslashes per separator and the
  expected sanitized name was wrong twice over
  (`got "..__..__evil __quoted___slash.rmdoc", want ".._.._evil _quoted_ _slash.rmdoc"`).
  Fixed by using single backslashes in the raw string (matching the service
  test fixture) and recomputing the expected name
  (`.._.._evil _quoted__slash.rmdoc`). Exact diagnostics preserved above.

### What I learned

- Go raw-string backslash doubling from a JSON-authored edit silently changes
  the fixture semantics; byte-level verification (`od -c`) of adversarial
  fixture literals is worth one command.
- `mime.ParseMediaType` decodes RFC 2231 `filename*=utf-8''...` back into the
  plain `filename` parameter, so the round trip can be asserted directly
  against the intended Unicode/space names.

### What warrants a second pair of eyes

- The CR/LF case relies on the sanitizer dropping control characters before
  header formatting; a future sanitizer change would surface here first.
- `TestDownloadStagingCleanupOnFetchFailure` asserts the staging dir is gone
  via `os.IsNotExist`; acceptable for the private temp dir, but it is not a
  proof that no other filesystem side effects occurred.

### What should be done in the future

- Continue P1 passes (F01, F05, then F02/F03 per parent assignment) in this
  diary; no P2 work.

### Code review instructions

- `go test ./cmd/remarquee/cmds/serve -run TestDownloadAttachmentHeaderIsSanitized -count=1 -v`
  and `go test ./pkg/rmfiles -run 'TestDownloadStaging' -count=1 -v`;
  `docmgr doctor --ticket RMQ-0025`.

### Technical details

- Handler table asserts exact parsed filenames: `.._.._evil _quoted__slash.rmdoc`,
  `unicode é中.rmdoc`, `sp ace.rmdoc.rmdoc`, `linebreak_q_.rmdoc`, `a_b.rmdoc`.
- Cleanup-on-failure contract: `os.MkdirTemp` + `defer os.RemoveAll` covers the
  `FetchDocument` error path; no staging file is written when fetch fails.
