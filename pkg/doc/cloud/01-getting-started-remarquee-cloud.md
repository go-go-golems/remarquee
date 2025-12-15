---
Title: Getting Started with remarquee cloud
Slug: remarquee-cloud-getting-started
Short: A practical guide to authenticating, refreshing the cloud tree, browsing, and running your first useful cloud workflows with remarquee.
Topics:
- remarquee
- cloud
- rmapi
- getting-started
- authentication
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
SectionType: Tutorial
---

# Getting Started with remarquee cloud

The `remarquee cloud` commands let you interact with your reMarkable account via the same Sync15 protocol that `rmapi` uses. The goal is to make common “cloud filesystem” workflows feel predictable: refresh once, browse and inspect paths, download `.rmdoc` archives for backup/processing, and upload documents back into your preferred folder structure.

If you want a concise command list, see:

```
remarquee help remarquee-cloud-reference
```

If you want copy/pasteable workflows, see:

```
remarquee help remarquee-cloud-usage-examples
```

## Prerequisites

You need a working `remarquee` binary (or `go run` from the repo) and an internet connection. Under the hood, these commands use `rmapi` for authentication and the cloud API, which means authentication is token-based and stored locally (like rmapi).

From the `remarquee/` repo root, you can run everything without installing:

```bash
go run ./cmd/remarquee --help
go run ./cmd/remarquee cloud --help
```

## Authentication: what happens and where tokens live

Authentication is handled by rmapi and uses a two-step token flow: a “device token” derived from a one-time code, and a “user token” derived from that device token. The tokens are written to a local config file so you usually only need to authenticate once.

rmapi determines the config file path in this order:

- If `RMAPI_CONFIG` is set: use that path.
- Else if `~/.rmapi` exists: use `~/.rmapi`.
- Else: use the XDG config directory, typically `~/.config/rmapi/rmapi.conf`.

The file is YAML and includes at least:

- `devicetoken`
- `usertoken`

## Your first successful “cloud round-trip”

The quickest end-to-end validation is `refresh`. It forces the CLI to load tokens (prompting if needed), parse the user token, create an API context, and refresh the remote file tree.

```bash
go run ./cmd/remarquee cloud refresh
```

For structured output (useful for scripting), use:

```bash
go run ./cmd/remarquee cloud refresh --with-glaze-output --output json
```

## Browsing: listing and inspecting entries

Once refreshed, you can browse the top level of your account and then drill down into folders. `ls` shows entries in a directory and can optionally filter/sort similarly to rmapi’s shell.

```bash
# list top-level
go run ./cmd/remarquee cloud ls /

# compact format (adds / to directories)
go run ./cmd/remarquee cloud ls / --compact

# sort by time and show timestamps
go run ./cmd/remarquee cloud ls /Books --compact --long --time
```

To inspect a single entry (folder or document), use `stat`:

```bash
go run ./cmd/remarquee cloud stat /Books
go run ./cmd/remarquee cloud stat "/Some Document With Spaces"
```

`ls` and `stat` also support structured output:

```bash
go run ./cmd/remarquee cloud ls / --with-glaze-output --output json
go run ./cmd/remarquee cloud stat /Books --with-glaze-output --output json
```

## Downloading and uploading

`get` downloads a single document as a `.rmdoc` archive (a container format used by rmapi). This is the safest way to back up content for local processing because it preserves the document data as rmapi understands it.

```bash
go run ./cmd/remarquee cloud get "/Some Document" --out-dir /tmp
```

`put` uploads a local file into a remote directory. It mirrors rmapi semantics, including `--force` and `--content-only` (PDF only). Because `put` can create or overwrite remote content, you should start by uploading into a dedicated test folder you can easily delete later.

```bash
# upload into /Books (or another folder)
go run ./cmd/remarquee cloud put ./doc.pdf /Books
```

To create a folder first (non-recursive; parent must exist):

```bash
go run ./cmd/remarquee cloud mkdir /Books/Inbox
```

## Non-interactive mode (CI / headless machines)

If you want commands to fail instead of prompting for the one-time code (for example in CI), pass `--non-interactive`. This is especially important if you are running scripts where a prompt would appear “stuck”.

```bash
go run ./cmd/remarquee cloud refresh --non-interactive
```

If you need to force re-authentication (refresh user token) even if cached tokens exist, pass `--reauth`.

## Troubleshooting

If `--non-interactive` fails with missing tokens, it means rmapi could not find a usable token file at the resolved config path. Either authenticate once interactively (without `--non-interactive`) or set `RMAPI_CONFIG` to a known config file path that already contains valid tokens.

If a command fails with “file doesn’t exist” but you can see it in the reMarkable UI, make sure you used the correct remote path, and consider running `cloud refresh` first. In general, a refresh is a cheap “stabilizer” step when you’re iterating on scripts.


