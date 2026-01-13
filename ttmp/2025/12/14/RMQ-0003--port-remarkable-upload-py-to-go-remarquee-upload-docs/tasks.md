# Tasks

## TODO

- [x] Add tasks here

- [x] Factor rmapi bootstrap into reusable package (pkg/rmcloud): move createApiCtx out of cmd/remarquee/cmds/cloud/rmapi.go
- [x] Add new command group: cmd/remarquee/cmds/upload/root.go (remarquee upload)
- [x] Implement remarquee upload md: accept markdown files and directories (directories recursively scanned for *.md), preprocess markdown (strip docmgr frontmatter + list normalization), run pandoc/xelatex, and upload via rmapi
- [x] Implement recursive remote mkdir helper (mkdir -p) for /ai/YYYY/MM/DD/... (and user-provided remote directories)
- [x] Add embedded help docs for upload commands (pkg/doc/upload/*) and update pkg/doc/doc.go embed patterns
- [x] Add unit tests for preprocessing + directory recursion + date formatting + remote path computation
