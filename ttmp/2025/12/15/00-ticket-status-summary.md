# Ticket Status Summary — Where We Stopped & Next Steps

**Generated:** 2025-12-15  
**Purpose:** Quick overview of active tickets and immediate next steps

---

## 🎯 Most Recent Work: RMQ-RMDOC-WEB-001 (remarquee-ui)

**Status:** Active  
**Last Updated:** 2025-12-15  
**Last Work:** Step 7 — Added internal structure display (shows .rm files with versions, content/metadata JSON, archive file list)

### V6 Rendering Status

**Answer: No, we are NOT rendering V6 yet.**

**What we CAN do:**
- ✅ Parse V6 `.rmdoc` files (schema detection, page plan)
- ✅ Detect V6 annotations in `.rm` files (version detection)
- ✅ Build background PDFs for V6 documents
- ✅ Inspect V6 document structure (internal structure display)

**What we CANNOT do yet:**
- ❌ Parse V6 scene tree (CRDT sequences, groups, lines)
- ❌ Render V6 strokes to PDF
- ❌ Render V6 highlights to PDF
- ❌ Render V6 typed text to PDF
- ❌ Merge V6 annotations with background PDF

**Current V6 Support:** The `cpage-pdf.rmdoc` fixture contains V6 annotations (version=6), which can be used for testing V6 structure detection and internal inspection. Full V6 rendering is tracked in RMQ-0004 (Section 4-6: V6 parsing + rendering).

### What's Complete ✅

- **Phase 0-5:** Fully implemented
  - Go backend scaffold + test documents
  - Core API endpoints (inspect, render background/legacy, outputs)
  - React frontend (Redux Toolkit + Vite)
  - Core UI components (DocumentSelector, InspectPanel, RenderActions, PDFViewer, ValidationForm)
  - Validation persistence (JSON + Markdown to ticket)
  - Production build with embedded assets
- **Phase 6:** UI enhancements complete
  - 3-column desktop layout
  - Validation guidance panel
  - Enhanced inspect panel with visual highlighting (duplicates, inserted pages)
  - Internal structure display (expandable sections for .rm files, JSON content)

### Immediate Next Steps 🔜

1. **Phase 2:** Smoke-test frontend dev mode
   - [ ] Run `npm run dev` and verify proxy works
   - [ ] Test full workflow: select document → inspect → render → validate

2. **Phase 4:** Validation history UI
   - [ ] Implement `GET /api/validation/history` endpoint
   - [ ] Display validation history list (collapsible) in UI

3. **Phase 6:** V6 test fixture
   - [x] Document V6 fixture status: cpage-pdf.rmdoc contains V6 annotations (version=6)
   - [ ] Test internal structure display with V6 document (using cpage-pdf.rmdoc)

### Acceptance Criteria Status

- ✅ Can select from pre-prepared test documents
- ✅ Can inspect document and see schema + pages
- ✅ Can build background PDF and view/download it
- ✅ Can render legacy PDF (V3/V5) and view/download it
- ✅ Can submit validation (PASS/FAIL + notes) and see it persisted
- ✅ Single `make build` produces standalone binary
- ✅ Dev mode works with hot reload
- ✅ Desktop-optimized 3-column layout
- ✅ Visual highlighting for duplicates and inserted pages
- ✅ Internal structure inspection

**Next Priority:** Add V6 test fixture and complete validation history UI.

---

## 🔧 RMQ-0004: Port .rmdoc Parsing + Rendering to Go

**Status:** Active  
**Last Updated:** 2025-12-15  
**Last Work:** Step 6 — Background PDF assembly + Glazed command conversion

### What's Complete ✅

- **Foundation (Step 1-2):** `.rmdoc` container + page plan
  - `pkg/rmdoc.OpenFile/OpenReaderAt` (zip open + read content/metadata/pagedata/pdf)
  - `pkg/rmdoc.ParseContent` (detect cPages vs legacy, build UI-ordered `[]PageRef`)
  - `pkg/rmdoc.ApplyPagedataTemplates` (fill missing templates)
  - Unit tests for legacy + cPages parsing/page plan
- **CLI (Step 3):** `remarquee rmdoc inspect` command
- **Tests (Step 4):** Integration tests for OpenFile (legacy + cPages fixtures)
- **Legacy Rendering (Step 5):** `remarquee rmdoc render-legacy` (rmapi-backed)
- **Background PDF (Step 6):** `pkg/rmdoc/render.BuildBackgroundPDF` + Glazed conversion

### Immediate Next Steps 🔜

1. **Legacy Pipeline (Section 2):** Complete V3/V5 rendering
   - [ ] Decide integration approach (Option A: wrap rmapi vs Option B: reimplement)
   - [ ] Implement adapter for V3/V5 `.rm` to PDF pages
   - [ ] Ensure legacy page order follows `pkg/rmdoc.Document.Pages`
   - [ ] Handle background PDF payload merge

2. **Background PDF (Section 3):** Template rendering
   - [ ] Define template-to-page-size/template rendering strategy
   - [ ] Replace blank page constants with proper template rendering

3. **V6 Parsing (Section 4):** Port rmscene concepts
   - [ ] Tagged block reader (header + main blocks + subblocks)
   - [ ] CRDT sequence decoding for scene items
   - [ ] Build scene tree (groups + lines)
   - [ ] Expose strokes as normalized primitives
   - [ ] Expand incrementally: highlights, typed text

4. **V6 Rendering (Section 5):** Stroke rendering + merge
   - [ ] Convert V6 line points to renderer stroke primitives
   - [ ] Apply coordinate transforms (SCALE, X_SHIFT, etc.)
   - [ ] Compute bounding boxes
   - [ ] Implement PDF merge algorithm (background + annotation overlay)

5. **Smart Highlights (Section 6):** V6 highlights
   - [ ] Use `PenColor` mapping to RGB
   - [ ] Create PDF highlight annotations with `x_translation`

6. **Fixtures + Tests (Section 7):** Real documents
   - [ ] Add at least one legacy PDF `.rmdoc` from device (V3/V5)
   - [ ] Add at least one notebook `.rmdoc` from device (V6)
   - [ ] Add reproducible test runner (visual diff or raster diff)

**Next Priority:** Complete legacy pipeline decision and V6 parsing foundation.

---

## 🛠️ GITCOMMIT-XXXX: Build Go CLI for Safe Git Commits

**Status:** Active  
**Last Updated:** 2025-12-14  
**Last Work:** Ticket created, prototype script exists

### What's Complete ✅

- Ticket workspace created
- Prototype script: `scripts/01-gitcommit-prototype.sh`
- Design doc: `design-doc/01-project-description-git-commit-helper-cli-go.md`

### Immediate Next Steps 🔜

**All tasks are TODO (not started):**

- [ ] Confirm scope: standalone tool vs integrated into existing repo CLIs
- [ ] Decide on configuration approach (flags-only vs optional config file)
- [ ] Implement `repo` command (git root, branch, clean/dirty summary)
- [ ] Implement `plan` command (preview-only; no writes)
- [ ] Implement `stage` command (explicit paths; optional `--ticket` preset)
- [ ] Implement `commit` command with guardrails:
  - [ ] refuse unrelated staged paths unless `--allow-unrelated`
  - [ ] separate "code" vs "docs" commit modes
  - [ ] print commit hash deterministically
- [ ] Implement docmgr integration (opt-in):
  - [ ] update diary/changelog with commit hash
  - [ ] relate changed files automatically with `--file-note` generation
- [ ] Add tests for path classification + repo discovery
- [ ] Write usage docs + playbook

**Next Priority:** Confirm scope and start with `repo` command.

---

## ✅ Completed/Nearly Complete Tickets

### RMQ-0002: Implement remarquee-cloud CLI
**Status:** Complete ✅ (2025-12-15)  
**All cloud commands implemented:** refresh, ls, stat, get, put, mkdir, mv, rm, find, account, version  
**Note:** Tests are optional and can be added later

### RMQ-0001: Build remarquee tool (unify rmapi/remarks/upload/stream/OCR)
**Status:** Complete ✅ (2025-12-15)  
**Analysis phase complete:** Comprehensive documentation created for rmapi, remarks, remarkable_upload.py, and goMarkableStream

### RMQ-0003: Port remarkable-upload.py to Go
**Status:** Complete ✅  
**All tasks checked off**

### RMQ-0005: remarquee-upload next features
**Status:** Complete ✅  
**All tasks checked off** (bundle, ToC, preserve-dirs, upload src)

---

## 📊 Summary

### Active Tickets (Priority Order)

1. **RMQ-RMDOC-WEB-001** — remarquee-ui web validation tool
   - **Status:** Nearly complete, needs validation history UI
   - **Next:** Implement validation history endpoint/UI, test with V6 document (cpage-pdf.rmdoc has V6 annotations)

2. **RMQ-0004** — Port .rmdoc parsing + rendering to Go
   - **Status:** Foundation complete, V6 parsing/rendering needed
   - **Next:** Complete legacy pipeline decision, start V6 parsing

3. **GITCOMMIT-XXXX** — Build Go CLI for safe git commits
   - **Status:** Not started (design + prototype only)
   - **Next:** Confirm scope, start with `repo` command

### Completed Tickets ✅

- RMQ-0003: Port remarkable-upload.py to Go
- RMQ-0005: remarquee-upload next features

---

## 🎯 Recommended Focus

**Immediate (Today):**
1. Complete RMQ-RMDOC-WEB-001 Phase 4: Validation history UI
2. Test RMQ-RMDOC-WEB-001 internal structure display with V6 document (cpage-pdf.rmdoc)

**Short-term (This Week):**
1. RMQ-0004: Complete legacy pipeline decision + start V6 parsing
2. RMQ-0004: Add pure V6 notebook fixture when cloud access available

**Medium-term (Next Week):**
1. RMQ-0004: V6 rendering + merge algorithm
2. GITCOMMIT-XXXX: Start implementation (confirm scope first)

---

## 📝 Notes

- **Most recent work:** RMQ-RMDOC-WEB-001 internal structure display (Step 7)
- **Most complete:** RMQ-RMDOC-WEB-001 (nearly done, just needs V6 fixture + history UI)
- **Biggest gap:** RMQ-0004 V6 parsing/rendering (foundation exists, but V6 pipeline incomplete)
- **Newest ticket:** GITCOMMIT-XXXX (not started, needs scope confirmation)

