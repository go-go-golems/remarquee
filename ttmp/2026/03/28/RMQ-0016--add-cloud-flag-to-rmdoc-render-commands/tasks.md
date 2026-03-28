# Tasks

## Completed

- [x] Create ticket `RMQ-0016` for cloud-backed `rmdoc` rendering work
- [x] Write a detailed design and implementation guide for adding `--cloud` to `rmdoc render-v6` and `rmdoc render-legacy`
- [x] Write a separate CLI verb review covering tightening and likely deprecation candidates
- [x] Write an investigation diary entry capturing commands, reasoning, and review guidance

## Remaining Ticket Follow-Ups

### 1. Task Breakdown And Execution Hygiene

- [ ] Replace the coarse implementation bullets with a step-by-step execution checklist
- [ ] Record implementation progress in the diary after each major slice
- [ ] Commit stable slices separately so the history mirrors the task breakdown

### 2. Reusable Cloud Download Primitive

- [ ] Add a reusable helper in `pkg/rmcloud` to download a document by remote path into a caller-provided directory
- [ ] Return useful metadata from that helper so callers can report source information if needed
- [ ] Refactor `cmd/remarquee/cmds/cloud/get.go` to use the new helper without changing its user-facing behavior
- [ ] Add focused tests for the new helper if a clean seam is available; otherwise document the seam and cover behavior indirectly

### 3. Shared RMDoc Input Resolution

- [ ] Add a shared `CloudInputSettings`/equivalent struct in `cmd/remarquee/cmds/rmdoc`
- [ ] Add a resolver that turns either a local path or a remote cloud path into a local working `.rmdoc` path
- [ ] Make the resolver create and clean up temporary download directories in cloud mode
- [ ] Keep the resolver package-local to the `rmdoc` command layer so `pkg/rmdoc` remains transport-agnostic

### 4. Wire `rmdoc render-v6`

- [ ] Add `--cloud`, `--non-interactive`, and `--reauth` to `rmdoc render-v6`
- [ ] Refactor `Run` and `RunIntoGlazeProcessor` through a shared execution helper so cloud resolution only lives in one place
- [ ] Preserve current schema validation, output naming, `--force`, and glaze output columns
- [ ] Add or extend tests that cover the new orchestration path

### 5. Wire `rmdoc render-legacy`

- [ ] Add `--cloud`, `--non-interactive`, and `--reauth` to `rmdoc render-legacy`
- [ ] Refactor `Run` and `RunIntoGlazeProcessor` through a shared execution helper
- [ ] Preserve legacy-specific flags and current output behavior
- [ ] Add or extend tests that cover the new orchestration path

### 6. Validation

- [ ] Run targeted tests for the touched packages
- [ ] Run command-level smoke validation for local mode
- [ ] Run command-level smoke validation for cloud mode if credentials and a suitable remote fixture are available
- [ ] Verify there are no regressions in `cloud get`

### 7. Follow-Up Decision Capture

- [ ] Decide whether `inspect`, `build-background`, and `render-v6-png` should adopt the same resolver now or remain follow-up work
- [ ] Update the ticket docs with any implementation-specific adjustments to the original design

## Delivery

- [x] Relate key files to the new docs
- [x] Update ticket index and changelog with final summaries
- [x] Run `docmgr doctor --ticket RMQ-0016 --stale-after 30`
- [x] Upload the ticket bundle to reMarkable and verify the remote listing
