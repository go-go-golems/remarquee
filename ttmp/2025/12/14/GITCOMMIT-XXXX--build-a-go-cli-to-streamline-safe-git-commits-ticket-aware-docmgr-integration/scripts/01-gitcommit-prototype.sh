#!/usr/bin/env bash
set -euo pipefail

# 01-gitcommit-prototype.sh
#
# Safe-by-default helper for preparing a "nice commit" in repos that follow
# ticket-scoped docmgr workflows.
#
# Goals:
# - Make it hard to accidentally commit unrelated files.
# - Make it obvious which git repo you're operating on.
# - Print a deterministic checklist + suggested next commands.
#
# Notes:
# - This script is intentionally "preview-first": it does not stage or commit unless you ask.
# - Avoids backticks in output and in shell snippets (backticks execute command substitution).

usage() {
  cat <<'EOF'
Usage:
  ./01-gitcommit-prototype.sh --ticket TICKET-ID [--repo ROOT] [--mode code|docs|any]
                              [--stage] [--commit "message"] [--allow-unrelated]

Examples:
  # Preview what would be staged/committed for a ticket
  ./01-gitcommit-prototype.sh --ticket RMQ-0004

  # Stage only files under the ticket directory (preview first, then stage)
  ./01-gitcommit-prototype.sh --ticket RMQ-0004 --stage

  # Commit (still stages ticket only unless --allow-unrelated)
  ./01-gitcommit-prototype.sh --ticket RMQ-0004 --mode docs --stage --commit "RMQ-0004 docs: Step N"

Flags:
  --ticket TICKET-ID       Ticket id (e.g. RMQ-0004)
  --repo PATH              Repo working directory (default: current dir)
  --mode code|docs|any     Commit mode policy (default: any)
  --stage                  Stage ticket files (only)
  --commit MESSAGE         Commit with MESSAGE (requires --stage)
  --allow-unrelated        Allow staged/changed files outside ticket scope
EOF
}

ticket=""
repo="."
mode="any"
do_stage="false"
commit_msg=""
allow_unrelated="false"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --ticket) ticket="${2:-}"; shift 2 ;;
    --repo) repo="${2:-}"; shift 2 ;;
    --mode) mode="${2:-}"; shift 2 ;;
    --stage) do_stage="true"; shift 1 ;;
    --commit) commit_msg="${2:-}"; shift 2 ;;
    --allow-unrelated) allow_unrelated="true"; shift 1 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown arg: $1" >&2; usage; exit 2 ;;
  esac
done

if [[ -z "$ticket" ]]; then
  echo "Error: --ticket is required" >&2
  usage
  exit 2
fi

if [[ "$mode" != "any" && "$mode" != "code" && "$mode" != "docs" ]]; then
  echo "Error: --mode must be one of: any|code|docs" >&2
  exit 2
fi

cd "$repo"

git_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "$git_root" ]]; then
  echo "Error: not inside a git repository (repo=$repo)" >&2
  exit 1
fi

cd "$git_root"

branch="$(git branch --show-current 2>/dev/null || true)"
if [[ -z "$branch" ]]; then
  branch="(detached)"
fi

echo "=== repo ==="
echo "root: $git_root"
echo "branch: $branch"
echo

# Best-effort ticket directory resolution inside this repo (docmgr creates dated folders).
# Example: ttmp/2025/12/14/RMQ-0004--slug...
# NOTE: keep this glob unquoted so the shell expands it.
ticket_dir="$(ls -dt ttmp/*/*/*/"${ticket}"--* 2>/dev/null | head -1 || true)"
if [[ -z "$ticket_dir" ]]; then
  echo "Warning: could not find ticket dir under ttmp/**/${ticket}--*"
  echo "I will still filter by the ticket id substring in paths when possible."
  echo
fi

echo "=== status ==="
git status --porcelain || true
echo

echo "=== unstaged diffstat ==="
git diff --stat || true
echo

echo "=== staged diffstat ==="
git diff --cached --stat || true
echo

changed_paths="$(git status --porcelain | awk '{print $2}' || true)"
if [[ -z "$changed_paths" ]]; then
  echo "No changes detected."
  exit 0
fi

ticket_paths=""
other_paths=""

while IFS= read -r p; do
  [[ -z "$p" ]] && continue
  if [[ -n "$ticket_dir" && "$p" == "$ticket_dir"* ]]; then
    ticket_paths+="$p"$'\n'
    continue
  fi
  if [[ "$p" == *"$ticket"* ]]; then
    ticket_paths+="$p"$'\n'
  else
    other_paths+="$p"$'\n'
  fi
done <<<"$changed_paths"

echo "=== classification ==="
if [[ -n "$ticket_dir" ]]; then
  echo "ticket_dir: $ticket_dir"
else
  echo "ticket_dir: (not found)"
fi
echo

echo "-- ticket-scoped paths --"
printf "%s" "$ticket_paths" | sed '/^$/d' || true
echo

echo "-- other paths (review carefully) --"
printf "%s" "$other_paths" | sed '/^$/d' || true
echo

# Guardrails
if [[ "$allow_unrelated" != "true" && -n "$other_paths" ]]; then
  echo "NOTE: there are changes outside ticket scope."
  echo "      This script will refuse to stage/commit unless you pass --allow-unrelated."
  echo
fi

if [[ "$mode" == "docs" && -n "$other_paths" ]]; then
  # Example policy: docs-only commit should usually touch ttmp/ only.
  # We'll warn (not error) here, but could make it strict later.
  echo "NOTE: mode=docs but non-ticket changes exist."
  echo
fi

# Stage
if [[ "$do_stage" == "true" ]]; then
  if [[ "$allow_unrelated" != "true" && -n "$other_paths" ]]; then
    echo "Refusing to stage due to unrelated changes (use --allow-unrelated to override)."
    exit 3
  fi
  if [[ -n "$ticket_dir" ]]; then
    echo "=== staging ticket dir ==="
    git add -- "$ticket_dir"
  else
    # fallback: stage the ticket-matching paths list
    echo "=== staging ticket-matching paths ==="
    while IFS= read -r p; do
      [[ -z "$p" ]] && continue
      git add -- "$p"
    done <<<"$ticket_paths"
  fi
  echo
  echo "=== staged diffstat (after staging) ==="
  git diff --cached --stat || true
  echo
fi

# Commit
if [[ -n "$commit_msg" ]]; then
  if [[ "$do_stage" != "true" ]]; then
    echo "Error: --commit requires --stage (preview-first policy)" >&2
    exit 2
  fi

  if [[ "$allow_unrelated" != "true" ]]; then
    # Ensure staged files are ticket-scoped.
    staged="$(git diff --cached --name-only || true)"
    bad=""
    while IFS= read -r p; do
      [[ -z "$p" ]] && continue
      if [[ -n "$ticket_dir" && "$p" == "$ticket_dir"* ]]; then
        continue
      fi
      if [[ "$p" == *"$ticket"* ]]; then
        continue
      fi
      bad+="$p"$'\n'
    done <<<"$staged"
    if [[ -n "$bad" ]]; then
      echo "Refusing to commit: staged files outside ticket scope detected:"
      printf "%s" "$bad"
      echo "Use --allow-unrelated to override."
      exit 3
    fi
  fi

  echo "=== commit ==="
  git commit -m "$commit_msg"
  hash="$(git rev-parse HEAD)"
  echo "commit: $hash"
  echo
  echo "=== next suggested commands (templates) ==="
  echo "docmgr changelog update --ticket $ticket --entry \"Step N: ... (commit $hash)\" --file-note \"/abs/path:Reason\""
  echo "docmgr doc relate --ticket $ticket --file-note \"/abs/path:Reason (commit $hash)\""
  echo
fi


