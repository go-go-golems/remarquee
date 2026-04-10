# Changelog

## 2026-04-10

- Initial workspace created
- Created `RMQ-0017` for the remarquee Glazed facade migration
- Scanned the codebase and identified 23 affected Go files plus legacy test helpers and OCR/Geppetto integration as the main special case
- Added `design/01-implementation-guide-glazed-facade-migration.md` with migration strategy, execution order, rename map, and validation plan
- Replaced the placeholder `tasks.md` with a detailed execution checklist grouped by cloud commands, dual-mode commands, rmdoc commands, tests, OCR, and delivery


## 2026-04-10

Step 2: migrated the entire cloud command package plus Glazed/Geppetto dependency bump to the facade API (commit 472aba7)

### Related Files

- /home/manuel/code/wesen/go-go-golems/remarquee/cmd/remarquee/cmds/cloud/account.go — Representative simple cloud migration
- /home/manuel/code/wesen/go-go-golems/remarquee/cmd/remarquee/cmds/cloud/find.go — Representative dual-mode cloud migration
- /home/manuel/code/wesen/go-go-golems/remarquee/cmd/remarquee/cmds/cloud/rmapi.go — Shared auth-settings tag migration
- /home/manuel/code/wesen/go-go-golems/remarquee/go.mod — Dependency bump required for the new Glazed and Geppetto APIs

