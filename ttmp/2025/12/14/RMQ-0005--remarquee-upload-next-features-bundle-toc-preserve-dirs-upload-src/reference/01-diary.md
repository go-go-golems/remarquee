---
Title: Diary
Ticket: RMQ-0005
Status: active
Topics:
    - backend
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/remarquee/cmds/upload/md.go
      Note: Baseline uploader to extend (bundle + preserve-dirs + src)
    - Path: pkg/mdpdf/pandoc.go
      Note: Pandoc runner to extend (ToC + highlight options)
    - Path: pkg/rmcloud/dirs.go
      Note: Remote mkdir -p used for preserve-dirs
ExternalSources: []
Summary: "Implementation diary for RMQ-0005 (bundle/ToC, preserve-dirs mirroring, syntax-highlighted source uploads)."
LastUpdated: 2025-12-15T00:00:00.000000000-05:00
---

# Diary

## Goal

Track RMQ-0005 implementation as a narrative (what changed, why, what worked, what failed, and what to do next), with enough detail that a new contributor can continue mid-stream.

Repo root (git + Go module): `/home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee`

## Step 1: Start Phase 1 (bundle multiple markdowns into one PDF with ToC)

Outcome: added `remarquee upload bundle` to collate multiple markdown inputs into **one PDF** with a clickable **ToC** (pandoc `--toc`) and upload it as a single document.

**Commit (code):** c31675371b0283df8557f0143c13a7607dba4003 — "remarquee: add upload bundle (pandoc ToC)"

### What I did
- Added `remarquee upload bundle`:
  - `cmd/remarquee/cmds/upload/bundle.go`
  - supports `--name`, `--toc-depth`, `--date`, `--remote-dir`, `--dry-run`, `--pdf-only`, `--output-dir`, `--force`, `--non-interactive`, `--reauth`
- Implemented deterministic bundle ordering:
  - explicit files preserve user-given order
  - directories recurse and sort by relative path (case-insensitive)
- Implemented wrapper markdown generation (stable section headings + page breaks):
  - `pkg/mdpdf/bundle.go` (`BuildBundleMarkdown`)
- Extended pandoc runner with ToC/highlight knobs (ToC used by bundle; highlight to be used by `upload src`):
  - `pkg/mdpdf/pandoc.go` (`TOC`, `TOCDepth`, `HighlightStyle`, `Listings`)
- Added unit tests:
  - `cmd/remarquee/cmds/upload/bundle_test.go` (ordering rules)
  - `pkg/mdpdf/bundle_test.go` (frontmatter strip + headings + pagebreak)

### What worked
- `go test ./... -count=1` passes after the change.

### Next
- Add a smoke test script for `upload bundle` (real upload + `cloud ls` verification).
- Decide/implement default ToC depth behavior and documentation.

