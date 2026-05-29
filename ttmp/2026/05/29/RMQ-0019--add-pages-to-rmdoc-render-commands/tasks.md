# Tasks

## TODO

### 1. Ticket setup and research

- [x] Create ticket `RMQ-0019` for `--pages` support on `rmdoc render-v6` and `rmdoc render-legacy`
- [x] Inspect current render command wiring and existing page-subset helpers
- [x] Write an analysis and implementation guide
- [x] Upload the guide bundle to reMarkable

### 2. Shared page selector

- [ ] Add a command-layer page selection parser for comma-separated 1-based pages and inclusive ranges
- [ ] Validate selectors against the source document page count
- [ ] Add focused parser unit tests

### 3. Wire `rmdoc render-v6`

- [ ] Add the `--pages` flag and settings field
- [ ] Dispatch subset requests to `MergeRMDocV6OntoBackgroundPDFWithInfoForPages`
- [ ] Preserve full-document behavior when `--pages` is omitted
- [ ] Include selected pages in Glaze output
- [ ] Add/adjust V6 command tests

### 4. Wire `rmdoc render-legacy`

- [ ] Add the `--pages` flag and settings field
- [ ] Generate a temporary all-pages legacy PDF for subset requests
- [ ] Extract requested PDF pages into the final output
- [ ] Preserve full-document behavior when `--pages` is omitted
- [ ] Include selected pages in Glaze output
- [ ] Add/adjust legacy command tests

### 5. Documentation and validation

- [ ] Update README examples for `render-v6 --pages` and `render-legacy --pages`
- [ ] Run focused Go tests for touched packages
- [ ] Run command smoke tests against local fixtures
- [ ] Update diary, changelog, and file relations after implementation commits
