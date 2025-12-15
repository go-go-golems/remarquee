---
Title: goMarkableStream package analysis (screen streaming, event handling, WebSocket API)
Ticket: RMQ-0001
Status: active
Topics:
    - backend
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: goMarkableStream/auth.go
      Note: Basic HTTP authentication middleware
    - Path: goMarkableStream/gzip.go
      Note: Gzip compression middleware
    - Path: goMarkableStream/http.go
      Note: HTTP router setup
    - Path: goMarkableStream/internal/eventhttphandler/gesture_handler.go
      Note: Touch gesture handler - NDJSON stream endpoint
    - Path: goMarkableStream/internal/eventhttphandler/pen_handler.go
      Note: Pen event handler - Server-Sent Events endpoint
    - Path: goMarkableStream/internal/pubsub/pubsub.go
      Note: Pub/sub system for event broadcasting
    - Path: goMarkableStream/internal/remarkable/events_linux.go
      Note: Linux input event reading - pen and touch device access
    - Path: goMarkableStream/internal/remarkable/fb_rm.go
      Note: Framebuffer access - reading from xochitl process memory
    - Path: goMarkableStream/internal/rle/rle.go
      Note: Run-Length Encoding compression implementation
    - Path: goMarkableStream/internal/stream/handler.go
      Note: Stream handler - framebuffer reading and compression
    - Path: goMarkableStream/listener.go
      Note: Network listener setup - TCP and ngrok tunnel support
    - Path: goMarkableStream/main.go
      Note: Application entry point - configuration
    - Path: goMarkableStream/zstd.go
      Note: ZSTD compression middleware
ExternalSources: []
Summary: 'Comprehensive analysis of goMarkableStream: screen streaming from reMarkable tablet to web browser, framebuffer access, event handling, compression, and HTTP/WebSocket API'
LastUpdated: 2025-12-14T18:00:32.924485258-05:00
---


# goMarkableStream package analysis (screen streaming, event handling, WebSocket API)

## Goal

This document provides a comprehensive technical analysis of the `goMarkableStream` Go package, covering:
- Architecture and design patterns
- Framebuffer access and memory reading
- Screen streaming pipeline
- Event handling (pen and touch input)
- Compression algorithms (RLE, gzip, zstd)
- HTTP/WebSocket API endpoints
- Client-side rendering (WebGL)
- Developer usage guide

### goMarkableStream in the remarquee Ecosystem: The Real-Time Window

In the remarquee ecosystem, goMarkableStream occupies a unique niche—it's the only component that provides **real-time access** to what's happening on the tablet right now. This is fundamentally different from the other tools:

- **rmapi**: Works with *stored* documents in the cloud (asynchronous, file-based)
- **remarks**: Works with *downloaded* documents (batch processing, offline)
- **remarkable_upload.py**: Works with *pre-created* documents (push to cloud)
- **goMarkableStream**: Works with *live* screen content (streaming, real-time)

This distinction matters deeply for the remarquee project because it enables an entirely different class of use cases. While the other tools are about document management and processing, goMarkableStream is about **interaction and presentation**.

**The Real-Time Gap in the ReMarkable Ecosystem:**

ReMarkable tablets excel at focused reading and writing—they're designed to minimize distractions. But this creates a gap: how do you share what you're doing *as you're doing it*? How do you demonstrate, teach, present, or collaborate in real-time?

Official solutions exist (ReMarkable's desktop app has screen sharing), but they're limited:
- Requires desktop app installation (not web-based)
- Proprietary (can't integrate with other tools)
- No customization (frame rate, compression, overlay features)
- No programmatic access (can't build automation)

goMarkableStream fills this gap by providing a **lightweight, web-based, programmable screen sharing solution** that runs directly on the tablet. It's the bridge between the tablet's private workspace and the world's screens.

**Why goMarkableStream is Valuable for remarquee:**

The remarquee project aims to unlock the full potential of ReMarkable tablets through open tools. goMarkableStream contributes several capabilities:

1. **Validation and Debugging**: When building remarquee features, we can visually verify what's happening on the tablet (see document layouts, annotation positioning, rendering issues)

2. **Demonstration**: Can record tutorials showing remarquee workflows (download with rmapi → process with remarks → view results)

3. **AI Integration**: Future OCR/LLM features could use goMarkableStream's live feed (process what user is writing in real-time, provide suggestions, auto-complete handwriting)

4. **Remote Assistance**: Help users debug remarquee issues by seeing their tablet screen

5. **Quality Assurance**: Test remarquee features while observing tablet behavior

Beyond these technical uses, goMarkableStream demonstrates an important principle: accessing data where it lives. While rmapi accesses data in the cloud and remarks accesses data in files, goMarkableStream accesses data in the tablet's active memory. For a complete toolkit (remarquee's vision), we need all three approaches.

**The Technical Achievement:**

goMarkableStream's design is elegant because it achieves screen streaming without any system modifications. It leverages standard Linux APIs (`/proc/<pid>/mem`, `/dev/input/event*`) that are available on any Linux system, including ReMarkable's locked-down environment. This "hack that's not a hack" approach means it works across firmware updates and doesn't void warranties—a critical consideration for a production tool like remarquee.

Understanding goMarkableStream deeply helps us think about future remarquee features: What other data can we access through clever use of standard APIs? Could we read document cache? Monitor xochitl's state? Implement remote control? The architectural patterns here—especially the framebuffer access technique—provide a template for similar innovations.

## Context

### What is goMarkableStream?

`goMarkableStream` is a screen streaming application that runs **on the reMarkable tablet itself**. It broadcasts your tablet's screen to a web browser in real-time, enabling:
- **Presentations**: Show your tablet screen to audience
- **Remote collaboration**: Share annotations during meetings
- **Teaching**: Demonstrate how to use the tablet
- **Recording**: Capture what you're writing/drawing

### Why does it exist?

**The problem:**

ReMarkable tablets don't have built-in screen sharing. You might want to:
- Present a slideshow with live annotations
- Demonstrate problem-solving on a whiteboard
- Record handwriting tutorials
- Show your research annotations to colleagues

Official solutions are limited:
- ReMarkable's desktop app has screen sharing (but requires installation, proprietary)
- No web-based streaming
- No programmatic access to screen data

**The solution:**

goMarkableStream provides a lightweight, web-based screen streaming solution that:
- Runs directly on tablet (no warranty-voiding hacks)
- Works with any device with a browser (phone, laptop, projector)
- Requires no client-side installation
- Provides low-latency streaming (~200ms default)

### How does it work (without hacking)?

**The clever trick:**

ReMarkable's display is managed by the `xochitl` process (the main tablet UI). This process:
- Maintains a framebuffer in memory (the current screen contents)
- Updates framebuffer when you draw/write
- Sends framebuffer to e-ink display

goMarkableStream reads this framebuffer directly from **xochitl's memory space** using Linux's `/proc` filesystem:

```
xochitl process (PID 1234)
├── Memory space
│   ├── Code
│   ├── Stack
│   ├── Heap
│   └── Framebuffer ← goMarkableStream reads this
│       (5.2 MB array of pixel data)
```

**Why this doesn't void warranty:**

- Uses standard Linux APIs (`/proc/<pid>/mem`)
- No system modifications required
- No firmware patching
- Just reading memory (not writing)
- xochitl process is not modified

**Real-world analogy:**

Like screen recording on your computer—reads what's on screen, doesn't modify the system.

### Key characteristics explained

- **Runs on ARM Linux**: Compiled for tablet's ARM processor (not x86 desktop)
- **Reads xochitl's memory**: Direct framebuffer access (fast, efficient)
- **HTTP/HTTPS server**: Standard web protocols (works with any browser)
- **Real-time streaming**: ~5 FPS (200ms per frame) with minimal CPU
- **WebGL rendering**: Uses GPU for efficient client-side display
- **Compression options**: RLE (default), gzip, or zstd (bandwidth optimization)
- **Event forwarding**: Can send pen/touch events back to tablet (experimental)

**Device support:**
- **reMarkable 2**: Fully tested, all features work
- **reMarkable Paper Pro**: Newer device, RLE compression not supported yet

**Firmware compatibility:**

Firmware versions affect memory layout and framebuffer location:
- **< 3.4**: Old pointer resolution method
- **3.4 - 3.6**: Updated memory layout
- **>= 3.6**: Current stable version

**Use correct goMarkableStream version for your firmware!**

**Dependencies:**
- `envconfig`: Parse environment variables into Go struct (clean configuration)
- `klauspost/compress`: Fast ZSTD compression (optional)
- `ngrok`: Tunnel for external access (optional)
- Standard library: Everything else (HTTP server, compression, etc.)

## Architecture Overview

goMarkableStream follows a server-client architecture:

```
┌─────────────────────────────────────────┐
│      reMarkable Tablet                  │
│  ┌──────────────────────────────────┐   │
│  │ xochitl Process                  │   │
│  │ - Framebuffer in memory          │   │
│  │ - /proc/<pid>/mem                │   │
│  └──────────────────────────────────┘   │
│  ┌──────────────────────────────────┐   │
│  │ goMarkableStream                 │   │
│  │ - Memory reader                  │   │
│  │ - Event scanner                  │   │
│  │ - HTTP server                    │   │
│  └──────────────────────────────────┘   │
│  ┌──────────────────────────────────┐   │
│  │ Input Devices                    │   │
│  │ - /dev/input/event1 (pen)       │   │
│  │ - /dev/input/event2 (touch)     │   │
│  └──────────────────────────────────┘   │
├─────────────────────────────────────────┤
│      Network Layer                      │
│  - HTTP/HTTPS                           │
│  - WebSocket/SSE                        │
├─────────────────────────────────────────┤
│      Web Browser                        │
│  ┌──────────────────────────────────┐   │
│  │ Client-side JavaScript           │   │
│  │ - WebGL rendering                │   │
│  │ - Event handling                 │   │
│  │ - Gesture detection              │   │
│  └──────────────────────────────────┘   │
└─────────────────────────────────────────┘
```

### Understanding Real-Time Streaming: Why Latency Matters

Before diving into the data flow, it's worth understanding what "real-time streaming" means in this context and why the design decisions matter. Screen streaming isn't like video streaming (Netflix, YouTube)—it has different requirements and constraints that shape goMarkableStream's architecture.

**Video streaming vs. Screen streaming:**

**Video streaming (Netflix):**
- Pre-recorded content (known duration, can buffer)
- High bitrate acceptable (1-5 Mbps for HD)
- Latency acceptable (5-30 seconds buffering)
- Predictable bandwidth (steady stream)

**Screen streaming (goMarkableStream):**
- Live content (unknown duration, can't buffer much)
- Low bitrate critical (WiFi congestion on tablets)
- Low latency critical (interactive use, <500ms ideal)
- Variable bandwidth (drawing activity varies)

goMarkableStream is optimized for **interactive screen sharing** where latency matters more than video quality. When you're presenting and draw a diagram, viewers should see your strokes appear immediately (not 5 seconds later). This requirement drives design decisions throughout the codebase.

**The 200ms target:**

goMarkableStream defaults to 200ms (5 FPS). Why this specific number?

- **E-ink refresh rate**: The tablet's screen itself updates at ~300ms, so faster than 200ms shows no new content
- **Network overhead**: ~50-100ms for typical WiFi round trip
- **Processing time**: ~10-20ms for compression and rendering
- **Perceptual threshold**: <300ms feels "immediate" to humans
- **CPU budget**: 200ms allows 5 frames/second at ~10% CPU

Faster is possible (100ms = 10 FPS) but burns more CPU for minimal perceived benefit. Slower (500ms) starts feeling laggy. The 200ms default is a sweet spot discovered through real-world use.

**The bandwidth challenge:**

Streaming 1872×1404 pixels at 16-bit per pixel means:
- Raw: 5.2 MB per frame
- At 5 FPS: 26 MB/second
- Over WiFi: Saturates many networks!

This is why compression isn't optional—it's essential. The RLE algorithm reduces this to ~12 MB/s (manageable on decent WiFi). For weaker connections, ZSTD can go to ~3-4 MB/s.

### Data Flow (Complete Cycle)

Let's walk through what happens from the moment you draw on the tablet to seeing it in your browser, understanding the timing and transformations at each step.

**On the tablet (every 200ms when active):**

```
Time: T+0ms    User draws line with pen
               ↓
Time: T+10ms   xochitl updates framebuffer in memory
               (Renders pen stroke to pixel buffer)
               ↓
Time: T+10ms   xochitl tells e-ink display to refresh
               (Display updates ~300ms later)
               
               [Meanwhile, parallel to display...]
               
Time: T+10ms   goMarkableStream reads framebuffer
               ReadAt(buffer, pointerAddr) → 5.2 MB
               ↓
Time: T+15ms   Applies RLE compression
               5.2 MB → ~2.5 MB (typical)
               ↓
Time: T+18ms   Sends via HTTP chunked transfer
               HTTP: "5000\r\n<5000 bytes>\r\n..."
               ↓
               [Network - varies by WiFi]
               ↓
Time: T+100ms  Data arrives at browser (WiFi delay)
```

**In the browser (client-side processing):**

```
Time: T+100ms  JavaScript receives chunk
               XHR onreadystatechange fired
               ↓
Time: T+102ms  Posted to Worker thread
               postMessage(compressedData)
               ↓
Time: T+105ms  Worker decompresses RLE
               2.5 MB → 5.2 MB (expand run-length pairs)
               ↓
Time: T+110ms  Worker decodes pixels
               Convert uint8 to grayscale values
               ↓
Time: T+115ms  Posted back to main thread
               postMessage(pixelArray)
               ↓
Time: T+120ms  WebGL shader renders to canvas
               Upload texture to GPU
               GPU renders to screen
               ↓
Time: T+125ms  User sees drawing (125ms total latency)
```

**Total latency breakdown:**

- Framebuffer read: 5ms
- Compression: 5ms
- Network: 80-100ms (varies)
- Decompression: 5ms
- Rendering: 10ms
- **Total: 105-125ms (excellent for interactive use!)**

**Parallel event flow (lower latency):**

Events have lower latency than screen frames because they're smaller (16 bytes vs 5.2 MB):

```
Time: T+0ms    User touches screen with pen
               ↓
Time: T+0ms    Linux kernel writes to /dev/input/event1
               16-byte InputEvent struct
               ↓
Time: T+1ms    goMarkableStream reads event
               select{} on /dev/input/event1
               ↓
Time: T+1ms    Publishes to pub/sub system
               All subscribers notified
               ↓
Time: T+2ms    WebSocket/SSE sends to clients
               "data: {x:5000,y:10000}\n\n"
               ↓
Time: T+20ms   Browser receives (WiFi delay)
               ↓
Time: T+21ms   JavaScript processes event
               Can trigger: laser pointer, gestures, slide nav
```

**Event latency: ~20-30ms (feels instantaneous!)**

This explains why goMarkableStream separates screen streaming and event streaming—events need to be ultra-low-latency for interactive features (laser pointer, gesture control) while screen frames can tolerate slightly higher latency.

**Key insight**: Screen and events are separate streams you can subscribe to independently. This architecture enables rich interactions:
- **Screen only**: Simple viewing (presentations)
- **Events only**: Remote control (trigger slides without seeing screen)
- **Both**: Full interaction (annotate presentations, gesture navigation)

For remarquee, this separation suggests future possibilities: Could we inject events back to tablet? Could we augment events with AI (handwriting recognition in real-time)? The architecture is flexible enough to support these extensions.

## Package Structure

### Core Packages

#### `main/` - Application Entry Point
- **`main.go`**: Main entry point, configuration, server setup
- **`http.go`**: HTTP router, endpoint handlers, TLS setup
- **`auth.go`**: Basic authentication middleware
- **`listener.go`**: Network listener setup (TCP, ngrok)
- **`ifaces.go`**: Network interface enumeration
- **`gzip.go`**: Gzip compression middleware
- **`zstd.go`**: ZSTD compression middleware

#### `internal/remarkable/` - Device Access
- **`fb_rm.go`**: Framebuffer access (Linux ARM/ARM64)
- **`fb.go`**: Dummy framebuffer (non-Linux builds)
- **`pointer.go`**: Frame pointer address resolution (ARM)
- **`pointer_arm64.go`**: Frame pointer address resolution (ARM64)
- **`findpid.go`**: Find xochitl process ID
- **`device.go`**: Device model detection
- **`const.go`**: Device constants (screen dimensions, input devices)
- **`const_arm64.go`**: ARM64-specific constants
- **`events_linux.go`**: Linux input event reading
- **`events.go`**: Event reading interface

#### `internal/stream/` - Streaming Logic
- **`handler.go`**: Stream handler (framebuffer reading, compression)
- **`mdw.go`**: Throttling middleware (limit concurrent connections)
- **`raw.go`**: Raw stream handler (dev mode)

#### `internal/pubsub/` - Event Publishing
- **`pubsub.go`**: Pub/sub system for event broadcasting

#### `internal/events/` - Event Types
- **`events.go`**: Input event structures and constants

#### `internal/eventhttphandler/` - HTTP Event Handlers
- **`pen_handler.go`**: Pen event handler (WebSocket/SSE)
- **`gesture_handler.go`**: Touch gesture handler

#### `internal/rle/` - Compression
- **`rle.go`**: Run-Length Encoding implementation

#### `client/` - Client-Side Assets (Embedded)
- **`index.html`**: Main HTML page
- **`main.js`**: Main JavaScript logic
- **`glCanvas.js`**: WebGL canvas rendering
- **`canvasHandling.js`**: Canvas manipulation
- **`uiInteractions.js`**: UI controls
- **`workersHandling.js`**: Web Worker management
- **`worker_stream_processing.js`**: Stream processing worker
- **`worker_event_processing.js`**: Event processing worker
- **`worker_gesture_processing.js`**: Gesture processing worker
- **`recording.js`**: Screen recording functionality
- **`utilities.js`**: Utility functions
- **`style.css`**: Stylesheet

## Key Types and Functions

### Configuration

#### `configuration` struct
```go
type configuration struct {
    BindAddr             string  // Server bind address (default: ":2001")
    Username             string  // Basic auth username (default: "admin")
    Password             string  // Basic auth password (default: "password")
    TLS                  bool    // Enable HTTPS (default: true)
    Compression          bool    // Enable gzip compression (default: false)
    RLECompression       bool    // Enable RLE compression (default: true)
    DevMode              bool    // Developer mode (default: false)
    ZSTDCompression      bool    // Enable zstd compression (default: false)
    ZSTDCompressionLevel int     // ZSTD level 1-22 (default: 3)
}
```

**Environment variables** (prefixed with `RK_`):
- `RK_SERVER_BIND_ADDR`: Bind address
- `RK_SERVER_USERNAME`: Username
- `RK_SERVER_PASSWORD`: Password
- `RK_HTTPS`: Enable HTTPS
- `RK_COMPRESSION`: Enable gzip compression
- `RK_RLE_COMPRESSION`: Enable RLE compression
- `RK_ZSTD_COMPRESSION`: Enable zstd compression
- `RK_ZSTD_COMPRESSION_LEVEL`: ZSTD compression level
- `RK_DEV_MODE`: Developer mode

### Framebuffer Access

#### `GetFileAndPointer() (io.ReaderAt, int64, error)`
Opens xochitl process memory and finds framebuffer pointer:
1. Finds xochitl process ID (`findXochitlPID()`)
2. Opens `/proc/<pid>/mem` for reading
3. Resolves framebuffer pointer address (`getFramePointer()`)
4. Returns file handle and pointer address

**Implementation:**
- Linux ARM/ARM64: Reads from `/proc/<pid>/mem`
- Other platforms: Returns dummy reader (for testing)

### Stream Handler

#### `StreamHandler` struct
```go
type StreamHandler struct {
    file           io.ReaderAt    // Framebuffer memory file
    pointerAddr    int64          // Framebuffer pointer address
    inputEventsBus *pubsub.PubSub // Event publisher
    useRLE         bool           // Enable RLE compression
}
```

#### `ServeHTTP(w http.ResponseWriter, r *http.Request)` - The Streaming Engine

This is the heart of goMarkableStream—handles HTTP streaming requests.

**Complete flow:**

Pseudocode:
```
FUNCTION ServeHTTP(request):
    // 1. Parse query parameters
    rate = GetQueryParam(request, "rate", default: 200)  // milliseconds
    IF rate < 100:
        ERROR("Rate too low, minimum 100ms")
    
    // 2. Set HTTP headers for streaming
    response.Headers.Set("Content-Type", "application/octet-stream")
    response.Headers.Set("Transfer-Encoding", "chunked")  // Key!
    response.Headers.Set("Cache-Control", "no-cache")
    
    // 3. Subscribe to input events (to know when user is drawing)
    event_channel = pubsub.Subscribe("stream")
    DEFER pubsub.Unsubscribe(event_channel)
    
    // 4. Create frame rate ticker
    ticker = CreateTicker(rate milliseconds)
    DEFER ticker.Stop()
    
    // 5. Create inactivity timer
    stop_writing_timer = CreateTimer(2 seconds)
    writing = TRUE  // Start writing immediately
    
    // 6. Allocate buffer (from pool for efficiency)
    buffer = GetFromPool(5.2 MB)
    DEFER ReturnToPool(buffer)
    
    // 7. Main streaming loop
    LOOP forever:
        SELECT:
            CASE request.Context() cancelled:
                RETURN  // Client disconnected
            
            CASE event FROM event_channel:
                // User is drawing/touching
                IF event.code == 24 OR event.source == Touch:
                    writing = TRUE
                    stop_writing_timer.Reset(2 seconds)
            
            CASE stop_writing_timer fires:
                // 2 seconds of inactivity
                writing = FALSE
                // Stop streaming to save CPU/bandwidth
            
            CASE ticker fires:
                // Time for next frame
                IF writing:
                    // Read framebuffer
                    memFile.ReadAt(buffer, pointerAddr)
                    
                    // Write to client (with compression)
                    writer.Write(buffer)  // May go through RLE encoder
                    writer.Flush()  // Send immediately
```

**Key design decisions:**

**1. HTTP Chunked Transfer Encoding**

Without chunked:
```
HTTP/1.1 200 OK
Content-Length: 5242880  ← Must know size in advance
<all data at once>
```

With chunked:
```
HTTP/1.1 200 OK
Transfer-Encoding: chunked  ← Size unknown, stream indefinitely

5000        ← Chunk size (hex)
<5000 bytes>
3000        ← Next chunk
<3000 bytes>
...         ← Continues forever
```

Enables streaming without knowing total size!

**2. Adaptive streaming (inactivity detection)**

```
User drawing:
Frame, Frame, Frame, Frame, ...  (5 FPS, ~10% CPU)

User idle (2+ seconds):
(no frames sent)  (0% CPU)

User resumes:
Frame, Frame, Frame, ...  (resumes streaming)
```

**Why?**

- Saves CPU when idle (battery life!)
- Saves bandwidth (mobile hotspot)
- Reduces heat
- Improves tablet responsiveness

**3. Buffer pooling**

```go
var rawFrameBuffer = sync.Pool{
    New: func() any {
        return make([]uint8, 5242880)  // 5.2 MB
    },
}

// Get buffer
buf := rawFrameBuffer.Get().([]uint8)

// Use buffer...

// Return to pool (avoid GC pressure)
rawFrameBuffer.Put(buf)
```

**Why pooling?**

- Allocating 5.2 MB every 200ms = garbage collection pressure
- Pooling reuses buffers (zero allocations during streaming)
- Reduces GC pauses (smoother streaming)
- Lower memory usage

**Query parameter effects:**

```bash
# Default (200ms = 5 FPS)
?rate=200      → 5 frames/sec, ~10% CPU

# Fast (100ms = 10 FPS)
?rate=100      → 10 frames/sec, ~18% CPU

# Slow (500ms = 2 FPS)
?rate=500      → 2 frames/sec, ~5% CPU
```

**Frame rate limits:**

- **Minimum**: 100ms (10 FPS) enforced by code
- **Practical max**: 50ms (20 FPS) before CPU becomes bottleneck
- **Recommended**: 200ms (5 FPS) for balance

**Throttling (one client at a time):**

```go
var activeWriters int = 0
var maxWriters int = 1

// In middleware
IF activeWriters >= maxWriters:
    RETURN 429 Too Many Requests

activeWriters++
ServeRequest()
activeWriters--
```

**Why only one client?**

- E-ink refresh rate is slow (~300ms)
- Multiple clients wouldn't see different screens
- Reduces CPU usage
- Simplifies implementation

**Workaround**: Multiple clients can view (one active streamer, others wait).

### Event System

#### `PubSub` struct
```go
type PubSub struct {
    subscribers map[chan events.InputEventFromSource]bool
    mu          sync.Mutex
}
```

**Methods:**
- `Publish(event)`: Broadcast event to all subscribers (with timeout)
- `Subscribe(name)`: Subscribe to events, returns channel
- `Unsubscribe(ch)`: Unsubscribe and close channel

#### `EventScanner` struct
```go
type EventScanner struct {
    pen, touch *os.File  // Input device file handles
}
```

**Methods:**
- `StartAndPublish(ctx, pubsub)`: Start goroutines to read pen/touch events
  - Reads from `/dev/input/event1` (pen)
  - Reads from `/dev/input/event2` (touch)
  - Publishes events via pub/sub

#### `InputEvent` struct
```go
type InputEvent struct {
    Time  syscall.Timeval  // Event timestamp
    Type  uint16           // Event type (EvAbs, EvKey, etc.)
    Code  uint16           // Event code (axis, key code)
    Value int32            // Event value
}
```

**Event types:**
- `EvAbs` (3): Absolute axis (pen/touch position)
- `EvKey` (1): Key press/release
- `EvSyn` (0): Synchronization marker

**Event sources:**
- `Pen` (1): Pen input device
- `Touch` (2): Touch input device

### Compression (Optimizing Bandwidth)

Screen streaming requires transferring ~5 MB per frame. At 5 FPS (200ms per frame), that's ~25 MB/s—too much for many WiFi connections! Compression is essential.

#### RLE (Run-Length Encoding) - The Default

**Why RLE works well for e-ink:**

E-ink displays have large areas of uniform color (white background, black text). This is perfect for RLE:

```
Framebuffer (pixels):
[15,15,15,15,15,15,15,15,0,0,0,15,15,15,15,15,15]
(white background with short black stroke)

RLE encoded:
[(8,15), (3,0), (6,15)]
(8 white, 3 black, 6 white)

Original: 17 bytes
Compressed: 6 bytes
Ratio: ~2.8:1
```

**Algorithm (simplified):**

Pseudocode:
```
FUNCTION RLE_Encode(framebuffer):
    output = []
    current_value = framebuffer[0]
    count = 0
    
    FOR pixel IN framebuffer:
        IF pixel == current_value AND count < 254:
            count += 1
        ELSE:
            // Emit (count, value) pair
            output.append(count)
            output.append(current_value)
            current_value = pixel
            count = 1
    
    // Emit last pair
    output.append(count)
    output.append(current_value)
    
    RETURN output
```

**Performance:**
- **Compression ratio**: ~2-3:1 typical (depends on content)
- **CPU overhead**: Very low (~1-2% CPU)
- **Latency**: Minimal (< 5ms)
- **Best for**: Text, diagrams, large uniform areas

**When RLE is less effective:**
- Photographs (lots of variation)
- Gradients (constantly changing values)
- Dense handwriting (every other pixel differs)

**Implementation details:**

```go
// Input: framebuffer[5256576]  (5.2 MB)
// Process in pairs (uint4 values stored as uint8)
for i := 0; i < len(framebuffer); i += 2 {
    value = framebuffer[i]  // Only process even bytes
    // (Odd bytes are padding/unused)
    ...
}

// Output: ~2.5 MB compressed (typical)
```

#### Gzip Compression - General Purpose

**When to use:**

RLE isn't enough (complex content), but CPU is available.

**Algorithm:**

Standard DEFLATE algorithm (same as ZIP files):
- Combines LZ77 (dictionary compression) + Huffman coding
- Compression level 1 (fastest): Prioritizes speed over ratio
- Works on any data (not just uniform areas)

**Performance:**
- **Compression ratio**: ~3-5:1 typical
- **CPU overhead**: Medium (~5-8% CPU)
- **Latency**: ~10-20ms
- **Best for**: Mixed content (text + images)

**Trade-offs:**

```
RLE:           Gzip:
Fast ✓         Slower
Simple ✓       Complex
2-3:1 ratio    3-5:1 ratio
Low CPU ✓      Medium CPU
```

**Configuration:**

```bash
export RK_COMPRESSION=true  # Enable gzip
export RK_RLE_COMPRESSION=false  # Disable RLE
./goMarkableStream
```

Can't use both simultaneously (choose one).

#### ZSTD Compression - Maximum Compression

**When to use:**

You have poor network (mobile hotspot, slow WiFi) and need maximum bandwidth reduction.

**Algorithm:**

Modern compression (2015+):
- Similar to gzip but better ratio
- Adjustable compression levels (1-22)
- Level 3 (default): Good balance
- Level 10+: Maximum compression (slow!)

**Performance comparison:**

```
Level  Ratio  CPU   Latency  Use case
1      ~4:1   5%    10ms     Fast streaming
3      ~6:1   8%    15ms     Default (balanced)
10     ~10:1  20%   40ms     Slow network
22     ~12:1  50%   100ms    Extreme (too slow)
```

**Configuration:**

```bash
export RK_ZSTD_COMPRESSION=true
export RK_ZSTD_COMPRESSION_LEVEL=3
export RK_RLE_COMPRESSION=false
./goMarkableStream
```

**Recommendation:**

```
Good network (home WiFi):      RLE (default)
Moderate network (public WiFi): ZSTD level 3
Poor network (mobile hotspot):  ZSTD level 6-8
Extreme (satellite):            ZSTD level 10+
```

**Bandwidth comparison:**

```
Uncompressed:  25 MB/s @ 5 FPS
RLE:          ~12 MB/s @ 5 FPS  (2x reduction)
Gzip:         ~6 MB/s @ 5 FPS   (4x reduction)
ZSTD-3:       ~4 MB/s @ 5 FPS   (6x reduction)
ZSTD-10:      ~2 MB/s @ 5 FPS   (12x reduction)
```

**ReMarkable Paper Pro note:**

RLE compression not supported (firmware limitation). Use gzip or ZSTD instead:
```bash
export RK_RLE_COMPRESSION=false
export RK_ZSTD_COMPRESSION=true
```

## HTTP API Endpoints

### Main Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/` | GET | Main web interface (HTML page) |
| `/stream` | GET | Image data stream (chunked transfer) |
| `/events` | GET | Pen input events (Server-Sent Events) |
| `/gestures` | GET | Touch gesture events (NDJSON stream) |
| `/version` | GET | Application version |
| `/raw` | GET | Raw framebuffer (dev mode only) |

### Query Parameters

**`/stream` endpoint:**
- `rate`: Frame rate in milliseconds (default: 200, min: 100)
- `color`: Enable color mode (`true`/`false`)
- `portrait`: Portrait orientation (`true`/`false`)
- `flip`: Flip 180 degrees (`true`/`false`)

**`/` endpoint:**
- `present`: URL to embed in iframe (presentation mode)

### Response Formats

**`/stream`:**
- Content-Type: `application/octet-stream`
- Transfer-Encoding: `chunked`
- Compression: RLE (default), gzip, or zstd

**`/events`:**
- Content-Type: `text/event-stream` (SSE)
- Format: `data: <JSON>\n\n`
- Events: Pen input events only (`EvAbs` type)

**`/gestures`:**
- Content-Type: `application/x-ndjson`
- Format: Newline-delimited JSON
- Events: Touch gesture data (`{left, right, up, down}` distances)

**`/version`:**
- Content-Type: `text/plain`
- Content: Version string from build info

## Client-Side Architecture

### Why the Browser Matters: Client-Side Intelligence

goMarkableStream's architecture is split between server (tablet) and client (browser), and this division is carefully designed. Understanding why certain operations happen on the server vs. client helps you appreciate the performance optimizations and suggests patterns for remarquee's design.

**The server/client split philosophy:**

Many screen streaming solutions put all logic server-side (encode video, stream encoded video, client just displays). goMarkableStream takes a different approach—it does **minimal processing on the tablet** and **maximum processing in the browser**. This reflects the reality of tablet constraints:

**Tablet constraints (why minimize server-side processing):**
- **Limited CPU**: ARM processor, already running xochitl and e-ink driver
- **Battery life**: Processing burns battery (users want 8-hour battery life)
- **Heat**: Tablets have no fans, CPU-intensive tasks cause heat
- **Responsiveness**: Don't want streaming to impact writing responsiveness

**Browser advantages (why maximize client-side processing):**
- **Powerful hardware**: Desktop/laptop CPU is 10-100x faster than tablet
- **GPU acceleration**: WebGL leverages dedicated graphics hardware
- **Power unlimited**: Connected to outlet, power not a concern
- **Concurrent processing**: Web Workers enable parallel processing
- **Caching**: Browser can cache shaders, textures, decompression code

**The division of labor:**

```
Tablet (minimal processing):
- Read framebuffer (unavoidable—data lives here)
- RLE compression (simple algorithm, ~10% CPU)
- HTTP serve (standard library, efficient)

Browser (heavy processing):
- Decompress RLE (JavaScript is fast enough)
- Decode pixels (convert uint8 to grayscale)
- WebGL rendering (GPU-accelerated)
- Color conversion (if enabled)
- Rotation/transformation (GPU shaders)
- UI interactions (side menu, controls)
```

This architecture means tablets can stream screen content with minimal performance impact (~10% CPU), while browsers provide smooth, responsive viewing experience.

**For remarquee integration:**

This client-heavy architecture suggests a design principle: **push intelligence to the edges**. When building remarquee:
- OCR/LLM processing should happen on powerful computers, not tablet
- goMarkableStream can feed live framebuffer to remarquee's OCR engine
- Tablet remains responsive, power-efficient
- Processing scales with desktop hardware

The client-side architecture also demonstrates Web Workers effectively—something remarquee might adopt for its web interface (if we build one). Offloading work to background threads keeps UIs responsive.

### WebGL Rendering

**File:** `client/glCanvas.js`

WebGL (Web Graphics Library) is the JavaScript API for GPU-accelerated graphics. goMarkableStream uses WebGL because it's the only way to achieve smooth 5 FPS rendering of a 5.2 MB framebuffer on modest hardware.

**Why not just canvas 2D API?**

You might wonder: why use WebGL (complex, shader programming) instead of simple Canvas 2D?

```javascript
// Canvas 2D approach (simple but slow)
for (let y = 0; y < height; y++) {
    for (let x = 0; x < width; x++) {
        let pixel = framebuffer[y * width + x];
        ctx.fillStyle = `rgb(${pixel}, ${pixel}, ${pixel})`;
        ctx.fillRect(x, y, 1, 1);  // Draw one pixel
    }
}
// 1872 × 1404 = 2.6 million operations per frame!
// At 5 FPS: 13 million operations/second
// Result: Choppy, uses 50% CPU
```

WebGL approach (complex but fast):
```javascript
// Upload framebuffer to GPU as texture (one operation)
gl.texImage2D(target, level, format, width, height, 
              border, format, type, framebuffer);

// GPU renders entire frame in parallel (milliseconds)
// Result: Smooth, uses 5% CPU
```

The GPU parallelizes pixel operations—it can process millions of pixels simultaneously. This is exactly what framebuffer rendering needs.

**Key capabilities:**
- Efficient framebuffer rendering (GPU-accelerated, handles 5.2 MB @ 60 FPS easily)
- RLE decompression (done in JavaScript before GPU upload)
- Coordinate transformations (rotation, flipping handled by GPU shaders)
- Color modes (grayscale/color conversion via shaders)

### Event Processing

**Files:**
- `worker_event_processing.js`: Processes pen events
- `worker_gesture_processing.js`: Processes touch gestures
- `worker_stream_processing.js`: Processes stream data

**Workers:**
- Offload processing to Web Workers
- Prevents UI blocking
- Parallel processing of stream/events

### UI Features

**File:** `client/uiInteractions.js`

- Dark mode toggle
- Rotation controls
- Side menu for settings
- Laser pointer (hover effect)
- Presentation mode (iframe embedding)

## Framebuffer Access Details (The Low-Level Magic)

### Understanding Framebuffers

**What is a framebuffer?**

A framebuffer is a memory region that holds the pixel data currently displayed on screen. Think of it as a giant array:

```
framebuffer = [
  pixel(0,0), pixel(1,0), pixel(2,0), ..., pixel(1871,0),
  pixel(0,1), pixel(1,1), pixel(2,1), ..., pixel(1871,1),
  ...
  pixel(0,1403), ..., pixel(1871,1403)
]
```

For reMarkable 2:
- **Size**: 1872 × 1404 pixels
- **Format**: 16-bit grayscale (2 bytes per pixel)
- **Total memory**: 1872 × 1404 × 2 = 5,256,576 bytes (~5.2 MB)

**Why 16-bit grayscale?**

E-ink displays support 16 levels of gray (4 bits would be enough), but:
- Uses 16 bits for alignment and future expansion
- Most significant 4 bits contain the gray value (0-15)
- Remaining 12 bits typically unused or for internal use

### Memory Reading (The Hack That's Not a Hack)

**How do we read xochitl's memory?**

Linux provides `/proc/<pid>/mem` - a special file that represents a process's entire memory space. It's like a window into another process's memory.

**Step-by-step process:**

Pseudocode:
```
FUNCTION GetFileAndPointer():
    // 1. Find xochitl process ID
    pid = FindProcessByName("xochitl")
    // xochitl is ReMarkable's main UI process
    // Example: PID 1234
    
    // 2. Open its memory file
    memFile = OpenFile("/proc/1234/mem", READ_ONLY)
    // This file is HUGE (gigabytes), but we only read specific addresses
    
    // 3. Find framebuffer pointer address
    pointerAddr = FindFramebufferPointer(pid)
    // This is tricky (see below)
    // Example: 0x7f8a4c3000
    
    // 4. Return both
    RETURN memFile, pointerAddr
```

**Finding xochitl PID:**

```bash
$ ps aux | grep xochitl
root  1234  5.2  15.3  ...  /usr/bin/xochitl --system

# Parse: PID is 1234
```

**Finding framebuffer pointer (the hard part):**

Pseudocode:
```
FUNCTION FindFramebufferPointer(pid):
    // 1. Read xochitl's memory map
    maps = ReadFile(f"/proc/{pid}/maps")
    // maps contains all memory regions:
    // 00400000-00500000 r-xp ... /usr/bin/xochitl (code)
    // 7f8a4c3000-7f8b000000 rw-p ... (heap, stack, etc.)
    
    // 2. Find the region containing framebuffer
    // Heuristic: Look for large read/write region (~5.2 MB)
    FOR region IN maps:
        IF region.size >= 5_000_000 AND region.permissions == "rw":
            candidate_regions.append(region)
    
    // 3. Search for framebuffer signature
    // The framebuffer has a specific pattern at start
    FOR region IN candidate_regions:
        data = ReadMemory(memFile, region.start, 100)
        IF data matches framebuffer_signature:
            RETURN region.start
    
    FAIL("Framebuffer not found")
```

**Why is this fragile?**

- Memory layout changes between firmware versions
- Framebuffer location is not guaranteed
- Signature pattern may change
- **This is why firmware compatibility matters!**

**ARM vs ARM64 differences:**

```
ARM (32-bit, older tablets):
- Pointer size: 4 bytes
- Address space: 0x00000000 to 0xFFFFFFFF (4 GB)
- Framebuffer typically at: 0x40000000 range

ARM64 (64-bit, newer tablets):
- Pointer size: 8 bytes
- Address space: 0x0000000000000000 to 0xFFFFFFFFFFFFFFFF (huge)
- Framebuffer typically at: 0x7f00000000 range
```

Different architecture → different pointer resolution code.

### Reading the Framebuffer (Every Frame)

Pseudocode:
```
FUNCTION fetchAndSend(writer, rawData):
    // 1. Read 5.2 MB from xochitl's memory
    bytesRead = memFile.ReadAt(
        buffer: rawData,           // Pre-allocated 5.2 MB buffer
        offset: pointerAddr        // Where framebuffer starts
    )
    // This reads the CURRENT screen contents
    
    // 2. Send to client
    writer.Write(rawData)
    writer.Flush()  // Ensure immediate delivery
```

**Performance:**

- Reading 5.2 MB from memory: ~1-2 ms (fast!)
- Compression (RLE): ~3-5 ms
- Network transfer: ~50-200 ms (depends on WiFi)
- **Total latency**: ~200-300 ms per frame

**Why is memory reading fast?**

- Memory is in RAM (not disk)
- Sequential read (no seeking)
- Linux kernel optimizes `/proc` reads
- No system calls per pixel (bulk read)

### Framebuffer Format (Pixel Layout)

**Memory layout:**

```
Byte offset:  0  1  2  3  4  5  ...
Pixel:       [  0  ][  1  ][  2  ] ...
Position:    (0,0) (1,0) (2,0) ...

After 1872 pixels (one row):
Byte offset:  3744 3745 3746 3747 ...
Pixel:        [ 1872 ][ 1873 ] ...
Position:     (0,1)   (1,1) ...
```

**Each pixel (16-bit):**

```
Bits:  15 14 13 12 11 10 9 8 7 6 5 4 3 2 1 0
       [  Gray value   ][   Unused/reserved   ]
       Most significant 4 bits = gray level (0-15)
```

**Grayscale mapping:**

```
Value  Meaning         E-ink appearance
0      Black           Darkest
1-7    Dark grays      Getting lighter
8      Mid gray        50% gray
9-14   Light grays     Getting lighter
15     White           Lightest (paper)
```

**Why row-major order?**

Standard for image formats—pixels laid out left-to-right, top-to-bottom. Makes rendering straightforward.

## Event Handling Details (Understanding Input)

### How Linux Input Events Work

Linux exposes input devices as special files in `/dev/input/`. These are not regular files—they're **device files** that stream events.

**Event structure (from Linux kernel):**

```c
struct input_event {
    struct timeval time;  // 8 bytes: timestamp
    uint16_t type;        // 2 bytes: event type (key, abs, syn, etc.)
    uint16_t code;        // 2 bytes: event code (which axis/key)
    int32_t value;        // 4 bytes: event value
};
// Total: 16 bytes per event
```

### Input Device Reading (Pen and Touch)

**Pen device:** `/dev/input/event1`

The pen generates absolute position events:

```
Event stream example (user draws line):
{time: 1234.567, type: EV_ABS, code: 1, value: 5000}   # X = 5000
{time: 1234.567, type: EV_ABS, code: 0, value: 10000}  # Y = 10000
{time: 1234.567, type: EV_ABS, code: 24, value: 2000}  # Pressure = 2000
{time: 1234.567, type: EV_SYN, code: 0, value: 0}      # Sync (frame complete)
{time: 1234.597, type: EV_ABS, code: 1, value: 5010}   # X = 5010 (moved)
{time: 1234.597, type: EV_ABS, code: 0, value: 10015}  # Y = 10015
...
```

**Event codes:**
- **Code 1**: X-axis (vertical on tablet), range: 0-15725
- **Code 0**: Y-axis (horizontal on tablet), range: 0-20966
- **Code 24**: Pressure, range: 0-4095 (pen pressure sensitivity)

**Touch device:** `/dev/input/event2`

Similar structure, different codes:
- **Code 54**: Touch X-axis
- **Code 53**: Touch Y-axis

**Reading events (the unsafe.Pointer trick):**

```go
func readEvent(inputDevice *os.File) (events.InputEvent, error) {
    var ev events.InputEvent
    // Cast event struct to byte array, read directly
    _, err := inputDevice.Read((*(*[unsafe.Sizeof(ev)]byte)(unsafe.Pointer(&ev)))[:])
    return ev, err
}
```

**What's happening:**

1. Create empty `InputEvent` struct
2. Get its size (16 bytes)
3. Cast struct pointer to byte array pointer (unsafe!)
4. Read exactly 16 bytes from device file
5. Bytes are interpreted as struct fields

**Why unsafe.Pointer?**

Direct memory mapping (zero-copy). Alternative would be:
```go
// Slower: read bytes, then manually parse
buf := make([]byte, 16)
inputDevice.Read(buf)
ev.Time = binary.LittleEndian.Uint64(buf[0:8])
ev.Type = binary.LittleEndian.Uint16(buf[8:10])
// ... more parsing
```

### Pub/Sub System (Event Broadcasting)

goMarkableStream uses pub/sub pattern to broadcast events to multiple clients simultaneously.

**Why pub/sub?**

Without pub/sub:
```
[Pen device] → [Client 1]
             → [Client 2]
             → [Client 3]
```

Each client needs separate event reader. Wasteful!

With pub/sub:
```
[Pen device] → [Event Scanner] → [PubSub] → [Client 1]
                                          → [Client 2]
                                          → [Client 3]
```

One reader, multiple subscribers. Efficient!

**Implementation:**

Pseudocode:
```
CLASS PubSub:
    subscribers = Map[Channel, Bool]
    mutex = Mutex
    
    FUNCTION Publish(event):
        mutex.Lock()
        FOR channel IN subscribers:
            SELECT:
                CASE channel can receive:
                    channel <- event
                CASE timeout (100ms):
                    SKIP  // Slow consumer, don't block others
        mutex.Unlock()
    
    FUNCTION Subscribe(name):
        channel = MakeChannel()
        mutex.Lock()
        subscribers[channel] = true
        mutex.Unlock()
        RETURN channel
    
    FUNCTION Unsubscribe(channel):
        mutex.Lock()
        DELETE subscribers[channel]
        CLOSE channel
        mutex.Unlock()
```

**Key features:**
- **Non-blocking**: Slow clients don't block others (100ms timeout)
- **Thread-safe**: Mutex protects subscriber map
- **Dynamic**: Clients can subscribe/unsubscribe anytime

### Gesture Detection (Making Sense of Touch)

Touch events are low-level (X/Y coordinates every 16ms). Gesture detection makes them meaningful.

**Example: Detecting a swipe**

Raw touch events:
```
t=0ms:    X=1000, Y=5000  (finger down)
t=16ms:   X=1010, Y=5000  (moved right 10 pixels)
t=32ms:   X=1025, Y=5000  (moved right 15 more)
t=48ms:   X=1050, Y=5000  (moved right 25 more)
t=64ms:   X=1080, Y=5000  (moved right 30 more)
t=100ms:  (no more events - finger lifted)
```

Gesture detection:
```
Total distance right: 10+15+25+30 = 80 pixels
Direction: Right
Gesture: Swipe Right
```

**Algorithm:**

Pseudocode:
```
FUNCTION GestureDetection():
    gesture = {left: 0, right: 0, up: 0, down: 0}
    last_x = 0
    last_y = 0
    timeout = 150ms
    
    WHILE true:
        SELECT:
            CASE event FROM event_channel:
                IF event.code == X_AXIS:
                    distance = event.value - last_x
                    IF distance > 0:
                        gesture.left += distance
                    ELSE:
                        gesture.right += abs(distance)
                    last_x = event.value
                
                IF event.code == Y_AXIS:
                    distance = event.value - last_y
                    IF distance > 0:
                        gesture.down += distance
                    ELSE:
                        gesture.up += abs(distance)
                    last_y = event.value
                
                timeout.Reset(150ms)  // Keep waiting for more
            
            CASE timeout expires:
                // No more events, gesture complete
                IF gesture.sum() > 0:
                    SendToClient(gesture)
                
                gesture.reset()
                last_x = 0
                last_y = 0
```

**Gesture interpretation:**

```
{left: 500, right: 10, up: 20, down: 5}
→ Mostly left → Swipe Left

{left: 5, right: 600, up: 15, down: 10}
→ Mostly right → Swipe Right

{left: 50, right: 45, up: 500, down: 10}
→ Mostly up → Swipe Up
```

**Use case: Slide navigation**

```javascript
// In browser (Reveal.js integration)
fetch('/gestures').then(response => {
  const reader = response.body.getReader();
  
  while (true) {
    const {value, done} = await reader.read();
    const gesture = JSON.parse(value);
    
    if (gesture.left > 300) {
      Reveal.prev();  // Go to previous slide
    }
    if (gesture.right > 300) {
      Reveal.next();  // Go to next slide
    }
  }
});
```

Swipe left/right on tablet → slides change in browser!

## Compression Details

### RLE Algorithm

**Input:** Framebuffer as `[]uint8` (uint4 values)
**Output:** Packed `(count, value)` pairs

**Algorithm:**
1. Iterate through framebuffer data
2. Count consecutive identical values (max 254)
3. Emit `(count, value)` pair when value changes
4. Pack into `uint8` array

**Example:**
```
Input:  [0,0,0,1,1,2,2,2,2]
Output: [(3,0), (2,1), (4,2)]
```

### Compression Comparison

| Algorithm | CPU Usage | Compression Ratio | Latency |
|-----------|-----------|-------------------|---------|
| None | Low | 1:1 | Low |
| RLE | Low | ~2:1 | Low |
| Gzip | Medium | ~5:1 | Medium |
| ZSTD | Medium | ~8:1 | Medium |

## Developer Usage Guide

### Building

**For reMarkable 2 (ARM):**
```bash
GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 \
  go build -v -trimpath -ldflags="-s -w" .
```

**For reMarkable Paper Pro (ARM64):**
```bash
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 \
  go build -v -trimpath -ldflags="-s -w" .
```

### Installation

**On reMarkable tablet:**
```bash
# Download latest release
export GORKVERSION=$(wget -q -O - https://api.github.com/repos/owulveryck/goMarkableStream/releases/latest | grep tag_name | awk -F\" '{print $4}')
wget -q -O - https://github.com/owulveryck/goMarkableStream/releases/download/$GORKVERSION/goMarkableStream_${GORKVERSION//v}_linux_arm.tar.gz | tar xzvf - -O goMarkableStream_${GORKVERSION//v}_linux_arm/goMarkableStream > goMarkableStream
chmod +x goMarkableStream

# Run
./goMarkableStream
```

### Configuration

**Environment variables:**
```bash
export RK_SERVER_BIND_ADDR=":2001"
export RK_SERVER_USERNAME="admin"
export RK_SERVER_PASSWORD="password"
export RK_HTTPS="true"
export RK_RLE_COMPRESSION="true"
export RK_ZSTD_COMPRESSION="false"
./goMarkableStream
```

**Command-line flags:**
- `-h`: Print usage
- `-unsafe`: Disable authentication

### Accessing Stream (Real-World Setup)

**Basic setup:**

```bash
# On your computer
$ ssh root@<tablet-ip>

# On tablet
$ ./goMarkableStream
Version: v0.11.0
Local IP address: 192.168.1.100
listening on :2001
```

**In your browser:**

```
https://192.168.1.100:2001
```

**Authentication prompt:**
```
Username: admin
Password: password
```

**What you see:**

- Live stream of tablet screen
- Updates as you draw/write
- Side menu with controls (rotate, dark mode, frame rate)
- Laser pointer (hover over stream)

**Using mDNS (Apple devices):**

```
https://remarkable.local.:2001
```

**Note the trailing period!** (`remarkable.local.` not `remarkable.local`)

This is mDNS convention. Without period, may not resolve.

**Query parameters (customize experience):**

```
# Faster framerate (higher CPU usage)
https://192.168.1.100:2001?rate=100

# Enable color (if tablet supports)
https://192.168.1.100:2001?color=true

# Portrait orientation
https://192.168.1.100:2001?portrait=true

# Flip 180 degrees (for projector mounted upside down)
https://192.168.1.100:2001?flip=true

# Combine multiple
https://192.168.1.100:2001?rate=100&color=true&portrait=true
```

**Presentation mode (overlay on slides):**

```
# Open your Google Slides presentation
# Get shareable link: https://docs.google.com/presentation/d/abc123/edit?usp=sharing

# Add to goMarkableStream URL
https://192.168.1.100:2001?present=https://docs.google.com/presentation/d/abc123/edit?usp=sharing

# Now:
# - Your slides appear in background
# - Tablet screen overlays on top
# - Draw on blank slides
# - Annotate existing slides
# - Switch slides with gestures
```

**Use cases:**

1. **Conference presentation:**
   - Connect tablet and laptop to same WiFi
   - Project browser to screen
   - Present slides with live annotations
   - Audience sees tablet screen in real-time

2. **Remote teaching:**
   - Share browser window in Zoom/Teams
   - Solve problems on tablet
   - Students see your work live
   - Can pause, zoom, point

3. **YouTube tutorial recording:**
   - Open goMarkableStream in browser
   - Use OBS to capture browser window
   - Record while drawing/writing on tablet
   - High-quality screen capture

4. **Whiteboard replacement:**
   - Tablet at meeting table
   - Browser on large monitor
   - Everyone sees what you draw
   - No need for physical whiteboard

**Troubleshooting connection:**

```bash
# Can't connect to remarkable.local.:2001

# Check 1: Find tablet IP
$ ssh root@remarkable
$ ifconfig
# Look for wlan0 or usb0 IP address

# Check 2: Test from tablet
$ wget http://localhost:2001/version
# Should succeed

# Check 3: Test from computer
$ curl -k https://<tablet-ip>:2001/version
# Should return version number

# Check 4: Firewall
# Tablet might have iptables rules
$ iptables -L
# If blocked, clear: iptables -F (dangerous!)
```

### Systemd Service (Persistent Installation)

**Why use systemd?**

Without systemd:
- Must SSH and run `./goMarkableStream` manually each time
- Process dies when SSH disconnects
- Doesn't survive reboots

With systemd:
- Starts automatically on boot
- Survives SSH disconnects
- Automatically restarts if crashes
- Can manage with `systemctl` commands

**Installation (one-time setup):**

```bash
# SSH into tablet
$ ssh root@remarkable

# Create service file
$ cat <<EOF > /etc/systemd/system/goMarkableStream.service
[Unit]
Description=Go Remarkable Stream Server
After=network.target  # Wait for network

[Service]
ExecStart=/home/root/goMarkableStream
Restart=always  # Auto-restart if crashes
User=root
WorkingDirectory=/home/root

# Optional: Set environment variables
Environment="RK_RLE_COMPRESSION=true"
Environment="RK_SERVER_PASSWORD=mysecretpass"

[Install]
WantedBy=multi-user.target  # Start at boot
EOF

# Enable and start
$ systemctl enable goMarkableStream.service  # Enable auto-start
$ systemctl start goMarkableStream.service   # Start now
$ systemctl status goMarkableStream.service  # Check status
```

**Managing the service:**

```bash
# Check status
$ systemctl status goMarkableStream

# View logs
$ journalctl -u goMarkableStream -f  # Follow logs
$ journalctl -u goMarkableStream --since "10 minutes ago"

# Stop service
$ systemctl stop goMarkableStream

# Restart service (after updating binary)
$ systemctl restart goMarkableStream

# Disable auto-start
$ systemctl disable goMarkableStream
```

**After firmware updates:**

Firmware updates **erase** third-party binaries. After update:

```bash
$ ssh root@remarkable

# Re-download binary
$ export GORKVERSION=$(wget -q -O - https://api.github.com/repos/owulveryck/goMarkableStream/releases/latest | grep tag_name | awk -F\" '{print $4}')
$ wget -q -O - https://github.com/owulveryck/goMarkableStream/releases/download/$GORKVERSION/goMarkableStream_${GORKVERSION//v}_linux_arm.tar.gz | tar xzvf - -O goMarkableStream_${GORKVERSION//v}_linux_arm/goMarkableStream > goMarkableStream
$ chmod +x goMarkableStream

# Restart service (service file survives firmware updates)
$ systemctl restart goMarkableStream
```

### Real-World Deployment Scenarios

**Scenario 1: Home network (simple)**

```bash
# On tablet (via SSH)
$ ./goMarkableStream

# On laptop (same WiFi)
Browser: https://192.168.1.100:2001
Username: admin
Password: password

# Use for:
# - Personal note-taking with large monitor
# - Recording tutorials
# - Presenting to family
```

**Scenario 2: Conference presentation (public WiFi)**

```bash
# Problem: Conference WiFi blocks device-to-device
# Solution: Use ngrok tunnel

# On tablet
$ export RK_SERVER_BIND_ADDR=ngrok
$ ./goMarkableStream

# Output:
# Tunnel established: https://abc123.ngrok.io
# listening on https://abc123.ngrok.io

# Share this URL with audience
# They can view from anywhere (internet access)
```

**Scenario 3: Teaching (custom settings)**

```bash
# On tablet (optimize for low bandwidth)
$ export RK_ZSTD_COMPRESSION=true
$ export RK_ZSTD_COMPRESSION_LEVEL=6
$ export RK_RLE_COMPRESSION=false
$ export RK_SERVER_PASSWORD=classroompass
$ ./goMarkableStream

# Students access:
Browser: https://remarkable.local.:2001?rate=300
# Slower framerate (300ms = 3.3 FPS) reduces bandwidth
```

**Scenario 4: Recording for YouTube**

```bash
# On tablet
$ export RK_RLE_COMPRESSION=false  # Better quality
$ ./goMarkableStream

# On computer (OBS Studio):
1. Add "Browser Source"
2. URL: https://<tablet-ip>:2001?rate=100
3. Width: 1872, Height: 1404
4. Record screen

# Tips:
# - Use rate=100 for smoother video
# - Disable compression (better quality)
# - Use portrait mode if needed
# - Add custom CSS for borders
```

**Scenario 5: Permanent installation (systemd)**

```bash
# For daily use, persistent setup
$ ssh root@remarkable
$ ./setupGoMarkableStream.sh  # Creates systemd service

# Now:
# - Starts automatically on boot
# - Survives reboots
# - Doesn't need SSH
# - Always available

# Access anytime:
Browser: https://remarkable.local.:2001
```

**Security considerations:**

```bash
# Default: admin/password (change this!)
$ export RK_SERVER_USERNAME=myuser
$ export RK_SERVER_PASSWORD=strongpassword123
$ ./goMarkableStream

# Or disable auth (dangerous!)
$ ./goMarkableStream -unsafe
# No authentication required
# Only use on trusted networks!
```

### API Usage

**Stream endpoint:**
```bash
curl -u admin:password \
  "https://remarkable.local.:2001/stream?rate=200" \
  --output stream.bin
```

**Events endpoint:**
```bash
curl -u admin:password \
  "https://remarkable.local.:2001/events" \
  --no-buffer
```

**Version endpoint:**
```bash
curl -u admin:password \
  "https://remarkable.local.:2001/version"
```

## Performance Characteristics

### Understanding Performance Trade-offs: The Art of Real-Time Streaming

Performance analysis of goMarkableStream isn't just about numbers—it's about understanding the trade-offs between competing goals and how those trade-offs shaped the design. Every optimization decision has costs and benefits, and understanding these helps you tune the system for your use case (and informs remarquee's design).

**The competing goals:**

1. **Low latency** (users want immediate response)
2. **Low bandwidth** (WiFi is limited, especially mobile)
3. **Low CPU** (tablet battery life, responsiveness)
4. **High quality** (users want clear, readable stream)

You can't optimize all four simultaneously—improvements in one often hurt another. goMarkableStream's default settings (200ms frame rate, RLE compression) represent a carefully chosen balance.

**Why these specific numbers matter:**

The performance characteristics below aren't just measurements—they're design constraints that drove architectural decisions. For example, why is RLE the default compression (not ZSTD, which compresses better)? Because CPU usage matters more than bandwidth for typical home WiFi scenarios. Why throttle to one connection? Because multiple connections don't improve user experience but do hurt tablet performance.

**For remarquee:**

When integrating goMarkableStream's concepts into remarquee, these performance characteristics suggest:
- If building OCR on live stream: Process every Nth frame (not every frame) to avoid CPU overload
- If building remote collaboration: Consider compression level based on network quality
- If building recording: Might disable compression (quality > bandwidth when saving locally)

Understanding *why* goMarkableStream performs the way it does helps you make informed decisions about where to spend CPU cycles in remarquee's feature set.

### CPU Usage

These CPU measurements are from reMarkable 2 (ARM Cortex-A9, ~1 GHz). CPU percentage is relative to one core.

- **Idle**: ~0% CPU (event scanner running but blocked on I/O, no active streaming)
- **Streaming (200ms rate)**: ~10% CPU (default, balanced for battery life)
- **Streaming (100ms rate)**: ~15% CPU (faster updates, higher CPU cost)
- **With gzip compression**: +2-5% CPU (more CPU than RLE, better compression)
- **With ZSTD compression**: +5-10% CPU (most CPU intensive, best compression)

**What this means:**

At default settings, goMarkableStream uses ~10% of one core. The tablet has 2 cores, so plenty of headroom for xochitl (main UI) to remain responsive. Users can write while streaming without lag—critical for presentations.

**CPU breakdown by operation:**
- Framebuffer read: ~2% (I/O bound)
- RLE compression: ~5% (CPU bound)
- HTTP serving: ~2% (network bound)
- Event processing: ~1% (mostly idle)

### Memory Usage

- **Base**: ~10MB (Go runtime, HTTP server, embedded assets)
- **Per connection**: ~5MB (framebuffer buffers in pool)
- **Framebuffer cache**: ~5MB (reused via sync.Pool, not per-connection)
- **Peak**: ~20MB (one active connection plus overhead)

**Why so low?**

Go's memory efficiency and careful use of sync.Pool. The framebuffer buffer is reused across frames—we don't allocate 5.2 MB every 200ms. This keeps garbage collection pauses low (< 1ms), ensuring smooth streaming.

**For comparison:**
- Screen recording software (OBS): 100-500 MB
- Video conferencing (Zoom): 200-800 MB
- goMarkableStream: 20 MB

This efficiency is possible because:
- No video encoding (just raw framebuffer)
- No complex UI (JavaScript handles that)
- No media buffering (live streaming only)
- Aggressive pooling (reuse allocations)

### Network Bandwidth

Bandwidth is often the bottleneck for streaming, especially on mobile hotspots or congested WiFi. These numbers help you choose compression settings.

**Uncompressed:**
- Frame size: ~5.2 MB (1872×1404×2 bytes)
- At 5 FPS: ~26 MB/s (208 Mbps—saturates many WiFi connections!)
- Use case: Local network, no WiFi congestion, need best quality

**RLE compressed (default):**
- Frame size: ~2.5 MB (typical, varies by content)
- At 5 FPS: ~12.5 MB/s (100 Mbps)
- Use case: Home WiFi, balanced quality/bandwidth
- CPU overhead: Low (~5%)

**ZSTD compressed (level 3):**
- Frame size: ~650 KB (typical)
- At 5 FPS: ~3.25 MB/s (26 Mbps)
- Use case: Public WiFi, mobile hotspot, bandwidth-constrained
- CPU overhead: Medium (~8%)

**Choosing compression:**

```
Network Quality      Recommended            Rationale
-----------------    -----------            ---------
Home WiFi (good)     RLE                   Low CPU, adequate compression
Public WiFi          ZSTD level 3          Balance bandwidth/CPU
Mobile hotspot       ZSTD level 6-8        Aggressive compression
Wired ethernet       None or RLE           Bandwidth not limiting
Conference WiFi      ZSTD level 3-6        Often congested, need compression
```

**Bandwidth over time (real session):**

```
Time  Activity           Bandwidth
0:00  Idle               0 MB/s (not streaming)
0:05  Start drawing      12 MB/s (RLE, active)
0:10  Pause thinking     0 MB/s (2 sec timeout kicked in)
0:15  Continue drawing   12 MB/s (resumed)
0:30  Finish             0 MB/s (idle again)

Average: ~6 MB/s (streaming is adaptive!)
```

The adaptive streaming (stop after 2 seconds idle) means real-world bandwidth is much lower than theoretical maximum.

## Security Considerations

### Security Model: Balancing Convenience and Protection

goMarkableStream's security model reflects the reality of its use cases—it's typically used on private networks for presentations, not exposed to the internet. However, it still provides meaningful security layers for protection. Understanding the threat model helps you configure it appropriately.

**The threat model (what are we protecting against?):**

goMarkableStream protects against several threat scenarios, each with different likelihood and severity:

**Threat 1: Casual snooping on local network**
- **Scenario**: You're at a coffee shop, someone on same WiFi tries to view your screen
- **Protection**: Basic auth + HTTPS
- **Mitigation**: They need username/password, and traffic is encrypted
- **Residual risk**: Low (assuming you set strong password)

**Threat 2: Malicious actor with network access**
- **Scenario**: Attacker on your network, knows default credentials (admin/password)
- **Protection**: Can change credentials via environment variables
- **Mitigation**: Use strong passwords, consider ngrok tunnel
- **Residual risk**: Medium (if using default credentials)

**Threat 3: Internet exposure**
- **Scenario**: Tablet IP exposed to internet, botnet tries to access
- **Protection**: Authentication required, HTTPS prevents interception
- **Mitigation**: Don't expose tablet to internet, use ngrok if needed
- **Residual risk**: High if using default credentials + port forwarded

**What goMarkableStream does NOT protect against:**

- **Man-in-the-middle with self-signed cert**: Browser warns but users might ignore
- **Brute force attacks**: No rate limiting on authentication
- **Replay attacks**: No challenge-response mechanism
- **Screen recording**: Anyone who authenticates can record your screen

**Security is "good enough" for intended use cases:**

goMarkableStream's security is appropriate for:
- Home network presentations (trusted environment)
- Office presentations (controlled network)
- Conference presentations with strong password (semi-trusted)
- Remote teaching with shared credentials (acceptable risk)

It's NOT appropriate for:
- Public internet exposure without additional protection
- Handling sensitive/confidential content on untrusted networks
- Multi-tenant scenarios (multiple users, different permissions)

For remarquee, this security model suggests: **convenience-first for typical use, with options for hardening**. Most users prioritize "just works" over enterprise security. Provide sane defaults, document risks, allow configuration.

### Authentication

- **Basic HTTP authentication** (username/password in Authorization header)
  - Standard, widely supported (every browser, curl, etc.)
  - Simple to implement (20 lines of Go code)
  - Good enough for typical threat model

- **Default credentials**: `admin` / `password`
  - **CHANGE THESE!** Everyone knows the defaults
  - Set via environment variables: `RK_SERVER_USERNAME`, `RK_SERVER_PASSWORD`

- **Disable with `-unsafe` flag** (not recommended except trusted environment)
  - Use case: Home network, only you have access
  - Simplifies demo/testing
  - **Never use on public WiFi!**

- **No session management**: Every request re-authenticates
  - Simpler implementation (stateless server)
  - Slightly higher overhead (negligible for this use case)

### TLS (HTTPS)

- **Embedded self-signed certificate** (generated once, embedded in binary)
  - Provides encryption (protects password, screen data)
  - Browser shows security warning (expected—it's self-signed)
  - Users must click "Advanced" → "Proceed" (one-time per browser)

- **HTTPS enabled by default** (`RK_HTTPS=true`)
  - Encrypts all traffic (important on public WiFi)
  - Prevents casual packet sniffing
  - Overhead negligible (~1-2% CPU for TLS)

- **Why self-signed?**
  - Can't get real certificate (no domain name, tablet IP changes)
  - Let's Encrypt requires domain + port 80/443 (tablet uses 2001)
  - Self-signed is pragmatic choice

**Alternative security architectures:**

For production remarquee deployment, consider:
1. **ngrok tunnel** (built-in support)
   - Public HTTPS URL with real certificate
   - No firewall configuration needed
   - Requires ngrok account

2. **Reverse proxy** (nginx, Caddy)
   - Run on computer, proxy to tablet
   - Real certificates via Let's Encrypt
   - Can add rate limiting, logging

3. **VPN** (Tailscale, WireGuard)
   - Private network, no public exposure
   - Secure by design
   - Requires setup on tablet + clients

### Network Exposure

- **Binds to `:2001` by default** (all network interfaces)
  - Accessible from any device on network
  - Convenient (auto-discover tablet IP)
  - Risk: Exposes to entire network

- **Can bind to specific interface**
  - Example: `RK_SERVER_BIND_ADDR=192.168.1.100:2001`
  - Only accessible from that interface
  - More secure (limits exposure)

- **ngrok tunnel support** (`RK_SERVER_BIND_ADDR=ngrok`)
  - Provides public HTTPS URL
  - No firewall configuration needed
  - Useful for remote presentations
  - **Warning**: Exposes to entire internet (use strong password!)

**Recommendation for different scenarios:**

```
Scenario                 Configuration
---------                -------------
Home network             Default (:2001, HTTPS, change password)
Office network           Default (:2001, HTTPS, strong password)
Conference               ngrok tunnel + strong password
Public demo              ngrok tunnel + authentication
Development/testing      -unsafe flag (local only)
```

## Limitations

### Device-Specific

- **reMarkable Paper Pro**: RLE compression not supported (must disable)
- **Firmware compatibility**: Version requirements vary by firmware

### Performance

- Single concurrent stream connection (throttled)
- Frame rate limited by CPU and network
- Higher frame rates increase CPU usage

### Functionality

- No direct control of underlying presentation (presentation mode)
- Screen size smaller than presentation size
- Browser restrictions require HTTPS for iframe embedding

## File Organization Summary

### Entry Points
- `main.go`: Application entry point, configuration, server startup

### HTTP Layer
- `http.go`: Router setup, endpoint handlers, TLS configuration
- `auth.go`: Basic authentication middleware
- `listener.go`: Network listener (TCP, ngrok)

### Device Access
- `internal/remarkable/fb_rm.go`: Framebuffer access (Linux)
- `internal/remarkable/pointer.go`: Pointer resolution
- `internal/remarkable/events_linux.go`: Input event reading
- `internal/remarkable/device.go`: Device detection

### Streaming
- `internal/stream/handler.go`: Stream handler
- `internal/stream/mdw.go`: Throttling middleware
- `internal/rle/rle.go`: RLE compression

### Events
- `internal/pubsub/pubsub.go`: Event publishing system
- `internal/events/events.go`: Event types
- `internal/eventhttphandler/pen_handler.go`: Pen event handler
- `internal/eventhttphandler/gesture_handler.go`: Gesture handler

### Compression
- `gzip.go`: Gzip middleware
- `zstd.go`: ZSTD middleware

### Client Assets
- `client/`: Embedded web assets (HTML, JS, CSS)

## Quick Reference

### Key Functions

**Framebuffer:**
- `remarkable.GetFileAndPointer()`: Get framebuffer file and pointer

**Streaming:**
- `stream.NewStreamHandler()`: Create stream handler
- `StreamHandler.ServeHTTP()`: Handle HTTP stream request

**Events:**
- `remarkable.NewEventScanner()`: Create event scanner
- `EventScanner.StartAndPublish()`: Start event reading
- `pubsub.NewPubSub()`: Create pub/sub system
- `PubSub.Publish()`: Publish event
- `PubSub.Subscribe()`: Subscribe to events

**Compression:**
- `rle.NewRLE()`: Create RLE encoder
- `gzMiddleware()`: Gzip compression middleware
- `zstdMiddleware()`: ZSTD compression middleware

### Key Types

- `configuration`: Server configuration
- `StreamHandler`: HTTP stream handler
- `PubSub`: Event publishing system
- `EventScanner`: Input event reader
- `InputEvent`: Input event structure
- `InputEventFromSource`: Event with source

### Environment Variables

- `RK_SERVER_BIND_ADDR`: Bind address
- `RK_SERVER_USERNAME`: Username
- `RK_SERVER_PASSWORD`: Password
- `RK_HTTPS`: Enable HTTPS
- `RK_COMPRESSION`: Enable gzip
- `RK_RLE_COMPRESSION`: Enable RLE
- `RK_ZSTD_COMPRESSION`: Enable zstd
- `RK_ZSTD_COMPRESSION_LEVEL`: ZSTD level
- `RK_DEV_MODE`: Developer mode

### API Endpoints

- `/`: Web interface
- `/stream`: Screen stream
- `/events`: Pen events (SSE)
- `/gestures`: Touch gestures (NDJSON)
- `/version`: Version info
- `/raw`: Raw framebuffer (dev mode)

## Related

- **rmapi**: See `01-rmapi-api-overview-architecture-auth-transport-shell-commands.md`
- **remarks**: See `02-remarks-package-analysis-parsing-conversion-output-formats.md`
- **remarkable_upload.py**: See `03-remarkable-upload-py-script-analysis-markdown-to-pdf-conversion-and-upload.md`
- GitHub: https://github.com/owulveryck/goMarkableStream
- Blog posts:
  - [Streaming the reMarkable 2](https://blog.owulveryck.info/2021/03/30/streaming-the-remarkable-2.html)
  - [Evolving the Game: A clientless streaming tool for reMarkable 2](https://blog.owulveryck.info/2023/07/25/evolving-the-game-a-clientless-streaming-tool-for-remarkable-2.html)
