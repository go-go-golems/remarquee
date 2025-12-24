# Changelog

## 2025-12-24

- Initial workspace created


## 2025-12-24

Step 2: Replace context.Background() with cobra/request context in CLI+UI (commit 4b73281)

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee-ui/api/render.go — Handlers now use r.Context()
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee/cmds/rmdoc/inspect.go — OpenFile now uses cmd context


## 2025-12-24

Step 3: Honor ctx in rmdoc open/render + add cancellation tests (commit 48d822e)

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/rmdoc/open.go — Context-aware zip entry reads
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/rmdoc/render/background.go — Check ctx.Err() between pages


## 2025-12-24

Step 4: Document legacy rmapi cancellation semantics + precheck ctx (commit 20d6ce3)

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee-ui/api/render.go — Cannot interrupt rmapi generator; avoid starting if ctx canceled

