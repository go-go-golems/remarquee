# Changelog

## 2026-04-07

- Initial workspace created


## 2026-04-07

Initial investigation: mapped all timestamp sources in reMarkable format (cloud stat, .metadata, .rm CRDT IDs, cPages). Confirmed no per-page/per-stroke wall-clock timestamps exist. Cloud modified_client == .metadata lastModified. Designed 3-phase cloud activity command.

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/remarquee/pkg/rmdoc/content.go — Examined legacy vs cPages parsing
- /home/manuel/code/wesen/corporate-headquarters/remarquee/pkg/rmdoc/rmv6_crdt_sequence.go — Confirmed CRDT IDs are not wall-clock timestamps

