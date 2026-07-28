# Changelog

## 2026-07-28

- Created RMQ-0020 for the Pandoc YAML parser failure caused by ordinary Markdown `---` separators.
- Reproduced the failure with Pandoc 3.1.3 and confirmed that `--from=markdown-yaml_metadata_block` succeeds without source rewriting.
- Mapped the shared `pkg/mdpdf` pipeline, `upload md`, `upload bundle`, conversion workers, and existing batch-resilience behavior.
- Added the intern-ready design document with architecture diagrams, API sketches, pseudocode, decision records, phased implementation plan, acceptance criteria, and file references.
- Added the investigation diary with exact prompt context, commands, output, failures, and review instructions.
- Pending: implement the shared converter change, add regression tests, update user-facing docs, and record implementation evidence.

## 2026-07-28

Investigation and design complete: reproduced Pandoc 3.1.3 YAML parser failure, isolated ordinary thematic breaks as the trigger, and specified shared converter fix plus regression tests.

### Related Files

- /home/manuel/workspaces/2026-07-28/fix-remarquee-md/remarquee/pkg/mdpdf/pandoc.go — Central conversion choke point for the proposed input-format contract.
- /home/manuel/workspaces/2026-07-28/fix-remarquee-md/remarquee/ttmp/2026/07/28/RMQ-0020--make-markdown-to-pdf-conversion-resilient-to-pandoc-yaml-parser-errors/design-doc/01-intern-guide-resilient-markdown-to-pdf-conversion.md — Intern-ready implementation and validation guide.


## 2026-07-28

Validated RMQ-0020 with docmgr doctor and uploaded the documentation bundle to reMarkable at /ai/2026/07/28/RMQ-0020.

### Related Files

- ttmp/2026/07/28/RMQ-0020--make-markdown-to-pdf-conversion-resilient-to-pandoc-yaml-parser-errors/design-doc/01-intern-guide-resilient-markdown-to-pdf-conversion.md — Uploaded intern-ready design and implementation guide


## 2026-07-28

Step 5: Implemented shared Pandoc input-format contract and regression tests (commit 5bbc3411b7db7d4a0ceb82cb2bafb83e4b2f5626). Focused mdpdf tests pass with GOWORK=off GOTOOLCHAIN=auto; repository hook remains blocked by missing frontend/dist.

### Related Files

- /home/manuel/workspaces/2026-07-28/fix-remarquee-md/remarquee/pkg/mdpdf/pandoc.go — Disable yaml_metadata_block for generated Markdown


## 2026-07-28

Step 6: Added bundle and batch continuation regression tests and documented yaml_metadata_block behavior (commit 955626a31ec57b3141220a6f0dca1556d5a2426a). Focused tests pass; full suite is blocked only by missing cmd/remarquee-ui/frontend/dist.

### Related Files

- /home/manuel/workspaces/2026-07-28/fix-remarquee-md/remarquee/cmd/remarquee/cmds/upload/md_test.go — Verify independent batch conversion continues after one failure


## 2026-07-28

Step 7: Final implementation audit recorded. Focused tests and docmgr validation pass; full suite remains blocked by missing cmd/remarquee-ui/frontend/dist.

### Related Files

- /home/manuel/workspaces/2026-07-28/fix-remarquee-md/remarquee/ttmp/2026/07/28/RMQ-0020--make-markdown-to-pdf-conversion-resilient-to-pandoc-yaml-parser-errors/reference/01-investigation-diary.md — Final implementation evidence and handoff instructions


## 2026-07-28

Step 8: Verified the original 6,275-line user document end to end with go run upload md --pdf-only; generated a 394K PDF and re-uploaded the final bundle to /ai/2026/07/28/RMQ-0020.

### Related Files

- /home/manuel/workspaces/2026-07-28/fix-remarquee-md/remarquee/pkg/mdpdf/pandoc.go — Original-document smoke test confirms shared parser fix

