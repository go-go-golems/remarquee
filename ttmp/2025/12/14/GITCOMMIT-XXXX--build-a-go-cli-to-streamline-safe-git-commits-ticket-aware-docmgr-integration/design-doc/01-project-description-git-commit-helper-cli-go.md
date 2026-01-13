---
Title: 'Project description: Git commit helper CLI (Go)'
Ticket: GITCOMMIT-XXXX
Status: active
Topics:
    - devtools
    - go
    - git
    - productivity
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-14T23:10:53.820040712-05:00
---

# Project description: Git commit helper CLI (Go)

## Executive Summary

This ticket proposes a small Go CLI tool (working name: `gitcommit`) that makes day-to-day git work safer and faster, especially when working with docmgr tickets and “nested” repos like `remarquee/` inside a larger workspace.

The tool’s purpose is to eliminate the two most common workflow costs:

- **Cognitive overhead**: remembering the exact sequence (`git status`, `git diff`, stage only the right files, commit, grab hash, update diary/changelog, commit docs).
- **Footguns**: accidentally staging unrelated files (e.g., docs normalization in another ticket), committing in the wrong git repo, or forgetting to update diary/changelog with the commit hash.

The output should be deterministic and script-friendly, and the tool should be safe-by-default (preview-first, explicit “apply” steps).

## Problem Statement

In this repo, we often:

- Work across multiple adjacent concerns (code + docmgr docs).
- Commit in a sub-repo (`remarquee/`) while the outer workspace might not be a git repo.
- Need to keep ticket docs consistent (tasks checked off, diary steps with commit hashes, changelog entries).

Today, this requires a careful manual sequence and it’s easy to make mistakes:

- **Wrong staging set**: “unrelated” files slip into a commit because they were already staged or because `git add -A` was used.
- **Wrong repo**: you run git commands from the wrong directory and think changes are committed, but they aren’t.
- **Broken workflow contract**: commit happens but diary/changelog aren’t updated (or vice versa).
- **Shell quoting hazards**: backticks in bash do command substitution; logs/commands that contain them must be escaped.

We want a tool that makes the “happy path” one command, and makes the “danger path” obviously hard.

## Proposed Solution

### Scope (v1)

The tool should support:

- **Repository selection**: detect git root and show which repo will be affected.
- **Ticket-aware commit flows**: guide committing code first, then docs, and then update docmgr in between.
- **Staging guardrails**:
  - stage **only** explicitly selected paths by default (no implicit `-A`),
  - highlight “suspicious” staged paths (different ticket directory, binary, temp artifacts),
  - optionally block commit unless `--allow-unrelated` is provided.
- **Docmgr integration** (optional but high value):
  - update diary step with commit hash,
  - update ticket changelog with a one-line entry and file-note links,
  - relate files with `docmgr doc relate` to ticket index or specific docs.

### CLI sketch

The tool should be built with Cobra.

Core commands:

- `gitcommit repo`
  - Print detected git root, current branch, clean/dirty state.
- `gitcommit plan --ticket RMQ-0004`
  - Print:
    - `git status --porcelain`
    - `git diff --stat` (unstaged)
    - `git diff --cached --stat` (staged)
    - proposed staging set (paths)
    - proposed commit messages (code commit, docs commit)
    - proposed docmgr commands (relate/changelog update)
  - **No writes**.
- `gitcommit stage [paths...]`
  - Stage only the provided paths; supports presets like `--ticket RMQ-0004` to stage only ticket dir.
- `gitcommit commit --ticket RMQ-0004 --type code|docs --message ...`
  - Enforces policy:
    - code commit should not include `ttmp/` by default unless `--include-docs`
    - docs commit should only include `ttmp/` by default unless `--include-code`
  - After commit, prints hash and optionally runs docmgr updates.
- `gitcommit docmgr sync --ticket RMQ-0004 --commit <hash>`
  - Runs `docmgr changelog update` and `docmgr doc relate` with provided file notes (or autodetected ones).

### Policy: safe defaults

- Never runs destructive commands without `--yes` / `--apply`.
- Never commits if there are staged files outside the intended scope unless explicitly allowed.
- Always prints the repo root + branch in every output.

### Output format

Text by default, plus optional machine output:

- `--with-glaze-output --output json` (mirroring remarquee patterns) OR a dedicated `--json`.

## Design Decisions

### Decision: explicit staging, no implicit `git add -A`

The biggest failure mode is committing unrelated files. The tool should stage only what the user explicitly lists, or what a ticket preset selects.

### Decision: multi-repo awareness

We regularly work inside `remarquee/` (a git repo) while the outer workspace might not be. `gitcommit` must print and honor the repo root it is operating on.

### Decision: treat docmgr updates as first-class

The tool should make diary/changelog updates easy, because that’s the difference between “we did work” and “we can continue the work”.

### Decision: be careful with shell quoting

When emitting shell commands for copy/paste, the tool must escape backticks and other shell metacharacters so commands aren’t executed unintentionally when pasted into a shell.

## Alternatives Considered

### Use git aliases + shell scripts

- **Pros**: fast to write.
- **Cons**: brittle across repos; hard to make safe-by-default; harder to integrate with docmgr consistently; quoting hazards are easy to miss.

### Use existing tools (e.g., `gh`, `git-town`, `lazygit`)

- **Pros**: mature UIs/workflows.
- **Cons**: they don’t know about our docmgr workflow contract; they won’t enforce ticket scoping.

### Build this inside `remarquee`

- **Pros**: reuse glazed/cobra patterns.
- **Cons**: the tool is broader than reMarkable workflows; keeping it separate makes reuse across repos easier.

## Implementation Plan

1. Create Go module + cobra CLI skeleton (`cmd/gitcommit`)
2. Implement repo detection:
   - current working directory
   - git root discovery
   - branch name, staged/unstaged summaries
3. Implement explicit staging command:
   - stage only passed paths
   - `--ticket` preset stages only `remarquee/ttmp/.../<ticket>/...`
4. Implement commit command:
   - enforce “code commit” vs “docs commit” scoping policy
   - print commit hash
5. Integrate docmgr (opt-in at first):
   - create file notes from staged file list
   - run docmgr relate/changelog update helpers
6. Add tests:
   - staged file classification tests
   - repo root detection tests (using temporary git repos in tests)
7. Write playbooks + usage examples

Non-goals for v1:
- interactive TUI staging (can be added later)
- rewriting history (no rebase/amend helpers initially)

## Open Questions

- Should the tool support multiple tickets in one commit, or treat that as an error by default?
- What is the right place to store “current ticket context” (env var, config file, or CLI flags only)?
- Do we want `gitcommit` to directly edit diary docs (search/replace commit hash), or only run `docmgr` commands?
- Should `gitcommit` integrate with `pre-commit` / `lefthook`?

## References

Internal:
- Cursor “Git Commit Instructions” (the checklist we’re automating)
- Cursor “docmgr workflow” and “diary workflow” docs (the contract we want to enforce)

Related repo examples:
- `remarquee` uses Cobra and Glazed patterns for CLIs (good reference for style)
