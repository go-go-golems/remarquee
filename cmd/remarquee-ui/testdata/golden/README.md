# Golden references (RMQ-0006)

This directory is intended for **committed reference artifacts** used by golden tests.

## What goes here

- **Reference PDFs** produced by a known-good pipeline (typically `remarks`) so that our Go renderer can be compared against stable baselines, even when `remarks` is not installed in CI.

## Naming convention

- `Test.rmdoc.remarks.pdf`
- `cpage-pdf.rmdoc.remarks.pdf`

In general:

- `<fixture_filename>.<reference>.pdf`
- where `<reference>` is usually `remarks`

## Updating goldens

Use the Go test flag (see `pkg/rmdoc/render/golden_remarks_test.go`):

- `go test ./pkg/rmdoc/render -run TestRenderV6Golden_RemarksReference_ -update-golden`

This will re-run the reference generator and overwrite the corresponding `*.remarks.pdf` files.

Note: We intentionally keep golden generation as an **opt-in** operation to avoid committing large binaries accidentally.


