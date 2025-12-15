# Changelog

## 2025-12-14

- Initial workspace created


## 2025-12-14

Defined RMQ-0002 scope (cloud-only remarquee CLI), linked RMQ-0001 implementation guide/design doc, and seeded an initial task list for module setup + one-file-per-command cloud verbs.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/14/RMQ-0002--implement-remarquee-cloud-cli/index.md — Added scope + links to RMQ-0001 playbook/design
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/14/RMQ-0002--implement-remarquee-cloud-cli/tasks.md — Seeded tasks for cloud CLI implementation


## 2025-12-14

Implemented initial remarquee CLI skeleton (Go submodule + cmd/remarquee) and first status command; fixed local glazed resolution for go run.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee/cmds/status.go — First command
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee/main.go — Root command wiring
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/go.mod — Added submodule + replace to local glazed
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/14/RMQ-0002--implement-remarquee-cloud-cli/reference/01-diary.md — Diary Step 1


## 2025-12-14

Step 1: initialized remarquee Go submodule + CLI skeleton and implemented first command (status). (commit aa44dec22ccaedd4ef7e02cbc32aa2d36dcfb9db)

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee/cmds/status.go — First command (commit aa44dec...)
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee/main.go — Root Cobra wiring (commit aa44dec...)
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/go.mod — Module definition (commit aa44dec...)


## 2025-12-14

Step 2: implemented rmapi-backed cloud CLI group and first verb `remarquee cloud refresh` with Glazed structured output. (commit 37b0012a81366a492e5439b4512a61081a29839f)

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee/cmds/cloud/refresh.go — Refresh command implementation (commit 37b0012...)
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee/cmds/cloud/root.go — Cloud command group wiring (commit 37b0012...)
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee/main.go — Root now registers cloud group (commit 37b0012...)
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/14/RMQ-0002--implement-remarquee-cloud-cli/reference/01-diary.md — Diary Step 2


## 2025-12-14

Step 3: wired Glazed help system at the remarquee root and added an embedded tutorial/playbook for adding new Glazed CLI commands. (commit daad86289b7658eb534e16bfc39637751b3e6e1e)

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee/main.go — Root now registers help system (commit daad862...)
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/doc/doc.go — Embedded doc loader (commit daad862...)
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/doc/tutorials/01-adding-a-glazed-command-to-remarquee.md — New tutorial/playbook page (commit daad862...)
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/14/RMQ-0002--implement-remarquee-cloud-cli/reference/01-diary.md — Diary Step 3


## 2025-12-14

Step 4: implemented rmapi-backed  and  with Glazed structured output; refactored shared rmapi bootstrap helper. (commit df3d3b2d34c84b86f6dac8f5151693c3bc162add)

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee/cmds/cloud/ls.go — Cloud ls (commit df3d3b2...)
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee/cmds/cloud/rmapi.go — Shared bootstrap (commit df3d3b2...)
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee/cmds/cloud/stat.go — Cloud stat (commit df3d3b2...)
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/14/RMQ-0002--implement-remarquee-cloud-cli/reference/01-diary.md — Diary Step 4

