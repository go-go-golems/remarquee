---
Title: 'RMQ-0007 Tasks — Template backgrounds'
Ticket: RMQ-0007
Status: active
Topics:
  - remarkable
  - rmdoc
  - rendering
  - templates
DocType: task-list
Intent: short-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: >
  Task list for implementing and validating notebook template background rendering.
LastUpdated: 2026-01-10
---

# RMQ-0007 Tasks — Template backgrounds

## Goals

- Render template backgrounds for notebook pages (at least the built-in templates we see in fixtures).
- Make template rendering **deterministic** and **comparable** (golden-friendly).
- Decide and document the **template→page size mapping** (screen units vs PDF points vs remarks scaling).

## Tasks

### 1) Inventory + scope

- [ ] Enumerate template names we need to support (from fixtures + user docs).
- [ ] Decide which are “must support now” vs “later”.

### 2) Rendering strategy (design decision)

- [ ] Decide representation:
  - [ ] SVG→PDF pipeline (closest to how many tools do it)
  - [ ] Programmatic PDF drawing (dots/lines as vectors; faster, no SVG deps)
  - [ ] Raster background (least ideal for PDFs; last resort)
- [ ] Define coordinate mapping:
  - [ ] RM screen units (1404×1872) → PDF points (72/226 scale or other)
  - [ ] How this interacts with the existing “remarks 0.75 px→pt” behavior
- [ ] Document the decision in `reference/`.

### 3) Implementation

- [ ] Load template name per page (from `.content` cPages template field and/or `.pagedata`).
- [ ] Implement background generation for:
  - [ ] “P Dots S” (dots)
  - [ ] at least one lined template
  - [ ] at least one grid template
- [ ] Integrate into `BuildBackgroundPDF` / V6 merge pipeline.

### 4) Validation

- [ ] Add/extend fixtures that exercise templates explicitly.
- [ ] Add golden tests (remarks reference) that include template backgrounds.
- [ ] Add a device-review workflow (PDF transport acceptable initially) to verify template density/spacing.

### 5) Documentation

- [ ] Update playbook with “templates: how to debug”.
- [ ] Update changelog and diary as decisions are made.


