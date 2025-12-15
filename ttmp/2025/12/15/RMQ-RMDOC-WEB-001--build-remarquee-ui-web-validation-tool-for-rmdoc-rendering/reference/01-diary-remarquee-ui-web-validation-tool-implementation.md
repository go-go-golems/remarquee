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
  - `testdata/cpage-pdf.rmdoc` (symlink or copy of `remarks/tests/in/copies of different pages.rmdoc`)
  - `testdata/legacy-notebook.zip` (symlink or copy of `rmapi/archive/test.zip`)
