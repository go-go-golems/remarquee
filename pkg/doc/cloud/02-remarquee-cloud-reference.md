---
Title: remarquee cloud Reference
Slug: remarquee-cloud-reference
Short: "Command-by-command reference for remarquee cloud (rmapi-backed): arguments, flags, semantics, and what each verb outputs."
Topics:
- remarquee
- cloud
- rmapi
- reference
- cli
Commands:
- remarquee
- remarquee cloud
Flags:
- non-interactive
- reauth
- with-glaze-output
- output
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

# remarquee cloud Reference

This page is a practical reference for the `remarquee cloud` command group. It describes each verb's purpose, arguments, safety properties, and where it intentionally mirrors rmapi behavior so existing rmapi users can transfer expectations. Think of this as the "man page" for cloud commands: it tells you what each verb does, what arguments it accepts, and what the output looks like. If you're looking for narrative explanations or step-by-step workflows, start with the getting started guide instead. For ready-to-run command sequences, see:

```
remarquee help remarquee-cloud-usage-examples
```

## Common flags across cloud commands

Most `remarquee cloud` commands accept the same authentication-related flags because they share rmapi’s bootstrap logic:

- `--non-interactive`: never prompt for the one-time device code; fail if tokens are missing.
- `--reauth`: force re-fetching a user token even if one exists locally.

These flags exist so you can choose between interactive “developer ergonomics” and deterministic “script/CI ergonomics” depending on your environment.

## Output modes and structured data

Some commands support "dual-mode" output using Glazed, which means the same command can produce either human-friendly text or machine-parseable structured data:

- **default mode**: human-oriented text, optimized for reading in a terminal
- **structured mode**: pass `--with-glaze-output` plus `--output json|yaml|csv|...`

Structured output is especially useful when you're building scripts, feeding results into tools like `jq` or spreadsheets, or integrating remarquee into larger workflows. You get consistent JSON/YAML schemas without needing to parse text output yourself.

Currently, the following verbs support structured output:

- `refresh`
- `ls`
- `stat`

Other verbs (`get`, `put`, `mkdir`, `mv`, `rm`, `find`, `account`, `version`) produce text output only, but they do so in a predictable format that's easy to wrap in shell scripts.

Example:

```bash
go run ./cmd/remarquee cloud ls / --with-glaze-output --output json
```

## Command reference

### cloud refresh

Refreshes the remote document tree and prints basic sync state. This is the best first command to validate that authentication and the Sync15 API are working end-to-end.

**Usage:**

```bash
remarquee cloud refresh [--non-interactive] [--reauth]
```

**Structured fields (when `--with-glaze-output`):**
- `user`
- `sync_version`
- `hash`
- `generation`

### cloud ls [path]

Lists entries in a directory or matches patterns similarly to rmapi’s filetree resolver. This command is meant for browsing and for scripts that need to discover IDs/paths.

**Usage:**

```bash
remarquee cloud ls [path] [flags]
```

**Flags:**
- `-c, --compact`: print only names; append `/` to directories
- `-l, --long`: include modified time (only meaningful with `--compact`)
- `-t, --time`: sort by modified time
- `-r, --reverse`: reverse sort order
- `-d, --group-directories`: group directories first
- `-s, --show-templates`: include template entries (rmapi hides these by default)

**Structured fields (when `--with-glaze-output`):**
- `id`, `name`, `type`, `is_dir`, `path`, `parent_id`, `version`, `modified_client`, `modified_time`

### cloud stat <path>

Shows metadata for a single entry (folder or document). This is useful when you need to confirm the resolved path is the one you intended, or when you need an entry’s ID for lower-level operations.

**Usage:**

```bash
remarquee cloud stat <path>
```

**Structured fields (when `--with-glaze-output`):**
- `path`, `name`, `id`, `type`, `is_dir`, `version`, `parent_id`, `modified_client`, `modified_time`

Note: the root entry (`/`) may have empty `modified_client`. In that case, `modified_time` is emitted as `null` in structured output.

### cloud get <remote>

Downloads a document as an `.rmdoc` archive into the current directory or `--out-dir`. This mirrors rmapi’s behavior: `get` is for documents, not directories.

**Usage:**

```bash
remarquee cloud get <remote> [--out-dir <dir>]
```

**Flags:**
- `--out-dir`: output directory (default: `.`)

### cloud put <local> [remote-dir]

Uploads a local file into the remote directory. The target document name is derived from the local path, similar to rmapi. This command can create or overwrite remote content depending on flags, so treat it as a “write” verb and prefer a test folder when experimenting.

**Usage:**

```bash
remarquee cloud put <local> [remote-dir] [flags]
```

**Flags:**
- `--force`: overwrite existing entry by deleting and uploading again
- `--content-only`: PDF only; replace PDF content of an existing document (or create if missing)
- `--coverpage 0|1`: set cover page behavior (mirrors rmapi)

Safety: `--force` and `--content-only` are mutually exclusive.

### cloud mkdir <path>

Creates a remote directory. This is currently non-recursive: the parent directory must already exist.

**Usage:**

```bash
remarquee cloud mkdir <path>
```

### cloud mv <src> <dst>

Moves or renames remote entries and completes sync. It mirrors rmapi behavior:

- If `<dst>` resolves to an existing directory, move `<src>` into that directory (keeping the same name).
- Otherwise, treat `<dst>` as the full destination path (rename/move).

**Usage:**

```bash
remarquee cloud mv <src> <dst>
```

Safety: prevents moving a folder into itself (e.g., `a` -> `a/subdir`).

### cloud rm <target...>

Deletes one or more entries. This is intentionally "safe by default": it will refuse to delete unless you pass `--yes`. Without `--yes`, it prints what it *would* delete and exits with an error, which gives you a chance to inspect the resolved targets before committing to a destructive operation. This design makes `rm` work well in scripts: you can do a "dry run" first (let it fail safely), review the output, and then add `--yes` once you're confident the paths are correct.

**Usage:**

```bash
remarquee cloud rm <target...> [--recursive] --yes
```

**Flags:**
- `-r, --recursive`: delete non-empty folders
- `--yes`: required confirmation

### cloud find [start] [pattern]

Recursively walks the tree from `start` (default: `/`) and prints entries. If `pattern` is provided, it is treated as a regexp and is matched against the formatted output path.

**Usage:**

```bash
remarquee cloud find [start] [pattern] [--compact]
```

### cloud account

Shows basic account information derived from the rmapi user token. This is a quick sanity check for “which account am I authenticated as?”.

**Usage:**

```bash
remarquee cloud account
```

### cloud version

Prints the `rmapi` library version embedded into the build.

**Usage:**

```bash
remarquee cloud version
```


