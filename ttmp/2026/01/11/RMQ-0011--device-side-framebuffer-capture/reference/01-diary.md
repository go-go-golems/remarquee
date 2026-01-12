---
Title: Diary
Ticket: RMQ-0011
Status: active
Topics:
    - backend
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: goMarkableStream/internal/remarkable/fb_rm.go
      Note: Root of framebuffer pointer access
    - Path: remarquee/cmd/remarquee/cmds/device/serve.go
      Note: REST server endpoints
    - Path: remarquee/cmd/remarquee/cmds/device/stream.go
      Note: CLI stream client
    - Path: remarquee/cmd/remarquee/cmds/device/stream_handler.go
      Note: Stream endpoint implementation
    - Path: remarquee/cmd/remarquee/main.go
      Note: Registers device command
    - Path: remarquee/pkg/devicecapture/pointer_arm64.go
      Note: Paper Pro pointer discovery
    - Path: remarquee/pkg/devicecapture/reader.go
      Note: Core framebuffer capture API
    - Path: remarquee/pkg/doc/topics/device-capture.md
      Note: Device capture guide
    - Path: remarquee/ttmp/2026/01/11/RMQ-0011--device-side-framebuffer-capture/analysis/01-gomarkablestream-framebuffer-streaming-analysis.md
      Note: Detailed analysis
    - Path: remarquee/ttmp/2026/01/11/RMQ-0011--device-side-framebuffer-capture/design-doc/01-remarquee-device-capture-api-cli-design.md
      Note: Integration design
    - Path: remarquee/ttmp/2026/01/11/RMQ-0011--device-side-framebuffer-capture/playbook/01-device-capture-validation.md
      Note: Validation playbook
ExternalSources: []
Summary: Diary for device-side framebuffer capture work
LastUpdated: 2026-01-11T20:31:06-05:00
WhatFor: Track progress on RMQ-0011 analysis and design
WhenToUse: Use while implementing device capture features
---




# Diary

## Goal

Document the analysis and design work needed to add device-side framebuffer capture (screenshots + streaming) to remarquee.

## Step 1: Create RMQ-0011 workspace and capture goMarkableStream internals

I created the RMQ-0011 ticket workspace and began a focused audit of goMarkableStream to understand how it finds and reads the framebuffer. This step gathered the file-level references and pipeline details that we will reuse for the device-side capture feature.

**Commit (code):** N/A

### What I did
- Created the RMQ-0011 ticket workspace via docmgr.
- Read goMarkableStream source files for framebuffer access, stream handlers, and raw capture tooling.
- Drafted the analysis doc describing pointer discovery, frame reading, and streaming flow.

### Why
- We need a precise understanding of the existing framebuffer access approach before copying or refactoring it into remarquee.

### What worked
- The goMarkableStream codebase clearly documents the pointer discovery and streaming loop.

### What didn't work
- N/A

### What I learned
- Framebuffer access is achieved by reading `/proc/<pid>/mem` for `xochitl`, with different pointer discovery on rM2 vs Paper Pro.

### What was tricky to build
- N/A (research step only).

### What warrants a second pair of eyes
- Confirm that the pointer discovery logic is still valid for current firmware builds.

### What should be done in the future
- Validate the framebuffer pointer logic on both rM2 and Paper Pro before shipping.

### Code review instructions
- Start in `goMarkableStream/internal/remarkable/fb_rm.go` and `goMarkableStream/internal/remarkable/pointer*.go`.

### Technical details
- Files reviewed: `goMarkableStream/main.go`, `goMarkableStream/http.go`, `goMarkableStream/internal/stream/handler.go`, `goMarkableStream/internal/stream/raw.go`, `goMarkableStream/tools/raw_client/client.go`.

## Step 2: Write RMQ-0011 analysis and design docs

I produced the analysis and design documents that lay out the framebuffer capture pipeline and the proposed integration into remarquee. The design includes REST endpoints and CLI verbs so a device-hosted remarquee can serve screenshots and streams to other machines.

**Commit (code):** N/A

### What I did
- Wrote the goMarkableStream analysis doc with code references, pseudocode, and reuse notes.
- Wrote the remarquee integration design with API/CLI surfaces.

### Why
- The ticket requires a clear blueprint before implementation begins.

### What worked
- Both docs are now populated with detailed guidance and concrete endpoints.

### What didn't work
- N/A

### What I learned
- The goMarkableStream raw client is a straightforward blueprint for PNG capture.

### What was tricky to build
- Balancing compatibility with goMarkableStream while keeping remarquee's API surface minimal.

### What warrants a second pair of eyes
- Validate that the REST endpoints align with how we intend to run remarquee on-device (TLS, auth, and bind address).

### What should be done in the future
- Add implementation tasks once the API shapes are agreed on.

### Code review instructions
- Read `remarquee/ttmp/2026/01/11/RMQ-0011--device-side-framebuffer-capture/analysis/01-gomarkablestream-framebuffer-streaming-analysis.md`.
- Read `remarquee/ttmp/2026/01/11/RMQ-0011--device-side-framebuffer-capture/design-doc/01-remarquee-device-capture-api-cli-design.md`.

### Technical details
- N/A

## Step 3: Upload analysis/design docs to the tablet and add tasks

I uploaded the analysis and design docs to the reMarkable so you can review them on-device, then expanded the ticket task list with the concrete implementation steps. This turns the research into a tracked execution plan and gives you immediate access to the PDFs on the tablet.

**Commit (code):** N/A

### What I did
- Uploaded the analysis and design Markdown via `remarquee upload md` to `/remarquee/rmq-0011`.
- Added implementation tasks for the capture package, REST server, and CLI verbs.

### Why
- Provide tablet-friendly copies and a clear task roadmap.

### What worked
- Uploads succeeded and the PDFs were created on the device.

### What didn't work
- N/A

### What I learned
- The upload md command is a quick path for getting docs onto the tablet.

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- Confirm the PDFs render correctly on-device and the remote folder path is acceptable.

### What should be done in the future
- N/A

### Code review instructions
- Review tasks in `remarquee/ttmp/2026/01/11/RMQ-0011--device-side-framebuffer-capture/tasks.md`.

### Technical details
- Command: `go run ./cmd/remarquee upload md ... --remote-dir /remarquee/rmq-0011`

## Step 4: Add device capture package and CLI scaffolding

I added a new device capture package that mirrors goMarkableStream's framebuffer access and wired a `remarquee device` command group with server and client subcommands. The package includes build-tagged pointer discovery for arm and arm64, a raw/PNG capture API, and a portable stub for non-device builds.

**Commit (code):** N/A

### What I did
- Created `pkg/devicecapture` with framebuffer reader, raw/PNG capture helpers, and raw-to-RGBA conversion.
- Ported pointer discovery logic for rM2 (`pointer_arm.go`) and Paper Pro (`pointer_arm64.go`), with a linux/amd64 stub.
- Added `remarquee device` CLI group with `serve`, `info`, `screenshot`, and `raw` commands.
- Registered the device command in `cmd/remarquee/main.go`.
- Ran `go test ./pkg/devicecapture -count=1`.

### Why
- Establish the core capture logic and CLI surface before moving to on-device validation.

### What worked
- Package compiles on the host after adding a linux/amd64 stub.

### What didn't work
- Initial build failed on linux/amd64 due to missing arm-only symbols; fixed by adding a stub.

### What I learned
- Build tags are required to keep pointer discovery isolated to device architectures.

### What was tricky to build
- Mapping raw framebuffer bytes to a reasonable PNG representation across 2-byte and 4-byte formats.

### What warrants a second pair of eyes
- Verify the grayscale mapping for rM2 and confirm the pointer logic remains valid on current firmware.

### What should be done in the future
- Revisit the raw-to-PNG conversion if the on-device results look inverted or washed out.

### Code review instructions
- Start in `remarquee/pkg/devicecapture/reader.go` and `remarquee/pkg/devicecapture/platform_linux.go`.
- Review `remarquee/cmd/remarquee/cmds/device/serve.go` and `remarquee/cmd/remarquee/cmds/device/screenshot.go`.

### Technical details
- Command: `go test ./pkg/devicecapture -count=1`

## Step 5: Fix device build failure due to sqlite help store

The first arm64 build failed on-device because the help system initializes a sqlite store that requires cgo. I added a cgo-gated help setup so the device build skips the sqlite-backed help system and falls back to Cobra default help when cgo is disabled.

**Commit (code):** 9b14099 — "Fix: skip help store when cgo disabled"

### What I did
- Split help setup into cgo and !cgo variants under `cmd/remarquee/help_setup*.go`.
- Updated `cmd/remarquee/main.go` to call `setupHelpSystem` rather than constructing the help system directly.
- Ran `go test ./cmd/remarquee -count=1`.

### Why
- The device build runs with `CGO_ENABLED=0` and cannot use `go-sqlite3` for help storage.

### What worked
- The CLI compiles without the sqlite dependency when cgo is disabled.

### What didn't work
- Initial device run failed with `Binary was compiled with 'CGO_ENABLED=0', go-sqlite3 requires cgo to work. This is a stub`.

### What I learned
- Help system initialization is a hidden dependency on sqlite that needs a no-cgo fallback for device builds.

### What was tricky to build
- Ensuring the help setup remains unchanged for cgo builds while avoiding sqlite at runtime on device.

### What warrants a second pair of eyes
- Confirm that help output is acceptable on device when using Cobra defaults.

### What should be done in the future
- Consider a pure-Go sqlite backend if we need full help features on device.

### Code review instructions
- Review `cmd/remarquee/help_setup.go` and `cmd/remarquee/help_setup_nocgo.go`.
- Validate with `go test ./cmd/remarquee -count=1`.

### Technical details
- Error log: `failed to create sections table: Binary was compiled with 'CGO_ENABLED=0', go-sqlite3 requires cgo to work. This is a stub`

## Step 6: Cross-compile and validate on the device

I cross-compiled the arm64 Linux binary, deployed it to the tablet, started the capture server, and verified the info and screenshot endpoints from the workstation. This confirmed the end-to-end flow on the Paper Pro device.

**Commit (code):** N/A

### What I did
- Built the arm64 binary with `CGO_ENABLED=0`.
- Copied it to `/home/root/remarquee` on the device via `scp`.
- Started the server on port 2718 and fetched `/api/v1/info`, `/api/v1/screenshot.png`, and `/api/v1/screenshot.raw`.
- Captured outputs into `/tmp` on the workstation.

### Why
- Validate the capture pipeline early on real hardware.

### What worked
- `/api/v1/info` returned model/geometry metadata.
- `device screenshot` produced a PNG file (`/tmp/rmq-0011-screenshot.png`).
- `device raw` returned a raw framebuffer dump.

### What didn't work
- N/A (after the help-store fix, the device server started cleanly).

### What I learned
- The Paper Pro reports `BytesPerPixel=4` and the API returns the expected screen dimensions.

### What was tricky to build
- Coordinating device-side service startup without `pkill` on the device shell.

### What warrants a second pair of eyes
- Verify the PNG screenshot visually matches the device display (no inversion or color mismatch).

### What should be done in the future
- Add a simple status endpoint or PID file to make server lifecycle management easier.

### Code review instructions
- Review the server handler in `remarquee/cmd/remarquee/cmds/device/serve.go`.
- Validate with the command sequence below.

### Technical details
- Build: `GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o /tmp/remarquee-arm64 ./cmd/remarquee`
- Deploy: `scp /tmp/remarquee-arm64 root@10.11.99.1:/home/root/remarquee`
- Server: `ssh root@10.11.99.1 'nohup /home/root/remarquee device serve --bind :2718 --username admin --password password > /tmp/remarquee-device.log 2>&1 &'`
- Client info: `go run ./cmd/remarquee device info --url http://10.11.99.1:2718 --username admin --password password`
- Client screenshot: `go run ./cmd/remarquee device screenshot --url http://10.11.99.1:2718 --username admin --password password --out /tmp/rmq-0011-screenshot.png`
- Client raw: `go run ./cmd/remarquee device raw --url http://10.11.99.1:2718 --username admin --password password --out /tmp/rmq-0011-raw.bin`

## Step 7: Validate screenshot output with plz-confirm

I used the plz-confirm image + form widgets to confirm the screenshot matches the device display. The user confirmed the screenshot matched perfectly, with no artifacts, and noted the capture was taken from a notebook view.

**Commit (code):** N/A

### What I did
- Ran `plz-confirm image` with `/tmp/rmq-0011-screenshot.png` to request a visual confirmation.
- Ran `plz-confirm form` to collect the screen context and artifact notes.

### Why
- Ensure the new capture flow is visually correct before moving to streaming or docs.

### What worked
- User confirmed `selected_json: true` and reported `match_quality=perfect` with no artifacts.

### What didn't work
- N/A

### What I learned
- The PNG output matches the Paper Pro display for a notebook view.

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- If we later see device firmware changes, re-validate screenshot mapping.

### What should be done in the future
- Consider adding a built-in validation command that captures and opens the PNG for quick checks.

### Code review instructions
- N/A (validation-only step).

### Technical details
- Command: `plz-confirm image --title \"RMQ-0011 screenshot validation\" --message \"Does this screenshot match the tablet display?\" --mode confirm --image /tmp/rmq-0011-screenshot.png`
- Command: `plz-confirm form --title \"RMQ-0011 screenshot details\" --schema /tmp/rmq-0011-screenshot-review-schema.json`

## Step 8: Add raw streaming endpoint and CLI

I added a `/api/v1/stream` endpoint that streams raw framebuffer bytes at a configurable rate, along with a `remarquee device stream` CLI command that writes the stream to a file for a fixed duration. This brings the basic streaming flow online without introducing compression or event streams yet.

**Commit (code):** N/A

### What I did
- Added a stream handler with rate validation and a single-connection limiter.
- Exposed `/api/v1/stream` in the device server.
- Added `remarquee device stream` with `--rate` and `--duration` flags.
- Ran `go test ./cmd/remarquee -count=1`.

### Why
- Streaming support is needed for live capture workflows and future UI mirroring.

### What worked
- The stream endpoint compiles and is wired into the server.

### What didn't work
- N/A

### What I learned
- The stream handler needs a concurrency limiter to avoid saturating the device.

### What was tricky to build
- Handling stream timeouts cleanly while still writing a partial output file.

### What warrants a second pair of eyes
- Confirm the raw stream rate behavior and file size growth match expectations.

### What should be done in the future
- Add compression (RLE/zstd) and event streaming endpoints.

### Code review instructions
- Review `remarquee/cmd/remarquee/cmds/device/stream_handler.go` and `.../stream.go`.
- Validate with `remarquee device stream --duration 5s --rate 200`.

### Technical details
- Command: `go test ./cmd/remarquee -count=1`

## Step 9: Document device capture API and add validation playbook

I wrote a guide-style device capture doc in `pkg/doc/topics` and added a ticket playbook for validation. This makes the feature discoverable via `remarquee help` and provides repeatable validation steps.

**Commit (code):** N/A

### What I did
- Added `remarquee/pkg/doc/topics/device-capture.md` with API + CLI usage.
- Created the playbook `playbook/01-device-capture-validation.md`.
- Updated RMQ-0011 tasks to split streaming vs events.

### Why
- The doc makes the workflow self-service and helps future contributors validate changes.

### What worked
- The documentation follows the Glazed style guide and includes runnable examples.

### What didn't work
- N/A

### What I learned
- The help system can expose device capture docs without extra wiring.

### What was tricky to build
- Balancing reference material with step-by-step guidance in a single doc.

### What warrants a second pair of eyes
- Confirm the doc examples match the actual CLI defaults and port numbers.

### What should be done in the future
- Add event endpoint documentation once those handlers land.

### Code review instructions
- Read `remarquee/pkg/doc/topics/device-capture.md`.
- Read `remarquee/ttmp/2026/01/11/RMQ-0011--device-side-framebuffer-capture/playbook/01-device-capture-validation.md`.

### Technical details
- N/A

## Step 10: Validate streaming on device

I rebuilt and redeployed the device binary, restarted the capture server, and exercised the `/api/v1/stream` endpoint via `remarquee device stream`. The stream produced a 58MB raw file over ~3 seconds, which confirms the endpoint is active and returning data.

**Commit (code):** N/A

### What I did
- Rebuilt the arm64 binary and copied it to `/home/root/remarquee` (via a `.new` swap to avoid overwriting a running binary).
- Restarted the device server.
- Pulled a 3-second raw stream to `/tmp/rmq-0011-stream.raw`.

### Why
- Validate the new streaming endpoint early on real hardware.

### What worked
- `remarquee device stream` returned a non-empty raw capture file (58MB).

### What didn't work
- Direct `scp` overwrote failed while the binary was running; resolved by stopping the process and swapping via `.new`.

### What I learned
- The device shell lacks `pkill`, so process management must use `ps` + `kill`.

### What was tricky to build
- Coordinating a safe update workflow for a running device binary.

### What warrants a second pair of eyes
- Confirm the stream file looks sane if decoded (future tool) and does not include repeated blank frames.

### What should be done in the future
- Add a small admin endpoint or PID file to simplify server restarts.

### Code review instructions
- N/A (validation-only step).

### Technical details
- Build: `GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o /tmp/remarquee-arm64 ./cmd/remarquee`
- Deploy: `scp /tmp/remarquee-arm64 root@10.11.99.1:/home/root/remarquee.new && ssh root@10.11.99.1 'mv /home/root/remarquee.new /home/root/remarquee'`
- Server: `ssh root@10.11.99.1 'nohup /home/root/remarquee device serve --bind :2718 --username admin --password password > /tmp/remarquee-device.log 2>&1 &'`
- Stream: `go run ./cmd/remarquee device stream --url http://10.11.99.1:2718 --username admin --password password --rate 200 --duration 3s --out /tmp/rmq-0011-stream.raw`
