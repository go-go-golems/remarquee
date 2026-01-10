---
Title: 'Generative validation suite (RMDoc-DSL): reproducible renderer improvement'
Ticket: RMQ-0008
Status: active
Topics:
  - remarkable
  - rmdoc
  - rendering
  - testing
  - validation
  - generative
  - dsl
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: >
  Build an extensive, reusable generative validation suite using RMDoc-DSL (YAML + JS/goja)
  to continuously improve and validate the renderer with reproducible fixtures, sweeps, and
  property-based checks.
LastUpdated: 2026-01-10
---

# Generative validation suite (RMDoc-DSL): reproducible renderer improvement

## Overview

This ticket formalizes the “sweeps + inverse + device review” work into a reusable, extensible test suite. The core idea is: instead of debugging with ad-hoc fixtures and screenshots, we generate controlled families of cases (ellipses, highlights, anchors, typed text layouts) and verify renderer invariants against:

- our own deterministic programmatic renderers
- remarks reference output (when available)
- device review loops (when necessary)

## Status

Current status: **active**

## Tasks

See [tasks.md](./tasks.md).

## Changelog

See [changelog.md](./changelog.md).


