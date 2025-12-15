---
Title: remarquee cloud Usage Examples
Slug: remarquee-cloud-usage-examples
Short: "Copy/pasteable workflows for browsing, searching, downloading, uploading, organizing, and safely deleting content in the reMarkable cloud."
Topics:
- remarquee
- cloud
- rmapi
- examples
- workflows
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
SectionType: Application
---

# remarquee cloud Usage Examples

This page provides "ready to run" command sequences for common workflows—the kind of tasks you'll do regularly when managing a reMarkable library: browsing, searching, downloading for backup, uploading new content, reorganizing folders, and safely deleting things you no longer need.

The examples are intentionally practical and include a few defensive habits that reduce surprises: refreshing when you're about to script against the tree (so your local view is current), quoting paths with spaces (so your shell doesn't split arguments), preferring structured output when you're feeding results into tools like `jq` (so you don't have to parse text yourself), and treating destructive operations like `rm` as a deliberate, two-step process (preview first, then confirm). These habits aren't strictly required, but they make your scripts more robust and easier to debug when something goes wrong.

If you need a complete command reference, see:

```
remarquee help remarquee-cloud-reference
```

## 1) Sanity check: am I authenticated, and is the tree fresh?

These commands are the fastest way to confirm you’re talking to the right account and have a current view of the cloud tree.

```bash
go run ./cmd/remarquee cloud account
go run ./cmd/remarquee cloud refresh
```

For scripting, use structured output:

```bash
go run ./cmd/remarquee cloud refresh --with-glaze-output --output json
```

## 2) Browse like a filesystem

List top-level folders and then drill down. Use `--compact` for a shell-friendly listing, and use `--time`/`--long` when you’re trying to locate “recently modified” items.

```bash
go run ./cmd/remarquee cloud ls /
go run ./cmd/remarquee cloud ls / --compact
go run ./cmd/remarquee cloud ls /Books --compact --long --time
```

Inspect a single entry:

```bash
go run ./cmd/remarquee cloud stat /Books
go run ./cmd/remarquee cloud stat "/Building BLE Central Applications with ESP-IDF NimBLE on M5Stack Cardputer"
```

## 3) Scriptable discovery: JSON output + filtering

`ls` can emit JSON rows when you pass `--with-glaze-output --output json`. This is useful when you want to build scripts that find a particular folder or document without relying on fragile text parsing. Each row in the JSON array is a complete document/folder record with fields like `id`, `name`, `path`, `is_dir`, `modified_time`, and more. You can pipe this into `jq`, load it into a spreadsheet, or feed it into other tools that speak JSON.

```bash
go run ./cmd/remarquee cloud ls / --with-glaze-output --output json
```

Example: find top-level folders only (requires `jq` installed):

```bash
go run ./cmd/remarquee cloud ls / --with-glaze-output --output json | jq '.[] | select(.is_dir == true) | .path'
```

## 4) Find things across your tree

Use `find` when you want a recursive scan. This is the "search everything under a path" command—it walks the entire tree from a starting point and prints every entry. The optional pattern is a regexp applied to the formatted output path, so it can match directory names, document names, and file-like suffixes you may have in names. This makes `find` a powerful tool for locating content when you remember part of a name but not the full path, or when you want to audit everything in a folder hierarchy.

```bash
# everything under /Books
go run ./cmd/remarquee cloud find /Books --compact

# only paths matching a regex
go run ./cmd/remarquee cloud find /Books "Selfish|Gene"
```

Tip: if you pipe long output into `head`, you may see a `broken pipe` error because `head` closes stdout early. That is normal for many CLI tools; rerun without piping if you want a clean exit.

## 5) Download a document as .rmdoc (backup / processing)

`get` downloads a single document as a `.rmdoc` archive. This is a good “raw” format for local processing that stays aligned with rmapi’s understanding of the document.

```bash
go run ./cmd/remarquee cloud get "/Some Document With Spaces" --out-dir /tmp
ls -la /tmp/*.rmdoc
```

## 6) Create an “Inbox” folder and upload into it

When you start using `put`, it helps to have a dedicated folder for experiments. `mkdir` is non-recursive, so create parent folders first if needed.

```bash
go run ./cmd/remarquee cloud mkdir /Books/Inbox
go run ./cmd/remarquee cloud put ./doc.pdf /Books/Inbox
```

Overwrite semantics:

```bash
# full replace (delete existing, then upload)
go run ./cmd/remarquee cloud put ./doc.pdf /Books/Inbox --force

# replace PDF content only (PDF files only; keeps metadata)
go run ./cmd/remarquee cloud put ./doc.pdf /Books/Inbox --content-only
```

## 7) Organize content: move and rename

`mv` can move entries into a directory or rename to an explicit destination path. The command mirrors rmapi semantics, which makes it predictable if you’ve used rmapi’s shell before.

```bash
# move a doc into a folder (keeping the same name)
go run ./cmd/remarquee cloud mv "/Some Document" /Books

# rename in place
go run ./cmd/remarquee cloud mv "/Books/Old Name" "/Books/New Name"
```

## 8) Safe deletion: dry-run by default, then confirm

`rm` is intentionally safe: without `--yes` it refuses to delete and prints what it *would* delete. This makes it easy to sanity-check path resolution and patterns before doing anything destructive.

```bash
# preview what would be deleted (will exit non-zero)
go run ./cmd/remarquee cloud rm "/Books/Inbox/doc" --non-interactive || true

# actually delete (document)
go run ./cmd/remarquee cloud rm "/Books/Inbox/doc" --yes --non-interactive
```

Recursive folder deletion:

```bash
go run ./cmd/remarquee cloud rm /Books/Inbox --recursive --yes --non-interactive
```

## 9) A safe end-to-end "manual validation" workflow (recommended)

If you want to validate write verbs (`mkdir`, `put`, `mv`, `rm`) without risking real content, use a dedicated sandbox folder and keep the workflow reversible. This is especially useful when you're first learning the commands or testing a new script—by working in a throwaway folder, you can experiment freely and then delete everything when you're done. The pattern below creates a sandbox folder at the root (or you can nest it under any existing folder you control), uploads a small PDF, renames it, and then cleans up with a recursive delete. This entire sequence exercises the main write operations in a safe, repeatable way.

```bash
go run ./cmd/remarquee cloud refresh
go run ./cmd/remarquee cloud mkdir /remarquee-sandbox
go run ./cmd/remarquee cloud put ./doc.pdf /remarquee-sandbox
go run ./cmd/remarquee cloud ls /remarquee-sandbox --compact
go run ./cmd/remarquee cloud mv /remarquee-sandbox/doc "/remarquee-sandbox/doc-renamed"
go run ./cmd/remarquee cloud rm /remarquee-sandbox --recursive --yes
```

If you prefer not to create folders at the root, choose any existing parent directory you control and create the sandbox folder underneath it.


