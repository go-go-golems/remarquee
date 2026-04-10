---
Title: Diary
Ticket: RMQ-0017
Status: active
Topics:
    - go
    - cli
    - migration
    - glazed
    - remarquee
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/remarquee/cmds/cloud/account.go
      Note: First simple cloud command targeted for the migration baseline
    - Path: cmd/remarquee/cmds/cloud/find.go
      Note: Representative dual-mode command for preserving structured-output behavior during migration
    - Path: cmd/remarquee/cmds/ocr/root.go
      Note: |-
        OCR is the most complex migration slice because it integrates Geppetto sections and middlewares
        OCR migrated in Step 4
    - Path: cmd/remarquee/cmds/rmdoc/input_resolver.go
      Note: Rmdoc helper migrated in Step 3
    - Path: cmd/remarquee/cmds/rmdoc/render_v6.go
      Note: Representative rmdoc orchestration command to migrate after the cloud slice
    - Path: cmd/remarquee/cmds/rmdoc/test_values_helpers_test.go
      Note: Test migration helper added in Step 3
    - Path: cmd/remarquee/main.go
      Note: Final root command cleanup in Step 4
    - Path: go.mod
      Note: Dependency bump and tidy noted in Steps 2 and 4
ExternalSources: []
Summary: Step-by-step implementation diary for migrating remarquee from Glazed's legacy layers/parameters API to the current schema/fields/values facade API.
LastUpdated: 2026-04-10T09:40:00-04:00
WhatFor: Record what changed, why, what failed, what was tricky, and how to validate each migration slice in RMQ-0017.
WhenToUse: Use when continuing implementation, reviewing commits, or reconstructing the migration/debugging path for RMQ-0017.
---


# Diary

## Goal

Capture the full implementation path for migrating `remarquee` to the new Glazed facade API, including the migration order, exact commands run, failures, and validation steps.

## Step 1: Baseline scan and migration plan

I started by turning the placeholder ticket workspace into a continuation-friendly implementation ticket before changing code. The goal of this first step was to answer three questions concretely: how many files still depend on the removed Glazed APIs, whether the repository still passes tests before the migration starts, and whether OCR is blocked on upstream Geppetto compatibility.

The result was a real execution plan rather than a generic intention note. I confirmed that most command files can be migrated mechanically, while OCR should be saved for the last slice because it touches Geppetto sections, inference engine creation, and a custom parsing middleware chain.

### Prompt Context

**User prompt (verbatim):** "go ahead, migrate them all, keep a diary, commit at appropriate intervals."

**Assistant interpretation:** Perform the full RMQ-0017 migration now, keep a detailed implementation diary, and break the work into reviewable commits.

**Inferred user intent:** The user wants the migration completed end-to-end in a disciplined way, with enough documentation and commit structure for later review or handoff.

### What I did
- Created ticket `RMQ-0017` in the `remarquee` docmgr workspace.
- Read the Glazed migration playbook at `/home/manuel/code/wesen/go-go-golems/glazed/pkg/doc/tutorials/migrating-to-facade-packages.md`.
- Scanned `remarquee` for legacy Glazed API usage across command files and tests.
- Created and filled:
  - `ttmp/2026/04/10/RMQ-0017--migrate-remarquee-to-the-new-glazed-facade-api/design/01-implementation-guide-glazed-facade-migration.md`
  - `ttmp/2026/04/10/RMQ-0017--migrate-remarquee-to-the-new-glazed-facade-api/tasks.md`
  - `ttmp/2026/04/10/RMQ-0017--migrate-remarquee-to-the-new-glazed-facade-api/reference/01-diary.md`
- Ran a full repo baseline test:
  - `cd /home/manuel/code/wesen/go-go-golems/remarquee && go test ./...`
- Checked the local Geppetto checkout for modern APIs to reduce uncertainty around OCR.
- Ran `docmgr doctor --ticket RMQ-0017 --stale-after 30` after adding vocabulary entries and relating the key files.

### Why
- A migration of this size is easy to derail if the execution order is vague.
- The user explicitly asked for a diary and appropriately spaced commits, so the ticket needed to be operational before code changes began.
- OCR needed early confirmation because it is the only slice that mixes Glazed and Geppetto migration patterns.

### What worked
- The repo passes `go test ./...` before the migration, giving a clean behavioral baseline.
- The scan produced a concrete scope:
  - 23 affected Go files
  - 75 `glazed.parameter` tag hits
  - 23 legacy Glazed settings-constructor hits
  - 16 legacy test-helper hits
- Geppetto is not blocked on legacy APIs:
  - `factory.NewEngineFromParsedValues(...)` exists
  - `geppettosections.CreateGeppettoSections(...)` exists
  - `geppettosections.GetCobraCommandGeppettoMiddlewares(...)` exists
- `docmgr doctor` passed after adding the missing vocabulary topics.

### What didn't work
- `docmgr doctor --ticket RMQ-0017 --stale-after 30` initially failed with unknown topics:

```text
[WARNING] unknown_topics — unknown topics: [glazed go migration remarquee]
```

- Fix applied by adding vocabulary entries:
  - `docmgr vocab add --category topics --slug glazed --description "Glazed CLI framework and command/schema/value APIs"`
  - `docmgr vocab add --category topics --slug go --description "Go language implementation work"`
  - `docmgr vocab add --category topics --slug migration --description "Planned or in-progress codebase migrations and API cutovers"`
  - `docmgr vocab add --category topics --slug remarquee --description "Repository-wide remarquee CLI and library work"`

### What I learned
- The migration is broad but mostly repetitive for cloud and rmdoc commands.
- OCR is best handled last, but it should not require an upstream Geppetto release first.
- The current `remarquee` command files are already partly modernized in places (for example dual-mode builder usage), which means preserving behavior matters more than inventing new abstractions.

### What was tricky to build
- The tricky part was separating “mechanically migratable” files from “special-case” files before touching code. The symptoms were that a simple grep made everything look equally legacy, but in practice OCR has a second dependency surface through Geppetto. I resolved that by checking the current local Geppetto checkout directly instead of assuming OCR was blocked.
- Another small but important issue was docmgr vocabulary drift. The ticket content was fine, but `docmgr doctor` would have stayed noisy until the new topics were formally registered.

### What warrants a second pair of eyes
- The final OCR migration should be reviewed carefully against the live Geppetto helper signatures in case there are subtle expectations around profile/config precedence.
- The dual-mode cloud migration should be checked for regressions in JSON default output behavior.

### What should be done in the future
- After the migration lands, consider a smaller follow-up ticket to normalize repeated cloud auth flag definitions if the migration reveals obvious duplication that is not worth refactoring inline.

### Code review instructions
- Start with:
  - `ttmp/2026/04/10/RMQ-0017--migrate-remarquee-to-the-new-glazed-facade-api/design/01-implementation-guide-glazed-facade-migration.md`
  - `ttmp/2026/04/10/RMQ-0017--migrate-remarquee-to-the-new-glazed-facade-api/tasks.md`
  - `ttmp/2026/04/10/RMQ-0017--migrate-remarquee-to-the-new-glazed-facade-api/reference/01-diary.md`
- Then review the first code slice once it lands, starting with the simple cloud commands.
- Baseline validation command:
  - `cd /home/manuel/code/wesen/go-go-golems/remarquee && go test ./...`

### Technical details
- Baseline scan command:

```bash
cd /home/manuel/code/wesen/go-go-golems/remarquee && rg -n "github.com/go-go-golems/glazed/pkg/cmds/parameters|github.com/go-go-golems/glazed/pkg/cmds/layers|glazed\.parameter:|NewGlazedParameterLayers|WithOutputParameterLayerOptions|WithLayersList|InitializeStruct\(|ShortHelpLayers|MiddlewaresFunc: func\(parsedCommandLayers \*layers\.ParsedLayers|cmdmiddlewares\.ParseFromCobraCommand|CreateGeppettoLayers|NewEngineFromParsedLayers|NewParameterLayer|NewParsedParameters|NewParsedLayers" cmd pkg -g '*.go'
```

- Geppetto compatibility checks:

```bash
cd /home/manuel/code/wesen/go-go-golems/geppetto && rg -n "CreateGeppettoSections|NewEngineFromParsedValues|GetCobraCommandGeppettoMiddlewares" pkg cmd -g '*.go'
```

- Ticket validation:

```bash
cd /home/manuel/code/wesen/go-go-golems/remarquee && docmgr doctor --ticket RMQ-0017 --stale-after 30
```

## Step 2: Upgrade dependencies and migrate the entire cloud command package

The first real code slice was the whole `cloud` package. I chose to migrate all cloud commands in one pass because the package has to compile as a unit, and the dual-mode commands (`ls`, `find`, `stat`, `refresh`) share the same value-decoding and section-construction patterns as the simpler commands.

This step also forced the dependency question into the open. The moment I switched the code to the new API, the existing `glazed v0.7.8` and `geppetto v0.6.0` pins stopped being viable, so I upgraded both modules to current releases and then fixed the remaining compile gaps in the cloud package itself.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Continue the migration in stable slices and record exactly what changed and why.

**Inferred user intent:** The user wants a real end-to-end migration, not just planning documents, with stable intermediate commits.

**Commit (code):** 472aba7 — "Migrate cloud commands to glazed facade API"

### What I did
- Upgraded module dependencies in `go.mod` / `go.sum`:
  - `github.com/go-go-golems/glazed` `v0.7.8` → `v1.2.1`
  - `github.com/go-go-golems/geppetto` `v0.6.0` → `v0.11.9`
- Migrated all cloud command files from legacy Glazed APIs to the facade APIs:
  - `cmd/remarquee/cmds/cloud/account.go`
  - `cmd/remarquee/cmds/cloud/find.go`
  - `cmd/remarquee/cmds/cloud/get.go`
  - `cmd/remarquee/cmds/cloud/ls.go`
  - `cmd/remarquee/cmds/cloud/mkdir.go`
  - `cmd/remarquee/cmds/cloud/mv.go`
  - `cmd/remarquee/cmds/cloud/put.go`
  - `cmd/remarquee/cmds/cloud/refresh.go`
  - `cmd/remarquee/cmds/cloud/rm.go`
  - `cmd/remarquee/cmds/cloud/rmapi.go`
  - `cmd/remarquee/cmds/cloud/search.go`
  - `cmd/remarquee/cmds/cloud/stat.go`
  - `cmd/remarquee/cmds/cloud/version.go`
- Applied the standard migration pattern in those files:
  - `layers` → `schema`
  - `parameters` → `fields`
  - `*layers.ParsedLayers` → `*values.Values`
  - `InitializeStruct(...)` → `DecodeSectionInto(...)`
  - `settings.NewGlazedParameterLayers(...)` → `settings.NewGlazedSection(...)`
  - `cli.NewCommandSettingsLayer()` → `cli.NewCommandSettingsSection()`
  - `WithLayersList(...)` → `WithSections(...)`
  - `ShortHelpLayers` → `ShortHelpSections`
  - `glazed.parameter` tags → `glazed`
- Preserved the existing dual-mode behavior and JSON default glaze output for:
  - `cloud ls`
  - `cloud find`
  - `cloud stat`
  - `cloud refresh`
- Added the transitive module entry Glazed needed for the upgraded command parser stack:
  - `go get github.com/go-go-golems/glazed/pkg/cmds/sources@v1.2.1`
- Validated the slice with:
  - `cd /home/manuel/code/wesen/go-go-golems/remarquee && go test ./cmd/remarquee/cmds/cloud/...`

### Why
- The cloud package was the largest cluster of directly related command files and the cleanest place to establish the new coding pattern.
- Upgrading dependencies early removes ambiguity about whether the migration actually works against the modern APIs or only against transitional shims.
- Preserving the dual-mode UX in the same slice keeps the migration behaviorally honest.

### What worked
- The dependency bump correctly exposed all old-package usage as real compile problems instead of letting them hide behind compatibility aliases.
- The cloud package now compiles cleanly against the modern Glazed release.
- The JSON default glaze output pattern migrated cleanly using `settings.NewGlazedSection(... settings.WithOutputSectionOptions(schema.WithDefaults(...)))`.
- The cloud-specific validation succeeded:

```bash
cd /home/manuel/code/wesen/go-go-golems/remarquee && go test ./cmd/remarquee/cmds/cloud/...
```

### What didn't work
- The first attempt to compile the new code failed because `remarquee` was still pinned to older module versions that do not expose the new section/value API:

```text
undefined: settings.NewGlazedSection
undefined: cli.NewCommandSettingsSection
undefined: glazecmds.WithSections
parsedValues.DecodeSectionInto undefined
unknown field ShortHelpSections in struct literal of type cli.CobraParserConfig
```

- After upgrading dependencies, `go get ... && go mod tidy` failed because the repo still contained legacy imports in non-cloud files, especially OCR:

```text
github.com/go-go-golems/remarquee/cmd/remarquee/cmds/ocr imports
	github.com/go-go-golems/geppetto/pkg/layers: module github.com/go-go-golems/geppetto@latest found (v0.11.9), but does not contain package github.com/go-go-golems/geppetto/pkg/layers
```

and:

```text
github.com/go-go-golems/remarquee/cmd/remarquee/cmds/ocr imports
	github.com/go-go-golems/glazed/pkg/cmds/layers: module github.com/go-go-golems/glazed@latest found (v1.2.1), but does not contain package github.com/go-go-golems/glazed/pkg/cmds/layers
```

- The first mechanical migration pass also missed one `refresh.go` signature, which left a broken `RunIntoGlazeProcessor` using `*layers.ParsedLayers` after the import had already been migrated.

### What I learned
- The migration and dependency bump really are inseparable. The code changes only make sense once the repo is on modern Glazed/Geppetto versions.
- Package-level migrations are safer than file-by-file commits when a package must compile as a unit.
- The dual-mode cloud commands were not materially harder than the simple commands once the output-default helper was updated correctly.

### What was tricky to build
- The dependency bump was the sharpest edge in this step. The symptom was a wall of `undefined` errors after switching to the new APIs, which looked like a code mistake at first glance. The real cause was version skew: the code had been migrated, but `remarquee` was still compiling against old module releases. I fixed that by upgrading both `glazed` and `geppetto`, then treating the remaining errors as the real migration backlog.
- The second tricky point was that `go mod tidy` could not complete yet because OCR and other non-migrated files still referenced removed packages. I handled that by validating the cloud package in isolation and deferring full-repo tidy until more of the migration is complete.

### What warrants a second pair of eyes
- Review the `go.mod` / `go.sum` bump for any surprising transitive changes beyond Glazed/Geppetto.
- Review the four dual-mode cloud commands specifically to confirm no human-readable behavior or field naming regressed.

### What should be done in the future
- Once `rmdoc` and `ocr` are migrated, run a final `go mod tidy` and verify that no transitive leftovers from the old dependency set remain.

### Code review instructions
- Start with:
  - `go.mod`
  - `cmd/remarquee/cmds/cloud/rmapi.go`
  - `cmd/remarquee/cmds/cloud/account.go`
  - `cmd/remarquee/cmds/cloud/find.go`
  - `cmd/remarquee/cmds/cloud/ls.go`
  - `cmd/remarquee/cmds/cloud/stat.go`
  - `cmd/remarquee/cmds/cloud/refresh.go`
- Validate with:
  - `cd /home/manuel/code/wesen/go-go-golems/remarquee && go test ./cmd/remarquee/cmds/cloud/...`

### Technical details
- Dependency upgrade command:

```bash
cd /home/manuel/code/wesen/go-go-golems/remarquee && go get github.com/go-go-golems/glazed@v1.2.1 github.com/go-go-golems/geppetto@v0.11.9
```

- Additional module fix for new Glazed transitive requirements:

```bash
cd /home/manuel/code/wesen/go-go-golems/remarquee && go get github.com/go-go-golems/glazed/pkg/cmds/sources@v1.2.1
```

- Cloud-package validation:

```bash
cd /home/manuel/code/wesen/go-go-golems/remarquee && go test ./cmd/remarquee/cmds/cloud/...
```

## Step 3: Migrate the rmdoc command family and tests

The second command-family slice was `rmdoc`. I tackled the shared input resolver and all command files first, then rewrote the tests to use the command descriptions' real default sections instead of fabricating legacy parsed-layer objects by hand.

This step was smoother than the cloud package because the migration pattern was already established. The only substantial extra work was modernizing the tests in a way that would stay aligned with the command schemas over time.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Continue the migration package by package, keeping the history reviewable and the diary current.

**Inferred user intent:** The user wants each major subsystem migrated and validated cleanly before the next one begins.

**Commit (code):** 1341fbb — "Migrate rmdoc commands to glazed facade API"

### What I did
- Migrated the shared helper and rmdoc command files:
  - `cmd/remarquee/cmds/rmdoc/input_resolver.go`
  - `cmd/remarquee/cmds/rmdoc/inspect.go`
  - `cmd/remarquee/cmds/rmdoc/build_background.go`
  - `cmd/remarquee/cmds/rmdoc/render_v6.go`
  - `cmd/remarquee/cmds/rmdoc/render_legacy.go`
  - `cmd/remarquee/cmds/rmdoc/render_v6_png.go`
  - `cmd/remarquee/cmds/rmdoc/vlm_validate.go`
- Applied the same Glazed migration pattern used in the cloud package:
  - `fields` / `schema` / `values`
  - `settings.NewGlazedSection()`
  - `cli.NewCommandSettingsSection()`
  - `WithSections(...)`
  - `DecodeSectionInto(...)`
  - `ShortHelpSections`
- Migrated the legacy rmdoc tests:
  - `cmd/remarquee/cmds/rmdoc/render_v6_test.go`
  - `cmd/remarquee/cmds/rmdoc/render_legacy_test.go`
- Added a new test helper:
  - `cmd/remarquee/cmds/rmdoc/test_values_helpers_test.go`
- The test helper now reuses each command's real default section via `CommandDescription.GetDefaultSection()` and builds `*values.Values` with `values.NewSectionValues(...)`.
- Validated the package with:

```bash
cd /home/manuel/code/wesen/go-go-golems/remarquee && go test ./cmd/remarquee/cmds/rmdoc/...
```

### Why
- `rmdoc` was the next most self-contained subsystem after `cloud`.
- Migrating its tests before OCR reduced risk: it gave a high-confidence signal that the shared command/value pattern was correct.
- Reusing the command description's actual default section in tests avoids a future drift problem where test fixtures no longer match the command schema.

### What worked
- The command files migrated with the same mechanical pattern as `cloud`.
- The test helper worked well and significantly simplified the rewritten tests.
- The package validation succeeded:

```bash
cd /home/manuel/code/wesen/go-go-golems/remarquee && go test ./cmd/remarquee/cmds/rmdoc/...
```

### What didn't work
- The first automated pass missed explicit slice types like:

```go
[]*parameters.ParameterDefinition
```

Those had to be migrated manually to:

```go
[]*fields.Definition
```

- The mechanical pass also required a follow-up search because it updated constructor calls more reliably than explicit type declarations.

### What I learned
- The new `values` test-construction approach is better than the old parsed-layer helpers because it can be driven directly from the real command section definitions.
- Shared helpers like `input_resolver.go` are worth migrating first inside a package because they reduce the number of later manual fixes.

### What was tricky to build
- The test migration was the trickiest part of this slice. The underlying issue was not just renaming helper functions; the old tests were constructing an entire legacy value stack (`ParameterLayer`, `ParsedParameters`, `ParsedLayers`) that no longer exists. I resolved this by switching mental models: instead of rebuilding the old internal structure, I built `values.SectionValues` from the command's own default section. That made the tests less brittle and kept the field definitions in one place.

### What warrants a second pair of eyes
- Review `test_values_helpers_test.go` to confirm the test-fixture construction pattern is acceptable as the standard for future command tests.
- Review `render_v6.go` and `render_legacy.go` to confirm the local/cloud input orchestration behavior stayed untouched.

### What should be done in the future
- Consider extracting the new test helper pattern into a shared test utility if more command tests are added later.

### Code review instructions
- Start with:
  - `cmd/remarquee/cmds/rmdoc/input_resolver.go`
  - `cmd/remarquee/cmds/rmdoc/render_v6.go`
  - `cmd/remarquee/cmds/rmdoc/render_legacy.go`
  - `cmd/remarquee/cmds/rmdoc/test_values_helpers_test.go`
  - `cmd/remarquee/cmds/rmdoc/render_v6_test.go`
  - `cmd/remarquee/cmds/rmdoc/render_legacy_test.go`
- Validate with:

```bash
cd /home/manuel/code/wesen/go-go-golems/remarquee && go test ./cmd/remarquee/cmds/rmdoc/...
```

### Technical details
- The test helper pattern used:

```go
defaultSection, ok := desc.GetDefaultSection()
sectionValues, err := values.NewSectionValues(defaultSection,
    values.WithFieldValue("file", "/tmp/doc.rmdoc"),
    values.WithFieldValue("out", "/tmp/out.pdf"),
)
parsedValues := values.New(values.WithSectionValues(schema.DefaultSlug, sectionValues))
```

## Step 4: Migrate OCR, finish the entrypoint cleanup, and validate the full repo

The last code slice was OCR plus the final repo-wide cleanup. OCR was the only place where both the Glazed and Geppetto integration surfaces changed together, so I rewrote it directly to the modern sections/middlewares/parsed-values APIs rather than trying to salvage the old shape incrementally.

Once OCR was migrated, the final full-repo test run exposed exactly one additional leftover: the renamed logging helper in `cmd/remarquee/main.go`. Fixing that brought the entire repo green on the upgraded dependency set.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Finish the remaining special-case migration work and prove the whole repository still builds and tests.

**Inferred user intent:** The user wants the migration completed end-to-end, including the awkward final cleanup items that only show up under full-repo validation.

**Commit (code):** 1034de5 — "Migrate OCR and CLI entrypoint to facade API"

### What I did
- Rewrote `cmd/remarquee/cmds/ocr/root.go` to use:
  - `fields`, `schema`, `values`
  - `glazedsettings.NewGlazedSection()`
  - `cli.NewCommandSettingsSection()`
  - `geppettosections.CreateGeppettoSections(...)`
  - `geppettosections.WithDefaultsFromInferenceSettings(...)`
  - `factory.NewEngineFromParsedValues(...)`
  - `cli.WithCobraMiddlewaresFunc(geppettosections.GetCobraCommandGeppettoMiddlewares)`
- Preserved OCR defaults:
  - model: `gpt-4o-mini`
  - max response tokens: `4096`
  - temperature: `0.0`
- Replaced the remaining legacy logging helper in `cmd/remarquee/main.go`:
  - `AddLoggingLayerToRootCommand(...)` → `AddLoggingSectionToRootCommand(...)`
- Ran final housekeeping:
  - `go mod tidy`
  - repo-wide legacy symbol grep
  - representative `--help` smoke checks
  - full repo test run

### Why
- OCR was the only command still depending on removed Geppetto and Glazed integration points.
- The entrypoint logging helper rename was a predictable final leftover from the migration playbook.
- The full-repo test run was the only honest signal that the migration was really complete.

### What worked
- The OCR rewrite mapped cleanly onto the current Geppetto examples.
- `go mod tidy` succeeded once the legacy imports were gone.
- The repo-wide legacy symbol grep came back clean.
- Representative `--help` smoke checks succeeded for cloud, rmdoc, and OCR commands.
- Full validation succeeded:

```bash
cd /home/manuel/code/wesen/go-go-golems/remarquee && go test ./...
```

### What didn't work
- The first OCR/full-repo validation attempt failed because of a missing `go.sum` entry after the dependency bump:

```text
missing go.sum entry for module providing package github.com/cespare/xxhash/v2
```

- After `go mod tidy`, the next full-repo run failed on the one remaining renamed helper in `cmd/remarquee/main.go`:

```text
cmd/remarquee/main.go:27:14: undefined: logging.AddLoggingLayerToRootCommand
```

- Fix applied:

```go
logging.AddLoggingSectionToRootCommand(rootCmd, "remarquee")
```

### What I learned
- OCR really was the only place where a direct rewrite was easier than a mechanical token replacement.
- Full-repo validation is essential after an API migration because tiny leftovers like renamed logging helpers may not surface in package-level tests.
- The final repo-wide grep is a valuable complement to `go test ./...` because it proves there are no quiet legacy references left behind.

### What was tricky to build
- The hardest part of the OCR migration was preserving behavior while swapping out two integration layers at once. The symptoms were that the old code depended on both removed Glazed packages and removed Geppetto layer helpers, so a naive mechanical rewrite would have been noisy and hard to trust. I solved that by following the current Geppetto example structure directly: create Geppetto sections, seed defaults with `InferenceSettings`, construct the engine from `parsedValues`, and let the shared Geppetto middleware helper handle config/env/profile parsing.

### What warrants a second pair of eyes
- Review `cmd/remarquee/cmds/ocr/root.go` carefully, especially the decision to switch from the old handwritten middleware chain to `geppettosections.GetCobraCommandGeppettoMiddlewares`.
- Review `go.mod` / `go.sum` one more time for any surprising transitive churn from the dependency upgrade.

### What should be done in the future
- N/A

### Code review instructions
- Start with:
  - `cmd/remarquee/cmds/ocr/root.go`
  - `cmd/remarquee/main.go`
  - `go.mod`
  - `go.sum`
- Validate with:

```bash
cd /home/manuel/code/wesen/go-go-golems/remarquee && go test ./...
```

- Optional smoke checks used during implementation:

```bash
cd /home/manuel/code/wesen/go-go-golems/remarquee
for args in \
  'cloud account --help' \
  'cloud version --help' \
  'cloud get --help' \
  'cloud mkdir --help' \
  'cloud mv --help' \
  'cloud rm --help' \
  'cloud put --help' \
  'cloud search --help' \
  'cloud ls --help' \
  'cloud find --help' \
  'cloud stat --help' \
  'cloud refresh --help' \
  'rmdoc inspect --help' \
  'rmdoc build-background --help' \
  'rmdoc render-v6 --help' \
  'rmdoc render-legacy --help' \
  'rmdoc render-v6-png --help' \
  'rmdoc vlm-validate --help' \
  'ocr --help'
do
  go run ./cmd/remarquee $args >/dev/null || exit 1
done
```

### Technical details
- Final legacy-symbol sweep:

```bash
cd /home/manuel/code/wesen/go-go-golems/remarquee && rg -n "github.com/go-go-golems/glazed/pkg/cmds/layers|github.com/go-go-golems/glazed/pkg/cmds/parameters|github.com/go-go-golems/glazed/pkg/cmds/middlewares|github.com/go-go-golems/geppetto/pkg/layers|glazed\.parameter:|NewGlazedParameterLayers|WithOutputParameterLayerOptions|WithLayersList|InitializeStruct\(|ShortHelpLayers|NewParameterLayer|NewParsedParameters|NewParsedLayers|NewEngineFromParsedLayers|CreateGeppettoLayers|AddLoggingLayerToRootCommand" cmd pkg -g '*.go'
```

- Final repo validation:

```bash
cd /home/manuel/code/wesen/go-go-golems/remarquee && go test ./...
```
