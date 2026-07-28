---
Title: 'Intern Guide: Resilient Markdown to PDF Conversion'
Ticket: RMQ-0020
Status: active
Topics:
    - remarquee
    - upload
    - markdown
    - pdf
    - mdpdf
    - pandoc
    - xelatex
    - cli
    - go
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/remarquee/cmds/upload/bundle.go
      Note: Single-unit bundle conversion and upload orchestration.
    - Path: repo://cmd/remarquee/cmds/upload/conversion_workers.go
      Note: Serial and parallel conversion error collection.
    - Path: repo://cmd/remarquee/cmds/upload/md.go
      Note: Per-file Markdown conversion, upload continuation, and aggregate failure reporting.
    - Path: repo://pkg/doc/upload/02-remarquee-upload-reference.md
      Note: User-facing Markdown upload behavior that must be updated after implementation.
    - Path: repo://pkg/mdpdf/bundle.go
      Note: Bundle preprocessing and concatenation before shared conversion.
    - Path: repo://pkg/mdpdf/pandoc.go
      Note: Shared Markdown preprocessing and Pandoc subprocess invocation; proposed fix belongs here.
    - Path: repo://pkg/mdpdf/pandoc_test.go
      Note: Real Pandoc-to-PDF integration coverage and tool availability handling.
    - Path: repo://pkg/mdpdf/preprocess.go
      Note: Current frontmatter and line-oriented normalization behavior that must remain compatible.
    - Path: repo://pkg/mdpdf/preprocess_test.go
      Note: Pure preprocessing regression coverage for separators, frontmatter, and fenced code.
ExternalSources: []
Summary: Intern-ready analysis and implementation guide for preventing Pandoc YAML parser failures caused by ordinary Markdown thematic breaks, while preserving frontmatter, fenced code, batch resilience, and clear diagnostics.
LastUpdated: 2026-07-28T06:20:00-04:00
WhatFor: Understand and implement RMQ-0020 without needing prior knowledge of the remarquee Markdown-to-PDF pipeline.
WhenToUse: Read before changing pkg/mdpdf preprocessing, Pandoc arguments, Markdown upload commands, or conversion tests.
---


# Intern Guide: Resilient Markdown to PDF Conversion

## 1. Executive summary

`remarquee upload md` converts Markdown into a temporary preprocessed Markdown file, invokes Pandoc with XeLaTeX, and uploads the resulting PDF to reMarkable. The reported failure is:

```text
ERROR-CONVERT: /home/manuel/Downloads/scraper_workflow_framework_design.md — pandoc failed: YAML parse exception at line 9, column 0:
did not find expected <document start>
: exit status 64
ERRORS: convert-failed=1 upload-failed=0
```

The input is not a docmgr document with YAML frontmatter. It begins with a Markdown heading and contains an ordinary thematic-break line `---` at source line 10. Pandoc 3.1.3, using its default Markdown extensions, interprets that later separator through the `yaml_metadata_block` extension and reports a YAML parser error. The current preprocessor only strips a YAML block when it is at the beginning of a file, so it does not prevent this failure.

The safest first implementation is to disable Pandoc's `yaml_metadata_block` extension for the generated input, while retaining the existing explicit frontmatter stripping behavior. The Pandoc format selector uses a minus before the extension name:

```text
--from=markdown-yaml_metadata_block
```

This preserves ordinary Markdown separators and avoids rewriting document text. The implementation must also add regression tests for the exact user document shape, frontmatter, fenced code containing `---`, bundle conversion, and the command-line argument construction. Batch commands should continue to use the already-established per-file failure collection behavior rather than aborting the complete batch.

This ticket is a design and implementation guide. It does not claim that code changes have already been made.

## 2. Problem statement and scope

### 2.1 User-visible problem

A user requested a single Markdown file upload:

```bash
remarquee upload md ~/Downloads/scraper_workflow_framework_design.md
```

Conversion failed before any upload occurred. The command correctly reported one conversion failure and zero upload failures, but the diagnostic exposed an internal Pandoc parser detail and did not explain that the document contained a normal Markdown separator that conflicts with Pandoc's enabled YAML extension.

The desired behavior is:

- ordinary Markdown documents containing thematic breaks should convert successfully;
- true leading frontmatter should continue to be removed from the PDF, as documented;
- YAML-looking examples inside fenced code blocks must remain unchanged;
- `upload md`, `upload bundle`, and any other caller of `mdpdf.ConvertMarkdownFileToPDF` should receive the same behavior;
- one broken input in a directory must not prevent other inputs from being attempted;
- a failure that remains after preprocessing must include the input path, conversion phase, Pandoc output, and exit status;
- tests must not depend on the developer's personal Downloads directory.

### 2.2 In scope

- `pkg/mdpdf` input normalization and Pandoc argument construction;
- frontmatter and thematic-break semantics;
- single-file, PDF-only, directory, and bundle conversion paths;
- test fixtures and integration-test skip behavior;
- user-facing conversion documentation;
- error-message and diagnostics improvements that are directly related to this failure.

### 2.3 Out of scope

- replacing Pandoc;
- implementing a complete CommonMark parser;
- changing XeLaTeX layout, fonts, image resolution, Mermaid rendering, or upload authentication;
- changing reMarkable cloud semantics;
- automatically repairing arbitrary invalid Markdown;
- silently converting genuinely malformed YAML metadata into valid metadata;
- making a single bundled PDF partially successful. A bundle is one conversion unit and should fail as one unit if its concatenated input cannot be converted.

## 3. Current system architecture

### 3.1 End-to-end flow

The relevant flow is:

```text
Cobra command
    |
    +--> upload md: collectMarkdownInputs
    |        |
    |        +--> buildMarkdownConversionJobs
    |        +--> mdpdf.ConvertMarkdownFileToPDF per job
    |        +--> upload PDF per successful job
    |
    +--> upload bundle: collectMarkdownFilesForBundle
             |
             +--> mdpdf.BuildBundleMarkdown
             |       - read each source
             |       - strip leading frontmatter
             |       - resolve images and Mermaid
             |       - concatenate headings and page breaks
             +--> mdpdf.ConvertMarkdownFileToPDF once
             +--> upload one PDF

mdpdf.ConvertMarkdownFileToPDF
    |
    +--> read source bytes
    +--> StripYAMLFrontmatter
    +--> ResolveImagePaths, if enabled
    +--> RenderMermaidBlocks
    +--> NormalizeListSpacing
    +--> FlattenDeepLists
    +--> write temporary input.md
    +--> build Pandoc argv
    +--> exec.CommandContext(...)
    +--> write output PDF or return wrapped Pandoc error
```

### 3.2 `pkg/mdpdf/pandoc.go`

`PandocOptions` at `pkg/mdpdf/pandoc.go:14-36` carries the executable path, PDF engine, fonts, geometry, optional LaTeX headers, table-of-contents settings, syntax highlighting, Mermaid configuration, and image-resolution behavior.

`DefaultPandocOptions` at lines 38-46 selects `pandoc`, `xelatex`, DejaVu fonts, one-inch geometry, and image resolution. `ConvertMarkdownFileToPDF` begins at line 56. It reads the input at lines 85-89, strips only leading YAML frontmatter, performs image and Mermaid preprocessing at lines 97-113, normalizes lists at lines 115-116, writes a fixed temporary `input.md` at lines 122-125, constructs Pandoc arguments at lines 145-169, and runs the subprocess at lines 171-178.

This function is the correct semantic choke point. Adding the input-format option here fixes all callers that eventually use the function, including `upload md`, `upload bundle`, and source conversion helpers that create Markdown before invoking it.

### 3.3 `pkg/mdpdf/preprocess.go`

`StripYAMLFrontmatter` at `preprocess.go:5-42` recognizes only a file whose first line is exactly `---`, then removes through the first later line whose trimmed content is exactly `---`. It intentionally leaves non-frontmatter input unchanged. This is useful for docmgr-style files, but it cannot solve a later `---` that Pandoc interprets as YAML metadata.

`NormalizeListSpacing` at lines 75-101 and `FlattenDeepLists` at lines 103 onward are line-oriented transformations. They demonstrate an important maintenance constraint: preprocessing must not accidentally modify fenced code examples or Markdown constructs that merely resemble list syntax. Any new transformation must make its scope explicit and must have tests for protected regions.

### 3.4 `upload md`

`NewUploadMarkdownCommand` and its flags are in `cmd/remarquee/cmds/upload/md.go:45-108`. `runUploadMarkdown` starts at line 111. Input collection, remote collision detection, and Pandoc option configuration occur before execution.

PDF-only mode calls `convertMarkdownJobs` at lines 218-234. Upload mode converts each job and uploads successful PDFs. The existing upload loop at lines 271-300 records conversion and upload failures separately and continues to the next file. The final summary at lines 303-317 returns a non-zero error if any file failed.

This is valuable existing resilience. RMQ-0020 must not regress it by moving conversion failures into a top-level return that bypasses the remaining jobs.

### 3.5 `upload bundle`

`runUploadBundle` is in `cmd/remarquee/cmds/upload/bundle.go:122-226`. `writeBundlePDF` at lines 228-256 calls `mdpdf.BuildBundleMarkdown` and then `mdpdf.ConvertMarkdownFileToPDF` once. `BuildBundleMarkdown` itself strips frontmatter from each input and creates section headings and page breaks.

A bundle is intentionally one PDF. It does not have the same per-file continuation semantics as `upload md`: if the combined document cannot be converted, the bundle conversion should fail with a useful aggregate diagnostic.

### 3.6 Parallel workers

`convertMarkdownJobs` at `cmd/remarquee/cmds/upload/conversion_workers.go:52-112` supports serial and parallel conversion. In both modes it collects individual errors rather than canceling all workers on the first failure. In parallel mode a mutex protects the error slice. This code should consume the fixed `mdpdf` behavior without needing to understand YAML parsing.

## 4. Reproducing and explaining the failure

### 4.1 Evidence from the reported input

The input begins with normal Markdown:

```text
1  # A Pragmatic Workflow Builder and Execution Framework for Go and Goja
2
3  ## Fresh architecture, API specification, and implementation manual
4
5  **Project context:** `go-go-golems/scraper`  
6  **Design date:** 2026-07-28  
7  **Audience:** a developer joining the project to implement a replacement framework from first principles  
8  **Status:** proposed clean-slate design; not a compatibility specification for the existing engine
9
10 ---
11
12 ## Preface
```

There is no leading frontmatter block. The `---` at line 10 is a thematic break after a paragraph, not a YAML document.

The file contains many later `---` lines as section separators. It also contains fenced code examples, so a repair based on blindly replacing every occurrence of three hyphens would be unsafe.

### 4.2 Direct command evidence

With Pandoc 3.1.3 installed, the following direct conversion reproduces the failure:

```bash
pandoc ~/Downloads/scraper_workflow_framework_design.md \
  -o /tmp/scraper.pdf --pdf-engine=xelatex
```

Observed result:

```text
YAML parse exception at line 9, column 0:
did not find expected <document start>
```

A controlled copy with the thematic-break lines removed still failed when later `---` lines were present. A copy with all exact thematic-break lines replaced by blank lines succeeded in plain-text output mode. More importantly, explicitly disabling the Markdown YAML extension also succeeds without changing the source text:

```bash
pandoc --from=markdown-yaml_metadata_block \
  ~/Downloads/scraper_workflow_framework_design.md \
  -t plain -o /tmp/scraper.txt
```

In Pandoc's format-extension syntax, the `-yaml_metadata_block` suffix disables the extension. This is the central implementation fact to encode in a regression test.

### 4.3 Why current frontmatter stripping is insufficient

The current sequence is effectively:

```text
source starts with '#'
    -> StripYAMLFrontmatter returns source unchanged
    -> later '---' remains in generated input.md
    -> Pandoc parses with yaml_metadata_block enabled
    -> YAML parser sees thematic content as a metadata document
    -> exit status 64
```

The function name and behavior are both reasonable; the missing piece is the explicit Pandoc input contract. The generated input is body Markdown with frontmatter already handled by remarquee. It should not ask Pandoc to discover YAML metadata again.

## 5. Proposed solution

### 5.1 Primary decision: disable Pandoc YAML metadata parsing

Add a fixed input-format argument to the Pandoc invocation:

```go
argv := []string{
    "--from=markdown-yaml_metadata_block",
    inputPath,
    "-o", absOutPDF,
    "--pdf-engine=" + opts.PDFEngine,
    "--standalone",
    // existing options follow
}
```

The argument should be documented with a comment explaining that the leading minus disables the extension. The option belongs in `ConvertMarkdownFileToPDF`, not in each Cobra command, because `mdpdf` owns the generated Markdown contract.

Do not add a user-facing flag in v1. The behavior is a correctness invariant: the package deliberately strips docmgr-style frontmatter before invoking Pandoc. Exposing a flag would let callers re-enable the exact parser behavior that caused the failure and would create inconsistent output across commands.

### 5.2 Keep explicit frontmatter stripping

Continue calling `StripYAMLFrontmatter` before other preprocessing. This preserves the existing documented behavior that docmgr metadata should not appear in the PDF.

A follow-up may improve the stripper to handle UTF-8 BOM, CRLF, and the YAML closing marker `...`, but that is separate from the reported failure. Such changes require their own fixtures because frontmatter detection is a source-preservation concern.

### 5.3 Do not rewrite thematic breaks as the primary fix

Replacing `---` with `***` outside code fences would work around this specific Pandoc behavior, but it is the wrong first abstraction:

- it changes input text unnecessarily;
- a `---` line can be a setext heading underline rather than a thematic break;
- a line-oriented replacement must understand fenced code blocks;
- future Markdown constructs could make the replacement incomplete;
- disabling an unwanted parser extension preserves Pandoc's Markdown semantics.

A thematic-break normalizer may still be useful as a compatibility fallback for another renderer, but it should not be introduced to solve a parser-extension configuration problem.

### 5.4 Improve the wrapped diagnostic

The current error wrapper at `pandoc.go:176-178` includes Pandoc output and the process error, which is good. Add enough context for an operator to distinguish the tool phase and input mode. A target shape is:

```text
pandoc conversion failed for /absolute/path/document.md:
  input format: Markdown without YAML metadata blocks
  pdf engine: xelatex
  pandoc output: YAML parse exception ...
  cause: exit status 64
```

Do not discard Pandoc's original output. Do not include the entire preprocessed document in the error. If the command needs structured diagnostics later, introduce a typed error containing input path, phase, executable, and captured output while retaining `errors.Is`/`errors.As` behavior.

## 6. API and implementation sketch

### 6.1 Internal option helper

A small helper keeps argument construction testable without starting Pandoc:

```go
func buildPandocArgs(inputPath, outputPath string, opts PandocOptions, headers []string) []string {
    args := []string{
        "--from=markdown-yaml_metadata_block",
        inputPath,
        "-o", outputPath,
        "--pdf-engine=" + opts.PDFEngine,
        "--standalone",
        "-V", "mainfont=" + opts.MainFont,
        "-V", "monofont=" + opts.MonoFont,
        "-V", "geometry:" + opts.Geometry,
    }
    for _, header := range headers {
        args = append(args, "-H", header)
    }
    return args
}
```

The helper should either receive already-defaulted options or call a defaulting helper. Avoid making it part of the public package API unless another package needs it.

### 6.2 Conversion pseudocode

```text
ConvertMarkdownFileToPDF(ctx, source, destination, options):
    output = absolute(destination)
    options = applyDefaults(options)
    headers = prepareHeaders(options)

    bytes = read(source)
    body = StripYAMLFrontmatter(bytes)
    body = ResolveImagePaths(body, directory(source), tempDir)
    body = RenderMermaidBlocks(ctx, body, tempDir, options.Mermaid)
    body = NormalizeListSpacing(body)
    body = FlattenDeepLists(body, 4)
    write(tempDir/input.md, body)

    args = buildPandocArgs(tempDir/input.md, output, options, headers)
    run pandoc with ctx, working directory tempDir
    if process fails:
        return ConversionError{Input: source, Args: redacted(args), Output: capturedOutput, Cause: processError}
    return nil
```

Only `args` changes for the reported bug. The ordering of existing preprocessing stages should remain stable unless a test demonstrates an interaction.

### 6.3 Optional typed conversion error

If implementation chooses a typed error, keep it small:

```go
type ConversionError struct {
    Input      string
    PandocPath string
    PDFEngine  string
    Output     string
    Err        error
}

func (e *ConversionError) Error() string
func (e *ConversionError) Unwrap() error
```

Never persist command arguments containing secrets. Current flags do not include credentials, but a future custom header or engine option could. Redact before logging if the typed error stores arguments.

## 7. Detailed test strategy

Tests must prove both the fix and the boundaries around it.

### 7.1 Unit tests for argument construction

Add a test near `pkg/mdpdf/pandoc_test.go` or a new `pandoc_args_test.go`:

- construct default options;
- call the internal argument helper;
- assert that `--from=markdown-yaml_metadata_block` occurs exactly once;
- assert that existing PDF engine, font, geometry, header, ToC, highlight, and listings arguments remain present;
- assert that source and output paths are passed as separate arguments, not shell-concatenated strings.

This test is fast and does not require Pandoc or XeLaTeX.

### 7.2 Preprocessor fixtures

Extend `pkg/mdpdf/preprocess_test.go` with cases for:

- a normal document with `---` after a paragraph;
- multiple thematic breaks;
- `---` inside a fenced `yaml` block, which must remain byte-for-byte unchanged;
- leading docmgr frontmatter, which must still be removed;
- a document beginning with `#`, proving it is not treated as frontmatter;
- CRLF input if the frontmatter follow-up is implemented;
- setext heading syntax if any source rewrite is introduced. The preferred fix does not rewrite it, so this is a safety test rather than a required transform.

The unit tests should not assert that the preprocessor disables Pandoc. That belongs to argument or integration tests.

### 7.3 Real Pandoc integration test

Add a test to `pkg/mdpdf/pandoc_test.go` using a temporary file:

````markdown
# Thematic break regression

Paragraph before the separator.

---

## After the separator

```yaml
---
key: value
---
```
````

The Go source must use a raw string with a safe fence representation or concatenate fence lines so the test itself is valid. Invoke `ConvertMarkdownFileToPDF` with `DefaultPandocOptions` and assert that the output PDF exists and starts with `%PDF`.

The test should skip only when the required tools are unavailable. The existing helper checks standard Pandoc paths, but the production code accepts any executable on `PATH`; improve the helper to use `exec.LookPath` and separately check the selected PDF engine if practical. Keep the test deterministic and avoid relying on a developer-specific font path.

### 7.4 Exact regression fixture

Add a compact fixture derived from the user's failure, not the 6,275-line document. It should include:

- a title and metadata-like prose;
- a blank line;
- `---`;
- a subsection;
- several later separators;
- fenced YAML and JSON examples.

The fixture should reproduce the old failure under a deliberately constructed command without the disabling argument, and succeed through `ConvertMarkdownFileToPDF`. Do not make tests depend on `/home/manuel/Downloads`.

### 7.5 Bundle test

`pkg/mdpdf/bundle_test.go` already verifies frontmatter stripping and bundle headings. Add a case where two bundle inputs contain ordinary `---` separators and one contains a fenced YAML example. Build the bundle and convert it through the real path if tools are available. This catches regressions in the sequence:

```text
BuildBundleMarkdown -> bundle.md -> ConvertMarkdownFileToPDF
```

### 7.6 Batch continuation tests

At the command layer, preserve the existing tests and add a test with a fake Pandoc executable that fails for one filename and succeeds for another. Assert that:

- both jobs are attempted;
- the successful PDF is reported/generated;
- the command returns an error after all work completes;
- the summary counts only the failing conversion;
- no upload is attempted for the failed conversion.

The fake executable should inspect its input file or an explicit environment variable, write a minimal output file for success, and return a non-zero exit status for failure. Avoid making this test invoke the cloud API.

### 7.7 Test matrix

| Area | Regression | Expected result |
|---|---|---|
| argument builder | disabled YAML extension present | exact flag once |
| frontmatter | leading `---` metadata block | metadata omitted from body |
| ordinary separator | `---` after paragraph | conversion succeeds |
| fenced code | YAML example with `---` | code preserved |
| setext heading | `Heading` followed by `---` | semantics preserved |
| bundle | separators in multiple inputs | one PDF succeeds |
| batch | one fake conversion fails | other jobs continue |
| tool absence | Pandoc/XeLaTeX missing | test skips with clear reason |

## 8. Implementation phases

### Phase 1: establish the failing regression

1. Add a compact Markdown fixture containing the line-10 shape and fenced YAML.
2. Add an integration test that currently fails with the YAML parse exception, or add a direct argument-level test that demonstrates the current default.
3. Run only the affected package tests and record the exact command and tool versions in the diary.

Deliverable: a reproducible red test with no production behavior change.

### Phase 2: add the Pandoc input contract

1. Add `buildPandocArgs` or equivalent internal helper.
2. Add `--from=markdown-yaml_metadata_block`.
3. Preserve all existing arguments and working-directory behavior.
4. Update the helper's unit tests.
5. Run `gofmt` and `go test ./pkg/mdpdf`.

Deliverable: the exact regression test passes.

### Phase 3: validate interactions

1. Run the full `pkg/mdpdf` test package.
2. Run `go test ./cmd/remarquee/cmds/upload`.
3. Test PDF-only mode with the real file shape.
4. Test bundle mode with separators and fenced code.
5. Test a directory with one failing fake conversion and one successful conversion.

Deliverable: no regression in image, Mermaid, layout, bundle, or batch behavior.

### Phase 4: diagnostics and documentation

1. Decide whether the existing wrapped error is sufficient or add `ConversionError`.
2. Update `pkg/doc/upload/02-remarquee-upload-reference.md` to say that the converter strips frontmatter and disables Pandoc YAML metadata parsing for generated input.
3. Update the root README only if it documents the detailed conversion pipeline.
4. Add a short troubleshooting entry showing `--pdf-only` and the required tools.

Deliverable: operators can understand the fix without reading Go code.

### Phase 5: release validation

1. Run `go test ./...` from `remarquee`.
2. Run the project's formatting and lint commands if configured.
3. Run a local command such as:

```bash
go run ./cmd/remarquee upload md --pdf-only \
  --output-dir /tmp/rmq-0020-output \
  /path/to/thematic-break-fixture.md
```

4. Inspect the generated PDF and confirm it is non-empty and readable.
5. Review the diff for unrelated changes.
6. Update the diary, task checklist, changelog, and file relations.

## 9. Design decisions

### Decision: disable the parser extension instead of rewriting Markdown

- **Context:** Pandoc's default Markdown parser treats a later `---` as YAML metadata input in the reported document, but `---` is also normal Markdown syntax and can occur inside fenced examples.
- **Options considered:** disable `yaml_metadata_block`; rewrite thematic breaks; parse Markdown into an AST and re-render; add a command flag.
- **Decision:** disable the extension in the `mdpdf` Pandoc invocation and retain explicit leading-frontmatter stripping.
- **Rationale:** it addresses the parser configuration at the narrowest boundary, preserves source semantics, avoids fence-aware rewriting, and applies consistently to all callers.
- **Consequences:** Pandoc will no longer interpret metadata from generated Markdown. That is intentional because remarquee already strips frontmatter. Any future metadata support must be explicit in `PandocOptions`, not inferred from arbitrary body content.
- **Status:** proposed

### Decision: fix at `pkg/mdpdf`, not Cobra commands

- **Context:** `upload md`, `upload bundle`, sync, and source helpers can all reach the shared converter.
- **Options considered:** add arguments in each command; add a flag in command settings; centralize in `ConvertMarkdownFileToPDF`.
- **Decision:** centralize in `pkg/mdpdf/pandoc.go`.
- **Rationale:** one source of truth prevents command drift and makes package-level tests meaningful.
- **Consequences:** callers automatically inherit the safe input contract; direct callers of `mdpdf` are also protected.
- **Status:** proposed

### Decision: preserve batch continuation semantics

- **Context:** earlier resilience work exists because one bad Markdown file should not abort a directory upload.
- **Options considered:** return on first error; collect errors and continue; silently skip failures.
- **Decision:** collect, print, and return a non-zero aggregate error after all independent jobs finish.
- **Rationale:** successful files are useful, but automation still needs a failure signal.
- **Consequences:** callers must interpret partial success from output and exit status; tests must assert both continuation and final error.
- **Status:** accepted by current implementation, preserved by this ticket

## 10. Alternatives and risks

### 10.1 Rewrite `---` to `***`

This is a tempting two-line fix but is not semantically safe for setext headings and fenced code. It also creates a growing Markdown repair layer. Reject as the primary implementation.

### 10.2 Use a full Markdown parser

An AST could distinguish thematic breaks, setext headings, frontmatter, code blocks, and raw blocks. It would be substantially more code and would introduce another parser whose behavior must be reconciled with Pandoc. Use only if future transformations require structural Markdown understanding.

### 10.3 Remove all metadata support

Passing the extension-disable option alone does not remove `StripYAMLFrontmatter`; both are needed. If frontmatter stripping were removed, docmgr YAML would be rendered as document content instead of metadata, which would be a user-visible regression.

### 10.4 Let users choose metadata behavior

A flag could support special documents that intentionally need Pandoc metadata, but it would make output depend on a hidden command option and reopen the failure. Defer until a concrete use case exists; then name the mode explicitly and test it.

### 10.5 Pandoc version drift

Pandoc's extension defaults or diagnostics may change. The explicit `--from` option reduces reliance on defaults, but the integration test should still run against the supported version range. Capture the executable version in failure messages when practical.

### 10.6 XeLaTeX failures after parser success

The fix only removes the YAML parser failure. Documents may still fail because of invalid LaTeX, unsupported Unicode, deeply nested lists, missing fonts, image failures, Mermaid failures, or a missing PDF engine. Those errors must continue to propagate with their original output and must remain distinguishable from the YAML-extension case.

## 11. Acceptance criteria

The implementation is complete when all of the following have evidence:

- a Markdown document with an ordinary `---` separator converts successfully through `mdpdf.ConvertMarkdownFileToPDF`;
- the Pandoc argument list disables `yaml_metadata_block` exactly once;
- leading docmgr frontmatter is still absent from generated content;
- fenced code containing `---` is preserved;
- setext-heading behavior is not changed;
- bundle conversion succeeds for the same constructs;
- batch conversion attempts independent files after one conversion fails;
- final exit status remains non-zero for partial failure;
- diagnostics include the source path and captured Pandoc output;
- `go test ./pkg/mdpdf` and `go test ./cmd/remarquee/cmds/upload` pass;
- full repository tests pass or unrelated existing failures are recorded;
- user-facing upload documentation describes the new input contract;
- the implementation diary records commands, failures, versions, and review instructions.

## 12. File reference map

| File | Responsibility | Intern starting point |
|---|---|---|
| `pkg/mdpdf/pandoc.go` | shared conversion pipeline and subprocess | read `ConvertMarkdownFileToPDF` first |
| `pkg/mdpdf/preprocess.go` | frontmatter/list/deep-list transformations | compare existing invariants before adding logic |
| `pkg/mdpdf/preprocess_test.go` | pure preprocessing tests | add fence and separator fixtures here |
| `pkg/mdpdf/pandoc_test.go` | real Pandoc integration | add the PDF regression test here |
| `pkg/mdpdf/bundle.go` | per-input bundle preprocessing | verify frontmatter and separator behavior |
| `pkg/mdpdf/bundle_test.go` | bundle unit tests | add multi-input separator case |
| `cmd/remarquee/cmds/upload/md.go` | independent Markdown job orchestration | preserve continuation and summary behavior |
| `cmd/remarquee/cmds/upload/bundle.go` | one-PDF bundle orchestration | preserve one-unit failure semantics |
| `cmd/remarquee/cmds/upload/conversion_workers.go` | serial/parallel conversion | preserve collected per-job errors |
| `pkg/doc/upload/02-remarquee-upload-reference.md` | user-facing conversion contract | update after behavior is implemented |
| `ttmp/2026/05/15-RMQ-0015...` | prior batch-resilience rationale | read for error collection history |

## 13. Review instructions for the intern

Start in this order:

1. Read this document's sections 3 and 4.
2. Read `pkg/mdpdf/pandoc.go` and `preprocess.go` completely.
3. Read `pandoc_test.go`, `preprocess_test.go`, and `bundle_test.go`.
4. Read the conversion portions of `upload/md.go`, `upload/bundle.go`, and `conversion_workers.go`.
5. Reproduce the direct Pandoc failure and the explicit-extension-disable success.
6. Implement Phase 1 before Phase 2; do not change multiple unrelated preprocessing rules at once.

Reviewers should pay particular attention to:

- whether the exact Pandoc syntax really disables, rather than enables, `yaml_metadata_block`;
- whether leading frontmatter behavior remains unchanged;
- whether code fences and setext headings retain their semantics;
- whether all shared conversion callers receive the fix;
- whether a failed batch still processes independent jobs;
- whether error output leaks temporary paths or sensitive command arguments;
- whether tests are reproducible on machines without Pandoc or XeLaTeX.

## 14. Useful commands

From `/home/manuel/workspaces/2026-07-28/fix-remarquee-md/remarquee`:

```bash
# Inspect the relevant package.
go test ./pkg/mdpdf -count=1
go test ./cmd/remarquee/cmds/upload -count=1

# Validate Pandoc's extension behavior directly.
pandoc --list-extensions=markdown | grep yaml
pandoc --from=markdown-yaml_metadata_block input.md -t plain -o /tmp/input.txt

# Generate a local PDF without cloud authentication.
go run ./cmd/remarquee upload md --pdf-only \
  --output-dir /tmp/rmq-0020-output \
  /path/to/fixture.md

# Full validation after implementation.
gofmt -w pkg/mdpdf/*.go cmd/remarquee/cmds/upload/*.go
go test ./...
```

The `--pdf-only` path is the preferred manual smoke test because it exercises collection, preprocessing, Pandoc, and filesystem output without requiring reMarkable authentication.

## 15. Open questions

- Which Pandoc versions are officially supported in CI and release documentation?
- Should `pandocAvailable` use `exec.LookPath` and test the configured PDF engine, or remain a narrowly scoped integration-test helper?
- Is a typed `ConversionError` worth adding now, or does the existing wrapped error provide sufficient operator context?
- Should frontmatter support later accept `...` as a closing marker and UTF-8 BOM/CRLF input?
- Should bundle conversion expose the intermediate preprocessed Markdown under a debug flag when conversion fails?

These questions do not block the primary parser-extension fix. They should be answered during implementation review rather than solved speculatively in the first patch.
