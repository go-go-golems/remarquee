# Tasks

## TODO

### Carry-over from RMQ-0004 (unclosed)

- [ ] Confirm scope + acceptance criteria:
  - [ ] Required outputs: PDF only vs PDF+PNG
  - [ ] Fidelity target: pixel-perfect vs “good enough” (strokes-only vs highlights vs typed text)
  - [ ] Supported inputs: `.rmdoc` only vs unpacked exports
  - [ ] Validation workflow: where PASS/FAIL + notes are stored (UI sessions vs markdown logs)

- [ ] Template backgrounds:
  - [ ] Define template-to-page-size strategy (blank page constants vs render templates)

- [ ] Typed text:
  - [ ] Implement typed text parsing/render/extraction plan (RootTextBlock is parsed enough for anchors; decide next outputs)

- [ ] Golden test runner (task 53):
  - [ ] Decide baseline strategy:
    - [ ] Compare against `remarks`/`rmc` (reference implementation)
    - [ ] and/or commit goldens and regression-test against them
    - [ ] and/or structured diff of primitives (strokes/highlights/anchor offsets)
  - [ ] Implement `remarquee rmdoc validate` (or similar) that:
    - [ ] renders target fixture(s) through our pipeline
    - [ ] optionally renders through reference implementation (when available)
    - [ ] produces per-page PNGs and a diff report (tolerance-based)

- [ ] Interactive validation UI (RMQ-RMDOC-WEB-001 follow-up):
  - [ ] Decide whether validation sessions live in RMQ-RMDOC-WEB-001 or here; keep one source of truth

### New validation work (this ticket)

- [ ] Write a detailed testing/validation playbook doc (feature-by-feature) in `reference/`
- [ ] Add stage-level debug CLIs to avoid console spam:
  - [ ] `remarquee rmdoc v6-stats <file>` (counts: strokes, glyph ranges, groups, anchors)
  - [ ] `remarquee rmdoc v6-dump-highlights <file> [--page N]` (rects + color + x_translation)
  - [ ] `remarquee rmdoc v6-dump-strokes <file> [--page N]` (bbox + sample points)


