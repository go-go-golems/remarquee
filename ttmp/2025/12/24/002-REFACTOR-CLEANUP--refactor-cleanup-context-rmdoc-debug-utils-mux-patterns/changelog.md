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


## 2025-12-24

Add ticket-local scripts/inspect-archive.sh for inspecting archives and .rm headers

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/24/002-REFACTOR-CLEANUP--refactor-cleanup-context-rmdoc-debug-utils-mux-patterns/scripts/inspect-archive.sh — Debug helper for upcoming pkg/rmdoc/debug extraction


## 2025-12-24

Step 6: Extract rmdoc archive introspection into pkg/rmdoc/debug + refactor UI internal_structure (commit 401d54b)

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee-ui/api/internal_structure.go — Remove duplicated zip/header code
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/rmdoc/debug/archive.go — Archive listing + .rm version sniff


## 2025-12-24

Step 7: Cleanup remarquee-ui routing + path parsing using stdlib ServeMux path variables + add routing tests (commit baeef41)

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee-ui/main.go — No suffix dispatch; use /api/document/{id}/...
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee-ui/main_test.go — Locks down 200/404/405/400 outcomes


## 2025-12-24

All tasks completed; ready for review


## 2026-01-12

Ticket closed

