# Changelog

## 2026-01-11

- Initial workspace created


## 2026-01-11

Step 1-2: analyze goMarkableStream and draft capture API design

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/11/RMQ-0011--device-side-framebuffer-capture/analysis/01-gomarkablestream-framebuffer-streaming-analysis.md — Analysis source


## 2026-01-11

Step 3: upload analysis/design PDFs + add implementation tasks

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/11/RMQ-0011--device-side-framebuffer-capture/tasks.md — Implementation task list


## 2026-01-11

Step 4: add device capture package + CLI scaffolding

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/devicecapture/reader.go — New framebuffer capture API


## 2026-01-11

Step 5: add no-cgo help setup for device builds (commit 9b14099)

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee/help_setup_nocgo.go — Disable sqlite help when cgo is off


## 2026-01-11

Step 6: cross-compile + validate device capture endpoints

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/11/RMQ-0011--device-side-framebuffer-capture/reference/01-diary.md — Record device validation commands


## 2026-01-11

Step 7: confirm screenshot matches device via plz-confirm

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/11/RMQ-0011--device-side-framebuffer-capture/reference/01-diary.md — Record user validation


## 2026-01-11

Step 8-9: add stream endpoint + write device capture docs/playbook

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee/cmds/device/stream_handler.go — Raw stream handler


## 2026-01-11

Step 10: validate stream endpoint on device

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/11/RMQ-0011--device-side-framebuffer-capture/reference/01-diary.md — Record stream validation


## 2026-01-11

Step 11: add input event streaming endpoints and validate on device (commit 486db83)

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee/cmds/device/events_handler.go — SSE events endpoint
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/deviceevents/scanner_linux.go — Input event scanner


## 2026-01-11

Step 12: document events/gestures usage (commit 7b14403)

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/doc/topics/device-capture.md — Documented events/gestures endpoints


## 2026-01-11

Step 13: validate gestures output during device swipe (no code change)

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2026/01/11/RMQ-0011--device-side-framebuffer-capture/playbook/01-device-capture-validation.md — Gestures validation reference

