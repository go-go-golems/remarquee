### RMQ-0006: Debugging Strategy (Device vs Export Rendering)

This doc is the “single source of truth” for how we debug **rendering mismatches** between:

- **Device view** (what reMarkable shows interactively)
- **`remarquee` output** (Go renderer → PDF/PNGs)
- **`remarks` output** (Python reference → PDF/PNGs)
- **Programmatic debug renders** (pure Go raster output from parsed strokes)

The core goal is to stop “eyeballing mismatched artifacts” and instead use a repeatable workflow that always compares **the same fixture** across all environments.

---

### What we’ve completed so far (current rendering work)

- **Golden visual comparison utilities**
  - `pkg/pdfcmp/`: page rasterization + diffing; robust fallback to Poppler `pdftoppm` when UniDoc render fails.
  - Dimension tolerance (±1px) to avoid false `maxDiffRatio=1.0` failures from rounding.

- **V6 stroke color fidelity**
  - Per-stroke color application in overlay rendering (fix “everything black”).
  - V6 trailing RGBA marker parsing for highlight/shader strokes (so highlights have real colors).
  - A unit test ensuring `Test.rmdoc` stroke colors are present.

- **Page size alignment vs `remarks`**
  - Adjusted blank/notebook page sizing to match `remarks`’ CairoSVG px→pt behavior (0.75 factor), eliminating size-mismatch goldens.

- **Human-in-the-loop image review**
  - We standardized on `plz-confirm image` for structured review questions.
  - We standardized on non-interactive `pinocchio` runs for VLM checks (optional).

- **Reusable scripts + artifacts**
  - Ticket scripts under `.../scripts/` to generate A/B PDFs, rasterize pages, and run validation.
  - Stable artifact stash under `remarquee/rendering/rmq-0006-ellipse/` (so we don’t lose `/tmp` paths).
  - Uploaded `Test-rmdoc-remarquee.pdf` to `/remarquee/rendering/rmq-0006-ellipse` on the device using `remarquee cloud put`.

---

### Current issues we’re facing

#### A) “Ellipse/oval misalignment” confusion (primary focus right now)

We have two competing narratives:

- Our parsed + rendered page (remarquee/remarks) shows a **large oval near the top**, plus a **tilted square near the bottom-right**.
- Your device photo shows the **same page’s highlighter strokes**, but the **oval appears near the bottom**.

Key nuance: **we must not compare a “new debug render” against an “old device photo of a different artifact”**.

Right now, the device photo that’s in-repo is the usual `Test.rmdoc` page photo, not a photo of any new generated fixture. So if we want to use device truth, we must:

- upload the exact `.rmdoc` fixture we’re discussing to the device,
- open the correct page on-device,
- take a screenshot/photo of that exact fixture/page,
- compare that screenshot against local outputs for the same fixture/page.

#### B) Structural gaps we still haven’t closed (carryover)

- **Typed text**: parsing exists, rendering doesn’t; anchoring math likely needs more validation.
- **Template backgrounds**: notebook templates aren’t rendered yet.
- **Device-view vs export-view mismatch**: device may show a particular zoom/viewport; exports are full-page.

---

### Systematic debugging workflow (the repeatable strategy)

#### 0) Artifact discipline: always compare the same “tuple”

For every investigation, define and record a tuple:

- **Fixture**: `X.rmdoc`
- **Page**: N
- **Local render**: `remarquee` PDF + PNG(s)
- **Reference render (optional)**: `remarks` PDF + PNG(s)
- **Device render**: photo/screenshot of that same fixture/page

Store the tuple under:

- `remarquee/rendering/<case>/...` (stable, in-repo)
- and (optionally) upload the PDF to device under `/remarquee/rendering/<case>/...`

#### 1) Local programmatic debug render (fast sanity checks)

Goal: isolate *parsing + coordinate transforms* without involving PDF libraries.

Use the programmatic renderer:

- `scripts/17-render-test-page1-programmatic-debug/main.go`
  - Produces:
    - `Test-rmdoc-page1-programmatic-strokes+bbox.png`
    - `Test-rmdoc-page1-programmatic-shapes-only.png` (tool=17 only)
    - `Test-rmdoc-page1-stroke-bboxes.json`

What it answers quickly:

- Are we parsing the expected strokes?
- Do “shape tool” strokes (tool=17) exist and where do they land in screen coordinates?
- Is the suspected oval in the parsed data, and what bbox does it have?

#### 2) Local PDF render (end-to-end render path)

Goal: confirm the PDF renderer isn’t introducing its own coordinate bug.

Use existing scripts:

- `scripts/07-render-test-rmdoc-page1-png.sh` (single page)
- `scripts/12-render-test-rmdoc-pages1-2-pngs.sh` (page mapping sanity)
- `scripts/01-generate-a-vs-b-pdfs-test-rmdoc.sh` (A=remarquee, B=remarks)

Always copy the outputs into `remarquee/rendering/<case>/...` immediately.

#### 3) Upload the **fixture** to device (not just the PDF)

Reason: if the device screenshot is the truth source, it must correspond to the same `.rmdoc` bytes.

Preferred upload mechanism in this repo:

- `go run ./cmd/remarquee cloud put <local-file> <remote-dir>`

Notes:

- `cloud put` uploads documents (PDFs) via rmapi-backed cloud.
- To upload an `.rmdoc` *as an editable notebook* we will need an “upload rmdoc” path (not in the current script set). If we don’t have one yet, the immediate workaround is:
  - use a local `.rmdoc` generation + sync method you already use on device, or
  - extend `remarquee` with an `upload rmdoc` command (future work; see below).

#### 4) Ask structured questions (plz-confirm)

Use `plz-confirm image` for classification questions. Keep them single-purpose:

- “Is the oval above or below the word ‘Test’?”
- “Does the tilted square align between A and B?”
- “Is the mismatch global (everything shifted) or local (subset of strokes)?”

Important gotcha we hit:

- `plz-confirm form` requires `--schema @file.json` or `--schema -` (stdin). It does **not** accept raw JSON as a positional “file path”.

#### 5) Close the loop: use answers to choose the next investigation branch

- If **programmatic render matches PDF render** but both disagree with device:
  - likely **device-view ≠ export-view** (zoom/viewport, page confusion, template, typed-text anchoring)
  - or **fixture mismatch** (not the same `.rmdoc` on device)
- If programmatic differs from PDF:
  - likely a **renderer transform bug** (y inversion / scale / xShift / page scale)
- If only **some strokes** are shifted:
  - likely **anchor/group translation logic** or “special anchor” handling.

---

### Applying the strategy to our current “ellipse” problem

#### Step 1: Lock the current local tuple

We currently have, locally:

- `remarquee/rendering/rmq-0006-ellipse/Test-rmdoc-remarquee-page-1.png`
- `remarquee/rendering/rmq-0006-ellipse/Test-rmdoc-device-page-1.png` (existing device photo)
- programmatic debug:
  - `Test-rmdoc-page1-programmatic-strokes+bbox.png`
  - `Test-rmdoc-page1-programmatic-shapes-only.png`
  - `Test-rmdoc-page1-stroke-bboxes.json`

But: the device photo is of the *existing* `Test.rmdoc` (highlighter page), not a newly generated fixture. So we can only use it as truth **for that exact fixture/page**.

#### Step 2: Identify the “oval” stroke in our parsed data

From `Test-rmdoc-page1-stroke-bboxes.json`, the shape-tool strokes are tool=17:

- index 0 bbox y≈151..253 (small scribble)
- index 6 bbox y≈145..390 (another scribble-ish)
- index 7 bbox y≈1028..1317 (**tilted square**)

The large top oval is visible in `Test-rmdoc-page1-programmatic-shapes-only.png` and is clearly in the top band. If the device photo claims “oval at bottom”, we must first decide:

- Is that device “oval” actually a *different stroke* (e.g. the purple circle), or
- Is the device showing the page at a different viewport/scroll, or
- Are we actually looking at different pages/fixtures?

#### Step 3: Fix the comparison mistake: upload the correct fixture/page

To answer definitively, we need a device screenshot of:

- the same `Test.rmdoc`,
- the same page number,
- preferably the same template/background context.

If we want a *fresh, controlled* test, we should generate a minimal calibration notebook and upload it as `.rmdoc` to the device, then take a photo of that exact notebook.

#### Step 4: Controlled calibration (recommended next)

We should create a calibration `.rmdoc` with:

- a single ellipse at a known Y (e.g. y=1500),
- a square at another known Y (e.g. y=300),
- a small “marker stroke” near y=0 and y=1872 bounds.

Then:

- render locally (programmatic + PDF),
- upload the `.rmdoc` to device,
- take a photo,
- and use `plz-confirm image` to answer: “does ellipse appear at intended Y?”

This removes ambiguity from templates, highlights, and dense content.

---

### Concrete next actions (what we should do next in RMQ-0006)

- **Implement minimal RMV6 writer / fixture generator** (new work):
  - Either port `rmscene/tagged_block_writer.py` concepts into Go, or generate via a tiny Python script.
  - Output a `.rmdoc` zip with 1 page and a few V6 line item blocks.

- **Add an `upload rmdoc` command** (future but important):
  - Upload `.rmdoc` notebooks as editable documents (not just PDFs).
  - This will make the “device truth” workflow first-class.

- **Use the strategy above on the ellipse**:
  - First verify we’re comparing the same fixture/page on device.
  - If yes and mismatch persists, use programmatic bbox json to identify the stroke and trace its group anchor transforms.


