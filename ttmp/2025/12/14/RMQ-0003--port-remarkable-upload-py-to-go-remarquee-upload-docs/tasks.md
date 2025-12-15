# Tasks

## TODO

- [ ] Add tasks here

- [ ] Factor rmapi bootstrap into reusable package (pkg/rmcloud): move createApiCtx out of cmd/remarquee/cmds/cloud/rmapi.go
- [ ] Add new command group: cmd/remarquee/cmds/upload/root.go (remarquee upload)
- [ ] Implement remarquee upload md: preprocess markdown (strip docmgr frontmatter + list normalization), run pandoc/xelatex, and upload via rmapi
- [ ] Implement recursive remote mkdir helper (mkdir -p) for /ai/YYYY/MM/DD/... and mirrored ticket subdirs
- [ ] Add embedded help docs for upload commands (pkg/doc/upload/*) and update pkg/doc/doc.go embed patterns
- [ ] Add unit tests for preprocessing + date inference + remote path computation
