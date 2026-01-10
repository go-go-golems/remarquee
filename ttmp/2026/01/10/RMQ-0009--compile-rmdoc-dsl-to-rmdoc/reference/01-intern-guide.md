---
Title: 'RMQ-0009 Intern Guide — Compile RMDoc-DSL → .rmdoc'
Ticket: RMQ-0009
Status: active
Topics:
  - remarkable
  - rmdoc
  - dsl
  - compiler
  - go
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: >
  Intern-ready guide: the full context, why this exists, where to change code, and how to
  implement a minimal V6 strokes-only .rmdoc compiler from RMDoc-DSL.
LastUpdated: 2026-01-10
---

# RMQ-0009 Intern Guide — Compile RMDoc-DSL → .rmdoc

This document is meant to be “day 1 ready” for a new contributor. It explains the **why**, the **current state**, the **target**, and gives a step-by-step, implementation-oriented plan with pseudocode, file pointers, and invariants to test.

---

## 1) Why this ticket exists (the “why” in plain language)

In RMQ-0006 we built a powerful debugging system around rendering:

- pixel diffing and goldens (vs `remarks`)
- human-in-the-loop review (`plz-confirm`)
- RMDoc-DSL fixtures (YAML + JS/goja) to generate controlled shapes and sweeps
- an “inverse loop” that can parse a device-authored `.rmdoc` and regenerate it into DSL

However, we still lack one critical capability:

> We cannot yet generate a **real notebook `.rmdoc`** from a DSL case and upload it to the tablet as an **editable notebook**.

Right now, when we want device truth, we upload a **PDF**. PDFs are “view-only”; they do not exercise the device’s native notebook format, and they are easy to confuse with existing screenshots or different fixtures.

The goal of this ticket is to close that loop:

> RMDoc-DSL → compiled `.rmdoc` (zip) → upload to device → editable notebook → screenshot → compare.

This reduces confusion, improves reproducibility, and gives us a stable foundation for future renderer improvements.

---

## 2) Current state (what exists today)

### DSL model + loader

The RMDoc-DSL model and loader already exist:

- `remarquee/pkg/rmdsl/`
  - `LoadFromFile(ctx, path, opts)` loads YAML or JS
  - JS runs in embedded goja and returns a DSL object
  - `Normalize()` fills defaults (canvas space, page ids)

### Rendering from DSL (debug transports)

We can already render DSL to:

- **PNGs** (programmatic grid renderer):
  - `ttmp/.../scripts/18-rmdsl-render-to-png/main.go`
- **PDF** (transport, for device viewing):
  - `ttmp/.../scripts/19-rmdsl-render-to-pdf/main.go`

### Parsing real `.rmdoc` and regenerating DSL

We can parse a V6 `.rmdoc` and emit DSL YAML that replays the strokes:

- `ttmp/.../scripts/21-rmdoc-v6-to-dsl-yaml/main.go`

This is a strong signal that our parser can interpret V6 strokes well enough for round-tripping **strokes**.

### RMDoc parsing primitives

Core parsing lives here:

- `.rmdoc` open/read: `remarquee/pkg/rmdoc/open.go`
- read page `.rm` bytes from archive: `remarquee/pkg/rmdoc/rm_archive_rmfiles.go` (see `ReadRMFileFromArchive`)
- V6 scene parsing: `remarquee/pkg/rmdoc/rmv6_scene_tree.go` (`ParseRMV6SceneTree`)
- V6 line decoding: `remarquee/pkg/rmdoc/rmv6_line_decode.go` (`DecodeRMV6Line`)
- anchor application to strokes: `remarquee/pkg/rmdoc/rmv6_strokes_extract.go` (`ExtractRMV6StrokesWithAnchors`)

---

## 3) Target output (what we need to generate)

We want to generate a `.rmdoc` zip archive that:

- is recognized as a V6 cPages notebook by our tooling (`doc.Schema == SchemaCPages`)
- includes per-page V6 `.rm` files containing a minimal scene tree + line items (strokes)
- includes `.content` with a `cPages.pages[]` array referencing those page IDs
- includes `.metadata` sufficient for cloud/device acceptance

Important: the “minimum viable compiler” should aim for **strokes-only**, and only later add text/templates/highlights.

---

## 4) Where to implement (files/packages)

### Recommended new package

Create a compiler package:

- `remarquee/pkg/rmdsl/compile/` (or `pkg/rmdsl/compiler/`)

Responsibilities:

- Take `rmdsl.Doc` (normalized) and emit:
  - `.content` JSON
  - `.metadata` JSON
  - per-page `.rm` bytes (V6)
  - a zip archive `.rmdoc`

### Recommended command

Add a CLI:

- `remarquee rmdsl compile <case.(yaml|js)> --out <file.rmdoc>`

Implementation can live under:

- `remarquee/cmd/remarquee/cmds/rmdsl/` (new command group)

---

## 5) The hard part: writing V6 `.rm` bytes

### What is a V6 `.rm` file?

It’s a “tagged block” binary format:

- a fixed header string: `"reMarkable .lines file, version=6          "`
- then a sequence of top-level blocks, each with:
  - length (uint32 LE)
  - header fields (unknown=0, min_version, current_version, block_type)
  - payload bytes

Inside payloads, values are encoded with tags:

- tags are varuint where:
  - `index = x >> 4`
  - `tag_type = x & 0xF`
- common tag types:
  - ID (CRDT id): type 0xF, encoded as `uint8 + varuint`
  - Length4 subblocks: type 0xC, followed by uint32 length and raw subpayload
  - Byte1/Byte4/Byte8 (ints/floats)

We already have a reader:

- `remarquee/pkg/rmdoc/rmv6_tagged_block_reader.go`

This is your map for what to write.

We also have the canonical writer in Python in this monorepo:

- `rmscene/src/rmscene/tagged_block_writer.py`
- `rmscene/src/rmscene/tagged_block_common.py`
- `rmscene/src/rmscene/scene_stream.py` (how blocks are serialized)

If you need a ground-truth reference implementation, start there.

---

## 6) Minimal `.rm` writer design (strokes-only)

### Required blocks (hypothesis)

To get `ParseRMV6SceneTree` to see strokes, we need to write blocks that it understands:

- **SceneTreeBlock** (block type 0x01): introduces a node (group) id and parent id
- **TreeNodeBlock** (block type 0x02): enriches a node with metadata (we can keep minimal)
- **SceneLineItemBlock** (block type 0x05): adds a CRDT sequence item whose value is a “line” payload

In the parser (`ParseRMV6SceneTree`), scene items are read via:

- tags 1..5: parent_id, item_id, left_id, right_id, deleted_length
- subblock(6): value bytes, first byte is item_type, remainder is payload

So for line items, we must write subblock(6) payload that matches what `DecodeRMV6Line` expects.

### How to encode line payload

`DecodeRMV6Line(version, payload)` reads:

- tool (tag 1) uint32
- color (tag 2) uint32
- thickness_scale (tag 3) float64
- starting_length (tag 4) float32
- points subblock (tag 5) length4 containing packed point structs
- optional timestamp id (tag 6)
- optional move_id (tag 7)
- optional trailing RGBA marker (bytes, special-case)

For strokes-only, we can choose:

- thickness_scale = 1.0
- starting_length = 0
- points: minimal point struct for the line version in the block header

You must match the point struct size for the chosen line version.
The reader code contains helpers:

- `rmv6PointSerializedSize(lineVersion)`
- `rmv6PointFromStream(...)`

The writer should mirror that format.

---

## 7) Pseudocode: end-to-end compilation

### 7.1 Compile entrypoint

```go
func CompileToRMDoc(ctx context.Context, d *rmdsl.Doc, outPath string) error {
  // 1) Normalize DSL
  if err := rmdsl.Normalize(d); err != nil { ... }

  // 2) Allocate page IDs (UUIDs) for each page
  //    - if DSL already has stable IDs, keep them
  //    - but device expects UUID-ish strings; decide policy

  // 3) Build .content (cPages JSON)
  //    - include pages[] entries with id + template
  //    - minimal required fields only

  // 4) For each page:
  //    - lower shapes to strokes if needed
  //    - create a V6 .rm byte stream:
  //      - header
  //      - scene tree blocks
  //      - line item blocks for each stroke

  // 5) Build .metadata (document metadata JSON)
  //    - visibleName, timestamps, type, etc.

  // 6) Zip:
  //    <doc-uuid>.content
  //    <doc-uuid>.metadata
  //    <doc-uuid>/<page-id>.rm  (or similar structure as seen in existing .rmdoc files)

  // 7) Write outPath
}
```

### 7.2 V6 line item encoding (sketch)

```go
func writeSceneLineItemBlock(w *TaggedBlockWriter, parentNodeID CrdtId, itemIDs...) {
  w.Block(0x05 /*SceneLineItemBlock*/, minVer, curVer, func() {
    w.ID(1, parentNodeID)
    w.ID(2, itemID)
    w.ID(3, leftID)
    w.ID(4, rightID)
    w.Uint32(5, 0) // deleted_length
    w.SubBlock(6, func(sw *SubWriter) {
      sw.ByteRaw(itemTypeLine)        // first byte: item_type
      sw.WriteBytes(linePayloadBytes) // rest: DecodeRMV6Line payload
    })
  })
}
```

The crucial part: the payload bytes after item_type must match `DecodeRMV6Line`.

---

## 8) Upload as an editable notebook (cloud/device integration)

We already have:

- `remarquee cloud put` (uploads a local file to the cloud)
- `remarquee cloud get` (downloads remote doc as `.rmdoc`)

Unknown (to verify in this ticket):

- Does `cloud put <file.rmdoc> <dir>` create a notebook or treat it as a file blob?

So the intern must:

- try uploading a generated `.rmdoc` using `cloud put`
- verify on device:
  - it appears as a notebook
  - it is editable (you can add strokes)
  - it contains the expected content

If it doesn’t:

- we may need a dedicated “upload `.rmdoc` notebook” command that uses rmapi APIs differently (document type / content vs metadata).

---

## 9) Test strategy (how to know you’re correct)

### Unit tests (fast)

- encoding primitives:
  - tag varuint encoding
  - id encoding (uint8 + varuint)
  - subblock boundaries (length correctness)
- line payload:
  - writer emits points that the decoder can read

### Integration tests (most important)

Run:

- compile DSL → `.rmdoc`
- `rmdoc.OpenFile` succeeds
- for each page:
  - `ReadRMFileFromArchive` returns ok
  - `ParseRMV6SceneTree` succeeds
  - `ExtractRMV6StrokesWithAnchors` returns the expected count

Then:

- `remarquee rmdoc render-v6 <compiled.rmdoc> --out /tmp/out.pdf --force`

That gives you confidence the produced notebook is usable by the renderer.

---

## 10) References (where to look)

### In this repo (Go)

- DSL model/loader:
  - `remarquee/pkg/rmdsl/`
- `.rmdoc` reader:
  - `remarquee/pkg/rmdoc/open.go`
- V6 tagged block reader:
  - `remarquee/pkg/rmdoc/rmv6_tagged_block_reader.go`
- V6 scene parsing:
  - `remarquee/pkg/rmdoc/rmv6_scene_tree.go`
  - `remarquee/pkg/rmdoc/rmv6_scene_item_block.go`
- V6 line decoding:
  - `remarquee/pkg/rmdoc/rmv6_line_decode.go`
- Anchor application:
  - `remarquee/pkg/rmdoc/rmv6_strokes_extract.go`

### In this monorepo (Python “ground truth”)

- tagged block writer:
  - `rmscene/src/rmscene/tagged_block_writer.py`
  - `rmscene/src/rmscene/tagged_block_common.py`
- scene stream encoding:
  - `rmscene/src/rmscene/scene_stream.py`

When stuck, open these first. They explain what bytes the device format expects.

---

## 11) “Why this is worth doing” (closing note)

This compiler turns RMDoc-DSL from “debug tooling” into a durable platform:

- future interns can add cases without learning binary formats
- renderer improvements become measurable via deterministic fixture families
- device validation becomes reliable because fixture identity is guaranteed by compilation

This is exactly the kind of leverage that pays off repeatedly as the renderer grows.


