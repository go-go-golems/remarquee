#!/usr/bin/env bash
#
# sync_obsidian_to_remarkable.sh
#
# One-way sync: upload new Obsidian vault project notes to reMarkable as PDFs.
# Skips files that already exist remotely (by document name, not content hash).
#
# Usage:
#   ./sync_obsidian_to_remarkable.sh <local-dir> <remote-dir>
#
# Example:
#   ./sync_obsidian_to_remarkable.sh \
#     /home/manuel/code/wesen/obsidian-vault/Projects/2026/04 \
#     /ai/2026/05/03/obsidian-projects

set -euo pipefail

LOCAL_DIR="${1:-}"
REMOTE_DIR="${2:-/ai/$(date +%Y/%m/%d)/obsidian-sync}"
REMARKEE="${REMARKEE:-remarquee}"

if [[ -z "$LOCAL_DIR" ]]; then
    echo "Usage: $0 <local-dir> [remote-dir]"
    echo ""
    echo "Environment variables:"
    echo "  REMARKEE   - path to remarquee binary (default: remarquee)"
    exit 1
fi

if [[ ! -d "$LOCAL_DIR" ]]; then
    echo "ERROR: local directory does not exist: $LOCAL_DIR"
    exit 1
fi

LOCAL_DIR=$(cd "$LOCAL_DIR" && pwd)

echo "=== Obsidian-to-reMarkable Sync ==="
echo "Local source : $LOCAL_DIR"
echo "Remote target: $REMOTE_DIR"
echo ""

# -----------------------------------------------------------------------------
# Step 1: Build a set of existing remote document names (leaf names only).
# -----------------------------------------------------------------------------
echo "Fetching remote file list from $REMOTE_DIR ..."

REMOTE_NAMES=$(mktemp)
trap "rm -f $REMOTE_NAMES" EXIT

if ! "$REMARKEE" cloud find "$REMOTE_DIR" --with-glaze-output --output json --non-interactive 2>/dev/null | \
     jq -r '.[] | select(.is_dir == false) | .name' > "$REMOTE_NAMES"; then
    echo "WARNING: failed to list remote directory (it may not exist yet). Proceeding with empty list."
    > "$REMOTE_NAMES"
fi

# Sort for fast lookup with comm(1)
sort -o "$REMOTE_NAMES" "$REMOTE_NAMES"

# -----------------------------------------------------------------------------
# Step 2: Find all local .md files and filter out those already on device.
# -----------------------------------------------------------------------------
echo "Scanning local markdown files ..."

LOCAL_FILES=$(mktemp)
trap "rm -f $REMOTE_NAMES $LOCAL_FILES" EXIT

find "$LOCAL_DIR" -type f -name '*.md' -print0 | \
    while IFS= read -r -d '' f; do
        # Compute the PDF name that remarquee would use.
        # remarquee strips .md and appends .pdf.
        basename "$f" .md
    done | sort > "$LOCAL_FILES"

# Files that are local but NOT remote -> candidates for upload
TO_UPLOAD=$(mktemp)
trap "rm -f $REMOTE_NAMES $LOCAL_FILES $TO_UPLOAD" EXIT

comm -23 "$LOCAL_FILES" "$REMOTE_NAMES" > "$TO_UPLOAD"

UPLOAD_COUNT=$(wc -l < "$TO_UPLOAD" | tr -d ' ')

if [[ "$UPLOAD_COUNT" -eq 0 ]]; then
    echo ""
    echo "All local files already exist remotely. Nothing to do."
    exit 0
fi

echo ""
echo "Found $UPLOAD_COUNT new file(s) to upload:"
cat "$TO_UPLOAD" | sed 's/^/  - /'
echo ""

# -----------------------------------------------------------------------------
# Step 3: Upload each missing file individually (preserving directory structure).
# -----------------------------------------------------------------------------
# We rebuild the full path from the basename by searching the local tree again.
# This is naive but robust for typical vault layouts.

DRY_RUN="${DRY_RUN:-}"

while IFS= read -r pdf_basename; do
    # pdf_basename is "Some File Name" (no .pdf, no .md)
    md_basename="${pdf_basename}.md"

    # Find the actual local file
    local_file=$(find "$LOCAL_DIR" -type f -name "$md_basename" -print | head -n1)

    if [[ -z "$local_file" ]]; then
        echo "WARNING: could not resolve local file for '$md_basename' — skipping"
        continue
    fi

    if [[ -n "$DRY_RUN" ]]; then
        echo "DRY-RUN: would upload $local_file -> $REMOTE_DIR"
    else
        echo "Uploading: $local_file"
        "$REMARKEE" upload md \
            --non-interactive \
            --remote-dir "$REMOTE_DIR" \
            --preserve-dirs \
            "$local_file"
    fi
done < "$TO_UPLOAD"

echo ""
echo "=== Sync complete ==="
