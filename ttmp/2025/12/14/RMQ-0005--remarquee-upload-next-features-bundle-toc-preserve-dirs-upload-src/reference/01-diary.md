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

Planned outcome:
- A way to take **multiple markdown inputs** (files and/or directories) and produce **one PDF** with a **table of contents**.
- Upload that single PDF to a user-specified remote directory (same rmapi integration as `upload md`).

Next actions:
- Read the RMQ-0005 design doc and the current `upload md` implementation.
- Decide CLI surface (new subcommand vs flag) based on the design doc.
- Implement the bundle wrapper generation + pandoc invocation + upload.

