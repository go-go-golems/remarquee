# Changelog

## 2025-12-24

- Initial workspace created


## 2025-12-24

Created analysis document on using remarks for golden testing and PDF comparison. Document covers invocation methods, comparison strategies (visual/structural/hybrid), integration approaches, and best practices. Added comprehensive task list for implementing golden testing infrastructure.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/analysis/01-using-remarks-for-golden-testing-and-pdf-comparison.md — Research document on golden testing approach


## 2025-12-24

Added VLM validation section to analysis document. Added investigation tasks for 5 rendering issues discovered through visual inspection. Performed deep code analysis tracing through codebase to identify root causes. Created diary entry documenting research and analysis work.


## 2025-12-24

Implemented Option B PDF comparison utilities (pure Go): UniDoc-based visual comparison (render pages to images + pixel diff with tolerance + diff PNG) and fast structural comparison (page count, annotation count, extracted text hash). Added unit tests generating small PDFs.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/pdfcmp/pdfcmp.go — New pdfcmp package
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/pdfcmp/pdfcmp_test.go — pdfcmp tests
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/tasks.md — Checked off Option B comparison tasks

