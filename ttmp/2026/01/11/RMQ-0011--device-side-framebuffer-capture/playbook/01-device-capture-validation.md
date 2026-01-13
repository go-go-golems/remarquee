---
Title: Device Capture Validation
Ticket: RMQ-0011
Status: active
Topics:
    - backend
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarquee/cmd/remarquee/cmds/device/screenshot.go
      Note: Client screenshot command
    - Path: remarquee/cmd/remarquee/cmds/device/serve.go
      Note: Server startup
    - Path: remarquee/cmd/remarquee/cmds/device/stream.go
      Note: Client stream command
ExternalSources: []
Summary: Validate device-side capture server and CLI outputs
LastUpdated: 2026-01-11T20:35:12-05:00
WhatFor: Repeatable validation of /api/v1/info, screenshot, and stream
WhenToUse: After changing device capture or deploying a new binary
---


# Device Capture Validation

## Purpose

Verify that the device-side capture server starts successfully and that screenshots and streams match the on-device display.

## Environment Assumptions

- `remarquee` binary is present on the device (`/home/root/remarquee`).
- You can SSH to the device as root.
- You can reach the device from your workstation over LAN.

## Commands

```bash
# Build + deploy (from workstation)
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o /tmp/remarquee-arm64 ./cmd/remarquee
scp /tmp/remarquee-arm64 root@10.11.99.1:/home/root/remarquee

# Start server on device
ssh root@10.11.99.1 'nohup /home/root/remarquee device serve --bind :2718 --username admin --password password > /tmp/remarquee-device.log 2>&1 &'

# Validate info endpoint
remarquee device info --url http://10.11.99.1:2718 --username admin --password password

# Validate screenshot
remarquee device screenshot --url http://10.11.99.1:2718 --username admin --password password --out ./screenshot.png

# Optional: validate stream for 5 seconds
remarquee device stream --url http://10.11.99.1:2718 --username admin --password password --rate 200 --duration 5s --out ./stream.raw

# Optional: validate events/gestures for 5 seconds
remarquee device events --url http://10.11.99.1:2718 --username admin --password password --duration 5s --out ./events.sse
remarquee device gestures --url http://10.11.99.1:2718 --username admin --password password --duration 5s --out ./gestures.ndjson
```

## Exit Criteria

- `remarquee device info` returns width/height/bytesPerPixel matching the device model.
- `screenshot.png` visually matches the device display.
- `stream.raw` is non-empty and grows during the stream duration.
- `events.sse` and `gestures.ndjson` are created (may be empty if the device is idle).

## Notes

- Run the server as root; framebuffer access requires privileged access to `/proc/<pid>/mem`.
- If the server fails to start, inspect `/tmp/remarquee-device.log` on the device.
