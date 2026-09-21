---
Title: 'SVG image support: design and implementation guide for a new intern'
Ticket: RMQ-0024
Status: active
Topics:
    - markdown
    - pdf
    - mdpdf
    - image-embed
    - pandoc
    - xelatex
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/remarquee/cmds/upload/mermaid_section.go
      Note: CLI flag wiring for new --svg flags
    - Path: repo://pkg/mdpdf/bundle.go
      Note: Bundle path that must get parity
    - Path: repo://pkg/mdpdf/images.go
      Note: Image regexes and ResolveImagePaths (referenced image staging)
    - Path: repo://pkg/mdpdf/literal_regions.go
      Note: Literal-region guard used by the inline <svg> scanner
    - Path: repo://pkg/mdpdf/mermaid.go
      Note: Design precedent for external-tool conversion and graceful degradation
    - Path: repo://pkg/mdpdf/pandoc.go
      Note: Pandoc options, args, and ConvertMarkdownFileToPDF (primary integration point)
ExternalSources: []
Summary: Evidence-based design and implementation guide for robust SVG embedding in remarquee's Markdown-to-PDF pipeline, including inline raw <svg> extraction, HTML <img> handling, converter fallback, and sizing policy.
LastUpdated: 2026-09-20T18:05:00-04:00
WhatFor: Onboarding a new engineer onto remarquee's mdpdf pipeline and giving them a complete, testable implementation plan for the remaining SVG gaps.
WhenToUse: Before starting implementation of RMQ-0024, or when you need to understand how Markdown becomes a reMarkable PDF.
---

# SVG image support: design and implementation guide for a new intern

## 0. Executive summary

This document is a complete, self-contained onboarding + design + implementation
guide for adding first-class **SVG image support** to remarquee's Markdown → PDF →
reMarkable pipeline.

It is written for a new intern who has never seen this repository. It starts from
"what is remarquee and how does a `.md` file become a drawing on an e-ink tablet"
and ends with a concrete, phased implementation plan and the exact test fixtures
to build.

> **Important correction to the original premise (evidence-based).**
> The work was originally scoped as "pandoc/xelatex cannot render SVG, so add an
> SVG→PDF converter step." **That premise is only partially true.** Empirical
> testing on the target machine (pandoc 3.1.3, rsvg-convert, TeX Live) shows that
> **referenced `.svg` files and `data:image/svg+xml` URIs already render today**,
> because pandoc itself shells out to `rsvg-convert` and includes the resulting
> vector PDF. See §5.2 for the full verified behavior matrix and §13 Appendix A
> for the exact reproduction commands.
>
> What is *actually* missing is:
>
> 1. **Inline raw `<svg>…</svg>` blocks** — silently dropped by pandoc's LaTeX
>    writer. This is the main feature to implement.
> 2. **HTML `<img src="….svg">` tags** — also silently dropped.
> 3. **Graceful degradation when `rsvg-convert` is missing** — currently a hard
>    XeLaTeX failure with a confusing error, instead of a warning that leaves the
>    source intact (the way Mermaid already degrades).
> 4. **Deterministic sizing** — SVGs without explicit dimensions can render at
>    surprising sizes; we want a documented, configurable policy.
> 5. **Bundle-mode parity, CLI flags, and tests** for all of the above.

Everything downstream of this box assumes that corrected scope.

> **Implementation revision (2026-09-20, post-Phase-4).** The feature is now
> implemented; a few details below were refined during implementation and are
> recorded here so the guide stays accurate:
> 1. **Ordering (supersedes §6.2/§6.5):** `ResolveInlineSVGBlocks` runs **after**
>    `ResolveImagePaths`, not before. It writes finished `.pdf` assets directly
>    into `<tmp>/images/`, and `ResolveImagePaths` resolves relative paths against
>    the *source* directory, so pre-image ordering would break things.
>    `ResolveHTMLImages` still runs **before** `ResolveImagePaths`.
> 2. **Inline conversion is direct:** extracted inline `<svg>` is converted to a
>    vector `.pdf` by our own converter (not left as `.svg` for pandoc). This makes
>    the no-converter degradation meaningful.
> 3. **API:** `ResolveHTMLImages(body, cfg)` (no `ctx`/`sourceDir`/`tmpDir`), since
>    conversion is delegated to pandoc after the rewrite.
> 4. **No `MaxWidth`:** the field was dropped as unused; only `DefaultWidth` ships.
> 5. **CLI flags:** `--svg`, `--svg-converter`, `--svg-default-width` in an
>    "SVG flags" Glazed section.

---

## 1. Document scope and audience

**Audience.**
- A new intern with working Go knowledge but no prior familiarity with remarquee,
  pandoc, XeLaTeX, or the reMarkable cloud.
- A reviewer who wants to sanity-check the design before implementation.

**Scope.**
- The `pkg/mdpdf` package: how it preprocesses Markdown and drives pandoc.
- The `cmd/remarquee/cmds/upload` commands that share that package.
- The SVG-specific gaps and their design.
- A phased, testable implementation plan.

**Non-goals.**
- Reimplementing a Markdown parser. We use targeted regex/scanning passes, matching
  the existing codebase style.
- Changing the reMarkable upload protocol, `.rmdoc` rendering, or OCR.
- Supporting exotic SVG features beyond what `rsvg-convert`/Inkscape support.

**How to read this document.**
- If you are brand new: read Part 1, then Part 2, then Part 3.
- If you just want to implement: read §3.4 (data flow), Part 4 (design), Part 6
  (plan), and Part 9 (gotchas). Keep Part 2 open as reference.
- Every claim about current behavior is backed by a command in Appendix A.

---

## 2. Vocabulary (read this first)

| Term | Meaning in this document |
|---|---|
| **remarquee** | The Go CLI in this repository; a unified toolkit for reMarkable workflows. |
| **rmapi** | Third-party library/CLI glue remarquee uses to talk to the reMarkable cloud. |
| **pandoc** | Universal document converter. remarquee uses `markdown → latex → PDF`. |
| **XeLaTeX** | The TeX engine pandoc uses to typeset the PDF (`--pdf-engine=xelatex`). |
| **mdpdf** | `pkg/mdpdf`, remarquee's Markdown-to-PDF preprocessing + pandoc driver. |
| **rsvg-convert** | A small, fast SVG→PDF/PNG converter from librsvg. Pandoc uses it. |
| **Inkscape** | A full SVG editor; its CLI can export SVG→PDF. Used by the `svg` LaTeX package. |
| **mmdc** | Mermaid CLI; renders ```` ```mermaid ```` blocks to PNG. |
| **bundle** | Multiple Markdown files concatenated into one PDF with a table of contents. |
| **raw block** | Markdown/HTML content passed through without interpretation (e.g. `<svg>`). |
| **CAS / temp dir** | The per-conversion `os.MkdirTemp` directory where assets are staged. |

---

## 3. Part 1 — The system end to end

### 3.1 What remarquee does

remarquee is a single Go binary with subcommands grouped by resource:

```
remarquee
├── cloud      # account, ls, get, rm, search  (talks to reMarkable cloud)
├── device     # USB / device-facing helpers
├── upload     # md, bundle, src  (Markdown/source → PDF → cloud)
├── ocr        # OCR of rendered documents
├── rmdoc      # parse / render reMarkable .rmdoc containers
├── rmdsl      # reMarkable DSL tooling
└── status     # environment / auth status
```

The relevant subsystem for this ticket is **`upload`**, and the library package
that does the heavy lifting is **`pkg/mdpdf`**.

The crucial mental model:

```
            ┌─────────────────────────────────────────────┐
            │  reMarkable NEVER receives your Markdown.   │
            │  It receives a PDF produced by pandoc.      │
            │  Everything about "rendering Markdown" is   │
            │  really "producing a good PDF".             │
            └─────────────────────────────────────────────┘
```

### 3.2 The four commands that share the pipeline

All of these funnel through `pkg/mdpdf`:

| Command | Purpose | Entry file |
|---|---|---|
| `remarquee upload md` | One or more `.md` files → one PDF each | `cmd/remarquee/cmds/upload/md.go` |
| `remarquee upload bundle` | Many `.md` files → one PDF with ToC | `cmd/remarquee/cmds/upload/bundle.go` |
| `remarquee upload sync` | Incremental version of `md` (skip unchanged) | `cmd/remarquee/cmds/upload/sync.go` |
| `remarquee upload src` | Source files → highlighted PDFs | `cmd/remarquee/cmds/upload/src.go` |

`upload md` also runs conversions in parallel via
`cmd/remarquee/cmds/upload/conversion_workers.go`.

**Key point for an intern:** if you add a preprocessing step, you must wire it into
**both** the single-file path (`mdpdf.ConvertMarkdownFileToPDF`) **and** the bundle
path (`mdpdf.BuildBundleMarkdown`). They are separate code paths that both call
image/mermaid preprocessing independently. Forgetting bundle mode is the single
most common integration bug in this package.

### 3.3 Package map

| File | Responsibility |
|---|---|
| `pkg/mdpdf/pandoc.go` | `PandocOptions`, `DefaultPandocOptions`, `buildPandocArgs`, `ConvertMarkdownFileToPDF`, the built-in LaTeX header. |
| `pkg/mdpdf/images.go` | Markdown image regexes; `ResolveImagePaths` / `ResolveImagePathsWithPrefix`; copies local assets into the temp dir and rewrites paths. |
| `pkg/mdpdf/mermaid.go` | `MermaidRendererConfig`, `RenderMermaidBlocks`, `renderMermaidToPNG`, `resolveMmdcPath`. **This is the design precedent for any external-tool asset conversion.** |
| `pkg/mdpdf/mermaid_config.go` | Small helper to clone a mermaid config with a per-input filename prefix (bundle mode). |
| `pkg/mdpdf/bundle.go` | `BundleInput`, `BuildBundleMarkdown`: per-file preprocessing, section headings, page breaks. |
| `pkg/mdpdf/preprocess.go` | `StripYAMLFrontmatter`, `NormalizeListSpacing`, `FlattenDeepLists`. |
| `pkg/mdpdf/literal_regions.go` | `literalLines`: protects fenced code and display math from line-based rewrites. **Read this before writing any new text pass.** |
| `pkg/mdpdf/layout.go` | `default` / `editor` layout presets (geometry + LaTeX header). |
| `pkg/mdpdf/math_pdf_test.go`, `images_test.go`, etc. | Unit tests; integration/golden tests render real PDFs. |
| `pkg/mdpdf/testdata/` | Fixtures (currently `pixel.png`). |
| `cmd/remarquee/cmds/upload/mermaid_section.go` | Glazed section for mermaid flags **and** `addResolveImagesFlag`. Your new flags go near here. |

### 3.4 End-to-end data flow

This is the diagram to memorize. Annotations mark where SVG work must hook in.

```
  .md file(s)
      │
      ▼
┌───────────────────────────────────────────────────────────────────────┐
│ cmd/remarquee/cmds/upload/{md,bundle,sync,src}.go                     │
│  - parse flags, build mdpdf.PandocOptions                             │
│  - bundle: BuildBundleMarkdown(...)                                   │
│  - single: ConvertMarkdownFileToPDF(...)                              │
└───────────────────────────────────────────────────────────────────────┘
      │
      ▼
┌───────────────────────────────────────────────────────────────────────┐
│ pkg/mdpdf — PREPROCESSING (in-memory string → string)                 │
│                                                                       │
│  1. StripYAMLFrontmatter         (preprocess.go)                      │
│  2. ResolveImagePaths            (images.go)   ── copies PNG/JPG/SVG  │
│                                                  ── and rewrites path │
│  3. RenderMermaidBlocks          (mermaid.go)  ── mmdc → PNG          │
│  4. NormalizeListSpacing         (preprocess.go)                      │
│  5. FlattenDeepLists             (preprocess.go)                      │
│                                                                       │
│  >>> NEW HOOKS FOR THIS TICKET:                                       │
│  2b. ResolveInlineSVGBlocks      (svg.go)   ── extract <svg> → .svg   │
│  2c. ResolveHTMLImgs             (svg.go)   ── <img src> → markdown   │
│  2d. ConvertSVGAssets            (svg.go)   ── ensure vector PDF      │
│                                                                       │
│  output: preprocessed Markdown written to <tmp>/input.md              │
│          assets staged in        <tmp>/images/                        │
└───────────────────────────────────────────────────────────────────────┘
      │   (cmd.Dir = <tmp>)
      ▼
┌───────────────────────────────────────────────────────────────────────┐
│ pandoc                                                                │
│  --from=markdown-yaml_metadata_block+tex_math_single_backslash        │
│  --pdf-engine=xelatex --standalone -H header.tex ...                  │
│                                                                       │
│  For a referenced "./images/foo.svg":                                 │
│     pandoc runs: rsvg-convert -f pdf -a --dpi-x 96 --dpi-y 96 \       │
│                       -o <hash>.pdf ./images/foo.svg                  │
│     then emits:  \includegraphics{<hash>.pdf}                         │
│  (Verified — see §5.2 and Appendix A.)                                │
└───────────────────────────────────────────────────────────────────────┘
      │
      ▼
   PDF  ──►  upload  ──►  reMarkable cloud  ──►  device
```

---

## 4. Part 2 — Foundational mechanisms you must understand

### 4.1 Frontmatter stripping (`preprocess.go`)

Docmgr documents start with YAML frontmatter:

```markdown
---
Title: My doc
Ticket: RMQ-0024
---

# Real content starts here
```

`StripYAMLFrontmatter` removes the leading `---` block so (a) pandoc does not try
to parse possibly-invalid YAML, and (b) metadata does not leak into the PDF.

Implementation note: it is deliberately simple — it only strips an *unindented*
`---` at byte 0 and finds the next line that is exactly `---`. It does **not**
parse YAML. If you add SVG handling, run it *after* this step, because frontmatter
can contain `---` and quoted strings that would confuse a naive scanner.

### 4.2 Image path resolution (`images.go`) — read carefully

This is the code your SVG work will extend, so understand it precisely.

**Regexes.**

```go
// ![alt](destination [title])
var inlineImageRegex = regexp.MustCompile(`!\[([^\]]*)\]\(([^)\n]+)\)`)

// ![alt][id]  and  ![alt][]
var referenceImageRegex = regexp.MustCompile(`!\[([^\]]*)\]\[([^\]]*)\]`)

// [id]: destination [title]   (at up to 3 leading spaces)
var referenceDefinitionRegex = regexp.MustCompile(`(?m)^([ \t]{0,3})\[([^\]]+)\]:[ \t]*(<[^>]+>|\S+)([^\n]*)$`)
```

**What `ResolveImagePathsWithPrefix` does, in order:**

1. Creates `<tmp>/images/`.
2. For every inline `![alt](target)`:
   - `splitInlineImageTarget` separates the path from an optional `"title"`.
   - `copyImageToTemp` is called.
   - If copied, the Markdown path is rewritten to `./images/<name>`.
3. Collects reference-style image labels (`![alt][id]`), then rewrites matching
   `[id]: destination` definitions the same way.
4. Returns the rewritten body.

**`copyImageToTemp` rules (these matter for SVG):**

```go
func copyImageToTemp(imgPath, sourceDir, imagesDir, filenamePrefix string,
                     usedNames map[string]bool) (string, bool) {
    if isURL(imgPath) || filepath.IsAbs(imgPath) {
        return "", false          // ← URLs and absolute paths are NOT copied
    }
    resolved := filepath.Clean(filepath.Join(sourceDir, imgPath))
    info, err := os.Stat(resolved)
    if err != nil || info.IsDir() {
        return "", false          // ← missing files silently pass through
    }
    // ... unique-filename logic, then copyFile ...
    return "./images/" + destName, true
}
```

**Consequences you must internalize:**

- The function is **format-agnostic**: it copies `.svg` exactly like `.png`. So SVG
  files already reach the temp dir. The question is only whether pandoc can render
  them.
- `isURL` treats `http://`, `https://`, and `data:` as non-local. So
  `data:image/svg+xml;...` is **not** staged by us — pandoc handles it inline
  (and, as we verified, it works).
- Absolute paths (`/abs/foo.svg`) are not copied and not rewritten. Pandoc would
  then see the absolute path. Keep this in mind for the inline-extraction pass: we
  will *write* absolute-ish paths, so use the same `./images/...` convention to
  stay consistent.
- Missing images pass through unchanged, producing a broken reference rather than
  an error. Match this lenient philosophy for SVG: warn, don't crash.

**Bundle prefixing.** `filenamePrefix` (e.g. `bundle-002-`) is prepended to copied
filenames so that two input files with `logo.svg` do not collide in the shared
`<tmp>/images/` directory. Any new SVG asset naming must honor this prefix.

### 4.3 Mermaid rendering (`mermaid.go`) — the precedent to copy

Mermaid blocks cannot be rendered by pandoc, so remarquee does the conversion
itself before pandoc runs:

```
```mermaid
graph TD; A-->B
```
        │
        ▼  RenderMermaidBlocks (regex ReplaceAllStringFunc)
   mmdc -i diagram.mmd -o images/mermaid-001.png -s 2 -b white -t default
        │
        ▼
![mermaid diagram 1](./images/mermaid-001.png)
```

Key engineering decisions in `mermaid.go` that you should replicate for SVG:

1. **Graceful absence.** `resolveMmdcPath` returns an error if `mmdc` is missing,
   but `RenderMermaidBlocks` treats that as *non-fatal*: it returns the body
   unchanged. The document still renders; the diagram appears as a code block.
2. **Per-block isolation.** If one diagram fails, it logs
   `WARNING: failed to render Mermaid block N` and leaves that block as text. One
   bad diagram never breaks the document.
3. **Config struct + defaults.** `MermaidRendererConfig` is a value type with a
   `DefaultMermaidRendererConfig()` constructor, injected via `PandocOptions.Mermaid`.
4. **Prefix support.** `mermaidConfigWithImagePrefix` clones + prefixes for bundle
   mode.
5. **File-based temp input.** The source is written to a temp `.mmd` file because
   the external tool wants a path.

This is the template for `pkg/mdpdf/svg.go`.

### 4.4 Literal-region protection (`literal_regions.go`)

`literalLines(lines []string) []bool` returns a parallel boolean slice marking
lines that are inside fenced code blocks or display math. The list-normalization
passes skip protected lines so they do not corrupt code.

**Why you must care:** most naive "extract `<svg>` blocks" implementations will
also match SVG *inside a fenced code block* that is teaching someone about SVG.
That would be wrong. Reuse `literalLines` (or equivalent state tracking) so fenced
code containing `<svg>` is preserved verbatim.

Helper functions available: `literalLines`, `fencePrefix`, `hasUnescapedDelimiter`.
Note `literalLines` is currently package-private but usable within `pkg/mdpdf`.

### 4.5 Pandoc invocation and LaTeX header (`pandoc.go`)

`buildPandocArgs` assembles:

```
--from=markdown-yaml_metadata_block+tex_math_single_backslash
<input> -o <output>
--pdf-engine=xelatex
--standalone
-V mainfont=DejaVu Sans
-V monofont=DejaVu Sans Mono
-V geometry:margin=1in
-H <header.tex>          (built-in defaultLatexHeader unless overridden)
[--toc] [--toc-depth=N] [--highlight-style=...] [--listings]
```

**Critical facts:**

- `--from` disables `yaml_metadata_block` (so `---` thematic breaks are not eaten)
  and enables `tex_math_single_backslash` (so `\(..\)` math parses). Do not change
  it casually; see `RMQ-0022`.
- **There is no `--shell-escape`.** This is why the LaTeX `svg` package route fails:
  `\includesvg` requires shell escape + Inkscape. See §5.4.
- `cmd.Dir = tmpDir`, so relative assets like `./images/foo.svg` resolve inside the
  temp dir.
- The default header loads math packages (`stmaryrd`, `centernot`, `mathtools`,
  `amscd`) and list spacing tweaks. It does **not** load `graphicx` or the `svg`
  package; pandoc's template adds `graphicx` (and `svg` only in its LaTeX-writer
  fallback path).

### 4.6 Bundle mode (`bundle.go`)

`BuildBundleMarkdown(ctx, inputs, tmpDir, mermaidCfg, resolveImages)` loops over
input files and for each one:

1. Reads the file, `StripYAMLFrontmatter`.
2. If `resolveImages`, calls `ResolveImagePathsWithPrefix` with prefix
   `bundle-00N-`.
3. Calls `RenderMermaidBlocks` with the prefixed config.
4. `NormalizeListSpacing`.
5. Emits `# <title>\n\n<body>` and a `\newpage` between documents.

The result is then fed to `ConvertMarkdownFileToPDF`, which runs its *own*
`ResolveImagePaths` on the already-rewritten paths (they point at `./images/...`,
which exists in `tmpDir`). **This double-resolution is intentional and works
because rewritten paths are idempotent.** Any new inline-SVG pass must also be
idempotent when run a second time on already-substituted Markdown.

---

## 5. Part 3 — The SVG problem, precisely

### 5.1 The SVG input forms

Authors embed SVG in Markdown in several different ways. They behave very
differently through the pipeline.

**Form A — Markdown image reference (file on disk):**

```markdown
![diagram](./assets/arch.svg)
![diagram](./assets/arch.svg){width=60%}
```

**Form B — Reference-style Markdown image:**

```markdown
![diagram][arch]

[arch]: ./assets/arch.svg
```

**Form C — Inline raw SVG block:**

```markdown
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">
  <circle cx="50" cy="50" r="40" fill="steelblue"/>
</svg>
```

**Form D — HTML `<img>` tag:**

```markdown
<img src="./assets/arch.svg" width="300">
```

**Form E — Data URI:**

```markdown
![diagram](data:image/svg+xml;base64,PHN2ZyB4bWxucz0i...)
```

**Form F — Inside a fenced code block (must NOT be touched):**

````markdown
```xml
<svg xmlns="http://www.w3.org/2000/svg">...</svg>
```
````

### 5.2 Verified behavior matrix (the heart of this document)

Tested on the target machine with pandoc **3.1.3**, `rsvg-convert` present, TeX
Live XeLaTeX, using the actual installed `remarquee` binary via
`remarquee upload md --pdf-only`. Reproduction commands are in Appendix A.

| # | Form | Today's result | Mechanism / evidence |
|---|---|---|---|
| A | `![x](a.svg)` | ✅ **Renders** | Pandoc calls `rsvg-convert -f pdf -a --dpi-x 96 --dpi-y 96`, then `\includegraphics{<hash>.pdf}`. Blue pixels present in output. |
| A | `![x](a.svg){width=40%}` | ✅ Renders, sized | Width is applied by `\includegraphics[width=...]`. |
| B | reference-style | ✅ Renders | `images.go` resolves the definition before pandoc sees it. |
| C | raw `<svg>…</svg>` | ❌ **Silently dropped** | Pandoc markdown reader emits a `RawBlock (Format "html")`; the LaTeX writer discards it. Verified: zero red pixels, only surrounding text survives. |
| D | `<img src="a.svg">` | ❌ **Silently dropped** | `inlineImageRegex` does not match HTML `<img>`; the raw HTML is discarded by the LaTeX writer. Verified: zero blue pixels. |
| E | `data:image/svg+xml;base64,…` | ✅ Renders | `isURL` skips staging; pandoc decodes the data URI to a temp file and converts it. Verified: blue pixels present. |
| F | SVG in fenced code | ✅ Preserved as code | `literalLines` protects it; pandoc emits a listing, not an image. |

**Additional verified behaviors:**

- **Default sizing = SVG intrinsic size.** A `120×60` SVG with explicit
  `width`/`height` rendered at ~1.25 in wide (96 dpi assumption). A `viewBox`-only
  `400×200` SVG rendered at ~70% of text width by default. Authors who expect "fill
  the page" may be surprised.
- **Missing `rsvg-convert` = hard failure.** With a broken `rsvg-convert` on
  `PATH`, pandoc warns `Could not convert image ...: conversion from SVG failed`
  and falls back to `\includesvg`, after which XeLaTeX dies with:

  ```
  ! Package svg Error: File `logo_svg-tex.pdf' is missing.
  ```

  There is **no graceful document-level degradation**. This is the concrete
  robustness bug to fix.

### 5.3 Why inline raw `<svg>` fails

Pandoc's Markdown reader recognizes raw HTML. For a `<svg>` block it produces
(conceptually):

```
RawBlock (Format "html") "<svg ...>...</svg>"
```

The LaTeX writer only emits raw content for formats it understands
(`latex`, `tex`). `html` raw content is omitted. So the LaTeX output contains
nothing between the surrounding paragraphs. There is no warning. This is the
"silently dropped" failure mode, and it is the primary reason for this ticket.

### 5.4 Why we do NOT use the LaTeX `svg` package

One tempting fix: add `\usepackage{svg}` and `\includesvg{...}`, and pass
`--shell-escape`. We verified this works **only** with shell escape:

```
$ xelatex t.tex                       # no -shell-escape
! Package svg Error: File `logo_svg-tex.pdf' is missing.

$ xelatex -shell-escape t.tex         # works, invokes inkscape
('inkscape' -V ) (./svg-inkscape/logo_svg-tex.pdf_tex) [1]
```

We reject this approach because:

- It requires `--shell-escape`, a security-relevant flag that allows arbitrary
  command execution from a document.
- It depends on Inkscape (slow, heavyweight) rather than the fast `rsvg-convert`.
- It produces an `svg-inkscape/` sidecar directory with `.pdf_tex` files.
- Pandoc's own PDF path already uses `rsvg-convert` and avoids all of this.

**We keep `--shell-escape` OFF** and do conversion ourselves, exactly as the
Mermaid path does.

### 5.5 Sizing semantics

Because pandoc includes the converted SVG as `\includegraphics`, sizing follows
LaTeX rules:

- No `{width=...}`: uses the PDF's natural size, derived from the SVG's
  `width`/`height`/`viewBox` at 96 dpi.
- `{width=60%}`: percentage of the current text width.
- `{width=8cm}` / `{width=400px}`: absolute dimensions (pandoc translates units).

Design implication: for inline SVG blocks (Form C) we have no Markdown image
syntax to carry a `{width=...}` attribute. We must choose a policy (default width,
configurable max width) and document it.

---

## 6. Part 4 — Design

### 6.1 Goals and non-goals

**Goals.**
1. Inline raw `<svg>` blocks render as vector images.
2. HTML `<img src="*.svg">` tags render (converted to Markdown image syntax).
3. Referenced SVG keeps working, but degrades gracefully when the converter is
   unavailable: warn, leave source intact, never produce a hard XeLaTeX failure.
4. Conversion is deterministic, offline, and does not require `--shell-escape`.
5. Behavior is identical in single-file and bundle mode.
6. All of it is covered by unit tests and at least one real PDF integration test.

**Non-goals.**
- Full HTML parsing. We handle `<svg>` and `<img>` with targeted scanning.
- Arbitrary HTML/CSS support.
- Remote SVG fetching (network URLs remain pandoc's concern).

### 6.2 Architecture overview

```
                        pkg/mdpdf/svg.go  (NEW)
    ┌───────────────────────────────────────────────────────────────┐
    │                                                               │
    │  SVGRendererConfig                                            │
    │    Enabled, ConverterPath, TargetFormat(always vector PDF),   │
    │    DefaultWidth, ImagePrefix                              │
    │                                                               │
    │  ResolveInlineSVGBlocks(ctx, body, tmpDir, cfg) (string,err)  │
    │    - scan lines, honor literalLines()                         │
    │    - extract <svg ...> ... </svg>                             │
    │    - write images/svg-XXXX.svg                                │
    │    - replace block with ![svg N](./images/svg-XXXX.svg){...}  │
    │                                                               │
    │  ResolveHTMLImages(body, cfg)                                 │
    │    - scan for <img src="...svg"> ...                           │
    │    - extract src + width/height attrs                          │
    │    - emit Markdown image, then normal ResolveImagePaths runs  │
    │                                                               │
    │  EnsureSVGConvertible(...) / converter availability probe     │
    │    - resolve rsvg-convert → inkscape → (optional) error       │
    │    - if none: warn once, leave SVG references untouched       │
    └───────────────────────────────────────────────────────────────┘
```

Data flow with the new pass inserted:

```
body
  │ StripYAMLFrontmatter
  ▼
body
  │ ResolveInlineSVGBlocks   ── writes inline SVGs to images/, rewrites body
  ▼
body
  │ ResolveHTMLImages        ── rewrites <img src> to ![](...)  [before image copy]
  ▼
body
  │ ResolveImagePaths        ── copies referenced .svg/.png into images/
  ▼
body
  │ RenderMermaidBlocks
  ▼
body
  │ NormalizeListSpacing / FlattenDeepLists
  ▼
input.md  +  images/*.svg
  │
  ▼ pandoc (rsvg-convert under the hood)
 PDF
```

Ordering matters: `ResolveHTMLImages` must run **before** `ResolveImagePaths`,
because it turns HTML into Markdown image syntax that `ResolveImagePaths` can then
stage. `ResolveInlineSVGBlocks` runs **after** `ResolveImagePaths`, because it
writes finished assets directly into `<tmp>/images/` and `ResolveImagePaths`
resolves relative paths against the *source* directory, not the temp dir.

### 6.3 New configuration type

```go
// pkg/mdpdf/svg.go

// SVGRendererConfig controls how SVG content is detected, extracted and made
// renderable for pandoc/XeLaTeX.
type SVGRendererConfig struct {
    // Enabled turns the whole feature on/off. Default: true.
    Enabled bool

    // ConverterPath overrides converter auto-detection.
    // Empty = search $PATH for the configured order.
    ConverterPath string

    // ConverterOrder is the preference list. Default:
    // ["rsvg-convert", "inkscape"].
    ConverterOrder []string

    // DefaultWidth is applied to converted SVG images that have no explicit
    // size, when we synthesize Markdown. Examples: "70%", "12cm", "".
    // Empty means "let pandoc/LaTeX use natural size".
    DefaultWidth string

    // ImagePrefix prevents filename collisions in bundle mode.
    ImagePrefix string

    // WarnWriter receives non-fatal warnings. Default: os.Stderr.
    WarnWriter io.Writer
}

func DefaultSVGRendererConfig() SVGRendererConfig {
    return SVGRendererConfig{
        Enabled:        true,
        ConverterOrder: []string{"rsvg-convert", "inkscape"},
        WarnWriter:     os.Stderr,
    }
}
```

Wire it into `PandocOptions` exactly like Mermaid:

```go
type PandocOptions struct {
    // ... existing fields ...
    Mermaid *MermaidRendererConfig
    SVG     *SVGRendererConfig   // NEW
}
```

### 6.4 Converter resolution

```go
// resolveSVGConverter finds a usable converter. Returns ("", nil) when the
// feature is enabled but no converter exists — callers must degrade, not fail.
func resolveSVGConverter(cfg *SVGRendererConfig) (string, error) {
    if cfg == nil || !cfg.Enabled {
        return "", nil
    }
    if cfg.ConverterPath != "" {
        if _, err := os.Stat(cfg.ConverterPath); err == nil {
            return cfg.ConverterPath, nil
        }
        return "", fmt.Errorf("configured svg converter %q not found", cfg.ConverterPath)
    }
    order := cfg.ConverterOrder
    if len(order) == 0 {
        order = []string{"rsvg-convert", "inkscape"}
    }
    for _, name := range order {
        if p, err := exec.LookPath(name); err == nil {
            return p, nil
        }
    }
    return "", nil // not an error: caller degrades gracefully
}
```

**Why not error when absent?** Because pandoc may still render referenced SVG via
its own rsvg-convert. Our converter is needed for *our* extraction pass (inline
SVG/HTML), not for referenced files. Returning `""` lets the caller skip extraction
and emit a single warning, preserving the rest of the document.

### 6.5 Inline SVG extraction algorithm

The core challenge is *not* conversion — it is correctly finding `<svg>…</svg>`
spans in Markdown without corrupting code fences or math.

**Properties we need:**

- Match `<svg` at the start of a line (possibly indented), case-insensitively.
- Balance nested `<svg>` elements (rare but legal).
- Respect quoted attribute values so `>` inside `"..."` does not terminate a tag.
- Honor `literalLines` so fenced code is untouched.
- Continue to EOF if unclosed; warn and leave unchanged.

**Recommended approach: a line scan with a small tag-depth counter.**

```
function ResolveInlineSVGBlocks(ctx, body, tmpDir, cfg) -> (string, error):
    if cfg == nil or not cfg.Enabled: return body
    converter = resolveSVGConverter(cfg)
    if converter == "": 
        warn("SVG: no converter found; inline <svg> blocks left as-is")
        return body

    lines   = split(body, "\n")
    protect = literalLines(lines)        # reuse existing helper
    out     = []
    i       = 0
    counter = 0

    while i < len(lines):
        if not protect[i] and looksLikeSVGOpen(lines[i]):
            (svgText, endIdx, ok) = collectSVGBlock(lines, i, protect)
            if not ok:
                warn("SVG: unclosed <svg> block starting at line %d; left as-is", i+1)
                append(out, lines[i]); i++; continue

            counter++
            name = cfg.ImagePrefix + fmt.Sprintf("svg-%03d.svg", counter)
            path = join(tmpDir, "images", name)
            ensure_dir(dirname(path))
            write(path, svgText)

            alt = "svg image " + counter
            if cfg.DefaultWidth != "":
                append(out, "![" + alt + "](./images/" + name + "){width=" + cfg.DefaultWidth + "}")
            else:
                append(out, "![" + alt + "](./images/" + name + ")")

            i = endIdx + 1
            continue

        append(out, lines[i])
        i++

    return join(out, "\n"), nil
```

**`collectSVGBlock`** must track depth while skipping quoted strings:

```
function collectSVGBlock(lines, start, protect) -> (text, endIdx, ok):
    depth = 0
    buf   = []
    for j from start to len(lines)-1:
        if j > start and protect[j]:
            # A fence started inside an <svg> block: treat as malformed
            return "", start, false
        line = lines[j]
        buf.append(line)
        depth += countTagDelta(line, "svg")   # +1 for <svg, -1 for </svg>, ignore self-closing
        if depth <= 0:
            return join(buf,"\n"), j, true
    return "", start, false
```

```go
// countTagDelta counts <svg ...> / </svg> occurrences on a line, ignoring
// occurrences inside quoted attribute values. Self-closing <svg .../> nets 0.
func countTagDelta(line string, tag string) int
```

**Pseudocode for the tokenizer (quoted-string aware):**

```
function countTagDelta(line, tag):
    delta = 0
    k = 0
    inQuote = 0      # 0=none, '"' or '\''
    while k < len(line):
        c = line[k]
        if inQuote != 0:
            if c == inQuote: inQuote = 0
            k++; continue
        if c == '"' or c == '\'':
            inQuote = c; k++; continue
        if c == '<':
            if startsWithCI(line, k+1, "/"+tag): delta--       # </svg...
            elif startsWithCI(line, k+1, tag)
                 and (k+1+len(tag) < len(line))
                 and isBoundary(line[k+1+len(tag)]):           # <svg or <svgfoo? require boundary
                # check for self-closing before '>'
                rest = line[k:]
                if isSelfClosing(rest): pass                    # <svg ... />
                else: delta++
            else: pass
        k++
    return delta
```

`isBoundary(c)` is true for whitespace, `>`, `/`, or end of line. This prevents
matching `<svgfoo>`.

> **Do not use a single mega-regex** like `(?s)<svg.*?</svg>`. It breaks on nested
> SVGs, on `</svg>` inside attribute values, and, most importantly, on fenced code.
> The line scan + `literalLines` is the correct, testable approach and matches the
> repository's existing style (`literal_regions.go`).

### 6.6 Substitution and asset naming

- Write extracted SVGs to `<tmp>/images/<prefix>svg-NNN.svg`.
- Substitute `![svg image N](./images/<prefix>svg-NNN.svg)`.
- Apply `{width=...}` only when `DefaultWidth` is configured.
- **Idempotency:** the generated Markdown uses standard image syntax, so a second
  run of `ResolveImagePaths` (which bundle mode does) simply copies the already
  staged file. Never emit raw `<svg>` again, or a re-run would re-extract.
- **Bundle mode:** `cfg.ImagePrefix` is prefixed per input file via
  `svgConfigWithImagePrefix`, mirroring `mermaidConfigWithImagePrefix`. Add this
  helper next to the existing one in `mermaid_config.go` (or a new
  `svg_config.go`).

### 6.7 Sizing policy

Decision to make explicitly (see Open Questions): what size should an inline SVG
render at?

Recommended default policy:

1. If `DefaultWidth` is set, always emit `{width=<DefaultWidth>}`.
2. Else emit no width and let pandoc use natural size.
3. If `MaxWidth` is set, never exceed it. (Note: `MaxWidth` was dropped during
   implementation; only `DefaultWidth` ships. This item is retained for history.)
4. Document the behavior in `--help` and the ticket.

Rationale: adding a width changes layout for everyone; defaulting to natural size
preserves authored SVG dimensions, and the flag gives control.

### 6.8 HTML `<img>` handling

Convert:

```html
<img src="./arch.svg" width="300">
```

to Markdown:

```markdown
![arch](./arch.svg){width=300px}
```

Rules:

- Only handle `<img ...>` tags whose `src` ends in `.svg` (optionally with query).
  Keep the blast radius small for the first version; PNG/JPG HTML imgs can be a
  follow-up.
- Extract `width`/`height` attributes; translate bare numbers to `px`, preserve
  units when present.
- If no converter is available, leave the `<img>` untouched and warn.
- Recognize self-closing (`/>`) and non-self-closing forms.
- Quoted-string-aware attribute scanning, same as the `<svg>` tokenizer.

### 6.9 Bundle integration

In `BuildBundleMarkdown`, after `StripYAMLFrontmatter` and before
`ResolveImagePathsWithPrefix`, insert:

```go
body, err = ResolveHTMLImages(body, svgConfigWithImagePrefix(svgCfg, assetPrefix))
body, err = ResolveInlineSVGBlocks(ctx, body, tmpDir, svgConfigWithImagePrefix(svgCfg, assetPrefix))
```

Pass the SVG config through `BundleInput` handling exactly as `mermaidCfg` is
passed. Update `cmd/.../bundle.go` to construct it.

### 6.10 Failure policy (uniform)

| Condition | Behavior |
|---|---|
| Feature disabled | No-op. |
| No converter found | Warn once: `SVG: no converter found; inline <svg>/<img> left as-is`. Continue. |
| Unclosed `<svg>` | Warn with line number; leave unchanged. Continue. |
| Converter exits non-zero | Warn per block; leave that block unchanged. Continue. |
| Converter produced no output | Same as non-zero exit. |
| Referenced `.svg` (pandoc path) | Untouched; pandoc handles or fails as today. Optional follow-up: pre-convert to avoid pandoc's fallback failure. |

This mirrors Mermaid's "warn and continue" philosophy and guarantees one bad SVG
never destroys a whole document.

### 6.11 Security

- We invoke `rsvg-convert`/`inkscape` with an explicit `argv` and a file path — no
  shell. No `--shell-escape` for LaTeX.
- SVG can contain external references (`<image href="http://...">`, XXE-style
  entities). `rsvg-convert` does not fetch remote resources by default; document
  this assumption and consider `--unlimited`/network flags explicitly if needed.
- Do not add `--shell-escape` to pandoc as part of this work.

---

## 7. Part 5 — Proposed API reference

New file `pkg/mdpdf/svg.go`:

```go
// SVGRendererConfig — see §6.3.
type SVGRendererConfig struct { /* ... */ }

func DefaultSVGRendererConfig() SVGRendererConfig

// ResolveInlineSVGBlocks finds raw <svg>...</svg> blocks (outside fenced code),
// writes each to <tmpDir>/images/, and replaces it with a Markdown image
// reference. It is non-fatal: on any problem it warns and leaves the source
// unchanged.
func ResolveInlineSVGBlocks(ctx context.Context, body, tmpDir string, cfg *SVGRendererConfig) (string, error)

// ResolveHTMLImages rewrites <img src="*.svg" ...> tags into Markdown image
// references. Non-SVG <img> tags are left unchanged.
func ResolveHTMLImages(body string, cfg *SVGRendererConfig) (string, error)

// ResolveSVGConverter exposes converter discovery for tests and diagnostics.
func ResolveSVGConverter(cfg *SVGRendererConfig) (string, error)

// svgConfigWithImagePrefix clones cfg with an added filename prefix.
func svgConfigWithImagePrefix(cfg *SVGRendererConfig, prefix string) *SVGRendererConfig
```

Modified:

```go
// pkg/mdpdf/pandoc.go
type PandocOptions struct {
    // ...
    SVG *SVGRendererConfig // NEW
}

// ConvertMarkdownFileToPDF: ResolveHTMLImages runs after StripYAMLFrontmatter
// and before ResolveImagePaths; ResolveInlineSVGBlocks runs after
// ResolveImagePaths and before RenderMermaidBlocks.

// pkg/mdpdf/bundle.go
func BuildBundleMarkdown(ctx, inputs, tmpDir string,
    mermaidCfg *MermaidRendererConfig,
    svgCfg *SVGRendererConfig,       // NEW
    resolveImages bool) (string, error)
```

CLI (`cmd/remarquee/cmds/upload/mermaid_section.go` or a new `svg_section.go`):

```
--svg                     bool    default true   Enable inline SVG handling
--svg-converter           string  default ""     Path to rsvg-convert/inkscape
--svg-default-width       string  default ""     Width for extracted inline SVG (e.g. 70%, 12cm)
```

---

## 8. Part 6 — Implementation plan (phased)

Do the phases in order. Each phase should compile, pass `go test ./pkg/mdpdf/...`,
and end with a focused commit.

### Phase 0 — Confirm the baseline (½ day)
- Read `pkg/mdpdf/{pandoc,images,mermaid,preprocess,literal_regions,bundle}.go`.
- Reproduce the behavior matrix in §5.2 using Appendix A.
- Add a short note to the diary with the observed results.

### Phase 1 — Config + converter resolution (½ day)
- Add `pkg/mdpdf/svg.go` with `SVGRendererConfig`,
  `DefaultSVGRendererConfig`, `ResolveSVGConverter`, `svgConfigWithImagePrefix`.
- Add `SVG *SVGRendererConfig` to `PandocOptions`.
- Unit tests: converter resolution with fake `PATH` entries; nil config; disabled.
- Files: `pkg/mdpdf/svg.go`, `pkg/mdpdf/svg_test.go`, `pkg/mdpdf/pandoc.go`.

### Phase 2 — Inline `<svg>` extraction (1–1.5 days)
- Implement `countTagDelta`, `collectSVGBlock`, `ResolveInlineSVGBlocks`.
- Reuse `literalLines`; add tests for:
  - single-line self-closing `<svg/>`
  - multi-line block
  - nested `<svg>`
  - `</svg>` inside an attribute value
  - `<svgfoo>` false positive
  - SVG inside a fenced code block (must be untouched)
  - unclosed block (warn + unchanged)
  - no converter (warn + unchanged)
- Files: `pkg/mdpdf/svg.go`, `pkg/mdpdf/svg_test.go`,
  `pkg/mdpdf/testdata/`.

### Phase 3 — HTML `<img>` conversion (½–1 day)
- Implement `ResolveHTMLImages` with quoted-string-aware attribute scanning.
- Tests: `src`/`width`/`height`, `/>`, missing converter, non-SVG left alone.
- Files: `pkg/mdpdf/svg.go`, `pkg/mdpdf/svg_test.go`.

### Phase 4 — Pipeline wiring (½ day)
- Wire both passes into `ConvertMarkdownFileToPDF`.
- Wire into `BuildBundleMarkdown` and update all call sites
  (`bundle.go`, `md.go`, `sync.go`, `conversion_workers.go` as needed).
- Integration test: render a document containing Form A + C + D + F and assert
  output pixels / no pandoc errors. `math_pdf_test.go` shows the real-PDF pattern.

### Phase 5 — CLI flags + docs (½ day)
- Add flags; parse into `SVGRendererConfig`.
- Update `README.md` / help text if appropriate.
- Add a `playbook` doc with manual QA steps.

### Phase 6 — Validation + delivery (½ day)
- `go build ./...`, `go test ./...`, `golangci-lint run`.
- Real end-to-end: `remarquee upload md --pdf-only` on a fixture; inspect PDF.
- Upload this guide to reMarkable (requested in the ticket).

---

## 9. Part 7 — Testing strategy

### Unit tests (fast, no external tools)

- `ResolveInlineSVGBlocks`: table-driven input → expected Markdown, using a fake
  converter script (a shell file that copies input to output) so no real
  rsvg-convert is needed.
- `ResolveHTMLImages`: table-driven.
- Tokenizer: `countTagDelta` with attribute-value edge cases.
- `ResolveSVGConverter`: manipulate `PATH` via `t.Setenv`.

### Integration tests (real PDF)

Follow `pkg/mdpdf/math_pdf_test.go`: skip if `pandoc`/`xelatex` are absent,
render a fixture, then assert the PDF exists and is non-trivial. Optionally use
`pdftoppm` + a pixel probe (as in Appendix A) to prove the SVG actually rendered.

### Fixtures (`pkg/mdpdf/testdata/`)

Add small, hand-written SVGs; keep them tiny and deterministic:

- `square.svg` — explicit `width`/`height`.
- `viewbox.svg` — `viewBox` only.
- `nested.svg` — nested `<svg>`.
- `attr-quote.svg` — contains `>`/`</svg>` inside an attribute.

### What "done" looks like

A single fixture document containing Form A, B, C, D, E, F renders to a PDF where
A/B/C/D/E visibly appear (vector) and F appears as a code listing, with no pandoc
errors and no `Package svg Error`.

---

## 10. Part 8 — CLI surface (draft help)

```
SVG flags:
      --svg                   Render inline <svg> blocks and <img src=...svg>
                              tags as vector images (default true)
      --svg-converter PATH    Path to rsvg-convert or inkscape
                              (default: auto-detect)
      --svg-default-width W   Width for extracted inline SVGs, e.g. 70%, 12cm
                              (default: natural size)
```

---

## 11. Part 9 — Edge cases and gotchas

- **Idempotency in bundle mode.** The bundle path runs image resolution twice.
  Emit only standard Markdown image syntax; never re-emit `<svg>`.
- **Fenced code must win.** Always consult `literalLines`. The most likely
  regression is corrupting a documentation block that *shows* SVG.
- **Attribute quoting.** `<svg data-x=">">` must not terminate the tag. The
  tokenizer must be quote-aware.
- **Self-closing tags.** `<svg .../>` nets zero depth; do not start an
  unterminated block.
- **Case sensitivity.** Real-world exports sometimes use `<SVG>`. Match
  case-insensitively, but preserve original bytes when no substitution happens.
- **`<img>` without `.svg`.** Leave PNG/JPG `<img>` tags alone for v1 to bound the
  change; they are currently dropped too, and fixing all HTML images is a larger
  follow-up.
- **Windows-style paths.** Not a concern on this project, but keep path joining
  via `filepath.Join`.
- **Converter differences.** `rsvg-convert` is fast but lacks some CSS/filters
  (`foreignObject`, advanced filters). If a conversion fails, warn and leave the
  block; do not fall back to Inkscape automatically in v1 (surprising slowness).
- **`--resolve-images=false`.** Decide whether SVG extraction is also skipped;
  recommend gating it on `ResolveImages` for consistency, and document it.
- **Large SVGs.** Cap or at least log very large embedded SVGs (they bloat the
  PDF). Not required for v1.
- **Missing converter + referenced SVG.** Even with our feature, pandoc's own
  fallback still hard-fails XeLaTeX. Consider a follow-up that pre-converts
  referenced SVGs when `rsvg-convert` is absent and Inkscape is present.

---

## 12. Part 10 — Open questions / decisions to raise

1. **Default inline SVG width.** Natural size vs. `100%` text width? Recommendation:
   natural size, with `--svg-default-width` to override.
2. **HTML `<img>` scope.** SVG-only in v1, or all image types? Recommendation:
   SVG-only v1.
3. **Should we pre-convert referenced SVGs too?** It would remove the pandoc
   fallback hard-failure. Trade-off: duplicate work (pandoc already calls
   rsvg-convert). Recommendation: defer, but log converter availability once.
4. **Converter preference.** `rsvg-convert` first (fast, verified), Inkscape
   fallback only if explicitly requested? Recommendation: `rsvg-convert` first;
   Inkscape fallback optional.
5. **Where do flags live?** New Glazed section (`SVG flags`) vs. the existing
   default group. Recommendation: new section, mirroring `Mermaid flags`.

---

## 13. Appendix A — Reproduction evidence

All commands were run on the target machine (pandoc 3.1.3, rsvg-convert present,
TeX Live XeLaTeX). They are reproduced here so the reader can re-verify.

**A.1 — Pandoc converts referenced SVG using rsvg-convert (trace via fake bin):**

```bash
mkdir -p /tmp/trace && cd /tmp/trace
cat > logo.svg <<'EOF'
<svg xmlns="http://www.w3.org/2000/svg" width="120" height="60">
  <rect width="120" height="60" fill="#336699"/>
</svg>
EOF
printf '![logo](./logo.svg)\n' > ref.md
mkdir -p fakebin
printf '#!/bin/bash\necho "CALLED rsvg-convert: $*" >> /tmp/trace/tools.log\nexec /usr/bin/rsvg-convert "$@"\n' > fakebin/rsvg-convert
chmod +x fakebin/rsvg-convert
PATH="/tmp/trace/fakebin:$PATH" pandoc ref.md -o ref.pdf --pdf-engine=xelatex
cat tools.log
# CALLED rsvg-convert: -f pdf -a --dpi-x 96 --dpi-y 96 -o .../hash.pdf .../logo.svg
```

**A.2 — remarquee renders referenced and inline SVG (blue vs red pixel probe):**

```bash
cd /tmp/svgtest/rmq
remarquee upload md --pdf-only --output-dir ./out ./doc.md   # doc.md has ![]svg and <svg>
pdftoppm -png -r 60 out/doc.pdf out/p
# Pixel scan: blue (referenced) present; red (inline raw) absent.
```

**A.3 — rsvg-convert absence causes a hard XeLaTeX failure:**

```bash
mkdir -p brokenbin
printf '#!/bin/bash\nexit 127\n' > brokenbin/rsvg-convert && chmod +x brokenbin/rsvg-convert
PATH="/tmp/svgtest/brokenbin:$PATH" pandoc ref.md -o broken.pdf --pdf-engine=xelatex
# [WARNING] Could not convert image ...: conversion from SVG failed
# ! Package svg Error: File `logo_svg-tex.pdf' is missing.
# NO PDF
```

**A.4 — data URI SVG renders:**

```bash
B64=$(base64 -w0 logo.svg)
printf '# Data URI\n\n![d](data:image/svg+xml;base64,%s)\n' "$B64" > dat.md
remarquee upload md --pdf-only --output-dir ./out ./dat.md
# PDF contains the SVG (pixel probe positive).
```

**A.5 — HTML `<img src=svg>` is dropped:**

```bash
printf '# HTML img\n\n<img src="./logo.svg" width="150">\n' > img.md
remarquee upload md --pdf-only --output-dir ./out ./img.md
pdftoppm -png -r 60 out/img.pdf out/img
# Pixel scan: zero blue pixels → tag discarded.
```

---

## 14. Appendix B — File and symbol index

| Path | Symbols |
|---|---|
| `pkg/mdpdf/pandoc.go` | `PandocOptions`, `DefaultPandocOptions`, `DefaultFromFormat`, `defaultLatexHeader`, `buildPandocArgs`, `ConvertMarkdownFileToPDF` |
| `pkg/mdpdf/images.go` | `inlineImageRegex`, `referenceImageRegex`, `referenceDefinitionRegex`, `ResolveImagePaths`, `ResolveImagePathsWithPrefix`, `splitInlineImageTarget`, `copyImageToTemp`, `isURL`, `copyFile` |
| `pkg/mdpdf/mermaid.go` | `MermaidRendererConfig`, `DefaultMermaidRendererConfig`, `mermaidBlockRegex`, `RenderMermaidBlocks`, `renderMermaidToPNG`, `resolveMmdcPath` |
| `pkg/mdpdf/mermaid_config.go` | `mermaidConfigWithImagePrefix` |
| `pkg/mdpdf/bundle.go` | `BundleInput`, `BuildBundleMarkdown` |
| `pkg/mdpdf/preprocess.go` | `StripYAMLFrontmatter`, `NormalizeListSpacing`, `FlattenDeepLists`, `isListItemLine`, `listIndentLevel` |
| `pkg/mdpdf/literal_regions.go` | `literalLines`, `fencePrefix`, `hasUnescapedDelimiter` |
| `pkg/mdpdf/layout.go` | `MarkdownLayoutDefault`, `MarkdownLayoutEditor`, `ApplyMarkdownLayoutPreset`, `NormalizeMarkdownLayout` |
| `cmd/remarquee/cmds/upload/mermaid_section.go` | `NewMermaidSection`, `addMermaidFlagsToCommand`, `addResolveImagesFlag`, `parseMermaidFlags` |
| `cmd/remarquee/cmds/upload/md.go` | `uploadMarkdownSettings`, `runUploadMarkdown` |
| `cmd/remarquee/cmds/upload/bundle.go` | `uploadBundleSettings`, `runUploadBundle`, `writeBundlePDF` |
| `cmd/remarquee/cmds/upload/sync.go` | `uploadSyncSettings`, `runUploadSync` |
| `cmd/remarquee/cmds/upload/conversion_workers.go` | parallel conversion workers |
| `pkg/rmdoc/render/v6_merge_background.go` | `cairoSVGScale` (context: another SVG-to-PDF path in the repo) |

---

## 15. Appendix C — One-page cheat sheet

```
Goal: make inline <svg> blocks render on reMarkable.

Reality check:
  ![](*.svg) and data: URIs ALREADY work (pandoc → rsvg-convert).
  raw <svg> and <img> are DROPPED.
  missing rsvg-convert = hard XeLaTeX crash.

Implement:
  1. pkg/mdpdf/svg.go: SVGRendererConfig + ResolveInlineSVGBlocks + ResolveHTMLImages
  2. reuse literalLines() so fenced code is safe
  3. substitute with ![](./images/prefix-svg-NNN.svg){width=...}
  4. wire into ConvertMarkdownFileToPDF AND BuildBundleMarkdown
  5. add --svg* flags next to mermaid flags
  6. warn-and-continue on every failure (never crash the doc)

Test:
  unit: tokenizer + extraction tables (fake converter)
  integration: render fixture, pixel-probe the PDF

Done when: a doc with forms A–F renders correctly; no "Package svg Error".
```
