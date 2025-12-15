---
Title: rmapi API overview (architecture, auth, transport, shell commands)
Ticket: RMQ-0001
Status: active
Topics:
    - backend
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: rmapi/api/api.go
      Note: ApiCtx interface definition - main API abstraction
    - Path: rmapi/api/auth.go
      Note: Authentication flow - device token and user token management
    - Path: rmapi/api/sync15/apictx.go
      Note: Main ApiCtx implementation - all document operations
    - Path: rmapi/api/sync15/blobdoc.go
      Note: BlobDoc structure - document representation in hash tree
    - Path: rmapi/api/sync15/blobstorage.go
      Note: Blob storage operations - upload/download blobs
    - Path: rmapi/api/sync15/tree.go
      Note: HashTree implementation - document index synchronization
    - Path: rmapi/config/config.go
      Note: Config file path resolution and token management
    - Path: rmapi/config/url.go
      Note: API endpoint URL configuration
    - Path: rmapi/filetree/filetree.go
      Note: FileTreeCtx - in-memory file tree with path resolution
    - Path: rmapi/main.go
      Note: CLI entry point
    - Path: rmapi/shell/put.go
      Note: Upload command implementation with flags
    - Path: rmapi/shell/shell.go
      Note: Shell setup and command registration
    - Path: rmapi/transport/transport.go
      Note: HTTP client layer with authentication and CRC32C checksums
ExternalSources: []
Summary: 'Comprehensive analysis of rmapi Go package: architecture, authentication, sync15 protocol, shell commands, and developer usage guide'
LastUpdated: 2025-12-14T17:45:10.551569138-05:00
---


# rmapi API overview (architecture, auth, transport, shell commands)

## Goal

This document provides a comprehensive technical analysis of the `rmapi` Go package, covering:
- Package structure and organization
- Architecture layers and data flow
- Key types, interfaces, and symbols
- Authentication mechanism
- Sync15 protocol implementation
- Shell command mapping to API operations
- Developer usage guide

### The Bigger Picture: rmapi in the remarquee Ecosystem

Before diving into rmapi's internals, it's important to understand where it fits in the broader remarquee tool we're building. The remarquee project aims to create a unified, powerful toolkit for working with ReMarkable tablets, and rmapi serves as the **foundational layer**—the data access layer that everything else builds upon.

Think of the remarquee ecosystem as a stack:

**Layer 1: Data Access (rmapi)**  
Handles all communication with ReMarkable's cloud. This is what you're reading about now. It provides the fundamental operations: upload, download, list, delete. Without rmapi, none of the other tools would have access to your documents.

**Layer 2: Content Processing (remarks)**  
Once rmapi downloads documents, remarks takes over to extract and convert annotations. While rmapi deals with the cloud API and document metadata, remarks understands the *content*—it knows how to parse .rm binary files, extract highlights, and generate human-readable outputs.

**Layer 3: Content Creation (remarkable_upload.py)**  
This script reverses the flow: it takes human-created content (Markdown documentation) and uses rmapi to push it *to* the tablet. It's the "write" side of the workflow, converting your development documentation into tablet-readable PDFs.

**Layer 4: Real-Time Interaction (goMarkableStream)**  
While the other tools work with stored documents, goMarkableStream provides *live* access to what's happening on the tablet right now. It runs on the device itself and streams the screen to your browser, enabling presentations, remote collaboration, and recording.

**Layer 5: Intelligence (geppetto - future)**  
The planned OCR/LLM integration will sit on top of everything, using remarks to extract content, then applying AI to make it searchable, summarizable, and queryable.

Understanding rmapi deeply is crucial because it's the foundation. Every document uploaded by remarkable_upload.py goes through rmapi's sync protocol. Every document processed by remarks was downloaded via rmapi. The quality, reliability, and performance of rmapi directly impacts everything built on top of it. This document will help you understand not just how rmapi works, but *why* it's designed the way it is, so you can make informed decisions when building the unified remarquee tool.

## Context

### What is rmapi?

`rmapi` is a Go application that provides programmatic access to the ReMarkable Cloud API. Think of it as a bridge between your computer and ReMarkable's cloud storage—it lets you upload, download, and manage documents on your tablet without using the official apps or web interface.

### Why does it exist?

The ReMarkable tablet syncs documents through proprietary cloud APIs. While ReMarkable provides official desktop/mobile apps, they don't expose a programmatic interface for automation. `rmapi` reverse-engineers and implements these APIs, enabling:
- **Automated workflows**: Backup scripts, batch uploads, CI/CD integration
- **Custom integrations**: Build your own sync tools, document processors
- **Command-line access**: Manage documents via terminal without GUI

### What problem does it solve?

ReMarkable's cloud uses a sophisticated synchronization protocol called "Sync15" (version 1.5). This protocol handles:
- **Conflict detection**: Multiple devices editing simultaneously
- **Efficient sync**: Only transfer changed data, not entire documents
- **Data integrity**: SHA256 hashes ensure data consistency
- **Version control**: Generation numbers track document changes

Without `rmapi`, you'd need to:
- Reverse-engineer the API yourself
- Handle authentication token management
- Implement the hash-based sync protocol
- Deal with conflict resolution
- Parse proprietary document formats

### Key characteristics

- **Implements ReMarkable Cloud API v1.5 (Sync15)**: Full protocol implementation
- **Hash-based document tree synchronization**: Only syncs what changed
- **Supports PDF, EPUB, and native `.rmdoc` formats**: All tablet-supported formats
- **Interactive shell with autocomplete and globbing**: User-friendly CLI
- **Non-interactive mode**: Scriptable for automation
- **Library usage**: Can be imported as a Go package

### How it works (high-level)

1. **Authentication**: Two-token system (device token + user token)
2. **Sync**: Download document index from cloud (list of all documents with hashes)
3. **Local cache**: Maintain local copy of document tree
4. **Operations**: Upload, download, move, delete documents
5. **Sync back**: Upload changes, update document index, handle conflicts

## Architecture Overview

### The Philosophy Behind rmapi's Design

rmapi's architecture reflects a classic separation of concerns pattern, but understanding *why* it's structured this way helps you navigate the codebase effectively. The architecture didn't emerge by accident—it evolved through years of reverse-engineering ReMarkable's API and adapting to their protocol changes.

The key insight driving rmapi's design is that **ReMarkable's Cloud API is complex and changes over time**. Different firmware versions use different sync protocols, authentication methods, and data formats. By layering the architecture, rmapi isolates these concerns:

- **If authentication changes**: Only the auth layer needs updating
- **If sync protocol changes**: Only the sync15 implementation changes (or add sync16)
- **If document formats change**: Only the archive layer changes
- **Shell commands stay stable**: Users don't see the internal complexity

This is critical for the remarquee project: when we build the unified tool, we can leverage rmapi's stable high-level API (`ApiCtx` interface) without worrying about low-level protocol details. If ReMarkable releases firmware 4.0 with a new sync protocol, only rmapi's internals would need updating—our remarquee tool would continue working with the same interface.

### The Layered Architecture

rmapi follows a layered architecture where each layer has a specific responsibility and communicates only with adjacent layers:

```
┌─────────────────────────────────────────┐
│         Shell Layer (shell/)           │  ← Interactive CLI commands
│  What: User-facing command interface   │
│  Why: Provides familiar Unix-like UX   │
│  Depends on: API Layer only            │
├─────────────────────────────────────────┤
│         API Layer (api/)                │  ← High-level operations
│  What: Document CRUD operations        │
│  Why: Protocol-agnostic interface      │
│  Depends on: Transport + Sync15        │
│  ┌──────────────────────────────────┐  │
│  │   sync15/ (Sync15 implementation) │  │
│  │   What: Protocol implementation   │  │
│  │   - HashTree (document index)     │  │
│  │   - BlobStorage (blob operations) │  │
│  │   - BlobDoc (document structure)  │  │
│  └──────────────────────────────────┘  │
├─────────────────────────────────────────┤
│      Transport Layer (transport/)       │  ← HTTP client & auth
│  What: HTTP requests with auth         │
│  Why: Isolates network concerns        │
│  Depends on: Model layer               │
├─────────────────────────────────────────┤
│      Model Layer (model/)               │  ← Data structures
│  What: Data transfer objects           │
│  Why: Shared types across layers       │
│  Depends on: Nothing (pure data)       │
├─────────────────────────────────────────┤
│      Archive Layer (archive/)           │  ← Document packaging
│  What: .rmdoc format handling          │
│  Why: Encapsulates format knowledge    │
│  Depends on: Model layer               │
└─────────────────────────────────────────┘
```

**Why this specific layering?**

Each layer serves a distinct purpose that emerged from practical needs:

The **Archive Layer** exists because ReMarkable documents aren't simple files—they're complex packages containing PDFs, metadata, annotations, and page data. By isolating this complexity in the archive layer, the rest of the codebase can work with simple abstractions (upload a file, download a document) without understanding ZIP structures or metadata formats.

The **Transport Layer** provides a clean HTTP abstraction that handles authentication, retries, and error mapping. This is crucial because ReMarkable's API has quirks: it uses CRC32C checksums (not MD5), special headers for filenames, and specific error codes that need interpretation. The transport layer translates these quirks into clean Go errors and HTTP semantics.

The **API Layer** is where the magic happens—it's the bridge between "I want to upload a document" (high-level intent) and "PUT these 7 blobs, update these indices, handle conflicts" (low-level protocol). This layer implements the ApiCtx interface, which is what external code (including our future remarquee tool) will use. The interface is stable even though the implementation (sync15) might change.

The **Shell Layer** is interesting because it's not just a thin wrapper—it adds significant value: autocomplete, globbing, path navigation, error messages. This layer demonstrates how to *use* the API layer effectively, serving as both a user tool and a reference implementation for developers.

### Data Flow Example: Upload Document (Complete Walkthrough)

Let's walk through what happens when you run `rmapi put document.pdf` in the shell. This touches every layer of the architecture.

**User perspective:**
```bash
$ rmapi
[/]> put document.pdf /Books
uploading: [document.pdf]...
OK
```

**What happens internally:**

**Layer 1: Shell (`shell/put.go`)**
```
User types: "put document.pdf /Books"

Shell parses command:
- Command: "put"
- Arguments: ["document.pdf", "/Books"]
- Flags: none

Shell calls: ctx.api.UploadDocument(
    parentId: <Books folder UUID>,
    sourceDocPath: "document.pdf",
    notify: true,
    coverpage: nil
)
```

**Layer 2: API (`api/sync15/apictx.go`)**
```
UploadDocument() orchestrates the entire upload:

1. Validate file:
   - Check file exists locally
   - Check extension is supported (.pdf, .epub, .rmdoc, .rm)

2. Create temporary directory for packaging

3. Call archive.Prepare() to package document
   → Goes to Layer 3
```

**Layer 3: Archive (`archive/doc.go`)**
```
Prepare() packages document into .rmdoc format:

1. Generate UUID for document (e.g., "f7e4a2c1-...")

2. Create metadata file:
   {
     "docName": "document",
     "collectionType": "DocumentType",
     "parent": "<Books UUID>",
     "lastModified": "1702598400000",
     "version": 1
   }
   Save as: f7e4a2c1-....metadata

3. Copy PDF content:
   Copy document.pdf → f7e4a2c1-....pdf

4. Create content file:
   {
     "fileType": "pdf",
     "pages": ["page-uuid-1", "page-uuid-2", ...],
     ...
   }
   Save as: f7e4a2c1-....content

5. Create pagedata file:
   (Page metadata)
   Save as: f7e4a2c1-....pagedata

Return: List of {name, path, extension} for each file
```

**Layer 2 (continued): API**
```
UploadDocument() receives packaged files:

4. For each file:
   a. Compute SHA256 hash
   b. Create Entry {hash, documentID, size}
   c. Call blobStorage.UploadBlob()
      → Goes to Layer 4
```

**Layer 4: BlobStorage (`api/sync15/blobstorage.go`)**
```
UploadBlob() uploads file to cloud:

1. Open file reader
2. Call transport.PutStream()
   → Goes to Layer 5
```

**Layer 5: Transport (`transport/transport.go`)**
```
PutStream() makes HTTP request:

1. Compute CRC32C checksum of file data

2. Create HTTP request:
   PUT /sync/v3/files/{hash}
   Headers:
     Authorization: Bearer <user-token>
     rm-filename: f7e4a2c1-....pdf
     x-goog-hash: crc32c=<checksum>
     content-type: application/octet-stream
   Body: <file data>

3. Send request
4. Check response (200 OK = success)
```

**Layer 4 (continued): Back to API**
```
5. After all files uploaded:
   - Add all file entries to BlobDoc
   - Compute document hash (SHA256 of all file hashes)
   - Upload document index (special blob with .docSchema extension)

6. Call Sync() to update root:
   - Add BlobDoc to HashTree
   - Recompute root hash
   - Upload root index blob
   - Update root pointer (with generation number)
   - Handle conflicts if needed (see Sync pseudocode above)

7. If notify=true:
   - Call SyncComplete() API
   - Tablet syncs within ~30 seconds
```

**Back to Shell:**
```
Shell receives success:
Prints: "OK"
Adds document to local file tree
```

**Total HTTP requests for one PDF:**
- 4 PUT requests (upload files)
- 1 PUT request (upload document index)
- 1 PUT request (upload root index)
- 1 PUT request (update root pointer)
- Optional: 1 POST request (sync complete notification)

**Performance notes:**
- Files uploaded in parallel (default: 20 concurrent)
- Hashing is CPU-intensive (SHA256 + CRC32C)
- Network is usually the bottleneck

## Package Structure

### Understanding the Package Organization

The rmapi codebase is organized into packages that reflect both technical concerns (separation of responsibilities) and the evolution of the ReMarkable API itself. When you first look at the structure, you might wonder why there are so many packages for what seems like a simple task (upload/download files). The answer lies in the complexity hiding beneath that simplicity.

ReMarkable's cloud infrastructure uses a sophisticated synchronization protocol designed for mobile devices with intermittent connectivity. It needs to handle:
- Multiple devices editing simultaneously (conflict resolution)
- Efficient bandwidth usage (hash-based sync, only transfer changes)
- Data integrity (checksums, version control)
- Complex document structures (PDFs with annotations, notebooks with layers)
- Authentication and security (two-token system, HTTPS, user privacy)

Each package in rmapi tackles one piece of this puzzle. As you read through the package descriptions below, keep in mind that each one exists because there's a genuine complexity that needs to be encapsulated. This isn't over-engineering—it's careful engineering of a complex system.

The packages can be grouped into three categories:
1. **Core protocol packages** (`api/`, `sync15/`, `transport/`): Implement ReMarkable's Sync15 protocol
2. **Supporting packages** (`model/`, `filetree/`, `archive/`): Data structures and utilities
3. **User interface packages** (`shell/`, `config/`, `log/`): CLI and configuration

Let's explore each package in detail, understanding not just what it does, but why it exists and how it fits into the larger picture.

### Core Packages

#### `api/` - High-Level API Interface

This package is the public face of rmapi—the interface that external code (including our remarquee tool) will interact with. It's designed to be stable across protocol changes, providing a consistent API even as ReMarkable's underlying protocols evolve.
- **`api.go`**: Defines `ApiCtx` interface (main API abstraction)
- **`auth.go`**: Authentication flow (device token → user token)
- **`sync15/`**: Sync15 protocol implementation
  - `apictx.go`: Main API context implementation
  - `tree.go`: HashTree (document index) management
  - `blobstorage.go`: Blob storage operations
  - `blobdoc.go`: Document structure representation
  - `entry.go`: Index entry format
  - `common.go`: Hash computation and caching

#### `transport/` - HTTP Client Layer
- **`transport.go`**: HTTP client with authentication, CRC32C checksums, error handling

#### `model/` - Data Models
- **`auth.go`**: `AuthTokens`, `DeviceTokenRequest`
- **`document.go`**: `Document`, `BlobRootStorageRequest/Response`
- **`node.go`**: `Node` (file tree node with parent/children)

#### `filetree/` - File Tree Management
- **`filetree.go`**: `FileTreeCtx` - in-memory file tree with path resolution
- **`treeutil.go`**: Tree traversal utilities

#### `shell/` - CLI Interface
- **`shell.go`**: Main shell setup (`RunShell()`)
- **`put.go`**: Upload command (with `--force`, `--content-only`, `--coverpage` flags)
- **`get.go`**: Download command
- **`mget.go`**: Recursive download
- **`mput.go`**: Recursive upload
- **`mkdir.go`**: Create directory
- **`mv.go`**: Move/rename
- **`rm.go`**: Delete
- **`ls.go`**: List directory
- **`cd.go`**: Change directory
- **`find.go`**: Search with regex
- **`stat.go`**: Show metadata
- **`geta.go`**: Download with annotations
- **`refresh.go`**: Refresh file tree
- **`nuke.go`**: Delete all documents
- **`account.go`**: Show account info
- **`version.go`**: Show version
- **`fs_completer.go`**: File system autocomplete
- **`rmfs_completer.go`**: ReMarkable file system autocomplete
- **`custom_completer.go`**: Shell command autocomplete

#### `archive/` - Document Packaging
- **`doc.go`**: Document archive structure (package-level docs)
- **`blob.go`**: Blob file handling
- **`file.go`**: File entry management
- **`writer.go`**: Archive writer
- **`reader.go`**: Archive reader
- **`zipdoc.go`**: ZIP document operations

#### `config/` - Configuration
- **`config.go`**: Config file path resolution (`ConfigPath()`), token loading/saving
- **`url.go`**: API endpoint URLs (configurable via env vars)

#### `util/` - Utilities
- **`util.go`**: File type detection, path parsing, JSON serialization

#### `log/` - Logging
- **`log.go`**: Structured logging (Trace, Info, Warning, Error)

#### `annotations/` - Annotation Support
- **`pdf.go`**: PDF annotation rendering

#### `encoding/rm/` - Native Format
- **`rm.go`**: `.rm` binary format structures
- **`marshal.go`**: Serialization
- **`unmarshal.go`**: Deserialization

## Key Types and Interfaces

### Understanding rmapi's Type System: Abstraction and Flexibility

The type system in rmapi is carefully designed to balance several competing concerns: flexibility (supporting future protocol versions), usability (clean APIs for developers), performance (efficient operations), and maintainability (clear code organization). Every type serves a purpose beyond just "storing data"—they embody design decisions about how to model ReMarkable's complex document ecosystem.

**The abstraction hierarchy:**

rmapi uses a three-tier type hierarchy that separates different levels of concern:

**Tier 1: Interfaces (`api.ApiCtx`)** - What you can do
- Defines operations without implementation details
- Stable across protocol changes
- Used by external code (including remarquee)
- Easy to mock for testing

**Tier 2: Protocol Types (`sync15.ApiCtx`, `sync15.HashTree`, `sync15.BlobDoc`)** - How it's done
- Implements specific protocol (Sync15)
- Contains sync logic, hash computation, network operations
- Can be swapped out (if Sync16 protocol is released)
- Internal to rmapi

**Tier 3: Data Models (`model.Document`, `model.Node`)** - What's transferred
- Pure data structures (no logic)
- Shared across layers
- Serializable (JSON, wire format)
- Protocol-agnostic

This separation is crucial for remarquee because we'll interact with Tier 1 (interfaces) without depending on Tier 2 or 3 implementations. If ReMarkable changes their protocol, rmapi can add a new implementation (sync16) while keeping the `ApiCtx` interface stable—our code doesn't break.

**Why this matters for remarquee:**

When integrating rmapi into remarquee, we should:
1. **Depend only on `ApiCtx` interface**: Never import sync15 types directly
2. **Use `model.*` types for data exchange**: They're stable and serializable
3. **Avoid depending on internal structures**: They can change

This architectural boundary protects remarquee from rmapi's internal changes while giving us a clean, stable API to work with.

### Core Interfaces

#### `api.ApiCtx` (interface)

This is the most important type in rmapi—it's the contract that defines what rmapi can do. Every operation you perform (upload, download, navigate) goes through this interface. Understanding it deeply is essential for integrating rmapi into remarquee.

**The interface design philosophy:**

The `ApiCtx` interface embodies several design principles:
- **Complete but minimal**: Includes all necessary operations, nothing superfluous
- **Stateful**: Maintains file tree, reducing API calls
- **Notify-aware**: Operations can trigger immediate tablet sync
- **Error-based**: Returns errors (Go idiom), not exceptions

Main API abstraction - all operations go through this:

```go
type ApiCtx interface {
    Filetree() *filetree.FileTreeCtx
    FetchDocument(docId, dstPath string) error
    CreateDir(parentId, name string, notify bool) (*model.Document, error)
    UploadDocument(parentId string, sourceDocPath string, notify bool, coverpage *int) (*model.Document, error)
    ReplaceDocumentFile(docId, sourceDocPath string, notify bool) error
    MoveEntry(src, dstDir *model.Node, name string) (*model.Node, error)
    DeleteEntry(node *model.Node, recursive, notify bool) error
    SyncComplete() error
    Nuke() error
    Refresh() (string, int64, error)
}
```

**Implementation**: `sync15.ApiCtx` (struct)

### Key Types

#### `sync15.HashTree`

The HashTree is the beating heart of rmapi's sync implementation—it's the in-memory representation of your entire document collection on ReMarkable's cloud. Understanding HashTree deeply is essential because it embodies all the clever tricks that make efficient synchronization possible.

**Why "HashTree" and not just "DocumentList"?**

The name is significant. It's not called DocumentTree or DocumentIndex—it's specifically a *Hash*Tree because the hash is the primary organizing principle. Every node in this tree is identified and validated by cryptographic hashes, enabling properties that a simple list couldn't provide:

- **Tamper detection**: Can't change documents without hash changing
- **Efficient comparison**: Compare one hash instead of thousands of files
- **Deduplication**: Automatically handled by hash-based storage
- **Incremental sync**: Only download what changed (determined by hash comparison)

The HashTree type represents years of optimization lessons learned from mobile sync protocols (think Dropbox, iCloud, Google Drive). ReMarkable chose this architecture for good reasons: mobile devices have intermittent connectivity, limited bandwidth, and multiple devices need to coordinate. Hash trees solve all these problems elegantly.

**For remarquee integration:**

When we build remarquee's sync engine, we'll interact with HashTree indirectly through the `ApiCtx` interface, but understanding its structure helps us:
- **Performance tuning**: Know when sync is expensive (full tree rebuild) vs cheap (cached)
- **Debugging**: Understand generation conflicts and how to resolve them
- **Caching strategy**: Leverage rmapi's cache for our own performance
- **Conflict handling**: Design remarquee's conflict resolution UI

Root document index - maintains hash tree of all documents:
- `Hash`: SHA256 hash of root index (changes when ANY document changes)
- `Generation`: Monotonically increasing generation number (for conflict detection, never decreases)
- `SchemaVersion`: Index schema version ("3" for legacy, "4" for current)
- `Docs`: Array of `BlobDoc` entries (every document in your account)
- `CacheVersion`: Cache format version (for compatibility checking)

**Operations:**
- `Mirror()`: Sync with remote storage (downloads changed documents, merges with local)
- `Rehash()`: Recompute root hash after changes (expensive: O(n) where n = number of documents)
- `Add()`: Add new document to tree (updates hash, increments generation)
- `Remove()`: Remove document from tree (updates hash, increments generation)
- `FindDoc()`: Find document by ID (O(n) linear search—consider this for optimization)

#### `sync15.BlobDoc`
Represents a document in the hash tree:
- `Entry`: Document index entry (hash, type, ID, size)
- `Files`: Array of file entries (PDF, metadata, annotations, etc.)
- `Metadata`: Document metadata (name, parent, version, timestamps)

**Operations:**
- `Rehash()`: Recompute document hash
- `AddFile()`: Add file entry
- `Mirror()`: Sync document with remote
- `ToDocument()`: Convert to `model.Document`

#### `sync15.Entry`
Index entry for a single file:
- `Hash`: SHA256 hash
- `Type`: "0" (file) or "80000000" (document)
- `DocumentID`: UUID filename
- `Subfiles`: Number of subfiles (for documents)
- `Size`: File size in bytes

#### `model.Document`
High-level document representation:
- `ID`: UUID
- `Name`: Display name
- `Version`: Version number
- `Type`: "CollectionType" (directory) or "DocumentType" (file)
- `Parent`: Parent directory ID
- `ModifiedClient`: RFC3339 timestamp
- `CurrentPage`: Last opened page

#### `model.Node`
File tree node (in-memory representation):
- `Document`: Associated document
- `Children`: Map of child nodes
- `Parent`: Parent node pointer

#### `transport.HttpClientCtx`
HTTP client with authentication:
- `Client`: `*http.Client`
- `Tokens`: `model.AuthTokens` (device + user tokens)

**Methods:**
- `Get()`, `Post()`, `Put()`, `Delete()`: Standard HTTP methods
- `GetStream()`, `PutStream()`: Stream operations
- `Request()`: Low-level request with auth headers

#### `filetree.FileTreeCtx`
In-memory file tree - represents the entire document hierarchy:
- `root`: Root node (represents "/" directory)
- `idToNode`: Map from UUID → Node (O(1) lookup)
- `pendingParent`: Temporary map for documents whose parents haven't been processed yet

**Why have an in-memory tree?**

The cloud API only exposes documents as a flat list with parent IDs. The file tree:
- Builds a hierarchical structure (directories and files)
- Enables path-based operations (`/Books/document.pdf`)
- Provides fast lookups by ID or path
- Handles ordering (children are sorted)

**Operations:**
- `NodeByPath(path, current)`: Resolve path to single node
- `NodesByPath(path, current, ignoreTrailingSlash)`: Resolve path with globbing (returns multiple)
- `AddDocument(doc)`: Add document to tree (handles parent resolution)
- `DeleteNode(node)`: Remove node from tree
- `MoveNode(src, dst)`: Move node to new parent

**Path resolution examples:**

```
Tree structure:
/
├── Books/
│   ├── novel.pdf
│   └── textbook.pdf
└── Notes/
    └── meeting.pdf

NodeByPath("/Books/novel.pdf", root) → novel.pdf node
NodeByPath("../Notes", booksNode)     → Notes node (relative path)
NodeByPath(".", currentNode)          → currentNode (current directory)
NodesByPath("/Books/*.pdf", root)     → [novel.pdf, textbook.pdf] (glob)
NodesByPath("Books", root)            → [Books node]
NodesByPath("Books/", root)           → [novel.pdf, textbook.pdf] (trailing slash = contents)
```

**Gotcha: Parent resolution**

Documents arrive in arbitrary order (not guaranteed parent-before-child). The file tree handles this:

Pseudocode:
```
FUNCTION AddDocument(doc):
    node = CreateNode(doc)
    idToNode[doc.ID] = node
    
    IF doc.Parent is "":
        // Root-level document
        root.Children[doc.ID] = node
        node.Parent = root
    ELSE IF idToNode[doc.Parent] EXISTS:
        // Parent already processed
        parentNode = idToNode[doc.Parent]
        parentNode.Children[doc.ID] = node
        node.Parent = parentNode
    ELSE:
        // Parent not processed yet, add to pending
        pendingParent[doc.Parent][doc.ID] = node
    
    // Check if any pending children can now be resolved
    IF pendingParent[doc.ID] EXISTS:
        FOR EACH childId IN pendingParent[doc.ID]:
            childNode = idToNode[childId]
            node.Children[childId] = childNode
            childNode.Parent = node
        DELETE pendingParent[doc.ID]
```

This ensures correct tree structure regardless of document processing order.

### Design Patterns in rmapi (Understanding the Architecture)

As you work with rmapi's codebase, you'll notice several recurring design patterns. Understanding these patterns helps you navigate the code and make architectural decisions for the remarquee integration.

**Pattern 1: Interface-Based Abstraction**

The `ApiCtx` interface is the central abstraction. Why use an interface instead of a concrete type?

```go
type ApiCtx interface {
    UploadDocument(...) error
    FetchDocument(...) error
    // ... more methods
}
```

This enables:
- **Multiple implementations**: Currently only `sync15.ApiCtx` exists, but sync16, sync20 could be added
- **Testing**: Can create mock implementations for testing without hitting real API
- **Flexibility**: External code depends on interface, not implementation
- **Stability**: Interface stays stable even if implementation changes dramatically

For remarquee, this means we can import rmapi as a library and depend on the `ApiCtx` interface. If ReMarkable releases a new protocol, rmapi updates internally, but our code doesn't break.

**Pattern 2: Hash-Based Synchronization (Content-Addressable Storage)**

Every file, document, and the root index itself is identified by its SHA256 hash. This is borrowed from Git's architecture and provides powerful guarantees:

- **Immutability**: If hash matches, content is identical (cryptographically guaranteed)
- **Deduplication**: Same content stored once (even across documents)
- **Integrity**: Corruption detected immediately (hash won't match)
- **Efficient sync**: Compare hashes instead of entire files

Understanding this pattern explains many rmapi design decisions. For example, why does uploading create a `BlobDoc` with file hashes instead of just uploading files? Because the hash IS the identity—it's how we know if files changed and how we reference them in the document index.

**Pattern 3: Optimistic Concurrency (Generation Numbers)**

Instead of locking the document tree (which would require server-side coordination and hurt performance), rmapi uses optimistic concurrency:

```
1. Read current state (generation N)
2. Make changes locally
3. Try to write back with generation N
4. If someone else wrote (now generation N+1), retry
```

This pattern is common in distributed systems (etcd, Cassandra, DynamoDB). It trades occasional retries for much better performance and simpler implementation. No locks means no deadlocks, no lock contention, and no need for distributed lock management.

For remarquee, this means our upload/download operations are naturally concurrent-safe. Multiple instances of remarquee tools can run simultaneously without coordination—the generation number mechanism handles conflicts automatically.

**Pattern 4: Lazy Loading with Caching**

The document tree is expensive to build (requires downloading all document indices). rmapi employs lazy loading with aggressive caching:

- First run: Download everything (slow but thorough)
- Subsequent runs: Load from cache (instant startup)
- Incremental updates: Only download changed documents (efficient)

This pattern is crucial for remarquee because we'll be running operations frequently (automated syncs, batch processing). The cache means most operations are fast, with occasional slow operations only when things actually changed in the cloud.

**Pattern 5: Pub/Sub for Events**

The shell layer uses an interactive pattern where commands publish changes and the file tree subscribes. This loose coupling means:
- Commands don't need to know about UI updates
- File tree updates automatically
- Multiple subscribers possible (future: progress bars, logging)

For remarquee, we might want to subscribe to events for progress tracking, logging, or UI updates. The pub/sub pattern makes this easy to add without modifying core logic.

## Authentication Flow

### Understanding the Two-Token System

ReMarkable uses a two-token authentication system similar to OAuth2, but with some unique characteristics. Understanding why this exists helps understand the code.

**Why two tokens?**

The two-token system separates **device identity** from **user sessions**:

1. **Device Token** (long-lived, like a device password):
   - **Purpose**: Identifies your specific device/installation to ReMarkable's servers
   - **Lifetime**: Permanent until explicitly revoked
   - **Analogy**: Like an API key for your installation
   - **Obtained**: Via one-time code from https://my.remarkable.com/device/browser/connect
   - **Storage**: Config file (`~/.rmapi` or `$XDG_CONFIG_HOME/rmapi/rmapi.conf`)
   - **Use**: To obtain user tokens (acts as a credential to get session tokens)

2. **User Token** (short-lived, refreshed as needed):
   - **Purpose**: Authorizes API operations for a specific user session
   - **Lifetime**: Expires after some time (hours to days)
   - **Analogy**: Like a session cookie
   - **Obtained**: Using device token (exchange device token for user token)
   - **Format**: JWT (JSON Web Token) containing user email, sync version, scopes
   - **Use**: Included in HTTP `Authorization` header for all API operations

**Why not just one token?**

- **Security**: User token expires, limiting damage if stolen
- **Multi-device**: Same device token can generate multiple user tokens
- **Revocation**: Can revoke device token without re-authenticating every session

### Authentication Code Flow (Step-by-Step)

**Step 1: Get or load device token**

Pseudocode:
```
FUNCTION AuthHttpCtx(reAuth, nonInteractive):
    configPath = FindConfigFile()  // ~/.rmapi or XDG location
    authTokens = LoadTokensFromConfig(configPath)
    
    IF authTokens.DeviceToken is EMPTY:
        IF nonInteractive:
            FAIL("No device token, can't prompt for code")
        
        // First-time setup: user must register this device
        PRINT("Go to https://my.remarkable.com/device/browser/connect")
        code = PromptUserFor8DigitCode()
        
        // Exchange one-time code for device token
        deviceToken = CallAPI(
            url: "https://webapp-prod.cloud.remarkable.engineering/token/json/2/device/new",
            method: POST,
            auth: Bearer (empty),
            body: {
                code: code,
                deviceDesc: "desktop-linux",
                deviceID: GenerateUUID()
            }
        )
        
        authTokens.DeviceToken = deviceToken
        SaveTokensToConfig(configPath, authTokens)
```

**What's happening here?**

- On first run, rmapi has no device token
- User visits ReMarkable's website to generate a one-time 8-digit code
- This code is valid for a short time (minutes)
- rmapi exchanges this code for a permanent device token
- Device token is saved locally for future use

**Step 2: Get or refresh user token**

Pseudocode:
```
    IF authTokens.UserToken is EMPTY OR reAuth:
        // Exchange device token for user token
        userToken = CallAPI(
            url: "https://webapp-prod.cloud.remarkable.engineering/token/json/2/user/new",
            method: POST,
            auth: Bearer <deviceToken>,
            body: null
        )
        
        IF response is 401 Unauthorized:
            // Device token is invalid/revoked
            authTokens.DeviceToken = ""
            RETRY  // Will prompt for new code
        
        authTokens.UserToken = userToken
        SaveTokensToConfig(configPath, authTokens)
    
    RETURN HttpClientCtx(authTokens)
```

**What's happening here?**

- Device token is used to get a fresh user token
- User token may expire, so we check if it's empty or if re-auth is requested
- If device token is invalid (401), we clear it and start over
- User token is saved for subsequent runs (until it expires)

**Step 3: Parse user token and initialize API**

Pseudocode:
```
FUNCTION main():
    FOR attempt = 1 TO 3:  // Retry up to 3 times
        authCtx = AuthHttpCtx(attempt > 1, nonInteractive)
        
        // Parse JWT to extract user info
        userInfo = ParseJWT(authCtx.Tokens.UserToken)
        // JWT contains: email, sync version ("1.5"), scopes
        
        // Create API context based on sync version
        apiCtx = CreateApiCtx(authCtx, userInfo.SyncVersion)
        // Downloads document tree from cloud
        
        IF success:
            BREAK
        
        // Token might have expired, try again
    
    IF failed after 3 attempts:
        FATAL("Authentication failed")
    
    // Now ready to run shell or commands
    RunShell(apiCtx, userInfo, commandLineArgs)
```

**What's happening here?**

- We parse the JWT user token to extract user info (email, sync version)
- Sync version determines which protocol implementation to use (currently only v1.5 exists)
- `CreateApiCtx` downloads the document tree from cloud (expensive operation)
- If this fails (token expired, network issue), retry up to 3 times
- Once successful, shell is ready to accept commands

### Authentication Code Flow

```go
// 1. Load or create device token
authTokens := config.LoadTokens(configPath)
if authTokens.DeviceToken == "" {
    deviceToken := newDeviceToken(httpCtx, readCode()) // User enters 8-digit code
    authTokens.DeviceToken = deviceToken
    config.SaveTokens(configPath, authTokens)
}

// 2. Obtain user token
if authTokens.UserToken == "" || reAuth {
    userToken := newUserToken(httpCtx) // Uses device token
    authTokens.UserToken = userToken
    config.SaveTokens(configPath, authTokens)
}

// 3. Parse user token to get sync version
userInfo := api.ParseToken(userToken) // Extracts email, sync version from JWT
```

### Token Storage

Config file format (YAML):
```yaml
devicetoken: "device-token-string"
usertoken: "user-token-jwt-string"
```

**Config path resolution** (`config.ConfigPath()`):
1. `RMAPI_CONFIG` environment variable
2. `~/.rmapi` (if exists)
3. `$XDG_CONFIG_HOME/rmapi/rmapi.conf` (default)

## Sync15 Protocol Details

### Understanding the Hash Tree Concept

The Sync15 protocol uses a **hash tree** (also called a Merkle tree) to efficiently synchronize documents. This is the same concept used in Git, blockchain, and many distributed systems.

**Why hash trees?**

Traditional sync approaches download everything and compare files. This is slow and bandwidth-intensive. Hash trees enable:
- **Fast detection**: Compare a single hash to know if anything changed
- **Minimal transfer**: Only download what actually changed
- **Integrity verification**: SHA256 hashes guarantee data hasn't been corrupted
- **Conflict detection**: Generation numbers prevent concurrent modification issues

**How it works:**

Imagine a tree where each node contains a hash of its children. If any child changes, the parent's hash changes, propagating up to the root. By comparing just the root hash, you know if anything anywhere in the tree changed.

### Document Structure

Each document is represented as a **hash tree**:

```
Root Index (root.docSchema)
├── Hash: SHA256(all document hashes concatenated)
├── Generation: Monotonic counter (1, 2, 3, ...)
└── Documents:
    ├── Document 1 (UUID.docSchema)
    │   ├── Hash: SHA256(all file hashes concatenated)
    │   ├── Files:
    │   │   ├── UUID.pdf (PDF content) [hash: abc123...]
    │   │   ├── UUID.metadata (JSON metadata) [hash: def456...]
    │   │   ├── UUID.content (content descriptor) [hash: ghi789...]
    │   │   └── UUID.pagedata (page metadata) [hash: jkl012...]
    │   └── Metadata: {name, parent, version, timestamps}
    └── Document 2 (UUID.docSchema)
        └── (same structure)
```

**Key insight**: Each document's hash is computed from its files' hashes. The root hash is computed from all document hashes. Change one file → document hash changes → root hash changes.

### Index Format

The root index and each document index use a simple line-based format:

**Schema V3** (legacy, used pre-2023):
```
3                                    ← Schema version
<hash>:<type>:<docId>:<subfiles>:<size>
<hash>:<type>:<docId>:<subfiles>:<size>
...
```

**Schema V4** (current, adds summary line):
```
4                                    ← Schema version
0:.<entries_count>:<total_size>     ← Summary line
<hash>:<type>:<docId>:<subfiles>:<size>
<hash>:<type>:<docId>:<subfiles>:<size>
...
```

**Field meanings:**
- `hash`: SHA256 hash (hex string)
- `type`: "0" for file, "80000000" for document (in V3)
- `docId`: UUID with extension (e.g., `abc123.pdf` or `abc123.docSchema`)
- `subfiles`: Number of files in document
- `size`: Total size in bytes

**Why the summary line in V4?**

Schema V4 adds a summary line with entry count and total size. This enables:
- Quick size calculation without parsing entire index
- Validation (ensure entry count matches actual entries)
- Future extensibility (reserved fields: 0, .)

### Synchronization Algorithm (The Heart of Sync15)

Understanding how sync works is crucial for using rmapi effectively or debugging issues.

**Initial sync (first run):**

Pseudocode:
```
FUNCTION CreateApiCtx():
    // 1. Try to load cached document tree from disk
    cachedTree = LoadFromCache("~/.cache/rmapi/tree.cache")
    
    IF cachedTree exists AND cachedTree.CacheVersion == 3:
        localHash = cachedTree.Hash
        localGeneration = cachedTree.Generation
    ELSE:
        localHash = ""
        localGeneration = 0
    
    // 2. Get current remote root hash and generation
    remoteHash, remoteGeneration = API_GET("/sync/v4/root")
    
    // 3. Compare hashes
    IF remoteHash == localHash:
        // Nothing changed, use cached tree
        RETURN cachedTree
    
    // 4. Something changed, download root index
    rootIndexData = API_GET("/sync/v3/files/" + remoteHash)
    documentEntries = ParseIndex(rootIndexData)
    // documentEntries = [{hash, type, docId, subfiles, size}, ...]
    
    // 5. For each document, check if it changed
    FOR EACH entry IN documentEntries:
        cachedDoc = cachedTree.FindDoc(entry.docId)
        
        IF cachedDoc EXISTS AND cachedDoc.Hash == entry.Hash:
            // Document unchanged, keep cached version
            newTree.Add(cachedDoc)
        ELSE:
            // Document changed or new, download it
            docIndexData = API_GET("/sync/v3/files/" + entry.Hash)
            fileEntries = ParseIndex(docIndexData)
            
            // Download metadata to get document name
            FOR EACH fileEntry IN fileEntries:
                IF fileEntry.docId ends with ".metadata":
                    metadataData = API_GET("/sync/v3/files/" + fileEntry.Hash)
                    metadata = ParseJSON(metadataData)
                    // metadata = {docName, parent, version, lastModified, ...}
            
            blobDoc = CreateBlobDoc(entry, fileEntries, metadata)
            newTree.Add(blobDoc)
    
    // 6. Cache the tree for next run
    SaveToCache(newTree, "~/.cache/rmapi/tree.cache")
    
    RETURN newTree
```

**What's happening:**

- On first run, cache doesn't exist → downloads everything
- On subsequent runs, compares root hash to detect changes
- Only downloads changed documents (efficient!)
- Uses concurrent requests (default: 20 parallel) for speed
- Metadata files contain human-readable document names

**Refresh sync (when you run `refresh` command):**

Same algorithm as initial sync, but always downloads changed documents.

**Update operations (upload, move, delete):**

Pseudocode:
```
FUNCTION UploadDocument(parentId, sourceFile, notify):
    // 1. Package document into .rmdoc format
    docId = GenerateUUID()
    tmpDir = CreateTempDir()
    documentFiles = PackageDocument(sourceFile, parentId, tmpDir)
    // documentFiles = {UUID.pdf, UUID.metadata, UUID.content, UUID.pagedata}
    
    // 2. Upload each file as a blob
    blobDoc = CreateBlobDoc(docId, parentId)
    FOR EACH file IN documentFiles:
        fileHash = ComputeSHA256(file.data)
        API_PUT("/sync/v3/files/" + fileHash, file.data, headers={
            "rm-filename": file.name,
            "x-goog-hash": "crc32c=" + ComputeCRC32C(file.data)
        })
        blobDoc.AddFile(fileHash, file.name, file.size)
    
    // 3. Recompute document hash (from all file hashes)
    blobDoc.Rehash()  // SHA256 of all file hashes
    
    // 4. Upload document index
    docIndexData = blobDoc.SerializeIndex()
    API_PUT("/sync/v3/files/" + blobDoc.Hash, docIndexData, headers={
        "rm-filename": docId + ".docSchema"
    })
    
    // 5. Update root index (with conflict handling)
    CALL Sync(operation=AddDocumentToTree, notify)
```

**What's happening:**

- Document is packaged into multiple files (PDF, metadata, etc.)
- Each file is uploaded as a "blob" identified by its SHA256 hash
- Document's hash is computed from its files' hashes
- Document index is uploaded
- Root index must be updated (see next section)

**Step 5: Sync function (handles concurrent modifications):**

Pseudocode:
```
FUNCTION Sync(operation, notify):
    FOR attempt = 1 TO 10:  // Max 10 retries
        // 1. Apply operation to local tree
        operation(localTree)  // e.g., AddDocument, RemoveDocument
        
        // 2. Recompute root hash
        localTree.Rehash()  // SHA256 of all document hashes
        
        // 3. Upload root index blob
        rootIndexData = localTree.SerializeIndex()
        API_PUT("/sync/v3/files/" + localTree.Hash, rootIndexData, headers={
            "rm-filename": "root.docSchema"
        })
        
        // 4. Update root pointer (critical section!)
        response = API_PUT("/sync/v3/root", body={
            hash: localTree.Hash,
            generation: localTree.Generation,  // Current generation
            broadcast: notify  // Trigger tablet sync
        })
        
        IF response.status == 200 OK:
            // Success! Update local generation
            localTree.Generation = response.generation  // Server returns new generation
            SaveToCache(localTree)
            RETURN success
        
        IF response.status == 412 Precondition Failed:
            // Someone else updated root before us (conflict!)
            PRINT("Wrong generation, re-syncing...")
            
            // Re-download remote tree
            localTree.Mirror(remoteStorage)
            // This merges remote changes with our changes
            
            CONTINUE  // Retry operation with updated tree
        
        IF response.status != 412:
            // Some other error (network, auth, etc.)
            RETURN error
    
    // Failed after 10 retries
    RETURN error("Too many conflicts")
```

**What's happening here (the tricky part!):**

This is **optimistic concurrency control**—a common pattern in distributed systems.

**The Problem:**
- You download tree at generation N
- You make local changes
- Meanwhile, another device uploads something (server is now at generation N+1)
- You try to upload with generation N
- Server rejects (412 error) because generation is wrong

**The Solution:**
- Server only accepts updates with correct generation number
- If wrong, client re-downloads tree (merges changes)
- Retries update with new generation
- Maximum 10 attempts (usually succeeds on retry 1-2)

**Why this works:**
- Prevents lost updates (two devices don't overwrite each other)
- Eventually consistent (retries until success)
- No locking required (optimistic approach)

### Synchronization Algorithm

### Blob Storage (Content-Addressable Storage)

ReMarkable uses **content-addressable storage** for all files. This is a key architectural decision that enables efficient sync.

**What is content-addressable storage?**

Instead of storing files by name (`document.pdf`), files are stored by their **hash** (their content's SHA256). The hash IS the address.

**Why is this powerful?**

1. **Deduplication**: Same content = same hash → stored once
2. **Integrity**: Can't corrupt data without changing hash
3. **Caching**: If you have hash `abc123`, you have the exact file (immutable)
4. **Efficient sync**: Compare hashes instead of file contents

**Example:**

```
Upload "document.pdf" (1 MB):
1. Compute hash: SHA256("document.pdf content") = "abc123..."
2. Upload to: /sync/v3/files/abc123...
3. Store mapping: document.pdf → abc123...

Later, download "document.pdf":
1. Look up mapping: document.pdf → abc123...
2. Download from: /sync/v3/files/abc123...
3. Verify: SHA256(downloaded content) == abc123...
```

### Upload Flow (Detailed)

Pseudocode for uploading a PDF:
```
FUNCTION UploadDocument(parentId, sourceFile):
    // Step 1: Package document
    docId = GenerateUUID()  // e.g., "f7e4a2c1-..."
    docName = ExtractFilename(sourceFile)  // e.g., "report.pdf"
    
    // Create .rmdoc package with multiple files:
    packagedFiles = [
        {name: docId + ".pdf", data: ReadFile(sourceFile)},
        {name: docId + ".metadata", data: CreateMetadataJSON(docName, parentId)},
        {name: docId + ".content", data: CreateContentJSON(docId)},
        {name: docId + ".pagedata", data: CreatePageDataJSON()}
    ]
    
    // Step 2: Upload each file as a blob
    blobDoc = NewBlobDoc(docName, docId, parentId)
    
    FOR EACH file IN packagedFiles:
        // Compute content hash (this IS the storage address)
        fileHash = SHA256(file.data)  // e.g., "a1b2c3..."
        
        // Compute CRC32C checksum (for data integrity verification)
        crc32c = CRC32C(file.data)
        
        // Upload blob (PUT with hash as URL path)
        API_PUT(
            url: "/sync/v3/files/" + fileHash,
            body: file.data,
            headers: {
                "rm-filename": file.name,
                "x-goog-hash": "crc32c=" + Base64(crc32c),
                "content-type": "application/octet-stream"
            }
        )
        
        // Add to document's file list
        blobDoc.AddFile({
            Hash: fileHash,
            DocumentID: file.name,
            Size: len(file.data)
        })
    
    // Step 3: Compute document hash
    blobDoc.Rehash()
    // documentHash = SHA256(concat(file1.hash, file2.hash, ...))
    
    // Step 4: Upload document index
    docIndexData = SerializeDocumentIndex(blobDoc.Files)
    API_PUT(
        url: "/sync/v3/files/" + blobDoc.Hash,
        body: docIndexData,
        headers: {"rm-filename": docId + ".docSchema"}
    )
    
    // Step 5: Update root index
    Sync(operation: ADD blobDoc to tree, notify: true)
    // This handles concurrent modification (see Sync pseudocode above)
    
    RETURN blobDoc
```

**Important notes:**

- Blobs are immutable (hash never changes)
- If you update a document, you upload NEW blobs (old ones stay)
- ReMarkable's cloud handles garbage collection
- CRC32C checksum catches transmission errors
- Each document has 4+ files (more for annotations)

### Download Flow (Detailed)

Pseudocode for downloading a PDF:
```
FUNCTION FetchDocument(docId, destinationPath):
    // Step 1: Find document in local tree
    blobDoc = localTree.FindDoc(docId)
    // blobDoc contains document's file list with hashes
    
    // Step 2: Create temporary ZIP file
    zipFile = CreateTempZip()
    
    // Step 3: Download each file by hash
    FOR EACH fileEntry IN blobDoc.Files:
        fileData = API_GET(
            url: "/sync/v3/files/" + fileEntry.Hash,
            headers: {"rm-filename": fileEntry.DocumentID}
        )
        
        // Add to ZIP
        zipFile.AddFile(fileEntry.DocumentID, fileData)
    
    // Step 4: Save ZIP as .rmdoc
    CopyFile(zipFile.path, destinationPath)
    
    RETURN success
```

**What you get:**

A `.rmdoc` file is just a ZIP archive containing:
- `UUID.pdf`: The actual PDF content
- `UUID.metadata`: Document metadata (JSON)
- `UUID.content`: Content descriptor (JSON)
- `UUID.pagedata`: Page metadata (JSON)
- `UUID/*.rm`: Annotation files (if any)

**To extract:**
```bash
unzip document.rmdoc
# You'll see: UUID.pdf, UUID.metadata, UUID/, etc.
```

### Conflict Resolution (Why Generation Numbers Matter)

**The scenario:**

```
Time  Device A           Server          Device B
----  ----------------   ---------       ----------------
T0    Downloads tree     Gen=5           Downloads tree
      (gen=5)                            (gen=5)

T1    Makes change       Gen=5           Makes change
      (local only)                       (local only)

T2                       Gen=6 ←─────── Uploads (success)
                                        Gen 5→6

T3    Uploads            Gen=6
      (gen=5)            ↓
                         REJECT!
                         412 Error
```

**What happens to Device A:**

Pseudocode:
```
// Device A tries to upload with gen=5
response = API_PUT("/sync/v3/root", {hash: newHash, generation: 5})

IF response == 412 Precondition Failed:
    PRINT("Conflict detected! Re-syncing...")
    
    // Re-download tree (now at gen=6)
    remoteTree = DownloadTree()  // Gets Device B's changes
    
    // Merge: Apply Device A's changes to updated tree
    mergedTree = ApplyChanges(remoteTree, Device_A_changes)
    
    // Retry upload with new generation
    response = API_PUT("/sync/v3/root", {hash: mergedHash, generation: 6})
    // Now succeeds with gen 6→7
```

**Why this is safe:**

- No updates are lost (Device A's changes are preserved)
- Server guarantees sequential generation numbers
- Worst case: 10 retries before giving up
- In practice: Usually succeeds on first retry

**Real-world analogy:**

Like Git merge conflicts:
- Two branches diverge (Device A and B)
- One branch merges first (Device B)
- Other branch must rebase (Device A)
- Then can merge its changes

## Shell Command Mapping

### The Philosophy Behind the Shell Interface

The shell layer in rmapi is more than just a command-line wrapper around the API—it's a carefully designed user interface that borrows decades of Unix wisdom. Understanding why specific commands exist and how they map to API operations reveals insights about both ReMarkable's architecture and good CLI design principles.

**Why a shell interface at all?**

rmapi could have been designed as a simple command-line tool where each operation is a separate command:
```bash
$ rmapi-upload file.pdf
$ rmapi-download file.pdf
$ rmapi-list /Books
```

Instead, it provides an interactive shell:
```bash
$ rmapi
[/]> ls
[/]> cd Books
[/Books]> put document.pdf
```

This design choice reflects the reality of document management workflows. Users rarely perform single isolated operations—they explore, navigate, batch operations together. An interactive shell provides:

1. **State preservation**: Current directory, cached file tree, established authentication
2. **Autocomplete**: Press TAB to see files, directories, commands
3. **Efficiency**: No authentication overhead per command (one login, many operations)
4. **Discoverability**: Type `help` to see available commands
5. **Familiar**: Users who know Bash/FTP/SFTP feel immediately comfortable

For the remarquee project, this suggests our unified CLI should also provide an interactive mode, not just one-shot commands. Users benefit from context-aware operations and stateful sessions.

**Unix command philosophy:**

rmapi's shell commands follow Unix principles:
- **Do one thing well**: Each command has a clear, focused purpose
- **Composable**: Commands can be scripted together
- **Predictable**: Similar to standard Unix tools (ls, cd, mv, rm)
- **Terse but clear**: Short names, mnemonic (put/get, ls/cd)

This isn't accidental—it's intentional design that makes the tool feel "native" to terminal users. When designing remarquee's interface, we should maintain this Unix-friendly approach while adding modern conveniences (progress bars, better error messages, interactive prompts).

**The mapping strategy:**

Each shell command maps to one or more API operations. The shell layer adds value by:
- **Path resolution**: Users think in paths (`/Books/doc.pdf`), API thinks in UUIDs
- **Error handling**: Converts Go errors to human-readable messages
- **Validation**: Checks arguments before hitting API (fail fast)
- **Feedback**: Progress indicators, success messages, warnings

Understanding this mapping helps you use rmapi effectively and guides how remarquee should expose rmapi's functionality.

### File Operations

| Command | API Method | Description |
|---------|-----------|-------------|
| `put <file>` | `UploadDocument()` | Upload document |
| `put --force <file>` | `DeleteEntry()` + `UploadDocument()` | Replace document |
| `put --content-only <file>` | `ReplaceDocumentFile()` | Replace PDF content only |
| `get <file>` | `FetchDocument()` | Download document |
| `mget <dir>` | `FetchDocument()` (recursive) | Download directory |
| `mput <dir>` | `UploadDocument()` (recursive) | Upload directory |

### Directory Operations

| Command | API Method | Description |
|---------|-----------|-------------|
| `mkdir <dir>` | `CreateDir()` | Create directory |
| `rm <entry>` | `DeleteEntry()` | Delete file/directory |
| `mv <src> <dst>` | `MoveEntry()` | Move/rename entry |

### Navigation

| Command | API Method | Description |
|---------|-----------|-------------|
| `ls` | `Filetree().NodeByPath()` | List directory |
| `cd <dir>` | `Filetree().NodeByPath()` | Change directory |
| `pwd` | `Filetree().NodeToPath()` | Print current path |
| `find <pattern>` | `Filetree().NodesByPath()` | Search with regex |

### Metadata

| Command | API Method | Description |
|---------|-----------|-------------|
| `stat <entry>` | `Filetree().NodeByPath()` | Show metadata |
| `account` | `UserInfo` | Show account info |
| `version` | `version.Version` | Show version |

### Sync Operations

| Command | API Method | Description |
|---------|-----------|-------------|
| `refresh` | `Refresh()` | Re-sync file tree |
| `nuke` | `Nuke()` | Delete all documents |

## Developer Usage Guide

### Getting Started (For New Developers)

If you're new to rmapi and want to use it as a library, here's what you need to know.

**Prerequisites:**
- Go installed (1.23+)
- ReMarkable account (for authentication)
- Basic understanding of Go interfaces and error handling

**Import the package:**
```
go get github.com/juruen/rmapi
```

### Basic Usage (Complete Example with Explanation)

This example shows the complete flow from authentication to document operations.

**Step 1: Initialize and authenticate**

```go
import (
    "fmt"
    "log"
    "github.com/juruen/rmapi/api"
)

// Authenticate with ReMarkable cloud
authCtx := api.AuthHttpCtx(
    false,  // reAuth: don't force re-authentication
    false,  // nonInteractive: allow prompting for code if needed
)

// On first run, this will:
// 1. Prompt: "Go to https://my.remarkable.com/device/browser/connect"
// 2. Prompt: "Enter one-time code: "
// 3. Exchange code for device token
// 4. Exchange device token for user token
// 5. Save tokens to ~/.rmapi

// On subsequent runs:
// 1. Load tokens from ~/.rmapi
// 2. Reuse device token
// 3. Refresh user token if needed
```

**Why `authCtx` is important:**

The `authCtx` contains:
- HTTP client (with timeout: 5 minutes)
- Authentication tokens (device + user)
- Will be used for all API calls

**Step 2: Parse user token and get sync version**

```go
// Parse JWT to extract user info
userInfo, err := api.ParseToken(authCtx.Tokens.UserToken)
if err != nil {
    log.Fatal("Failed to parse token:", err)
}

fmt.Printf("Logged in as: %s\n", userInfo.User)  // Email address
fmt.Printf("Sync version: %s\n", userInfo.SyncVersion)  // "1.5"
```

**Why parse the token?**

- JWT contains user email and sync version
- Sync version determines which protocol implementation to use
- Currently only v1.5 exists, but future-proofing

**Step 3: Create API context (expensive operation!)**

```go
// This downloads document tree from cloud (may take 5-30 seconds)
ctx, err := api.CreateApiCtx(authCtx, userInfo.SyncVersion)
if err != nil {
    log.Fatal("Failed to create API context:", err)
}

// What happened:
// 1. Downloaded root index from cloud
// 2. Downloaded all document indices
// 3. Downloaded all document metadata
// 4. Built in-memory file tree
// 5. Cached tree to disk
```

**Performance note:**

`CreateApiCtx` is slow because it:
- Makes 1 API call to get root hash/generation
- Makes 1 API call to download root index
- Makes N API calls to download document indices (N = number of documents)
- Makes M API calls to download metadata (M = number of documents)

For 100 documents, this is ~200 API calls! But it's cached, so subsequent runs are fast.

**Step 4: Use the API (common operations)**

```go
// Get file tree
filetree := ctx.Filetree()
root := filetree.Root()  // Root node (represents "/" directory)

// List root directory (like "ls /")
fmt.Println("Documents in root:")
for _, child := range root.Nodes() {
    if child.IsDirectory() {
        fmt.Printf("[DIR]  %s\n", child.Name())
    } else {
        fmt.Printf("[FILE] %s\n", child.Name())
    }
}

// Upload document (like "put file.pdf /Books")
doc, err := ctx.UploadDocument(
    "",                    // parentId: "" = root
    "/path/to/file.pdf",  // local file path
    true,                  // notify: trigger tablet sync
    nil,                   // coverpage: nil = no coverpage
)
if err != nil {
    log.Fatal("Upload failed:", err)
}
fmt.Printf("Uploaded: %s (ID: %s)\n", doc.Name, doc.ID)

// Download document (like "get file.pdf")
err = ctx.FetchDocument(doc.ID, "/tmp/downloaded.rmdoc")
if err != nil {
    log.Fatal("Download failed:", err)
}
fmt.Println("Downloaded to /tmp/downloaded.rmdoc")

// Create directory (like "mkdir /Books")
dir, err := ctx.CreateDir(
    "",          // parentId: "" = create in root
    "MyBooks",   // directory name
    true,        // notify: trigger tablet sync
)
if err != nil {
    log.Fatal("Create directory failed:", err)
}
fmt.Printf("Created directory: %s (ID: %s)\n", dir.Name, dir.ID)

// Navigate file tree (like "cd Books")
booksNode, err := filetree.NodeByPath("/MyBooks", nil)
if err != nil {
    log.Fatal("Directory not found:", err)
}

// Upload to specific directory
doc2, err := ctx.UploadDocument(
    booksNode.Id(),       // parentId: Books folder UUID
    "/path/to/book.pdf",
    true,
    nil,
)

// Move document (like "mv document.pdf /Books/")
srcNode, _ := filetree.NodeByPath("/document.pdf", nil)
dstNode, _ := filetree.NodeByPath("/MyBooks", nil)
movedNode, err := ctx.MoveEntry(srcNode, dstNode, "document.pdf")
if err != nil {
    log.Fatal("Move failed:", err)
}

// Delete document (like "rm document.pdf")
node, _ := filetree.NodeByPath("/MyBooks/document.pdf", nil)
err = ctx.DeleteEntry(
    node,
    false,  // recursive: false = fail if directory not empty
    true,   // notify: trigger tablet sync
)
```

**Understanding notify parameter:**

Most API functions have a `notify bool` parameter. This controls whether to trigger tablet sync:
- `true`: Tablet syncs within ~30 seconds (makes API call to trigger)
- `false`: Tablet syncs on next scheduled sync (~5 minutes)

**Use `false` when:**
- Doing multiple operations (batch upload)
- Don't need immediate sync
- Want to minimize API calls

**Use `true` when:**
- Single operation
- Need immediate sync to tablet
- End of batch operations

**Example batch upload:**
```go
// Upload 10 files without triggering sync each time
for i, file := range files {
    notify := (i == len(files)-1)  // Only notify on last file
    ctx.UploadDocument("", file, notify, nil)
}
```

### Non-Interactive Mode

Set `nonInteractive=true` to avoid prompting for one-time code:

```go
authCtx := api.AuthHttpCtx(false, true) // nonInteractive=true
// Will fail if device token missing
```

### Environment Variables

- `RMAPI_CONFIG`: Override config file path
- `RMAPI_TRACE=1`: Enable trace logging
- `RMAPI_USE_HIDDEN_FILES=1`: Include hidden files/directories
- `RMAPI_THUMBNAILS`: Generate thumbnails
- `RMAPI_AUTH`: Override auth host URL
- `RMAPI_DOC`: Override document storage URL
- `RMAPI_HOST`: Override all URLs
- `RMAPI_CONCURRENT`: Max concurrent HTTP requests (default: 20)
- `RMAPI_FORCE_SCHEMA_VERSION`: Force schema version ("3" or "4")

### Error Handling (Common Pitfalls and Solutions)

#### Error: `transport.ErrUnauthorized` (401)

**What it means:**
- User token expired or invalid
- Device token revoked

**How to handle:**
```go
authCtx := api.AuthHttpCtx(false, false)
userInfo, err := api.ParseToken(authCtx.Tokens.UserToken)
if err != nil {
    // Token parse failed (expired)
    log.Println("Token expired, re-authenticating...")
    authCtx = api.AuthHttpCtx(true, false)  // reAuth=true
    userInfo, err = api.ParseToken(authCtx.Tokens.UserToken)
}
```

**Prevention:**
- Catch `ErrUnauthorized` and retry with `reAuth=true`
- Don't store user tokens long-term (they expire)
- Device token should work indefinitely

#### Error: `transport.ErrWrongGeneration` (412)

**What it means:**
- Another device/client updated the document tree
- Your local generation number is stale

**How to handle:**

This is automatically handled by the `Sync()` function (retries up to 10 times). But if you're implementing custom sync logic:

```go
for retries := 0; retries < 10; retries++ {
    err := ctx.UploadDocument(parentId, file, true, nil)
    
    if err == transport.ErrWrongGeneration {
        log.Println("Conflict detected, refreshing tree...")
        ctx.Refresh()  // Re-download tree from cloud
        continue  // Retry operation
    }
    
    if err != nil {
        return err  // Other error, don't retry
    }
    
    break  // Success
}
```

**When this happens:**
- Multiple devices uploading simultaneously
- Long-running operations (tree changed while you were working)
- Network delays

#### Error: `transport.ErrNotFound` (404)

**What it means:**
- Document/blob doesn't exist in cloud
- Blob has been garbage collected

**Common causes:**
- Using stale document ID
- Cache is out of sync
- Document was deleted on another device

**How to handle:**
```go
err := ctx.FetchDocument(docId, dstPath)
if err == transport.ErrNotFound {
    log.Println("Document not found, refreshing tree...")
    ctx.Refresh()  // Re-sync to get latest tree
    // Document might have been deleted
}
```

#### Common Mistake: Forgetting to Refresh

**Problem:**
```go
ctx, _ := api.CreateApiCtx(authCtx, userInfo.SyncVersion)
// Downloads tree (generation=5)

// ... time passes, other devices make changes ...

ctx.UploadDocument(...)  // Tries to use generation=5
// FAIL: Generation is now 8!
```

**Solution:**
```go
// Before important operations, refresh if tree is stale
if timeSinceLastRefresh > 5*time.Minute {
    ctx.Refresh()  // Re-sync with cloud
}
```

#### Common Mistake: Not Handling Notify Parameter

**Problem:**
```go
// Upload 100 files with notify=true each time
for _, file := range files {
    ctx.UploadDocument("", file, true, nil)  // Triggers sync 100 times!
}
```

**Solution:**
```go
// Only notify on last file
for i, file := range files {
    notify := (i == len(files)-1)
    ctx.UploadDocument("", file, notify, nil)
}
```

### Caching Strategy (Performance Optimization)

HashTree is cached to `$XDG_CACHE_HOME/rmapi/tree.cache` (usually `~/.cache/rmapi/tree.cache`).

**Why cache?**

Initial sync is expensive:
- Downloads root index
- Downloads all document indices
- Downloads all metadata files
- Can take 30+ seconds for 100 documents

Cache enables:
- Instant startup (load from disk)
- Only download changed documents
- Offline operation (read-only)

**Cache invalidation:**

Cache is invalidated when:
- Cache version changes (code update)
- Root hash changes (detected on startup)
- Cache file corrupted

**Cache format:**

JSON file containing:
```json
{
  "Hash": "root-hash-here",
  "Generation": 42,
  "SchemaVersion": "4",
  "CacheVersion": 3,
  "Docs": [
    {
      "Hash": "doc-hash-here",
      "DocumentID": "uuid-here",
      "Files": [...],
      "Metadata": {...}
    },
    ...
  ]
}
```

**Manual cache management:**

```bash
# Clear cache (force full re-sync)
rm ~/.cache/rmapi/tree.cache

# Inspect cache
cat ~/.cache/rmapi/tree.cache | jq '.Generation'

# Check cache size
du -h ~/.cache/rmapi/tree.cache
```

## File Organization Summary

### Entry Points
- `main.go`: CLI entry point, authentication, shell invocation

### Core API
- `api/api.go`: `ApiCtx` interface definition
- `api/auth.go`: Authentication flow
- `api/sync15/apictx.go`: Main implementation
- `api/sync15/tree.go`: HashTree operations
- `api/sync15/blobstorage.go`: Blob storage
- `api/sync15/blobdoc.go`: Document structure

### Transport
- `transport/transport.go`: HTTP client with auth

### Models
- `model/auth.go`: Token structures
- `model/document.go`: Document structures
- `model/node.go`: File tree node

### File Tree
- `filetree/filetree.go`: In-memory tree management

### Shell
- `shell/shell.go`: Shell setup
- `shell/*.go`: Individual command implementations

### Archive
- `archive/doc.go`: Document packaging (package docs)
- `archive/*.go`: Archive read/write operations

### Configuration
- `config/config.go`: Config file management
- `config/url.go`: API endpoint URLs

## Quick Reference


This quick reference section consolidates the most commonly used functions and types you'll need when working with rmapi. Think of it as a cheat sheet you can refer to while coding, without having to scroll through the detailed explanations above.

**How to use this section:**

When integrating rmapi into remarquee, you'll repeatedly need to:
1. **Authenticate and initialize**: Start every session (cache makes this fast after first run)
2. **Navigate the file tree**: Find documents by path or UUID
3. **Perform operations**: Upload, download, move, delete
4. **Handle errors**: Recover from common failures

The functions and types listed below are the ones you'll use 80% of the time. The remaining 20% (edge cases, advanced features) are covered in the detailed sections above.

**Integration strategy for remarquee:**

When building remarquee's rmapi integration layer, start with these core functions:
- Create a remarquee session that wraps `ApiCtx`
- Implement basic operations (sync, upload, download) using these functions
- Add error handling for the common errors listed
- Extend with advanced features as needed

The beauty of rmapi's interface design is that you can build a powerful integration using just this small set of functions—the complexity is hidden behind clean abstractions.

### Key Functions

**Authentication:**
- `api.AuthHttpCtx(reAuth, nonInteractive)`: Get authenticated HTTP context
- `api.ParseToken(userToken)`: Parse JWT to get user info
- `api.CreateApiCtx(httpCtx, syncVersion)`: Create API context

**Document Operations:**
- `ctx.UploadDocument(parentId, path, notify, coverpage)`: Upload document
- `ctx.FetchDocument(docId, dstPath)`: Download document
- `ctx.CreateDir(parentId, name, notify)`: Create directory
- `ctx.MoveEntry(src, dstDir, name)`: Move/rename
- `ctx.DeleteEntry(node, recursive, notify)`: Delete
- `ctx.ReplaceDocumentFile(docId, path, notify)`: Replace PDF content

**File Tree:**
- `ctx.Filetree().NodeByPath(path, current)`: Resolve path
- `ctx.Filetree().Root()`: Get root node
- `ctx.Filetree().AddDocument(doc)`: Add document to tree

**Sync:**
- `ctx.Refresh()`: Re-sync with remote
- `ctx.SyncComplete()`: Notify sync complete

### Key Types

- `api.ApiCtx`: Main API interface
- `sync15.ApiCtx`: Implementation struct
- `sync15.HashTree`: Document index
- `sync15.BlobDoc`: Document representation
- `model.Document`: High-level document
- `model.Node`: File tree node
- `filetree.FileTreeCtx`: File tree manager
- `transport.HttpClientCtx`: HTTP client

## Related

- ReMarkable Wiki: https://remarkablewiki.com/tech/filesystem
- Archive format documentation: See `archive/doc.go` package docs
- `.rm` format: See `encoding/rm/` package
