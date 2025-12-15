# Tasks

## TODO

- [ ] Add tasks here

- [x] Create Go submodule at /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/go.mod with module github.com/go-go-golems/remarquee; add ./remarquee to root go.work
- [x] Rename remarquee/cmd/XXX -> remarquee/cmd/remarquee and remove/rename all placeholder XXX references
- [x] Scaffold Cobra root command + Glazed wiring for remarquee (follow pinocchio patterns: help system + logging init)
- [x] Implement cloud client bootstrap (rmapi AuthHttpCtx/ParseToken/CreateApiCtx) with non-interactive option
- [x] Implement cloud command group (remarquee cloud) and one file per command in cmd/remarquee/cmds/cloud/*.go
- [x] Implement cloud commands: refresh, ls, stat (with --with-glaze-output structured output)
- [ ] Implement cloud commands: get, put, mkdir
- [ ] Implement cloud commands: mv, rm (safe defaults, explicit confirmation flag for destructive ops)
- [ ] Implement cloud command: find
- [ ] Implement cloud commands: account, version
- [ ] Add minimal tests / smoke checks (go test) and document manual validation commands in RMQ-0002 playbook or notes
