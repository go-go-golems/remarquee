# Tasks

## TODO

- [ ] Open PR for branch `task/remarquee-fix-root-schema` (commit `d08e2e9`) to close issue #23
- [ ] TODO(#23): once ddvk/rmapi#77 merges to ddvk master, re-point the `go.mod` `replace` at the merge commit / ddvk tag and drop the `FNStudios-NI/rmapi` contributor-fork dependency

## DONE

- [x] Investigate issue #23 and confirm the bug is live at origin/main tip (`183a9d3`)
- [x] Create docmgr ticket RMQ-0023 and intern-level design doc
- [x] Apply the fix: bump `go.mod` replace pin to `FNStudios-NI/rmapi@v0.0.0-20260817154736-f295d5466978` (head of ddvk/rmapi#77)
- [x] Verify the fix live: `cloud mkdir` (Add), second `Add`, `cloud rm` (Remove/swap-delete) all succeed
- [x] Run `go vet` + `go test ./pkg/rmcloud/... ./cmd/remarquee/cmds/upload/...` — PASS
- [x] Commit as `d08e2e9`
- [x] Write investigation diary
- [x] Relate key files to the design doc
- [x] Run `docmgr doctor --ticket RMQ-0023 --stale-after 30`
- [x] Upload the design-doc bundle to reMarkable `/ai/2026/08/17/RMQ-0023`
- [x] Write Obsidian vault deep-dive report (textbook style) and push go-go-parc vault
