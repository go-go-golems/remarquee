# Tasks

## TODO

- [x] 1. Phase 1: extract `annotationCanvasBBox` + `overlayOnlyPageGeometry` helpers in pkg/rmdoc/render/v6_merge_background.go (byte-identical refactor)
- [x] 2. Phase 2: implement `RenderRMDocV6AnnotationsOnlyWithInfo` in pkg/rmdoc/render/v6_annotations_only.go + unit tests
- [ ] 3. Phase 3: wire `--annotations-only` flag in cmd/remarquee/cmds/rmdoc/render_v6.go + glaze field + CLI tests
- [ ] 4. Phase 3b: wire `--annotations-only` flag in cmd/remarquee/cmds/rmdoc/render_v6_png.go (scope extension per user)
- [ ] 5. Phase 4: README/help updates + manual validation on TTC Garden workbook via --cloud
- [ ] 6. Full test suite green (`go test ./... -count=1`) and lint clean

## DONE

- [x] 6. Ticket workspace created (RMQ-0021)
- [x] 7. Evidence gathering: both render verbs, render pipeline, rmapi legacy semantics mapped with file:line anchors
- [x] 8. Real-world reproduction: TTC Garden workbook pages 1-3 (render-v6 works; render-legacy fails "Unknown header")
- [x] 9. Design/implementation guide written (design-doc/01)
- [x] 10. Diary started (reference/01)
