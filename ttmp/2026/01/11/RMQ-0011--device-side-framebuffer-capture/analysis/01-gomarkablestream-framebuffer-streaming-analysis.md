---
Title: goMarkableStream framebuffer streaming analysis
Ticket: RMQ-0011
Status: active
Topics:
    - backend
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: goMarkableStream/internal/remarkable/fb_rm.go
      Note: Framebuffer reader and pointer entrypoint
    - Path: goMarkableStream/internal/remarkable/pointer.go
      Note: Pointer discovery on rM2
    - Path: goMarkableStream/internal/remarkable/pointer_arm64.go
      Note: Pointer discovery on Paper Pro
    - Path: goMarkableStream/internal/stream/handler.go
      Note: Frame streaming loop
    - Path: goMarkableStream/internal/stream/raw.go
      Note: Single-frame raw capture
    - Path: goMarkableStream/tools/raw_client/client.go
      Note: Raw to PNG conversion
ExternalSources: []
Summary: Deep dive into goMarkableStream framebuffer access and streaming pipeline
LastUpdated: 2026-01-11T19:14:06-05:00
WhatFor: Explain goMarkableStream internals and identify reusable code paths for device-side capture
WhenToUse: Use before implementing framebuffer capture or streaming in remarquee
---


# goMarkableStream framebuffer streaming analysis

## Executive summary

goMarkableStream streams the reMarkable UI by reading raw framebuffer bytes out of the `xochitl` process memory via `/proc/<pid>/mem`. It locates the framebuffer base pointer by parsing `/proc/<pid>/maps` (rM2) or the last `/dev/dri/card0` mapping (Paper Pro), then repeatedly `ReadAt` into a pre-sized buffer and pushes it to HTTP clients. It couples this with an input event scanner to throttle streaming (only stream for 2s after pen/touch activity), and exposes multiple endpoints (`/stream`, `/raw`, `/events`, `/gestures`).

For our use case, the core reusable pieces are:

- **Framebuffer pointer discovery**: `internal/remarkable.GetFileAndPointer` plus `getFramePointer` in `pointer.go` and `pointer_arm64.go`.
- **Frame read loop**: `internal/stream.StreamHandler.fetchAndSend` and the raw buffer pool.
- **Device metadata constants**: `internal/remarkable/const.go` and `const_arm64.go` for width/height/byte size.
- **Raw screenshot decoding**: `tools/raw_client/client.go` shows raw -> RGBA -> PNG pipeline.

These can be reworked into a minimal capture subsystem inside remarquee, with optional streaming.

## How goMarkableStream finds the framebuffer

The code assumes the visible UI is represented by a contiguous framebuffer owned by `xochitl`, and that the pointer can be found by inspecting `/proc/<pid>/maps` and `/proc/<pid>/mem`.

Key flow (device builds only):

- `internal/remarkable/fb_rm.go:GetFileAndPointer` opens `/proc/<pid>/mem` and calls `getFramePointer`.
- `internal/remarkable/findpid.go:findXochitlPID` walks `/proc` entries, resolves symlinks, and finds the pid whose executable is `/usr/bin/xochitl`.
- On rM2 (arm, !arm64): `internal/remarkable/pointer.go:getFramePointer` scans `/proc/<pid>/maps`, finds the `/dev/fb0` region, and returns `start_addr + 8`.
- On Paper Pro (arm64): `internal/remarkable/pointer_arm64.go` uses the last `/dev/dri/card0` mapping end address, then seeks in `/proc/<pid>/mem` to skip internal header blocks until `ScreenSizeBytes` is covered, returning the computed pointer.

The frame buffer size is defined per device:

- rM2: `internal/remarkable/const.go` sets `ScreenSizeBytes = ScreenWidth * ScreenHeight * 2` (2 bytes per pixel).
- Paper Pro: `internal/remarkable/const_arm64.go` uses `ScreenSizeBytes = ScreenWidth * ScreenHeight * 4`.

This code path is the primary dependency for both streaming and single-shot screenshots.

## How streaming works (server + handlers)

### Server wiring

`main.go` orchestrates startup:

- `remarkable.GetFileAndPointer()` returns the mem file handle (`io.ReaderAt`) and base pointer address.
- `remarkable.NewEventScanner()` begins reading `/dev/input/event*` devices and publishing to a `pubsub.PubSub`.
- `setMuxer(...)` in `http.go` wires:
  - `/stream` to `internal/stream.NewStreamHandler` (optionally with gzip or zstd)
  - `/raw` to `internal/stream.NewRawHandler` (only in dev mode)
  - `/events` to `internal/eventhttphandler.NewEventHandler`
  - `/gestures` to `internal/eventhttphandler.NewGestureHandler`
  - `/version` to a build info string
- `auth.go` wraps handlers with basic auth unless `-unsafe` is set.

### Stream handler logic

`internal/stream/handler.go:StreamHandler.ServeHTTP` is the heart of the stream:

- Reads a `rate` query param (milliseconds). Minimum is 100ms.
- Subscribes to the input event bus and sets `writing = true` for 2s after pen/touch activity.
- On each tick, if `writing` is true, it reads `ScreenSizeBytes` starting at `pointerAddr` using `file.ReadAt(...)`.
- It writes raw bytes to the response, optionally RLE-encoded (`internal/rle.NewRLE`).
- It flushes each chunk using `http.Flusher` for streaming behavior.

The per-request buffer is pooled:

```go
rawData := rawFrameBuffer.Get().([]uint8)
_, _ = h.file.ReadAt(rawData, h.pointerAddr)
_, _ = w.Write(rawData)
```

The throttling middleware in `internal/stream/mdw.go` limits concurrent stream writers to 1.

### Raw handler (single snapshot)

`internal/stream/raw.go:RawHandler.ServeHTTP` reads the framebuffer once and returns the raw bytes. This is a minimal primitive we can adapt for screenshot capture.

### Input events and throttling

The event scanner (`internal/remarkable/events_linux.go`) reads from `PenInputDevice` and `TouchInputDevice` and publishes `events.InputEventFromSource` into a `pubsub.PubSub`. The stream handler uses this to avoid spamming frames when the device is idle.

This event layer can also be exposed as a low-latency UI stream if we want to mirror pen/touch input to remote clients.

## How client decoding works (raw -> image)

goMarkableStream's browser client uses WebGL for fast rendering, but the CLI tool `tools/raw_client/client.go` shows the simplest raw conversion:

- GET `/raw` (TLS + basic auth)
- `image.NewRGBA(Rect(...))`
- `copy(img.Pix, rawData)`
- `png.Encode(file, img)`

This is a usable blueprint for a `remarquee device capture` CLI that can output PNGs by either:

- doing the raw capture on the device and writing PNG locally, or
- returning raw bytes over REST and decoding on the client.

## Reusable components for our capture feature

Reusable logic (direct copy or refactor):

- `internal/remarkable/fb_rm.go:GetFileAndPointer`
- `internal/remarkable/pointer.go:getFramePointer`
- `internal/remarkable/pointer_arm64.go:getFramePointer`
- `internal/remarkable/findpid.go:findXochitlPID`
- `internal/remarkable/const*.go` (width/height/bytes and input device paths)
- `internal/stream/raw.go:RawHandler` (single-shot frame read)
- `tools/raw_client/client.go` (raw -> PNG conversion)

Important behavior to preserve:

- The pointer calculation differs between rM2 and Paper Pro.
- The raw buffer size depends on `ScreenSizeBytes` (2 or 4 bytes per pixel).
- Accessing `/proc/<pid>/mem` requires root on-device.

## Suggested capture pseudocode (single-shot)

```pseudo
function capture_framebuffer_png(outPath):
    file, ptr = remarkable.GetFileAndPointer()
    raw = make([]byte, remarkable.ScreenSizeBytes)
    file.ReadAt(raw, ptr)
    img = RGBA(width=ScreenWidth, height=ScreenHeight)
    img.Pix = raw
    png.Encode(outPath, img)
```

For a REST endpoint, the handler can return PNG directly:

```pseudo
handler /api/v1/screenshot:
    file, ptr = get_file_and_ptr()
    raw = read(ptr, ScreenSizeBytes)
    png = encode_png(raw)
    write Response(content_type="image/png", body=png)
```

## Risks and constraints

- **Privilege**: reading `/proc/<pid>/mem` requires root; remarquee on-device must run as root or use elevated permissions.
- **Model variance**: pointer discovery and pixel format differ between rM2 and Paper Pro.
- **Latency**: repeated full-frame reads are expensive; for streaming we should throttle or provide a diff mechanism.
- **Security**: goMarkableStream uses basic auth + self-signed TLS; we need a comparable security story for remarquee.

## References (code)

- `goMarkableStream/main.go`
- `goMarkableStream/http.go`
- `goMarkableStream/auth.go`
- `goMarkableStream/internal/remarkable/fb_rm.go`
- `goMarkableStream/internal/remarkable/pointer.go`
- `goMarkableStream/internal/remarkable/pointer_arm64.go`
- `goMarkableStream/internal/remarkable/const.go`
- `goMarkableStream/internal/remarkable/const_arm64.go`
- `goMarkableStream/internal/remarkable/findpid.go`
- `goMarkableStream/internal/stream/handler.go`
- `goMarkableStream/internal/stream/raw.go`
- `goMarkableStream/internal/eventhttphandler/pen_handler.go`
- `goMarkableStream/internal/eventhttphandler/gesture_handler.go`
- `goMarkableStream/tools/raw_client/client.go`
