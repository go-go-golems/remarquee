---
Title: Implement remarquee cloud CLI
Ticket: RMQ-0002
Status: active
Topics:
    - backend
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go.work
      Note: Workspace module list includes ./remarquee
    - Path: remarquee/cmd/remarquee/cmds/status.go
      Note: First command implementation (status)
    - Path: remarquee/cmd/remarquee/main.go
      Note: remarquee root Cobra command + logging init
    - Path: remarquee/go.mod
      Note: remarquee Go submodule definition (module name + replace to local glazed)
    - Path: remarquee/ttmp/2025/12/14/RMQ-0002--implement-remarquee-cloud-cli/reference/01-diary.md
      Note: Implementation diary for RMQ-0002
ExternalSources: []
Summary: Implement remarquee cloud-only CLI (rmapi-backed) using Glazed+Cobra, one file per command; REPL deferred.
LastUpdated: 2025-12-14T19:22:56.210287111-05:00
---


# Implement remarquee cloud CLI

## Overview

This ticket implements the **cloud-only CLI** part of remarquee: `remarquee cloud ...` commands backed by rmapi, built using the repo’s established **Glazed + Cobra** patterns.

**Scope (this ticket):**

- `remarquee` Go submodule setup (`github.com/go-go-golems/remarquee`)
- Cobra root + `cloud` command group
- One file per command (`cmd/remarquee/cmds/cloud/*.go`)
- Implement the initial cloud verbs: `refresh`, `ls`, `stat`, `get`, `put`, `mkdir`, `mv`, `rm`, `find`, `account`, `version`

**Out of scope (explicitly later):**

- REPL / interactive shell
- remarks integration (extract)
- upload pipeline integration
- web UI

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Implementation guide (RMQ-0001)**: `../RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/playbook/01-implementation-guide-remarquee-cli-cloud-module-glazed-cobra-rmapi.md`
- **Product design (RMQ-0001)**: `../RMQ-0001--build-remarquee-tool-unify-rmapi-remarks-upload-stream-ocr/design-doc/01-product-design-remarquee-capability-scope-and-ux-surfaces-cli-repl-web.md`

## Status

Current status: **active**

## Topics

- backend

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
