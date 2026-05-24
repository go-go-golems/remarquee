---
Title: Investigation Diary
Ticket: RMQ-0016
Status: active
Topics:
    - mdpdf
    - mermaid
    - image-embed
    - diary
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Chronological investigation and implementation diary for RMQ-0016"
LastUpdated: 2026-05-23
WhatFor: "Track investigation steps, decisions, and implementation"
WhenToUse: "Read before resuming work on this ticket"
---

# Investigation Diary

## Goal

Capture the investigation, design, and implementation steps for adding Mermaid diagram rendering and image embed support to the remarquee markdown-to-PDF pipeline.

## Step 1: Architecture Discovery and Design Document Creation

This step covers the initial analysis of the remarquee codebase, understanding the markdown-to-PDF pipeline, identifying gaps, and producing a comprehensive design document for an intern audience.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to add mermaid diagram and image embed support for remarquee's markdown rendering. Only work within this workspace, and remarquee is in ./remarquee. Create a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet points and pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and then upload to remarkable."

**Assistant interpretation:** Create a docmgr ticket, deeply analyze the remarquee codebase (specifically the mdpdf package and upload commands), write an intern-ready design document, store in ticket, upload to reMarkable.

**Inferred user intent:** Produce a thorough, actionable design document that a new team member can use to implement Mermaid diagram rendering and image embedding.

### What I did

1. Read skill files (ticket-research-docmgr-remarkable, docmgr, remarkable-upload, diary).
2. Mapped full remarquee codebase: Go packages, CLI commands, frontend, docs.
3. Deep-read all files in `pkg/mdpdf/` and `cmd/remarquee/cmds/upload/`.
4. Identified critical gaps: temp dir breaks relative image paths; pandoc renders mermaid as plain text.
5. Created docmgr ticket RMQ-0016 (after RMQ-0014/0015 collisions).
6. Wrote design document (~38 KB, 13 sections).
7. Uploaded bundle to reMarkable at `/ai/2026/05/23/RMQ-0016`.

### What worked

- The `ConvertMarkdownFileToPDF` function has clear insertion points for new preprocessing steps.
- The `tmpDir` pattern provides a natural location for image files.

### What didn't work

- Ticket ID collisions with RMQ-0014 and RMQ-0015.

### What was tricky to build

- Bundle mode image path uniqueness: chose collision avoidance via suffixed names.
- Preprocessing order: image resolution before mermaid rendering.

### What should be done in the future

- mmdc Puppeteer persistent browser mode for faster rendering.
- Content-hash dedup for bundle mode images.

---

## Step 2: Open Question Decisions and Task Setup

### Prompt Context

**User prompt (verbatim):** "10.3.1 : yes 10.3.2 : no upload src mermaid 10.3.3 : no. Add phases and tasks to the ticket, then implement them one by one, commit at appropriate intervals, keep a diary as you work."

**Inferred user intent:** Record the design decisions, create granular tasks, and implement all phases.

### What I did

1. Recorded decisions in design doc: `--mermaid-width` flag yes (10.3.1), no upload src mermaid (10.3.2), no bundle image dedup (10.3.3).
2. Updated tasks.md with 5 phases of granular implementation tasks.

---

## Step 3: Phase 1 — Image Path Resolution (commit 6a960c9)

Implemented `ResolveImagePaths` in `pkg/mdpdf/images.go`. Copies referenced images into the pandoc temp directory and rewrites Markdown image paths. Handles relative paths, collision avoidance, and skips URLs/absolute paths. Integrated into `ConvertMarkdownFileToPDF`.

### What I did

1. Created `pkg/mdpdf/images.go` with `ResolveImagePaths`, `isURL`, `copyFile` helpers.
2. Created `pkg/mdpdf/images_test.go` with 10 unit tests.
3. Updated `ConvertMarkdownFileToPDF` in `pandoc.go` to create `tmpDir` before preprocessing, then call `ResolveImagePaths`.
4. Added `pandocAvailable` helper and `TestConvertMarkdownFileToPDFWithImage` integration test.
5. All 25 mdpdf tests pass.

### What was tricky to build

- The tmpDir must be created **before** `ResolveImagePaths` runs (it needs tmpDir to copy images into). This required moving the `os.MkdirTemp` call earlier in `ConvertMarkdownFileToPDF`.
- The collision avoidance test initially called `ResolveImagePaths` twice with the same tmpDir, expecting state to persist. Fixed: the test now puts both images in the same body so the `usedNames` map works within a single call.

### Code review instructions

- `pkg/mdpdf/images.go`: Full review, ~120 lines.
- `pkg/mdpdf/pandoc.go`: Lines 67-84 (the reordered preprocessing chain).
- `pkg/mdpdf/pandoc_test.go`: New `TestConvertMarkdownFileToPDFWithImage`.

---

## Step 4: Phase 2 — Mermaid Block Rendering (commit 162a75c)

Implemented `RenderMermaidBlocks` in `pkg/mdpdf/mermaid.go`. Detects ```` ```mermaid ```` blocks, renders to PNG via mmdc CLI, replaces with image embeds. Graceful fallback when mmdc is not installed.

### What I did

1. Created `pkg/mdpdf/mermaid.go` with `MermaidRendererConfig`, `RenderMermaidBlocks`, `renderMermaidToPNG`, `resolveMmdcPath`.
2. Created `pkg/mdpdf/mermaid_test.go` with 12 tests (regex, config, path resolution, fallback).
3. Added `Mermaid *MermaidRendererConfig` field to `PandocOptions`.
4. Wired `RenderMermaidBlocks` into `ConvertMarkdownFileToPDF`.
5. All 36 mdpdf tests pass.

### What worked

- The regex `(?s)```mermaid\s*\n(.*?)\n\s*``` ` correctly handles multi-line mermaid blocks.
- Per-block error handling (warn + leave-as-is) means one bad diagram doesn't break the document.

### What was tricky to build

- mmdc is not installed on this machine, so we can only test the fallback path (mmdc not found → leave blocks unchanged). The actual rendering pipeline can only be tested manually after installing mmdc.

---

## Step 5: Phase 3 — CLI Flag Integration (commit 2ccac3e)

Added Mermaid and image flags to `upload md` and `upload bundle` commands.

### What I did

1. Added `mermaidFlags` struct in `layout.go` with `ToConfig()` method.
2. Added mermaid fields to `uploadMarkdownSettings` and `uploadBundleSettings`.
3. Added 7 new CLI flags to both commands: `--mermaid`, `--mmdc-path`, `--mermaid-scale`, `--mermaid-theme`, `--mermaid-bg`, `--mermaid-width`, `--resolve-images`.
4. Updated `configureMarkdownPandocOptions` to accept `*MermaidRendererConfig`.
5. Updated all 3 callers: md.go, bundle.go, sync.go (sync passes nil).
6. All 22 upload tests pass.

---

## Step 6: Phase 4 — Bundle Mode Support (commit 5606cb8)

Updated `BuildBundleMarkdown` to do per-file image resolution and Mermaid rendering before concatenation.

### What I did

1. Changed `BuildBundleMarkdown` signature to `(ctx context.Context, inputs []BundleInput, tmpDir string, mermaidCfg *MermaidRendererConfig)`.
2. For each input file: read → strip YAML → resolve images → render mermaid → normalize lists → concatenate.
3. Updated `writeBundlePDF` to create tmpDir early (before `BuildBundleMarkdown` call).
4. Added `TestBuildBundleMarkdown_ResolvesImages` test.
5. All 38 mdpdf tests pass, all 22 upload tests pass.

### What was tricky to build

- The `writeBundlePDF` function had `tmpDir` declared after `BuildBundleMarkdown` was called. Had to reorder so tmpDir exists first. This was caught during compilation, not testing.

### What warrants a second pair of eyes

- The double image copy in bundle mode: `BuildBundleMarkdown` copies images into `tmpDir1/images/`, then `ConvertMarkdownFileToPDF` copies them again from there into `tmpDir2/images/`. Works correctly but is wasteful. Acceptable for typical document sizes.

---

## Step 7: Phase 5 — Documentation (commit cd216c6)

Updated README with Mermaid and image embed examples.

### What I did

1. Added examples for `--mermaid`, `--resolve-images`, `--mermaid-scale`, `--mermaid-width`, `--mermaid-theme` in the "Upload Markdown" section.
2. Updated "What it does" section to mention image and Mermaid support.

---

## Summary

### Commits

| Phase | Commit | Description |
|-------|--------|-------------|
| 1 | `6a960c9` | Image path resolution (`pkg/mdpdf/images.go`) |
| 2 | `162a75c` | Mermaid block rendering (`pkg/mdpdf/mermaid.go`) |
| 3 | `2ccac3e` | CLI flags on upload md/bundle |
| 4 | `5606cb8` | Bundle mode per-file preprocessing |
| 5 | `cd216c6` | README documentation |

### Files created

- `pkg/mdpdf/images.go` — `ResolveImagePaths`
- `pkg/mdpdf/images_test.go` — 10 tests
- `pkg/mdpdf/mermaid.go` — `RenderMermaidBlocks`, `MermaidRendererConfig`
- `pkg/mdpdf/mermaid_test.go` — 12 tests

### Files modified

- `pkg/mdpdf/pandoc.go` — added `Mermaid` field to `PandocOptions`, reordered preprocessing chain
- `pkg/mdpdf/pandoc_test.go` — added integration test with image
- `pkg/mdpdf/bundle.go` — updated `BuildBundleMarkdown` signature
- `pkg/mdpdf/bundle_test.go` — added image resolution test
- `cmd/remarquee/cmds/upload/layout.go` — added `mermaidFlags`, updated `configureMarkdownPandocOptions`
- `cmd/remarquee/cmds/upload/md.go` — added 7 CLI flags
- `cmd/remarquee/cmds/upload/bundle.go` — added 7 CLI flags, updated `writeBundlePDF`
- `cmd/remarquee/cmds/upload/sync.go` — pass nil mermaid config
- `README.md` — added examples

### Test results

- 38 mdpdf tests pass
- 22 upload tests pass
- No regressions in existing tests

---

## Step 8: PR Review Cleanup and Mermaid Flag Section (commit b331544)

This step took over the feature branch after review surfaced several correctness issues and after the Mermaid flag wiring had started to drift into an awkward half-Glazed, half-Cobra shape. The goal was to preserve the working behavior while making the command surface cleaner, eliminating collision bugs, and restoring compatibility around pandoc path handling.

### Prompt Context

**User prompt (verbatim):** "Make a custom glazed schema section for the mermaid stuff, in order to not register them in the default section"

**Follow-up prompt (verbatim):** "Address the code review issues as well: https://github.com/go-go-golems/remarquee/pull/13

Basically, we're takig over for our little brother who has started to struggle cleaning and wrapping things up. Go through their work, tighten things up, add proper patterns where things are confused, remove deprecated or unnecessary complexity.

Keep a diary and commit at appropriate intervals"

**Assistant interpretation:** Finish the partially-started Mermaid flag grouping using a proper Glazed section, then address all automated PR review comments and clean up confused implementation patterns.

**Inferred user intent:** Make the PR production-quality: correct edge cases, reduce hand-rolled complexity, keep the command help organized, and document what changed.

### What I did

- Fetched PR #13 inline review comments with `gh api repos/go-go-golems/remarquee/pulls/13/comments --paginate`.
- Replaced direct Mermaid flag registration in `upload md` and `upload bundle` with a dedicated Glazed `schema.SectionImpl` in `cmd/remarquee/cmds/upload/mermaid_section.go`.
- Kept parsing simple by using Cobra flag accessors (`GetBool`, `GetString`, `GetInt`) after the Glazed section adds and groups the flags.
- Verified `remarquee upload md --help` and `remarquee upload bundle --help` now render a separate `## Mermaid flags:` group.
- Fixed PR review P1: Mermaid filenames now get a per-input `bundle-###-` prefix in bundle mode, avoiding repeated `mermaid-001.png` overwrites.
- Fixed PR review P1: copied image filenames now get the same per-input prefix in bundle mode, avoiding same-basename image overwrites across bundled files.
- Fixed PR review P1: `ConvertMarkdownFileToPDF` resolves relative `outPDF` and `--latex-header-file` to absolute paths before setting `cmd.Dir = tmpDir`, preserving existing CLI behavior.
- Wired `--resolve-images` into `PandocOptions.ResolveImages` so the flag is no longer decorative.
- Added regression tests for grouped Mermaid flag annotations, Mermaid flag defaults/overrides, bundle image collisions, bundle Mermaid collisions, and relative output/header paths.

### What worked

- Glazed already had the exact lower-level primitive needed: `schema.SectionImpl.AddSectionToCobraCommand` adds flags and registers a flag group through Cobra annotations.
- The existing custom help template already understands those annotations, so no help-template changes were needed.
- The code review comments were accurate: the filename collision bugs were real, and `cmd.Dir = tmpDir` did change how relative output/header paths resolved.

### What didn't work

- My first attempt at parsing the Glazed section via `SectionValues` was too complicated and initially failed to include default values correctly.
- The better pattern was: use Glazed only for schema/registration/grouped help, and use Cobra's typed flag getters for runtime parsing in these plain Cobra upload commands.

### What I learned

- Glazed flag grouping is annotation-driven (`glazed:flag-group:*`) and can be applied to plain Cobra commands without converting the command to a full Glazed command.
- Pandoc path behavior has two separate concerns:
  - relative resources need `cmd.Dir = tmpDir` so copied images are visible;
  - relative output/header paths must be made absolute before changing `cmd.Dir`.

### What was tricky to build

- Bundle mode has two layers of temporary paths: the bundle staging dir and the inner pandoc conversion dir. Prefixing assets at bundle-staging time prevents collisions; the later conversion pass can safely copy those already-prefixed paths again.
- The Mermaid flags needed to be grouped without duplicating the same flag declarations in `md.go` and `bundle.go`. A reusable `NewMermaidSection()` plus `addMermaidFlagsToCommand()` solved this.

### What warrants a second pair of eyes

- `ResolveImagePathsWithPrefix` intentionally prefixes copied basenames rather than preserving source subdirectories. This keeps paths short and stable, but reviewers should confirm that is acceptable for very large bundles.
- `mermaidConfigWithImagePrefix` clones the config and prepends the bundle prefix to any existing `ImagePrefix`; this is intended but worth checking if future callers use `ImagePrefix` directly.

### What should be done in the future

- Consider consolidating all markdown-render preprocessing options into a small explicit options struct if more flags are added.
- Consider a more complete Markdown image parser if we need to preserve title attributes or handle nested parentheses in image URLs.

### Code review instructions

- Start with `cmd/remarquee/cmds/upload/mermaid_section.go` for the new Glazed section and parser.
- Then review `pkg/mdpdf/bundle.go`, `pkg/mdpdf/images.go`, and `pkg/mdpdf/mermaid.go` for the bundle collision fixes.
- Then review `pkg/mdpdf/pandoc.go` for path handling around `cmd.Dir = tmpDir`.
- Validate with:
  - `go test ./pkg/mdpdf/ ./cmd/remarquee/cmds/upload/ -count=1`
  - `go run ./cmd/remarquee upload md --help | grep -n "Mermaid flags\|--mermaid-pdf-width"`

### Technical details

PR review comments addressed:

1. `pkg/mdpdf/mermaid.go`: globally unique Mermaid image names in bundle mode.
2. `pkg/mdpdf/pandoc.go`: resolve relative header/output paths before changing pandoc working directory.
3. `pkg/mdpdf/images.go`: avoid image filename collisions across bundled Markdown files.

Verification from the commit hook:

- `golangci-lint run -v` — 0 issues.
- `go test ./...` — passed.
