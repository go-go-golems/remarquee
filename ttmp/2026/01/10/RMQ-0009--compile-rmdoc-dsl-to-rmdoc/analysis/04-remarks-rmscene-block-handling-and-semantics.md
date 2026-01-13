---
Title: Remarks + rmscene block handling and semantics
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
    - Path: remarks/remarks/conversion/parsing.py
      Note: Remarks V6 parse flow and output metadata
    - Path: rmscene/src/rmscene/scene_items.py
      Note: Line/GlyphRange/Text models and color semantics
    - Path: rmscene/src/rmscene/scene_stream.py
      Note: Block classes
    - Path: rmscene/src/rmscene/scene_tree.py
      Note: SceneTree structure and add_item semantics
    - Path: rmscene/src/rmscene/tagged_block_common.py
      Note: Tag encoding and DataStream helpers
    - Path: rmscene/src/rmscene/tagged_block_reader.py
      Note: Block framing and tag parsing rules
    - Path: rmscene/tests/test_scene_stream.py
      Note: Reference block ordering in tests
ExternalSources: []
Summary: "Detailed mapping of RMV6 block parsing in rmscene and how remarks uses SceneTree, text, and glyph ranges."
LastUpdated: 2026-01-10T21:35:05-05:00
WhatFor: "Guide RMDoc-DSL compiler decisions by matching reader expectations for block structure and semantics."
WhenToUse: "When adding RMV6 block writers, investigating device render errors, or extending highlights/text."
---


# Remarks + rmscene block handling and semantics

## Goal

This document captures how the Python `remarks` toolchain and the `rmscene` parser library read RMV6 `.rm` files, what each block type is used for, and how block data becomes a `SceneTree` plus typed text/highlight data. The intent is to make the RMQ-0009 compiler writer more compatible with what existing readers expect, with specific file references, symbols, and pseudocode for the read path.

## Key files and symbols

- `rmscene/src/rmscene/tagged_block_common.py`
  - `TagType`, `CrdtId`, `DataStream.read_header`, `DataStream.read_tag`
- `rmscene/src/rmscene/tagged_block_reader.py`
  - `TaggedBlockReader.read_block`, `read_subblock`, `bytes_remaining_in_block`
- `rmscene/src/rmscene/scene_stream.py`
  - `Block.read`, `UnreadableBlock`, per-block classes like `SceneTreeBlock`, `TreeNodeBlock`, `SceneLineItemBlock`, `RootTextBlock`
  - `read_blocks`, `build_tree`, `simple_text_document`
- `rmscene/src/rmscene/scene_tree.py`
  - `SceneTree`, `SceneTree.add_node`, `SceneTree.add_item`, `SceneTree.walk`
- `rmscene/src/rmscene/scene_items.py`
  - `Group`, `Line`, `GlyphRange`, `Text`, `Pen`, `PenColor`, `ParagraphStyle`
- `remarks/remarks/conversion/parsing.py`
  - `parse_v6`, `determine_document_dimensions`, `parse_rm_file`

## High-level parsing flow

The flow in `remarks` for V6 `.rm` files uses `rmscene` to parse blocks, then constructs a `SceneTree` and extracts highlights and typed text.

Pseudocode (actual names retained):

```python
# remarks/remarks/conversion/parsing.py

def parse_v6(file_path):
    output = {
        "highlights": [],
        "glyph_ranges": [],
        "text": None,
        "scene_tree": None,
    }
    with open(file_path, "rb") as f:
        blocks = list(read_blocks(f))
        tree = SceneTree()
        build_tree(tree, blocks)
        output["scene_tree"] = tree

        for block in blocks:
            if isinstance(block, RootTextBlock):
                output["text"] = {
                    "pos_x": block.value.pos_x,
                    "pos_y": block.value.pos_y,
                    "width": block.value.width,
                    "text": TextDocument.from_scene_item(tree.root_text),
                }

        for item in tree.walk():
            if isinstance(item, GlyphRange):
                rectangles = [translate(rect) for rect in item.rectangles]
                output["glyph_ranges"].append(item)
                output["highlights"].append(RectangleList(color=item.color, rectangles=rectangles))

    return output, False
```

Key takeaways:
- `read_blocks` parses raw blocks using the typed block classes in `scene_stream.py`.
- `build_tree` attaches items to a CRDT-based `SceneTree` using `SceneTreeBlock`, `TreeNodeBlock`, and `SceneItemBlock` variants.
- `remarks` only extracts text from `RootTextBlock` and highlights from `GlyphRange`; it does not consume line strokes for output in `parse_v6`.
- `determine_document_dimensions` separately walks `Line` items to compute page bounds.

## Block framing and tag encoding (rmscene)

The block framing and tag encoding logic lives in `rmscene/src/rmscene/tagged_block_common.py` and `rmscene/src/rmscene/tagged_block_reader.py`.

Block header structure (as read by `TaggedBlockReader.read_block`):
- `uint32 block_length`
- `uint8 unknown` (asserted to be 0)
- `uint8 min_version`
- `uint8 current_version`
- `uint8 block_type`
- then block payload bytes

Tag encoding details (from `TagType` and `DataStream.read_tag`):
- tags are varuint values encoding `index` and `tag_type`:
  - `index = tag >> 4`
  - `tag_type = tag & 0xF`
- `TagType` values:
  - `ID = 0xF` (CRDT ids)
  - `Length4 = 0xC` (subblock length)
  - `Byte8 = 0x8` (float64)
  - `Byte4 = 0x4` (float32/uint32)
  - `Byte1 = 0x1` (byte/bool)

`TaggedBlockReader` enforces block length at the end of each `read_block` and `read_subblock` context, and raises `UnexpectedBlockError` when tags do not match the expected index or type.

## Block types and semantics (rmscene)

`rmscene/src/rmscene/scene_stream.py` defines block classes mapped by block type. Parsing is driven by `Block.read`, which uses `Block.lookup` to pick a subclass, and falls back to `UnreadableBlock` when unknown or parse errors occur.

Top-level block types and their roles:

- `0x00 MigrationInfoBlock`
  - `migration_id: CrdtId`
  - `is_device: bool`
  - optional `_unknown` flag (written for versions >= 3.2.2)
  - establishes CRDT migration origin and device flag
- `0x01 SceneTreeBlock`
  - `tree_id`, `node_id`, `parent_id`, `is_update`
  - used to create node slots in the tree before `TreeNodeBlock` fills in labels/visibility
- `0x02 TreeNodeBlock`
  - `Group` properties: `label`, `visible`, and optional anchor metadata
  - anchor fields: `anchor_id`, `anchor_type`, `anchor_threshold`, `anchor_origin_x`
- `0x03 SceneGlyphItemBlock`
  - `ITEM_TYPE = 0x01`
  - holds `GlyphRange` (highlight rectangles + text + color)
- `0x04 SceneGroupItemBlock`
  - `ITEM_TYPE = 0x02`
  - value is a `CrdtId` pointing at a group node; used to attach groups to a parent
- `0x05 SceneLineItemBlock`
  - `ITEM_TYPE = 0x03`
  - holds `Line` data (points, tool, color, highlight color tail marker)
- `0x06 SceneTextItemBlock`
  - `ITEM_TYPE = 0x05`
  - parser is stubbed; currently no text payload logic
- `0x07 RootTextBlock`
  - typed text with CRDT sequence items + paragraph styles + position
- `0x08 SceneTombstoneItemBlock`
  - tombstone, value read/write stubbed
- `0x09 AuthorIdsBlock`
  - maps small author ids to UUIDs (UUID stored in little-endian bytes)
- `0x0A PageInfoBlock`
  - counts for loads, merges, text chars, text lines, optional `type_folio_use_count`
- `0x0D SceneInfo`
  - LWW values: `current_layer`, `background_visible`, `root_document_visible`, optional `paper_size`

`SceneItemBlock` shared structure:
- `parent_id`, `item_id`, `left_id`, `right_id`, `deleted_length`
- value is stored in subblock index 6 with a leading `item_type` byte
- the CRDT sequence information (left/right/deleted length) controls ordering and deletions

## SceneTree assembly semantics

`rmscene/src/rmscene/scene_tree.py` and `build_tree` (`scene_stream.py`) define how the blocks are converted into a tree and sequence of items.

Pseudocode (simplified):

```python
# rmscene/src/rmscene/scene_stream.py

def build_tree(tree, blocks):
    for b in blocks:
        if isinstance(b, SceneTreeBlock):
            tree.add_node(b.tree_id, parent_id=b.parent_id)
        elif isinstance(b, TreeNodeBlock):
            node = tree[b.group.node_id]
            node.label = b.group.label
            node.visible = b.group.visible
            node.anchor_id = b.group.anchor_id
            node.anchor_type = b.group.anchor_type
            node.anchor_threshold = b.group.anchor_threshold
            node.anchor_origin_x = b.group.anchor_origin_x
        elif isinstance(b, SceneGroupItemBlock):
            node_id = b.item.value
            item = replace(b.item, value=tree[node_id])
            tree.add_item(item, b.parent_id)
        elif isinstance(b, (SceneLineItemBlock, SceneGlyphItemBlock)):
            tree.add_item(b.item, b.parent_id)
        elif isinstance(b, RootTextBlock):
            tree.root_text = b.value
```

Important behavior details:
- `SceneTree.add_node` only registers the node id; the parent relation is not stored in the tree in this implementation.
- `SceneGroupItemBlock` is the step that actually attaches a group node under a parent in the CRDT sequence.
- `SceneTree.walk` yields only leaf items, not groups, so highlights and lines are extracted via the `SceneItemBlock` entries.
- `RootTextBlock` is stored separately as `SceneTree.root_text` and is not part of the normal item walk.

## Remarks-specific use of blocks

The `remarks` code uses these blocks as follows:

- `RootTextBlock` is converted into `TTextBlock` in `parse_v6`.
  - `TextDocument.from_scene_item(tree.root_text)` builds higher level text content.
  - `pos_x`, `pos_y`, `width` are exposed for layout.
- `SceneGlyphItemBlock` provides `GlyphRange` items in the tree, which are converted into per-rectangle highlight regions.
- `SceneLineItemBlock` is not used directly by `parse_v6`, but it is used by `determine_document_dimensions` to expand min/max bounds.
- `SceneTreeBlock` and `TreeNodeBlock` are required for `build_tree` to attach items; missing nodes cause `ValueError`.
- `AuthorIdsBlock`, `MigrationInfoBlock`, `PageInfoBlock`, `SceneInfo` are parsed by `rmscene` but not consumed by `remarks` directly.

## Observed expectations from rmscene tests

`rmscene/tests/test_scene_stream.py` (see `test_normal_ab`) shows a canonical order for a typed-text example:

1. `AuthorIdsBlock`
2. `MigrationInfoBlock`
3. `PageInfoBlock`
4. `SceneTreeBlock` (root tree)
5. `RootTextBlock`
6. `TreeNodeBlock` (root group)
7. `TreeNodeBlock` (layer group)
8. `SceneGroupItemBlock` (attach layer)

This is useful as a minimal baseline for `.rm` files containing only text, and demonstrates the structural dependency that `SceneTreeBlock` and `TreeNodeBlock` must exist before `SceneGroupItemBlock` is applied.

## Implications for RMQ-0009 (.rmdoc compiler)

- Readers like `rmscene` and `remarks` expect a specific set of top-level blocks to exist, even if the current consumer does not use them directly.
- `SceneTreeBlock` and `TreeNodeBlock` must be emitted before `SceneGroupItemBlock` if you want `build_tree` to succeed.
- Missing `AuthorIdsBlock`, `MigrationInfoBlock`, `PageInfoBlock`, or `SceneInfo` can still parse in `rmscene`, but device-side behavior has been observed to reject files without them.
- If future RMQ-0009 work adds highlights or typed text, the required block types are `SceneGlyphItemBlock` and `RootTextBlock` (plus `SceneTextItemBlock` if inline text items are ever introduced).
- If only strokes are present, `SceneLineItemBlock` plus a valid tree/group structure is sufficient for `rmscene` and `remarks` to extract strokes and build a tree.
