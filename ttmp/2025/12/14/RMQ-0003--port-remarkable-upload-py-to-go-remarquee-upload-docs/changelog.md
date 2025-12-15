# Changelog

## 2025-12-14

- Initial workspace created


## 2025-12-14

Created design-doc proposing Go port of remarkable_upload.py into remarquee (upload md): pandoc/xelatex conversion + rmapi-backed upload under /ai/YYYY/MM/DD/; seeded tasks and related key files.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/14/RMQ-0003--port-remarkable-upload-py-to-go-remarquee-upload-docs/design-doc/01-port-remarkable-upload-py-to-go-remarquee-upload-docs.md — Design proposal for the port


## 2025-12-14

Updated design scope: upload tool is now general-purpose (accepts only markdown files or directories scanned recursively for *.md); removed ticket-specific CLI concepts from the proposal.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/14/RMQ-0003--port-remarkable-upload-py-to-go-remarquee-upload-docs/design-doc/01-port-remarkable-upload-py-to-go-remarquee-upload-docs.md — Adjusted CLI spec to remove ticket logic


## 2025-12-14

Step 1: Add 'remarquee upload md' (general-purpose markdown uploader) and factor rmapi bootstrap (commit d70169e...)

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee/cmds/upload/md.go — New upload command implementation (commit d70169e...)
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/mdpdf/pandoc.go — Pandoc/xelatex integration (commit d70169e...)
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/rmcloud/auth.go — Shared rmapi CreateApiCtx helper (commit d70169e...)

