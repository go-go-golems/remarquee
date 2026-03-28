# Tasks

## Execution Plan

- [x] Task 1: Inspect the markdown upload pipeline and choose a shared extension seam for a named layout preset
- [x] Task 2: Implement the editor-friendly layout preset in `pkg/mdpdf`, including layered LaTeX header support
- [x] Task 3: Wire `--layout` into `remarquee upload md` and `remarquee upload bundle`, preserving explicit override precedence
- [x] Task 4: Add focused tests and update the embedded markdown upload reference docs
- [x] Task 5: Create the RMQ-0014 design doc, intern guide, diary, and related ticket bookkeeping
- [x] Task 6: Validate with `go test` and `docmgr doctor`, then upload the bundled docs to reMarkable and verify the remote listing

## Commit Plan

- [x] Commit A: feature code, tests, and embedded upload docs (`14858d3`)
- [x] Commit B: RMQ-0014 ticket docs, diary, changelog, tasks, and vocabulary updates

## Follow-up Candidates

- [ ] Evaluate whether `upload src` should also grow a layout preset, or whether code-review PDFs should remain separately tuned
- [ ] Test the editor layout on-device across longer documents and adjust the preset if the text block feels too narrow
