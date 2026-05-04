# Tasks

## TODO

- [x] Create docmgr ticket RM-SYNC-001
- [x] Investigate remarquee upload md / bundle / cloud find / stat / ls commands
- [x] Map Obsidian vault structure and naming conventions
- [x] Identify architectural gaps (wasteful conversion, no pre-flight delta, no sync verb)
- [x] Write bash prototype sync script
- [x] Write comprehensive design/analysis doc for intern onboarding
- [x] Relate key source files to design doc
- [x] Update changelog
- [x] Run docmgr doctor and fix any issues
- [x] Upload ticket docs to reMarkable
- [x] Diagnose sync performance (pandoc bottleneck, sequential processing, rmapi tree refresh)
- [x] Implement sync planning helpers for local/remote key computation
- [x] Add unit tests for sync planning: upload, skip, stale, orphan, preserve-dirs, flatten
- [x] Implement `remarquee upload sync` command with dry-run delta reporting
- [x] Wire `upload sync` into the upload command tree
- [x] Validate `upload sync --dry-run` against temporary markdown fixtures
- [x] Implement `upload sync` execution for upload/stale delta items
- [x] Refactor or document shared upload-md/sync conversion behavior
- [x] Add `--workers N` parallel conversion to `upload md`
- [x] Add mtime comparison and orphaned-file cleanup to sync execution
- [x] Investigate rmapi tree refresh suppression for bulk uploads

- [x] Migrate remarquee Cobra commands to Glazed config plan loading API
- [x] Live-upload Obsidian Projects/2026 vault reports to /Vault/Projects/2026 on reMarkable
