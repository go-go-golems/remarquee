# Tasks

## TODO

- [x] Add tasks here

- [ ] Phase 1: Cloud-only timeline command (no downloads, just cloud ls JSON → formatted per-day output)
- [ ] Phase 2: Full activity scan with metadata (download, extract .metadata, detect .rm annotations, classify interaction)
- [ ] Phase 3: Caching layer (skip re-download if modified_client unchanged)
- [ ] Extract .metadata parser into reusable internal/pkg (epoch ms → time.Time, handle empty/zero values)
