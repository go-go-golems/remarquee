# Tasks

## TODO

### 1. Ticket setup and research

- [x] Create ticket `RMQ-0019` for `--pages` support on `rmdoc render-v6` and `rmdoc render-legacy`
- [x] Inspect current render command wiring and existing page-subset helpers
- [x] Write an analysis and implementation guide
- [x] Upload the guide bundle to reMarkable

### 2. Shared page selector

- [x] Add a command-layer page selection parser for comma-separated 1-based pages and inclusive ranges
- [x] Validate selectors against the source document page count
- [x] Add focused parser unit tests

### 3. Wire `rmdoc render-v6`

- [x] Add the `--pages` flag and settings field
- [x] Dispatch subset requests to `MergeRMDocV6OntoBackgroundPDFWithInfoForPages`
- [x] Preserve full-document behavior when `--pages` is omitted
- [x] Include selected pages in Glaze output
- [x] Add/adjust V6 command tests

### 4. Wire `rmdoc render-legacy`

- [x] Add the `--pages` flag and settings field
- [x] Generate a temporary all-pages legacy PDF for subset requests
- [x] Extract requested PDF pages into the final output
- [x] Preserve full-document behavior when `--pages` is omitted
- [x] Include selected pages in Glaze output
- [x] Add/adjust legacy command tests

### 5. Documentation and validation

- [x] Update README examples for `render-v6 --pages` and `render-legacy --pages`
- [x] Run focused Go tests for touched packages
- [x] Run command smoke tests against local fixtures
- [x] Update diary, changelog, and file relations after implementation commits
