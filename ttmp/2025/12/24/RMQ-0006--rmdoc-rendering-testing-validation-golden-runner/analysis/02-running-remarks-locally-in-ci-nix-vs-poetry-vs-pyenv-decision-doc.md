---
Title: 'Running remarks locally + in CI: nix vs poetry vs pyenv (decision doc)'
Ticket: RMQ-0006
Status: active
Topics:
    - go
    - remarkable
    - testing
    - validation
    - rmdoc
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../remarks/flake.nix
      Note: Nix flake defines remarks app + devShell; key for CI and local
    - Path: ../../../../../../../remarks/poetry.toml
      Note: In-project venv settings
    - Path: ../../../../../../../remarks/pyproject.toml
      Note: Poetry project defines entrypoints + Python 3.12 requirement + git deps
    - Path: ../../../../../../../remarks/remarks/__main__.py
      Note: CLI contract (remarks INPUT OUTPUT [--log_level])
    - Path: cmd/remarquee/cmds/rmdoc/vlm_validate.go
      Note: VLM helper that needs reference PDF; blocked by remarks availability
    - Path: pkg/refimpl/remarks/runner.go
      Note: Current Go wrapper assumes remarks is on PATH (impacts choice)
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-24T15:29:02.445665965-05:00
WhatFor: ""
WhenToUse: ""
---


# Running `remarks` locally + in CI (decision doc)

## Goal

We want a **reliable way to run `remarks`** as the reference implementation for RMQ-0006 golden tests / VLM validation comparisons.

Today, `pinocchio` exists on PATH, but `remarks` does **not** (`command not found: remarks`). This document enumerates the viable ways to get `remarks` running in this repo and how each option impacts:

- local developer workflow
- CI reproducibility
- speed + debugging ergonomics
- how we wire it into the Go wrapper (`remarquee/pkg/refimpl/remarks`)

## Facts from the `remarks/` repo (what we actually have)

### Nix flake exists

`remarks/flake.nix` defines:
- `packages.default` and `packages.remarks` = a Poetry2nix-built Python application
- `apps.default.program` = `${remarksBin}/bin/remarks` (so `nix run` works)
- a dev shell that includes: Python 3.12 env, `poetry`, `poppler_utils`, etc.

Key lines:

```1:88:remarks/flake.nix
packages = {
  default = remarksBin;
  remarks = remarksBin;
};
apps.default = {
  type = "app";
  program = "${remarksBin}/bin/remarks";
};
devShells.default = environment;
```

### Poetry project with Python 3.12 requirement

`remarks/pyproject.toml`:
- requires Python `^3.12`
- exposes `remarks` and `remarks-server` entrypoints
- depends on **git dependencies** for `rmscene` and `rmc` (so you need git + network access unless cached)

```1:36:remarks/pyproject.toml
[tool.poetry.dependencies]
python = "^3.12"
PyMuPDF = "^1.23.5"
rmscene = { git = "https://github.com/scrybbling-together/rmscene.git", branch = "main" }
rmc = { git = "https://github.com/scrybbling-together/rmc.git", branch = "main" }

[tool.poetry.scripts]
remarks = 'remarks.__main__:main'
remarks-server = "remarks.server:main_prod"
```

### Poetry is configured to create an in-project venv

`remarks/poetry.toml`:

```1:3:remarks/poetry.toml
[virtualenvs]
create = true
in-project = true
```

Meaning `poetry install` will likely create `remarks/.venv/`.

## How `remarks` is invoked (CLI contract)

The CLI is `remarks INPUT OUTPUT_DIR [--log_level ...]`:

```11:63:remarks/remarks/__main__.py
def main():
    ...
    parser.add_argument("input_dir", metavar="INPUT_DIRECTORY")
    parser.add_argument("output_dir", metavar="OUTPUT_DIRECTORY")
    parser.add_argument("--log_level", default="INFO", metavar="LOG_LEVEL")
    ...
    run_remarks(input_dir, output_dir)
```

Our Go wrapper (`remarquee/pkg/refimpl/remarks`) assumes the executable is called `remarks` by default (but can be overridden by `Runner.Bin`).

## Options matrix (local + CI)

### Option 1 — Nix flake (recommended if you already use Nix)

#### What it looks like

From the `remarks/` directory:

```bash
cd /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarks
nix develop
remarks --help
remarks /abs/path/to/Test.rmdoc /tmp/remarks-out --log_level ERROR
```

Or without entering a shell:

```bash
cd /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarks
nix run .# -- /abs/path/to/Test.rmdoc /tmp/remarks-out --log_level ERROR
```

#### Why it works well here

- **Reproducible**: pinned `nixpkgs` + `poetry2nix` in `flake.nix`
- **Python 3.12 guaranteed** (flake uses `pkgs.python312`)
- Avoids “pyenv vs system python” drift
- Includes extra tools in devShell (`poppler_utils`, `poetry`, etc.)

#### Risks / caveats

- Requires Nix installed and enabled in CI runners
- Git deps (`rmc`, `rmscene`) still need network unless Nix cache covers them
- Some devs dislike Nix or don’t have it installed

#### How to integrate with `remarquee/pkg/refimpl/remarks`

Two patterns:

1) **Put `remarks` on PATH** by entering `nix develop` in `remarks/` before running tests.
   - simplest; no code changes

2) **Call via nix run** (requires small wrapper changes):
   - set runner to execute `nix` and pass `run ...` args
   - today `Runner` assumes `remarks` is the executable; it doesn’t natively support `nix run ... -- ...`
   - we can either:
     - add a `Runner.Mode` (direct vs nix-run), or
     - keep `Runner` and add a separate `NixRunner`

#### CI story

- Add a job step: `nix develop` / `nix run` and run golden tests
- Or pre-build `remarks` as a nix package and use cache

### Option 2 — Poetry (works if you prefer Python-native tooling)

#### What it looks like

From `remarks/`:

```bash
cd /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarks
poetry install
poetry run remarks --help
poetry run remarks /abs/path/to/Test.rmdoc /tmp/remarks-out --log_level ERROR
```

#### Why it’s attractive

- Familiar workflow for Python devs
- Uses project-locked dependencies (`poetry.lock`)
- `.venv` is created locally inside repo (easy to find / activate)

#### Risks / caveats (important)

- Must have **Python 3.12** available (pyenv helps)
- Has **git dependencies** (`rmc`, `rmscene`), so CI needs `git` and network
- PyMuPDF sometimes has native wheels issues on older glibc / missing libs (usually ok on modern Linux)

#### Integrating with `remarquee/pkg/refimpl/remarks`

You don’t necessarily need a global `remarks` on PATH:

- **Local tests**: run `poetry run remarks ...` directly from the `remarks/` dir
- Our Go wrapper currently can’t call `poetry run remarks` without enhancement.
  - Similar to nix-run: we’d need the runner to support “prefix commands”

If we want zero code changes:

- Install via `poetry install`
- Create a shim on PATH:
  - `ln -s /abs/path/to/remarks/.venv/bin/remarks ~/bin/remarks` (manual)
  - Or activate the venv before running tests

#### CI story

- Install Python 3.12 (pyenv or apt or actions/setup-python)
- `poetry install`
- Add venv bin to PATH

### Option 3 — pyenv + pip (no Poetry, manual dependency management)

#### What it looks like

You said you have pyenv. The “manual” option is:

```bash
cd /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarks
pyenv install 3.12.x   # if needed
pyenv local 3.12.x
python -m venv .venv
source .venv/bin/activate
pip install -U pip
pip install -e .
remarks --help
```

#### Why it’s viable

- Doesn’t require Poetry (some people prefer plain pip)
- Still uses pyenv to satisfy Python 3.12
- `pip install -e .` creates an entrypoint `remarks` in `.venv/bin/`

#### Risks / caveats (biggest downside)

- You lose the lockfile enforcement unless you strictly reproduce Poetry’s resolved set
- Git deps still apply
- More “it works on my machine” drift

#### CI story

Harder to make reproducible unless you still rely on `poetry.lock` exported to requirements.

### Option 4 — pipx (isolated CLI install)

#### What it looks like

```bash
pipx install /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarks
remarks --help
```

#### Pros

- Installs a global-ish CLI without polluting system python
- Nice for “I just need the tool”

#### Cons

- Reproducibility depends on how pipx resolves deps (again: lockfile drift)
- Git deps + Python 3.12 still apply

### Option 5 — uv (fast pip/venv toolchain)

If you’re open to new tools, `uv` can replace pip/venv and be much faster.

#### Pros

- Speed, caches wheels aggressively

#### Cons

- Another tool to standardize across devs/CI
- Still doesn’t naturally consume Poetry’s lock unless we add an export step

## Recommended decision (pragmatic)

### For CI stability: Nix

Because `remarks/flake.nix` already defines a first-class app (`nix run .#`), this is the cleanest route to a consistent CI baseline.

### For local dev: Poetry or Nix

- If you already use Nix: `nix develop` is lowest friction.
- If you prefer Python tooling and already have pyenv: `poetry install` is fine.

## What I need from you (decision points)

1) Do you want to **standardize on Nix for CI**? (Yes/No)
2) For local dev, do you prefer:
   - Nix shell (`nix develop`)
   - Poetry (`poetry install`)
   - pyenv + pip
3) Do we want to extend the Go runner to support non-`remarks` binaries?
   - Example: `poetry run remarks ...` or `nix run .# -- ...`

## Follow-up implementation ideas (if you choose Poetry or Nix-run)

If you choose a workflow where `remarks` is *not* directly on PATH, we should enhance `remarquee/pkg/refimpl/remarks.Runner`:

- Add `Runner.Prefix []string` so it can run:
  - `["poetry", "run"] + ["remarks", ...]`
  - `["nix", "run", "/abs/path/to/remarks#"] + ["--", ...]`
- Or add a dedicated runner type per toolchain (`PoetryRunner`, `NixRunner`)

This makes golden tests self-contained and prevents surprises like we saw (“remarks not found”).

