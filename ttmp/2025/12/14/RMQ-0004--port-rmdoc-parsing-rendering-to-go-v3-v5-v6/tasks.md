# Tasks

## TODO

- [ ] Confirm scope: required outputs (PDF only vs PDF+PNG) and fidelity expectations (pixel match vs “good enough”)
- [ ] Implement `.rmdoc` format detection (legacy vs `cPages`) and document-level `PageRef` plan
- [ ] Implement legacy pipeline (V3/V5): reuse rmapi archive + renderer behind new interfaces
- [ ] Implement `cPages` parsing + deterministic page ordering (filter deleted pages)
- [ ] Implement V6 `.rm` parsing (port `rmscene` block reader + scene tree; start with strokes)
- [ ] Implement V6 rendering to PDF (stroke rendering + merge with background PDF/template)
- [ ] Implement smart highlights for V6 (PenColor mapping + PDF highlight annotations)
- [ ] Add fixtures and golden tests (at least 1 legacy PDF doc + 1 V6 notebook doc)
- [ ] Wire into a `remarquee` command (likely `remarquee upload` / a new `remarquee render` command)

