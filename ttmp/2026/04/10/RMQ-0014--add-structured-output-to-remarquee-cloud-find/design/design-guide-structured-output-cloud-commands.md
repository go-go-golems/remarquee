---
Title: 'Design Guide: Structured Output for Cloud Commands'
Ticket: ""
Status: ""
Topics:
    - architecture
    - glazed
    - cloud-commands
    - design
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: cmd/remarquee/cmds/cloud/find.go
      Note: Target for structured output enhancement
    - Path: cmd/remarquee/cmds/cloud/ls.go
      Note: Reference implementation with dual-mode
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Design Guide: Structured Output for Cloud Commands

## Context

The `remarquee cloud` subcommand group currently contains several commands that output data to the terminal:

- `cloud ls` - List directory contents
- `cloud find` - Recursive search with optional regex
- `cloud account` - Show account info
- `cloud sync` - Sync status

Some of these (like `ls`) produce human-readable output that's not machine-parseable. Users who want to script around these commands or integrate with other tools need structured output formats (JSON, YAML, CSV).

## Design Decision: Dual-Mode Pattern

We chose the **dual-mode pattern** for adding structured output:

| Property | Value |
|----------|-------|
| Pattern | Implement both `BareCommand` + `GlazeCommand` interfaces |
| Toggle | `--with-glaze-output` flag enables structured output |
| Default | Human-readable text output (backward compatible) |
| Benefit | Existing scripts/workflows continue to work unchanged |

### Alternative Considered: Pure GlazeCommand

A pure `GlazeCommand` approach would:
- ✅ Simpler code (one interface instead of two)
- ❌ Breaking change (default output becomes structured)
- ❌ Different UX from standard CLI tools

### Alternative Considered: Separate Subcommands

Separate `cloud ls` vs `cloud ls --format json`:
- ✅ Clear separation
- ❌ Duplication of logic
- ❌ Inconsistent UX within the command group

## Architecture

### Interface Requirements

```
                    ┌─────────────────────────┐
                    │  CommandDescription     │
                    │  (name, help, flags)    │
                    └───────────┬─────────────┘
                                │
          ┌─────────────────────┴─────────────────────┐
          │                                           │
          ▼                                           ▼
   ┌─────────────┐                           ┌─────────────┐
   │ BareCommand │                           │ GlazeCommand │
   │─────────────│                           │─────────────│
   │ Run()       │                           │ RunIntoGP() │
   └─────────────┘                           └─────────────┘
          │                                           │
          ▼                                           ▼
   Human output                                    Structured
   (fmt.Println)                                   output
                                                  (types.Row → gp)

                    ┌─────────────────────────────┐
                    │        DualMode             │
                    │  (Both interfaces + flag)   │
                    └─────────────────────────────┘
```

### Field Consistency

All cloud commands that output file/directory data should use consistent field names:

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier |
| `name` | string | Display name |
| `type` | string | Document type (DocumentType enum) |
| `is_dir` | bool | True if node is a directory |
| `path` | string | Full path from root |
| `parent_id` | string | Parent node ID (empty for root) |
| `version` | string | Version string |
| `modified_client` | string | Client-side modification timestamp |
| `modified_time` | time.Time | Parsed modification time |

### File Organization

```
cmd/remarquee/cmds/cloud/
├── ls.go      # Reference implementation (complete)
├── find.go    # Target for this enhancement
├── account.go
└── sync.go
```

## Testing Strategy

### Unit Tests

For each command implementing dual-mode:

```go
func TestFindCommand_GlazeMode(t *testing.T) {
    cmd, _ := NewFindCommand()
    
    // Verify interface compliance
    var _ glazecmds.GlazeCommand = cmd
    
    // Test RunIntoGlazeProcessor produces rows
    gp := &mockGlazeProcessor{}
    cmd.RunIntoGlazeProcessor(ctx, parsedLayers, gp)
    
    assert.Greater(t, len(gp.Rows), 0)
}
```

### Integration Tests

```bash
# Classic mode (default)
remarquee cloud find /Books
remarquee cloud find /Books -c

# Glaze mode
remarquee cloud find /Books --with-glaze-output --output json
remarquee cloud find /Books --with-glaze-output --output yaml
remarquee cloud find /Books --with-glaze-output --output csv
```

## Rollout Plan

1. **Phase 1**: Implement `cloud find` structured output (this ticket)
2. **Phase 2**: Audit other cloud commands for structured output need
3. **Phase 3**: Document pattern in codebase for future commands

## Related Documentation

- Tutorial: `glazed/pkg/doc/tutorials/05-build-first-command.md`
- Reference: `cmd/remarquee/cmds/cloud/ls.go`
- This guide: `design/design-guide-structured-output-cloud-commands.md`