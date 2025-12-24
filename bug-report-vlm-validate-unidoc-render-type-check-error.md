---
Title: "Bug report: vlm-validate UniDoc rasterizer fails with 'type check error' on some PDFs"
Ticket: RMQ-0006
Status: active
Created: 2025-12-24
Owners: []
RelatedFiles:
  - Path: cmd/remarquee/cmds/rmdoc/vlm_validate.go
    Note: VLM helper that rasterizes PDF pages before calling pinocchio
  - Path: pkg/pdfcmp/pdfcmp.go
    Note: Uses UniDoc renderer too; similar failure modes may apply
  - Path: cmd/remarquee/cmds/rmdoc/render_v6.go
    Note: Produces the PDF we used as input A
ExternalSources: []
Summary: UniDoc's PDF renderer errors with "type check error" when vlm-validate tries to render page(s) to PNG, blocking VLM runs. Workaround is to rasterize with Poppler (pdftoppm).
---

## Summary

The helper command `remarquee rmdoc vlm-validate` is intended to:

1. Rasterize selected PDF pages to PNGs
2. Call `pinocchio code professional --images <pngs> "<prompt>"`

However, rasterization using UniDoc (`render.NewImageDevice().Render(page)`) fails for at least one real-world PDF with:

> `render page 1: type check error`

This prevents PNG creation, so pinocchio never runs.

## Environment

- **OS**: Linux
- **Go**: `remarquee/go.mod` uses `github.com/unidoc/unipdf/v3 v3.6.1`
- **pinocchio**: present on PATH (`/home/manuel/go/bin/pinocchio`)
- **remarks**: installed via Poetry+pyenv (Python 3.12.3), used to generate a reference PDF
- **Poppler**: `pdftoppm` available (`/usr/bin/pdftoppm`, version 24.02.0)

## Reproduction

1) Produce PDFs:

- A: Go renderer output (example):
  - `go run ./cmd/remarquee rmdoc render-v6 ./cmd/remarquee-ui/testdata/Test.rmdoc --out /tmp/remarquee-Test-v6.pdf --force`
- B: remarks output (example):
  - `remarks ./cmd/remarquee-ui/testdata/Test.rmdoc /tmp/remarks-out --log_level ERROR`
  - which yields `/tmp/remarks-out/Test _remarks.pdf`

2) Run VLM validation with UniDoc rasterizer:

```bash
go run ./cmd/remarquee rmdoc vlm-validate \
  --pdf-b /tmp/remarks-out/Test\\ _remarks.pdf \
  --pages 1,2 \
  /tmp/remarquee-Test-v6.pdf
```

Observed error:

> `Error: render page 1: type check error`

## What this error likely means

UniDoc uses a strict PDF object model. During rendering it parses page resources/content streams and resolves PDF objects. A **"type check error"** generally indicates that a PDF object encountered at runtime was **not of the expected type** (for example, a name was expected but an array was found; a dictionary was expected but got a stream; etc.).

This can happen if:

- the PDF is malformed (or has edge-case constructs)
- UniDoc’s renderer does not support a specific PDF feature used in the document
- there is a bug/limitation in UniDoc’s type checker / content processing

## Impact

- Blocks `vlm-validate` from generating images (PNG), so we cannot run pinocchio/VLM checks.
- Could also affect any future “render PDF to image” utilities that rely on UniDoc’s renderer.

## Workaround / Mitigation

Use Poppler’s rasterizer (`pdftoppm`) to render pages to PNG instead of UniDoc.

This is already present on the system and is the standard PDF→image toolchain on Linux.

## Proposed fix

- Change `vlm-validate` to **default to Poppler** (`pdftoppm`) rasterization.
- Keep UniDoc rasterization as an optional fallback (debug mode).
- Add flags:
  - `--rasterizer poppler|unidoc`
  - `--dpi N`
  - `--pdftoppm /path/to/pdftoppm`

## Notes / Follow-ups

- If we want to fix UniDoc rendering, we should isolate a minimal PDF that triggers the error and open an upstream issue (if possible) with the exact failing object trace.
- For RMQ-0006’s purposes, **Poppler rasterization is sufficient** for VLM and for producing “human inspection” PNGs.


