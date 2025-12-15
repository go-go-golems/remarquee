---
Title: "Design: interactive rmdoc render validation UI (web, quick feedback loop)"
Ticket: RMQ-0004
Status: draft
Topics:
  - backend
  - go
  - remarkable
  - web
  - react
  - frontend
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
  - Path: remarquee/pkg/rmdoc/open.go
    Note: Open/parse .rmdoc to compute page plan
  - Path: remarquee/pkg/rmdoc/render/background.go
    Note: Background PDF assembly logic to validate visually
  - Path: remarquee/cmd/remarquee/cmds/rmdoc/inspect.go
    Note: Inspect command logic (can be reused in backend API)
  - Path: remarquee/cmd/remarquee/cmds/rmdoc/render_legacy.go
    Note: Legacy renderer logic (rmapi-backed)
  - Path: remarquee/cmd/remarquee/cmds/rmdoc/build_background.go
    Note: Background PDF builder logic
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-15T00:00:00Z
---

## 1. Purpose and scope

We need a **fast human-in-the-loop validation loop** for `.rmdoc` parsing + PDF generation, because many failures are *visual* (page ordering, duplicated pages, inserted blank pages, annotation alignment).

This document proposes a small **web UI** that sits next to the main `remarquee` binary and helps a human reviewer (manuel) quickly:

- **select from pre-prepared test documents** (covering different use cases: legacy PDF, cPages PDF with duplicates/insertions, notebooks)
- run rendering steps (inspect, build background, legacy render)
- **view generated PDFs inline** (via embedded PDF viewer or download)
- capture "PASS/FAIL + notes" as durable validation records

Non-goals (for the first iteration):

- pixel-perfect image diffs (just visual inspection)
- full V6 annotation rendering (this comes later; the UI should accommodate it)
- multi-user / auth / cloud deployment (single-developer local tool)

## 2. User stories

1) **As a reviewer**, I can **select from a pre-prepared list of test documents** (each representing a key use case: legacy, cPages with duplicates, cPages with insertions, notebook) and immediately see schema/docType/page plan.

2) **As a reviewer**, I can **trigger rendering operations** (inspect, build background, render legacy) with one click and see the status update in real-time.

3) **As a reviewer**, I can **view the generated PDF inline** in the browser (via embedded viewer or direct download) and provide structured feedback:
   - "UI page order is correct / incorrect"
   - "inserted pages are blank"
   - "duplicate pages actually appear twice"
   - "annotations align with background" (legacy)

4) **As a reviewer**, I can **mark a validation run as PASS/FAIL** and add freeform notes, which are persisted as a validation record (JSON + Markdown) under the RMQ-0004 ticket directory.

5) **As a reviewer**, I can **see a history of validation sessions** and re-run tests against different documents to track regressions.

## 3. Architecture overview

### 3.1. High-level stack

```text
┌─────────────────────────────────────────────────────────┐
│  Browser                                                 │
│  ┌───────────────────────────────────────────────────┐  │
│  │  React + Redux Toolkit + TypeScript (Vite)        │  │
│  │  - Test document selector                          │  │
│  │  - Action buttons (inspect, render)                │  │
│  │  - PDF viewer / download links                     │  │
│  │  - Validation form (PASS/FAIL + notes)             │  │
│  └───────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
                          ▲
                          │ HTTP (REST + static assets)
                          │ Dev: Vite proxy → :8080
                          │ Prod: Go embed (bundled JS)
                          ▼
┌─────────────────────────────────────────────────────────┐
│  Go backend (net/http)                                   │
│  ┌───────────────────────────────────────────────────┐  │
│  │  API handlers:                                      │  │
│  │  - GET /api/test-documents (list)                  │  │
│  │  - GET /api/document/:id (inspect)                 │  │
│  │  - POST /api/render/background                     │  │
│  │  - POST /api/render/legacy                         │  │
│  │  - GET /api/outputs/:id.pdf (serve PDF)            │  │
│  │  - POST /api/validation (save session)             │  │
│  └───────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────┐  │
│  │  Business logic (reuse remarquee/pkg):             │  │
│  │  - pkg/rmdoc.OpenFile                              │  │
│  │  - pkg/rmdoc/render.BuildBackgroundPDF             │  │
│  │  - rmapi annotations.PdfGenerator (legacy)         │  │
│  └───────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

### 3.2. Directory structure (self-contained under `cmd/remarquee-ui`)

```text
remarquee/
├── cmd/
│   └── remarquee-ui/
│       ├── main.go                    # Go HTTP server entrypoint
│       ├── api/
│       │   ├── handlers.go            # HTTP handlers (REST API)
│       │   ├── documents.go           # Test document registry
│       │   └── validation.go          # Validation session persistence
│       ├── frontend/                  # React + Vite app
│       │   ├── package.json
│       │   ├── vite.config.ts         # Vite + proxy config
│       │   ├── tsconfig.json
│       │   ├── index.html
│       │   ├── src/
│       │   │   ├── main.tsx
│       │   │   ├── App.tsx
│       │   │   ├── store/             # Redux Toolkit slices
│       │   │   │   ├── store.ts
│       │   │   │   ├── documentsSlice.ts
│       │   │   │   ├── renderSlice.ts
│       │   │   │   └── validationSlice.ts
│       │   │   ├── components/
│       │   │   │   ├── DocumentSelector.tsx
│       │   │   │   ├── InspectPanel.tsx
│       │   │   │   ├── RenderActions.tsx
│       │   │   │   ├── PDFViewer.tsx
│       │   │   │   └── ValidationForm.tsx
│       │   │   └── api/
│       │   │       └── client.ts      # API client (fetch wrappers)
│       │   └── dist/                  # (gitignored, build output)
│       ├── static/                    # (gitignored, embedded in prod)
│       │   └── index.html + JS bundle
│       ├── testdata/                  # Pre-prepared .rmdoc test fixtures
│       │   ├── test-documents.json    # Manifest
│       │   ├── legacy-pdf.rmdoc
│       │   ├── cpages-duplicates.rmdoc
│       │   ├── cpages-insertions.rmdoc
│       │   └── notebook.rmdoc
│       ├── Makefile                   # Build orchestration
│       └── embed.go                   # Go embed directives for prod bundle
```

### 3.3. Build modes

**Development mode:**
- Frontend: `cd frontend && npm run dev` → Vite dev server on `:5173` with HMR
- Backend: `go run main.go` → Go server on `:8080`
- Vite proxy: `/api/*` → `http://localhost:8080/api/*`

**Production mode:**
- Frontend: `cd frontend && npm run build` → outputs to `frontend/dist/`
- Go: embed `frontend/dist/*` into binary via `//go:embed`
- Single binary serves both static assets and API

## 4. API definitions (backend REST endpoints)

All endpoints under `/api/` prefix. JSON request/response.

### 4.1. `GET /api/test-documents`

**Purpose:** List available pre-prepared test documents.

**Response:**
```json
{
  "documents": [
    {
      "id": "legacy-pdf",
      "path": "testdata/legacy-pdf.rmdoc",
      "description": "Legacy .content + V3/V5 .rm (PDF-backed)",
      "expected_schema": "legacy",
      "expected_type": "pdf",
      "expected_pages": 1
    },
    {
      "id": "cpages-duplicates",
      "path": "testdata/cpages-duplicates.rmdoc",
      "description": "cPages with same SourcePDFPage used twice",
      "expected_schema": "cPages",
      "expected_type": "pdf",
      "expected_pages": 4
    }
  ]
}
```

### 4.2. `GET /api/document/:id/inspect`

**Purpose:** Inspect a test document and return parsed metadata.

**Response:**
```json
{
  "uuid": "d4e3ce4f-564a-43ee-a54f-203043de37b4",
  "schema": "cPages",
  "type": "pdf",
  "pages": [
    {
      "index": 0,
      "page_id": "ce65b731-9fc5-469d-9d87-0672d21c06db",
      "source_pdf_page": 0,
      "template": "Blank",
      "deleted": false
    }
  ]
}
```

### 4.3. `POST /api/render/background`

**Purpose:** Build a UI-ordered background PDF.

**Request:**
```json
{
  "document_id": "cpages-duplicates"
}
```

**Response:**
```json
{
  "job_id": "bg-20251215-001",
  "status": "completed",
  "output_path": "/api/outputs/bg-20251215-001.pdf",
  "duration_ms": 234
}
```

### 4.4. `POST /api/render/legacy`

**Purpose:** Render a legacy .rmdoc to annotated PDF (rmapi-backed).

**Request:**
```json
{
  "document_id": "legacy-pdf",
  "options": {
    "add_page_numbers": false,
    "all_pages": true,
    "annotations_only": false
  }
}
```

**Response:**
```json
{
  "job_id": "legacy-20251215-002",
  "status": "completed",
  "output_path": "/api/outputs/legacy-20251215-002.pdf",
  "duration_ms": 567
}
```

### 4.5. `GET /api/outputs/:filename`

**Purpose:** Serve generated PDF files.

**Response:** PDF binary (Content-Type: application/pdf).

### 4.6. `POST /api/validation`

**Purpose:** Save a validation session record.

**Request:**
```json
{
  "document_id": "cpages-duplicates",
  "actions": [
    {"kind": "inspect", "timestamp": "2025-12-15T00:00:00Z"},
    {"kind": "background_pdf", "job_id": "bg-20251215-001"}
  ],
  "review": {
    "status": "pass",
    "notes": "UI page order correct; duplicates rendered twice as expected."
  }
}
```

**Response:**
```json
{
  "session_id": "session-20251215-001",
  "saved_paths": [
    "reference/validation/session-20251215-001.json",
    "reference/validation/session-20251215-001.md"
  ]
}
```

## 5. Frontend state model (Redux Toolkit slices)

### 5.1. `documentsSlice`

```typescript
interface DocumentsState {
  list: TestDocument[]
  selected: string | null
  inspectResult: InspectResult | null
  loading: boolean
  error: string | null
}
```

### 5.2. `renderSlice`

```typescript
interface RenderState {
  jobs: Record<string, RenderJob>
  loading: boolean
  error: string | null
}

interface RenderJob {
  job_id: string
  kind: "background" | "legacy"
  status: "pending" | "completed" | "failed"
  output_path: string | null
  duration_ms: number | null
}
```

### 5.3. `validationSlice`

```typescript
interface ValidationState {
  current: ValidationSession | null
  history: ValidationSession[]
  loading: boolean
}

interface ValidationSession {
  session_id: string
  document_id: string
  started_at: string
  actions: Action[]
  review: { status: "pass" | "fail" | "unknown"; notes: string }
}
```

## 6. Control flow diagrams

### 6.1. Document selection → inspect flow

```text
User clicks document in list
  ↓
Frontend: dispatch selectDocument(id)
  ↓
Frontend: dispatch fetchInspect(id)
  ↓
API: GET /api/document/:id/inspect
  ↓
Backend: pkg/rmdoc.OpenFile(path)
  ↓
Backend: serialize Document to JSON
  ↓
Frontend: store inspectResult in documentsSlice
  ↓
UI: render InspectPanel with schema/pages table
```

### 6.2. Render action → view PDF flow

```text
User clicks "Build Background"
  ↓
Frontend: dispatch renderBackground(docId)
  ↓
API: POST /api/render/background {document_id}
  ↓
Backend: pkg/rmdoc/render.BuildBackgroundPDF()
  ↓
Backend: write PDF to outputs/<job_id>.pdf
  ↓
Backend: return {job_id, output_path}
  ↓
Frontend: store job in renderSlice
  ↓
UI: render PDFViewer with iframe src="/api/outputs/<job_id>.pdf"
     OR download link
```

### 6.3. Validation submission flow

```text
User fills form (PASS/FAIL + notes) and clicks "Save"
  ↓
Frontend: dispatch submitValidation({docId, actions, review})
  ↓
API: POST /api/validation {...}
  ↓
Backend: persist session as JSON + Markdown in ticket dir
  ↓
Backend: return {session_id, saved_paths}
  ↓
Frontend: add session to validationSlice.history
  ↓
UI: show success notification + reset form
```

## 7. Makefile targets (cmd/remarquee-ui/Makefile)

```makefile
.PHONY: dev build clean install test

# Development: run backend + frontend in separate terminals
dev-backend:
	go run main.go --dev

dev-frontend:
	cd frontend && npm install && npm run dev

# Production build: bundle frontend + embed in Go binary
build:
	cd frontend && npm install && npm run build
	go build -o remarquee-ui main.go

# Clean build artifacts
clean:
	rm -rf frontend/dist frontend/node_modules
	rm -f remarquee-ui

# Install dependencies
install:
	cd frontend && npm install
	go mod download

# Run tests
test:
	go test ./api/...
	cd frontend && npm run test
```

## 8. Implementation plan (phased)

### 8.1. Phase 0 — Go backend scaffold + test documents

- [ ] Create `cmd/remarquee-ui/main.go` with `net/http` server
- [ ] Implement `/api/test-documents` handler
- [ ] Add `testdata/` directory with pre-prepared `.rmdoc` fixtures
- [ ] Create `testdata/test-documents.json` manifest

### 8.2. Phase 1 — Core API endpoints

- [ ] Implement `GET /api/document/:id/inspect` (calls `pkg/rmdoc.OpenFile`)
- [ ] Implement `POST /api/render/background` (calls `pkg/rmdoc/render.BuildBackgroundPDF`)
- [ ] Implement `POST /api/render/legacy` (calls rmapi `PdfGenerator`)
- [ ] Implement `GET /api/outputs/:filename` (serve PDFs)

### 8.3. Phase 2 — React frontend scaffold

- [ ] Init React + Vite + TypeScript in `frontend/`
- [ ] Configure Vite proxy (`/api/*` → `http://localhost:8080`)
- [ ] Setup Redux Toolkit store with 3 slices
- [ ] Create basic layout (sidebar + main panel)

### 8.4. Phase 3 — Core UI components

- [ ] `DocumentSelector` (fetches + displays test document list)
- [ ] `InspectPanel` (displays parsed schema/pages table)
- [ ] `RenderActions` (buttons for inspect/background/legacy)
- [ ] `PDFViewer` (iframe or download link)
- [ ] `ValidationForm` (PASS/FAIL radio + notes textarea)

### 8.5. Phase 4 — Validation persistence

- [ ] Implement `POST /api/validation` handler
- [ ] Backend: write JSON + Markdown to ticket `reference/validation/`
- [ ] Frontend: dispatch validation submission
- [ ] UI: show validation history list

### 8.6. Phase 5 — Production build + embed

- [ ] Add `//go:embed frontend/dist` directive to `embed.go`
- [ ] Makefile: `make build` → npm build + go build
- [ ] Serve embedded static files in prod mode
- [ ] Test single-binary deployment

## 9. Integration points and constraints

- **No external dependencies for rendering**: reuse `pkg/rmdoc` and rmapi directly
- **Self-contained**: all test documents live in `testdata/` directory
- **Non-destructive**: outputs written to `outputs/` subdirectory (gitignored)
- **No cloud auth required**: purely local tool
- **Dev/prod mode detection**: `--dev` flag or env var to switch between embedded assets vs proxy

## 10. Open questions

- [ ] PDF viewer: inline iframe vs force download vs both options?
- [ ] Do we want real-time progress updates (e.g., WebSocket) for long renders?
- [ ] Should validation history be persistent across restarts (SQLite) or session-only (in-memory)?
- [ ] Do we need a "compare" mode to show two PDFs side-by-side for regression testing?


