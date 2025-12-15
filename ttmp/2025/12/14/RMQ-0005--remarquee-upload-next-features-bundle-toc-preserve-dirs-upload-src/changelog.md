# Changelog

## 2025-12-14

- Initial workspace created


## 2025-12-14

Initialized RMQ-0005: moved next-features design-doc from RMQ-0003, seeded tasks, and added intern guide playbook.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/14/RMQ-0005--remarquee-upload-next-features-bundle-toc-preserve-dirs-upload-src/design-doc/01-upload-next-features-bundle-pdfs-w-toc-mirror-dirs-syntax-highlight-source.md — Design for bundle/ToC
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/14/RMQ-0005--remarquee-upload-next-features-bundle-toc-preserve-dirs-upload-src/playbook/01-intern-guide-implementing-upload-next-features-bundle-toc-preserve-dirs-upload-src.md — Intern onboarding guide


## 2025-12-15

Phase 1 (bundle): added `remarquee upload bundle` to collate multiple markdown inputs into one PDF with a clickable ToC, then upload as a single document (commit c3167537...).

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee/cmds/upload/bundle.go — New `upload bundle` command
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/mdpdf/bundle.go — Wrapper markdown generation for stable ToC entries
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/mdpdf/pandoc.go — Added ToC/highlight options plumbing


## 2025-12-15

Phase 2 (mirror dirs): extended `remarquee upload md` with `--preserve-dirs` to recreate local relative subfolders under the chosen remote base (commit 6d883d6c...).

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee/cmds/upload/md.go — Added preserve-dirs mode and per-file remote dir creation

