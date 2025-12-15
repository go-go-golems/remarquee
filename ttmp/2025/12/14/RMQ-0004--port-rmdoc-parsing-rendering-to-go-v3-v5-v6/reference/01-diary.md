---
Title: Diary
Ticket: RMQ-0004
Status: active
Topics:
    - backend
    - go
    - remarkable
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarquee/ttmp/2025/12/14/RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/reference/06-diary-rmdoc-format-analysis-and-go-reimplementation-prep.md
      Note: Prior research diary that this ticket builds on
    - Path: remarquee/ttmp/2025/12/14/RMQ-0004--port-rmdoc-parsing-rendering-to-go-v3-v5-v6/design-doc/01-design-go-rmdoc-data-model-and-apis.md
      Note: Design decisions and API proposal for this ticket
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-14T20:59:20.929388092-05:00
---


# Diary

## Goal

This diary tracks the step-by-step work of porting `.rmdoc` parsing and annotation rendering into Go for remarquee, using RMQ-0001’s research as the baseline.

## Step 1: Bootstrap ticket + write initial design (seeded from RMQ-0001 research)

This step creates a clean implementation ticket (RMQ-0004) and turns the prior research into an actionable design. The point is to stop “re-learning” the `.rmdoc`/`.rm` ecosystem and instead lock down the data model boundaries and APIs we’ll build against.

**Commit:** TBD (docs-only; will be filled once committed)

### What I did
- Created the RMQ-0004 docmgr ticket workspace.
- Created a design doc describing proposed Go packages, data structures, and APIs.
- Created this diary document and a first task list for the port.
- Linked the ticket overview back to RMQ-0001 primary research documents.

### Why
- We already did the hard “what’s inside `.rmdoc` and how does `remarks` render” work in RMQ-0001. This ticket should focus on **implementation**, not rediscovery.
- A clear data model + API boundary is the fastest way to make the port tractable and reviewable.

### What worked
- docmgr ticket creation + doc scaffolding makes it easy to keep design + diary + tasks in sync.
- The design doc captures the main architectural split we want: archive parsing vs annotation decoding vs rendering.

### What didn't work
- N/A (no implementation changes yet; this step is scaffolding + design).

### What I learned
- Even at the API-design stage, dual-format support (legacy V3/V5 + modern V6) should be treated as a first-class requirement, not a later “compat” add-on.
- Deterministic page ordering must come from `.content` (not filesystem iteration), because the Python reference implementation uses nondeterministic iteration in at least one place.

### What was tricky to build
- Choosing the right abstraction level: too “normalized” too early risks losing fidelity; too “raw” makes downstream features painful.
- The best compromise seems to be: **format-specific parsing + a small normalized surface** for rendering/extraction.

### What warrants a second pair of eyes
- The proposed package boundaries and the recommended “Proposal B” approach in the design doc:
  - Are we over-committing to unipdf for V6 rendering?
  - Should we keep a transitional Python-backed path for V6 longer than planned?
  - Do the proposed `PageRef` fields cover enough of the `cPages` edge cases (deleted pages, inserted pages, duplicates)?

### What should be done in the future
- Add fixture-based tests for both formats (downloaded from the device) and start a golden-output workflow.
- Start implementation with the archive layer + format detection + page plan, since everything else depends on it.

### Code review instructions
- Start with:
  - `design-doc/01-design-go-rmdoc-data-model-and-apis.md`
  - `tasks.md`
  - `index.md` (links to RMQ-0001 research)

### Technical details
- Primary research lives in RMQ-0001:
  - deep dive: `../RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/analysis/01-deep-dive-rmdoc-format-container-layout-parsing-png-rendering.md`
  - diary: `../RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/reference/06-diary-rmdoc-format-analysis-and-go-reimplementation-prep.md`
  - gap analysis: `../RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/scripts/go_reimplementation_gaps.md`
