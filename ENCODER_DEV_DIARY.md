# RMDoc Encoder Development Diary

**Date:** January 19, 2026  
**Goal:** Create a working encoder by reverse-engineering from remarquee's decoder

---

## Understanding the Format

### Key Insights from the Decoder

1. **File Structure:**
   ```
   [43-byte header: "reMarkable .lines file, version=6          "]
   [Block 1]
   [Block 2]
   ...
   ```

2. **Block Structure:**
   ```
   uint32 blockLength  (little-endian)
   uint8  unknown (must be 0)
   uint8  minVersion
   uint8  currentVersion
   uint8  blockType
   [blockLength bytes of payload]
   ```

3. **Tagged Values:**
   - Tags are varuint encoded: `(index << 4) | tagType`
   - Tag types:
     - 0xF: ID (CRDT ID: uint8 + varuint)
     - 0xC: Length4 (uint32 length + data)
     - 0x8: Byte8 (8 bytes)
     - 0x4: Byte4 (4 bytes)
     - 0x1: Byte1 (1 byte)

4. **Varuint Encoding:**
   - 7 bits per byte, MSB=1 means more bytes follow
   - Little-endian bit order

### What We Need to Encode

From the Python rmscene library and remarquee's decoder, a minimal .rm file needs:

1. **Header** (43 bytes)
2. **Main Block** containing:
   - Tree ID (tagged)
   - Root Node (subblock) containing:
     - Node type (byte)
     - Children sequence (subblock) containing:
       - Sequence items (CRDT entries) containing:
         - Item ID, Left ID, Right ID
         - Deleted length
         - Line item containing:
           - Tool, Color, Thickness
           - Points (subblock) containing:
             - Point count
             - Point data (x, y, speed, direction, width, pressure)

---

## Attempt 1: Our Custom Writer (Failed)

**What we did:**
- Created `writer.go` with custom tagged block writer
- Wrote everything in one large block
- Used our own interpretation of the format

**Result:**
- Files generated but render blank
- Block length: 5533 bytes (too large)
- Structure doesn't match working files

**Why it failed:**
- Didn't follow the exact nesting structure
- Tag indices were guesses
- CRDT sequence encoding was wrong

---

## Attempt 2: Study Working Files (In Progress)

**Observations from working file:**
- Block length: 25 bytes (very small!)
- Multiple nested subblocks
- Specific tag indices at specific positions
- CRDT IDs follow a pattern

**Next steps:**
1. Decode a working file completely
2. Document the exact structure
3. Create an encoder that mirrors it exactly

---

## The Right Approach

Instead of guessing, we should:

1. **Use remarquee's decoder** to fully parse a working file
2. **Document the structure** at every level
3. **Write the inverse** - an encoder that produces exactly that structure
4. **Test incrementally** - start with one point, then one stroke, then multiple

---

## Minimal Test Case Strategy

### Step 1: Single Point
Create the absolute minimum:
- One stroke
- One point at (100, 100)
- Default pen settings

### Step 2: Two Points
Extend to:
- One stroke
- Two points: (100, 100) → (200, 200)

### Step 3: One Complete Stroke
Full stroke with:
- Multiple points
- Proper pressure/width/speed

### Step 4: Multiple Strokes
Multiple strokes on one page

### Step 5: Complex Scenes
Full scene tree with groups, etc.

---

## Key Questions to Answer

1. What is the exact block/subblock nesting?
2. What are the correct tag indices for each field?
3. How are CRDT IDs assigned?
4. What are the default values for optional fields?
5. How is the point data encoded exactly?

---

## Action Items

- [ ] Create a CLI command to dump the structure of a working .rm file
- [ ] Document the exact byte-by-byte structure
- [ ] Create encoder functions that mirror the decoder exactly
- [ ] Test with minimal case
- [ ] Iterate and expand

---

## Resources

- remarquee decoder: `pkg/rmdoc/rmv6_tagged_block_reader.go`
- Scene tree decoder: `pkg/rmdoc/rmv6_scene_tree.go`
- Line decoder: `pkg/rmdoc/rmv6_line_decode.go`
- Python rmscene: https://github.com/ricklupton/rmscene


---

## Major Discovery!

**Date:** January 19, 2026 - Evening

### The Problem Found

Used our decoder tool on both files:

**Working file:**
- Root has 6 children (strokes)
- Each child has proper CRDT IDs
- Items are in the sequence

**Our generated file:**
- Root has **0 children**!
- The scene tree is empty!

### Root Cause

In `builder.go`:
1. We call `buildSceneTree(strokes)` which creates a tree and calls `tree.AddItem()`
2. We pass that tree to `WriteSceneTree(w, tree, strokes)`
3. But `WriteSceneTree` **ignores the tree's items** and just uses the strokes array!
4. The writer never actually writes the items from the tree!

### The Fix

The writer needs to:
1. Get the items from `tree.Root.Children.Items()`
2. Write those items to the binary format
3. NOT use the strokes array directly

The strokes array should only be used to BUILD the tree, not to WRITE it.

---

## Next Steps

1. Fix `WriteSceneTree` to use the tree's items
2. Ensure we're writing the CRDT sequence correctly
3. Test with minimal case
4. Verify it renders

---
