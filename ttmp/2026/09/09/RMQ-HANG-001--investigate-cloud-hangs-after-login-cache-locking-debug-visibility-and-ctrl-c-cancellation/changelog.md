# Changelog

## 2026-09-09

- Initial workspace created


## 2026-09-09

Created investigation ticket, source-backed analysis, diary, and remediation tasks. Identified eager account tree sync, bootstrap logging gap, disconnected cancellation, and transport buffering risks; live hang and cache/SIGINT causes remain unconfirmed.


## 2026-09-09

Design checkpoint: specify stderr phase/elapsed/HTTP progress with sync request cancellation; exact document totals require a future rmapi callback API.


## 2026-09-09

Implemented first sync-progress increment (c9f621e): stderr heartbeat/HTTP activity, streaming-safe metadata logs before sync, explicit request contexts, and bounded SIGINT shutdown. CLI/library tests, focused race tests, vet and help smoke pass; full suite blocked by missing UI frontend/dist. Verified maintained upstream adds no progress/context hooks relative to pin; exact document counts and cooperative auth remain open.

## 2026-09-09

Built real frontend assets with pinned pnpm 10.15.1 via npm exec. Whole-repository go test, go build, and go vet now pass. Installed updated CLI at ~/.local/bin/remarquee (source e6d0fc9) and verified account help; live cloud behavior remains unverified. Generated assets remain ignored.

## 2026-09-09

Opened PR #27 (https://github.com/go-go-golems/remarquee/pull/27) from fix/cloud-sync-progress-cancellation to main using the Pinocchio create-pull-request format. Preserved YAML title/body/changelog/release notes and documented API changes, local validation, and remaining limitations.
