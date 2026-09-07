---
Title: Math safe preprocessing and default symbol support
Ticket: RM-MATH-PREPROCESS
Status: active
Topics:
    - markdown
    - pdf
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://README.md
      Note: External dependency and parser-scope documentation
    - Path: repo://pkg/mdpdf/literal_regions.go
      Note: Shared lexical protection for both list passes
    - Path: repo://pkg/mdpdf/literal_regions_test.go
      Note: Boundary preservation and idempotence cases
    - Path: repo://pkg/mdpdf/math_pdf_test.go
      Note: Actual direct and bundle PDF regression
    - Path: repo://pkg/mdpdf/pandoc.go
      Note: Default stmaryrd package and custom-header behavior
    - Path: repo://pkg/mdpdf/preprocess.go
      Note: Spacing and flattening skip literal lines
ExternalSources: []
Summary: Preserve display math and fenced code during list normalization and load stmaryrd in the default PDF preamble.
LastUpdated: 2026-09-07T01:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Design

## Problem

Two real ServiceLang guide renders failed. `NormalizeListSpacing` treated `+` at the start of a display-equation line as a Markdown bullet and inserted a blank line inside the equation. After correcting that document, LaTeX rejected `\llbracket` because the default header does not load `stmaryrd`, although it is installed locally.

`pkg/mdpdf/bundle.go` normalizes each input before concatenation. `pkg/mdpdf/pandoc.go` normalizes again and flattens deeply nested lists before invoking Pandoc. Both passes need the same literal-region protection. Fixing only bundle assembly would leave ordinary Markdown conversion broken.

## Changes

1. Add `\usepackage{stmaryrd}` to `defaultLatexHeader`. Document the TeX dependency. Keep `--latex-header-file` replacement semantics unchanged: callers supplying custom headers control their own packages.
2. Share a line-region scanner between the existing list passes. Preserve opening lines, contents and closing lines of backtick/tilde fenced code and display math introduced at line start by `$$` or `\[` (allow leading whitespace).
3. Track fence character and length. A closing fence must use the same character, have at least the opening length, and contain only trailing whitespace. Fence content cannot change math state, and math content cannot open a code fence.
4. Handle same-line math, escaped closing delimiters and unclosed literal regions. Preserve the rest of an unclosed region rather than rewriting malformed input. Resume ordinary list behavior immediately after closure.

This is a targeted lexical guard, not a replacement Markdown parser. Existing list depth/spacing rules outside these regions are unchanged. Arbitrary raw LaTeX environments, inline math spanning lines and full container/indented-code parsing are not added in this fix; the tests and documentation must not claim full Pandoc grammar coverage.

## Why not a new Markdown renderer?

Re-rendering a Markdown AST would risk changing source formatting, Pandoc extensions and existing image/Mermaid processing. A small shared region guard preserves literal text byte-for-byte while leaving the existing transformations in place. It has no new Go dependency and scans linearly in input size.

## Validation

- Unit regressions for dollar/bracket display math, inline delimiters, escaped closers, code fences of both characters and varying lengths, indentation, CRLF, unclosed blocks and list processing after closure.
- Assert both normalization and flattening preserve protected bodies and repeated preprocessing is stable.
- Keep existing ordinary-list tests passing.
- A real local Pandoc/XeLaTeX test exercises unmodified multiline equations plus `\llbracket`/`\rrbracket`, both directly and through bundle assembly. Skip that test when required external tools/packages are absent; pure preprocessing and header tests remain unconditional.
- Run affected package and upload-command tests. No cloud upload or device access is needed.

## Delivery

Use a dedicated worktree and feature branch, record tests and limitations, commit scoped source/docs and open a PR against origin/main. Do not alter the original checkout or re-upload existing user documents.
