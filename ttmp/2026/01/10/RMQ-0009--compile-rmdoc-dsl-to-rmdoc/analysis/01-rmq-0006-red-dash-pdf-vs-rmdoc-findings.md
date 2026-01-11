---
Title: 'RMQ-0006 red-dash PDF vs .rmdoc: findings'
Ticket: RMQ-0009
Status: active
Topics:
    - remarkable
    - rmdoc
    - rendering
    - dsl
    - compiler
    - go
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/cases/03-ellipse-sweep.js
      Note: Defines the red dash page markers used in the device review
    - Path: remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/reference/02-diary-golden-testing-research-and-rendering-issues-analysis.md
      Note: Documents the ellipse sweep PDF transport workflow
    - Path: remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/19-rmdsl-render-to-pdf/main.go
      Note: States the DSL->PDF renderer is a debug transport
    - Path: remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/20-ellipse-sweep-generate-upload-review.sh
      Note: Uploads the ellipse-sweep PDF with red dash markers to the device
    - Path: remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/tasks.md
      Note: Tracks DSL->.rmdoc compilation as a pending next step
ExternalSources: []
Summary: RMQ-0006 red-dash ellipse sweep used a DSL-to-PDF transport, not a .rmdoc compiler; evidence and implications for RMQ-0009.
LastUpdated: 2026-01-10T18:27:15.620642621-05:00
WhatFor: Answer whether the RMQ-0006 red-dash fixture was a real .rmdoc generated from JS.
WhenToUse: When clarifying prior RMQ-0006 outputs or justifying the need for a DSL-to-.rmdoc compiler.
---


# RMQ-0006 red-dash PDF vs .rmdoc: findings

## Goal

Capture evidence about the "red dashes" device document in RMQ-0006 and whether it was a real `.rmdoc` generated from JS, so RMQ-0009 can build on the right mental model.

## Question

Did RMQ-0006 generate a real `.rmdoc` from JS for the red-dash ellipse sweep, or was it a PDF transport uploaded to the device?

## Findings (short)

- The red-dash document was a PDF generated from RMDoc-DSL JS, not a `.rmdoc`.
- The JS case in `cases/03-ellipse-sweep.js` produces a DSL doc; `scripts/19-rmdsl-render-to-pdf/main.go` renders it to PDF; `scripts/20-ellipse-sweep-generate-upload-review.sh` uploads that PDF to the device.
- The PDF renderer explicitly states it is a "debug transport" and not a notebook compiler.
- RMQ-0006 task tracking lists "Compile RMDoc-DSL → real `.rmdoc`" as a next-step TODO, confirming the compiler did not exist at the time.

## Evidence

- `remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/20-ellipse-sweep-generate-upload-review.sh`
  - Runs `scripts/19-rmdsl-render-to-pdf/main.go` and then `remarquee cloud put` to upload a PDF to `/remarquee/rendering/rmq-0006-ellipse`.
- `remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/19-rmdsl-render-to-pdf/main.go`
  - Header comment: "Render RMDoc-DSL ... into a multi-page PDF ... This is a pragmatic 'debug transport' (PDF), not a `.rmdoc` notebook compiler."
- `remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/cases/03-ellipse-sweep.js`
  - Defines the red dash page markers and is invoked by the PDF renderer (JS -> DSL -> PDF).
- `remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/tasks.md`
  - "Compile RMDoc-DSL → real `.rmdoc`" appears under "Next (recommended)" (unchecked).
- `remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/reference/02-diary-golden-testing-research-and-rendering-issues-analysis.md`
  - Describes the "Ellipse sweep" workflow and explicitly calls it a PDF transport for device review.

## Implications for RMQ-0009

- RMQ-0009 fills the missing capability: compile the same RMDoc-DSL (YAML/JS) into a real `.rmdoc` so device truth can be based on notebook bytes, not PDF transports.
- The existing JS cases and PDF renderer are still useful for quick visual checks, but not for editable notebook workflows or end-to-end `.rmdoc` validation.

## References (files)

- `remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/20-ellipse-sweep-generate-upload-review.sh`
- `remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/19-rmdsl-render-to-pdf/main.go`
- `remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/cases/03-ellipse-sweep.js`
- `remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/tasks.md`
- `remarquee/ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/reference/02-diary-golden-testing-research-and-rendering-issues-analysis.md`
