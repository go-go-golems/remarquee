# Changelog

## 2026-05-03

- Initial workspace created


## 2026-05-03

Created ticket, investigated remarquee upload/cloud commands, identified sync gap, wrote bash prototype and design doc

### Related Files

- /home/manuel/code/wesen/claw-stuff/ttmp/2026/05/03/RM-SYNC-001--obsidian-vault-to-remarkable-sync-script/design-doc/01-obsidian-to-remarkable-sync-analysis-design-and-implementation-guide.md — Primary design document


## 2026-05-03

Resumed RM-SYNC-001, fixed docmgr vocabulary hygiene for obsidian/sync topics, reran doctor successfully, verified existing reMarkable bundle upload, and checked off validation/upload tasks.

### Related Files

- /home/manuel/workspaces/2026-05-03/add-upload-sync/remarquee/ttmp/2026/05/03/RM-SYNC-001--obsidian-vault-to-remarkable-sync-script/tasks.md — Marked doctor and reMarkable upload verification tasks complete
- /home/manuel/workspaces/2026-05-03/add-upload-sync/remarquee/ttmp/vocabulary.yaml — Added obsidian and sync topic vocabulary entries for doctor validation


## 2026-05-03

Implemented pure sync planning helpers and unit tests for upload/skip/stale/orphan decisions using full remote path keys.

### Related Files

- /home/manuel/workspaces/2026-05-03/add-upload-sync/remarquee/cmd/remarquee/cmds/upload/sync_plan.go — Pure sync planning model and delta classification helpers
- /home/manuel/workspaces/2026-05-03/add-upload-sync/remarquee/cmd/remarquee/cmds/upload/sync_plan_test.go — Unit coverage for sync plan actions and path-key behavior
- /home/manuel/workspaces/2026-05-03/add-upload-sync/remarquee/ttmp/2026/05/03/RM-SYNC-001--obsidian-vault-to-remarkable-sync-script/tasks.md — Marked sync planning helper and test tasks complete


## 2026-05-03

Added the upload sync Cobra command, wired it under upload, and implemented dry-run plan reporting backed by the sync planner and remote index construction.

### Related Files

- /home/manuel/workspaces/2026-05-03/add-upload-sync/remarquee/cmd/remarquee/cmds/upload/root.go — Registered the upload sync subcommand
- /home/manuel/workspaces/2026-05-03/add-upload-sync/remarquee/cmd/remarquee/cmds/upload/sync.go — New upload sync command
- /home/manuel/workspaces/2026-05-03/add-upload-sync/remarquee/cmd/remarquee/cmds/upload/sync_plan_test.go — Added command registration and plan output tests
- /home/manuel/workspaces/2026-05-03/add-upload-sync/remarquee/ttmp/2026/05/03/RM-SYNC-001--obsidian-vault-to-remarkable-sync-script/tasks.md — Marked upload sync command and wiring tasks complete


## 2026-05-03

Enabled upload sync execution for upload items and forced stale replacements, while keeping orphan deletion explicitly non-mutating for now.

### Related Files

- /home/manuel/workspaces/2026-05-03/add-upload-sync/remarquee/cmd/remarquee/cmds/upload/sync.go — Executes sync plans by converting/uploading upload items and force-overwriting stale items
- /home/manuel/workspaces/2026-05-03/add-upload-sync/remarquee/cmd/remarquee/cmds/upload/sync_plan.go — Remote entries now carry node handles for stale replacement execution
- /home/manuel/workspaces/2026-05-03/add-upload-sync/remarquee/ttmp/2026/05/03/RM-SYNC-001--obsidian-vault-to-remarkable-sync-script/tasks.md — Added and checked sync execution task


## 2026-05-03

Refactored shared PDF upload-to-remote behavior out of upload md and upload sync into a common helper.

### Related Files

- /home/manuel/workspaces/2026-05-03/add-upload-sync/remarquee/cmd/remarquee/cmds/upload/md.go — Reuses shared PDF upload helper
- /home/manuel/workspaces/2026-05-03/add-upload-sync/remarquee/cmd/remarquee/cmds/upload/sync.go — Reuses shared PDF upload helper during sync execution
- /home/manuel/workspaces/2026-05-03/add-upload-sync/remarquee/cmd/remarquee/cmds/upload/upload_helpers.go — Shared uploadPDFToRemote helper for mkdir/cache/upload/add-document behavior
- /home/manuel/workspaces/2026-05-03/add-upload-sync/remarquee/ttmp/2026/05/03/RM-SYNC-001--obsidian-vault-to-remarkable-sync-script/tasks.md — Marked shared upload behavior refactor complete


## 2026-05-03

Attempted live upload sync dry-run validation with a temporary markdown fixture; blocked because go run ./cmd/remarquee currently fails on an unrelated geppetto/glazed dependency mismatch.

### Related Files

- /home/manuel/workspaces/2026-05-03/add-upload-sync/remarquee/ttmp/2026/05/03/RM-SYNC-001--obsidian-vault-to-remarkable-sync-script/reference/01-investigation-diary.md — Recorded blocked CLI validation attempt and exact error


## 2026-05-03

Diagnosed the live dry-run build issue as go.work selecting local glazed; validated upload sync dry-run successfully with GOWORK=off.

### Related Files

- /home/manuel/workspaces/2026-05-03/add-upload-sync/go.work — Workspace file that selects local glazed and triggers the geppetto/glazed mismatch
- /home/manuel/workspaces/2026-05-03/add-upload-sync/remarquee/ttmp/2026/05/03/RM-SYNC-001--obsidian-vault-to-remarkable-sync-script/tasks.md — Marked dry-run validation complete after GOWORK=off run


## 2026-05-03

Migrated remarquee Glazed/Cobra commands to the new ConfigPlanBuilder-based config loading API, added a shared appconfig parser config, and updated Glazed/Geppetto module versions so workspace builds use compatible APIs.

### Related Files

- /home/manuel/workspaces/2026-05-03/add-upload-sync/remarquee/cmd/remarquee/cmds/cloud/ls.go — Representative cloud command migrated from custom default middleware to app config parser config
- /home/manuel/workspaces/2026-05-03/add-upload-sync/remarquee/cmd/remarquee/cmds/ocr/root.go — OCR command migrated to app config parser config
- /home/manuel/workspaces/2026-05-03/add-upload-sync/remarquee/cmd/remarquee/cmds/rmdoc/render_v6.go — Representative rmdoc command migrated to app config parser config
- /home/manuel/workspaces/2026-05-03/add-upload-sync/remarquee/cmd/remarquee/internal/appconfig/parser.go — Shared CobraParserConfig with AppName and ConfigPlanBuilder for remarquee config discovery
- /home/manuel/workspaces/2026-05-03/add-upload-sync/remarquee/go.mod — Updated Glazed and Geppetto versions for the new config plan API


## 2026-05-04

Added --workers N to upload md for parallel pandoc conversion in pdf-only and upload modes, keeping rmapi uploads sequential.

### Related Files

- /home/manuel/workspaces/2026-05-03/add-upload-sync/remarquee/cmd/remarquee/cmds/upload/conversion_workers.go — Worker-pool conversion helper for markdown-to-PDF jobs
- /home/manuel/workspaces/2026-05-03/add-upload-sync/remarquee/cmd/remarquee/cmds/upload/md.go — Added workers flag and integrated parallel conversion before upload
- /home/manuel/workspaces/2026-05-03/add-upload-sync/remarquee/cmd/remarquee/cmds/upload/md_test.go — Tests for workers flag validation and conversion job path behavior
- /home/manuel/workspaces/2026-05-03/add-upload-sync/remarquee/ttmp/2026/05/03/RM-SYNC-001--obsidian-vault-to-remarkable-sync-script/tasks.md — Marked workers task complete

