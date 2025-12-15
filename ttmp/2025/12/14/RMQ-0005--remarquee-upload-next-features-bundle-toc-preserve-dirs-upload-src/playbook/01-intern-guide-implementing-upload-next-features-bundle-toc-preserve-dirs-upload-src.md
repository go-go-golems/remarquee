---
Title: 'Intern guide: implementing upload next features (bundle/ToC, preserve dirs, upload src)'
Ticket: RMQ-0005
Status: active
Topics:
    - backend
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Intern guide for RMQ-0005: context + lay of the land for implementing upload bundle/ToC, preserve-dirs mirroring, and syntax-highlighted source uploads in remarquee."
LastUpdated: 2025-12-14T21:26:16.01337945-05:00
---

# Intern guide: implementing upload next features (bundle/ToC, preserve dirs, upload src)

## Purpose

This playbook is a “start here” guide for an intern joining RMQ-0005. It explains:
- what the repo does (at a high level),
- what we already shipped (`remarquee upload md`),
- what the next batch of features is (bundle/ToC, preserve-dirs mirroring, upload src with highlighting),
- which files and docs to read in what order,
- and how to run development + verification commands safely.

## High-level mental model

For this ticket, you only need this model:

- **Input**: local files (markdown or source code)
- **Transform**: to PDF using **pandoc + xelatex**
- **Upload**: to reMarkable cloud using **rmapi as a Go library**

The upload pipeline is intentionally straightforward: convert locally → upload PDFs.

## Environment Assumptions

You’re working in:
- **Repo root (git + Go module)**: `/home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee`

Notes:
- Treat `remarquee/` as the only repo root for this work (don’t run commands from the parent directory).
- Ticket docs live under `remarquee/ttmp/`.

You have installed:
- `pandoc`
- `xelatex` (TeX Live)

rmapi auth tokens are present (so `--non-interactive` works). Validate quickly:

```bash
cd /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee
go run ./cmd/remarquee cloud account --non-interactive
```

## What to read (in order)

1) **This ticket’s design doc (the spec)**
- `ttmp/2025/12/14/RMQ-0005--remarquee-upload-next-features-bundle-toc-preserve-dirs-upload-src/design-doc/01-upload-next-features-bundle-pdfs-w-toc-mirror-dirs-syntax-highlight-source.md`

2) **The current implementation you will extend**
- `cmd/remarquee/cmds/upload/md.go` (upload loop, file collection, remote dir resolution)
- `pkg/mdpdf/*` (frontmatter stripping, list normalization, pandoc runner)
- `pkg/rmcloud/*` (`CreateApiCtx`, `MkdirAll`)

3) **The embedded help docs (user-facing UX)**
- `pkg/doc/upload/01-remarquee-upload-getting-started.md`
- `pkg/doc/upload/02-remarquee-upload-reference.md`

4) **The existing smoke test script (how we verify quickly)**
- `ttmp/2025/12/14/RMQ-0003--port-remarkable-upload-py-to-go-remarquee-upload-docs/scripts/01-smoke-test-upload-md.sh`

## Commands

```bash
cd /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee

# Always run tests before/after changes
go test ./... -count=1

# Show current upload command UX
go run ./cmd/remarquee upload md --help

# Recommended: start with a dry-run into a sandbox folder
go run ./cmd/remarquee upload md --dry-run --remote-dir /ai/test/your-sandbox /abs/path/to/dir

# Real upload into sandbox
go run ./cmd/remarquee upload md --non-interactive --remote-dir /ai/test/your-sandbox /abs/path/to/dir

# Verify visibility in the cloud tree
go run ./cmd/remarquee cloud ls /ai/test/your-sandbox
```

## Exit Criteria

You’ve succeeded on RMQ-0005 when you can:
- implement at least one feature from the design doc (bundle, preserve-dirs, or upload src),
- run `go test ./... -count=1`,
- upload to a sandbox folder and verify via `remarquee cloud ls`,
- update docs (design-doc and/or diary) with what changed and how to validate it.

## Notes

- Prefer `/ai/test/...` as the remote base while developing so it’s obvious what’s test data.
- Be careful with `--force`: it deletes existing documents (and annotations) before uploading replacements.
