---
Title: Device Capture (Framebuffer) — API + CLI
Slug: device-capture
Short: Run remarquee on the reMarkable device to capture screenshots, raw frames, and streams via REST and CLI.
Topics:
- remarquee
- remarkable
- device
- capture
- api
- cli
Commands:
- remarquee
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

# Device Capture (Framebuffer) — API + CLI

Device capture lets you run remarquee on the tablet itself to read the framebuffer and expose it over HTTP. This gives you reliable screenshots and raw frame dumps without relying on cloud sync, and it provides a foundation for live streaming or UI automation.

---

## Overview

The capture pipeline reads the framebuffer directly from the `xochitl` process via `/proc/<pid>/mem`, so it runs only on-device (and typically as root). The server provides a minimal REST API for raw bytes and PNG screenshots, and the CLI offers a client-side wrapper so you can grab frames from your workstation.

Key capabilities:

- Single-shot PNG capture (`/api/v1/screenshot.png`)
- Single-shot raw capture (`/api/v1/screenshot.raw`)
- Raw streaming endpoint (`/api/v1/stream`)
- Pen event stream (`/api/v1/events`)
- Gesture summaries (`/api/v1/gestures`)
- CLI wrappers (`remarquee device info|screenshot|raw|stream|events|gestures`)

---

## Quick Start (Device Server)

Start the capture server on the tablet and keep it running while you pull frames from your workstation.

```bash
# on the device (ssh root@remarkable)
/home/root/remarquee device serve --bind :2718 --username admin --password password
```

Notes:

- The server uses basic auth by default. Use `--unsafe` only on trusted networks.
- The device binary should be built with `GOOS=linux GOARCH=arm64`.

---

## Quick Start (Client Capture)

Use the CLI from your workstation to fetch info, a PNG screenshot, or a raw frame dump.

```bash
# fetch device info (width/height/bytes)
remarquee device info --url http://10.11.99.1:2718 --username admin --password password

# grab a PNG screenshot
remarquee device screenshot --url http://10.11.99.1:2718 --username admin --password password --out ./screenshot.png

# grab a raw framebuffer dump
remarquee device raw --url http://10.11.99.1:2718 --username admin --password password --out ./screenshot.raw
```

---

## REST API Reference

The REST API is intentionally small so it can run on the device with minimal overhead. All endpoints use basic auth unless the server runs with `--unsafe`.

### `GET /api/v1/info`

This endpoint returns the device geometry and pixel format.

**Response (example):**

```json
{
  "model": "RemarkablePaperPro",
  "width": 1632,
  "height": 2154,
  "bytesPerPixel": 4,
  "screenSizeBytes": 14061312
}
```

### `GET /api/v1/screenshot.png`

This endpoint returns a PNG rendering of the current framebuffer state.

**Response:** `Content-Type: image/png`

### `GET /api/v1/screenshot.raw`

This endpoint returns raw framebuffer bytes.

**Response:** `Content-Type: application/octet-stream`

Headers include:

- `X-Device-Model`
- `X-Screen-Width`
- `X-Screen-Height`
- `X-Bytes-Per-Pixel`

### `GET /api/v1/stream`

This endpoint streams raw framebuffer bytes repeatedly at a fixed rate.

**Query parameters:**

- `rate` — frame rate in milliseconds (default: 200, minimum: 100)

**Response:** `Content-Type: application/octet-stream` (chunked)

### `GET /api/v1/events`

This endpoint streams pen input events as Server-Sent Events (SSE).

**Response:** `Content-Type: text/event-stream`

Each SSE event contains a JSON payload like:

```json
{"source":1,"type":3,"code":54,"value":12345}
```

### `GET /api/v1/gestures`

This endpoint streams gesture summaries as newline-delimited JSON (NDJSON).

**Response:** `Content-Type: application/x-ndjson`

Each line contains a JSON object like:

```json
{"left":120,"right":0,"up":30,"down":0}
```

---

## CLI Reference

The `remarquee device` command group provides a thin wrapper around the REST API.

### `remarquee device serve`

Starts the HTTP server on the device.

```bash
remarquee device serve --bind :2718 --username admin --password password
```

### `remarquee device info`

Prints the `/api/v1/info` JSON to stdout.

```bash
remarquee device info --url http://10.11.99.1:2718 --username admin --password password
```

### `remarquee device screenshot`

Downloads a PNG screenshot to a local file.

```bash
remarquee device screenshot --url http://10.11.99.1:2718 --username admin --password password --out ./screenshot.png
```

### `remarquee device raw`

Downloads raw framebuffer bytes.

```bash
remarquee device raw --url http://10.11.99.1:2718 --username admin --password password --out ./screenshot.raw
```

### `remarquee device stream`

Streams raw frames to a file for a fixed duration.

```bash
remarquee device stream --url http://10.11.99.1:2718 --username admin --password password \
  --rate 200 --duration 5s --out ./stream.raw
```

### `remarquee device events`

Streams pen events as SSE and writes the raw output to a file or stdout.

```bash
remarquee device events --url http://10.11.99.1:2718 --username admin --password password \
  --duration 5s --out ./events.sse
```

Use `--duration 0` to stream until interrupted. If the device is idle, the output may be empty.

### `remarquee device gestures`

Streams gesture summaries as NDJSON.

```bash
remarquee device gestures --url http://10.11.99.1:2718 --username admin --password password \
  --duration 5s --out ./gestures.ndjson
```

Use `--duration 0` to stream until interrupted. If the device is idle, the output may be empty.

---

## Build and Deploy Notes

Device capture requires an arm64 Linux binary with CGO disabled. The capture code reads from `/proc/<pid>/mem`, so it must run as root on the tablet.

```bash
# build on workstation
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o /tmp/remarquee-arm64 ./cmd/remarquee

# deploy to device
scp /tmp/remarquee-arm64 root@10.11.99.1:/home/root/remarquee
```

---

## Permissions and Security

Framebuffer access is privileged. Keep the following in mind:

- The server should run on a trusted network only.
- Use `--unsafe` only for quick local tests.
- The default basic auth credentials should be rotated in shared environments.

---

## Troubleshooting

Common failure modes and fixes:

- **`xochitl pid not found`** — ensure the UI process is running; reboot if needed.
- **`permission denied`** — run the server as root on the tablet.
- **Blank or inverted output** — confirm bytes-per-pixel and model reported by `/api/v1/info`.
- **Connection refused** — verify the server is running and listening on the chosen port.

## Roadmap

Upcoming additions include:

- Optional streaming compression (RLE or zstd) for lower bandwidth.
- A validation playbook to standardize screenshot comparisons.
