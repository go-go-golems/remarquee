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


## 2025-12-24

Implemented remarks reference runner (Go wrapper around the  CLI): run remarks with context, capture stdout/stderr, handle missing binary via ErrNotFound, and locate generated  outputs (including nested UI-path dirs).

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/refimpl/remarks/runner.go — remarks CLI runner
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/refimpl/remarks/runner_test.go — remarks runner tests
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/tasks.md — Checked off remarks wrapper tasks


## 2025-12-24

(Follow-up) Implemented remarks reference runner (Go wrapper around the remarks CLI): run remarks with context, capture stdout/stderr, handle missing binary via ErrNotFound, and locate generated '* _remarks.pdf' outputs (including nested UI-path dirs).

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/refimpl/remarks/runner.go — remarks CLI runner
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/refimpl/remarks/runner_test.go — remarks runner tests


## 2025-12-24

Added first golden test comparing remarquee V6 render against remarks reference for cmd/remarquee-ui/testdata/Test.rmdoc. Test renders with MergeRMDocV6OntoBackgroundPDF, runs remarks via pkg/refimpl/remarks, compares via pkg/pdfcmp with tolerance, emits diff PNGs on mismatch, and skips when remarks is not on PATH.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/rmdoc/render/golden_remarks_test.go — Golden test using remarks reference
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/tasks.md — Checked off first golden test


## 2025-12-24

Added second V6 golden test (cpage-pdf.rmdoc) comparing remarquee output against remarks reference, using pdfcmp visual comparison with tolerance and diff PNGs on mismatch. Test skips when remarks is not on PATH.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/rmdoc/render/golden_remarks_test.go — Added CpagePdf golden test
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/tasks.md — Checked off CpagePdf golden test


## 2025-12-24

Added legacy golden-style smoke test for legacy-pdf-a4.zip using rmapi PdfGenerator backend. Test validates end-to-end legacy rendering and asserts output PDF page count matches parsed UI page plan.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/rmdoc/render/golden_legacy_rmapi_test.go — Legacy rmapi-backed golden smoke test
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/tasks.md — Checked off legacy test


## 2025-12-24

Added basic golden file management: created cmd/remarquee-ui/testdata/golden/ with README and naming convention, and added go test flag -update-golden so remarks-based golden tests can write/update committed reference PDFs when desired (otherwise fall back to running remarks or skip).

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee-ui/testdata/golden/README.md — Golden directory documentation
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/rmdoc/render/golden_remarks_test.go — Update-golden flag + golden reference selection
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/tasks.md — Checked off golden management tasks

