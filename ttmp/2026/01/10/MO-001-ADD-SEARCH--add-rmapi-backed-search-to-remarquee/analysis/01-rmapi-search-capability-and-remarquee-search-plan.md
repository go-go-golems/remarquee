---
Title: RMAPI Search Capability and Remarquee Search Plan
Ticket: MO-001-ADD-SEARCH
Status: active
Topics:
    - backend
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarquee/cmd/remarquee/cmds/cloud/find.go
      Note: existing cloud find command behavior
    - Path: remarquee/cmd/remarquee/cmds/cloud/ls.go
      Note: existing cloud ls output + filters
    - Path: rmapi/api/sync15/apictx.go
      Note: rmapi tree mirror + refresh behavior
    - Path: rmapi/filetree/filetree.go
      Note: filetree traversal and path resolution primitives
    - Path: rmapi/model/node.go
      Note: node pattern matching and metadata fields
    - Path: rmapi/shell/find.go
      Note: rmapi regex-based find behavior and output formatting
ExternalSources: []
Summary: Map current rmapi/remarquee search-like behavior and propose a concrete, rmapi-backed search command with filters, output formats, and implementation steps.
LastUpdated: 2026-01-10T15:37:41.578969383-05:00
WhatFor: Design and implement a remarquee cloud search command that builds on rmapi filetree data.
WhenToUse: Use when adding a search command or extending rmapi-backed cloud querying in remarquee.
---


# RMAPI Search Capability and Remarquee Search Plan

## Goal

Provide a detailed, implementation-ready analysis for adding a `search` command to remarquee's rmapi-backed cloud tooling. The analysis captures current rmapi functionality, existing remarquee command behavior, and a concrete plan for a search command that is consistent with rmapi semantics while providing structured output and useful filters.

## Current rmapi capabilities relevant to search

### Filetree model and traversal

rmapi's search-like behavior is built on the in-memory **filetree** constructed from the sync15 hash tree.

Key components:

- `rmapi/api/sync15/apictx.go`:
  - `CreateCtx(...)` mirrors the server tree into a local hash tree and builds a `filetree.FileTreeCtx`.
  - `Refresh()` can update the tree, but `CreateCtx(...)` already mirrors on creation.
- `rmapi/filetree/filetree.go`:
  - `NodeByPath(path, current)` resolves a single absolute/relative path.
  - `NodesByPath(path, current, ignoreTrailingSlash)` supports glob-like patterns on the *last* path segment via `FindByPattern`.
  - `WalkTree(start, visitor)` traverses recursively with a visitor callback.
- `rmapi/model/node.go`:
  - `FindByPattern` uses `filepath.Match` on lowercased names (glob-style, case-insensitive).
  - `LastModified()` parses the `ModifiedClient` field into a `time.Time`.
- `rmapi/model/document.go`:
  - `Document` fields: `ID`, `Name`, `Version`, `ModifiedClient`, `Type`, `CurrentPage`, `Parent`.
  - **No pinned/deleted/synced flags** in the `Document` struct (those live in metadata files but are not surfaced in `Document`).

Implications:

- rmapi search can only query on **name/path/type/modified time** without additional metadata fetch.
- Any content-based or metadata-based search requires **downloading documents** and/or reading metadata blobs.

### rmapi shell commands (user-facing search-like behaviors)

- `rmapi/shell/find.go`:
  - Command: `find [dir] [regexp]`
  - Uses `filetree.WalkTree` and matches a **Go regexp** against a formatted output string.
  - Output format (non-compact): `[d] path/to/entry` or `[f] path/to/entry`.
  - Default start directory is the current shell path (not necessarily root).
- `rmapi/shell/ls.go`:
  - Uses `NodesByPath` with globbing on the last path segment.
  - Supports sorting and directory-first grouping.
  - Optional `--show-templates` flag to include `TemplateType` entries.
- `rmapi/shell/stat.go`:
  - `stat <path>` prints JSON for the underlying `Document` metadata.

rmapi README documents `find` as the primary search mechanism, and it is regex-based with Go standard regexp syntax.

### rmapi metadata beyond `Document`

- `rmapi/archive/file.go` defines metadata fields such as `Pinned`, `Deleted`, `LastOpened`, `Synced`.
- The sync15 `BlobDoc` converts metadata into `Document` (see `rmapi/api/sync15/blobdoc.go`).
- These extended metadata fields are **not** exposed through the `ApiCtx` interface or `filetree` nodes.

Implication: **Search filters on pinned/synced/deleted are not available unless the search path is extended to read metadata files per document.**

## Current remarquee rmapi-backed functionality

### Existing cloud commands and search-like behavior

- `remarquee cloud find` (`remarquee/cmd/remarquee/cmds/cloud/find.go`):
  - Similar to rmapi `find`.
  - Walks the filetree from a start node (default `/`).
  - Matches a regexp against the **formatted output string** (includes `[d]`/`[f]` prefix in non-compact mode).
  - Output is plain text only (no Glazed output).
- `remarquee cloud ls` (`remarquee/cmd/remarquee/cmds/cloud/ls.go`):
  - Uses `NodesByPath` with globbing on the last segment.
  - Supports Glazed output (`--with-glaze-output`), including fields like `path`, `id`, `type`, `modified_time`.
  - Hides templates by default (unless `--show-templates` is set).
- `remarquee cloud stat` (`remarquee/cmd/remarquee/cmds/cloud/stat.go`):
  - Prints metadata for a single entry, plain or Glazed output.

### Notable behavior differences vs rmapi

- **Path formatting**:
  - rmapi `find` uses the traversal path as built during `WalkTree` (relative to the start node).
  - remarquee `find` uses `buildPathFromParents(node)` and always prints absolute paths.
- **Match target**:
  - Both rmapi and remarquee `find` match regex against the *formatted output string*, not against raw `path` or `name`.
  - This means a regex anchored to `^/` will not match when non-compact output is used (because output starts with `[d] ` or `[f] `).

Implication: a new `search` command should clarify whether it matches against **raw path**, **name**, or the **formatted output**, to avoid confusion.

## What "search" should mean for remarquee

Given rmapi's limitations, there are two realistic scopes:

1. **Name/path search** (fast, based on local filetree):
   - Regex, substring, or glob matching on path or name.
   - Filter by type (directory/document/template).
   - Filter by modified time (from `Document.ModifiedClient`).

2. **Content or metadata search** (slow, requires downloads):
   - Match inside PDF/EPUB/notebook content, OCR text, or annotations.
   - Filter by pinned, deleted, last-opened, etc.
   - Requires fetching `*.metadata` or full `.rmdoc` content per document.

**Recommendation:** implement scope (1) now and make scope (2) a future optional extension.

## Proposed command design (Phase 1)

Add a new command: `remarquee cloud search`. This command should be explicit in its matching semantics and provide Glazed output.

### Proposed CLI

```
remarquee cloud search [start] [query]
```

Suggested flags:

- Matching:
  - `--regex` (default: false) - treat `query` as regex.
  - `--match path|name` (default: path) - match against full path or just entry name.
  - `--case-sensitive` (default: false) - apply to substring matches.
- Filters:
  - `--type dir|file|template` (optional) - filter by `Document.Type`.
  - `--include-templates` (default: false) - include `TemplateType` entries (if default is hidden).
  - `--modified-after <time>` / `--modified-before <time>` - filter by modification time; allow RFC3339 and natural date parsing if possible.
- Output:
  - `--compact` (default: false) - same output style as `find`.
  - `--with-glaze-output` - structured output (JSON, CSV, etc.).
  - `--sort path|time` (optional) - deterministic ordering.
  - `--limit N` (optional) - stop after N matches.

### Output structure

Text mode (non-compact) should align with `find` output but with matching done against **raw path/name** for clarity.

Glazed output should include:

- `id`, `name`, `type`, `is_dir`, `path`, `parent_id`, `version`, `modified_client`, `modified_time`

This matches existing `ls` and `stat` output fields, enabling easy scripting.

## Implementation details (Phase 1)

### High-level flow

1. Parse auth + search flags.
2. Call `createApiCtx(...)` to build `ApiCtx` and filetree.
3. Resolve `start` to a node using `NodeByPath`.
4. Walk the tree using `filetree.WalkTree`.
5. For each node:
   - Compute `path := buildPathFromParents(node)`.
   - Apply filters in order: type, templates, time, query.
   - Emit results (text or Glazed).
   - Stop early if `--limit` reached.

Pseudo-code sketch:

```go
startNode, err := apiCtx.Filetree().NodeByPath(s.Start, nil)
if err != nil { return errors.New("start directory doesn't exist") }

var matcher func(node *model.Node, path string) (bool, error)
// Build matcher from flags (regex or substring on path/name).

count := 0
filetree.WalkTree(startNode, filetree.FileTreeVistor{
    Visit: func(node *model.Node, _ []string) bool {
        path := buildPathFromParents(node)
        ok, err := matcher(node, path)
        if err != nil { return true } // stop on error
        if ok {
            emit(node, path)
            count++
            if s.Limit > 0 && count >= s.Limit {
                return true
            }
        }
        return false
    },
})
```

### Filter implementation details

- **Regex match**:
  - Compile once (`regexp.Compile`), error on invalid regex.
  - Apply to raw `path` or `node.Name()` depending on `--match`.
- **Substring match**:
  - Lowercase both strings if case-insensitive.
  - Use `strings.Contains`.
- **Type filter**:
  - Map `dir` -> `CollectionType`, `file` -> `DocumentType`, `template` -> `TemplateType`.
- **Time filter**:
  - Use `node.LastModified()` for a `time.Time`.
  - If parse fails, either exclude the node or surface an error; prefer explicit error to avoid silent omission.

### Template handling

Current behavior:

- `remarquee cloud ls` hides templates by default.
- `remarquee cloud find` includes templates (no filter).

For `search`, pick one behavior and document it. Recommendation: **default to hiding templates** for safety and consistency with `ls`, with `--include-templates` opt-in.

### Ordering

The filetree traversal uses map iteration order, which is **non-deterministic**. For stable search output, either:

- Collect matches and sort by `path` (default), or
- Provide `--sort` flag to sort by time/name/path.

Sorting costs memory but gives consistent output and testability.

## Future extension (Phase 2: metadata/content search)

If required, add optional flags that **download metadata or content**:

- `--with-metadata`:
  - Read `*.metadata` per document (requires extra API access not currently exposed in `ApiCtx`).
  - Enables filters like `pinned`, `deleted`, `last_opened`.
- `--with-content`:
  - Use `FetchDocument` to download `.rmdoc` archives.
  - Parse `content.json` or PDFs to extract OCR/text (if available).
  - This is expensive and should be opt-in with explicit warnings.

This would likely require either:

- Extending the `ApiCtx` interface to expose lower-level sync15 APIs, or
- Adding a new helper in `pkg/rmcloud` that asserts `ApiCtx` to `*sync15.ApiCtx` and accesses metadata.

## Documentation updates

When adding `cloud search`, update:

- `remarquee/pkg/doc/cloud/02-remarquee-cloud-reference.md` - new section documenting command usage and flags.
- `remarquee/pkg/doc/cloud/01-getting-started-remarquee-cloud.md` - mention search in browsing/discovery examples.

## Testing strategy

Because live rmapi tokens are required for integration tests, add unit-level tests that run without network access:

- Build a synthetic `filetree.FileTreeCtx` and nodes to test filtering logic.
- Test regex vs substring matching and case sensitivity.
- Test type filtering and template inclusion.
- Test time-based filters using known `ModifiedClient` timestamps.

For manual testing (documented in a playbook or README):

```
remarquee cloud search / "\.pdf$" --regex
remarquee cloud search /Books "meeting" --match name
remarquee cloud search / --modified-after "2025-01-01"
```

## Open questions / decisions to make

1. **Match target default**: path vs name.
2. **Template inclusion default**: match `ls` (hide) or `find` (show).
3. **Output format**: reuse `find` formatting or use `ls`-style rows in text mode.
4. **Ordering**: deterministic sorting vs fast traversal order.
5. **Should `search` be an alias of `find` or a new command?**

## Summary recommendation

Implement `remarquee cloud search` as a **name/path search over the rmapi filetree**, with regex/substring matching, Glazed output, and filters for type and modified time. Keep content/metadata search out of scope for now; document it as a future extension that requires additional API support and explicit opt-in behavior.
