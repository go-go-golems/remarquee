# Changelog

## 2026-05-15

- Initial workspace created


## 2026-05-15

Made sync, md, and conversion_workers resilient to per-file errors. Pandoc/upload/delete errors now skip the failing file and continue, printing ERROR-CONVERT/ERROR-UPLOAD/ERROR-DELETE and a summary line. Added syncActionError. (commit c30268c)

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/remarquee/cmd/remarquee/cmds/upload/conversion_workers.go — convertMarkdownJobs now collects errors per file instead of canceling all workers
- /home/manuel/code/wesen/corporate-headquarters/remarquee/cmd/remarquee/cmds/upload/md.go — runUploadMarkdown upload loop now collects errors instead of aborting
- /home/manuel/code/wesen/corporate-headquarters/remarquee/cmd/remarquee/cmds/upload/sync.go — executeSyncPlan now collects errors instead of aborting
- /home/manuel/code/wesen/corporate-headquarters/remarquee/cmd/remarquee/cmds/upload/sync_plan.go — Added syncActionError constant


## 2026-05-15

Fixed multi-worker upload path to continue uploading successful PDFs after conversion errors (commit 2350d2a). Fixed pdf-only mode to return non-zero exit code on failure (commit e7d1acd).

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/remarquee/cmd/remarquee/cmds/upload/md.go — Multi-worker and pdf-only paths now handle partial success correctly

