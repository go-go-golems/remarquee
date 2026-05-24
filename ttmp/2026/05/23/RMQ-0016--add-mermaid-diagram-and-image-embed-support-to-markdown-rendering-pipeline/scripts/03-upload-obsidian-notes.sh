#!/bin/bash
# Re-render Obsidian vault notes from today and upload to reMarkable.
# These files contain both local image references and Mermaid code blocks.

set -euo pipefail

VAULT_DIR="$HOME/code/wesen/go-go-golems/go-go-parc/Projects/2026/05/23"
REMARQUEE=${REMARQUEE:-remarquee}
REMOTE_BASE="/ai/2026/05/23/obsidian-individual"
DATE=$(date +%Y-%m-%d)

echo "=== Uploading individual Obsidian notes ==="
$REMARQUEE upload md \
    --mermaid-no-sandbox \
    --remote-dir "$REMOTE_BASE" \
    --force \
    --non-interactive \
    "$VAULT_DIR/ARTICLE - DMETA Semantic Inheritance - From Flat Tags to Deli Ordering.md" \
    "$VAULT_DIR/PROJ - Geppetto Embedding Profiles - Profile-Backed Vector Search.md" \
    "$VAULT_DIR/REVIEW - go-go-goja PR 38 - UIDSL attrs and per-call context propagation.md"

echo ""
echo "=== Uploading bundled Obsidian notes ==="
$REMARQUEE upload bundle \
    --mermaid-no-sandbox \
    --name "${DATE} Obsidian Notes" \
    --remote-dir "/ai/2026/05/23/obsidian-notes" \
    --toc-depth 2 \
    --force \
    --non-interactive \
    "$VAULT_DIR/ARTICLE - DMETA Semantic Inheritance - From Flat Tags to Deli Ordering.md" \
    "$VAULT_DIR/PROJ - Geppetto Embedding Profiles - Profile-Backed Vector Search.md" \
    "$VAULT_DIR/REVIEW - go-go-goja PR 38 - UIDSL attrs and per-call context propagation.md"

echo ""
echo "=== Done ==="
$REMARQUEE cloud ls /ai/2026/05/23/ --non-interactive
