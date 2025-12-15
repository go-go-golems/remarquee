# Diary — remarquee-ui web validation tool implementation

## Goal

Implement `remarquee-ui`: an interactive **web UI** (React + Go) for validating `.rmdoc` parsing and PDF rendering with pre-prepared test fixtures. The UI enables rapid human-in-the-loop feedback on visual correctness (page ordering, duplicates, blanks, annotation alignment) that's hard to automate.

## Step 1: Bootstrap ticket + scaffold Phase 0 (Go backend + test documents)

This step creates the ticket workspace, populates tasks from the design doc, and begins Phase 0: the minimal Go HTTP server + test document manifest. The goal is to get a runnable backend that serves the test documents list via `/api/test-documents`, laying the foundation for the React frontend and core API endpoints.

### What I did

- Created ticket RMQ-RMDOC-WEB-001 with `docmgr ticket create-ticket`
- Created diary document with `docmgr doc add --doc-type reference`
- Populated `tasks.md` with 5 phases extracted from the design doc (Phase 0 through Phase 5)
- Created `cmd/remarquee-ui/` directory structure
- Implemented `cmd/remarquee-ui/main.go`: minimal HTTP server with `/api/test-documents` endpoint and dev/prod mode detection
- Created `cmd/remarquee-ui/Makefile` with dev/build/clean/install/test targets
- Copied test fixtures to `cmd/remarquee-ui/testdata/` (cPages fixture and legacy fixture)
- Created `cmd/remarquee-ui/testdata/test-documents.json` manifest describing the 2 fixtures

### Why

- **Phased approach**: Phase 0 gives us a smoke-testable backend before tackling React/Redux/Vite setup
- **Pre-prepared fixtures**: storing test documents with the tool makes validation reproducible and fast
- **Self-contained**: everything lives under `cmd/remarquee-ui/` per user requirement
- **Dev/prod modes**: `--dev` flag controls whether backend proxies to Vite or serves embedded assets

### What worked

- `docmgr ticket create-ticket` command created a clean ticket workspace with standard structure
- Copying existing `.rmdoc` fixtures from RMQ-0004 test runs gives us real-world test data immediately
- Simple `net/http` server with `http.ServeMux` is sufficient for the initial API

### What didn't work

- (No blockers yet; this is the initial scaffold)

### What I learned

- The ticket's `testdata/` directory is the natural place for fixtures (avoids coupling to external paths)
- Manifest JSON makes it easy to add metadata (description, schema type, expected page count) for each test document

### What was tricky to build

- N/A (straightforward scaffold so far)

### What warrants a second pair of eyes

- **Test document selection**: confirm the 2 initial fixtures (cPages PDF-backed, legacy notebook) cover the critical validation cases
- **Dev/prod mode detection**: verify `--dev` flag logic is simple and deterministic

### What should be done in the future

- Add more fixtures as we discover edge cases (e.g., inserted pages, deleted pages, redirection quirks)
- Consider adding a "fixture validation" test that ensures all paths in `test-documents.json` actually exist

### Code review instructions

- Start in `cmd/remarquee-ui/main.go`: verify server initialization, `/api/test-documents` handler, and dev/prod mode logic
- Check `cmd/remarquee-ui/Makefile`: verify targets are correct and self-contained
- Review `cmd/remarquee-ui/testdata/test-documents.json`: confirm fixture metadata is accurate
- Run: `cd cmd/remarquee-ui && make dev-backend`, then `curl http://localhost:8080/api/test-documents`

### Technical details

- **Server**: `net/http` with `http.ServeMux`, listens on `:8080` by default
- **Dev mode**: `--dev` flag sets a global boolean; in the future this will control asset serving logic
- **Test documents manifest**: JSON array of objects with `{id, name, path, description, schema, docType, expectedPages}`
- **Fixtures**: 
  - `testdata/cpage-pdf.rmdoc` (copy of `remarks/tests/in/copies of different pages.rmdoc`)
  - `testdata/legacy-notebook.zip` (copy of `rmapi/archive/test.zip`)

**Commit (code):** 6c01a1a — "RMQ-RMDOC-WEB-001: Phase 0-2 scaffold (Go backend + API + React frontend init)"

## Step 2: Implement Phase 1 API endpoints + Phase 2 Vite proxy

This step completes the core backend API (inspect, render background, render legacy, outputs serving) and configures the Vite dev server proxy. The backend now provides a complete REST API for document inspection and PDF generation, validated with curl smoke tests.

### What I did

- Implemented `cmd/remarquee-ui/api/inspect.go`: GET /api/document/:id/inspect endpoint
  - Parses document ID from URL path
  - Calls `pkg/rmdoc.OpenFile` to parse .rmdoc
  - Returns JSON with UUID, schema, docType, pageCount, pages array, hasPayloadPDF
- Implemented `cmd/remarquee-ui/api/render.go`: POST /api/render/background and POST /api/render/legacy endpoints
  - Accepts JSON body `{document_id}`
  - Calls `pkg/rmdoc/render.BuildBackgroundPDF` for background rendering
  - Calls `rmapi/annotations.PdfGenerator` for legacy V3/V5 rendering
  - Writes PDFs to `outputs/` directory
  - Returns JSON with `{job_id, output_path}`
- Implemented `cmd/remarquee-ui/api/outputs.go`: GET /api/outputs/:filename endpoint
  - Serves PDF files from `outputs/` directory
  - Sets `Content-Type: application/pdf` header
- Implemented `cmd/remarquee-ui/api/utils.go`: helper functions for JSON responses and document lookup
- Updated `main.go` to wire all API endpoints
- Updated `frontend/vite.config.ts` to add proxy configuration (`/api/*` → `http://localhost:8080`)
- Smoke-tested all endpoints:
  - `/api/test-documents` → returns fixture manifest
  - `/api/document/cpage-pdf/inspect` → returns parsed document metadata
  - `/api/render/background` → generates background PDF
  - `/api/render/legacy` → generates annotated PDF (V3/V5)
  - `/api/outputs/<filename>` → serves PDF files

### Why

- **Complete API surface**: frontend will call these endpoints to perform all document operations
- **Reuse existing packages**: `pkg/rmdoc` and `rmapi` handle the hard work; API layer is just glue code
- **Vite proxy**: dev workflow requires frontend (Vite :5173) to proxy `/api/*` requests to backend (:8080)

### What worked

- All API endpoints work as expected and return proper JSON responses
- `pkg/rmdoc.OpenFile` correctly parses both cPages and legacy fixtures
- PDF generation works for both background (cPages) and legacy (V3/V5) documents
- Vite proxy configuration is straightforward

### What didn't work

- (No blockers)

### What I learned

- The `http.ServeMux` pattern matching for `/api/document/` requires manual path parsing (no built-in URL params)
- `pkg/rmdoc` and `pkg/rmdoc/render` abstractions make the API layer very thin and testable

### What was tricky to build

- **URL path parsing**: extracting document ID from `/api/document/{id}/inspect` requires string splitting
- **Document lookup**: need to read `test-documents.json` manifest to map document ID to file path

### What warrants a second pair of eyes

- **Error handling**: confirm JSON error responses are consistent and informative
- **Path traversal**: verify `findDocumentPath` doesn't allow path traversal attacks (e.g., `../../etc/passwd`)
- **Output directory**: confirm `outputs/` is properly gitignored and cleaned up if needed

### What should be done in the future

- Add request ID/correlation ID for better logging and debugging
- Consider adding a `/api/render/status/:job_id` endpoint for async long-running renders (not needed for MVP)
- Add rate limiting or request validation if exposing to untrusted clients

### Code review instructions

- Start in `cmd/remarquee-ui/main.go`: verify all routes are wired correctly
- Review `cmd/remarquee-ui/api/inspect.go`: confirm document parsing and JSON serialization
- Review `cmd/remarquee-ui/api/render.go`: confirm both render endpoints work and write to outputs/
- Review `cmd/remarquee-ui/api/outputs.go`: confirm PDF serving is safe
- Run smoke tests:
  ```bash
  cd cmd/remarquee-ui
  go run main.go --dev &
  curl http://localhost:8080/api/test-documents
  curl http://localhost:8080/api/document/cpage-pdf/inspect
  curl -X POST http://localhost:8080/api/render/background -H 'Content-Type: application/json' -d '{"document_id":"cpage-pdf"}'
  ls -lh outputs/
  ```

### Technical details

- **API package structure**: `cmd/remarquee-ui/api/` with separate files for inspect, render, outputs, utils
- **JSON response types**: `InspectResponse`, `RenderRequest`, `RenderResponse`
- **Error handling**: all errors return JSON with `{error: "message"}` field
- **Output file naming**: `{docId}-{action}-{timestamp}.pdf`
- **Vite proxy**: configured in `vite.config.ts` with `target: 'http://localhost:8080'`

## Step 2: Implement Phase 1 — Core API endpoints

This step implements the complete backend API for document inspection and PDF rendering. All 5 endpoints now work: test documents listing, document inspection (schema + pages), background PDF building (cPages/PDF-backed docs), legacy PDF rendering (V3/V5), and output file serving. The API is ready for frontend integration.

**Commit (code):** c107adb — "RMQ-RMDOC-WEB-001: Phase 1 - Core API endpoints"

### What I did

- Created `cmd/remarquee-ui/api/` package with 4 files:
  - `inspect.go`: `GET /api/document/:id/inspect` endpoint
  - `render.go`: `POST /api/render/background` and `POST /api/render/legacy` endpoints
  - `outputs.go`: `GET /api/outputs/:filename` endpoint  
  - `utils.go`: JSON helpers (`respondJSON`, `readJSON`)
- Added `String()` methods to `pkg/rmdoc.ArchiveSchema` and `pkg/rmdoc.DocumentType` for JSON serialization
- Wired all handlers into `main.go` with proper path routing
- Tested all endpoints with `curl`: inspect, render background, render legacy, outputs serving

### Why

- **API-first design**: implementing the backend API before the frontend ensures the data contracts are solid
- **Reuses existing code**: all handlers delegate to `pkg/rmdoc` and `rmapi` directly (no duplicate logic)
- **Simple routing**: plain `http.ServeMux` with path-based routing is sufficient for this small API
- **Error handling**: each endpoint returns JSON errors with appropriate HTTP status codes

### What worked

- Path-based routing in `http.ServeMux` (e.g., `/api/document/` catches `/api/document/:id/inspect`)
- Extracting document ID from URL path by splitting on `/`
- Using `findDocumentPath` helper to resolve document IDs to file paths via the manifest
- `http.ServeFile` for outputs serving (handles Range requests, Last-Modified, etc. automatically)

### What didn't work

- Initially used `doc.DocType` instead of `doc.Type` (field name mismatch)
- `ArchiveSchema` and `DocumentType` didn't have `String()` methods initially (added them to `types.go`)
- Forgot to allow HEAD requests in outputs handler (fixed by checking both GET and HEAD)

### What I learned

- Adding `String()` methods to enum types in Go makes JSON serialization cleaner
- `http.ServeFile` is more robust than manually reading and writing file bytes (handles caching, range requests)
- Simple path splitting is adequate for small APIs; no need for a full router library yet

### What was tricky to build

- **Document ID resolution**: mapping document IDs (from URL) to file paths (from manifest) required a helper function that parses the manifest JSON
- **Schema validation**: the legacy render endpoint checks that the document is actually legacy before calling rmapi

### What warrants a second pair of eyes

- **Path parsing**: confirm the URL path splitting logic is robust for the expected paths (e.g., `/api/document/cpage-pdf/inspect`)
- **Error responses**: verify all error paths return appropriate HTTP status codes and JSON error messages
- **Security**: outputs handler blocks path traversal (`..` and `/` in filenames), but confirm this is sufficient

### What should be done in the future

- If the API grows beyond ~10 endpoints, consider a proper router library (gorilla/mux, chi)
- Add request logging middleware (currently just using `log.Printf` in handlers)
- Consider rate limiting for render endpoints if they become expensive

### Code review instructions

- Start in `cmd/remarquee-ui/api/inspect.go`: verify document ID extraction and JSON response structure
- Check `cmd/remarquee-ui/api/render.go`: verify background and legacy rendering logic
- Review `cmd/remarquee-ui/api/outputs.go`: confirm path traversal protection
- Run smoke tests: `cd cmd/remarquee-ui && make dev-backend`, then curl all 5 endpoints

### Technical details

- **API endpoints**:
  - `GET /api/health` — health check
  - `GET /api/test-documents` — list test documents from manifest
  - `GET /api/document/:id/inspect` — inspect document (schema, pages, etc.)
  - `POST /api/render/background` — build background PDF (cPages/PDF-backed)
  - `POST /api/render/legacy` — render legacy PDF (V3/V5 annotations via rmapi)
  - `GET /api/outputs/:filename` — serve generated PDF files
- **JSON response format**: all API responses use `application/json` with proper status codes
- **Output files**: PDFs written to `outputs/` directory with job-ID-based filenames
- **Job ID format**: `{documentId}-{action}-{timestamp}` (e.g., `cpage-pdf-background-1765810673`)
