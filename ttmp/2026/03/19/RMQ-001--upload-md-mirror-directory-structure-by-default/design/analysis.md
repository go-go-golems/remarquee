---
Title: Implementation Analysis - Mirror directory structure by default
DocType: design
---

# Implementation Analysis

## Problem

`remarquee upload md foo --remote-dir /bla` with `foo/a/b.md` currently uploads to
`/bla/b.pdf` (flat). The user expects `/bla/a/b.pdf` — mirroring the local directory
structure.

The `--preserve-dirs` flag already implements this behavior but defaults to `false`.

## Current Flow

1. `collectMarkdownInputs()` walks directories and computes `RelPath` for each file
   (relative to the input directory root).
2. `markdownInput.RelDir()` extracts the directory portion of `RelPath`.
3. In the upload/dry-run/pdf-only loops, `RelDir()` is only used when `s.PreserveDirs`
   is true.
4. For individual file inputs, `RelPath` is just the basename, so `RelDir()` returns `""`.

## Proposed Change

**Make `--preserve-dirs` default to `true`.**

This is safe because:
- Individual file inputs have `RelDir() == ""`, so their behavior is unchanged.
- Only directory inputs are affected — they will now mirror structure by default.
- Users who want flat behavior can pass `--preserve-dirs=false`.

Additionally, add a `--flatten` convenience flag as the inverse of `--preserve-dirs`
for ergonomics.

## Affected Code Paths (all in `md.go`)

| Location | What changes |
|---|---|
| Flag definition (line 76) | Default `false` → `true`, update help text |
| New `--flatten` flag | Adds inverse convenience flag |
| `RunE` preamble | Resolve `--flatten` → `PreserveDirs = false` |
| Collision detection (lines 110-122) | Already uses `s.PreserveDirs` — works as-is |
| Dry-run loop (lines 132-156) | Already uses `s.PreserveDirs` — works as-is |
| PDF-only loop (lines 160-185) | Already uses `s.PreserveDirs` — works as-is |
| Upload loop (lines 204-261) | Already uses `s.PreserveDirs` — works as-is |

## Test Impact

- Existing tests don't test `PreserveDirs` behavior directly (only `collectMarkdownFiles`,
  `resolveRemoteDir`, `joinRemoteDir`, `remoteDocKey`).
- Need new tests for `collectMarkdownInputs` with directory structure validation.
- Need test for `--flatten` flag interaction.

## Scope

Small, focused change: ~15 lines of code + tests + help text updates.
