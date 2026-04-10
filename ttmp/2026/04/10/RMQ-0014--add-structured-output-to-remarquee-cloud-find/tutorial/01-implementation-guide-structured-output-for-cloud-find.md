---
Title: 'Implementation Guide: Add Structured Output to `remarquee cloud find`'
Ticket: ""
Status: ""
Topics:
    - glazed
    - cli
    - structured-output
    - cloud-commands
    - go
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: cmd/remarquee/cmds/cloud/find.go
      Note: Target file to modify
    - Path: cmd/remarquee/cmds/cloud/ls.go
      Note: Reference implementation to follow
    - Path: glazed/pkg/doc/tutorials/05-build-first-command.md
      Note: Glazed tutorial with dual-mode pattern
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Implementation Guide: Add Structured Output to `remarquee cloud find`

This guide walks through implementing structured output (JSON, YAML, CSV, table) for the `remarquee cloud find` command, following the same pattern already implemented in `remarquee cloud ls`.

**Learning objectives:**
- Understand the dual-mode command pattern in Glazed
- Add `GlazeCommand` interface to an existing command
- Implement `RunIntoGlazeProcessor` for structured data output
- Configure dual mode with toggle flag for `--with-glaze-output`

---

## Background: Why This Matters

The `remarquee cloud find` command currently only produces human-readable text output:

```
[d] /Books
[d] /Books/Python Notes
[f] /Books/Python Notes/session-2025-11-05.pdf
```

While this is useful for interactive use, it's not scriptable. By implementing the `GlazeCommand` interface, we enable:

- `remarquee cloud find --output json` → Machine-parseable JSON
- `remarquee cloud find --output yaml` → YAML format
- `remarquee cloud find --output csv` → CSV for spreadsheets
- `remarquee cloud find --fields name,type,path` → Select specific columns
- `remarquee cloud find --sort-columns modified_time` → Sort by any field

This mirrors the pattern already established in `cloud ls`, so we have a reference implementation to follow.

---

## Step 1: Understand the Current State

Examine the current `find.go` implementation:

```go
// cmd/remarquee/cmds/cloud/find.go

package cloud

type FindCommand struct {
    *glazecmds.CommandDescription
}

type FindSettings struct {
    AuthSettings
    Compact bool `glazed.parameter:"compact"`
    Start   string `glazed.parameter:"start"`
    Pattern string `glazed.parameter:"pattern"`
}

var _ glazecmds.BareCommand = &FindCommand{}  // Only implements BareCommand!
```

**Key observation:** `FindCommand` currently only implements `BareCommand` (the simple text-output interface). It needs to also implement `GlazeCommand` to support structured output.

Compare this to `ls.go`:

```go
var _ glazecmds.BareCommand = &LsCommand{}
var _ glazecmds.GlazeCommand = &LsCommand{}  // Also implements GlazeCommand!
```

---

## Step 2: What to Add

You need to make the following changes to `find.go`:

### 2.1 Add the interface compliance check

After the existing `var _ glazecmds.BareCommand = &FindCommand{}`, add:

```go
var _ glazecmds.GlazeCommand = &FindCommand{}
```

### 2.2 Add imports for structured output types

Ensure these imports are present:

```go
import (
    // ... existing imports ...
    
    "github.com/go-go-golems/glazed/pkg/middlewares"
    "github.com/go-go-golems/glazed/pkg/types"
    
    // ... remaining imports ...
)
```

### 2.3 Implement `RunIntoGlazeProcessor`

Add this method to `FindCommand`:

```go
func (c *FindCommand) RunIntoGlazeProcessor(ctx context.Context, parsedLayers *layers.ParsedLayers, gp middlewares.Processor) error {
    s := &FindSettings{}
    if err := parsedLayers.InitializeStruct(layers.DefaultSlug, s); err != nil {
        return err
    }

    _, apiCtx, err := createApiCtx(s.AuthSettings)
    if err != nil {
        return err
    }

    startNode, err := apiCtx.Filetree().NodeByPath(s.Start, nil)
    if err != nil {
        return errors.New("start directory doesn't exist")
    }

    var matchRegexp *regexp.Regexp
    if s.Pattern != "" {
        matchRegexp, err = regexp.Compile(s.Pattern)
        if err != nil {
            return errors.New("failed to compile regexp")
        }
    }

    filetree.WalkTree(startNode, filetree.FileTreeVistor{
        Visit: func(node *model.Node, _ []string) bool {
            p := buildPathFromParents(node)
            modTime, _ := node.LastModified()

            row := types.NewRow(
                types.MRP("id", node.Id()),
                types.MRP("name", node.Name()),
                types.MRP("type", node.Document.Type),
                types.MRP("is_dir", node.IsDirectory()),
                types.MRP("path", p),
                types.MRP("parent_id", node.Document.Parent),
                types.MRP("version", node.Version()),
                types.MRP("modified_client", node.Document.ModifiedClient),
                types.MRP("modified_time", modTime),
            )

            // Apply pattern filter
            if matchRegexp == nil || matchRegexp.MatchString(p) {
                if err := gp.AddRow(ctx, row); err != nil {
                    return err
                }
            }
            return false
        },
    })

    return nil
}
```

### 2.4 Update `NewFindCobraCommand`

Change the Cobra command builder to enable dual mode:

```go
func NewFindCobraCommand() (*cobra.Command, error) {
    cmd, err := NewFindCommand()
    if err != nil {
        return nil, err
    }

    return cli.BuildCobraCommand(cmd,
        cli.WithDualMode(true),                          // Add this!
        cli.WithGlazeToggleFlag("with-glaze-output"),   // Add this!
        cli.WithParserConfig(cli.CobraParserConfig{
            ShortHelpLayers: []string{layers.DefaultSlug},
            MiddlewaresFunc: cli.CobraCommandDefaultMiddlewares,
        }),
    )
}
```

---

## Step 3: Compare with `ls.go` Implementation

The `cloud ls` command is your reference. Here's the key differences:

| Aspect | `ls.go` | `find.go` (current) |
|--------|---------|---------------------|
| Interface compliance | Both `BareCommand` + `GlazeCommand` | Only `BareCommand` |
| Dual mode | Enabled with toggle flag | Not enabled |
| `RunIntoGlazeProcessor` | Implemented | Not implemented |

The structure of your `RunIntoGlazeProcessor` should mirror `ls.go`:

```
1. Parse settings from parsedLayers into FindSettings struct
2. Create API context (same as Run method)
3. Get nodes or walk tree
4. For each item, create a types.Row with consistent field names
5. Add row to gp (GlazeProcessor)
```

---

## Step 4: Field Names Consistency

When implementing `RunIntoGlazeProcessor`, use the same field names as `ls.go` for consistency:

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique node ID |
| `name` | string | Node name |
| `type` | string | Document type (DocumentType) |
| `is_dir` | bool | True if directory |
| `path` | string | Full path from root |
| `parent_id` | string | Parent node ID |
| `version` | string | Version string |
| `modified_client` | string | Client modification timestamp |
| `modified_time` | time.Time | Parsed modification time |

---

## Step 5: Testing Your Implementation

After making the changes, test in both modes:

### 5.1 Classic mode (default, unchanged behavior)

```bash
remarquee cloud find /Books
remarquee cloud find /Books -c  # compact mode
remarquee cloud find /Books ".*\.pdf$"  # pattern filter
```

### 5.2 Glaze mode (structured output)

```bash
remarquee cloud find /Books --with-glaze-output
remarquee cloud find /Books --with-glaze-output --output json
remarquee cloud find /Books --with-glaze-output --output yaml
remarquee cloud find /Books --with-glaze-output --output csv
remarquee cloud find /Books --with-glaze-output --fields name,type,path
remarquee cloud find /Books --with-glaze-output --sort-columns modified_time
```

### 5.3 Debug features

```bash
remarquee cloud find /Books --print-parsed-fields
remarquee cloud find /Books --with-glaze-output --print-schema
remarquee cloud find /Books --with-glaze-output --output json --print-yaml
```

---

## Step 6: Build and Verify

```bash
# Build
go build -o ./bin/remarquee ./cmd/remarquee

# Run tests
go test ./...

# Test manually
./bin/remarquee cloud find --help
```

Expected `--help` output should now show the `--with-glaze-output` flag.

---

## Complete Diff (Summary)

Here's what you need to add to `find.go`:

```diff
 import (
     "context"
     "fmt"
     "regexp"

     "github.com/go-go-golems/glazed/pkg/cli"
     glazecmds "github.com/go-go-golems/glazed/pkg/cmds"
     "github.com/go-go-golems/glazed/pkg/cmds/layers"
     "github.com/go-go-golems/glazed/pkg/cmds/parameters"
+    "github.com/go-go-golems/glazed/pkg/middlewares"
     "github.com/go-go-golems/glazed/pkg/settings"
+    "github.com/go-go-golems/glazed/pkg/types"
     "github.com/juruen/rmapi/filetree"
     "github.com/juruen/rmapi/model"
     "github.com/pkg/errors"
     "github.com/spf13/cobra"
 )

 var _ glazecmds.BareCommand = &FindCommand{}
+var _ glazecmds.GlazeCommand = &FindCommand{}

+func (c *FindCommand) RunIntoGlazeProcessor(ctx context.Context, parsedLayers *layers.ParsedLayers, gp middlewares.Processor) error {
+    // ... implementation ...
+}

 func NewFindCobraCommand() (*cobra.Command, error) {
     cmd, err := NewFindCommand()
     if err != nil {
         return nil, err
     }

     return cli.BuildCobraCommand(cmd,
+        cli.WithDualMode(true),
+        cli.WithGlazeToggleFlag("with-glaze-output"),
         cli.WithParserConfig(cli.CobraParserConfig{
             ShortHelpLayers: []string{layers.DefaultSlug},
             MiddlewaresFunc: cli.CobraCommandDefaultMiddlewares,
         }),
     )
 }
```

---

## Troubleshooting

### "undefined: glazecmds.GlazeCommand"

Make sure you have the interface compliance check `var _ glazecmds.GlazeCommand = &FindCommand{}` after your struct definition.

### "cannot use gp (variable of type middlewares.GlazeProcessor) as middlewares.Processor"

The `RunIntoGlazeProcessor` interface expects `middlewares.Processor`. Both types should be compatible. If you get this error, verify your import is correct and the interface signature matches exactly.

### Dual mode not working

Check that you added both `cli.WithDualMode(true)` and `cli.WithGlazeToggleFlag("with-glaze-output")`. Both are required.

### Rows not appearing in output

Make sure you're calling `gp.AddRow(ctx, row)` inside the walk callback, and that your pattern filter is not blocking all results.

---

## Reference Files

- Reference implementation: `cmd/remarquee/cmds/cloud/ls.go`
- Glazed tutorial: `glazed/pkg/doc/tutorials/05-build-first-command.md`
- This guide: `design/implementation-guide.md`