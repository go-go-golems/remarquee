---
Title: 'Bug report: PDF render crops right edge'
Ticket: RMQ-0013
Status: active
Topics:
    - rendering
    - remarkable
DocType: working-note
Intent: long-term
Owners: []
RelatedFiles:
    - Path: remarquee/pkg/rmdoc/render/v6_merge_background.go
      Note: Potential merge math source for cropping
ExternalSources: []
Summary: Rendered PDFs crop content on the right edge; likely page layout or merge math issue.
LastUpdated: 2026-01-12T12:41:32.841225774-05:00
WhatFor: Track right-edge cropping bug in rendered PDFs for later investigation.
WhenToUse: When debugging PDF render alignment or page sizing issues.
---


# Bug report: PDF render crops right edge

## Summary

Rendered PDFs cut off content on the right-hand side by a noticeable margin. This suggests a page layout or merge offset issue in the rendering pipeline.

## Notes

- Symptom: rendered PDF output crops content on the right edge.
- Impact: annotations or content near the right margin are missing in the rendered output.
- Context: observed during current render output review (no specific page indices recorded yet).

## Decisions

N/A

## Next Steps

- Reproduce with a known .rmdoc page and collect screenshots/PNGs.
- Compare rendered output vs device screenshot to quantify the right-edge offset.
