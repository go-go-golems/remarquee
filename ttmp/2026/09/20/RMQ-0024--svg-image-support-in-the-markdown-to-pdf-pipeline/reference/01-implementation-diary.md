---
Title: Implementation diary
Ticket: RMQ-0024
Status: active
Topics:
    - markdown
    - pdf
    - mdpdf
    - image-embed
    - pandoc
    - xelatex
DocType: reference
Intent: long-term
Owners: []
ExternalSources: []
Summary: "Investigation diary for RMQ-0024: empirical baseline of SVG behavior in the mdpdf pipeline and the resulting corrected design scope."
LastUpdated: 2026-09-20T18:10:00-04:00
WhatFor: "Chronological record of the SVG investigation, evidence, and decisions."
WhenToUse: "When resuming RMQ-0024 or reviewing why the original scope was corrected."
---

# Implementation diary — RMQ-0024

## Step 1: Reconnaissance and empirical baseline

This step established what the Markdown-to-PDF pipeline actually does with SVG
today, before writing any design. The original request assumed XeLaTeX cannot
render SVG at all and that a converter step had to be added; I wanted to verify
that assumption against the real pandoc/rsvg-convert/TeX Live stack on this
machine rather than trust it. The investigation was intentionally read-only with
respect to the repository (all experiments ran in `/tmp`), and it materially
changed the ticket's scope.

### Prompt Context
**User prompt (verbatim):** "ok, crate a newdocmgr ticket for option A and for
extracting inline blocks to a temp .svg, converting it, and substituting in the
markdown as well.

Create  a detailed analysis / design / implementation guide that is for a new
intern, explaining all the parts of the system needed to understand what it is,
with prose paragraphs and bullet point sand and pseudocode and diagrams and api
references and file references. It should be very clear and technical. Store in
the ticket and the nupload to remarkable."

(Preceded in the same session by a read-only question: "What would it take to add /
embed SVGs in markdowns to be uploaded and rendered to the remarkable? Answer here,
do not touch anything.")

**Assistant interpretation:** Create a docmgr ticket, write an intern-grade design
and implementation guide for (a) converting referenced SVG and (b) extracting
inline `<svg>` blocks, store it in the ticket, and upload it to reMarkable.

**Inferred user intent:** Give a new engineer a complete, self-contained plan they
can implement without further research, and get a readable copy onto the tablet.

**Commit (code):** n/a (documentation-only step; no code changed).

### What I did
- Read the pipeline sources: `pkg/mdpdf/pandoc.go`, `images.go`, `mermaid.go`,
  `bundle.go`, `preprocess.go`, `literal_regions.go`, `layout.go`, and the
  `cmd/remarquee/cmds/upload/*.go` command layer.
- Checked tool availability: `rsvg-convert`, `inkscape`, `convert`, `pandoc`,
  `xelatex`, `dvisvgm` present; `cairosvg` and `mmdc` missing.
- Ran isolation experiments in `/tmp/svgtest` with pandoc 3.1.3:
  - `-t latex` on a referenced SVG emits `\usepackage{svg}` + `\includesvg`.
  - Actual PDF generation instead runs
    `rsvg-convert -f pdf -a --dpi-x 96 --dpi-y 96` and emits
    `\includegraphics{<hash>.pdf}` (captured by wrapping the pdf engine).
  - A raw `<svg>` block produces no LaTeX output at all (dropped).
- Ran the real installed `remarquee upload md --pdf-only` on fixtures containing
  referenced SVG, sized SVG, inline raw SVG, `data:image/svg+xml`, and HTML
  `<img src=svg>`, then pixel-probed the rendered PDFs with `pdftoppm` + PIL.
- Tested the missing-converter path with a failing `rsvg-convert` stub on `PATH`.

### Why
- The ticket's premise determines the entire implementation. If pandoc already
  converts referenced SVG, writing a converter for it would duplicate work and
  miss the actual gaps. Evidence before design is cheaper than building the wrong
  thing.
- Pixel probing (rather than trusting exit codes) is necessary because pandoc can
  produce a PDF that silently omits an image.

### What worked
- Confirmed pandoc's actual mechanism via a fake `rsvg-convert` that logs argv:
  `-f pdf -a --dpi-x 96 --dpi-y 96 -o <hash>.pdf <input.svg>`.
- Confirmed with the real `remarquee` binary:
  - Referenced `![](*.svg)` renders (blue pixels present).
  - `{width=40%}` sizing is applied (`\includegraphics[width=...]`).
  - `data:image/svg+xml;base64,...` renders (blue pixels present).
  - Inline raw `<svg>` is dropped (zero red pixels).
  - HTML `<img src="logo.svg">` is dropped (zero blue pixels).
- Confirmed `xelatex -shell-escape` makes `\includesvg` work via Inkscape, and
  that without shell escape it fails with
  `! Package svg Error: File 'logo_svg-tex.pdf' is missing.`.

### What didn't work
- A first attempt to trace pandoc's engine invocation with `--pdf-engine=./faketex`
  failed: pandoc rejects unclassified engine paths
  (`pdf-engine must be one of wkhtmltopdf, weasyprint, ..., xelatex, ...`).
  Recovered by placing a fake `xelatex` earlier on `PATH`.
- A first attempt to remove `rsvg-convert` from `PATH` failed because `/usr/bin`
  still contained it. Recovered by shadowing it with a stub that exits 127, which
  then reproduced the hard-failure path.

### What I learned
- The original premise was wrong in an important way: **referenced SVG already
  works** through pandoc's own `rsvg-convert` integration. The real gaps are
  inline raw `<svg>`, HTML `<img>`, graceful degradation when `rsvg-convert` is
  absent, and sizing policy.
- Pandoc's LaTeX **writer** output (`-t latex`) is not the same as its **PDF**
  output path; the former can show `\includesvg` while the latter converts to PDF
  and uses `\includegraphics`. Do not diagnose the pipeline from `-t latex` alone.
- Default sizing follows the SVG's intrinsic dimensions (96 dpi), which can
  surprise authors; a `viewBox`-only SVG rendered at roughly 70% of text width in
  the test.

### What was tricky to build
- N/A for this step (investigation only). The tricky part was distinguishing the
  two pandoc code paths and proving image presence with pixels rather than exit
  status.

### What warrants a second pair of eyes
- The claim that referenced SVG "already works" is environment-dependent: it
  requires `rsvg-convert` (or another converter pandoc can find). On a machine
  without it, the same document hard-fails. The design must therefore treat
  graceful degradation as first-class, not optional.
- Whether `data:image/svg+xml` behavior is stable across pandoc versions.

### What should be done in the future
- Implement Phases 1–6 from the design guide (§8).
- Consider pre-converting referenced SVGs when no converter is available but
  Inkscape is, to remove the pandoc fallback hard-failure (open question §12.3).

### Code review instructions
- Read `ttmp/2026/09/20/RMQ-0024--svg-image-support-in-the-markdown-to-pdf-pipeline/design-doc/01-svg-image-support-design-and-implementation-guide-for-a-new-intern.md`
  §5.2 (behavior matrix) and §13 (Appendix A) first.
- Re-run Appendix A commands to reproduce the baseline before trusting the scope.
- Starting files for implementation: `pkg/mdpdf/pandoc.go`, `images.go`,
  `mermaid.go`, `bundle.go`, `cmd/remarquee/cmds/upload/mermaid_section.go`.

### Technical details
- pandoc 3.1.3; `rsvg-convert` present at `/usr/bin/rsvg-convert`; TeX Live
  XeLaTeX present; `cairosvg`/`mmdc` absent.
- Evidence commands and outputs are recorded in design guide Appendix A.
- No repository files were modified during the investigation.

## Step 2: Ticket, design guide, and reMarkable delivery

This step created the ticket, wrote the intern-facing design and implementation
guide, and prepared the requested reMarkable upload. The guide records the
corrected scope from Step 1 and includes the full system tour, pseudocode,
diagrams, API reference, file index, and a phased implementation plan.

### Prompt Context
**User prompt (verbatim):** see Step 1.
**Assistant interpretation:** Store the guide in the ticket and upload it.
**Inferred user intent:** A durable, reviewable, tablet-readable handoff.
**Commit (code):** n/a (documentation only).

### What I did
- Created ticket `RMQ-0024` with topics
  `markdown,pdf,mdpdf,image-embed,pandoc,xelatex`.
- Added a `design-doc` and a `reference` diary.
- Wrote
  `design-doc/01-svg-image-support-design-and-implementation-guide-for-a-new-intern.md`.
- Added implementation tasks to `tasks.md`.

### Why
- The guide is the deliverable; the diary preserves the evidence and the reason
  the scope was corrected.

### What worked
- Ticket, docs, and guide created under
  `ttmp/2026/09/20/RMQ-0024--svg-image-support-in-the-markdown-to-pdf-pipeline/`.

### What didn't work
- No failures observed in this step.

### What I learned
- N/A (bookkeeping step).

### What was tricky to build
- Keeping the design guide honest about the corrected premise while still
  delivering a complete implementation plan for the remaining work.

### What warrants a second pair of eyes
- Review the sizing policy recommendation and the HTML `<img>` scope in §12.

### What should be done in the future
- Begin Phase 1 implementation; keep this diary updated per phase.

### Code review instructions
- See design guide §8 (plan) and §12 (open questions).

### Technical details
- Ticket root: `ttmp/2026/09/20/RMQ-0024--svg-image-support-in-the-markdown-to-pdf-pipeline/`.

## Step 3: reMarkable delivery

The guide and this diary were bundled into a single PDF and uploaded to the
ticket-aware reMarkable destination. This is a routine delivery step: the command
result and destination are the evidence.

### What I did
- Ran `remarquee upload bundle` over the design guide and this diary with
  `--toc-depth 2 --non-interactive`, destination `/ai/2026/09/20/RMQ-0024`.

### What worked
- `OK: uploaded RMQ-0024 SVG Image Support Guide.pdf -> /ai/2026/09/20/RMQ-0024`.
  Cloud delivery is confirmed; physical device synchronization is separate.

### What didn't work
- No failures observed.

### Technical details
- Inputs: design-doc/01-...-guide-...md and reference/01-implementation-diary.md.
- Document name: `RMQ-0024 SVG Image Support Guide.pdf`.

## Step 4: Phase 1 — SVG config and converter discovery

This step laid the foundation for all later SVG work: a configuration type, a
non-fatal converter-discovery function, and wiring into `PandocOptions`. I chose
to return `("", nil)` when no converter exists rather than an error, because the
design (guide §6.4) requires graceful degradation — the document must still
render even when `rsvg-convert`/Inkscape are absent. The explicit-`ConverterPath`
case is the one genuine configuration error, so it alone returns an error.

### Prompt Context
**User prompt (verbatim):** "Implement, commit docs, commit at appropriate
intervals and keep a detailed diary as you work (using the diary format from the
skill)" plus the work-slip instructions. See Step 1 for the originating request.
**Assistant interpretation:** Implement the guide's phased plan, committing at
phase boundaries and maintaining a detailed diary.
**Inferred user intent:** A reviewable, incremental implementation with evidence.
**Commit (code):** see below (Phase 1 commit).

### What I did
- Added `pkg/mdpdf/svg.go`: `SVGRendererConfig`, `DefaultSVGRendererConfig`,
  `ResolveSVGConverter`, `svgConfigWithImagePrefix`, and `warnf`.
- Added `SVG *SVGRendererConfig` to `PandocOptions` in `pkg/mdpdf/pandoc.go`.
- Added `pkg/mdpdf/svg_test.go` with 10 tests covering nil/disabled/explicit/
  PATH-order/fallback/no-converter/prefix/warning cases.

### Why
- A single config type mirrors `MermaidRendererConfig` and keeps CLI wiring simple.
- `("", nil)` on absence keeps the "warn and continue" contract from the guide.
- `svgConfigWithImagePrefix` mirrors `mermaidConfigWithImagePrefix` for bundle mode.

### What worked
- `go build ./...` succeeded.
- `go test ./pkg/mdpdf/ -run SVG` passed all 10 tests.
- Full `go test ./pkg/mdpdf/...` passed (`ok ... 13.759s`).

### What didn't work
- No failures observed in this phase.

### What I learned
- `exec.LookPath` is trivially testable by overriding `PATH` with `t.Setenv`, so
  no real converters are needed for unit tests.

### What was tricky to build
- Nothing complex yet. The deliberate design choice was error semantics: absence
  must not be an error, but an explicitly configured missing path must be.

### What warrants a second pair of eyes
- Whether `DefaultSVGRendererConfig` should be enabled by default and injected in
  `DefaultPandocOptions`, or only constructed by the CLI (currently the latter).

### What should be done in the future
- Phase 2: implement the quote-aware `<svg>` block scanner and
  `ResolveInlineSVGBlocks`.

### Code review instructions
- Read `pkg/mdpdf/svg.go` top to bottom; it is small.
- Run `go test ./pkg/mdpdf/ -run SVG -v`.

### Technical details
- New symbols: `SVGRendererConfig` (Enabled, ConverterPath, ConverterOrder,
  DefaultWidth, MaxWidth, ImagePrefix, WarnWriter), `DefaultSVGRendererConfig`,
  `ResolveSVGConverter`, `svgConfigWithImagePrefix`, `(*SVGRendererConfig).warnf`.
- `PandocOptions.SVG` added.

## Step 5: Phase 2 — inline `<svg>` extraction and conversion

This step implemented the core feature: detecting raw `<svg>…</svg>` blocks that
pandoc silently drops, converting them to vector PDFs, and substituting standard
Markdown image syntax. The hardest part is not conversion but *finding* the block
correctly: a naive `<svg.*?</svg>` regex would corrupt fenced code that documents
SVG, break on nested `<svg>`, and fail when an attribute value contains `>` or
`</svg>`. I used a line scan with a quote-aware tag-depth tokenizer and reused the
existing `literalLines` guard, matching the repository's style.

### Prompt Context
**User prompt (verbatim):** see Step 4 (implement the plan, commit at intervals,
keep a detailed diary). Mid-phase the user also asked to update the work-slip skill,
see Step 6. **Assistant interpretation:** implement guide Phase 2.
**Inferred user intent:** Working inline-SVG support with tests and evidence.
**Commit (code):** see below (Phase 2 commit).

### What I did
- Extended `pkg/mdpdf/svg.go` with `looksLikeSVGOpen`, `hasPrefixFold`,
  `isTagBoundary`, `countTagDelta`, `tagEnd`, `collectSVGBlock`,
  `convertSVGToPDF`, `ResolveInlineSVGBlocks`, `containsInlineSVG`.
- Added 11 tests (tokenizer, extraction, fenced-code safety, unclosed, no
  converter, failure, width/prefix).

### Why
- `collectSVGBlock` + `countTagDelta` handle nested tags and quoted attributes,
  which a regex cannot do safely.
- `convertSVGToPDF` supports rsvg-convert and Inkscape argv styles and produces a
  vector PDF next to the extracted SVG, so pandoc includes it via
  `\includegraphics` with no shell escape.
- Every failure path warns and leaves the source unchanged, matching the design.

### What worked
- All 20 SVG-related tests pass; full `go test ./pkg/mdpdf/...` passes.
- `gofmt` clean, `go build ./...` clean.

### What didn't work
- First test compile failed: two tests passed a value `cfg` where a pointer was
  expected (`cannot use cfg ... as *SVGRendererConfig`). Recovered by making
  `newInlineTestConfig` return a value so all call sites use `&cfg` uniformly.

### What I learned
- Ordering matters subtly: `ResolveInlineSVGBlocks` must run **after**
  `ResolveImagePaths` (like Mermaid), because it writes finished files directly
  into `<tmp>/images/` and `ResolveImagePaths` resolves relative paths against the
  *source* directory, not the temp dir. Writing generated assets directly to the
  temp dir sidesteps that. The committed design guide's ordering note is
  superseded here; the implementation (and Phase 4 wiring) uses post-image order.

### What was tricky to build
- The quote-aware tokenizer. ``<svg data-x="<\/svg>">`` must count as depth +1,
  and `<svg/>` must be depth-neutral. `tagEnd` skips quoted values and detects a
  trailing `/`; `countTagDelta` skips quoted spans before interpreting `<`.

### What warrants a second pair of eyes
- Whether generated `.pdf` assets should be substituted directly versus emitting
  `.svg` and letting pandoc convert. We chose direct conversion for determinism and
  to make the no-converter degradation meaningful; referenced SVGs still rely on
  pandoc. Reviewers should confirm this asymmetry is acceptable.
- `looksLikeSVGOpen` only matches a line-leading `<svg>`; inline mid-paragraph
  `<svg>` is out of scope per the guide's non-goals.

### What should be done in the future
- Phase 3: `ResolveHTMLImages` for `<img src="...svg">`.

### Code review instructions
- Read `pkg/mdpdf/svg.go` functions `countTagDelta`, `collectSVGBlock`,
  `ResolveInlineSVGBlocks`.
- Run `go test ./pkg/mdpdf/ -run 'SVG|InlineSVG|CountTag|LooksLike' -v`.

### Technical details
- Generated assets: `<tmp>/images/<prefix>svg-NNN.svg` (source) and `.pdf`
  (converted). Substitution: `![svg image N](./images/<prefix>svg-NNN.pdf)` plus
  `{width=DefaultWidth}` when set.
- rsvg-convert args: `-f pdf -a --dpi-x 96 --dpi-y 96 -o OUT IN`.
- Inkscape args: `IN --export-type=pdf --export-filename=OUT`.

## Step 6: Work-slip skill — require goal and intent summaries

During Phase 2 the user asked to update the `brutalist-work-slip` skill so that
plan slips always explain the ticket's overall goal (what the effort is about),
and step/status slips always carry a short statement of the step's *intent* rather
than only its mechanical actions. The motivation is future reconstruction: a slip
is often read cold, so it must be self-explanatory.

### Prompt Context
**User prompt (verbatim):** "update the brutalist work slip skill to say to write
up what the ticket's goal is / what it is about, like you did, on the plan ticket,
and then a smaller version of the intent of each step (not ust what is going to be
done), so that it is easier to track / put things back together in the future."
**Assistant interpretation:** Edit the skill guidance; no script change required.
**Inferred user intent:** Make printed slips durable, self-contained work records.

### What I did
- Edited `~/.pi/agent/skills/brutalist-work-slip/SKILL.md`:
  - Mode 1 (`status`): mandate `--summary` state the step's INTENT, with a
    good/weak example.
  - Mode 2 (`plan`): mandate `--summary` state the ticket's GOAL/problem context,
    prefixed `GOAL:` when useful.
  - Content rules: summary is required in practice and must not just re-list
    bullets; assume the slip is read cold.
  - Updated both example invocations to model goal/intent summaries.

### Why
- Slips are the long-lived artifact of a work session; without intent they only
  record activity and cannot be reconstructed later.

### What worked
- Guidance and examples updated in place; no script behavior changed.

### What didn't work
- No failures observed.

### What was tricky to build
- Distinguishing "what was done" (`--did` bullets) from "why it mattered"
  (`--summary`) and enforcing it as guidance rather than a hard schema.

### What warrants a second pair of eyes
- Whether to enforce a non-empty `--summary` in the script (currently guidance
  only, to avoid breaking existing usage).

### What should be done in the future
- Consider a script-side warning when `--summary` is omitted.

### Code review instructions
- Read `~/.pi/agent/skills/brutalist-work-slip/SKILL.md` Mode 1, Mode 2, and
  Content rules sections.

### Technical details
- File: `~/.pi/agent/skills/brutalist-work-slip/SKILL.md` (outside this repo, so
  not part of the repository commit).

## Step 7: Phase 3 — HTML `<img>` SVG conversion

This step recovered the other silently-dropped SVG form: HTML `<img>` tags. Pandoc's
LaTeX writer discards raw HTML, so `<img src="arch.svg">` vanished. The fix
rewrites SVG `<img>` tags into standard Markdown image syntax, which then flows
through the existing `ResolveImagePaths` copy step. Because the rewrite must
happen before path resolution, `ResolveHTMLImages` is designed to run early in the
pipeline (Phase 4 wires that ordering).

### Prompt Context
**User prompt (verbatim):** see Step 4. **Assistant interpretation:** implement
guide Phase 3. **Inferred user intent:** HTML-embedded SVGs render like Markdown
ones. **Commit (code):** see below (Phase 3 commit).

### What I did
- Added to `pkg/mdpdf/svg.go`: `ResolveHTMLImages`, `rewriteHTMLImgTags`,
  `indexImgTag`, `htmlImgToMarkdown`, `parseTagAttrs`, `isHTMLSpace`,
  `normalizeHTMLDimension`, `isSVGSource`, `containsHTMLSVGImg`.
- Added 9 tests covering attributes, unit normalization, quoted `>`, fenced-code
  safety, non-SVG passthrough, no-converter warning, and default width.

### Why
- Only SVG sources are rewritten to bound the change; PNG/JPG `<img>` remain a
  known follow-up.
- Bare numeric dimensions become `px`, preserving `%`/`cm` units.
- The converter-availability guard prevents rewriting into Markdown that would
  hard-fail at pandoc when no converter exists.

### What worked
- All HTML-related tests pass; full `go test ./pkg/mdpdf/` passes.

### What didn't work
- One test assertion was wrong: I asserted the output must not contain `"> y"`,
  but that substring is legitimate alt text (`![x > y](a.svg)`). Fixed the
  assertion to check for absence of the raw `<img` tag instead.

### What I learned
- The API signature was reduced versus the guide: `ResolveHTMLImages(body, cfg)`
  needs neither `ctx`, `sourceDir`, nor `tmpDir`, because conversion is delegated
  to pandoc after the rewrite. This is a deliberate refinement of §7 of the guide.

### What was tricky to build
- Reusing `tagEnd` (from the `<svg>` tokenizer) for `<img>` so that quoted `>` in
  an attribute does not terminate the tag; then `parseTagAttrs` parses the tag
  body quote-aware.

### What warrants a second pair of eyes
- Attribute parsing is intentionally minimal (no entity decoding, no boolean
  attributes). Confirm it handles the HTML shapes authors actually use.

### What should be done in the future
- Phase 4: wire both passes into `ConvertMarkdownFileToPDF` and
  `BuildBundleMarkdown` in the correct order.

### Code review instructions
- Read `ResolveHTMLImages` and `htmlImgToMarkdown`.
- Run `go test ./pkg/mdpdf/ -run 'HTML|IsSVG|NormalizeHTML|ParseTag' -v`.

### Technical details
- `<img src="a.svg" width="100">` -> `![svg image](./a.svg){width=100px}`.
- `alt` is preserved; `height` is used when `width` is absent; `DefaultWidth` is
  the final fallback.

## Step 8: Phase 4 — pipeline and bundle wiring

This step connected the new SVG passes into both conversion paths. The critical
decision was ordering relative to `ResolveImagePaths`: HTML `<img>` rewriting must
happen *before* image path resolution (so the produced Markdown reference is copied
from the source directory), while inline `<svg>` extraction must happen *after*
(because it writes finished assets directly into the temp dir and
`ResolveImagePaths` resolves relative paths against the source dir, not the temp
dir). Getting this backwards would silently break referenced images or double-copy.

### Prompt Context
**User prompt (verbatim):** see Step 4. **Assistant interpretation:** implement
guide Phase 4. **Inferred user intent:** the feature actually runs for real uploads.
**Commit (code):** see below (Phase 4 commit).

### What I did
- `pkg/mdpdf/pandoc.go`: call `ResolveHTMLImages` before `ResolveImagePaths` and
  `ResolveInlineSVGBlocks` after it, inside `ConvertMarkdownFileToPDF`.
- `pkg/mdpdf/pandoc.go`: `DefaultPandocOptions` now injects a default
  `SVGRendererConfig`, so the feature is on by default for library callers.
- `pkg/mdpdf/bundle.go`: `BuildBundleMarkdown` gained an `svgCfg` parameter and
  applies both passes with the per-input `bundle-00N-` prefix.
- Updated callers: `cmd/.../upload/bundle.go` passes `pandocOpts.SVG`; four test
  call sites pass `nil`.
- Added `pkg/mdpdf/svg_pdf_test.go` with a real-converter test and a full
  pandoc/XeLaTeX pipeline test.

### Why
- Ordering preserves the existing image-copy semantics and avoids regressions.
- Enabling SVG in `DefaultPandocOptions` makes the feature live without requiring
  every caller to opt in; the CLI can still override via flags in Phase 5.

### What worked
- `TestResolveInlineSVGBlocks_RealConverter` produced a real `%PDF-` file.
- `TestSVGPipelinePDF` rendered a doc with referenced + inline + HTML SVG plus a
  fenced code block through the full pipeline without error.
- `go test ./pkg/mdpdf/... ./cmd/remarquee/cmds/upload/...` passes.

### What didn't work
- No failures observed in this phase.

### What I learned
- The committed design guide's ordering ("inline before images") was wrong for
  this codebase; the implementation uses post-image ordering and this entry
  supersedes it. The guide should be corrected in a docs pass.

### What was tricky to build
- Updating `BuildBundleMarkdown`'s signature rippled to 6 call sites (5 tests, 1
  command). Handled with targeted edits to avoid accidental changes.

### What warrants a second pair of eyes
- Whether enabling SVG by default in `DefaultPandocOptions` could surprise callers
  that do not expect subprocess conversion. It is a no-op when no inline SVG or
  HTML `<img svg>` is present.

### What should be done in the future
- Phase 5: expose `--svg*` flags and construct the config from them (currently the
  CLI inherits the default config).

### Code review instructions
- Read the preprocessing block in `ConvertMarkdownFileToPDF` for ordering.
- Run `go test ./pkg/mdpdf/ -run 'SVGPipelinePDF|RealConverter' -v`.

### Technical details
- Order: StripYAMLFrontmatter -> ResolveHTMLImages -> ResolveImagePaths ->
  ResolveInlineSVGBlocks -> RenderMermaidBlocks -> list normalization.

## Step 9: Phase 5 — CLI flags and documentation

This step exposed SVG handling to users through the upload commands and corrected
the design guide's now-stale ordering and API notes. The feature was already
enabled by default via `DefaultPandocOptions`; the flags exist so users can disable
it, point at a specific converter, or set a default width for extracted inline
SVGs. Mirroring the Mermaid Glazed section keeps help output consistent and
discoverable.

### Prompt Context
**User prompt (verbatim):** see Step 4. **Assistant interpretation:** implement
guide Phase 5. **Inferred user intent:** users can control the feature from the CLI.
**Commit (code):** see below (Phase 5 commit).

### What I did
- Added `cmd/remarquee/cmds/upload/svg_section.go` with a Glazed "SVG flags"
  section: `--svg` (default true), `--svg-converter`, `--svg-default-width`, plus
  `svgConfigFromCommand` and `addSVGFlagsToCommand`.
- Wired the section into `upload md`, `upload bundle`, and `upload sync`; each now
  parses an `svgCfg` and passes it to `configureMarkdownPandocOptions`.
- Extended `configureMarkdownPandocOptions` with an `svgCfg` parameter and set
  `opts.SVG`.
- Removed the unused `MaxWidth` field from `SVGRendererConfig`.
- Added `svg_section_test.go` (help grouping, defaults, overrides, disabled).
- Updated `README.md` with SVG examples and the design guide with an
  "Implementation revision" box plus corrected ordering/API/flag sections.

### Why
- Disabling and converter selection are operationally necessary (CI without
  rsvg-convert; explicit tool paths).
- The guide is the onboarding artifact; leaving wrong ordering advice would mislead
  the next implementer.

### What worked
- `go run ./cmd/remarquee upload md --help` shows a grouped "SVG flags" section
  with all three flags.
- Upload package tests and `pkg/mdpdf` tests pass.

### What didn't work
- No failures observed.

### What I learned
- The Glazed section pattern (`schema.NewSection` + `AddSectionToCobraCommand`)
  works identically for a second section; no parser plumbing was needed beyond
  `cmd.Flags().GetBool/GetString`.

### What was tricky to build
- Updating `configureMarkdownPandocOptions` and its three callers without missing
  one; a compile error would have caught omissions, and `go build ./...` was clean.

### What warrants a second pair of eyes
- Flag naming (`--svg` bool could be confused with a path). Consider
  `--inline-svg` if review prefers.

### What should be done in the future
- Phase 6: end-to-end validation, pixel verification, guide/ticket closeout.

### Code review instructions
- Read `svg_section.go`; run `go test ./cmd/remarquee/cmds/upload/ -run SVG -v`.
- Check the guide's revision box at the top.

### Technical details
- Config precedence: CLI flags -> `configureMarkdownPandocOptions` -> `opts.SVG`;
  `--svg=false` yields a nil config (no-op).

## Step 10: Phase 6 — end-to-end validation and closeout

This step proved the feature on real PDFs rather than only in unit tests. The key
question was whether inline raw `<svg>` now actually appears as an image; exit
codes alone cannot prove this because the pre-fix behavior silently dropped the
block while still producing a valid PDF. I rendered fixtures and counted pixels,
and I exercised the missing-converter and disable paths.

### Prompt Context
**User prompt (verbatim):** see Step 4. **Assistant interpretation:** implement
guide Phase 6 (validation, ticket closeout, requested reMarkable delivery).
**Inferred user intent:** an evidence-backed, delivered feature.
**Commit (code):** see below (Phase 6 docs commit).

### What I did
- Built the CLI from source (`go build -o remarquee-test ./cmd/remarquee`) and
  rendered a fixture containing referenced SVG, inline raw SVG, HTML `<img>`, and
  a fenced `<svg>` code block via `upload md --pdf-only`.
- Pixel-probed the rendered PDF.
- Ran the missing-converter path with a minimal `PATH` (pandoc + xelatex only).
- Ran bundle mode with two inline-SVG inputs, `--svg=false`, and
  `--svg-default-width 50%`.
- Ran `go test ./...`; all packages pass.

### What worked
- Direct render (`out/doc.pdf`, 1 page):
  - blue `#336699` pixels = **9838** (referenced + HTML `<img>` both rendered).
  - red `#cc3333` pixels = **3044** (inline raw `<svg>` now rendered; was 0 before).
  - `pdftotext` shows `Figure 2: svg image 1` and `Figure 3: svg image`, and the
    fenced `<svg ...>` appears as literal text → code fences preserved.
- Missing converter: printed
  `WARNING: SVG: no SVG converter found; inline <svg> blocks left as-is` and still
  produced `OK: generated out-noConv/inline-only.pdf` (exit 0) — graceful.
- Bundle: produced `SVG Bundle Test.pdf`; red pixels present on both pages
  (3741 and 3828) → per-input prefixing works.
- `--svg=false`: red pixels = 0 → disabled correctly.
- `--svg-default-width 50%`: produced a PDF without error.
- `go test ./...` all green.

### What didn't work
- No failures observed in this phase.

### What I learned
- Pixel probing is essential for this feature: the broken behavior still produced
  a valid PDF, so only image-content checks distinguish fixed from broken.

### What was tricky to build
- Constructing a `PATH` that keeps pandoc/xelatex working while hiding
  rsvg-convert; solved by symlinking only `pandoc` and `xelatex` into a temp dir.

### What warrants a second pair of eyes
- The `--svg=false` disable path silently drops inline SVG (by design), which may
  surprise; documented in the flag help.

### What should be done in the future
- Optional: HTML `<img>` support for PNG/JPG (currently left untouched).
- Optional: pre-convert referenced SVGs when no converter is present.

### Code review instructions
- Re-run the fixture render and pixel probe (commands in the design guide
  Appendix A and this entry).
- `go test ./...`.

### Technical details
- Fixture and outputs under `/tmp/svgtest/rmq2` (throwaway).
- Commits for this ticket: docs 5d9f0b9, P1 a06e76d, P2 2c3ac2b, P3 4b6483b,
  P4 7d9bda7, P5 ebaf952.

## Step 11: Documentation gap — Markdown format guide

After Phase 6 the user asked whether the remarquee documentation was updated and
whether a document exists describing the supported Markdown format. The answer was
partly no: only `README.md` had received SVG examples, and the in-CLI Glazed help
docs under `pkg/doc/` were stale and had no format/support overview. This step
closes that gap and fixes a related correctness issue discovered while documenting.

### Prompt Context
**User prompt (verbatim):** "ldid you update the remarquee documentation? is there
a document about the markdown format and what is possible and what isn't in there?"
**Assistant interpretation:** verify docs coverage, then add the missing
format/support document and update command references.
**Inferred user intent:** durable user-facing documentation of what renders.
**Commit (code):** see below (Step 11 commit).

### What I did
- Created `pkg/doc/topics/markdown-format-support.md` (embedded by the existing
  `//go:embed topics/*.md` wildcard): support matrix, frontmatter, math, images,
  SVG (all three forms + flags + degradation + limitations), Mermaid, code blocks,
  unsupported features, layout/bundles, troubleshooting.
- Updated `pkg/doc/upload/02-remarquee-upload-reference.md`: conversion behavior,
  image/SVG/Mermaid flags, and a pointer to `remarquee help markdown-format-support`.
- Fixed gating: `ResolveHTMLImages` is now called only when `ResolveImages` is true,
  in both direct and bundle paths, because a rewritten HTML image becomes a
  relative Markdown reference that must be staged by `ResolveImagePaths`. Previously
  `--resolve-images=false` combined with HTML `<img src=*.svg>` could produce a
  broken reference.

### Why
- The support matrix is the single most useful onboarding document for authors and
  was entirely missing; command references alone do not explain what renders.

### What worked
- `gofmt`/`go build ./...` clean.
- `go test ./pkg/mdpdf/... ./cmd/remarquee/... ./cmd/remarquee/cmds/upload/...` pass.
- `go run ./cmd/remarquee help markdown-format-support` prints the new guide.

### What didn't work
- The top-level `remarquee help` listing groups by section and does not print topic
  slugs, so the new doc is found by slug (`help markdown-format-support`), not by
  scanning the default listing. Acceptable for Glazed help.

### What I learned
- The in-repo help docs live in `pkg/doc/**` and are embedded via
  `pkg/doc/doc.go`; adding a file under `topics/` or `upload/` is enough to publish
  it to the CLI help system.

### What was tricky to build
- Documenting SVG accurately required reflecting the implementation asymmetry:
  referenced SVG is converted by pandoc, while inline/HTML SVG is converted by
  remarquee. The doc states both and the degradation paths.

### What warrants a second pair of eyes
- The support matrix's "unsupported" claims should be sanity-checked against real
  pandoc behavior; they are based on the LaTeX writer's HTML handling and the
  list-depth flattening in `preprocess.go`.

### What should be done in the future
- Consider adding a golden test that asserts the help system loads the new slug, so
  the doc cannot silently become unregistered.

### Code review instructions
- Read `pkg/doc/topics/markdown-format-support.md`.
- Run `go run ./cmd/remarquee help markdown-format-support`.

### Technical details
- New slug: `markdown-format-support`.
- Code fix: `pkg/mdpdf/pandoc.go` and `pkg/mdpdf/bundle.go` gate `ResolveHTMLImages`
  on image resolution.

## Step 12: Glazed help frontmatter conformance

The user asked whether the new help page used the Glazed help entry format. It did
not fully: I had mirrored the repo's topic frontmatter but omitted `ShowPerDefault`
and `SectionType`. Glazed defaults a missing `SectionType` to `SectionGeneralTopic`,
so the page loaded and was queryable, but it was not conformant and lacked the
`Commands`/`Flags` facet tags the format supports. I also had a top-level `#`
heading, which the authoring guidance says not to add because Glazed renders the
title from frontmatter.

### Prompt Context
**User prompt (verbatim):** "did you use the glazed help entry format?"
**Assistant interpretation:** audit frontmatter against the canonical Glazed spec.
**Inferred user intent:** conformant, complete help metadata.
**Commit (docs):** see below (Step 12 commit).

### What I did
- Checked the authoritative spec via `glaze help writing-help-entries` and the
  Glazed source (`pkg/help/help.go`, `LoadSectionFromMarkdown`), which confirms
  `SectionType` defaults to `SectionGeneralTopic` and reads `Topics`, `Commands`,
  `Flags`, `IsTopLevel`, `IsTemplate`, `ShowPerDefault`, `Order`.
- Added `ShowPerDefault: true` and `SectionType: GeneralTopic` to
  `pkg/doc/topics/markdown-format-support.md`.
- Added `Commands:` and `Flags:` facet lists.
- Removed the top-level `#` heading; the title now comes from frontmatter.
- Replaced the bullet troubleshooting list with a Problem/Cause/Solution table and
  added a `See Also` section, per the authoring guidance.

### Why
- Without `SectionType`, ranking/grouping behavior relies on an implicit default;
  explicit metadata is required for a well-formed entry and future filtering.

### What worked
- `go build ./...` clean; `go run ./cmd/remarquee help markdown-format-support`
  renders with the frontmatter title and no duplicate H1.

### What didn't work
- No failures observed. Note that several pre-existing `pkg/doc/upload/*.md` pages
  are non-conformant (they use `DocType`/`Slug` only, no `SectionType`); out of
  scope for this ticket.

### What I learned
- The canonical field set is `Title, Slug, Short, Topics, Commands, Flags,
  IsTopLevel, IsTemplate, ShowPerDefault, SectionType` (plus optional `Order`).

### What was tricky to build
- N/A.

### What warrants a second pair of eyes
- Whether to also normalize the older `pkg/doc/upload/*` pages in a follow-up.

### What should be done in the future
- Optional follow-up: add a load-time validation test asserting every `pkg/doc`
  markdown declares a `SectionType` and unique slug.

### Code review instructions
- Inspect the frontmatter of `pkg/doc/topics/markdown-format-support.md`.
- Run `go run ./cmd/remarquee help markdown-format-support`.

### Technical details
- Reference: `glaze help writing-help-entries`;
  `glazed@v0.7.3/pkg/help/help.go` `LoadSectionFromMarkdown`.

## Step 13: Tablet validation document

The user installed the updated CLI and asked for a document with the different SVG
integration forms to validate on the tablet. I generated a self-explanatory
checklist document and its SVG assets, verified it locally (4 pages, figures A–I
present, J preserved as code), and uploaded it to the ticket-aware destination.

### Prompt Context
**User prompt (verbatim):** "I installed the updated remarquee, can you send a test
document with the different kinds of svg integrations so I can validate things on
the tablet?"
**Assistant interpretation:** render and upload an SVG-feature validation doc.
**Inferred user intent:** confirm real device rendering of every supported form.
**Commit (docs):** see below (Step 13 commit).

### What I did
- Created `/tmp/svg-validate` with `doc.md` and six labeled SVG assets.
- Covered: referenced (A), sized 40% (B), inline raw (C), nested inline (D), HTML
  `<img>` (E), data URI (F), viewBox-only (G), pattern+text (H), quoted `>` attr
  (I), and a fenced `<svg>` code block (J, expected as code).
- Local render: 4 pages; `pdftotext` shows figures 1–9 and inline images 1–3;
  pixel probe confirmed C/D/E/I rendered (green 5670, teal 6228, yellow 4080,
  purple 1061).
- Uploaded via `remarquee upload md --name "RMQ-0024 SVG Integration Test"
  --remote-dir /ai/2026/09/20/RMQ-0024`, getting
  `OK: uploaded RMQ-0024_SVG_Integration_Test.pdf -> /ai/2026/09/20/RMQ-0024`.
- Copied the doc and assets into the ticket `scripts/svg-validation/` for
  reproducibility.

### What worked
- All forms rendered in the local PDF; upload succeeded on the first attempt.

### What didn't work
- No failures observed.

### What was tricky to build
- Avoided an XML-invalid attribute (unescaped `<`) in the quoted-attribute test;
  used `data-note="a > b"`, which still exercises the tokenizer's `>` handling
  without breaking rsvg's XML parser.

### What warrants a second pair of eyes
- The tablet is grayscale, so distinct colors become gray levels; the in-diagram
  letter labels make identification unambiguous.

### What should be done in the future
- Optionally fold these fixtures into an automated golden test.

### Code review instructions
- See `ttmp/2026/09/20/RMQ-0024--svg-image-support-in-the-markdown-to-pdf-pipeline/scripts/svg-validation/`.

### Technical details
- Document name on device: `RMQ-0024 SVG Integration Test.pdf`.
- Destination: `/ai/2026/09/20/RMQ-0024`.

## Step 14: Pull request

The user confirmed the tablet validation looked good and asked for a pull request.
I pushed `task/add-svg-support` to `origin` and opened the PR against `main`,
matching the structure of the prior PR (#26): Summary, Implementation, Validation,
Scope and limitations, and a ticket reference.

### Prompt Context
**User prompt (verbatim):** "it's fine, i have color dislay. it looks good. Create a
pull request"
**Assistant interpretation:** open a PR for the RMQ-0024 work.
**Inferred user intent:** get the feature reviewed/merged.
**Commit (docs):** see below (PR-link docs commit).

### What I did
- Confirmed branch `task/add-svg-support` was 10 commits ahead of `origin/main`,
  0 behind, with a clean working tree.
- `git push -u origin task/add-svg-support` (ADMIN on the repo, direct push).
- `gh pr create --base main --head task/add-svg-support` with a structured body.
- Recorded the PR link in the ticket index.

### What worked
- PR #28 opened: https://github.com/go-go-golems/remarquee/pull/28
  (30 files, +4095/-11 at creation).
- Uploaded tablet validation document was confirmed readable on the device.

### What didn't work
- `gh pr checks` reports "no checks reported on the branch" and `mergeable` was
  UNKNOWN immediately after creation (GitHub still computing). Neither blocks
  review.

### What I learned
- The repository has no PR template in `.github/`; prior PRs (#26) use a
  Summary/Validation/Scope structure, which this PR follows.

### What was tricky to build
- N/A.

### What warrants a second pair of eyes
- The support matrix and `Scope and limitations` section in the PR body make
  explicit claims about unsupported HTML; reviewers should confirm they match the
  intended product boundary.

### What should be done in the future
- Address review comments; merge when CI/checks are available.
- Optional follow-ups: HTML `<img>` for non-SVG images; pre-convert referenced SVG
  when no converter is present; normalize older `pkg/doc/upload/*` help pages.

### Code review instructions
- Start from the PR description; then `pkg/mdpdf/svg.go` and `pkg/mdpdf/pandoc.go`
  ordering; then `cmd/remarquee/cmds/upload/svg_section.go`.
- Run `go test ./...`.

### Technical details
- PR: #28, https://github.com/go-go-golems/remarquee/pull/28
- Branch: `task/add-svg-support` -> `main`

## Step 15: Address PR #28 SVG review findings

The user requested that the issues on PR #28 be addressed. Codex identified three P2 correctness defects: generated SVG assets from `upload bundle --resolve-images=false` lived in the bundle builder's temporary directory rather than the later pandoc converter's directory; quote skipping in the SVG depth scanner treated an apostrophe in SVG text as an unterminated attribute quote; and verbatim HTML `img` attributes could produce malformed Markdown for paths containing spaces or alt text containing `]`.

The fixes preserve the original contract rather than changing flag semantics. Bundle-generated SVG PDFs and Mermaid PNGs now use absolute references into the still-live bundle temporary directory. SVG tag scanning is quote-aware only while scanning tags. HTML SVG rewriting now uses angle-bracket destinations and Pandoc-decoded HTML character references for structural Markdown characters in alt text. The latter required one correction during validation because a conventional `\[` escape is interpreted as display math by remarquee's `tex_math_single_backslash` pandoc input format.

### Prompt Context
**User prompt (verbatim):** "Address issues: https://github.com/go-go-golems/remarquee/pull/28"
**Assistant interpretation:** resolve all Codex review findings on PR #28, validate them, and update the branch for review.
**Inferred user intent:** remove flag-dependent SVG loss and malformed-Markdown cases before merging the SVG feature.
**Commit (code):** pending — review-fix checkpoint, to be committed after validation.

### What I did
- Read PR #28 review comments with `gh`; found three P2 comments at `pkg/mdpdf/bundle.go:69` and `pkg/mdpdf/svg.go:155,456`.
- Added `AbsolutePaths` to `SVGRendererConfig` and `MermaidRendererConfig`; `BuildBundleMarkdown` clones its per-input configs with that flag set so generated assets reference the builder's absolute `<tmp>/images/...` paths.
- Reworked `countTagDelta` to use `tagEnd` only for tag-like regions. Quotes in ordinary SVG text no longer suppress later `</svg>` detection, while quoted attributes in other tags remain protected.
- Extended `inlineImageRegex` for escaped alt characters and angle-bracket destinations. `ResolveImagePaths` now preserves angle-bracket form when re-emitting staged destinations containing whitespace or parentheses.
- Changed HTML SVG-image alt escaping from backslashes to `&#92;`, `&#91;`, and `&#93;`; `htmlImgToMarkdown` uses `<...>` destinations for source paths containing whitespace or parentheses.
- Added unit, bundle, and real pandoc/xelatex regression tests, including `TestSVGPipelineBundlePDF_NoResolveImages`.

### Why
- `ConvertMarkdownFileToPDF` intentionally creates a fresh pandoc working directory. In bundle mode, a relative `./images/...` reference to an asset generated by `BuildBundleMarkdown` cannot work when the final conversion intentionally skips image re-staging.
- Quote syntax belongs to markup, not text nodes. Skipping unmatched text apostrophes makes a valid compact SVG look unclosed.
- Backslash escaping is normally valid Markdown, but this repository enables `tex_math_single_backslash`; `\[` has a different, unsafe meaning in the selected pandoc reader. Numeric HTML entities preserve the intended caption characters without entering the math parser.

### What worked
- `go test ./pkg/mdpdf/...` passes, including the real `rsvg-convert` + pandoc + xelatex regression test for bundle conversion with `ResolveImages=false`.
- `go test ./...` passes and `golangci-lint run ./pkg/mdpdf/... ./cmd/...` reports `0 issues.`
- Real CLI evidence:
  - `upload bundle doc1.md --resolve-images=false --pdf-only` produced `apostrophe.pdf` with 76 red pixels from a compact `<svg><text>don't</text>...</svg>` input.
  - Default-resolution `upload bundle doc2.md --pdf-only` produced `spaced-html.pdf` with 11,025 blue pixels from `<img src="./architecture diagram.svg" alt="arch [v2] diagram" width="200">`.
  - `pdftotext` reports `Figure 1: arch [v2] diagram`, proving the entity-escaped alt text decoded correctly.

### What didn't work
- The initial backslash-based alt escaping failed under the real CLI:
  ```text
  Error: pandoc failed: Error producing PDF.
  ! LaTeX Error: Bad math environment delimiter.
  l.97 \caption{arch \[v2\] diagram}
  : exit status 43
  ```
  The failure was reproduced with `upload bundle doc2.md --pdf-only`. Replaced `\[` and `\]` with `&#91;` and `&#93;`, which pandoc decodes before producing the caption.

### What I learned
- `tex_math_single_backslash` changes the safety of otherwise conventional Markdown escaping. Any generated Markdown must be validated with remarquee's exact `DefaultFromFormat`, not a default pandoc invocation.
- Bundle-generated Mermaid assets had the same temporary-directory lifetime boundary as inline SVG assets. The shared `AbsolutePaths` mechanism fixes both generated asset types without changing how user-authored relative images behave under `--resolve-images=false`.

### What was tricky to build
- The HTML-rewrite fix crosses three parsers: HTML attribute parsing, remarquee's regex-based Markdown image staging, and pandoc's Markdown/LaTeX conversion. A destination can be valid for one but not the next. Angle-bracket destinations keep spaces and parentheses intact through the staging pass; entities keep brackets out of Markdown structural parsing and out of the math delimiter parser.

### What warrants a second pair of eyes
- Absolute generated asset references intentionally exist only during conversion, while the bundle temporary directory remains alive. Review the `writeBundlePDF` lifetime (`defer os.RemoveAll(tmpDir)` occurs after `ConvertMarkdownFileToPDF` returns) if that flow is refactored.
- Confirm that making Mermaid use the same absolute-reference correction is acceptable; it is the identical pre-existing bundle failure mode, but expands this review-fix commit beyond SVG alone.

### What should be done in the future
- Optional: add tests for XML comments, CDATA, and `foreignObject` constructs containing tag-like text if supporting arbitrary hand-authored inline SVG becomes a product requirement.
- Optional: consolidate generated-asset lifetime handling behind a single internal asset-reference type instead of per-renderer config flags.

### Code review instructions
- Start at `BuildBundleMarkdown` and follow `writeBundlePDF` into `ConvertMarkdownFileToPDF` to verify the two temporary-directory lifetimes.
- Review `countTagDelta`, `tagEnd`, and the new apostrophe/quoted-other-tag cases in `pkg/mdpdf/svg_test.go`.
- Review `formatImageTarget` and `escapeMarkdownAltText`, then run:
  ```bash
  go test ./pkg/mdpdf/...
  go test ./...
  golangci-lint run ./pkg/mdpdf/... ./cmd/...
  ```

### Technical details
- PR and review comments: https://github.com/go-go-golems/remarquee/pull/28
- Absolute bundle regression: `TestSVGPipelineBundlePDF_NoResolveImages`.
- HTML regression: `TestResolveHTMLImages_EscapesAltAndDelimitsDestinations` and `TestResolveHTMLImages_SpaceInSourceSurvivesStaging`.
- Scanner regression: `TestResolveInlineSVGBlocks_ApostropheInText`.
