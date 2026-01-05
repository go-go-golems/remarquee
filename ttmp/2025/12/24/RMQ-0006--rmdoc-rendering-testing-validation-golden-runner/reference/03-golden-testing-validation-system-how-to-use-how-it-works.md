---
Title: Golden testing + validation system (how to use + how it works)
Ticket: RMQ-0006
Status: active
Topics:
    - go
    - remarkable
    - testing
    - validation
    - rmdoc
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: bug-report-vlm-validate-unidoc-render-type-check-error.md
      Note: Bug report explaining UniDoc rasterizer failure
    - Path: cmd/remarquee-ui/testdata/golden/README.md
      Note: Golden reference directory conventions
    - Path: cmd/remarquee/cmds/rmdoc/vlm_validate.go
      Note: VLM validation helper (rasterize via pdftoppm
    - Path: pkg/pdfcmp/pdfcmp.go
      Note: PDF visual+structural comparator used by golden tests
    - Path: pkg/refimpl/remarks/runner.go
      Note: Go wrapper that runs remarks and discovers reference PDFs
    - Path: pkg/rmdoc/render/golden_legacy_rmapi_test.go
      Note: Legacy smoke/golden test (rmapi-backed)
    - Path: pkg/rmdoc/render/golden_remarks_test.go
      Note: V6 golden tests (remarquee vs remarks)
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-24T16:20:57.26316822-05:00
WhatFor: ""
WhenToUse: ""
---


# Golden testing + validation system (how to use + how it works)

This guide explains the **golden testing / validation system** we built in RMQ-0006. It is intentionally written for humans: it tells you what to run, what to expect, and where to look when something is not set up correctly.

The system has three layers that complement each other:

- **Deterministic golden checks**: compare remarquee output against a stable reference PDF (`remarks`), using Go tests.
- **Automated diffs for debugging**: create diff PNGs when a comparison fails.
- **Human-in-the-loop validation**: optional VLM checks via `pinocchio`, driven by `remarquee rmdoc vlm-validate`.

## Goal

Provide a reliable workflow to answer:

- “Did remarquee’s renderer regress compared to last week?”
- “How exactly does our output differ from `remarks` (reference implementation)?”
- “Is this failure due to setup (missing tools) or a real rendering change?”

In practice, we want to be able to run a command/test, get a PASS/FAIL, and if it fails, immediately have **artifacts** (PDFs, diff PNGs, logs) that explain why.

## Context

### What makes this system “brittle”

This is a multi-language, multi-tool pipeline. Brittleness comes from:

- `remarks` (Python) may not be installed on PATH, or may require Python 3.12 + git deps.
- Some PDF libraries are strict; rendering PDFs to images can fail on edge-case PDF constructs.
- The comparison step is sensitive to page size, rotations, and subtle rendering differences.

We addressed the biggest brittleness points:

- A Go wrapper to run `remarks` (`pkg/refimpl/remarks`)
- A Go PDF comparator (`pkg/pdfcmp`)
- A VLM helper that rasterizes via Poppler (`pdftoppm`) rather than UniDoc (avoids UniDoc “type check error”)
- A clear “golden directory” for committed references: `cmd/remarquee-ui/testdata/golden/`

### Directory map (where to look)

- **Fixtures**: `cmd/remarquee-ui/testdata/`
  - `Test.rmdoc`
  - `cpage-pdf.rmdoc`
  - `legacy-pdf-a4.zip`
- **Committed goldens (optional)**: `cmd/remarquee-ui/testdata/golden/`
- **Golden tests**: `pkg/rmdoc/render/`
  - `golden_remarks_test.go` (V6: compares against remarks)
  - `golden_legacy_rmapi_test.go` (legacy: rmapi-backed smoke test)
- **Reference runner**: `pkg/refimpl/remarks/`
- **Comparator**: `pkg/pdfcmp/`
- **VLM helper**: `cmd/remarquee/cmds/rmdoc/vlm_validate.go`
- **Bug report (UniDoc rasterizer)**: `bug-report-vlm-validate-unidoc-render-type-check-error.md`

## Quick Reference

This section is copy/paste oriented. If you only remember one thing: start with the V6 golden tests and look at the diff PNGs when it fails.

### 1) Install `remarks` (Poetry + pyenv)

This repo includes `remarks/` as a sibling project. `remarks` requires Python 3.12.

```bash
cd /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarks && \
PYENV_VERSION=3.12.3 poetry env use "$(pyenv which python3.12)" --no-interaction && \
poetry install --no-interaction
```

Make it visible on PATH for golden tests and tools:

```bash
export PATH="/home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarks/.venv/bin:$PATH"
command -v remarks
```

### 2) Run V6 golden tests (remarks reference)

From `remarquee/`:

```bash
go test ./pkg/rmdoc/render -run TestRenderV6Golden_RemarksReference_ -count=1
```

What this does:

- renders with remarquee (Go)
- generates reference PDF with `remarks` (Python), unless a committed golden exists
- compares output visually with `pkg/pdfcmp`
- writes diff PNGs to `t.TempDir()` on mismatch

### 3) Update committed goldens (opt-in)

If you want to store reference PDFs under `cmd/remarquee-ui/testdata/golden/`:

```bash
go test ./pkg/rmdoc/render -run TestRenderV6Golden_RemarksReference_ -update-golden -count=1
```

This will write `*.remarks.pdf` files into the golden directory (overwriting existing files).

If your goal is **only** to regenerate committed `remarks` reference PDFs (without depending on remarquee-vs-remarks comparisons passing), use:

```bash
go test ./pkg/rmdoc/render -run TestUpdateRemarksGoldens -update-golden -count=1
```

### 4) Run the VLM helper (pinocchio)

The `vlm-validate` command renders pages to PNG using **Poppler** (`pdftoppm`) and then calls `pinocchio` with the PNG list.

Example:

```bash
go run ./cmd/remarquee rmdoc vlm-validate \
  --rasterizer poppler --dpi 200 \
  --pdf-b /tmp/reference.pdf \
  --pages 1,2 \
  --prompt "Compare A vs B. Focus on strokes/highlights/typed text/page size/template." \
  /tmp/remarquee.pdf
```

### 5) If something fails: triage checklist

- **“remarks not found”**:
  - `export PATH=".../remarks/.venv/bin:$PATH"`
  - `command -v remarks`
- **No reference PDF produced**:
  - check `remarks <input> <outputDir> --log_level ERROR` output
  - verify the output includes a `* _remarks.pdf` file (note the space)
- **VLM helper fails**:
  - verify `pdftoppm` exists: `command -v pdftoppm`
  - verify `pinocchio` exists: `command -v pinocchio`
- **Golden comparison fails**:
  - inspect diff PNGs (printed as `diff image: ...` in test logs)
  - inspect both PDFs (remarquee vs reference)

## Usage Examples

This section explains workflows with narrative context (when and why to use which tool).

### Workflow A: “I changed rendering code, did I break anything?”

When you touch `pkg/rmdoc/render/*` (merge math, strokes, highlights), run the V6 golden tests first. Even if they fail due to known missing features (color, typed text), they will still tell you if you made things worse.

```bash
go test ./pkg/rmdoc/render -run TestRenderV6Golden_RemarksReference_ -count=1
```

If a test fails:

- find the diff PNGs path in the test output
- open the temp directory and compare A vs B page renderings
- decide if the diff is expected (known gaps) or a regression

### Workflow B: “I want a human-readable explanation of differences”

Golden diffs are objective but not always easy to interpret. VLM validation is useful when:

- highlights look misaligned and you want a second opinion
- page size/cropping issues are subtle
- you want a text report that you can paste into an issue/ticket

Generate two PDFs (A=remarquee, B=remarks) and run:

```bash
go run ./cmd/remarquee rmdoc vlm-validate \
  --rasterizer poppler --dpi 200 \
  --pdf-b /tmp/B.pdf \
  --pages 1,2 \
  /tmp/A.pdf
```

The output includes:

- a directory containing `A-page-001.png`, `B-page-001.png`, etc.
- pinocchio’s natural-language comparison

### Workflow C: “Legacy rendering sanity”

Legacy rendering (V3/V5) is not compared against `remarks`. Instead we run a smoke test that ensures the rmapi-backed renderer can generate a PDF with the expected page count:

```bash
go test ./pkg/rmdoc/render -run TestRenderLegacyGolden_Rmapi_Backend_LegacyPdfA4 -count=1
```

## How it works (implementation guide)

This section is for understanding the code and debugging brittleness.

### 1) `remarks` reference runner (`pkg/refimpl/remarks`)

We treat `remarks` as an external reference implementation. The runner:

- shells out to `remarks INPUT OUTPUT_DIR [--log_level ...]`
- captures stdout/stderr
- returns a sentinel `ErrNotFound` if `remarks` is not on PATH
- finds produced `* _remarks.pdf` files recursively in the output dir

Key files:
- `pkg/refimpl/remarks/runner.go`

Why this matters:
- it lets Go tests generate a reference PDF without embedding Python logic into Go
- it centralizes “where did `remarks` write the output?” logic (nested dirs, naming quirks)

### 2) PDF comparison (`pkg/pdfcmp`)

The comparator provides two approaches:

- **Visual**: render pages to images with UniDoc and count pixel differences with tolerance; emit diff PNG
- **Structural**: compare page counts, annotation counts, extracted text hashes

Key files:
- `pkg/pdfcmp/pdfcmp.go`
- `pkg/pdfcmp/pdfcmp_test.go`

Why this matters:
- it turns “PDFs differ” into actionable “page 2 differs by 3.2% pixels, here’s a diff image”

### 3) Golden tests (`pkg/rmdoc/render/*`)

#### V6 tests (remarks reference)

`golden_remarks_test.go` implements:

- render fixture to PDF using `MergeRMDocV6OntoBackgroundPDF`
- acquire reference PDF:
  - prefer committed golden in `cmd/remarquee-ui/testdata/golden/`
  - otherwise run `remarks`
  - otherwise skip (no golden and no `remarks`)
- run visual compare (`pdfcmp.CompareFilesVisual`)
- write diff PNGs on mismatch

#### Legacy test (rmapi-backed)

`golden_legacy_rmapi_test.go`:

- uses `rmapi/annotations.PdfGenerator` to generate output
- asserts PDF page count matches parsed page plan

### 4) VLM helper (`remarquee rmdoc vlm-validate`)

The VLM helper exists because the “describe these images” loop is extremely effective for debugging rendering issues, but you don’t want to manually run 4 different commands every time.

Implementation details:

- It rasterizes PDF pages to PNGs.
- It then calls:
  - `pinocchio code professional --images <comma-separated pngs> "<prompt>"`

Important implementation choice:

- We default to **Poppler `pdftoppm`** rasterization.
- We previously tried UniDoc rasterization; it can fail with **“type check error”**.
  - See `bug-report-vlm-validate-unidoc-render-type-check-error.md`

## Troubleshooting (common failure modes)

### Problem: `remarks` is installed but tests still say “not found”

Symptoms:
- `ErrNotFound`
- `command not found: remarks`

Fix:

- ensure your PATH includes the Poetry venv:
  - `export PATH="/.../remarks/.venv/bin:$PATH"`
- verify:
  - `command -v remarks`

### Problem: `remarks` runs, but no `* _remarks.pdf` is found

This usually means you are looking in the wrong output directory, or `remarks` errored before saving.

Fix:

- run manually with verbose logging:
  - `remarks <input> <output> --log_level DEBUG`
- look for files with the exact suffix:
  - `find <output> -name "* _remarks.pdf"`

### Problem: VLM helper fails

Checklist:

- `command -v pdftoppm`
- `command -v pinocchio`
- ensure `--rasterizer poppler` (default)

### Problem: Golden test fails “all the time”

This can be real (missing features) or setup (wrong reference PDF).

Triage:

- Confirm which reference is used:
  - committed golden: `cmd/remarquee-ui/testdata/golden/*.remarks.pdf`
  - or generated via `remarks`
- Inspect the diff PNG output from the test logs

## Related

- `reference/01-testing-and-validation-playbook.md` (manual stage loop)
- `analysis/01-using-remarks-for-golden-testing-and-pdf-comparison.md` (research + deeper code analysis)
- `analysis/02-running-remarks-locally-in-ci-nix-vs-poetry-vs-pyenv-decision-doc.md` (toolchain decision)
- `cmd/remarquee-ui/testdata/golden/README.md` (golden directory conventions)
