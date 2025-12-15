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

## Step 3: Implement Phase 2+3 (Redux store + UI components)

This step completes the frontend by implementing the Redux Toolkit store (3 slices: documents, render, validation) and all 5 core UI components (DocumentSelector, InspectPanel, RenderActions, PDFViewer, ValidationForm). The UI is now fully wired to the backend API and provides a complete validation workflow: select document → inspect → render → view PDF → submit validation.

**Commit (code):** 3598a7b — "RMQ-RMDOC-WEB-001: Phase 2+3 - Redux store + UI components (DocumentSelector, InspectPanel, RenderActions, PDFViewer, ValidationForm)"

### What I did

- Created Redux store structure in `frontend/src/store/`:
  - `store.ts`: main store configuration with 3 reducers
  - `hooks.ts`: typed hooks (`useAppDispatch`, `useAppSelector`)
  - `documentsSlice.ts`: manages test documents list, document selection, and inspect results
  - `renderSlice.ts`: manages render jobs (background, legacy) and output paths
  - `validationSlice.ts`: manages validation review state (status, notes) and submission
- Updated `main.tsx` to wrap App in Redux `<Provider>`
- Created 5 UI components in `frontend/src/components/`:
  - `DocumentSelector.tsx`: displays test documents list, handles selection + inspect dispatch
  - `InspectPanel.tsx`: displays parsed document metadata and pages table
  - `RenderActions.tsx`: buttons for "Build Background" and "Render Legacy" actions
  - `PDFViewer.tsx`: displays latest PDF output in iframe with open/download links, shows job history
  - `ValidationForm.tsx`: PASS/FAIL/UNKNOWN radio buttons, notes textarea, submit/reset buttons
- Updated `App.tsx` to use new layout (header + sidebar + main panel) and render all components
- Updated `App.css` with comprehensive styling for all components (layout, buttons, tables, forms)
- Fixed TypeScript errors: used `type` imports for `PayloadAction` and `TypedUseSelectorHook`
- Verified build success: `npm run build` completes without errors

### Why

- **Redux Toolkit**: provides type-safe state management with minimal boilerplate
- **3 slices**: logical separation of concerns (documents, rendering, validation)
- **Typed hooks**: enforce type safety throughout the app
- **Component-per-concern**: each component has a single responsibility and is easy to test/modify
- **Iframe PDF viewer**: allows in-page PDF preview without requiring external viewer

### What worked

- Redux Toolkit async thunks handle API calls cleanly
- All components compile and build successfully
- TypeScript strict mode catches import errors early
- CSS Grid/Flexbox layout provides responsive UI

### What didn't work

- Initial TypeScript errors due to `verbatimModuleSyntax` requiring type-only imports (fixed with `type` keyword)

### What I learned

- React 19 + Redux Toolkit 2.x work well together with modern TypeScript
- `type` imports are required when `verbatimModuleSyntax` is enabled in TypeScript
- Vite builds are very fast (~1.3s for this app)

### What was tricky to build

- **Type imports**: needed to use `import { type PayloadAction }` syntax to satisfy TypeScript strict mode
- **Job history UI**: balancing between showing all jobs vs. only the latest (solved with `<details>` for history)

### What warrants a second pair of eyes

- **Redux slice design**: confirm the 3-slice separation makes sense (vs. combining documents+render)
- **PDF viewer iframe**: confirm this works across browsers (especially with CORS/CSP)
- **Validation form**: confirm UX is intuitive (status radio buttons + notes textarea)

### What should be done in the future

- Add loading spinners for async operations
- Add error notifications (toast/banner) for API failures
- Add keyboard shortcuts (e.g., Ctrl+Enter to submit validation)
- Consider adding a "Clear all outputs" button to clean up the outputs/ directory

### Code review instructions

- Start in `frontend/src/store/store.ts`: verify 3 slices are configured
- Review each slice (`documentsSlice.ts`, `renderSlice.ts`, `validationSlice.ts`): confirm async thunks and reducers
- Review `App.tsx`: confirm layout and component hierarchy
- Review each component in `components/`: confirm Redux hooks usage and event handlers
- Run frontend build: `cd frontend && npm run build` (should succeed)
- Visually inspect `App.css` for layout/styling

### Technical details

- **Redux slices**:
  - `documents`: `{testDocuments, selectedDocumentId, inspectResult, loading, error}`
  - `render`: `{jobs: RenderJob[], loading, error}`
  - `validation`: `{currentReview: {status, notes}, history, loading, error}`
- **Async thunks**: `fetchTestDocuments`, `fetchInspect`, `renderBackground`, `renderLegacy`, `submitValidation`
- **Component props**: all components use Redux hooks (no prop drilling)
- **CSS organization**: all styles in `App.css` (could be split per-component in the future)
- **Build output**: `frontend/dist/` (ignored by git, served in prod mode)

## Step 6: UI improvements for desktop - 3-column layout, validation guidance, enhanced content display

This step improves the UI for desktop use by adding a 3-column layout (left: document selector, center: content, right: validation guidance), fixing all CSS contrast issues (white-on-white text), and enhancing the inspect panel to show rmdoc content details with visual highlighting for duplicates and inserted pages.

**Commits:** a163e7b (contrast fix), b78506c (3-column layout), 2866449 (final contrast fixes)

### What I did

- Fixed CSS contrast issues:
  - Added explicit `color: #2c3e50` to all text elements (main panel, sidebar, guidance panel)
  - Fixed white-on-white text in document selector buttons, checklist items, and guidance panel paragraphs
  - Added proper color inheritance for `strong` and `small` elements
- Converted layout to 3-column grid:
  - Left sidebar (280px): Document selector + render actions
  - Center panel (flexible): Main content with inspect, PDF viewer, validation form
  - Right sidebar (320px): New validation guidance panel
- Created `ValidationGuidance` component:
  - Shows document-specific info (schema, type, page count)
  - Dynamic checklist based on schema (legacy vs cPages) and doc type (PDF vs Notebook)
  - Common issues reference with troubleshooting tips
  - Visual checkboxes (□) for manual tracking
- Enhanced `InspectPanel` component:
  - Shows full document description from test manifest
  - Color-coded schema badges (cPages = green, legacy = orange)
  - Color-coded doc type badges (PDF = blue, Notebook = purple)
  - Grid layout for metadata (better desktop use)
  - Visual page highlighting in table:
    - Yellow background = Inserted pages (sourcePdfPage === -1)
    - Blue background = Duplicate pages (same sourcePdfPage appears multiple times)
    - "Type" column shows: "Inserted", "Duplicate", or "Normal"
  - Improved table styling with better contrast
- Fixed dev mode: Changed Makefile `dev-backend` from `go run main.go` to `go run .` (to include embed.go)

### Why

- **Desktop-first**: most validation work happens on desktop, not mobile
- **Side-by-side**: guidance panel always visible while inspecting/validating
- **Visual feedback**: color highlighting makes duplicates and inserted pages obvious at a glance
- **Contrast**: dark text on light backgrounds is essential for readability

### What worked

- 3-column CSS Grid layout scales well on desktop (2560px+ monitors)
- Visual highlighting (yellow for inserted, blue for duplicates) is immediately clear
- Guidance panel provides helpful context without being intrusive
- All text is now readable (dark on light)

### What didn't work

- Initial attempts missed several white-on-white cases (buttons, guidance panel text, links)
- Required 3 commits to catch all contrast issues

### What I learned

- CSS color inheritance is tricky: need explicit colors on nested elements (strong, small, span)
- `color: inherit` on buttons helps maintain contrast in different states (normal vs selected)
- CSS Grid with named columns is cleaner than flexbox for 3-column layouts

### What was tricky to build

- **Duplicate detection**: needed to scan all pages to find duplicates (same sourcePdfPage appears multiple times)
- **Color inheritance**: had to set explicit colors at multiple levels (button, button strong, button small, etc.)

### What warrants a second pair of eyes

- **Desktop layout**: confirm 3-column layout works well on standard desktop resolutions (1920x1080, 2560x1440)
- **Color contrast**: verify all text is readable (WCAG AA compliance)
- **Guidance panel**: confirm checklist is comprehensive and helpful

### What should be done in the future

- Add responsive breakpoints for smaller screens (collapse to 2-column or single-column)
- Add "check all" / "uncheck all" buttons for guidance checklist
- Make checklist interactive (persist checked state in local storage)
- Add keyboard shortcuts for common actions (e.g., space to check/uncheck)

### Code review instructions

- Review `App.css`: verify 3-column grid layout, guidance panel styling, and all color definitions
- Review `App.tsx`: verify ValidationGuidance component is wired correctly
- Review `ValidationGuidance.tsx`: confirm dynamic content based on schema/docType
- Review `InspectPanel.tsx`: verify duplicate detection logic and color highlighting
- Test: run `make build`, open browser to http://localhost:8080, verify layout and contrast

### Technical details

- **3-column grid**: `grid-template-columns: 280px 1fr 320px` (fixed sidebars, flexible center)
- **Color scheme**:
  - Primary text: `#2c3e50` (dark blue-gray)
  - Guidance panel background: `#fff9e6` (light yellow)
  - Guidance panel accent: `#f39c12` (orange)
  - cPages badge: `#27ae60` (green)
  - Legacy badge: `#f39c12` (orange)
  - PDF badge: `#3498db` (blue)
  - Notebook badge: `#9b59b6` (purple)
  - Inserted page highlight: `#fff3cd` (yellow)
  - Duplicate page highlight: `#d1ecf1` (blue)
- **Duplicate detection**: `pages.filter(p => p.sourcePdfPage === page.sourcePdfPage && page.sourcePdfPage !== -1).length > 1`

## Step 4: Implement Phase 4 (validation persistence)

This step adds backend validation persistence, writing validation sessions as both JSON (machine-readable) and Markdown (human-readable) to the ticket's `reference/validation/` directory. This enables durable tracking of validation results across UI sessions and provides copy/paste-ready artifacts for documentation.

**Commit (code):** 6061879 — "RMQ-RMDOC-WEB-001: Phase 4 - Validation persistence (backend writes JSON and MD to ticket reference/validation)"

### What I did

- Created `cmd/remarquee-ui/api/validation.go`:
  - `HandleValidation`: POST endpoint that receives `ValidationSession` JSON
  - Generates session ID from Unix timestamp
  - Creates `reference/validation/` directory if it doesn't exist
  - Writes JSON file: `validation-<timestamp>.json`
  - Writes Markdown file: `validation-<timestamp>.md` with formatted session details
  - Returns `{session_id, saved_paths}` response
  - `formatValidationMarkdown`: helper to generate human-readable Markdown
- Updated `main.go` to wire `/api/validation` endpoint with ticket directory path
- Smoke-tested: submitted validation via curl, verified files created in ticket

### Why

- **Dual format**: JSON for programmatic access, Markdown for human review
- **Ticket-relative paths**: validation sessions stored in ticket (not ephemeral outputs/)
- **Timestamp-based IDs**: deterministic, sortable, collision-resistant

### What worked

- Both JSON and Markdown files created successfully
- Markdown format is readable and includes all relevant context (document ID, actions, status, notes)

### What didn't work

- (No issues)

### What I learned

- Go's `os.MkdirAll` with `0755` creates parent directories automatically
- Relative paths in `ticketDir` work fine when running from `cmd/remarquee-ui/`

### What was tricky to build

- **Relative paths**: ensuring `ticketDir` path is correct relative to where the binary runs

### What warrants a second pair of eyes

- **Path security**: confirm `ticketDir` path is safe and doesn't allow path traversal
- **File permissions**: confirm 0644 for files and 0755 for directories is appropriate

### What should be done in the future

- Add validation history GET endpoint to list past sessions (skipped for MVP)
- Add timestamp to UI (currently only in backend response)
- Consider compressing old validation sessions (e.g., archive after 30 days)

### Code review instructions

- Review `cmd/remarquee-ui/api/validation.go`: confirm JSON/MD generation logic
- Review `main.go`: confirm ticket directory path is correct
- Test: run server, submit validation, check files in `reference/validation/`

### Technical details

- **Session ID format**: `validation-<unix_timestamp>`
- **JSON structure**: matches frontend `ValidationSession` interface
- **Markdown format**: heading + metadata + actions list + notes section
- **Directory**: `ttmp/2025/12/15/RMQ-RMDOC-WEB-001--build-remarquee-ui-web-validation-tool-for-rmdoc-rendering/reference/validation/`

## Step 5: Implement Phase 5 (production build + embed)

This step completes the tool by implementing production build with embedded frontend assets via `go:embed`. The result is a single self-contained binary (`remarquee-ui`) that serves the entire web application without requiring a separate Vite dev server. This makes deployment and distribution trivial: copy one file and run it.

**Commit (code):** b01ec7e — "RMQ-RMDOC-WEB-001: Phase 5 - Production build with embedded assets (go:embed frontend/dist)"

### What I did

- Created `cmd/remarquee-ui/embed.go`:
  - `//go:embed frontend/dist` directive to embed frontend build artifacts
  - `GetFrontendFS()` helper to extract embedded filesystem
- Updated `main.go`:
  - Added prod mode logic: serves embedded assets via `http.FileServer(http.FS(frontendFS))`
  - Dev mode unchanged: still expects Vite dev server on `:5173`
- Updated `Makefile` `build` target:
  - Runs `npm run build` first to generate `frontend/dist/`
  - Then runs `go build` to embed and compile
- Smoke-tested: `make build` → `./remarquee-ui` → verified index.html served + API endpoints work

### Why

- **Single binary deployment**: no need to ship frontend/ directory or run separate Vite server
- **Production-ready**: embedded assets are minified and optimized by Vite
- **Dev/prod parity**: same codebase, different modes via `--dev` flag

### What worked

- `go:embed` works perfectly with Vite's `dist/` output
- `fs.Sub` correctly strips the `frontend/dist` prefix
- `http.FileServer(http.FS(...))` serves embedded assets correctly
- Production binary is ~10MB (includes React + Redux Toolkit)

### What didn't work

- (No issues)

### What I learned

- `go:embed` requires the embedded directory to exist at build time (Makefile ensures this)
- `fs.Sub` is necessary to strip the embed path prefix before serving

### What was tricky to build

- **Embed path**: needed `fs.Sub(frontendDist, "frontend/dist")` to strip the prefix
- **Makefile dependency**: must build frontend before Go binary

### What warrants a second pair of eyes

- **Embed directive**: confirm `//go:embed frontend/dist` includes all necessary files
- **Build order**: confirm Makefile always builds frontend first
- **Prod mode detection**: confirm `--dev` flag is the right way to switch modes

### What should be done in the future

- Add versioning/build info (e.g., `--version` flag with commit hash + build timestamp)
- Consider adding a health check endpoint that returns build info
- Add CORS headers if exposing to non-localhost clients

### Code review instructions

- Review `embed.go`: confirm `go:embed` directive and `GetFrontendFS` logic
- Review `main.go` prod mode: confirm file server setup
- Test: `cd cmd/remarquee-ui && make build && ./remarquee-ui` → open browser to `http://localhost:8080`

### Technical details

- **Embed directive**: `//go:embed frontend/dist` (must be directly above var)
- **Binary size**: ~10MB (includes Go runtime + React + Redux Toolkit + app code)
- **Dev mode**: `./remarquee-ui --dev` expects Vite on :5173
- **Prod mode**: `./remarquee-ui` serves embedded assets + API on :8080

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
