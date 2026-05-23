#!/bin/bash
# End-to-end test: generate PDFs from markdown with images and mermaid diagrams,
# then upload to reMarkable.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

echo "=== Setting up test files ==="
mkdir -p "$TMPDIR/assets/img"
python3 "$SCRIPT_DIR/01-gen-test-png.py" "$TMPDIR/assets/img/photo.png"
python3 "$SCRIPT_DIR/01-gen-test-png.py" "$TMPDIR/assets/img/banner.png"

# Mermaid test
cat > "$TMPDIR/test-mermaid.md" << 'EOF'
# Mermaid Diagram Test Suite

## Flowchart

```mermaid
graph TD
    A[Start] --> B{Decision}
    B -->|Yes| C[Action 1]
    B -->|No| D[Action 2]
    C --> E[End]
    D --> E
```

## Sequence Diagram

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server
    C->>S: POST /api/v1/orders
    S-->>C: 201 Created
```

## State Diagram

```mermaid
stateDiagram-v2
    [*] --> Draft
    Draft --> Review: Submit
    Review --> Approved: Accept
    Approved --> Published: Publish
    Published --> [*]
```
EOF

# Image + mermaid test
cat > "$TMPDIR/test-images-and-mermaid.md" << EOF
# Full Feature Test

## Embedded Image

![photo](./assets/img/photo.png)

## Mermaid Diagram

\`\`\`mermaid
graph LR
    A[Input] --> B[Process]
    B --> C[Output]
\`\`\`

## Another Image

![banner](./assets/img/banner.png)
EOF

echo "=== Generating PDFs ==="
REMARQUEE=${REMARQUEE:-remarquee}

$REMARQUEE upload md --pdf-only --output-dir "$TMPDIR/out" \
    --mermaid-no-sandbox "$TMPDIR/test-mermaid.md"

$REMARQUEE upload md --pdf-only --output-dir "$TMPDIR/out" \
    --mermaid-no-sandbox "$TMPDIR/test-images-and-mermaid.md"

echo "=== PDFs generated ==="
ls -la "$TMPDIR/out/"

echo ""
echo "To upload to reMarkable, run:"
echo "  $REMARQUEE upload md --mermaid-no-sandbox --remote-dir /ai/2026/05/23/test --non-interactive $TMPDIR/test-mermaid.md"
echo "  $REMARQUEE upload md --mermaid-no-sandbox --remote-dir /ai/2026/05/23/test --non-interactive $TMPDIR/test-images-and-mermaid.md"
