# Changelog

## 2026-01-10

- Ticket created (split from RMQ-0006/RMQ-0008) to implement “DSL → real `.rmdoc`” compilation and editable device uploads.



## 2026-01-10

Step 1: trace RMQ-0006 red-dash PDF workflow and record analysis for RMQ-0009 scope

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/analysis/01-rmq-0006-red-dash-pdf-vs-rmdoc-findings.md — Stores the evidence and conclusion
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/reference/02-diary.md — Records RMQ-0009 Step 1 research


## 2026-01-10

Index: link analysis + diary docs

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/index.md — Added pointers to new analysis and diary


## 2026-01-10

Step 2: add strokes-only DSL -> .rmdoc compiler, CLI, tests, and scripts

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee/cmds/rmdsl/compile.go — CLI compile command
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/rmdsl/compile/compile.go — Compiler entry + archive assembly
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/rmdsl/compile/compile_test.go — Integration tests


## 2026-01-10

Step 3: upload compiled .rmdoc + device confirmation attempt (timeout)

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/reference/02-diary.md — Recorded upload/confirm attempt and timeout


## 2026-01-10

Add incremental compiler test plan and link it from the index

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/analysis/02-incremental-rmdoc-compiler-test-plan.md — New testing ladder doc
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/index.md — Added analysis link


## 2026-01-10

Step 4: write incremental compiler test ladder

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/analysis/02-incremental-rmdoc-compiler-test-plan.md — Test ladder doc
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/reference/02-diary.md — Recorded Step 4


## 2026-01-10

Step 5: create core 8 DSL cases + upload batch

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/cases/01-empty-1page.yaml — Core validation cases
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/reference/02-diary.md — Recorded Step 5
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/scripts/03-batch-compile-upload-tests.sh — Batch compile/upload script


## 2026-01-10

Step 6: compare device empty vs compiled empty .rm blocks

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/reference/02-diary.md — Recorded Step 6 findings
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/scripts/04-dump-rm-blocks/main.go — Block dump script


## 2026-01-10

Step 7: document RMV6 block semantics in rmscene + Go parser

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/analysis/03-rmv6-block-semantics-in-rmscene-go-parser.md — New analysis doc
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/reference/02-diary.md — Recorded Step 7


## 2026-01-10

Created analysis doc on remarks+rmscene block handling and semantics, capturing block roles and parse flow.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/analysis/04-remarks-rmscene-block-handling-and-semantics.md — Documented block roles


## 2026-01-10

Added complex DSL cases and batch script to compile, render-v6 PDFs, and upload paired artifacts for device comparison.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/cases/09-complex-spiral-grid.yaml — New complicated validation case
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/scripts/05-batch-compile-render-upload-complicated.sh — Batch compile/render/upload script


## 2026-01-10

Renamed PDF outputs with -pdf suffix to avoid basename collisions; reuploaded paired PDFs alongside notebooks.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/scripts/05-batch-compile-render-upload-complicated.sh — Use -pdf suffix to keep PDF uploads distinct

