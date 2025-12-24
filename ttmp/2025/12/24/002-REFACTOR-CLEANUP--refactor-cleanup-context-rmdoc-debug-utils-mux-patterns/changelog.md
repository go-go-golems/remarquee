# Changelog

## 2025-12-24

- Initial workspace created


## 2025-12-24

Step 2: Replace context.Background() with cobra/request context in CLI+UI (commit 4b73281)

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee-ui/api/render.go — Handlers now use r.Context()
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee/cmds/rmdoc/inspect.go — OpenFile now uses cmd context

