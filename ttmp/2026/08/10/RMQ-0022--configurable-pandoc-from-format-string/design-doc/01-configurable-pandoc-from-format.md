---
Title: Configurable pandoc from-format
Ticket: RMQ-0022
Status: active
Topics:
    - backend
    - upload
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/mdpdf/pandoc.go
      Note: Where the hardcoded --from lived
ExternalSources: []
Summary: Make the hardcoded pandoc --from format string configurable via a --pandoc-from flag on upload commands.
LastUpdated: 2026-08-10T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Configurable pandoc from-format

## Problem

`pkg/mdpdf/pandoc.go` (`buildPandocArgs`) hardcodes the pandoc input format:

```go
"--from=markdown-yaml_metadata_block",
```

Pandoc Markdown extensions (e.g. `+tex_math_single_backslash` for ChatGPT-style
`\(...\)` math) can only be enabled inside this format string. Because pandoc's
rule is "last `-f`/`--from` wins", remarquee's hardcoded `--from` silently
overrides any custom pandoc wrapper script passed via `--pandoc` that tries to
set its own `-f`. Observed 2026-08-10: the `pandoc-math` wrapper from the
chatgpt-transcript-archiving skill was silently ignored, causing
`! Missing $ inserted` failures on math-heavy transcripts.

## Decision

Add `FromFormat string` to `mdpdf.PandocOptions` and a `--pandoc-from` CLI flag
on the commands that convert markdown: `upload md`, `upload bundle`,
`upload sync`, and `upload src`.

- Default: `markdown-yaml_metadata_block` (current behavior, unchanged).
- Empty `FromFormat` in `PandocOptions` falls back to the default constant, so
  programmatic callers are unaffected.
- Full format string (not an extension-append flag) for generality and simple
  implementation.

### Alternatives considered

1. `--pandoc-ext +foo,-bar` appended to the default: narrower, needs parsing
   and validation; the full-string flag subsumes it.
2. Do nothing now that surf-cli normalizes math at the source (commit f0dcd64
   in surf-cli): rejected — the silent wrapper override is a general footgun,
   not just a math problem.

## Implementation

1. `pkg/mdpdf/pandoc.go`:
   - `const DefaultFromFormat = "markdown-yaml_metadata_block"`
   - `PandocOptions.FromFormat string`
   - `DefaultPandocOptions()` sets `FromFormat: DefaultFromFormat`
   - `buildPandocArgs` uses `opts.FromFormat`, falling back to
     `DefaultFromFormat` when empty: `"--from=" + fromFormat`
2. `cmd/remarquee/cmds/upload/layout.go`: add `fromFormat string` param to
   `configureMarkdownPandocOptions`; set `opts.FromFormat` when the flag was
   changed (or non-empty).
3. Add `--pandoc-from` flag + settings field in `md.go`, `bundle.go`,
   `sync.go`; `src.go` sets `pandocOpts.FromFormat` directly (it does not use
   the shared helper).

## Testing

- Unit test: `buildPandocArgs` (or a Convert call with a fake pandoc) includes
  the custom `--from` when set and the default otherwise.
- Manual: `remarquee upload md <math transcript> --pandoc-from
  "markdown-yaml_metadata_block+tex_math_single_backslash" --dry-run` shows the
  flag accepted; real conversion of a raw `\(` transcript succeeds without the
  surf-cli-side normalization.
