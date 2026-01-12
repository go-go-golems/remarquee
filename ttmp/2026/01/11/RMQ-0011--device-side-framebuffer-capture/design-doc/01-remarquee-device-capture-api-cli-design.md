---
Title: Remarquee device capture API + CLI design
Ticket: RMQ-0011
Status: active
Topics:
    - backend
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: goMarkableStream/internal/remarkable/fb_rm.go
      Note: Source for pointer and frame access
    - Path: goMarkableStream/internal/stream/handler.go
      Note: Prototype for streaming endpoint
    - Path: goMarkableStream/internal/stream/raw.go
      Note: Prototype for screenshot endpoint
ExternalSources: []
Summary: Design for on-device framebuffer capture, streaming, and a REST/CLI interface
LastUpdated: 2026-01-11T19:14:06-05:00
WhatFor: Specify how to integrate capture into remarquee and expose REST + CLI
WhenToUse: Use when implementing or reviewing device-side capture features
---


# Remarquee device capture API + CLI design

## Executive Summary

We will add a device-side capture subsystem to remarquee that can take screenshots and (optionally) stream the framebuffer and input events. The subsystem will reuse the framebuffer pointer discovery and read loop logic from goMarkableStream and expose it via a lightweight REST API. CLI verbs will wrap that API so a user can run remarquee on the device as a server and issue capture or stream commands from their workstation.

## Problem Statement

We need a first-class way to capture the reMarkable UI directly from the device. The existing workflow relies on external tools (goMarkableStream) and is not integrated into the remarquee toolchain. We want to:

- Capture screenshots (PNG) on demand.
- Optionally stream the framebuffer for live preview.
- Access pen/touch events to enable richer clients later.
- Provide a consistent REST API and CLI surface that works when remarquee runs on-device.

## Proposed Solution

### 1) Add a capture package in remarquee

Create a new package (suggested path: `remarquee/pkg/devicecapture`) that wraps the framebuffer read primitives with a stable API.

Core types and functions (pseudo-API):

```go
// FramebufferReader abstracts device-specific reads.
type FramebufferReader interface {
    ReadFrame(dst []byte) error
    ScreenInfo() ScreenInfo
}

// ScreenInfo describes device properties.
type ScreenInfo struct {
    Width, Height int
    BytesPerPixel int
    Model string
}

func NewFramebufferReader() (FramebufferReader, error)
func CaptureRaw() ([]byte, ScreenInfo, error)
func CapturePNG() ([]byte, ScreenInfo, error)
```

Implementation details:

- `NewFramebufferReader` uses `internal/remarkable.GetFileAndPointer` logic from goMarkableStream.
- `ReadFrame` uses `io.ReaderAt.ReadAt` into a pooled buffer.
- `CapturePNG` wraps `image.NewRGBA` + `png.Encode` (mirrors `goMarkableStream/tools/raw_client/client.go`).

### 2) Add device-side REST server

Add a new server command (suggested): `remarquee device serve` that runs on the tablet and exposes HTTP endpoints. It can be kept minimal (no UI) and only serve JSON/binary.

Suggested endpoints:

- `GET /api/v1/info` -> JSON with `model`, `width`, `height`, `bytesPerPixel`, `screenSizeBytes`.
- `GET /api/v1/screenshot.png` -> `image/png` bytes (single snapshot).
- `GET /api/v1/screenshot.raw` -> `application/octet-stream` raw framebuffer.
- `GET /api/v1/stream` -> streaming raw frames (same format as goMarkableStream `/stream`).
- `GET /api/v1/events` -> SSE pen events (optional, for future remote UI).
- `GET /api/v1/gestures` -> NDJSON gesture events (optional).
- `GET /api/v1/health` -> `200 OK` for readiness checks.

Authentication options:

- Start with HTTP Basic Auth to match goMarkableStream (simple, portable).
- Allow `--unsafe` for local-only testing.
- Optionally support token headers in the future.

### 3) Add CLI verbs to call the REST API

Add a `remarquee device` command group with two modes:

- **Server mode (on-device):**
  - `remarquee device serve --bind :2718 --auth user:pass`
- **Client mode (workstation):**
  - `remarquee device screenshot --url https://remarkable.local:2718 --out screenshot.png`
  - `remarquee device raw --url ... --out frame.raw`
  - `remarquee device stream --url ... --out stream.raw` (or pipe to viewer)
  - `remarquee device info --url ...` (prints JSON)

The client verbs can reuse shared request helpers and optionally decode the raw stream into PNGs for debugging.

## Design Decisions

- **Reuse goMarkableStream logic** instead of reinventing framebuffer discovery. The pointer finding is hardware-specific and already validated.
- **Separate capture core from HTTP server** so CLI can capture locally on-device without running an HTTP server (future `remarquee device screenshot --local` mode).
- **Start with raw and PNG outputs**; defer incremental diff/compression until basic capture is stable.
- **Match goMarkableStream endpoints where useful** (`/stream` format) for compatibility with existing clients.

## Alternatives Considered

- **Directly vendoring goMarkableStream** as a library module.
  - Rejected because it is a full server with a UI; we only need the device access primitives.
- **Relying on rmapi or other cloud sync** to pull screenshots.
  - Rejected because it does not capture live UI and adds latency.
- **Implementing a separate binary** (not remarquee).
  - Rejected to avoid fragmentation; remarquee should be the one tool.

## Implementation Plan

1. Create `pkg/devicecapture` with `FramebufferReader`, `CaptureRaw`, and `CapturePNG`.
2. Add build-tagged files for device-specific pointer discovery (port from goMarkableStream).
3. Add `cmd/remarquee/cmds/device/serve.go` with REST endpoints.
4. Add client verbs (`device screenshot`, `device raw`, `device info`) that call the REST API.
5. Add optional stream endpoints and client verbs (`device stream`, `device events`) once basic capture is stable.
6. Document usage in a new `pkg/doc/topics/device-capture.md` and update ticket diary.

## REST API contract (initial)

- `GET /api/v1/info` -> `application/json`

```json
{
  "model": "remarkable2",
  "width": 1872,
  "height": 1404,
  "bytesPerPixel": 2,
  "screenSizeBytes": 525,  // computed
  "source": "xochitl"      // informational
}
```

- `GET /api/v1/screenshot.png` -> `image/png`
- `GET /api/v1/screenshot.raw` -> `application/octet-stream`
- `GET /api/v1/stream?rate=200` -> streaming raw frames

## Open Questions

- Do we want TLS enabled by default or HTTP-only on device LAN?
- Should the server read frames on-demand only, or keep a shared frame cache?
- How should we expose pixel format (2 bytes vs 4 bytes) to downstream decoders?
- Should we expose the pen/touch event stream in v1 or wait?

## References

- `goMarkableStream/internal/remarkable/fb_rm.go:GetFileAndPointer`
- `goMarkableStream/internal/remarkable/pointer.go:getFramePointer`
- `goMarkableStream/internal/remarkable/pointer_arm64.go:getFramePointer`
- `goMarkableStream/internal/remarkable/const.go`
- `goMarkableStream/internal/remarkable/const_arm64.go`
- `goMarkableStream/internal/stream/handler.go` (stream loop)
- `goMarkableStream/internal/stream/raw.go` (single frame)
- `goMarkableStream/tools/raw_client/client.go` (raw -> PNG)
