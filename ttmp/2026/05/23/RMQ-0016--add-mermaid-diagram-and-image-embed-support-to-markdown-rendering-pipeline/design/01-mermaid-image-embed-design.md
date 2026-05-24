---
Title: Mermaid Diagram and Image Embed Design
Ticket: RMQ-0016
Status: active
Topics:
    - mdpdf
    - mermaid
    - image-embed
    - pandoc
    - xelatex
    - upload
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/remarquee/cmds/upload/bundle.go
      Note: upload bundle command that needs new CLI flags and BuildBundleMarkdown call update
    - Path: cmd/remarquee/cmds/upload/md.go
      Note: upload md command that needs new CLI flags
    - Path: pkg/mdpdf/bundle.go
      Note: BuildBundleMarkdown needs signature change for context and mermaid config
    - Path: pkg/mdpdf/pandoc.go
      Note: Core conversion function where new preprocessing steps will be inserted
    - Path: pkg/mdpdf/preprocess.go
      Note: Existing preprocessing functions (StripYAMLFrontmatter
ExternalSources: []
Summary: Exhaustive design and implementation guide for adding Mermaid diagram rendering and Markdown image embed support to remarquee's markdown-to-PDF pipeline (pkg/mdpdf). Written for an intern joining the project.
LastUpdated: 2026-05-23T00:00:00Z
WhatFor: Reference document for implementing mermaid + image embed features
WhenToUse: Read this before implementing any changes to pkg/mdpdf
---



# Mermaid Diagram and Image Embed Design

## 1. Executive Summary

remarquee converts Markdown files into PDFs using pandoc and xelatex, then uploads those PDFs to a reMarkable tablet. Today the pipeline handles plain text, code blocks, lists, and headings well — but it **cannot render Mermaid diagrams** (```` ```mermaid ```` fenced code blocks) and it **does not resolve or embed local image references** (`![alt](path.png)`). Both features are critical for technical documentation workflows where architecture diagrams, flowcharts, and annotated screenshots are common.

This document describes exactly how the remarquee Markdown rendering pipeline works today, identifies the gaps, and provides a detailed, phased implementation plan with pseudocode, API references, and test strategies. It is written so that a new intern can read it from top to bottom and understand every moving part.

---

## 2. Problem Statement and Scope

### 2.1 What is broken today

**Mermaid diagrams are silently dropped.** When a Markdown file contains a fenced code block with language tag `mermaid`, pandoc passes it through as a plain-text listing. The output PDF shows the raw Mermaid source text instead of a rendered diagram. There is no error, no warning — just a confusing rendering.

**Local image references are not resolved.** When a Markdown file references a relative image like `![diagram](./images/arch.png)`, pandoc resolves the path relative to the Markdown file's original location. But remarquee preprocesses the Markdown into a **temporary directory** (see `ConvertMarkdownFileToPDF` in `pandoc.go`), and the preprocessed file lives in `/tmp/remarquee-mdpdf-XXX/input.md`. The image path is now broken — pandoc either silently drops the image or fails with an error.

### 2.2 Scope

This ticket covers:

1. **Mermaid diagram rendering**: Detect `mermaid` fenced code blocks in the Markdown preprocessing step, render them to PNG images using the Mermaid CLI (`mmdc`), and replace the code block with an image embed.
2. **Image path resolution**: Rewrite relative image paths in Markdown so that pandoc can find them from the temp directory where the preprocessed Markdown lives.
3. **Both features must work** for `upload md`, `upload bundle`, and `--pdf-only` modes.

### 2.3 Out of scope

- Live-rendering Mermaid in the remarquee-ui web frontend.
- Adding new pandoc LaTeX filters or custom diagram types (PlantUML, GraphViz, etc.).
- SVG output (we target PNG for maximum compatibility with pandoc/xelatex).
- Modifying the reMarkable device rendering pipeline (rmdoc render-v6, etc.).

---

## 3. Current-State Architecture

### 3.1 The Markdown-to-PDF Pipeline

The pipeline has three stages: **collect**, **preprocess**, and **convert**.

```
┌──────────────────────────────────────────────────────────────────┐
│                     remarquee upload pipeline                     │
│                                                                    │
│  User input:         Collection:           Preprocessing:         │
│  ┌──────────┐       ┌──────────┐          ┌──────────────┐      │
│  │ *.md     │──────>│ collect  │─────────>│ preprocess   │      │
│  │ files +  │       │ files    │          │ (pkg/mdpdf)  │      │
│  │ dirs     │       └──────────┘          └──────┬───────┘      │
│  └──────────┘                                     │              │
│                                                    ▼              │
│                                            Conversion:           │
│                                           ┌──────────────┐      │
│                                           │ pandoc +     │      │
│                                           │ xelatex      │      │
│                                           └──────┬───────┘      │
│                                                   │              │
│                                                   ▼              │
│                                           Upload:               │
│                                           ┌──────────────┐      │
│                                           │ rmapi cloud  │      │
│                                           │ upload       │      │
│                                           └──────────────┘      │
└──────────────────────────────────────────────────────────────────┘
```

### 3.2 Key files and their responsibilities

| File | Package | Responsibility |
|------|---------|---------------|
| `pkg/mdpdf/preprocess.go` | `mdpdf` | Preprocessing functions: strip YAML frontmatter, normalize list spacing, flatten deep lists. Runs on the raw Markdown string **before** pandoc sees it. |
| `pkg/mdpdf/pandoc.go` | `mdpdf` | The core `ConvertMarkdownFileToPDF` function. Reads a Markdown file, applies preprocessing, writes a temp `input.md`, invokes pandoc with xelatex, produces a PDF. |
| `pkg/mdpdf/bundle.go` | `mdpdf` | `BuildBundleMarkdown` concatenates multiple Markdown files into one, inserting `\newpage` between them. Used by `upload bundle`. |
| `pkg/mdpdf/layout.go` | `mdpdf` | Layout presets (default, editor) that adjust pandoc geometry and LaTeX headers. |
| `cmd/remarquee/cmds/upload/md.go` | `upload` | The `upload md` Cobra command. Collects files, resolves remote dir, calls `mdpdf.ConvertMarkdownFileToPDF` for each, then uploads. |
| `cmd/remarquee/cmds/upload/bundle.go` | `upload` | The `upload bundle` Cobra command. Collects files, calls `BuildBundleMarkdown` then `ConvertMarkdownFileToPDF`, then uploads one PDF. |
| `cmd/remarquee/cmds/upload/conversion_workers.go` | `upload` | Parallel pandoc conversion worker pool. |
| `cmd/remarquee/cmds/upload/upload_helpers.go` | `upload` | Upload retry logic with auth refresh. |

### 3.3 The preprocessing call chain (critical path)

When a user runs `remarquee upload md ./doc.md`, here is exactly what happens:

```
1. md.go: runUploadMarkdown()
   ├── collectMarkdownInputs(args)         // resolve .md paths
   ├── resolveRemoteDir(date, remoteDir)   // compute /ai/YYYY/MM/DD/
   ├── configureMarkdownPandocOptions()    // build PandocOptions struct
   └── for each input:
       ├── mdpdf.ConvertMarkdownFileToPDF(ctx, absPath, outPDF, opts)
       │   ├── os.ReadFile(mdPath)                    // read original .md
       │   ├── StripYAMLFrontmatter(body)             // remove --- blocks
       │   ├── NormalizeListSpacing(body)             // fix list blanks
       │   ├── FlattenDeepLists(body, 4)              // cap nesting
       │   ├── os.WriteFile(tmpDir + "/input.md")     // write preprocessed
       │   └── exec.CommandContext("pandoc", ...)      // run pandoc
       └── uploadPDFToRemoteWithAuthRetry()            // upload to cloud
```

The key insight: **preprocessing happens on the string content** before writing to the temp directory. The preprocessed Markdown is written to `/tmp/remarquee-mdpdf-XXX/input.md`. Pandoc then converts this temp file.

### 3.4 Why image paths break

Consider this Markdown:

```markdown
![Architecture](./images/arch.png)
```

The original file is at `/home/user/docs/README.md`. The image is at `/home/user/docs/images/arch.png`.

During preprocessing, remarquee reads the Markdown, transforms it, and writes it to `/tmp/remarquee-mdpdf-XXX/input.md`. Pandoc runs on this temp file and tries to resolve `./images/arch.png` relative to `/tmp/remarquee-mdpdf-XXX/`. That path doesn't exist there, so the image is dropped or pandoc errors.

### 3.5 Why Mermaid blocks are ignored

Pandoc does not understand the `mermaid` language tag in fenced code blocks. It renders the content as a plain-text code listing. The Mermaid source text appears verbatim in the PDF.

---

## 4. Gap Analysis

| Feature | Current State | Desired State |
|---------|---------------|---------------|
| Mermaid ` ```mermaid ` blocks | Rendered as plain text code listings | Rendered as inline PNG diagrams |
| `![](path.png)` relative images | Broken (paths resolved from temp dir) | Resolved from original Markdown file's directory |
| `![](https://...)` remote images | Not tested, likely dropped | Should work if pandoc can fetch; out of scope for now |
| Error handling for missing mmdc | N/A | Clear error message with installation instructions |
| Bundle mode image paths | N/A | Must resolve per-file relative paths correctly |

---

## 5. Proposed Architecture

### 5.1 High-level approach

We add two new preprocessing steps to the pipeline:

1. **`ResolveImagePaths(body, sourceDir, tmpDir)`** — rewrites `![](./rel/path.png)` Markdown image syntax to use absolute paths or copies images into the temp directory.
2. **`RenderMermaidBlocks(body, sourceDir, tmpDir)`** — finds ```` ```mermaid ```` fenced code blocks, renders each one to a PNG using `mmdc`, and replaces the block with `![mermaid-N](rendered-mermaid-001.png)`.

Both run **after** YAML stripping but **before** list normalization, because they may introduce new image references.

### 5.2 Updated preprocessing call chain

```
1. Read original Markdown
2. StripYAMLFrontmatter(body)
3. ResolveImagePaths(body, sourceDir, tmpDir)         // NEW
4. RenderMermaidBlocks(body, sourceDir, tmpDir)         // NEW
5. NormalizeListSpacing(body)
6. FlattenDeepLists(body, 4)
7. Write preprocessed Markdown to tmpDir/input.md
8. Run pandoc
```

### 5.3 Design decisions and tradeoffs

**Decision 1: Copy images into the temp directory (not absolute paths).**

- *Why:* Pandoc resolves relative paths from the input file's directory. If we copy images into the same temp directory and rewrite paths to be relative to the preprocessed `input.md`, pandoc just works. Absolute paths would also work on Linux but would break if the Markdown is ever processed inside a container or sandbox.
- *Tradeoff:* Extra disk I/O to copy images. Acceptable because images are typically small and the temp dir is already ephemeral.

**Decision 2: Render Mermaid via `mmdc` CLI (mermaid-cli).**

- *Why:* `mmdc` (the official Mermaid CLI, an npm package) is the standard way to render Mermaid diagrams headlessly. It uses Puppeteer under the hood. It produces PNG output that pandoc can embed directly.
- *Alternatives considered:*
  - Mermaid Go library: none exist with good coverage.
  - Browser-based rendering: too heavy for a CLI tool.
  - Pandoc filter (Lua): possible but adds complexity; Mermaid rendering requires a JavaScript runtime.
- *Tradeoff:* Adds an external dependency (Node.js + mmdc). We mitigate this by making it opt-in: if `mmdc` is not found, Mermaid blocks remain as code listings with a warning.

**Decision 3: Per-file Mermaid rendering (not batch).**

- *Why:* Each Markdown file may have its own set of Mermaid blocks. In bundle mode, files are concatenated **after** preprocessing. Each file's Mermaid images are rendered into that file's temp directory, then the final concatenated Markdown uses the correct paths.
- *Tradeoff:* In bundle mode, Mermaid rendering happens once per source file, not once per bundle. This is actually the correct behavior — each source file owns its diagrams.

**Decision 4: New `MermaidOptions` struct on `PandocOptions`.**

- *Why:* We need to configure `mmdc` path, output scale, and background color. Adding these to `PandocOptions` keeps the configuration surface centralized.
- *Tradeoff:* `PandocOptions` grows. Acceptable because the struct is already a bag of rendering knobs.

---

## 6. Detailed Design

### 6.1 New types and constants

```go
// pkg/mdpdf/mermaid.go

// MermaidRendererConfig controls how Mermaid code blocks are rendered to images.
type MermaidRendererConfig struct {
    // MmdcPath is the path to the mmdc binary. If empty, "mmdc" is looked up
    // in $PATH. If not found, Mermaid blocks are left as-is with a warning.
    MmdcPath string

    // Enabled controls whether Mermaid rendering is attempted at all.
    // Default: true. Set to false to skip even if mmdc is available.
    Enabled bool

    // Scale controls the pixel scale of rendered diagrams (1x, 2x, 3x).
    // Default: 2 (good balance of quality and file size for PDF embedding).
    Scale int

    // BackgroundColor sets the diagram background. Default: "white".
    BackgroundColor string

    // Width sets the diagram width in pixels. 0 = auto (use Mermaid default).
    Width int

    // Theme sets the Mermaid theme. Default: "default". Options: "default",
    // "dark", "forest", "neutral".
    Theme string
}

// DefaultMermaidRendererConfig returns sensible defaults.
func DefaultMermaidRendererConfig() MermaidRendererConfig {
    return MermaidRendererConfig{
        Enabled:         true,
        Scale:           2,
        BackgroundColor: "white",
        Theme:           "default",
    }
}
```

The `PandocOptions` struct gains a new field:

```go
type PandocOptions struct {
    // ... existing fields ...

    // Mermaid configures Mermaid diagram rendering. If nil, Mermaid blocks
    // are left as plain-text code listings.
    Mermaid *MermaidRendererConfig
}
```

### 6.2 Image path resolution

```go
// pkg/mdpdf/images.go

import (
    "path/filepath"
    "regexp"
    "strings"
    "os"
    "fmt"
)

// imageEmbedRegex matches Markdown image syntax: ![alt](path)
// It captures the alt text and the path separately.
var imageEmbedRegex = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)

// ResolveImagePaths rewrites relative image paths in Markdown to be resolvable
// from the target directory (tmpDir). It copies image files into tmpDir and
// rewrites paths to be relative to the preprocessed Markdown file.
//
// Parameters:
//   - body: the Markdown content (after YAML stripping)
//   - sourceDir: the directory of the original Markdown file (for resolving relative paths)
//   - tmpDir: the temp directory where the preprocessed Markdown will be written
//
// Returns the rewritten Markdown body.
func ResolveImagePaths(body string, sourceDir string, tmpDir string) (string, error) {
    // For each ![alt](path) match:
    //   1. Skip if path is absolute or URL (http://, https://)
    //   2. Resolve path relative to sourceDir
    //   3. Copy the image file into tmpDir/images/
    //   4. Rewrite the Markdown to use ./images/<filename>
    //
    // Error handling:
    //   - If the image file doesn't exist, emit a warning and leave the path unchanged.
    //   - If the copy fails, return an error.
    ...
}
```

**Pseudocode for ResolveImagePaths:**

```
function ResolveImagePaths(body, sourceDir, tmpDir):
    imagesDir = path.join(tmpDir, "images")
    os.MkdirAll(imagesDir, 0755)

    for each match of ![alt](path) in body:
        if path starts with "http://" or "https://" or path starts with "/":
            continue  // skip URLs and absolute paths

        resolvedPath = filepath.Join(sourceDir, path)
        if not fileExists(resolvedPath):
            log warning: "image not found: <resolvedPath>"
            continue

        filename = filepath.Base(resolvedPath)
        destPath = filepath.Join(imagesDir, filename)

        // Handle filename collisions by adding a numeric suffix
        if fileExists(destPath) and not sameFile(resolvedPath, destPath):
            filename = addNumericSuffix(filename)
            destPath = filepath.Join(imagesDir, filename)

        copyFile(resolvedPath, destPath)

        // Rewrite the path in the Markdown
        body = replace match with ![alt](./images/<filename>)

    return body
```

### 6.3 Mermaid block rendering

```go
// pkg/mdpdf/mermaid.go

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "regexp"
    "strings"
)

// mermaidBlockRegex matches ```mermaid ... ``` fenced code blocks.
// It captures the Mermaid source code inside.
var mermaidBlockRegex = regexp.MustCompile(
    "(?s)" +                          // dot matches newlines
    "```mermaid\\s*\\n" +             // opening fence
    "(.*?)" +                         // capture group 1: mermaid source
    "\\n\\s*```",                     // closing fence
)

// RenderMermaidBlocks finds Mermaid fenced code blocks, renders each to a PNG
// using mmdc, and replaces the block with an image embed.
//
// Parameters:
//   - ctx: context for cancellation
//   - body: the Markdown content (after YAML stripping and image resolution)
//   - tmpDir: the temp directory for writing rendered PNGs
//   - config: Mermaid renderer configuration
//
// Returns the rewritten Markdown body.
func RenderMermaidBlocks(ctx context.Context, body string, tmpDir string, config *MermaidRendererConfig) (string, error) {
    if config == nil || !config.Enabled {
        return body, nil
    }

    mmdcPath, err := resolveMmdcPath(config.MmdcPath)
    if err != nil {
        // mmdc not found: warn and leave blocks as-is
        log.Warn().Msg("mmdc not found; Mermaid blocks will render as plain text. " +
            "Install with: npm install -g @mermaid-js/mermaid-cli")
        return body, nil
    }

    imagesDir := filepath.Join(tmpDir, "images")
    if err := os.MkdirAll(imagesDir, 0o755); err != nil {
        return body, fmt.Errorf("failed to create images directory: %w", err)
    }

    counter := 0
    result := mermaidBlockRegex.ReplaceAllStringFunc(body, func(match string) string {
        counter++
        submatch := mermaidBlockRegex.FindStringSubmatch(match)
        if len(submatch) < 2 {
            return match // shouldn't happen, but be safe
        }
        mermaidSource := strings.TrimSpace(submatch[1])
        if mermaidSource == "" {
            return match // empty block, skip
        }

        imgFilename := fmt.Sprintf("mermaid-%03d.png", counter)
        imgPath := filepath.Join(imagesDir, imgFilename)

        if err := renderMermaidToPNG(ctx, mmdcPath, mermaidSource, imgPath, config); err != nil {
            log.Warn().Err(err).Msgf("failed to render Mermaid block %d; leaving as text", counter)
            return match // leave as-is on error
        }

        return fmt.Sprintf("![mermaid diagram %d](./images/%s)", counter, imgFilename)
    })

    return result, nil
}

// renderMermaidToPNG writes the Mermaid source to a temp .mmd file and
// invokes mmdc to render it to a PNG at the specified output path.
func renderMermaidToPNG(ctx context.Context, mmdcPath string, source string, outPath string, config *MermaidRendererConfig) error {
    // 1. Write source to a temp .mmd file
    // 2. Build mmdc command: mmdc -i input.mmd -o output.png -s <scale> -b <bg> -t <theme>
    // 3. Run command with context for cancellation
    // 4. Verify output file exists
    ...
}

// resolveMmdcPath finds the mmdc binary.
func resolveMmdcPath(override string) (string, error) {
    if override != "" {
        if _, err := os.Stat(override); err == nil {
            return override, nil
        }
        return "", fmt.Errorf("mmdc path %q not found", override)
    }
    return exec.LookPath("mmdc")
}
```

**Pseudocode for renderMermaidToPNG:**

```
function renderMermaidToPNG(ctx, mmdcPath, source, outPath, config):
    // Write Mermaid source to temp file
    mmdFile = path.join(tmpDir, "mermaid-input.mmd")
    os.WriteFile(mmdFile, source, 0644)

    // Build mmdc command
    args = ["-i", mmdFile, "-o", outPath]
    if config.Scale > 0:
        args = append(args, "-s", strconv.Itoa(config.Scale))
    if config.BackgroundColor != "":
        args = append(args, "-b", config.BackgroundColor)
    if config.Theme != "":
        args = append(args, "-t", config.Theme)
    if config.Width > 0:
        args = append(args, "-w", strconv.Itoa(config.Width))

    cmd = exec.CommandContext(ctx, mmdcPath, args...)
    output, err = cmd.CombinedOutput()
    if err != nil:
        return fmt.Errorf("mmdc failed: %s: %w", string(output), err)

    // Verify output
    if _, err = os.Stat(outPath); err != nil {
        return fmt.Errorf("mmdc output not found: %w", err)
    }

    return nil
```

### 6.4 Integration into ConvertMarkdownFileToPDF

The main conversion function in `pandoc.go` needs to be updated:

```go
// Updated excerpt from pkg/mdpdf/pandoc.go

func ConvertMarkdownFileToPDF(ctx context.Context, mdPath string, outPDF string, opts PandocOptions) error {
    // ... existing setup code ...

    mdBytes, err := os.ReadFile(mdPath)
    if err != nil {
        return errors.Wrap(err, "failed to read markdown file")
    }
    body := StripYAMLFrontmatter(string(mdBytes))

    // --- NEW: resolve image paths ---
    sourceDir := filepath.Dir(mdPath)
    body, err = ResolveImagePaths(body, sourceDir, tmpDir)
    if err != nil {
        return errors.Wrap(err, "failed to resolve image paths")
    }

    // --- NEW: render Mermaid blocks ---
    body, err = RenderMermaidBlocks(ctx, body, tmpDir, opts.Mermaid)
    if err != nil {
        // Non-fatal: log warning, continue with unrendered blocks
        log.Warn().Err(err).Msg("Mermaid rendering had errors")
    }

    // --- existing preprocessing ---
    body = NormalizeListSpacing(body)
    body = FlattenDeepLists(body, 4)

    // ... write to tmpDir/input.md and run pandoc ...
}
```

**Key change:** We now pass `sourceDir` (the directory of the original Markdown file) to `ResolveImagePaths` so it can find relative images. We also pass `tmpDir` so both functions can write images there.

### 6.5 Integration into BuildBundleMarkdown

For bundle mode, each file is preprocessed individually **before** concatenation. The approach:

```go
// Updated excerpt from pkg/mdpdf/bundle.go

func BuildBundleMarkdown(ctx context.Context, inputs []BundleInput, mermaidCfg *MermaidRendererConfig) (string, error) {
    // For each input:
    //   1. Read and strip YAML frontmatter
    //   2. Create a per-input temp directory for images
    //   3. Resolve image paths (relative to that input's sourceDir)
    //   4. Render Mermaid blocks
    //   5. Normalize lists and flatten deep lists
    //   6. Concatenate all processed bodies

    // The tricky part: each input's images are in different subdirectories.
    // We need to copy them all into a shared images directory so the final
    // pandoc run can find them.
    ...
}
```

**Pseudocode for bundle Mermaid/image handling:**

```
function BuildBundleMarkdown(ctx, inputs, mermaidCfg):
    sharedImagesDir = path.join(bundleTmpDir, "images")
    os.MkdirAll(sharedImagesDir, 0755)

    for i, input in inputs:
        mdBytes = os.ReadFile(input.Path)
        body = StripYAMLFrontmatter(string(mdBytes))

        sourceDir = filepath.Dir(input.Path)

        // Resolve images for this file into a per-file staging area
        fileImagesDir = path.join(bundleTmpDir, fmt.Sprintf("file-%d-images", i))
        body, err = ResolveImagePaths(body, sourceDir, bundleTmpDir)
        // This copies images from sourceDir into sharedImagesDir

        // Render Mermaid for this file
        body, err = RenderMermaidBlocks(ctx, body, bundleTmpDir, mermaidCfg)
        // This writes mermaid-001.png etc. into sharedImagesDir

        body = NormalizeListSpacing(body)
        body = FlattenDeepLists(body, 4)

        // Append to bundle with section heading
        builder.WriteString("# " + input.Title + "\n\n")
        builder.WriteString(body)
        builder.WriteString("\n\n")
        if i < len(inputs) - 1:
            builder.WriteString("```{=latex}\n\\newpage\n```\n\n")

    return builder.String()
```

**Important:** `BuildBundleMarkdown` currently does not take a `context.Context` or `MermaidRendererConfig`. The function signature must change to accept both. All callers (in `bundle.go` command handler) must be updated.

### 6.6 Bundle mode path uniqueness

When multiple Markdown files reference images with the same filename (e.g., both files have `./images/diagram.png`), we need to avoid collisions. Two approaches:

1. **Prefix with source file name:** `file1-diagram.png`, `file2-diagram.png`.
2. **Use a hash:** `diagram-a1b2c3.png` based on content.

We choose approach 1 because it is debuggable. The `ResolveImagePaths` function receives a `fileIndex` parameter for bundle mode:

```
// In bundle mode, ResolveImagePaths prefixes image filenames with "file-N-"
// to avoid collisions when multiple source files reference images with the same name.
```

---

## 7. CLI Flag Integration

### 7.1 New flags for `upload md` and `upload bundle`

Both commands get the same new flags:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--mermaid` | bool | `true` | Enable Mermaid diagram rendering |
| `--mmdc-path` | string | `""` (auto-detect) | Path to the `mmdc` binary |
| `--mermaid-scale` | int | `2` | Pixel scale for rendered Mermaid diagrams |
| `--mermaid-theme` | string | `"default"` | Mermaid theme (default, dark, forest, neutral) |
| `--mermaid-bg` | string | `"white"` | Background color for Mermaid diagrams |
| `--resolve-images` | bool | `true` | Enable local image path resolution |

### 7.2 Flag wiring pseudocode

```go
// In md.go and bundle.go command setup:

// Mermaid flags
cmd.Flags().BoolVar(&s.Mermaid, "mermaid", true, "Render Mermaid code blocks as diagrams (requires mmdc)")
cmd.Flags().StringVar(&s.MmdcPath, "mmdc-path", "", "Path to mmdc binary (default: auto-detect from $PATH)")
cmd.Flags().IntVar(&s.MermaidScale, "mermaid-scale", 2, "Pixel scale for rendered Mermaid diagrams")
cmd.Flags().StringVar(&s.MermaidTheme, "mermaid-theme", "default", "Mermaid theme: default, dark, forest, neutral")
cmd.Flags().StringVar(&s.MermaidBg, "mermaid-bg", "white", "Background color for Mermaid diagrams")

// Image flags
cmd.Flags().BoolVar(&s.ResolveImages, "resolve-images", true, "Resolve and embed local image references")
```

These flags map into the `PandocOptions.Mermaid` field and a new `ResolveImages` field on the command settings structs.

---

## 8. Phased Implementation Plan

### Phase 1: Image Path Resolution (estimated: 1 day)

**Goal:** Make `![alt](./images/foo.png)` work in `upload md` mode.

**Files to modify:**

1. `pkg/mdpdf/images.go` — **NEW FILE**. Implement `ResolveImagePaths`.
2. `pkg/mdpdf/images_test.go` — **NEW FILE**. Unit tests.
3. `pkg/mdpdf/pandoc.go` — Add call to `ResolveImagePaths` in `ConvertMarkdownFileToPDF`.
4. `pkg/mdpdf/pandoc_test.go` — Add integration test with an image.

**Validation:**
```bash
# Create a test Markdown file with a local image reference
mkdir -p /tmp/test-images
echo "# Test" > /tmp/test-images/README.md
echo "![logo](./logo.png)" >> /tmp/test-images/README.md
convert -size 100x100 xc:blue /tmp/test-images/logo.png

# Run the pipeline
remarquee upload md --pdf-only --output-dir /tmp/out /tmp/test-images/README.md

# Verify the PDF contains the image (manual inspection or pdfcmp)
```

### Phase 2: Mermaid Block Rendering (estimated: 2 days)

**Goal:** Render ```` ```mermaid ```` blocks as inline PNG diagrams.

**Files to modify:**

1. `pkg/mdpdf/mermaid.go` — **NEW FILE**. Implement `RenderMermaidBlocks`, `renderMermaidToPNG`, `resolveMmdcPath`, types.
2. `pkg/mdpdf/mermaid_test.go` — **NEW FILE**. Unit tests for block detection, rendering (with mmdc mock), and error handling.
3. `pkg/mdpdf/pandoc.go` — Add call to `RenderMermaidBlocks`. Add `Mermaid` field to `PandocOptions`.
4. `pkg/mdpdf/preprocess_test.go` — Add test for interaction with other preprocessors.

**Prerequisite:** Install mmdc.

```bash
# Install mermaid-cli globally
npm install -g @mermaid-js/mermaid-cli

# Verify
mmdc --version

# Test rendering a simple diagram
echo 'graph TD\n  A --> B' > /tmp/test.mmd
mmdc -i /tmp/test.mmd -o /tmp/test.png -s 2 -b white
```

**Validation:**
```bash
# Create a test Markdown with a Mermaid block
cat > /tmp/test-mermaid.md << 'EOF'
# Architecture

\`\`\`mermaid
graph TD
    A[User] --> B[API Gateway]
    B --> C[Backend]
    C --> D[(Database)]
\`\`\`
EOF

remarquee upload md --pdf-only --output-dir /tmp/out /tmp/test-mermaid.md
# Verify the PDF shows a diagram, not source text
```

### Phase 3: CLI Flag Integration (estimated: 0.5 day)

**Goal:** Expose Mermaid and image flags on the CLI.

**Files to modify:**

1. `cmd/remarquee/cmds/upload/md.go` — Add flags, wire into `PandocOptions`.
2. `cmd/remarquee/cmds/upload/bundle.go` — Add flags, wire into `PandocOptions`.
3. `cmd/remarquee/cmds/upload/root.go` — (no changes needed, flags are on subcommands).

**Validation:**
```bash
# Test --mermaid=false skips rendering
remarquee upload md --pdf-only --mermaid=false --output-dir /tmp/out /tmp/test-mermaid.md

# Test --mermaid-scale=3
remarquee upload md --pdf-only --mermaid-scale=3 --output-dir /tmp/out /tmp/test-mermaid.md

# Test --resolve-images=false
remarquee upload md --pdf-only --resolve-images=false --output-dir /tmp/out /tmp/test-images/README.md
```

### Phase 4: Bundle Mode Support (estimated: 1 day)

**Goal:** Make both features work in `upload bundle` mode.

**Files to modify:**

1. `pkg/mdpdf/bundle.go` — Update `BuildBundleMarkdown` signature to accept context and Mermaid config. Per-file image resolution and Mermaid rendering.
2. `cmd/remarquee/cmds/upload/bundle.go` — Update callers.

**Validation:**
```bash
# Create two Markdown files with images and Mermaid
# Bundle them and verify both are rendered correctly
remarquee upload bundle --pdf-only --output-dir /tmp/out /tmp/bundle-dir/
```

### Phase 5: Documentation and Help Pages (estimated: 0.5 day)

**Goal:** Update remarquee help pages and README.

**Files to modify:**

1. `README.md` — Mention Mermaid and image support.
2. `pkg/doc/upload/02-remarquee-upload-reference.md` — Document new flags.
3. `pkg/doc/upload/03-remarquee-upload-bundle.md` — Document bundle behavior.

---

## 9. Test Strategy

### 9.1 Unit tests

| Test | File | What it validates |
|------|------|-------------------|
| `TestResolveImagePaths_RelativePath` | `images_test.go` | Rewrites `./img.png` to `./images/img.png` and copies the file |
| `TestResolveImagePaths_AbsolutePath` | `images_test.go` | Leaves `/absolute/path.png` unchanged |
| `TestResolveImagePaths_URL` | `images_test.go` | Leaves `https://example.com/img.png` unchanged |
| `TestResolveImagePaths_MissingImage` | `images_test.go` | Logs warning, leaves path unchanged |
| `TestResolveImagePaths_CollisionAvoidance` | `images_test.go` | Adds numeric suffix on filename collision |
| `TestRenderMermaidBlocks_Basic` | `mermaid_test.go` | Replaces block with image embed |
| `TestRenderMermaidBlocks_MultipleBlocks` | `mermaid_test.go` | Handles N blocks, generates N images |
| `TestRenderMermaidBlocks_EmptyBlock` | `mermaid_test.go` | Leaves empty blocks unchanged |
| `TestRenderMermaidBlocks_Disabled` | `mermaid_test.go` | Returns body unchanged when config is nil |
| `TestRenderMermaidBlocks_MmdcNotFound` | `mermaid_test.go` | Warns and returns body unchanged |

### 9.2 Integration tests

These tests require `mmdc` and `pandoc` to be installed. Skip with `t.Skip()` if unavailable.

| Test | What it validates |
|------|-------------------|
| `TestConvertWithImage` | Full pipeline: Markdown with `![](img.png)` → PDF with embedded image |
| `TestConvertWithMermaid` | Full pipeline: Markdown with Mermaid block → PDF with rendered diagram |
| `TestConvertWithMermaidAndImages` | Full pipeline: Markdown with both features → PDF |
| `TestBundleWithImages` | Bundle mode with per-file image resolution |
| `TestBundleWithMermaid` | Bundle mode with per-file Mermaid rendering |

### 9.3 Manual validation

After implementation, validate with real-world Markdown files:

1. The remarquee README itself (which contains Mermaid in its docs).
2. A design document with architecture diagrams.
3. A bundle of multiple docs, some with images and some with Mermaid.

---

## 10. Risks, Alternatives, and Open Questions

### 10.1 Risks

1. **mmdc startup time:** Puppeteer (used by mmdc) can take 2-5 seconds per diagram. For documents with many Mermaid blocks, this adds up. *Mitigation:* mmdc supports `--puppeteerConfigFile` for persistent browser instances. Consider adding this optimization in a follow-up.

2. **mmdc not installed:** Many users won't have mmdc. *Mitigation:* Mermaid rendering is opt-in with a graceful fallback. The `--mermaid` flag defaults to `true`, but the code checks for mmdc availability and degrades gracefully.

3. **Image path edge cases:** Symlinks, Unicode filenames, Windows paths (if ever ported). *Mitigation:* Use `filepath.Abs()` and `filepath.EvalSymlinks()` for robust resolution. Add tests for edge cases.

4. **Large images:** High-resolution images can bloat PDFs. *Mitigation:* Consider adding an `--image-max-width` flag in a follow-up to downscale large images.

5. **Mermaid syntax errors:** Invalid Mermaid source will cause mmdc to fail. *Mitigation:* Per-block error handling — one bad diagram doesn't break the whole document.

### 10.2 Alternatives considered

1. **Pandoc Lua filter for Mermaid:** Instead of preprocessing in Go, use a Lua filter that calls mmdc. *Rejected:* More complex, harder to debug, doesn't handle the image path problem.

2. **Server-side Mermaid rendering:** Send Mermaid source to a cloud API. *Rejected:* Privacy concerns, requires network access, adds latency.

3. **Go-native Mermaid parser/renderer:** Use a Go library to render Mermaid. *Rejected:* No mature Go Mermaid library exists. mmdc is the standard tool.

4. **SVG output instead of PNG:** mmdc can output SVG. *Rejected:* SVG embedding in LaTeX/PDF is fragile (requires `inkscape` or `rsvg-convert`). PNG is universally supported.

### 10.3 Open questions

1. **Should we add a `--mermaid-width` flag?** Mermaid diagrams default to auto-width, which can be very wide. A max-width constraint might improve PDF readability. Decision: **Yes, add `--mermaid-width` flag in Phase 2.**

2. **Should `upload src` get Mermaid support?** Decision: **No.** Source code files don't contain Mermaid blocks. Out of scope.

3. **Bundle mode image deduplication:** If two Markdown files reference the same image (same path, same content), should we copy it once or twice? Decision: **No dedup.** Copy twice with prefixed names. Simpler, debuggable, and the disk cost is negligible.

4. **mmdc configuration file support:** Should we support a `.mermaidrc` or `mermaid.json` config file? Decision: not in Phase 1. CLI flags are sufficient.

---

## 11. API Reference

### 11.1 New exported functions

| Function | Package | Signature |
|----------|---------|-----------|
| `ResolveImagePaths` | `mdpdf` | `(body string, sourceDir string, tmpDir string) (string, error)` |
| `RenderMermaidBlocks` | `mdpdf` | `(ctx context.Context, body string, tmpDir string, config *MermaidRendererConfig) (string, error)` |
| `DefaultMermaidRendererConfig` | `mdpdf` | `() MermaidRendererConfig` |

### 11.2 Modified exported functions

| Function | Change |
|----------|--------|
| `ConvertMarkdownFileToPDF` | No signature change; reads `opts.Mermaid` internally |
| `BuildBundleMarkdown` | Signature changes to accept `ctx context.Context` and `mermaidCfg *MermaidRendererConfig` |
| `DefaultPandocOptions` | Returns options with `Mermaid: nil` (disabled by default for backward compat) |

### 11.3 New CLI flags

| Flag | Commands | Type | Default |
|------|----------|------|---------|
| `--mermaid` | `upload md`, `upload bundle` | bool | true |
| `--mmdc-path` | `upload md`, `upload bundle` | string | "" |
| `--mermaid-scale` | `upload md`, `upload bundle` | int | 2 |
| `--mermaid-theme` | `upload md`, `upload bundle` | string | "default" |
| `--mermaid-bg` | `upload md`, `upload bundle` | string | "white" |
| `--resolve-images` | `upload md`, `upload bundle` | bool | true |

---

## 12. File Reference

### 12.1 Existing files that will be modified

| File | Change description |
|------|-------------------|
| `pkg/mdpdf/pandoc.go` | Add `Mermaid` field to `PandocOptions`. Add `ResolveImagePaths` and `RenderMermaidBlocks` calls in `ConvertMarkdownFileToPDF`. |
| `pkg/mdpdf/bundle.go` | Update `BuildBundleMarkdown` to accept context and Mermaid config. Add per-file image/Mermaid processing. |
| `cmd/remarquee/cmds/upload/md.go` | Add Mermaid and image CLI flags. Wire into `PandocOptions`. |
| `cmd/remarquee/cmds/upload/bundle.go` | Add Mermaid and image CLI flags. Update `BuildBundleMarkdown` call. |

### 12.2 New files

| File | Description |
|------|------------|
| `pkg/mdpdf/images.go` | `ResolveImagePaths` implementation |
| `pkg/mdpdf/images_test.go` | Unit tests for image path resolution |
| `pkg/mdpdf/mermaid.go` | `RenderMermaidBlocks`, `MermaidRendererConfig`, helper functions |
| `pkg/mdpdf/mermaid_test.go` | Unit tests for Mermaid rendering |

---

## 13. Glossary

| Term | Definition |
|------|-----------|
| **mmdc** | Mermaid CLI (`@mermaid-js/mermaid-cli`). A Node.js command-line tool that renders Mermaid diagrams to PNG/SVG using a headless browser (Puppeteer). |
| **Mermaid** | A text-based diagramming language. You write `graph TD; A-->B` and it renders a flowchart. |
| **pandoc** | The universal document converter. remarquee uses it to convert Markdown → PDF via LaTeX. |
| **xelatex** | A LaTeX engine that supports Unicode and system fonts. Used by pandoc as the PDF backend. |
| **Puppeteer** | A Node.js library that controls a headless Chrome/Chromium browser. Used by mmdc under the hood. |
| **pkg/mdpdf** | The Go package in remarquee that handles Markdown-to-PDF conversion. |
| **tmpDir** | A temporary directory (created via `os.MkdirTemp`) where remarquee writes preprocessed Markdown and image files before invoking pandoc. |
| **bundle mode** | The `remarquee upload bundle` command that concatenates multiple Markdown files into one PDF with a table of contents. |
| **YAML frontmatter** | A `---`-delimited block at the top of a Markdown file containing metadata. Removed during preprocessing. |
