# Tasks

## TODO

- **Context unification (follow analysis/02)**
  - Ref: `/home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/24/001-REVIEW-CODE--review-remarquee-codebase-architecture/analysis/02-flow-a-context-propagation-audit-context-background-usage-unification-plan.md`
- [x] Replace `context.Background()` with Cobra `cmd.Context()` in `remarquee/cmd/remarquee/cmds/rmdoc/inspect.go` (`(*InspectCommand).Run`)
- [x] Replace `context.Background()` with Cobra `cmd.Context()` in `remarquee/cmd/remarquee/cmds/rmdoc/build_background.go` (`(*BuildBackgroundCommand).Run`) for `OpenFile`
- [x] Replace `context.Background()` with Cobra `cmd.Context()` in `remarquee/cmd/remarquee/cmds/rmdoc/build_background.go` (`(*BuildBackgroundCommand).Run`) for `BuildBackgroundPDF`
- [x] Replace `context.Background()` with `r.Context()` in `remarquee/cmd/remarquee-ui/api/inspect.go` (`HandleInspect`)
- [x] Replace `context.Background()` with `r.Context()` in `remarquee/cmd/remarquee-ui/api/internal_structure.go` (`HandleInternalStructure`)
- [x] Replace `context.Background()` with `r.Context()` in `remarquee/cmd/remarquee-ui/api/render.go` (`HandleRenderBackground`)
- [x] Replace `context.Background()` with `r.Context()` in `remarquee/cmd/remarquee-ui/api/render.go` (`HandleRenderLegacy`)
- [x] Make `remarquee/pkg/rmdoc/open.go` honor `ctx` in `OpenReaderAt` (add `ctx.Err()` checks)
- [x] Make `remarquee/pkg/rmdoc/open.go` honor `ctx` during zip entry reads (replace `io.ReadAll` with context-aware read loop in `readZipFile`)
- [x] Make `remarquee/pkg/rmdoc/render/background.go` honor `ctx` in `BuildBackgroundPDF` (check `ctx.Err()` between pages)
- [x] Add tests covering cancellation (at least one for `OpenReaderAt` and one for `BuildBackgroundPDF`)
- [x] Decide/document cancellation semantics for rmapi legacy generator (cannot truly interrupt; only stop waiting)

- **Extract UI rmdoc introspection into `pkg/rmdoc/debug`**
- [x] Create new package `remarquee/pkg/rmdoc/debug` (new directory + doc.go)
- [x] Add `DetectRMVersionFromHeader([]byte) (string, bool)` to `pkg/rmdoc/debug` (move logic from UI)
- [x] Add helper to list archive entries (e.g. `ListArchiveFiles(...)`) to `pkg/rmdoc/debug`
- [x] Add helper to inspect `.rm` files inside archive (e.g. `InspectRMFiles(...)`) to `pkg/rmdoc/debug`
- [x] Refactor `remarquee/cmd/remarquee-ui/api/internal_structure.go` to use `pkg/rmdoc/debug` instead of duplicating zip + header sniff logic
- [x] Add unit tests for `pkg/rmdoc/debug` using existing UI `testdata` archives (`cmd/remarquee-ui/testdata/*`)

- **Cleanup `remarquee-ui` mux + path parsing**
- [ ] Refactor `remarquee/cmd/remarquee-ui/main.go` routing to avoid suffix checks and to use a consistent pattern per endpoint
- [ ] Refactor `remarquee/cmd/remarquee-ui/api/inspect.go` to avoid manual `strings.Split` path parsing (extract id via mux variables or shared helper)
- [ ] Refactor `remarquee/cmd/remarquee-ui/api/internal_structure.go` to avoid manual `strings.Split` path parsing (extract id via mux variables or shared helper)
- [ ] Refactor `remarquee/cmd/remarquee-ui/api/outputs.go` to avoid manual `strings.Split` path parsing (extract filename via mux variables or shared helper)
- [ ] Add/adjust handler tests to ensure no behavior regression (400 vs 404 vs 200) for routing/path parsing changes
