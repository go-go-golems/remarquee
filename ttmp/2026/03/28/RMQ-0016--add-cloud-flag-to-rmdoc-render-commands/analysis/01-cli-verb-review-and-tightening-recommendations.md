---
Title: CLI verb review and tightening recommendations
Ticket: RMQ-0016
Status: active
Topics:
    - cli
    - cloud
    - rendering
    - rmdoc
    - remarkable
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/remarquee/cmds/cloud/find.go
      Note: |-
        Overlaps with `cloud search`
        Overlap with search
    - Path: cmd/remarquee/cmds/cloud/ls.go
      Note: Stable filetree browsing surface
    - Path: cmd/remarquee/cmds/cloud/root.go
      Note: Current cloud subtree breadth
    - Path: cmd/remarquee/cmds/cloud/search.go
      Note: |-
        Richer search surface that likely supersedes `find`
        Richer search surface
    - Path: cmd/remarquee/cmds/cloud/stat.go
      Note: Stable metadata lookup surface
    - Path: cmd/remarquee/cmds/rmdoc/build_background.go
      Note: |-
        Explicitly labeled debug utility
        Explicit debug utility
    - Path: cmd/remarquee/cmds/rmdoc/render_v6_png.go
      Note: |-
        Wrapper around PDF render plus Poppler rasterization
        PNG wrapper around PDF render plus Poppler
    - Path: cmd/remarquee/cmds/rmdoc/root.go
      Note: |-
        Current `rmdoc` verb sprawl
        Rmdoc subtree breadth
    - Path: cmd/remarquee/cmds/rmdoc/vlm_validate.go
      Note: |-
        Optional manual validation helper, likely not a stable product verb
        Manual validation helper
    - Path: cmd/remarquee/main.go
      Note: Root command taxonomy
    - Path: ttmp/2026/01/17/MO-001-CLEANUP-CLI--consolidate-and-improve-the-remarquee-cli/analysis/01-remarquee-cli-inventory-and-analysis.md
      Note: Previous CLI inventory and context
ExternalSources: []
Summary: 'The CLI is functional but too wide in a few places: `cloud find` overlaps with `cloud search`, `rmdoc` mixes stable render verbs with internal/debug tooling, and auth/output conventions are inconsistent across subtrees.'
LastUpdated: 2026-03-28T10:35:12.571906693-04:00
WhatFor: Evidence-backed review of which verbs feel stable, which feel experimental, and where the command surface should be tightened over time.
WhenToUse: Use when planning CLI cleanup, deprecations, aliases, or a future `rmdoc render` consolidation.
---


# CLI verb review and tightening recommendations

## Executive Summary

The current remarquee CLI is coherent enough to use, but it still feels like a workbench rather than a trimmed product surface. That is not a criticism of the implementation quality. It is a reflection of the project history: the CLI accumulated validation tools, migration shims, and internal helpers while multiple subsystems were being explored in parallel.

A prior inventory ticket counted seven top-level verbs and thirty-one leaf verbs, with `cloud` at twelve leaves and `rmdoc` at six (`ttmp/2026/01/17/MO-001-CLEANUP-CLI--consolidate-and-improve-the-remarquee-cli/analysis/01-remarquee-cli-inventory-and-analysis.md:45-69`). That count still feels directionally right today. The main tightening opportunities are:

1. reduce overlap between `cloud find` and `cloud search`,
2. separate stable `rmdoc` user commands from debug/validation tooling,
3. converge on a smaller render surface long-term,
4. centralize repeated cloud-auth behavior and structured-output conventions.

## Method

This review is intentionally practical. I did not start from a naming theory. I started from the code and asked:

- Which verbs look like stable user tasks?
- Which verbs look like internal debugging aids that escaped into the public tree?
- Which verbs overlap enough that a new user would not know which one to pick?
- Which conventions are repeated often enough to deserve shared infrastructure?

## Current Taxonomy Snapshot

### Top-level verbs

The root command registers:

- `status`
- `cloud`
- `device`
- `ocr`
- `rmdsl`
- `rmdoc`
- `upload`

Evidence: `cmd/remarquee/main.go:17-37`.

### `cloud` subtree

The `cloud` subtree currently exposes:

- `account`
- `get`
- `find`
- `search`
- `ls`
- `mv`
- `mkdir`
- `rm`
- `put`
- `refresh`
- `stat`
- `version`

Evidence: `cmd/remarquee/cmds/cloud/root.go:7-170`.

### `rmdoc` subtree

The `rmdoc` subtree currently exposes:

- `inspect`
- `build-background`
- `render-legacy`
- `render-v6`
- `render-v6-png`
- `vlm-validate`

Evidence: `cmd/remarquee/cmds/rmdoc/root.go:5-88`.

## What Feels Stable Versus Experimental

### Stable-feeling verbs

These read like durable end-user tasks:

- `cloud ls`
- `cloud stat`
- `cloud get`
- `cloud put`
- `cloud rm`
- `cloud mv`
- `upload md`
- `upload bundle`
- `upload src`
- `rmdoc inspect`
- `rmdoc render-legacy`
- `rmdoc render-v6`

Why they feel stable:

- They correspond to obvious user outcomes.
- Their names match the user's intent.
- Their help text reads like product behavior, not debugging guidance.

### Experimental or workbench-feeling verbs

These read like implementation-support tools:

- `rmdoc build-background`
- `rmdoc render-v6-png`
- `rmdoc vlm-validate`
- arguably `status`

Evidence:

- `build-background` literally calls itself a "debug utility" in its short description (`cmd/remarquee/cmds/rmdoc/build_background.go:42-50`).
- `render-v6-png` is described as "generate a merged PDF and rasterize selected pages with Poppler's pdftoppm" (`cmd/remarquee/cmds/rmdoc/render_v6_png.go:52-62`). That is a pipeline convenience wrapper, not a core concept.
- `vlm-validate` explicitly says it is an "optional validation helper (manual/interactive workflow)" (`cmd/remarquee/cmds/rmdoc/vlm_validate.go:61-75`).
- `status` is intentionally just a wiring smoke test that prints `remarquee: ok` (`cmd/remarquee/cmds/status.go:9-20`).

## Review Findings

## 1. `cloud find` and `cloud search` overlap too much

This is the clearest verb-level duplication in the current tree.

`cloud find`:

- recursive walk
- optional regex
- prints formatted paths
- compact mode available

Evidence: `cmd/remarquee/cmds/cloud/find.go:44-139`.

`cloud search`:

- recursive walk
- substring or regex
- `path` vs `name`
- type filter
- case sensitivity
- result limit
- template inclusion toggle

Evidence: `cmd/remarquee/cmds/cloud/search.go:51-245`.

Assessment:

- `search` is strictly more expressive for most human workflows.
- `find` mostly survives as a thinner, more rmapi-like alias.
- New users will not know when to choose one over the other.

Recommendation:

- keep `cloud search` as the preferred stable verb,
- mark `cloud find` as legacy/compatibility-oriented in help text,
- later deprecate `find` or reframe it as a simple alias to `search --regex --match path`.

## 2. `rmdoc` mixes product verbs with debug utilities

The `rmdoc` subtree currently puts three different categories in one flat list:

1. core user tasks: `inspect`, `render-legacy`, `render-v6`
2. render-adjacent debug helpers: `build-background`, `render-v6-png`
3. validation-lab tooling: `vlm-validate`

That makes the surface harder to scan. A new user looking for "render a document" sees six peers, even though only two or three are stable product actions.

Recommendation:

- long-term, introduce a `rmdoc debug` or `rmdoc lab` subgroup for:
  - `build-background`
  - `render-v6-png`
  - `vlm-validate`
- keep the current commands as aliases for a migration period

A possible future shape:

```text
remarquee rmdoc inspect
remarquee rmdoc render ...
remarquee rmdoc debug build-background
remarquee rmdoc debug png
remarquee rmdoc debug vlm-validate
```

## 3. `render-legacy` and `render-v6` are understandable now, but not ideal long-term

The split exists for a good reason. RMQ-0004 explicitly discovered that the tool must support both legacy and `cPages` archives, so separate pipelines made sense during the port (`ttmp/2025/12/14/RMQ-0004--port-rmdoc-parsing-rendering-to-go-v3-v5-v6/index.md:56-66`).

Current evidence:

- `render-legacy` rejects non-legacy schemas (`cmd/remarquee/cmds/rmdoc/render_legacy.go:111-120`)
- `render-v6` rejects non-`cPages` schemas (`cmd/remarquee/cmds/rmdoc/render_v6.go:93-102`)

Assessment:

- As implementation scaffolding, this is fine.
- As a long-term product surface, it leaks internal format differences into user-facing verbs.

Recommendation:

- short term: keep both, because the behavior is explicit and safe
- long term: converge on `rmdoc render` with automatic schema detection
- preserve `render-legacy` and `render-v6` as aliases for scripted compatibility until a major cleanup pass is accepted

This is especially relevant because the new `--cloud` feature will otherwise need to be added twice, and every future input/output enhancement will also need to be added twice.

## 4. `render-v6-png` is probably a format option masquerading as a verb

The command is useful, but its implementation reveals its true role:

1. open `.rmdoc`
2. validate schema
3. render merged PDF
4. shell out to `pdftoppm`
5. write PNGs

Evidence: `cmd/remarquee/cmds/rmdoc/render_v6_png.go:114-198`.

That makes it feel less like a separate conceptual action and more like:

```text
rmdoc render --format png --pages ...
```

Recommendation:

- do not remove it immediately,
- but treat it as a candidate to fold into a unified render surface later,
- especially if `render-v6` and `render-legacy` eventually collapse into `rmdoc render`.

## 5. `vlm-validate` should be treated as an advanced/internal tool

The command is valuable for renderer development. It is also highly specific:

- requires `pinocchio`
- cares about Poppler vs UniDoc rasterization
- accepts a bespoke mix of PDF, PNG, and `.rmdoc` inputs
- exists to support manual comparison workflows

Evidence: `cmd/remarquee/cmds/rmdoc/vlm_validate.go:61-75`, `cmd/remarquee/cmds/rmdoc/vlm_validate.go:209-220`.

Recommendation:

- keep it available,
- but document it as internal/advanced,
- and strongly consider moving it under `rmdoc debug` in a cleanup pass.

## 6. Auth-flag duplication is a CLI smell

The same `--non-interactive` and `--reauth` flags are repeated across:

- `cloud/*`
- `upload/*`
- future `rmdoc --cloud` work

This repetition shows up directly in the code search results across many command files.

Assessment:

- the flags themselves are reasonable,
- the duplication suggests missing infrastructure.

Recommendation:

- standardize on a shared cloud-auth settings/helper layer,
- ideally one that handles both struct definition and flag registration,
- so future commands do not have to hand-copy the same pair again.

## 7. Output conventions are inconsistent across the CLI

The CLI currently mixes:

- pure Cobra text commands (`status`, `upload/*`, `device/*`)
- glazed parameter commands with only human output
- glazed dual-mode commands with `--with-glaze-output`
- one writer-style LLM command (`ocr`)

This is already documented in the earlier inventory (`MO-001`, lines 58-62), and it remains true.

Assessment:

- the inconsistency is understandable historically,
- but it increases surprise for users and maintenance cost for implementers.

Recommendation:

- define a policy for which subtrees are meant to support structured output,
- especially for `cloud` and `rmdoc`,
- and let debug-only commands be explicit exceptions if needed.

## Suggested Command Tiers

### Tier 1: Stable user-facing verbs

- `cloud ls`
- `cloud stat`
- `cloud get`
- `cloud put`
- `cloud mv`
- `cloud rm`
- `cloud mkdir`
- `cloud account`
- `upload md`
- `upload bundle`
- `upload src`
- `rmdoc inspect`
- `rmdoc render-legacy`
- `rmdoc render-v6`

### Tier 2: Transitional or compatibility verbs

- `cloud find`
- `status`

### Tier 3: Debug/lab verbs

- `rmdoc build-background`
- `rmdoc render-v6-png`
- `rmdoc vlm-validate`

## Concrete Tightening Plan

### Phase A: Documentation-level tightening

No breaking changes yet.

1. Update help text to mark `find` as simpler/legacy compared with `search`.
2. Mark `build-background`, `render-v6-png`, and `vlm-validate` as debug/advanced tools.
3. Add one short "which command should I use?" guide to help or docs.

### Phase B: Alias-based cleanup

1. Add a grouped debug subtree.
2. Keep old debug verbs as hidden or documented aliases.
3. Introduce `rmdoc render` as the preferred long-term surface.

### Phase C: Deprecation cleanup

Only after docs and scripts have migrated:

1. deprecate `cloud find` in favor of `cloud search`
2. consider hiding old debug aliases from the primary help tree
3. keep format-specific render verbs only if real workflow differences remain

## What I Would Not Change Yet

I would not:

- remove `render-legacy` or `render-v6` immediately,
- remove `cloud find` abruptly,
- force every command into glazed right now,
- collapse `upload` into `cloud put` even though both move content to the device.

Those would create more churn than value in the short term.

## Practical Recommendation For This Ticket

For RMQ-0016 specifically:

1. Add `--cloud` to `render-v6` and `render-legacy`.
2. Implement the shared resolver in a way that could later support `rmdoc render`.
3. Do not widen the scope into a CLI migration.
4. Capture this review so the next cleanup pass starts from evidence instead of intuition.

That is the right size. It improves the current UX without prematurely redesigning the whole command tree.

## References

- `cmd/remarquee/main.go:17-37`
- `cmd/remarquee/cmds/status.go:9-20`
- `cmd/remarquee/cmds/cloud/root.go:7-170`
- `cmd/remarquee/cmds/cloud/find.go:44-139`
- `cmd/remarquee/cmds/cloud/search.go:51-245`
- `cmd/remarquee/cmds/cloud/ls.go:54-224`
- `cmd/remarquee/cmds/cloud/stat.go:42-177`
- `cmd/remarquee/cmds/rmdoc/root.go:5-88`
- `cmd/remarquee/cmds/rmdoc/build_background.go:42-50`
- `cmd/remarquee/cmds/rmdoc/render_legacy.go:105-204`
- `cmd/remarquee/cmds/rmdoc/render_v6.go:87-183`
- `cmd/remarquee/cmds/rmdoc/render_v6_png.go:52-198`
- `cmd/remarquee/cmds/rmdoc/vlm_validate.go:61-220`
- `ttmp/2026/01/17/MO-001-CLEANUP-CLI--consolidate-and-improve-the-remarquee-cli/analysis/01-remarquee-cli-inventory-and-analysis.md:45-69`
- `ttmp/2025/12/14/RMQ-0004--port-rmdoc-parsing-rendering-to-go-v3-v5-v6/index.md:56-66`
