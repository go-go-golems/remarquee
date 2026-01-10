---
Title: RMDoc-DSL (YAML + JS) — Getting Started
Slug: rmdsl-getting-started
Short: A hands-on guide to authoring RMDoc-DSL fixtures in YAML or JavaScript (goja), rendering them to debug PNGs, and using them to systematically debug rendering issues.
Topics:
- remarquee
- rmdoc
- rendering
- testing
- validation
- yaml
- javascript
Commands:
- remarquee
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

# RMDoc-DSL (YAML + JS) — Getting Started

RMDoc-DSL is a small, versioned DSL that lets you describe “reMarkable-like” documents in a **human-readable** way. The immediate payoff is *debug velocity*: you can generate controlled test fixtures (like “ellipse at y=1500”) without hand-authoring binary `.rm` files, and without the ambiguity of reusing a screenshot from a different fixture.

In the short term, RMDoc-DSL is about producing **programmatic PNG debug renders** quickly (grid overlays, predictable coordinate space, stable outputs). In the long term, it’s also a foundation for generating real `.rmdoc` notebooks and for building interoperability (“create documents from other applications”) without baking format details into every tool.

This page shows:

- how to write fixtures in **YAML**
- how to write fixtures in **JavaScript** (executed in an embedded **goja** VM)
- how to render fixtures to PNGs (without PDF rendering)
- how to use this workflow to debug renderer issues (device vs export, anchors, transforms)

If you want the spec itself (the canonical schema and design notes), see:

```
remarquee help rmdsl-spec
```

(That document lives in the RMQ-0006 ticket; this page focuses on usage.)

---

## Mental model (what’s going on)

RMDoc-DSL sits between “author intent” and “renderer output”.

```
           +--------------------+
           |  YAML or JS Case   |
           |  (RMDoc-DSL v0)    |
           +---------+----------+
                     |
                     v
           +--------------------+
           |  Loader/Validator  |
           |  (defaults, ids,   |
           |   schema sanity)   |
           +---------+----------+
                     |
                     v
           +--------------------+
           | Programmatic PNG   |
           | Renderer (no PDF)  |
           | (grid + overlays)  |
           +--------------------+
```

Key design principle: **fixture identity is everything**. We only ever compare outputs if they come from the same tuple:

- fixture source (`.yaml` or `.js`)
- generated DSL doc
- rendered PNG(s)
- (optional) uploaded `.rmdoc` and device screenshot of the *same fixture*

---

## Prerequisites

You need:

- a working `remarquee` repo checkout
- Go toolchain (you can use `go run` from the repo root)

RMDoc-DSL rendering currently uses a ticket script:

- `ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/18-rmdsl-render-to-png/main.go`

This script produces **programmatic PNGs** (no PDF renderer involved).

---

## Coordinate space (critical to not get confused)

By default, RMDoc-DSL uses `rm_screen_v6` coordinate space:

- **Canvas**: 1404 × 1872
- **X**: centered \([-702..+702]\) (the renderer shifts by +702 to map into PNG pixel space)
- **Y**: top-down \([0..1872]\)

Think of it as:

```
Y=0 (top)
 |
 |     X=-702        X=0        X=+702
 |       |------------|------------|
 v
Y=1872 (bottom)
```

This matters because a lot of “ellipse is off” debates are actually:

- wrong coordinate assumptions (centered vs left-origin)
- mixing device “viewport” vs exported full-page
- comparing the wrong fixture/page

The DSL tries to make the coordinate model explicit so we can debug *systematically*.

---

## Part 1: YAML fixtures (static, review-friendly)

YAML is ideal for:

- small canonical fixtures
- PR reviews (“what did you change in the test case?”)
- long-lived goldens

### A minimal example: ellipse at the bottom

The ticket includes:

- `ttmp/.../cases/01-ellipse-at-bottom.yaml`

It describes:

- an ellipse centered at `y: 1500`
- a rotated square near the lower-right area
- two “marker strokes” near the top and bottom bounds (red/green)

### Render YAML → PNG

Run this from the `remarquee/` repo root:

```bash
go run ./ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/18-rmdsl-render-to-png/main.go \
  --in  ./ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/cases/01-ellipse-at-bottom.yaml \
  --out /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/rendering/rmq-0006-ellipse
```

Expected output:

- a PNG in the `--out` directory named like `ellipse-at-bottom-p1.png` (the exact name depends on the document/page IDs).

### What to look for in the image

- **Grid lines**: these are there so you can describe offsets as “one grid cell” instead of “vibes”.
- **Axis line**: a faint vertical line at x=0 (center of screen coordinates).
- **Markers**: the red marker near y=50 and green near y=1820 confirm your mental model of “top vs bottom”.

---

## Part 2: JavaScript fixtures (scriptable, parametric)

YAML is intentionally static. JavaScript is for when you want:

- sweeps (generate 20 variants)
- property-based tests (seeded randomness)
- helper libraries and reusable “case generators”

The JS runtime is **goja** embedded in Go. This is not Node.js. It’s a controlled sandbox intended for case generation.

### The JS execution contract

A JS case file must provide an entry point in one of these forms:

- `function main(params) { ... return dslObject; }`
- `exports.default = function(params) { ... }`
- `module.exports = function(params) { ... }`

It must return a plain object matching the DSL schema:

- `{ rm_dsl: "v0", document: { ... } }`

### The `rm` builder API

The runtime injects a global `rm` namespace that lets you build the DSL object fluently.

At a high level, you can think of it as:

```text
rm.doc(name)
  -> .page(id).canvas(space,w,h)
     -> .layer(name)
        -> add items (ellipse/rect/stroke)
  -> .done()  // returns plain object
```

### Example: the same ellipse-at-bottom case (JS)

The ticket includes:

- `ttmp/.../cases/02-ellipse-at-bottom.js`

It uses the `rm` builder:

```javascript
function main(params) {
  return rm.doc("ellipse-at-bottom-js")
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

### Render JS → PNG

```bash
go run ./ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/18-rmdsl-render-to-png/main.go \
  --in  ./ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/cases/02-ellipse-at-bottom.js \
  --out /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/rendering/rmq-0006-ellipse
```

Expected output (example):

- `ellipse-at-bottom-js-p1.png`

### Reuse with `rm.include()`

JS cases can load helper code using:

```javascript
rm.include("./lib/helpers.js");
```

Rules:

- includes are resolved relative to the current JS file
- includes are restricted to a “case root” directory (to prevent surprising filesystem access)

---

## Debug workflow patterns (how to use this to debug rendering)

This section is intentionally opinionated. It’s the workflow we want to standardize on.

### Pattern A: “Is my coordinate transform wrong?”

Use a sweep case to quickly check whether your renderer has:

- a constant Y shift
- a scale mismatch
- a flip (top/bottom)

Pseudo-approach:

```text
for y in [200, 600, 1000, 1400, 1700]:
  render ellipse centered at (0, y)
  render marker at y=50 and y=1820
  compare outputs between renderers
```

If the ellipse moves consistently with y in the PNGs, your coordinate mapping is likely correct. If it “compresses” or “drifts”, you likely have a scaling bug.

### Pattern B: “Device vs export mismatch”

If you’re using the device as ground truth, you must ensure you’re looking at the same fixture.

Checklist:

- **Same fixture**: the exact same `.rmdoc` bytes are on device
- **Same page**: page mapping sanity (page N)
- **Same view**: note zoom/scroll (device can show a viewport; exports are full-page)

RMDoc-DSL helps here because you can generate “obvious markers”:

- an axis line
- top/bottom colored markers
- a label stroke near a corner

### Pattern C: “Only some strokes are shifted”

That typically suggests:

- anchor/group transforms
- typed text anchoring
- tool-specific transforms

RMDoc-DSL can isolate this quickly:

- put the suspect element on its own layer
- render a shapes-only view
- add bbox overlays (planned: the PNG renderer will optionally draw per-item bboxes)

---

## Troubleshooting

### “Output directory must exist”

The renderer script currently expects `--out` to already exist.

Reason: it’s intentionally conservative and avoids side effects in scripts unless explicitly requested.

### “No JS entry point found”

Your `.js` file must define one of:

- `function main(params) { ... }`
- `exports.default = function(params) { ... }`
- `module.exports = function(params) { ... }`

### “rm is undefined”

The loader injects `rm`, but only when the input is a `.js` case.
If you’re executing the JS with a different runtime (or copying code into another tool), you won’t have the builder.

---

## Where to look in the code (for developers extending this)

The core implementation lives in:

- `pkg/rmdsl/`
  - loader (`LoadFromFile`) and normalization (`Normalize`)
  - JS runner (`js.go`) providing:
    - `rm` builder prelude
    - `rm.include()`

The initial consumer is:

- `ttmp/.../scripts/18-rmdsl-render-to-png/main.go`

---

## Next steps (suggested)

- Add a canonical **sweep** JS case:
  - ellipse y = 200..1700
  - one page per y
- Extend the PNG renderer with optional overlays:
  - per-item bbox outlines
  - layer visibility toggles
- Add a compiler target:
  - RMDoc-DSL → `.rmdoc` (so the exact fixture can be uploaded to device and photographed)


