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
    - Path: remarquee/cmd/remarquee/main.go
      Note: Registers device command
    - Path: remarquee/pkg/devicecapture/pointer_arm64.go
      Note: Paper Pro pointer discovery
    - Path: remarquee/pkg/devicecapture/reader.go
      Note: Core framebuffer capture API
    - Path: remarquee/ttmp/2026/01/11/RMQ-0011--device-side-framebuffer-capture/analysis/01-gomarkablestream-framebuffer-streaming-analysis.md
      Note: Detailed analysis
    - Path: remarquee/ttmp/2026/01/11/RMQ-0011--device-side-framebuffer-capture/design-doc/01-remarquee-device-capture-api-cli-design.md
      Note: Integration design
ExternalSources: []
Summary: Diary for device-side framebuffer capture work
LastUpdated: 2026-01-11T20:09:37-05:00
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
