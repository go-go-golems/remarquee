---
Title: 'Files UI additions: directory picker column clarity and sorting'
Ticket: RMQ-0025
Status: draft
Topics:
    - remarkable
    - cloud
    - web
    - frontend
    - ui
    - go
    - remarquee
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/remarquee/cmds/serve/frontend/app.js
      Note: Move picker isolation and deterministic column sorting
    - Path: repo://cmd/remarquee/cmds/serve/frontend/index.html
      Note: Dedicated move dialog and sortable column headers
    - Path: repo://cmd/remarquee/cmds/serve/frontend/styles.css
      Note: Subtle directory text and non-wrapping dates
    - Path: repo://cmd/remarquee/cmds/serve/handlers.go
      Note: Existing list/move APIs and canonical list response path
    - Path: repo://pkg/rmfiles/entries.go
      Note: Proposed normalized authoritative fileType projection
    - Path: repo://pkg/rmfiles/manage.go
      Note: Authoritative move conflict and subtree checks
ExternalSources: []
Summary: 'Proposed additions to PR 29: navigable move destination picker, subtle directory text color, actual file-format metadata, non-wrapping dates and accessible deterministic column sorting.'
LastUpdated: 2026-10-04T08:15:00-04:00
WhatFor: Specify the requested files UI additions precisely enough for an intern to implement and test without confusing cloud kind with document format.
WhenToUse: When extending the files list or replacing the typed move destination prompt after addressing the PR review safety findings.
---

# Files UI additions: directory picker, column clarity, and sorting

## 1. Scope and requirement mapping

This is the **second design document** for RMQ-0025. It adds a focused interaction and presentation layer to the implementation submitted in PR 29. It is a proposal, not a record of implemented features. The original design remains historical architecture context; the implementation review describes the actual current API and unresolved correctness/security findings.

The user requested:

- “file browser on move”
- “subtle color for dirs in listing for better readability”
- “add file type column”
- “make sure date column is not wrapping”
- “allow dir browser in move command”
- “allow sorting by columns”

The first and fifth items describe the same user need: a **directory browser in the web UI's Move action**, replacing the need to type a destination path. This document treats them as one feature. It does not introduce an interactive terminal browser into `remarquee cloud mv`; that would be a separate CLI scope if the user intended it.

| Requirement | Proposed behavior | Main implementation area |
| --- | --- | --- |
| Browse destination when moving | Modal with current path, breadcrumbs, Up, folders, and “Move here” | Move dialog/controller and existing list/move API |
| Subtle directory color | Muted blue directory-name text only; monochrome backgrounds | CSS directory token and row label |
| File type column | Separate KIND from FILE TYPE; PDF, EPUB, Notebook, unknown | Cloud metadata projection, Entry contract, list renderer |
| Date never wraps | Single-line timestamp, fixed-content column, horizontal overflow | Date formatter and table CSS |
| Sort by columns | Accessible header buttons; ascending/descending; deterministic ties | Pure comparators and list view state |

**Not in scope:** bulk move, drag-to-move, overwrite, an embedded PDF viewer, folder-tree virtualization, or implementing the previous review's fixes in this documentation step. No reMarkable upload is requested for this second document, so delivery here is the ticket artifact rather than another automatic device upload.

## 2. Current system and the gaps

The browser holds `cwd`, `entries`, `selected`, `query`, and `status` in `frontend/app.js`. It gets a directory's children from `GET /api/files?dir=...`, and sends an ID-based move to `POST /api/entries/{id}/move` with `{destDir}`. The service resolves the path under its mutex and rejects sibling conflicts or moves into the source's subtree.

Current `promptMove` (`app.js:254`) uses the shared text prompt, initialized to `/`. There is no navigable destination list or persistent indication of which entry is being moved. `renderRows` (`app.js:156`) creates NAME, TYPE, and MODIFIED cells. `shortType` (`app.js:64`) reports “folder” or “document”; it does not know PDF versus EPUB. The existing TYPE cell also appends a version, which mixes unrelated metadata into a sortable category.

The server already returns folder-first, name-sorted directory children (`pkg/rmfiles/entries.go:105`). Search returns relevance-ranked results (`pkg/rmfiles/search.go:17`). The UI has no sorting controls, so explicit frontend sorting must preserve relevance as the default in search mode rather than silently replacing it.

The date cells have no nowrap rule (`styles.css:133`). Directory labels have a `[DIR]` prefix but no dedicated color. The detail pane and the list share the main state; a destination picker must not borrow that state and unexpectedly navigate the underlying browser.

### State and data ownership

```text
Main files browser                     Move destination dialog
+---------------------------+          +-----------------------------+
| view: directory / search  |          | source ID + source snapshot |
| cwd, rows, selected       |          | candidate destination       |
| sort key + direction      |          | folder rows + load epoch    |
+-------------+-------------+          | loading / error / submitting|
              |                        +---------------+-------------+
              | GET files/search                       | GET files
              +-------------------+--------------------+
                                  v
                          existing HTTP API
                                  |
                          serialized rmfiles service
                                  |
                          one cloud context + tree
```

The main browser and picker share transport functions and display helpers, not mutable navigation state. Existing HTTP and cloud boundaries stay in place.

## 3. Move destination browser

### 3.1 Interaction contract

Clicking Move on a selected file or folder opens a dedicated destination dialog. It shows the source name and path, the currently browsed destination, and directory children. Browsing changes the **candidate destination**, not the source or the main view.

```text
+----------------------------------------------------------------+
| MOVE “Project Plan”                                            |
| From: /ai/Notes/Project Plan                                    |
| Destination: / / Archive / 2026                                 |
| [Up]                         [Paste a path...]                   |
| -------------------------------------------------------------- |
| [DIR] Research                                                 |
| [DIR] Reading                                                  |
| -------------------------------------------------------------- |
| Current destination: /Archive/2026                              |
|                                     [Cancel] [Move here]        |
+----------------------------------------------------------------+
```

- Initialize at the source's current parent, not an unrelated `/` default.
- Root `/` is a valid destination and is always reachable through breadcrumbs.
- List only directories; documents are not selectable destinations.
- One activation (click or Enter) opens a directory. Avoid requiring double-click in a modal browser.
- Breadcrumbs and Up navigate; Up is disabled at root.
- Optional path entry is an escape hatch, not the only interface. It must perform a directory lookup before confirmation.
- “Move here” refers to the **currently loaded directory**, even when it has no child folders.
- Disable submission while loading, after a failed lookup, while submitting, or when the chosen parent is unchanged.
- Cancel and native Escape close without moving. Both settle any pending dialog promise exactly once.
- Keep a conflict/error message inside the dialog and retain the candidate destination for correction.
- After success, close and refresh the main browser according to its explicit view mode. Do not redirect the main browser to the destination unless a separate “Open destination” action is chosen.

No typed DELETE-style confirmation is needed for ordinary moves, but the dialog must show source and destination clearly and must never overwrite a conflicting entry. The service remains the final validator.

### 3.2 Protect invalid destination choices

If the source is a folder, its own directory and descendants cannot be destinations. Disable those paths in the picker for immediate feedback and keep backend `isSubdir` enforcement (`manage.go:151`) as authoritative. A stale dialog or direct HTTP client can bypass frontend checks.

Use exact segment boundaries for path comparisons, not naïve prefix matching:

```text
insideSource(sourcePath, candidate):
    return candidate == sourcePath
        or candidate starts with sourcePath + "/"
```

Thus `/Notes2` is not beneath `/Notes`. This check assumes canonical paths; navigation results must use a canonical resolved destination. If the source was renamed/moved externally after opening the dialog, an ID-based backend check still protects the tree. On errors, invalidate stale source/path assumptions and reload rather than offering a misleading destination.

### 3.3 State model and pseudocode

```text
movePicker = {
    source: {id, parentId, name, path, isDir},
    requestedPath: "/",
    loadedPath: null,
    folders: [],
    generation: 0,
    phase: "loading",  // loading | ready | error | submitting
    error: null
}

openMove(source):
    reset picker state
    navigatePicker(parentPath(source.path))
    open modal

navigatePicker(path):
    generation = increment picker generation
    phase = loading
    response = await listFiles(path)
    if dialog closed or generation no longer current: ignore
    loadedPath = canonical resolved response path
    folders = response.entries filtered by isDir
    phase = ready

submitMove():
    require ready + loadedPath + allowed destination
    phase = submitting
    POST /api/entries/{source.id}/move {destDir: loadedPath}
    on success: settle dialog; close; refresh main active view
    on failure: phase = error; display specific message
```

Use a picker-local generation or cancellation token; it must not invalidate main-browser requests. Closing increments the generation so late responses cannot repopulate a closed/reopened dialog. A source-ID snapshot, not the main selection variable, determines the entry being moved.

### 3.4 API impact

The existing list and move routes are sufficient for the first picker. No per-folder cloud calls beyond the existing service list are needed; listing reads the local tree under its lock.

**Small recommended correction:** return the canonical resolved path in `GET /api/files`, not the raw query string echoed by `handlers.go:112`. Typed aliases such as `/A/../B` should display the resolved destination and generate coherent breadcrumbs. This can be a corrected response `path`, derived by `BuildPath` from the resolved node. It must not be inferred solely by frontend string trimming.

A full `/api/files/tree` download is unnecessary for navigating a destination and would couple the picker to a larger, potentially stale snapshot. A future folder-search capability can be designed separately.

## 4. Directory readability and column layout

### 4.1 Subtle directory color

Add a dedicated token, distinct from bright link/error colors:

```css
:root { --directory-ink: #3f6075; }
td.name .folder { color: var(--directory-ink); }
table.files tr.row.selected td.name .folder { color: var(--paper); }
```

The value is a proposed muted blue, to be checked against white for readable contrast. Use it for directory names and `[DIR]` labels in both main listing and picker. Do not tint row backgrounds, date/type cells, borders, or the selected surface. The `[DIR]` prefix remains a non-color cue. Selected directory text must invert to white along with the rest of the row.

This preserves the standing user rule: **color only as text highlights**. The prior review's red button-hover/toast backgrounds are still separate defects; this addition must not introduce more colored fills.

### 4.2 New column meaning

Proposed list:

```text
NAME                         KIND       FILE TYPE    MODIFIED
[DIR] Reading                Folder     —            2026-10-04 08:10
[DOC] Plan                   Document   PDF          2026-10-04 08:05
[DOC] Novel                  Document   EPUB         2026-10-03 18:22
[DOC] Meeting notes          Document   Notebook     2026-10-02 09:15
```

- **KIND:** folder, document, or template; uses current model type.
- **FILE TYPE:** actual content format, not cloud object kind or source filename extension.
- **MODIFIED:** a formatted single-line date; raw timestamp remains available to the comparator/detail pane.
- Version remains in the detail pane instead of being appended to a category cell.

A Markdown upload becomes a PDF document: its FILE TYPE is PDF, not Markdown. A name ending in `.pdf` is not evidence that its underlying cloud content is PDF. Unknown type is shown honestly as `Unknown`; directories use an em dash and are not misclassified as notebooks.

### 4.3 Non-wrapping dates

```css
table.files th.mod,
table.files td.mod {
  white-space: nowrap;
  width: 1%;
  font-variant-numeric: tabular-nums;
}
table.files td.type,
table.files td.file-type { white-space: nowrap; }
.listpane { overflow-x: auto; }
```

Apply matching class names to header cells as well as data cells. `width: 1%` here requests a compact intrinsic column; it is not a promise that a timestamp fits into 1% of the table. `nowrap` provides the actual non-wrapping guarantee. Use a table minimum width or column sizing after measuring realistic long names; at narrow widths, scroll rather than wrap or silently truncate dates. Names may wrap or ellipsize under an explicit independent policy.

Use a robust date helper that treats missing, invalid, and Go zero timestamps as unknown. Render unknown as `—`; never show year 1 as a real modification date. Keep the chosen existing local-time display for this feature and document the timezone rather than silently changing to UTC.

## 5. Getting actual file types without guessing

### 5.1 Why this needs a backend projection change

Current `model.Document` contains `Type`, whose values are `CollectionType`, `DocumentType`, and `TemplateType`. It has **no file-format field**. Current `Entry` mirrors that omission (`pkg/rmfiles/entries.go:16`). Therefore adding a header and displaying `DocumentType` does not implement a useful PDF/EPUB/Notebook column.

The pinned dependency has the data earlier in its pipeline: `archive.Content.FileType` is documented as `pdf`, `epub`, or empty for a simple note (`rmapi/archive/file.go:129-134`). `sync15.BlobDoc` owns `Content`, but `BlobDoc.ToDocument` (`rmapi/api/sync15/blobdoc.go:273`) drops file type when constructing `model.Document`.

### 5.2 Proposed contract and narrow dependency change

Add a normalized API field:

```go
// Proposed additions, not implemented in the current PR.
type Entry struct {
    // existing fields retained
    FileType string `json:"fileType"` // pdf | epub | notebook | unknown; empty for folders
}
```

Preferred implementation is a small typed patch to the maintained rmapi fork, then a pinned dependency update:

1. Carry content format and whether content metadata was actually loaded into the projected `model.Document` (a separate known/loaded indicator prevents ambiguity).
2. Populate these fields in `BlobDoc.ToDocument`, not by reflection into the unexported hash tree.
3. Normalize them in `nodeToEntry` so HTTP clients see the documented vocabulary.
4. Verify cache loading, cloud refresh, and freshly uploaded documents all preserve the projection.
5. Add tests for PDF, EPUB, notebook, folder, template, absent metadata and unfamiliar future formats.

Mapping:

```text
folder                         -> fileType empty / display —
known loaded content "pdf"     -> pdf / PDF
known loaded content "epub"    -> epub / EPUB
known loaded empty content type -> notebook / Notebook
unavailable or unfamiliar type -> unknown / Unknown
```

An empty value **without proof of loaded content** is not enough to infer Notebook. Avoid downloading full `.rmdoc` archives merely to fill a table cell. If the fork patch cannot be shipped immediately, the UI may display Unknown explicitly, but **actual file-type support is not complete until the metadata path is implemented and validated**.

This is a new domain field, not a compatibility shim. It does not require a new HTTP route: list, tree, search, and mutation Entry responses all carry the same additive field.

### Decision: authoritative metadata rather than extension inference

- **Context:** cloud display names often omit extensions and can be arbitrary.
- **Alternatives:** infer from name; remember only uploads from this process; fetch each archive; expose already-loaded content metadata.
- **Decision:** expose the loaded content metadata through a typed dependency projection.
- **Rationale:** accurate for existing account documents without per-row downloads or fragile reflection.
- **Consequences:** a small fork/dependency update is required; Unknown is explicit when data is unavailable.
- **Status:** proposed.

## 6. Sorting by columns

### 6.1 User-facing behavior

Each sortable header contains a keyboard-focusable button. First activation sorts ascending; activating the same header toggles descending. Activating a different header starts ascending. An arrow and `aria-sort` communicate the active direction. The arrows are monochrome; direction is not conveyed by color alone.

Sort NAME, KIND, FILE TYPE, and MODIFIED. Keep **directories first** in ordinary directory browsing for either direction; direction reverses the selected key within groups, not the folder/file grouping. In global search, default order stays Relevance with no misleading active column arrow. Selecting a header explicitly overrides relevance; offer a visible “Relevance” reset control while searching.

Persist the active directory sort across navigation/refresh within the session. Keep separate search sort state so sorting search does not unexpectedly change folder browsing. URL encoding of sort preferences and long-term localStorage persistence are optional follow-ups, not required by this request.

### 6.2 Comparator rules

- Name: locale-aware `Intl.Collator` with `numeric: true`, `sensitivity: "base"` for readable numbered names.
- Kind: compare the displayed kind's defined rank/label, consistently with the folder-first grouping.
- File type: compare normalized display labels; unknown types last for either direction.
- Modified: compare epoch timestamps, not formatted date strings; missing/invalid dates last for either direction.
- Equal primary keys: deterministic name, path, then ID tie-breaks. The final ID comparison makes duplicate-name rows stable.
- Sort a copy of entries, not the canonical fetched array. Rendering and selection remain ID-based.

```text
compare(a, b, sort, mode):
    if mode is directory and a.isDir != b.isDir:
        return a.isDir ? -1 : 1
    ka = valueForColumn(a, sort.key)
    kb = valueForColumn(b, sort.key)
    if exactly one key is unknown: unknown goes last
    primary = compareTypedKeys(ka, kb)
    if primary != 0: return primary * directionSign(sort.direction)
    return collator.compare(a.name, b.name)
        or compare(a.path, b.path)
        or compare(a.id, b.id)

visibleRows():
    if search mode and sort is relevance: return fetched ranked order
    return copy(entries).sort(activeComparator)
```

Do not reverse the whole array for descending: doing so also reverses folder grouping, unknown-last behavior, and tie-break policy.

The KIND column may look redundant when folders are pinned first and almost every remaining row is Document. Its value is clear distinction from actual file type, including templates. If a later usability pass removes KIND, that is a deliberate layout choice; it must not overload FILE TYPE with version or cloud-kind strings.

### 6.3 Sorting is not request-order protection

The review already identifies asynchronous list/search response races and stale view modes. Sorting the newest response cannot fix accepting an older response. Implement the explicit mode/shared request-generation fix before or alongside column controls, and give the picker independent generation state. This avoids presenting neatly sorted data for the wrong directory.

### Decision: client-side sorting over fetched snapshots

- **Context:** directory/search rows are already returned as snapshots, without pagination.
- **Alternatives:** add server sort query parameters; client-side deterministic comparators.
- **Decision:** frontend sorting for this iteration.
- **Rationale:** immediate interaction, no redundant cloud work, reuse one comparator across views.
- **Consequences:** only fetched rows are sorted. Search is capped at 200 results by the current UI; sorting that subset does not globally sort every matching document in the account. Label “200 returned matches” honestly rather than implying complete account-wide ordering.
- **Status:** proposed; revisit server-side sort with pagination/global search requirements.

## 7. Files and implementation phases

### Phase 1 — confirm the contracts and metadata path

- Add the typed rmapi projection and pin its reviewed revision.
- Extend `pkg/rmfiles/entries.go` with normalized `fileType` and tests.
- Ensure Entry-producing create/rename/move/upload paths and search snapshots retain it.
- Canonicalize list response path using the resolved node rather than echoing the query.
- Address previous review F05/F06 prerequisites: live lookup and coherent view state.

**Gate:** existing PDF/EPUB/notebook fixtures identify correctly through list and search; unknown remains distinguishable; the source of metadata is documented.

### Phase 2 — implement listing readability and pure sorting

- `frontend/index.html`: add KIND/FILE TYPE headers and header buttons/classes.
- `frontend/app.js`: pure type/date normalization helpers, sort state and comparators, copy-sort rendering, no version suffix in kind.
- `frontend/styles.css`: muted directory text, selected inversion, nowrap metadata/date cells, horizontal overflow and accessible focus.

**Gate:** every column toggles deterministically; date never wraps; directory color obeys text-only accents; default search relevance survives until explicitly overridden.

### Phase 3 — implement the dedicated Move picker

- Add dialog DOM and an isolated picker controller; do not repurpose the shared rename input dialog.
- Add source summary, breadcrumbs, Up, directory rows, optional validated path field, inline error, and Move here.
- Wire existing list/move requests with local request generation, invalid-subtree feedback and native close/cancel handling.
- On success refresh main view without moving its navigation state.

**Gate:** keyboard and pointer users can move to root, an empty directory or a nested folder without typing. Errors do not close/reset the picker and no late response changes a reopened dialog.

### Phase 4 — regression and qualification

- Unit-test comparators, timestamp normalization and type mapping.
- Handler/service-test canonical paths and existing move guards with updated Entry metadata.
- Browser-test date cell line count/white-space, selected directory contrast, header keyboard activation, modal Escape and stale-response ordering.
- Run affected tests, build and race checks at this feature boundary.
- If real-cloud qualification is authorized, use only an isolated known-ID fixture and reconcile/verify cleanup; do not depend on the currently flawed recursive-delete path.

**Gate:** all requested behaviors are demonstrable; original PR safety findings remain separately tracked rather than silently marked resolved by this UI addition.

## 8. Acceptance matrix

| Scenario | Expected result |
| --- | --- |
| Move a file without typing a path | Browse folders, confirm loaded destination, send one move request |
| Move into empty folder or root | Move here is usable once loaded; no child selection required |
| Move folder into itself/descendant | Disabled in UI; backend still rejects direct request |
| Source/destination changes externally | Clear error/reload; no implicit overwrite or stale destination confirmation |
| Cancel/Escape/late picker response | No move; promise settles; main navigation unaffected |
| Long name + narrow viewport | Date is single line; list may scroll horizontally |
| Directory in normal/selected row | Subtle colored name normally; white text on black selection |
| PDF/EPUB/notebook without extension | Correct type from content metadata, not name |
| Missing metadata or invalid timestamp | Unknown type / em dash date; sorted last as specified |
| Sort ascending/descending with equal dates | Deterministic ties; folder-first preserved |
| Search then sort | Starts ranked; explicit override sorts fetched results; Relevance resets |
| Refresh after move/sort | Main active mode and sort retained; correct snapshots displayed |

## 9. References and unresolved decisions

### Primary source references

- `cmd/remarquee/cmds/serve/frontend/app.js:64,156,254`: kind label, row rendering, typed move prompt.
- `cmd/remarquee/cmds/serve/frontend/index.html:33`: current three-column table.
- `cmd/remarquee/cmds/serve/frontend/styles.css:119-138`: rows, selection and metadata styles.
- `cmd/remarquee/cmds/serve/handlers.go:102,164`: directory-list and move contracts.
- `pkg/rmfiles/entries.go:16,50,105`: Entry projection and folder-first children.
- `pkg/rmfiles/manage.go:70,151`: move validation and subtree protection.
- Pinned rmapi `model/document.go`: no actual format field in current projection.
- Pinned rmapi `archive/file.go:129-134`: actual `Content.FileType` semantics.
- Pinned rmapi `api/sync15/blobdoc.go:273`: current ToDocument projection drops format.

### Related ticket documents

- [Original architecture proposal](01-remarkable-files-web-ui-design-and-implementation-guide-for-a-new-intern.md).
- [Evidence-backed PR 29 review](../analysis/01-pr-29-intern-facing-architecture-design-and-implementation-review.md).
- [Implementation diary](../reference/01-implementation-diary.md).

### Questions to resolve during implementation

1. Confirm that “move command” means the web UI action; terminal browsing is not included here.
2. Approve the narrow typed metadata enhancement in the maintained rmapi fork before promising true document formats for existing cloud entries.
3. Confirm local-time date display and the proposed folder-first rule; these are documented defaults, not hidden comparator behavior.
4. Decide whether the main list needs KIND as a separate column after adding FILE TYPE; this design keeps both to avoid semantic ambiguity.

The feature should remain small: isolated picker state, one accurate metadata projection, a few pure comparators, and targeted CSS. None of these additions require adopting another frontend framework or weakening the service's cloud serialization.
