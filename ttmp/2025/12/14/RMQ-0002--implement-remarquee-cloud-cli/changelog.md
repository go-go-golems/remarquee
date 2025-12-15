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

