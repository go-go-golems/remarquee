# Tasks

## Investigation and design

- [x] Create RMQ-0020 ticket workspace.
- [x] Reproduce the reported Pandoc YAML parser failure with the user-provided document shape.
- [x] Map the shared `pkg/mdpdf` conversion pipeline and all upload callers.
- [x] Confirm that disabling `yaml_metadata_block` fixes the failure without rewriting Markdown.
- [x] Write the intern-ready analysis, design, API sketch, pseudocode, diagrams, implementation phases, and test strategy.
- [x] Write the chronological investigation diary.
- [x] Relate the design and diary to the relevant source files.
- [x] Validate ticket metadata with `docmgr doctor`.
- [x] Upload the design bundle to reMarkable.

## Production implementation

- [x] Add a testable Pandoc argument builder or equivalent internal argument contract.
- [x] Pass `--from=markdown-yaml_metadata_block` to every shared Markdown conversion.
- [x] Add a regression test for ordinary thematic breaks after prose.
- [x] Add a regression test for fenced YAML containing `---`.
- [x] Preserve and test leading docmgr frontmatter stripping.
- [x] Add bundle conversion coverage for separators and fenced examples.
- [x] Add or preserve batch continuation coverage with one failed and one successful conversion.
- [x] Decide whether a typed conversion error is needed and improve diagnostics if required.
- [x] Update user-facing Markdown upload documentation.
- [ ] Run package tests, upload-command tests, and full repository validation.
- [x] Record the implementation commit and final test evidence in the diary and changelog.
