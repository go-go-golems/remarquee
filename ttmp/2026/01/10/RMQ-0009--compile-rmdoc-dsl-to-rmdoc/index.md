---
Title: 'Compile RMDoc-DSL → .rmdoc (editable notebooks)'
Ticket: RMQ-0009
Status: active
Topics:
  - remarkable
  - rmdoc
  - rendering
  - dsl
  - compiler
  - go
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: >
  Implement a compiler that converts RMDoc-DSL (YAML + JS/goja) into real .rmdoc archives
  (V6 cPages notebooks) so that generated fixtures can be uploaded to the device as editable
  notebooks and used as ground-truth references.
LastUpdated: 2026-01-10
---

# Compile RMDoc-DSL → .rmdoc (editable notebooks)

## Overview

RMQ-0006 proved that RMDoc-DSL is an excellent tool for *reproducible* fixture generation and debugging, but we still rely on PDFs as “transport” for device review. PDFs are good for viewing, but they are not the same as an editable notebook on-device.

This ticket implements the missing bridge:

- RMDoc-DSL (declarative intent) → **real `.rmdoc` archive** (device-native representation)

Once this exists, we can:

- generate fixtures programmatically,
- upload them to the tablet as editable notebooks,
- capture device screenshots that correspond to exactly the same bytes,
- and use those as stable, repeatable truth for renderer validation.

## Tasks

See [tasks.md](./tasks.md).

## Reference / intern guide

See [reference/01-intern-guide.md](./reference/01-intern-guide.md).


