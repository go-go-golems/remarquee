# Changelog

## 2026-08-17

- Initial workspace created.
- Investigated issue #23: confirmed the root-index sorting bug is live at origin/main tip (`183a9d3`); `go.mod:22` still pins the pre-fix `marcobarcelos/rmapi` fork; ddvk/rmapi#77 still `OPEN`.
- Authored intern-level design/implementation guide (`design-doc/01-root-index-sorting-bug-analysis-design-and-implementation-guide.md`) covering the sync15 stack, the root-cause asymmetry, fix options, and a 7-step implementation guide.
- Applied the fix: repointed `go.mod` replace directive from `marcobarcelos/rmapi` (pre-fix) to `FNStudios-NI/rmapi@v0.0.0-20260817154736-f295d5466978`, the head of ddvk/rmapi#77, which adds `sort.Slice(t.Docs, ...)` in `HashTree.IndexReader()` and `sort.Slice(d.Files, ...)` in `BlobDoc.IndexReaderWithSchema()`. Drop-in module-compatible bump; v4-always behavior preserved. No application code changed.
- Verified live: exact issue #23 repro (`cloud mkdir /ai/2026/08/17`) now succeeds; second `Add` and `Remove` (swap-delete) paths succeed; cleanup left no trace. `go vet` + tests pass. Committed as `d08e2e9`.
- Wrote investigation diary (`reference/01-investigation-diary.md`).
- Related key files to the design doc.
- Uploaded the design-doc bundle to reMarkable.
- Wrote Obsidian vault deep-dive report and pushed go-go-parc vault.
