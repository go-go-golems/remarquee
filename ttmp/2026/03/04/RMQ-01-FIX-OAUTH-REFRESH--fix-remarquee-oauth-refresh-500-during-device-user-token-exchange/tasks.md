# Tasks

## Investigation + Implementation

- [x] Create ticket workspace + primary docs (design doc, diary, playbook)
- [x] Reproduce/confirm error path and capture logs/traces (redacted where needed)
- [x] Identify exact failing code path (device token → user token exchange)
- [x] Implement fix so auth refresh failures do not hard-exit the process
- [x] Add retry/backoff so transient 5xx is survivable

## Documentation + Delivery

- [x] Write intern-grade analysis / design / implementation guide (diagrams + pseudocode + file references)
- [x] Update diary with chronological steps (include failures verbatim)
- [x] Relate key files in docmgr + update changelog entries
- [x] Run `docmgr doctor` cleanly
- [x] Upload bundle to reMarkable (dry-run first, then upload)
