# Changelog

## 2026-10-04

- Initial workspace created

## 2026-10-04

Created RMQ-0025 and wrote the intern-facing design/implementation guide for a macOS-1 monochrome web UI to upload, manage, and search reMarkable cloud files. Applied the 'modern fonts only, no classic bitmap typeface' constraint. Added web/frontend/ui/react vocabulary topics.

### Related Files

- /Users/manuel.odendahl/code/go-go-golems/remarquee/ttmp/2026/10/04/RMQ-0025--remarkable-files-web-ui-macos-1-monochrome-upload-manage-and-find/design-doc/01-remarkable-files-web-ui-design-and-implementation-guide-for-a-new-intern.md — Primary design and implementation guide
- /Users/manuel.odendahl/code/go-go-golems/remarquee/ttmp/2026/10/04/RMQ-0025--remarkable-files-web-ui-macos-1-monochrome-upload-manage-and-find/reference/01-implementation-diary.md — Reconnaissance diary and evidence

## 2026-10-04

Recorded two user constraints in the guide: modern fonts only (classic Macintosh bitmap typeface explicitly excluded) and delivery as a remarquee serve subcommand rather than a separate binary. Updated architecture, code layout, packaging, tests, and the resolved open question.

## 2026-10-04

Uploaded the finalized design guide to reMarkable: OK: uploaded RMQ-0025_Remarkable_Files_Web_UI_-_Intern_Design_Guide.pdf -> /ai/2026/10/04/RMQ-0025

## 2026-10-04

Implemented the feature: pkg/rmfiles (single-context, mutex-serialized service with search index, manage ops, and upload jobs) and the remarquee serve subcommand with an embedded no-build macOS-1 monochrome frontend. Registered serve in cmd/remarquee/main.go. Added unit + handler tests; validated end-to-end against the live cloud (13661 docs, create folder + md upload + recursive delete). Commits f62e278, 0a692bb, 137d9dc.

## 2026-10-04

Added folder uploads with preserved directory structure and browser drag-and-drop. Backend accepts folder-relative paths (sanitized) and creates intermediate remote folders; frontend stages files from a folder picker or drag-and-drop and sends parallel paths. Verified live (nested md+pdf upload and recursive delete). Commits 30b228e, bef63ad, 76997c4.

## 2026-10-04

Moved the work onto branch task/rmq-0025-serve-files-ui, forked go-go-golems/remarquee to manuel-tulip/remarquee, pushed the branch, and opened PR #29 against origin main.

## 2026-10-04

Reviewed PR 29 at f2dd68c against the design and pinned rmapi; added a roughly 9100-word intern-facing architecture/API/implementation review with twelve prioritized findings, offline probes, validation receipts and open remediation tasks. Uploaded the review to /ai/2026/10/04/RMQ-0025 as RMQ-0025 PR29 Architecture and Implementation Review; fixes remain unimplemented.

### Related Files

- /Users/manuel.odendahl/code/go-go-golems/remarquee/ttmp/2026/10/04/RMQ-0025--remarkable-files-web-ui-macos-1-monochrome-upload-manage-and-find/analysis/01-pr-29-intern-facing-architecture-design-and-implementation-review.md — Evidence-backed review and remediation plan

## 2026-10-04

Created a second proposed design for the requested move directory browser, subtle directory text color, actual file-format column, non-wrapping dates and column sorting. Specifies independent picker state, authoritative metadata instead of extension guessing, comparator rules and acceptance tests. Design only: implementation tasks remain open; no new device upload requested.

### Related Files

- /Users/manuel.odendahl/code/go-go-golems/remarquee/ttmp/2026/10/04/RMQ-0025--remarkable-files-web-ui-macos-1-monochrome-upload-manage-and-find/design-doc/02-files-ui-additions-directory-picker-column-clarity-and-sorting.md — Second additions design

## 2026-10-04

F11 fix: download staging uses fixed internal filename (document.rmdoc) in private temp dir; suggested attachment names are sanitized (control chars dropped, separators/quotes replaced, dot-only fallback) and Content-Disposition is built with mime.FormatMediaType; regression tests at service and handler level

### Related Files

- /Users/manuel.odendahl/code/go-go-golems/remarquee/cmd/remarquee/cmds/serve/handlers.go — mime.FormatMediaType attachment header
- /Users/manuel.odendahl/code/go-go-golems/remarquee/pkg/rmfiles/manage.go — Fixed staging filename and sanitizeAttachmentName

## 2026-10-04

F01/F05 fix: Delete prevalidates and deduplicates live IDs, rejects root/trash and nonrecursive nonempty folders before writes, and executes a deterministic postorder plan removing every cloud record; service ID lookup requires edge reachability so deleted IDs/descendants fail; failures reindex, reconcile via Refresh, and return typed PartialDeleteError; HTTP reports 207 partial outcomes with deleted/remainingIds/failedId/reconciled; dependency-faithful HashTree/Rebuild tests added

### Related Files

- /Users/manuel.odendahl/code/go-go-golems/remarquee/cmd/remarquee/cmds/serve/handlers.go — 207 partial-delete reporting
- /Users/manuel.odendahl/code/go-go-golems/remarquee/pkg/rmfiles/errors.go — PartialDeleteError contract
- /Users/manuel.odendahl/code/go-go-golems/remarquee/pkg/rmfiles/manage.go — postorder delete plan and partial outcomes
- /Users/manuel.odendahl/code/go-go-golems/remarquee/pkg/rmfiles/service.go — edge-reachable ID lookup and reconcile
