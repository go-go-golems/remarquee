---
Title: YAML DSL for RMDoc (spec + design notes)
Ticket: RMQ-0006
Status: active
Topics:
  - remarkable
  - rmdoc
  - rendering
  - testing
  - yaml
DocType: reference
Intent: long-term
Summary: >
  A versioned YAML DSL for describing reMarkable-like documents (RMDoc superset),
  used to generate reproducible rendering fixtures (programmatic PNGs now, and
  real .rmdoc notebooks later). Includes design process, goals, and the final
  spec with examples.
---

### Why we’re doing this

We repeatedly hit a “debugging tax” when investigating rendering mismatches:

- fixtures are hard to create/modify quickly,
- the on-device view is easy to confuse with exported views,
- and we lack a declarative, reviewable way to express “the simplest possible test case”.

This YAML DSL (working name: **RMDoc-DSL**) is intended to be:

- **Readable**: humans can review it in PRs and reason about expected output.
- **Declarative**: describe *what* is on the page, not how to encode it.
- **Versioned + extensible**: can model more than we can render today.
- **Two-output capable**:
  - **Now**: produce *programmatic PNG debug renders* (fast iteration; no PDF libs).
  - **Later**: compile to real `.rmdoc` (V6 scene tree + strokes + text + templates).

---

### Design constraints / non-goals

- **Do not tie the DSL to current `remarquee` internals**. It should remain useful if renderer implementation changes.
- **Do not require rmdoc encoding knowledge to author a test**.
- **Non-goal (for initial milestone)**: perfect one-to-one round-tripping to `.rm` bytes.

---

### Conceptual model: “RMDoc superset”

RMDoc-DSL models a superset of the `.rmdoc` container and scene content:

- Document-level metadata (name, kind, formatVersion-like fields).
- Pages:
  - notebook pages with templates
  - PDF-backed pages
  - blank pages
- Page content as ordered layers/groups/items:
  - strokes (with tool/color/points)
  - shapes (ellipse/rect/line) as a convenience layer that can be lowered to strokes
  - typed text blocks (paragraph style, anchors)
  - images / embedded backgrounds
  - raw/unmodeled blocks (escape hatch)

This provides two major benefits:

- We can author *clean* fixtures even when the actual device format is messy.
- We can model future features (templates/text) early and use them as acceptance criteria.

---

### Implementation status: supported now vs planned

**Supported now (initial implementation target):**

- `document.pages[].canvas` in RM screen coordinate space (1404×1872).
- `items`:
  - `stroke` (polyline of points)
  - `shape` (ellipse/rect/line) rendered directly (programmatic), and optionally “lowered” to strokes later.
- Palette + tool enums (names as well as numeric values).
- Programmatic PNG renderer:
  - grid overlay
  - per-item bbox overlay
  - “shapes-only” filter (for isolating tool=17 vs others)

**Planned (modeled in DSL; may be unimplemented initially):**

- `template` backgrounds (named templates, template-to-page-size strategy).
- typed text:
  - `root_text` blocks
  - anchoring rules / group anchors
- groups + anchor transforms
- PDF-backed pages and page coordinate transforms
- raw/escape hatch items for unknown blocks
- compiler: YAML → `.rmdoc` (zip) + V6 `.rm` writer

The key is: the DSL should already be able to *describe* these features, even if our renderer can’t yet output them.

---

### Coordinate systems (critical)

RMDoc-DSL uses explicit coordinate spaces.

#### `rm_screen_v6` (default)

- **Canvas size**: \(W=1404\), \(H=1872\)
- **X**: centered, in range \([-W/2, +W/2]\)
- **Y**: top-down, in range \([0, H]\)

This matches the conventions used in our V6 PDF renderer (x-shift by half width; y inversion happens at export time, not in the DSL).

---

## RMDoc-DSL Spec (v0)

### Top-level document

```yaml
rm_dsl: v0
document:
  name: "Example"
  kind: notebook            # notebook | pdf | mixed
  format: v6                # v3 | v5 | v6 | mixed
  pages:
    - id: "page-1"          # optional; if omitted, generated deterministically
      template: "P Dots S"  # optional
      canvas:
        space: rm_screen_v6
        width: 1404
        height: 1872
      layers:
        - name: "Layer 1"
          items:
            - kind: shape
              shape: ellipse
              stroke:
                tool: fineliner_2
                color: black
                width: 1
              center: { x: 0, y: 1500 }
              rx: 250
              ry: 150
```

### Enums

#### Pen/tool

Tools can be provided as:

- **name**: `fineliner_2`, `highlighter_2`, …
- **numeric id** (advanced): `17`, `18`, …

We adopt `rmscene`’s mapping:

- `fineliner_2`: 17
- `highlighter_2`: 18
- (others may be added as needed)

#### Colors

Colors can be:

- palette name: `black`, `red`, `highlight_pink`, …
- numeric `pen_color` id
- explicit RGBA (advanced)

The palette should align with `rmscene`/`rmc`’s known values where possible.

---

### Page layers and items

#### `stroke` item

```yaml
- kind: stroke
  stroke:
    tool: fineliner_2
    color: black
    width: 1             # logical width (renderer-specific mapping)
  points:
    - { x: -200, y: 120, pressure: 1 }
    - { x:  200, y: 120, pressure: 1 }
```

Notes:

- `pressure/speed/direction/width` can be added later to match true `.rm` points.
- For programmatic rendering, points are treated as a polyline.

#### `shape` item

Convenience item that can be lowered to strokes later.

```yaml
- kind: shape
  shape: rect            # rect | ellipse | line
  stroke:
    tool: fineliner_2
    color: black
    width: 1
  rect:
    x: -200
    y: 1400
    w: 400
    h: 250
  rotate_deg: 15         # optional
```

---

### Escape hatch items (future)

#### `raw` item

```yaml
- kind: raw
  description: "Unmodeled V6 block bytes"
  rm_blocks_hex: "..."         # or base64
```

---

## Canonical debug cases we want to express

These are the “must-have” YAML fixtures for renderer debugging:

- **ellipse_at_bottom**: ellipse centered at y≈1500
- **ellipse_at_top**: ellipse centered at y≈200
- **square_rotation_sanity**: rotated rect near bottom-right
- **anchor_translation_sanity** (future): shapes inside a group with anchor transforms
- **typed_text_anchor_sanity** (future): a root text block + anchored group

---

### Open questions / decisions to make later

- **Compilation target**: do we compile YAML → `.rmdoc` directly in Go (preferred), or via a Python helper using `rmscene`’s writer as the source of truth?
- **Template strategy**: do we treat templates as “background images” in the DSL, or as a named type that maps to reMarkable templates?
- **Text shaping**: do we model typed text as “high-level text” or as “root_text blocks” close to device format?

---

## JS scriptability (Goja embedded VM): proposed JS API

YAML is great for static fixtures, but we also want **scriptable fixture generation**:

- parametrized cases (e.g., sweep ellipse Y across [200..1700])
- randomized / property-based cases (seeded)
- shared helpers (grid markers, reference axes, labels)
- “case libraries” that can be imported by multiple tests

The proposal: support `.js` case generators executed inside an embedded **goja** VM.
The JS script returns an RMDoc-DSL object that is then validated/normalized and rendered.

### Inputs/outputs contract

- **Input**: a JS file (plus optional JSON params)
- **Output**: a plain JS object matching the RMDoc-DSL schema:
  - `{ rm_dsl: "v0", document: { ... } }`

If `rm_dsl` is missing, we treat it as `v0` (but only for JS output; YAML should remain explicit).

### Proposed JS “builder” API (global `rm`)

The VM injects a single global namespace, `rm`, which provides:

- **`rm.doc(name)`** → `DocumentBuilder`
- **`rm.page(id?)`** → `PageBuilder`
- **`rm.layer(name)`** → `LayerBuilder`
- **`rm.stroke()` / `rm.shape()`** convenience constructors
- **Enums**:
  - `rm.tool.fineliner_2` etc (string or numeric ids)
  - `rm.color.black`, `rm.color.highlight_pink`, …
  - `rm.space.rm_screen_v6`
- **Math helpers**:
  - `rm.deg(x)` returns radians (or we keep degrees and normalize later)
  - `rm.seed(n)` returns a seeded RNG object for repeatable randomness

#### Example 1: a single ellipse-at-bottom fixture

```javascript
export default function main() {
  return rm.doc("ellipse-at-bottom")
    .notebook().v6()
    .page("p1").canvas(rm.space.rm_screen_v6, 1404, 1872)
    .layer("shapes")
      .ellipse({ x: 0, y: 1500 }, 240, 140).stroke(rm.tool.fineliner_2, rm.color.black, 1)
      .rect({ x: 280, y: 1050, w: 320, h: 320 }).rotateDeg(15).stroke(rm.tool.fineliner_2, rm.color.black, 1)
      .stroke(rm.tool.fineliner_2, rm.color.red, 1).polyline([{ x: -650, y: 50 }, { x: -450, y: 50 }])
      .stroke(rm.tool.fineliner_2, rm.color.green, 1).polyline([{ x: -650, y: 1820 }, { x: -450, y: 1820 }])
    .done();
}
```

Notes:

- `.done()` returns the final plain JSON-serializable object.
- Builders return themselves to enable chaining; `.page()`/`.layer()` set the current context.

#### Example 2: sweep ellipse Y (parametrized)

```javascript
export default function main(params) {
  const ys = params.ys ?? [200, 600, 1000, 1400, 1700];
  const d = rm.doc("ellipse-sweep").notebook().v6();

  ys.forEach((y, i) => {
    d.page(`p${i+1}`).canvas(rm.space.rm_screen_v6, 1404, 1872)
      .layer("shapes")
        .ellipse({ x: 0, y }, 240, 140).stroke(rm.tool.fineliner_2, rm.color.black, 1)
      .endLayer()
    .endPage();
  });

  return d.done();
}
```

### How this maps to Go (goja execution model)

In Go we expose:

```go
// Pseudocode only
func LoadCase(ctx context.Context, path string, params map[string]any) (*DSLDoc, error) {
  switch ext(path) {
  case ".yaml", ".yml": return LoadYAML(path)
  case ".js": return LoadJSWithGoja(ctx, path, params)
  default: ...
  }
}
```

`LoadJSWithGoja`:

- creates a goja runtime
- injects the `rm` global (builders + enums + helpers)
- executes the JS module
- calls exported `default` function as `main(params)` (or falls back to global `main`)
- reads the returned JS object and converts it to Go:
  - `goja.ExportTo(v, &dslDoc)`
- validates and normalizes (fill defaults, ids, canvas size)

### File/module loading

To allow reuse, we propose a very small import system:

- `rm.include("relative/path.js")` that loads and executes another file from a predefined root:
  - ticket folder root, or repo root, or a passed-in “case root” directory
- no network, no filesystem writes

We do **not** start with full Node-compatible module resolution; it’s overkill for fixtures.

### Safety / determinism / constraints

Because scripts can hang or be non-deterministic, we enforce:

- **Execution deadline**: respect `ctx` timeout (interrupt the runtime)
- **No network**
- **No unrestricted filesystem access**
- **Determinism**:
  - recommend using `rm.seed(n)` for randomness
  - optionally require scripts to declare `seed:` metadata (future)

### Output normalization rules (same as YAML)

Regardless of origin (YAML or JS), we apply a single normalization pass:

- fill default `canvas` if missing (`rm_screen_v6`, 1404×1872)
- resolve tool/color names → canonical internal representation
- ensure stable page IDs if missing
- validate required fields and error with precise paths (e.g. `document.pages[0].layers[0].items[3]...`)


