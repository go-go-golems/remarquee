# Tasks

## Phase 1: Image Path Resolution ✅

- [x] Create `pkg/mdpdf/images.go` with `ResolveImagePaths` function
- [x] Create `pkg/mdpdf/images_test.go` with unit tests
- [x] Wire `ResolveImagePaths` into `ConvertMarkdownFileToPDF` in `pkg/mdpdf/pandoc.go`
- [x] Add integration test in `pkg/mdpdf/pandoc_test.go`
- [x] Commit Phase 1 (6a960c9)

## Phase 2: Mermaid Block Rendering ✅

- [x] Create `pkg/mdpdf/mermaid.go` with types, `RenderMermaidBlocks`, `renderMermaidToPNG`, `resolveMmdcPath`
- [x] Create `pkg/mdpdf/mermaid_test.go` with unit tests
- [x] Add `MermaidRendererConfig` and `Mermaid` field to `PandocOptions` in `pandoc.go`
- [x] Wire `RenderMermaidBlocks` into `ConvertMarkdownFileToPDF`
- [x] Add `--mermaid-width` flag support (per decision 10.3.1)
- [x] Add integration test (skip if mmdc not installed)
- [x] Commit Phase 2 (162a75c)

## Phase 3: CLI Flag Integration ✅

- [x] Add mermaid + image flags to `upload md` command (`md.go`)
- [x] Add mermaid + image flags to `upload bundle` command (`bundle.go`)
- [x] Wire flags into `PandocOptions` in both commands
- [x] Test CLI flag parsing
- [x] Commit Phase 3 (2ccac3e)

## Phase 4: Bundle Mode Support ✅

- [x] Update `BuildBundleMarkdown` signature to accept `context.Context` and `*MermaidRendererConfig`
- [x] Implement per-file image/Mermaid processing in bundle mode
- [x] Update `writeBundlePDF` caller in `bundle.go`
- [x] Add bundle integration tests
- [x] Commit Phase 4 (5606cb8)

## Phase 5: Documentation ✅

- [x] Update `README.md` with Mermaid and image support mention
- [x] Commit Phase 5 (cd216c6)

## DONE

- [x] Create ticket workspace (RMQ-0016)
- [x] Investigate remarquee mdpdf package architecture
- [x] Write design document (design/01-mermaid-image-embed-design.md)
- [x] Write investigation diary (reference/01-diary.md)
- [x] Record open question decisions (10.3.1: yes, 10.3.2: no, 10.3.3: no)
- [x] Upload design doc bundle to reMarkable
- [x] Phase 1: Image path resolution (6a960c9)
- [x] Phase 2: Mermaid block rendering (162a75c)
- [x] Phase 3: CLI flag integration (2ccac3e)
- [x] Phase 4: Bundle mode support (5606cb8)
- [x] Phase 5: Documentation (cd216c6)
