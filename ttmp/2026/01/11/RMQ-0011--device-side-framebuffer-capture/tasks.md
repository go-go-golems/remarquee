# Tasks

## TODO

- [ ] Port framebuffer pointer discovery from goMarkableStream into pkg/devicecapture (build-tagged)
- [ ] Implement CaptureRaw/CapturePNG and ScreenInfo API (width/height/bytes per pixel)
- [ ] Add device-side REST server (device serve) with /api/v1/info + /api/v1/screenshot.png + /api/v1/screenshot.raw
- [ ] Add optional streaming + event endpoints (/api/v1/stream, /api/v1/events, /api/v1/gestures)
- [ ] Add client CLI verbs (device info/screenshot/raw/stream) that call the REST API
- [ ] Document device capture usage + add a validation playbook
- [ ] Cross-compile GOOS=linux GOARCH=arm64 and validate /api/v1/info + screenshot on device
