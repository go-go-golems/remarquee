# Changelog

## 2026-03-28

- Initial workspace created
- Added primary design doc describing how to support `--cloud` for `rmdoc render-v6` and `rmdoc render-legacy` while preserving the current `pkg/rmdoc` package boundary.
- Added CLI review document covering command overlap, likely debug-only verbs, and long-term tightening opportunities.
- Added diary entry capturing the investigation path and delivery workflow.

## 2026-03-28

Added RMQ-0016 design and review deliverables covering cloud-backed rmdoc rendering and CLI verb cleanup guidance.

### Related Files

- /home/manuel/workspaces/2026-03-28/remarquee-render-cloud/remarquee/ttmp/2026/03/28/RMQ-0016--add-cloud-flag-to-rmdoc-render-commands/analysis/01-cli-verb-review-and-tightening-recommendations.md — CLI review deliverable
- /home/manuel/workspaces/2026-03-28/remarquee-render-cloud/remarquee/ttmp/2026/03/28/RMQ-0016--add-cloud-flag-to-rmdoc-render-commands/design-doc/01-design-and-implementation-guide-for-cloud-backed-rmdoc-rendering.md — Primary design and implementation guide
- /home/manuel/workspaces/2026-03-28/remarquee-render-cloud/remarquee/ttmp/2026/03/28/RMQ-0016--add-cloud-flag-to-rmdoc-render-commands/reference/01-diary.md — Diary entry for the investigation


## 2026-03-28

Validated RMQ-0016 with docmgr doctor and uploaded the ticket bundle to reMarkable at /ai/2026/03/28/RMQ-0016.

### Related Files

- /home/manuel/workspaces/2026-03-28/remarquee-render-cloud/remarquee/ttmp/2026/03/28/RMQ-0016--add-cloud-flag-to-rmdoc-render-commands/changelog.md — Ticket validation and upload recorded
- /home/manuel/workspaces/2026-03-28/remarquee-render-cloud/remarquee/ttmp/2026/03/28/RMQ-0016--add-cloud-flag-to-rmdoc-render-commands/tasks.md — Delivery checklist completed
- /home/manuel/workspaces/2026-03-28/remarquee-render-cloud/remarquee/ttmp/vocabulary.yaml — Added topic vocabulary required by doctor


## 2026-03-28

Implemented reusable rmcloud download plumbing and added shared cloud input resolution to both rmdoc render commands. Commits: d5c71ad, 2a9d770, 367171d, 2a922b8.

### Related Files

- /home/manuel/workspaces/2026-03-28/remarquee-render-cloud/remarquee/cmd/remarquee/cmds/cloud/get.go — Now reuses rmcloud download helper
- /home/manuel/workspaces/2026-03-28/remarquee-render-cloud/remarquee/cmd/remarquee/cmds/rmdoc/input_resolver.go — Shared local/cloud input resolver
- /home/manuel/workspaces/2026-03-28/remarquee-render-cloud/remarquee/cmd/remarquee/cmds/rmdoc/render_legacy.go — Legacy render command wired to resolver
- /home/manuel/workspaces/2026-03-28/remarquee-render-cloud/remarquee/cmd/remarquee/cmds/rmdoc/render_v6.go — V6 render command wired to resolver
- /home/manuel/workspaces/2026-03-28/remarquee-render-cloud/remarquee/pkg/rmcloud/download.go — Reusable remote-download helper

