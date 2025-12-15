# Changelog

## 2025-12-15

- Initial workspace created


## 2025-12-15

Phase 0: Go backend scaffold + test documents (commit 8855874)

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee-ui/Makefile — Build targets for dev/prod modes
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee-ui/main.go — HTTP server with /api/test-documents endpoint
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee-ui/testdata/test-documents.json — Test document manifest


## 2025-12-15

Phase 1: Core API endpoints (commit c107adb)

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee-ui/api/inspect.go — GET /api/document/:id/inspect endpoint
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee-ui/api/outputs.go — GET /api/outputs/:filename endpoint
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee-ui/api/render.go — POST /api/render/background and /api/render/legacy endpoints
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/rmdoc/types.go — Added String() methods for JSON serialization


## 2025-12-15

Phase 0-5 complete: remarquee-ui web validation tool fully implemented (Go backend + React frontend + validation persistence + production build)

