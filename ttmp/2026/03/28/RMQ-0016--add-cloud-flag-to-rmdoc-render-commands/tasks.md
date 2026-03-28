# Tasks

## Completed

- [x] Create ticket `RMQ-0016` for cloud-backed `rmdoc` rendering work
- [x] Write a detailed design and implementation guide for adding `--cloud` to `rmdoc render-v6` and `rmdoc render-legacy`
- [x] Write a separate CLI verb review covering tightening and likely deprecation candidates
- [x] Write an investigation diary entry capturing commands, reasoning, and review guidance

## Remaining Ticket Follow-Ups

### 1. Task Breakdown And Execution Hygiene

- [x] Replace the coarse implementation bullets with a step-by-step execution checklist
- [x] Record implementation progress in the diary after each major slice
- [x] Commit stable slices separately so the history mirrors the task breakdown

### 2. Reusable Cloud Download Primitive

- [x] Add a reusable helper in `pkg/rmcloud` to download a document by remote path into a caller-provided directory
- [x] Return useful metadata from that helper so callers can report source information if needed
- [x] Refactor `cmd/remarquee/cmds/cloud/get.go` to use the new helper without changing its user-facing behavior
- [x] Add focused tests for the new helper if a clean seam is available; otherwise document the seam and cover behavior indirectly

### 3. Shared RMDoc Input Resolution

- [x] Add a shared `CloudInputSettings`/equivalent struct in `cmd/remarquee/cmds/rmdoc`
- [x] Add a resolver that turns either a local path or a remote cloud path into a local working `.rmdoc` path
- [x] Make the resolver create and clean up temporary download directories in cloud mode
- [x] Keep the resolver package-local to the `rmdoc` command layer so `pkg/rmdoc` remains transport-agnostic

### 4. Wire `rmdoc render-v6`

- [x] Add `--cloud`, `--non-interactive`, and `--reauth` to `rmdoc render-v6`
- [x] Refactor `Run` and `RunIntoGlazeProcessor` through a shared execution helper so cloud resolution only lives in one place
- [x] Preserve current schema validation, output naming, `--force`, and glaze output columns
- [x] Add or extend tests that cover the new orchestration path

### 5. Wire `rmdoc render-legacy`

- [x] Add `--cloud`, `--non-interactive`, and `--reauth` to `rmdoc render-legacy`
- [x] Refactor `Run` and `RunIntoGlazeProcessor` through a shared execution helper
- [x] Preserve legacy-specific flags and current output behavior
- [x] Add or extend tests that cover the new orchestration path

### 6. Validation

- [x] Run targeted tests for the touched packages
- [x] Run command-level smoke validation for local mode
- [x] Run command-level smoke validation for cloud mode if credentials and a suitable remote fixture are available
- [x] Verify there are no regressions in `cloud get`

### 7. Follow-Up Decision Capture

- [x] Decide whether `inspect`, `build-background`, and `render-v6-png` should adopt the same resolver now or remain follow-up work
- [x] Update the ticket docs with any implementation-specific adjustments to the original design

## Delivery

- [x] Relate key files to the new docs
- [x] Update ticket index and changelog with final summaries
- [x] Run `docmgr doctor --ticket RMQ-0016 --stale-after 30`
- [x] Upload the ticket bundle to reMarkable and verify the remote listing
