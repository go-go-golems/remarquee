# Changelog

## 2026-03-28

- Initial workspace created
- Added a named markdown layout preset (`default|editor`) in `pkg/mdpdf`, including editor-specific geometry and spacing overrides layered through an extra LaTeX header.
- Wired `--layout` into `remarquee upload md` and `remarquee upload bundle` through a shared option builder so both commands resolve presets consistently while still honoring explicit overrides.
- Added focused tests for layout validation and dry-run visibility and updated the embedded markdown upload reference docs.
- Authored a design doc, diary, and intern playbook for the feature and its extension path.

## 2026-03-28

Implemented editor-friendly markdown upload layout, tests, and user-facing docs for markdown upload and bundle workflows.

### Related Files

- /home/manuel/workspaces/2026-03-28/remarquee-draft-layout/remarquee/cmd/remarquee/cmds/upload/bundle.go — Added --layout to markdown bundle upload
- /home/manuel/workspaces/2026-03-28/remarquee-draft-layout/remarquee/cmd/remarquee/cmds/upload/md.go — Added --layout to markdown upload
- /home/manuel/workspaces/2026-03-28/remarquee-draft-layout/remarquee/pkg/mdpdf/layout.go — Added named preset catalog

Commit: `14858d3` — `upload: add editor layout preset for markdown PDFs`


## 2026-03-28

Validated the ticket with docmgr doctor, uploaded the bundled analysis/playbook/diary with the editor layout, and verified the remote document under /ai/2026/03/28/RMQ-0014.

### Related Files

- /home/manuel/workspaces/2026-03-28/remarquee-draft-layout/remarquee/ttmp/2026/03/28/RMQ-0014--add-editor-friendly-markdown-upload-layout/design-doc/01-analysis-and-design-for-editor-friendly-markdown-upload-layout.md — Included in uploaded bundle
- /home/manuel/workspaces/2026-03-28/remarquee-draft-layout/remarquee/ttmp/2026/03/28/RMQ-0014--add-editor-friendly-markdown-upload-layout/playbook/01-intern-implementation-guide-for-markdown-upload-editor-layout.md — Included in uploaded bundle
- /home/manuel/workspaces/2026-03-28/remarquee-draft-layout/remarquee/ttmp/2026/03/28/RMQ-0014--add-editor-friendly-markdown-upload-layout/reference/01-diary.md — Included in uploaded bundle

## 2026-03-28

Converted RMQ-0014 into an explicit task-by-task workflow with commit boundaries, updated the diary to record the first feature commit and the pre-commit hook failure, and finalized the ticket docs/bookkeeping as a focused follow-up commit.
