# Tasks

## Completed

- [x] Create ticket `RMQ-0017` for migrating `remarquee` to the new Glazed facade API
- [x] Scan the codebase for legacy Glazed command/value API usage and identify affected files
- [x] Write a migration implementation guide with execution order, risks, and validation steps

## Remaining

### 1. Baseline and execution hygiene

- [x] Create a short implementation diary in `reference/` before the first code change
- [x] Record the exact pre-migration grep baseline in the diary
- [x] Record the exact pre-migration `go test ./...` status in the diary
- [x] Split implementation into reviewable commits by migration slice rather than one large cutover commit

### 2. Shared migration decisions

- [x] Confirm the exact modern helper set to standardize on across the repo (`schema`, `fields`, `values`, `settings.NewGlazedSection`, `cli.NewCommandSettingsSection`, `ShortHelpSections`)
- [x] Confirm the modern Geppetto integration points for OCR (`geppettosections.CreateGeppettoSections`, `factory.NewEngineFromParsedValues`, and the recommended middleware hook)
- [x] Decide whether to keep using `cli.WithParserConfig(...)` everywhere or move opportunistically to the newer convenience options where that improves clarity without changing behavior

### 3. Migrate shared cloud settings and auth primitives

- [x] Migrate `cmd/remarquee/cmds/cloud/rmapi.go`
  - [x] Replace `glazed.parameter` tags with `glazed`
  - [x] Verify all cloud commands still decode auth settings correctly
- [x] Identify any repeated auth flag definitions worth normalizing while touching the cloud files

### 4. Migrate standard cloud commands (simple / non-dual)

- [x] Migrate `cmd/remarquee/cmds/cloud/account.go`
  - [x] Replace legacy imports (`layers`, `parameters`) with `schema`, `fields`, `values`
  - [x] Replace `settings.NewGlazedParameterLayers()` with `settings.NewGlazedSection()`
  - [x] Replace `cli.NewCommandSettingsLayer()` with `cli.NewCommandSettingsSection()`
  - [x] Replace `glazecmds.WithLayersList(...)` with `glazecmds.WithSections(...)`
  - [x] Replace `*layers.ParsedLayers` with `*values.Values`
  - [x] Replace `InitializeStruct(...)` with `DecodeSectionInto(...)`
  - [x] Replace `ShortHelpLayers` with `ShortHelpSections`
- [x] Migrate `cmd/remarquee/cmds/cloud/version.go`
- [x] Migrate `cmd/remarquee/cmds/cloud/get.go`
- [x] Migrate `cmd/remarquee/cmds/cloud/mkdir.go`
- [x] Migrate `cmd/remarquee/cmds/cloud/mv.go`
- [x] Migrate `cmd/remarquee/cmds/cloud/rm.go`
- [x] Migrate `cmd/remarquee/cmds/cloud/put.go`
- [x] Migrate `cmd/remarquee/cmds/cloud/search.go`
- [x] Smoke-check `--help` for each migrated simple cloud command before moving on

### 5. Migrate dual-mode cloud commands while preserving structured-output UX

- [x] Migrate `cmd/remarquee/cmds/cloud/ls.go`
  - [x] Preserve current `BareCommand` behavior
  - [x] Preserve current `GlazeCommand` behavior and field names
  - [x] Replace default glaze output setup with `settings.NewGlazedSection(... settings.WithOutputSectionOptions(schema.WithDefaults(...)))`
  - [x] Preserve the current JSON default in glaze mode
  - [x] Preserve help text mentioning `--with-glaze-output`
- [x] Migrate `cmd/remarquee/cmds/cloud/find.go`
  - [x] Preserve regex behavior and current field names
  - [x] Preserve JSON default in glaze mode
  - [x] Keep the existing dual-mode toggle behavior
- [x] Migrate `cmd/remarquee/cmds/cloud/stat.go`
  - [x] Preserve JSON default in glaze mode
  - [x] Preserve current structured row fields
- [x] Migrate `cmd/remarquee/cmds/cloud/refresh.go`
  - [x] Preserve JSON default in glaze mode
  - [x] Preserve current structured output behavior
- [x] Run command-level smoke checks for all four dual-mode commands in both classic and glaze mode

### 6. Migrate rmdoc shared helper and commands

- [x] Migrate `cmd/remarquee/cmds/rmdoc/input_resolver.go`
  - [x] Replace shared cloud-input struct tags with `glazed`
  - [x] Replace decoding from parsed layers to parsed values
- [x] Migrate `cmd/remarquee/cmds/rmdoc/inspect.go`
- [x] Migrate `cmd/remarquee/cmds/rmdoc/build_background.go`
- [x] Migrate `cmd/remarquee/cmds/rmdoc/render_v6.go`
  - [x] Preserve cloud/local input resolution behavior
  - [x] Preserve output naming and `--force`
  - [x] Preserve glaze output behavior if any structured mode is exposed
- [x] Migrate `cmd/remarquee/cmds/rmdoc/render_legacy.go`
  - [x] Preserve cloud/local input resolution behavior
  - [x] Preserve legacy-specific flags and output behavior
- [x] Migrate `cmd/remarquee/cmds/rmdoc/render_v6_png.go`
- [x] Migrate `cmd/remarquee/cmds/rmdoc/vlm_validate.go`
  - [x] Be careful with the large flag surface and keep defaults identical
- [x] Smoke-check `--help` for all migrated rmdoc commands before touching OCR

### 7. Migrate tests off legacy parsed-layer helpers

- [x] Migrate `cmd/remarquee/cmds/rmdoc/render_v6_test.go`
  - [x] Replace `layers.NewParameterLayer(...)`
  - [x] Replace `parameters.NewParsedParameters()`
  - [x] Replace `layers.NewParsedLayers(...)`
  - [x] Rebuild test fixtures with `values.NewSectionValues(...)` and `values.New(values.WithSectionValues(...))`
- [x] Migrate `cmd/remarquee/cmds/rmdoc/render_legacy_test.go`
  - [x] Replace legacy parsed-layer test helpers with the modern values API
- [x] Re-run the focused rmdoc tests before and after the OCR slice

### 8. Migrate OCR and Geppetto integration

- [x] Migrate `cmd/remarquee/cmds/ocr/root.go`
  - [x] Replace Glazed command/value imports with `schema`, `fields`, and `values`
  - [x] Replace `glazedsettings.NewGlazedParameterLayers()` with `glazedsettings.NewGlazedSection()`
  - [x] Replace `cli.NewCommandSettingsLayer()` with `cli.NewCommandSettingsSection()`
  - [x] Replace `geppettolayers.CreateGeppettoLayers(...)` with `geppettosections.CreateGeppettoSections(...)`
  - [x] Replace `factory.NewEngineFromParsedLayers(...)` with `factory.NewEngineFromParsedValues(...)`
  - [x] Replace the legacy custom middleware chain with the modern Geppetto middleware helper or equivalent sources-based chain
  - [x] Preserve current OCR defaults, prompt behavior, media-type detection, and profile/config behavior
- [x] Run a targeted OCR smoke test at least through command initialization and `--help`

### 9. Validation and cleanup

- [x] Run repo-wide grep to confirm no command code still uses removed Glazed layer/parameter symbols
- [x] Run `go test ./...`
- [x] If any slice required deviations from the original plan, update the implementation guide with the final pattern
- [x] Add final changelog entries summarizing what moved in each migration slice
- [x] Run `docmgr doctor --ticket RMQ-0017 --stale-after 30`

### 10. Delivery

- [x] Relate key migrated files to the ticket docs
- [x] Update `index.md` with final summary and key links
- [x] Add a concise continuation note describing any remaining follow-up work after the migration lands
