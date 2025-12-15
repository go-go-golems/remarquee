# Tasks — RMQ-RMDOC-WEB-001

## Overview

Build `remarquee-ui`: an interactive web UI for validating `.rmdoc` parsing and PDF rendering with pre-prepared test fixtures.

**Design doc**: `../RMQ-0004--port-rmdoc-parsing-rendering-to-go-v3-v5-v6/design-doc/02-design-interactive-rmdoc-render-validation-ui.md`

## Phase 0 — Go backend scaffold + test documents

- [x] Create `cmd/remarquee-ui/main.go` with `net/http` server (dev/prod modes)
- [x] Create `cmd/remarquee-ui/Makefile` with dev/build/clean targets
- [x] Implement `/api/test-documents` handler (returns manifest)
- [x] Add `cmd/remarquee-ui/testdata/` directory with pre-prepared `.rmdoc` fixtures
- [x] Create `cmd/remarquee-ui/testdata/test-documents.json` manifest file
- [x] Smoke-test: `curl http://localhost:8080/api/test-documents`

## Phase 1 — Core API endpoints

- [x] Create `cmd/remarquee-ui/api/` package structure
- [x] Implement `GET /api/document/:id/inspect` (calls `pkg/rmdoc.OpenFile`)
- [x] Implement `POST /api/render/background` (calls `pkg/rmdoc/render.BuildBackgroundPDF`)
- [x] Implement `POST /api/render/legacy` (calls rmapi `PdfGenerator`)
- [x] Implement `GET /api/outputs/:filename` (serve PDFs from outputs/)
- [x] Add basic error handling and JSON responses
- [x] Smoke-test all endpoints with `curl`

## Phase 2 — React frontend scaffold

- [x] Init React + Vite + TypeScript in `cmd/remarquee-ui/frontend/`
- [x] Add `package.json` with React, Redux Toolkit, Vite dependencies
- [x] Configure `vite.config.ts` with proxy (`/api/*` → `http://localhost:8080`)
- [x] Add `tsconfig.json` with strict TypeScript settings
- [x] Setup Redux Toolkit store with 3 slices (documents, render, validation)
- [x] Create basic App layout (header, sidebar, main panel)
- [ ] Smoke-test: `npm run dev` and verify proxy works

## Phase 3 — Core UI components

- [x] Implement `DocumentSelector` component (list + select from test documents)
- [x] Implement `InspectPanel` component (display schema/pages table)
- [x] Implement `RenderActions` component (buttons for inspect/background/legacy)
- [x] Implement `PDFViewer` component (iframe or download link for outputs)
- [x] Implement `ValidationForm` component (PASS/FAIL radio + notes textarea + submit)
- [x] Wire all components to Redux slices
- [ ] Manual UI smoke-test: select document → inspect → render → validate

## Phase 4 — Validation persistence

- [ ] Implement `POST /api/validation` handler in backend
- [ ] Backend: write validation session as JSON to `reference/validation/<timestamp>.json`
- [ ] Backend: write validation session as Markdown to `reference/validation/<timestamp>.md`
- [ ] Frontend: dispatch validation submission from `ValidationForm`
- [ ] Frontend: implement `GET /api/validation/history` to fetch past sessions
- [ ] UI: display validation history list (collapsible)
- [ ] Manual test: submit validation → verify files created in ticket

## Phase 5 — Production build + embed

- [ ] Create `cmd/remarquee-ui/embed.go` with `//go:embed frontend/dist`
- [ ] Update `main.go` to serve embedded assets in prod mode (no `--dev` flag)
- [ ] Update Makefile `build` target: `npm run build` → `go build`
- [ ] Test single-binary deployment: `./remarquee-ui` (prod mode)
- [x] Add `.gitignore` entries: `frontend/dist`, `frontend/node_modules`, `outputs/`

## Acceptance Criteria

- [ ] Can select from pre-prepared test documents via web UI
- [ ] Can inspect a document and see schema + pages in a table
- [ ] Can build background PDF and view/download it
- [ ] Can render legacy PDF (V3/V5) and view/download it
- [ ] Can submit validation (PASS/FAIL + notes) and see it persisted to ticket
- [ ] Single `make build` produces a standalone binary with embedded frontend
- [ ] Dev mode (`make dev-backend` + `make dev-frontend`) works with hot reload

