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
