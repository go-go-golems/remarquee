---
Title: RMV6 block semantics in rmscene + Go parser
Ticket: RMQ-0009
Status: active
Topics:
    - remarkable
    - rmdoc
    - rendering
    - dsl
    - compiler
    - go
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarquee/pkg/rmdoc/rmv6_scene_item_block.go
      Note: Go scene item parsing
    - Path: remarquee/pkg/rmdoc/rmv6_scene_tree.go
      Note: Go scene tree parser behavior
    - Path: remarquee/pkg/rmdoc/rmv6_tagged_block_reader.go
      Note: Go block reader implementation
    - Path: remarquee/ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/scripts/04-dump-rm-blocks/main.go
      Note: Block dump script used to compare device vs compiled
    - Path: rmscene/src/rmscene/scene_stream.py
      Note: Defines RMV6 block types and build_tree
    - Path: rmscene/src/rmscene/tagged_block_common.py
      Note: Tag types and data stream encoding
    - Path: rmscene/src/rmscene/tagged_block_writer.py
      Note: Reference writer implementation
ExternalSources: []
Summary: Deep-dive on RMV6 block types in rmscene and the Go parser, with implications for the DSL -> .rmdoc compiler.
LastUpdated: 2026-01-10T21:19:12.464037772-05:00
WhatFor: Explain RMV6 block semantics and why missing blocks cause device render failures.
WhenToUse: When implementing or debugging the RMV6 writer in RMQ-0009.
---


# RMV6 block semantics in rmscene + Go parser

## Goal

Document how RMV6 block types are defined and consumed in `rmscene` (Python) and the Go parser, and highlight which blocks are required for device acceptance. This is intended to guide RMQ-0009 compiler changes after the device reported “unable to render document.”

## Context

The compiled `.rmdoc` notebooks currently fail to render on-device even for the empty notebook case. A device-authored empty notebook (`device-empty-1page`) contains RMV6 blocks that our compiler does not emit. This doc spells out what those blocks are, where they are defined, and how they are used by parsers/renderers.

## TL;DR (what matters most)

- The device empty `.rm` includes **AuthorIdsBlock**, **MigrationInfoBlock**, **PageInfoBlock**, **SceneInfo**, and a **TreeNodeBlock for the root group**. Our compiler emits none of these.
- `rmscene` parses these blocks explicitly and writes them in its reference generators (e.g., `simple_text_document`).
- The Go parser (`ParseRMV6SceneTree`) ignores these blocks entirely; it still parses our `.rm` locally, but the **device likely requires them** for notebook render acceptance.

## RMV6 format framing (shared by rmscene + Go)

### Header + blocks

The RMV6 `.rm` file is:

1) A fixed header: `reMarkable .lines file, version=6          `
2) A sequence of **top-level blocks**, each encoded as:

```text
uint32 length
uint8  unknown (0)
uint8  min_version
uint8  current_version
uint8  block_type
bytes  payload[length]
```

### Tagged fields in block payloads

Within block payloads, values are written as **tagged fields**:

```text
tag = varuint(index << 4 | tag_type)
data = encoded value
```

Common tag types (`rmscene.tagged_block_common.TagType`):

- `0xF` ID (CRDT id: uint8 + varuint)
- `0xC` Length4 subblock (uint32 length + raw bytes)
- `0x8` Byte8 (float64)
- `0x4` Byte4 (uint32 or float32)
- `0x1` Byte1 (uint8/bool)

References:

- `rmscene/src/rmscene/tagged_block_common.py` (`TagType`, `DataStream`)
- `rmscene/src/rmscene/tagged_block_writer.py` (`TaggedBlockWriter`)
- `remarquee/pkg/rmdoc/rmv6_tagged_block_reader.go` (Go reader)

## RMV6 block types (rmscene)

### AuthorIdsBlock (0x09)

**Purpose:** map author IDs to UUIDs. Device files always include this.

Implementation:

- File: `rmscene/src/rmscene/scene_stream.py`
- Symbol: `AuthorIdsBlock`
- Format:
  - varuint count
  - repeated subblock(0):
    - varuint len (16)
    - uuid bytes (little-endian)
    - uint16 author_id

Pseudocode:

```python
writer.write_varuint(len(author_ids))
for id, uuid in author_ids:
    with writer.write_subblock(0):
        writer.write_varuint(16)
        writer.write_bytes(uuid.bytes_le)
        writer.write_uint16(id)
```

### MigrationInfoBlock (0x00)

**Purpose:** migration metadata for device sync/format changes.

Implementation:

- File: `rmscene/src/rmscene/scene_stream.py`
- Symbol: `MigrationInfoBlock`
- Fields:
  - tag 1: `migration_id` (CrdtId)
  - tag 2: `is_device` (bool)
  - tag 3: `_unknown` (bool, used for versions >= 3.2.2)

### PageInfoBlock (0x0A)

**Purpose:** page stats counters (loads, merges, text counts).

Implementation:

- File: `rmscene/src/rmscene/scene_stream.py`
- Symbol: `PageInfoBlock`
- Fields:
  - tag 1: `loads_count`
  - tag 2: `merges_count`
  - tag 3: `text_chars_count`
  - tag 4: `text_lines_count`
  - tag 5: `type_folio_use_count` (optional, version >= 3.2.2)

### SceneInfo (0x0D)

**Purpose:** global per-page display state (current layer, background visibility, page size).

Implementation:

- File: `rmscene/src/rmscene/scene_stream.py`
- Symbol: `SceneInfo`
- Fields:
  - tag 1: `current_layer` (LWW id)
  - tag 2: `background_visible` (LWW bool)
  - tag 3: `root_document_visible` (LWW bool)
  - tag 5: `paper_size` (int pair)

### SceneTreeBlock (0x01)

**Purpose:** declare a scene group node and its parent.

Implementation:

- File: `rmscene/src/rmscene/scene_stream.py`
- Symbol: `SceneTreeBlock`
- Fields:
  - tag 1: `tree_id` (node id)
  - tag 2: `node_id` (unused in practice; often CrdtId(0,0))
  - tag 3: `is_update` (bool)
  - subblock(4): parent_id (tag 1)

**Note:** In device empty notebook, `tree_id=CrdtId(0,11)`, `parent_id=CrdtId(0,1)`.

### TreeNodeBlock (0x02)

**Purpose:** attach metadata to a node (label, visibility, anchors).

Implementation:

- File: `rmscene/src/rmscene/scene_stream.py`
- Symbol: `TreeNodeBlock`
- Fields:
  - tag 1: `node_id`
  - tag 2: `label` (LWW string)
  - tag 3: `visible` (LWW bool)
  - optional tags 7–10 for anchors

### SceneGroupItemBlock (0x04)

**Purpose:** add a layer/group as a child of another group via CRDT sequence.

Implementation:

- File: `rmscene/src/rmscene/scene_stream.py`
- Symbol: `SceneGroupItemBlock`
- Fields:
  - tag 1: `parent_id`
  - tag 2: `item_id`
  - tag 3: `left_id`
  - tag 4: `right_id`
  - tag 5: `deleted_length`
  - subblock(6):
    - byte: item_type (0x02)
    - tag 2: `node_id` (group id)

### SceneLineItemBlock (0x05)

**Purpose:** add a line (stroke) to a group via CRDT sequence.

Implementation:

- File: `rmscene/src/rmscene/scene_stream.py`
- Symbol: `SceneLineItemBlock`
- value encoding: `line_to_stream` (tool/color/points, highlight marker)

## rmscene parsing behavior (how these blocks are used)

Key flow:

- `read_blocks()` yields all blocks, including unknown ones (`UnreadableBlock`).
- `build_tree()` consumes:
  - `SceneTreeBlock` to add nodes
  - `TreeNodeBlock` to enrich node metadata
  - `SceneItemBlock` (group/line/glyph/text) to add children
  - `RootTextBlock` for text layout
  - **Everything else is ignored** but still present in the data stream.

References:

- `rmscene/src/rmscene/scene_stream.py`:
  - `read_blocks`
  - `build_tree`
  - `SceneTreeBlock`, `TreeNodeBlock`, `SceneGroupItemBlock`, `SceneLineItemBlock`

## Go parser behavior (current)

### RMV6TaggedBlockReader

Go reader decodes header + block boundaries + tagged fields:

- File: `remarquee/pkg/rmdoc/rmv6_tagged_block_reader.go`
- Key types: `rmV6TaggedBlockReader`, `rmV6MainBlockInfo`, `rmV6SubBlockInfo`

### ParseRMV6SceneTree

Go parser currently reads:

- `SceneTreeBlock` (0x01)
- `TreeNodeBlock` (0x02)
- `RootTextBlock` (0x07)
- `SceneItemBlock` types 0x03..0x08

Everything else is ignored (no error), which is why compiled notebooks parse locally even when devices reject them.

References:

- `remarquee/pkg/rmdoc/rmv6_scene_tree.go` (`ParseRMV6SceneTree`)
- `remarquee/pkg/rmdoc/rmv6_scene_item_block.go`
- `remarquee/pkg/rmdoc/rmv6_root_text.go`

## Device empty notebook vs compiled empty (observed)

### Device empty `.rm` block list (from rmscene)

```
AuthorIdsBlock (0x09)
MigrationInfoBlock (0x00)
PageInfoBlock (0x0A)
SceneInfo (0x0D)
SceneTreeBlock (0x01) tree_id=CrdtId(0,11) node_id=CrdtId(0,0) is_update=True parent_id=CrdtId(0,1)
TreeNodeBlock (0x02) node_id=CrdtId(0,1) label="" visible=true
TreeNodeBlock (0x02) node_id=CrdtId(0,11) label="Layer 1" visible=true
SceneGroupItemBlock (0x04) parent=CrdtId(0,1) value=CrdtId(0,11)
```

### Compiled empty `.rm` block list (current)

```
SceneTreeBlock (0x01)
TreeNodeBlock (0x02) [layer only]
SceneGroupItemBlock (0x04)
```

### Key gaps

- Missing AuthorIdsBlock, MigrationInfoBlock, PageInfoBlock, SceneInfo.
- Missing TreeNodeBlock for root group (CrdtId(0,1)).
- Different CRDT id scheme (`tree_id` not aligned to device’s CrdtId(0,11) pattern).

## Implications for RMQ-0009 compiler

1) **Add missing required blocks**:
   - AuthorIdsBlock (0x09)
   - MigrationInfoBlock (0x00)
   - PageInfoBlock (0x0A)
   - SceneInfo (0x0D)
   - TreeNodeBlock for root group (CrdtId(0,1))

2) **Align CRDT id patterns**:
   - Use root group id = CrdtId(0,1).
   - Use first layer id = CrdtId(0,11), and group item id = CrdtId(0,13) (as observed).

3) **Match SceneTree defaults**:
   - `node_id = CrdtId(0,0)`
   - `is_update = true`

4) **Match SceneInfo defaults**:
   - `current_layer = LWW(CrdtId(0,0) -> CrdtId(0,0))`
   - `background_visible = true`
   - `root_document_visible = true`
   - `paper_size = (1620, 2160)` (observed in device empty)

## Pseudocode: device-like empty `.rm` emission

```go
write_header()

AuthorIdsBlock({
  1: device_or_generated_uuid,
})

MigrationInfoBlock(migration_id=CrdtId(1,1), is_device=true)

PageInfoBlock(loads=1, merges=0, text_chars=0, text_lines=0)

SceneInfo(
  current_layer=LWW(CrdtId(0,0), CrdtId(0,0)),
  background_visible=LWW(CrdtId(0,0), true),
  root_document_visible=LWW(CrdtId(0,0), true),
  paper_size=(1620,2160),
)

SceneTreeBlock(tree_id=CrdtId(0,11), node_id=CrdtId(0,0), is_update=true, parent=CrdtId(0,1))

TreeNodeBlock(node_id=CrdtId(0,1), label="", visible=true)
TreeNodeBlock(node_id=CrdtId(0,11), label="Layer 1", visible=true)

SceneGroupItemBlock(parent=CrdtId(0,1), item_id=CrdtId(0,13), left=0, right=0, value=CrdtId(0,11))
```

## References (files + symbols)

- `rmscene/src/rmscene/scene_stream.py`
  - `AuthorIdsBlock`, `MigrationInfoBlock`, `PageInfoBlock`, `SceneInfo`
  - `SceneTreeBlock`, `TreeNodeBlock`, `SceneGroupItemBlock`, `SceneLineItemBlock`
  - `read_blocks`, `build_tree`, `simple_text_document`
- `rmscene/src/rmscene/tagged_block_common.py` (`DataStream`, `TagType`)
- `rmscene/src/rmscene/tagged_block_writer.py` (`TaggedBlockWriter`)
- `remarquee/pkg/rmdoc/rmv6_tagged_block_reader.go` (Go reader)
- `remarquee/pkg/rmdoc/rmv6_scene_tree.go` (`ParseRMV6SceneTree`)
