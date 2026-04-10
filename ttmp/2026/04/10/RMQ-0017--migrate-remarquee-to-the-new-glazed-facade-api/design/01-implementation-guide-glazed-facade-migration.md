---
Title: Implementation guide for migrating remarquee to the new Glazed facade API
Ticket: RMQ-0017
Status: active
Topics:
    - go
    - cli
    - migration
    - glazed
    - remarquee
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/remarquee/cmds/cloud/account.go
      Note: |-
        Representative simple bare cloud command using old layers/parameters construction
        Representative simple-command migration example
    - Path: cmd/remarquee/cmds/cloud/find.go
      Note: |-
        Representative dual-mode cloud command already partially modernized at the builder level but still using legacy Glazed value types
        Representative dual-mode migration example
    - Path: cmd/remarquee/cmds/cloud/rmapi.go
      Note: |-
        Shared auth settings struct whose tags must migrate from glazed.parameter to glazed
        Shared auth-settings migration seam
    - Path: cmd/remarquee/cmds/ocr/root.go
      Note: |-
        OCR command is the main special case because it integrates Geppetto settings and middleware
        Representative OCR and Geppetto migration seam
    - Path: cmd/remarquee/cmds/rmdoc/render_v6.go
      Note: |-
        Representative rmdoc command with local/cloud orchestration and dual execution helpers
        Representative rmdoc migration seam
    - Path: cmd/remarquee/cmds/rmdoc/render_v6_test.go
      Note: |-
        Representative test file still constructing legacy parsed layers directly
        Representative test migration seam
ExternalSources: []
Summary: Migrate remarquee command construction, settings decoding, test helpers, and OCR/geppetto integration from the removed Glazed layers/parameters API to the current schema/fields/values/sources facade API.
LastUpdated: 2026-04-10T09:30:00-04:00
WhatFor: Intern-facing migration guide and execution order for updating remarquee after the hard Glazed facade cutover.
WhenToUse: Use when implementing RMQ-0017 or reviewing any Glazed migration slice in remarquee.
---


# Implementation guide for migrating remarquee to the new Glazed facade API

## Executive summary

`remarquee` still relies heavily on the removed Glazed `layers` / `parameters` API. A repo scan found:

- **23 affected Go files** in `cmd/` using legacy Glazed command/value APIs
- **75 struct-tag hits** using `glazed.parameter:"..."`
- **23 constructor/configuration hits** using `NewGlazedParameterLayers` or `WithOutputParameterLayerOptions`
- **16 test-helper hits** using `NewParameterLayer`, `NewParsedParameters`, or `NewParsedLayers`

The migration is broad but mechanically consistent for most commands. The only slice that deserves special treatment is `cmd/remarquee/cmds/ocr/root.go`, because it also integrates Geppetto settings, profile/config middleware, and inference-engine construction.

The recommended approach is:

1. migrate the **shared/simple command patterns** first,
2. then migrate the **dual-mode cloud commands** while preserving the current structured-output behavior,
3. then migrate the **rmdoc command family** and its tests,
4. finish with the **OCR + Geppetto** integration,
5. and only then do the final sweep and validation.

## Primary source of truth

The main migration reference is:

- `glazed/pkg/doc/tutorials/migrating-to-facade-packages.md`

Use that document as the rename map and checklist. However, **verify exact current helper names against the current Glazed and Geppetto code**, because some docs/examples evolved further after the initial migration playbook was written.

Two especially relevant current examples:

- `glazed/cmd/examples/new-api-dual-mode/main.go`
- `geppetto/pkg/sections/sections.go`

## Current-state findings in remarquee

### 1. Command files still on legacy Glazed APIs

Affected command files currently cluster into four groups:

#### A. Shared cloud command plumbing

- `cmd/remarquee/cmds/cloud/rmapi.go`

This file holds shared auth settings and is referenced by most cloud commands. Its struct tags should be migrated early because many later slices depend on it.

#### B. Cloud commands using the standard command pattern

- `cmd/remarquee/cmds/cloud/account.go`
- `cmd/remarquee/cmds/cloud/get.go`
- `cmd/remarquee/cmds/cloud/mkdir.go`
- `cmd/remarquee/cmds/cloud/mv.go`
- `cmd/remarquee/cmds/cloud/put.go`
- `cmd/remarquee/cmds/cloud/rm.go`
- `cmd/remarquee/cmds/cloud/search.go`
- `cmd/remarquee/cmds/cloud/version.go`

These mostly use the same legacy patterns:

- `layers` / `parameters` imports
- `settings.NewGlazedParameterLayers()`
- `cli.NewCommandSettingsLayer()`
- `glazecmds.WithLayersList(...)`
- `*layers.ParsedLayers`
- `parsedLayers.InitializeStruct(...)`
- `ShortHelpLayers`

These are the easiest migration slice and should establish the house style for the rest of the repo.

#### C. Dual-mode cloud commands with structured-output defaults

- `cmd/remarquee/cmds/cloud/find.go`
- `cmd/remarquee/cmds/cloud/ls.go`
- `cmd/remarquee/cmds/cloud/stat.go`
- `cmd/remarquee/cmds/cloud/refresh.go`

These already have the newer dual-mode builder pattern (`cli.WithDualMode(true)`, `cli.WithGlazeToggleFlag(...)`) but still use old Glazed schema/value APIs internally.

These files are important because they must preserve:

- current human-readable output,
- current `RunIntoGlazeProcessor` behavior,
- current help text about `--with-glaze-output`,
- and the **default JSON output** behavior introduced for dual-mode structured commands.

#### D. RMDoc + OCR command family

- `cmd/remarquee/cmds/rmdoc/build_background.go`
- `cmd/remarquee/cmds/rmdoc/input_resolver.go`
- `cmd/remarquee/cmds/rmdoc/inspect.go`
- `cmd/remarquee/cmds/rmdoc/render_legacy.go`
- `cmd/remarquee/cmds/rmdoc/render_v6.go`
- `cmd/remarquee/cmds/rmdoc/render_v6_png.go`
- `cmd/remarquee/cmds/rmdoc/vlm_validate.go`
- `cmd/remarquee/cmds/ocr/root.go`

These commands include more orchestration and shared helpers, so they should come after the basic cloud slice.

### 2. Tests still construct parsed layers directly

Affected tests:

- `cmd/remarquee/cmds/rmdoc/render_v6_test.go`
- `cmd/remarquee/cmds/rmdoc/render_legacy_test.go`

These tests currently use:

- `layers.NewParameterLayer(...)`
- `parameters.NewParsedParameters()`
- `layers.NewParsedLayers(...)`

Those must migrate to the new `values` API.

A likely replacement pattern is:

- create a default `schema.Section`
- create `values.NewSectionValues(...)`
- wrap it with `values.New(values.WithSectionValues(...))`
- call the migrated command with `*values.Values`

Use the current Glazed values tests as a reference:

- `glazed/pkg/cmds/values/section-values_test.go`

### 3. OCR is a special case, but it is not blocked

`cmd/remarquee/cmds/ocr/root.go` currently uses old Glazed APIs plus older Geppetto integration points:

- `geppettolayers.CreateGeppettoLayers(...)`
- `factory.NewEngineFromParsedLayers(...)`
- custom parser middleware using legacy `cmdmiddlewares`

The good news is that Geppetto already has modern replacements:

- `geppettosections.CreateGeppettoSections(...)`
- `factory.NewEngineFromParsedValues(...)`
- `geppettosections.GetCobraCommandGeppettoMiddlewares`

That means OCR should be treated as a **special final slice**, not as a blocker requiring upstream work first.

## Desired end state

After this ticket, `remarquee` should no longer depend on removed Glazed layer/parameter APIs in its command surface.

Concretely:

- no command code should import `pkg/cmds/layers`
- no command code should import `pkg/cmds/parameters`
- no command settings should use `glazed.parameter` tags
- no command constructors should use `NewGlazedParameterLayers`
- no command descriptions should use `WithLayersList`
- no command implementations should take `*layers.ParsedLayers`
- no command decoding should call `InitializeStruct`
- no command help config should use `ShortHelpLayers`
- no tests should build legacy parsed-layer objects directly

## Mechanical rename map for remarquee

For this repo, the most common code transformations will be:

| Old | New |
| --- | --- |
| `pkg/cmds/layers` | `pkg/cmds/schema` |
| `pkg/cmds/parameters` | `pkg/cmds/fields` |
| `*layers.ParsedLayers` | `*values.Values` |
| `parsedLayers.InitializeStruct(...)` | `vals.DecodeSectionInto(...)` |
| `settings.NewGlazedParameterLayers(...)` | `settings.NewGlazedSection(...)` |
| `settings.WithOutputParameterLayerOptions(...)` | `settings.WithOutputSectionOptions(...)` |
| `cli.NewCommandSettingsLayer()` | `cli.NewCommandSettingsSection()` |
| `glazecmds.WithLayersList(...)` | `glazecmds.WithSections(...)` |
| `ShortHelpLayers` | `ShortHelpSections` |
| ``glazed.parameter:"foo"`` | ``glazed:"foo"`` |
| `parameters.NewParameterDefinition(...)` | `fields.New(...)` |
| `parameters.ParameterTypeString` | `fields.TypeString` |
| `parameters.WithDefault(...)` | `fields.WithDefault(...)` |
| `parameters.WithHelp(...)` | `fields.WithHelp(...)` |
| `parameters.WithShortFlag(...)` | `fields.WithShortFlag(...)` |
| `parameters.WithIsArgument(true)` | `fields.WithIsArgument(true)` |
| `parameters.WithRequired(true)` | `fields.WithRequired(true)` |

## Important behavioral requirement: preserve current dual-mode output defaults

For `cloud ls`, `cloud find`, `cloud stat`, and `cloud refresh`, do **not** lose the current “JSON by default in glaze mode” behavior.

The old pattern in `remarquee` is:

```go
settings.NewGlazedParameterLayers(
    settings.WithOutputParameterLayerOptions(
        layers.WithDefaults(map[string]interface{}{"output": "json"}),
    ),
)
```

The migrated equivalent should use the current section API:

```go
glazedSection, err := settings.NewGlazedSection(
    settings.WithOutputSectionOptions(
        schema.WithDefaults(map[string]interface{}{"output": "json"}),
    ),
)
```

That behavior needs explicit smoke validation after migration.

## Recommended execution order

### Slice 1: establish the new command pattern on simple cloud commands

Start with a few low-risk files:

- `cmd/remarquee/cmds/cloud/account.go`
- `cmd/remarquee/cmds/cloud/version.go`
- `cmd/remarquee/cmds/cloud/get.go`
- `cmd/remarquee/cmds/cloud/mkdir.go`

Why first:

- no dual-mode output to preserve,
- limited command logic,
- shared auth settings are exercised,
- easy to review,
- creates copyable patterns for later files.

### Slice 2: finish the remaining standard cloud commands

Next migrate:

- `cmd/remarquee/cmds/cloud/mv.go`
- `cmd/remarquee/cmds/cloud/rm.go`
- `cmd/remarquee/cmds/cloud/put.go`
- `cmd/remarquee/cmds/cloud/search.go`
- `cmd/remarquee/cmds/cloud/rmapi.go`

By the end of slice 2, the non-dual cloud surface should be entirely on the new API.

### Slice 3: migrate dual-mode cloud commands

Then migrate:

- `cmd/remarquee/cmds/cloud/ls.go`
- `cmd/remarquee/cmds/cloud/find.go`
- `cmd/remarquee/cmds/cloud/stat.go`
- `cmd/remarquee/cmds/cloud/refresh.go`

Special review points for this slice:

- preserve dual-mode registration
- preserve help text mentioning `--with-glaze-output`
- preserve JSON default output in glaze mode
- preserve current output field names

### Slice 4: migrate rmdoc shared helpers and commands

Start with the shared helper:

- `cmd/remarquee/cmds/rmdoc/input_resolver.go`

Then migrate:

- `cmd/remarquee/cmds/rmdoc/inspect.go`
- `cmd/remarquee/cmds/rmdoc/build_background.go`
- `cmd/remarquee/cmds/rmdoc/render_v6.go`
- `cmd/remarquee/cmds/rmdoc/render_legacy.go`
- `cmd/remarquee/cmds/rmdoc/render_v6_png.go`
- `cmd/remarquee/cmds/rmdoc/vlm_validate.go`

Why this order:

- the resolver settings are reused,
- `inspect` / `build-background` are simpler than the renderers,
- render commands already have tests,
- `vlm_validate` has a large flag surface and should come after the pattern is stable.

### Slice 5: migrate tests

Migrate:

- `cmd/remarquee/cmds/rmdoc/render_v6_test.go`
- `cmd/remarquee/cmds/rmdoc/render_legacy_test.go`

Reference pattern:

- `glazed/pkg/cmds/values/section-values_test.go`

### Slice 6: migrate OCR + Geppetto integration

Finally migrate:

- `cmd/remarquee/cmds/ocr/root.go`

Recommended OCR strategy:

1. replace Glazed imports with `schema`, `fields`, and `values`
2. replace `geppettolayers.CreateGeppettoLayers(...)` with `geppettosections.CreateGeppettoSections(...)`
3. replace `factory.NewEngineFromParsedLayers(...)` with `factory.NewEngineFromParsedValues(...)`
4. replace the legacy custom middleware chain with `geppettosections.GetCobraCommandGeppettoMiddlewares` or the modern equivalent demonstrated in current Geppetto examples
5. keep OCR behavior unchanged while swapping the plumbing

## Review checklist for each migrated file

For every file touched, confirm all of the following:

- imports use `schema`, `fields`, and `values`
- struct tags use `glazed:"..."`
- command settings section uses `cli.NewCommandSettingsSection()`
- command description uses `WithSections(...)`
- help config uses `ShortHelpSections`
- `Run`, `RunIntoWriter`, or `RunIntoGlazeProcessor` now take `*values.Values`
- settings decode through `vals.DecodeSectionInto(...)`
- no old `layers` / `parameters` symbols remain in the file

## Validation plan

### Repo-wide static sweep

Run after each major slice and again at the end:

```bash
cd /home/manuel/code/wesen/go-go-golems/remarquee
rg -n "github.com/go-go-golems/glazed/pkg/cmds/layers|github.com/go-go-golems/glazed/pkg/cmds/parameters|glazed\.parameter:|NewGlazedParameterLayers|WithOutputParameterLayerOptions|WithLayersList|InitializeStruct\(|ShortHelpLayers|NewParameterLayer|NewParsedParameters|NewParsedLayers" cmd pkg -g '*.go'
```

The final result should be empty.

### Build/test validation

```bash
go test ./...
```

### Focused command smoke validation

At minimum, manually smoke:

```bash
# simple cloud commands
remarquee cloud account --help
remarquee cloud version --help
remarquee cloud get --help
remarquee cloud search --help

# dual-mode cloud commands
remarquee cloud ls --help
remarquee cloud ls --with-glaze-output --output json
remarquee cloud find --help
remarquee cloud find --with-glaze-output --output json
remarquee cloud stat --help
remarquee cloud stat --with-glaze-output --output json
remarquee cloud refresh --help
remarquee cloud refresh --with-glaze-output --output json

# rmdoc
remarquee rmdoc inspect --help
remarquee rmdoc render-v6 --help
remarquee rmdoc render-legacy --help
remarquee rmdoc render-v6-png --help
remarquee rmdoc vlm-validate --help

# ocr
remarquee ocr --help
```

## Acceptance criteria

This ticket is done when:

1. `remarquee` command code no longer uses removed Glazed layer/parameter APIs
2. all affected tests pass on the new values API
3. dual-mode cloud commands still default to JSON in glaze mode
4. OCR still constructs its Geppetto engine successfully with the modern parsed-values API
5. repo-wide grep for the legacy command/value symbols is clean
6. ticket docs are updated with any surprises discovered during implementation

## Advice for the implementer

- Do **not** migrate everything in one commit.
- Keep each slice reviewable and runnable.
- Prefer copying a known-good migrated pattern from current Glazed/Geppetto examples over improvising helper names.
- Treat OCR as the last slice even though it is only one file.
- If any third-party helper still expects legacy parsed layers, stop and document the blocker in this ticket before forcing a workaround.
