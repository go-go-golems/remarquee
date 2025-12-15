---
Title: Adding a Glazed CLI Command to remarquee
Slug: remarquee-add-glazed-command
Short: Step-by-step guide to add a new (one-file-per-command) Glazed command to the remarquee binary, plus how to add a help entry.
Topics:
- remarquee
- commands
- tutorial
- help-system
Commands:
- remarquee
Flags:
- with-glaze-output
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

remarquee uses **Cobra** for command routing and **Glazed** for consistent parameter parsing and structured output. This tutorial shows the exact workflow to add a new command in the “one file per command” layout, wire it into the binary, and create a help entry that shows up in `remarquee help`.

## Before you start

Glazed’s help system loads Markdown files at runtime from an embedded filesystem. Follow the conventions described in:

```
glaze help writing-help-entries
glaze help how-to-write-good-documentation-pages
```

In particular:

- Use a unique `Slug` (it becomes `remarquee help <slug>`).
- Start each major `##` section with a short paragraph that explains the concept.
- Keep examples copy/paste runnable.

## Mental model: two layers of “help”

remarquee exposes help in two complementary ways:

- Cobra `--help`: usage + flags for a specific command (generated from the Cobra tree and Glazed parameter definitions).
- Glazed help system (`remarquee help ...`): rich Markdown pages embedded in the binary.

You should add both for non-trivial commands: `--help` for immediate usage, and a help page for narrative context and examples.

## Step 1: Create a new command file (one file per command)

Each command lives in its own `.go` file under `cmd/remarquee/cmds/...`.

Example layout (cloud group):

- `cmd/remarquee/cmds/cloud/ls.go`
- `cmd/remarquee/cmds/cloud/get.go`
- `cmd/remarquee/cmds/cloud/put.go`

Pick an existing command file as a template. For rmapi-backed cloud commands, start from:

- `cmd/remarquee/cmds/cloud/refresh.go`

## Step 2: Implement the Glazed command

A Glazed command typically consists of:

- A `Command` struct embedding `*cmds.CommandDescription`
- A settings struct with `glazed.parameter` tags
- One of:
  - `Run(ctx, parsedLayers)` for “human output” mode (`BareCommand`)
  - `RunIntoGlazeProcessor(...)` for structured mode (`GlazeCommand`)
  - both (dual-mode), gated by `--with-glaze-output`

You define flags via `parameters.NewParameterDefinition(...)` and read them via:

- `parsedLayers.InitializeStruct(layers.DefaultSlug, &Settings{})`

This keeps defaults, validation, and help text consistent across output modes.

## Step 3: Build a Cobra command from the Glazed command

Use `cli.BuildCobraCommand` to convert the Glazed command into a Cobra command.

For dual-mode commands, use:

- `cli.WithDualMode(true)`
- `cli.WithGlazeToggleFlag("with-glaze-output")`

This gives you:

- default human output
- structured output when `--with-glaze-output` is set (JSON/YAML/CSV via `--output`)

## Step 4: Wire the command into the remarquee binary

Command registration happens in the Cobra tree:

- Root command: `cmd/remarquee/main.go`
- Command groups: e.g. `cmd/remarquee/cmds/cloud/root.go`

For a new cloud command:

1. Add it to the group’s constructor (e.g., `cloud.NewCloudCommand()`).
2. Keep the group file small; the actual logic belongs in the command file.

## Step 5: Add (or update) a help entry in `remarquee/pkg/doc`

remarquee embeds docs from `pkg/doc/` and registers them once at startup.

- Docs loader: `pkg/doc/doc.go`
- Docs content: `pkg/doc/tutorials/*.md` (this file is one example)

To add a new help page:

1. Create a new Markdown file in `pkg/doc/tutorials/` or `pkg/doc/topics/`
2. Give it a unique `Slug`
3. Write it with short conceptual intros + runnable examples

Then verify:

```bash
go run ./cmd/remarquee help
go run ./cmd/remarquee help remarquee-add-glazed-command
```

## Step 6: Validate locally (the “definition of done” loop)

From the `remarquee/` repo root:

```bash
# Format
gofmt -w cmd/remarquee/cmds/cloud/your-command.go

# Run command help
go run ./cmd/remarquee cloud your-command --help

# If it supports structured output
go run ./cmd/remarquee cloud your-command --with-glaze-output --output json

# Validate help entry
go run ./cmd/remarquee help remarquee-add-glazed-command
```

## Common pitfalls

- **Slug collisions**: two docs with the same `Slug` will collide at load-time.
- **Reading flags from Cobra directly**: prefer `InitializeStruct` so Glazed defaults/validation remain the source of truth.
- **Forgetting to register the command**: the file can compile but the command won’t exist until it’s added to the Cobra tree.


