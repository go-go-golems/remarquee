---
Title: OCR via LLM vision using Geppetto Turns/Blocks
Ticket: RMQ-0001
Status: active
Topics:
    - backend
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Analysis of implementing one-shot OCR via LLM vision using Geppetto Turns/Blocks, plus the `remarquee ocr` dev command."
LastUpdated: 2025-12-14T23:51:43.223789952-05:00
---

## Goal

Build a minimal “one-shot” OCR workflow for development: provide an image, run a single multimodal LLM request via Geppetto, and print the extracted text to stdout.

This is intentionally not “document understanding” yet (no tool loops, no structured extraction); it’s just OCR output.

## How Geppetto represents images in Turns/Blocks

Geppetto’s Turn model allows attaching images to chat blocks via a standard payload key:

- `turns.PayloadKeyImages` (`"images"`) contains `[]map[string]any`
- Each image map should include:
  - `media_type` (string, e.g. `image/png`)
  - either `url` (string) or `content` (`[]byte` or base64 string)

Convenience helper:

- `turns.NewUserMultimodalBlock(text, images)` builds a `BlockKindUser` with both `text` and `images`.

## How images get mapped into provider requests

Engines translate a Turn into provider-native request formats. The relevant logic lives in provider-specific helpers:

### OpenAI (chat completions)

OpenAI request building checks the `images` slice and converts the user message into “multi content” parts:

- first a text part
- then one `image_url` part per image
  - if `content` is provided, it becomes a `data:<media_type>;base64,<...>` URL

### Claude (messages API)

Claude request building converts user content into “parts”, including base64 image parts that preserve `media_type`.

## One-shot OCR flow (conceptual)

1. Read image bytes (`os.ReadFile`)
2. Detect/choose `media_type` (sniff with `http.DetectContentType`, allow override)
3. Build a Turn:
   - system block: “You are an OCR engine …”
   - user block: `NewUserMultimodalBlock(prompt, images)`
4. Create an engine from configuration with `factory.NewEngineFromParsedLayers(parsedLayers)`
5. Call `engine.RunInference(ctx, turn)`
6. Extract the last assistant text block and print it

## The dev command: `remarquee ocr`

We implemented a development command:

- `remarquee ocr <image> [flags]`

Key behavior:

- Reads a local image file and sends it as a multimodal user block.
- Defaults the model (`--ai-engine`) to a vision-capable one (`gpt-4o-mini`) to avoid accidental “text-only model” failures.
- Prints the last assistant text block to stdout.

Common usage:

- `remarquee ocr ./scan.png --openai-api-key "$OPENAI_API_KEY"`
- `remarquee ocr ./scan.png --ai-api-type openai --ai-engine gpt-4o --openai-api-key "$OPENAI_API_KEY"`

Environment support:

- The command loads env vars with prefix `REMARQUEE` (example: `REMARQUEE_OPENAI_API_KEY`).

Profiles support (optional):

- Uses Glazed profiles (default location: `~/.config/pinocchio/profiles.yaml`) via `--profile`/`--profile-file`.

## Constraints / gotchas

- The selected model must support vision input.
- This sends the full image to the provider; be mindful of privacy and provider limits.
- OCR quality depends heavily on image quality and model.

## Next steps (later tickets)

- Reuse this flow with uploads / streaming sources (rmapi / goMarkableStream).
- Add a second stage: parse extracted text into structured, actionable data (likely with tool calling + schema).
