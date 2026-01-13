---
Title: Compile RMDoc-DSL → .rmdoc (editable notebooks)
Ticket: RMQ-0009
Status: active
Topics:
    - remarkable
    - rmdoc
    - rendering
    - dsl
    - compiler
    - go
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/analysis/01-rmq-0006-red-dash-pdf-vs-rmdoc-findings.md
      Note: Background analysis for RMQ-0009
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/analysis/02-incremental-rmdoc-compiler-test-plan.md
      Note: Incremental test ladder for compiler validation
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/analysis/03-rmv6-block-semantics-in-rmscene-go-parser.md
      Note: Block semantics analysis
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/analysis/04-remarks-rmscene-block-handling-and-semantics.md
      Note: Remarks + rmscene parsing and block semantics reference
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/reference/02-diary.md
      Note: Ongoing implementation diary
ExternalSources: []
Summary: |
    Implement a compiler that converts RMDoc-DSL (YAML + JS/goja) into real .rmdoc archives (V6 cPages notebooks) so that generated fixtures can be uploaded to the device as editable notebooks and used as ground-truth references.
LastUpdated: 2026-01-10T21:35:05-05:00
WhatFor: ""
WhenToUse: ""
---




# Compile RMDoc-DSL → .rmdoc (editable notebooks)

## Overview

RMQ-0006 proved that RMDoc-DSL is an excellent tool for *reproducible* fixture generation and debugging, but we still rely on PDFs as “transport” for device review. PDFs are good for viewing, but they are not the same as an editable notebook on-device.

This ticket implements the missing bridge:

- RMDoc-DSL (declarative intent) → **real `.rmdoc` archive** (device-native representation)

Once this exists, we can:

- generate fixtures programmatically,
- upload them to the tablet as editable notebooks,
- capture device screenshots that correspond to exactly the same bytes,
- and use those as stable, repeatable truth for renderer validation.

## Tasks

See [tasks.md](./tasks.md).

## Reference / intern guide

See [reference/01-intern-guide.md](./reference/01-intern-guide.md).

## Analysis

- [analysis/01-rmq-0006-red-dash-pdf-vs-rmdoc-findings.md](./analysis/01-rmq-0006-red-dash-pdf-vs-rmdoc-findings.md)
- [analysis/02-incremental-rmdoc-compiler-test-plan.md](./analysis/02-incremental-rmdoc-compiler-test-plan.md)
- [analysis/03-rmv6-block-semantics-in-rmscene-go-parser.md](./analysis/03-rmv6-block-semantics-in-rmscene-go-parser.md)
- [analysis/04-remarks-rmscene-block-handling-and-semantics.md](./analysis/04-remarks-rmscene-block-handling-and-semantics.md)

## Diary

See [reference/02-diary.md](./reference/02-diary.md).
