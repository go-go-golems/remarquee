---
Title: Remarquee Call Sequence Analysis Design
Ticket: REMARQUEE-IMPROVE
Status: active
Topics:
    - remarquee
    - minitrace
    - transcript-analysis
    - tool-churn
DocType: design
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Analysis design and improvement recommendations from transcript mining"
LastUpdated: 2026-05-08T06:25:00.000000000-04:00
WhatFor: Concrete improvements to reduce remarquee tool-call churn
WhenToUse: When implementing CLI or skill changes to reduce churn
---

# Remarquee Call Sequence Analysis Design

## Goal

Reduce the number of remarquee-related tool calls in Pi agent sessions by
improving both the CLI and the skill, based on evidence from transcript analysis.

## Data Summary

- **2,861** remarquee bash calls across **458** Pi sessions
- **182** failed calls (6.4% failure rate)
- **"other"** subcommand category (multi-subcommand scripts) has 23.2 calls/session — the highest
- The **upload-then-verify** anti-pattern accounts for the majority of churn

## Identified Anti-Patterns

### 1. Upload-Then-Verify Spiral (3-5 calls → should be 1)

**Current flow:**
```
remarquee status              → check prerequisites
remarquee upload bundle ...   → do the upload
remarquee cloud ls ...        → verify it landed
remarquee cloud ls ...        → verify again with different flags
```

**Observed in:** virtually every upload session (186 sessions with `upload bundle`, 66 with `upload md`)

**Root cause:** The agent doesn't trust the upload result and feels compelled to verify. The CLI doesn't emit enough structured information to confirm success without a separate verification call.

**Recommendation:**
- **CLI**: Make `remarquee upload bundle/md` emit a structured success message including the remote path, file size, and document ID. Add `--json` output mode.
- **Skill**: Instruct the agent that a successful `remarquee upload` output (containing "OK: uploaded") is sufficient confirmation — no need for `cloud ls` verification unless the upload explicitly fails.

### 2. Status-Check Before Every Upload (1 redundant call)

**Current flow:**
```
remarquee status              → always called first
remarquee upload bundle ...   → then the actual upload
```

**Observed in:** 202 calls to `remarquee status` across 164 sessions (1.2 calls/session)

**Root cause:** The skill instructs agents to verify prerequisites. But remarquee is almost always available — 100% success rate on `status`.

**Recommendation:**
- **Skill**: Remove the "check status first" instruction. If the upload fails with a clear error, the agent can diagnose then. Don't pre-check.
- **CLI**: If `remarquee upload` fails due to missing prerequisites, emit a clear error with remediation instructions, so the agent doesn't need a separate status check.

### 3. Cloud-Account Re-Auth Loop (1-2 extra calls)

**Current flow:**
```
remarquee upload md ...       → fails with 401
remarquee cloud account       → check auth
remarquee cloud account --reauth → re-authenticate
remarquee upload md ...       → retry
```

**Observed in:** 5 auth failures, 134 cloud account calls across 109 sessions

**Recommendation:**
- **CLI**: When an upload gets a 401, automatically attempt re-auth once before failing. This eliminates 2 tool calls (cloud account + reauth) from the agent's perspective.
- **Skill**: Tell the agent to pass `--reauth` on the first upload attempt if the session has been idle for >1 hour, rather than waiting for failure.

### 4. Pandoc/PDF Filename Retry Loop (3-5 calls)

**Current flow:**
```
remarquee upload md "Big Brother Review.md" → fails (spaces in filename → status 400)
remarquee upload md --help                  → check available flags
remarquee upload md (with underscores)     → fails again
rmapi mput / rmapi refresh                 → try underlying tool
remarquee upload md (yet another variant)   → finally succeeds
```

**Observed in:** 38 pandoc-pdf failures, 10 http-400 failures. Session 019dbe19 had 5 upload failures in a row.

**Recommendation:**
- **CLI**: Sanitize PDF filenames automatically (replace spaces with underscores, strip special characters) before upload. This is a client-side fix that eliminates the retry loop.
- **CLI**: When a 400 error occurs, parse the rmapi error and suggest the specific fix (filename, directory name, etc.) in the error message.
- **Skill**: Tell the agent to use `--name` with a simple name (no special chars) instead of relying on auto-naming from the markdown title.

### 5. Help-Flag Exploration (1-2 extra calls per session)

**Current flow:**
```
remarquee upload --help       → agent doesn't know the exact flags
remarquee cloud --help        → checking subcommand options
remarquee cloud put --help    → checking another subcommand
```

**Observed in:** 8 explicit `help` calls, plus the `flag-first` category (51 calls where `--` flags come before the subcommand)

**Recommendation:**
- **Skill**: Include a compact reference of all common remarquee commands and their key flags in the SKILL.md itself, so the agent doesn't need to call `--help`.
- **CLI**: Add `remarquee upload bundle --dry-run` to show what would be uploaded without actually doing it.

### 6. Cloud LS as Directory Browser (2-3 calls)

**Current flow:**
```
remarquee cloud ls /ai/2026/04/24          → browse parent
remarquee cloud ls /ai/2026/04/24/CSSVD    → browse child
remarquee cloud ls /ai/2026/04/24/CSSVD --long → get details
```

**Observed in:** 497 `cloud ls` calls across 185 sessions. 2.7 calls/session means agents are repeatedly browsing.

**Recommendation:**
- **CLI**: Add `--recursive` flag to `cloud ls` to show nested structure in one call.
- **Skill**: Tell the agent to use `--long --non-interactive` on the first call instead of making two separate calls.

## Implementation Priority

| Priority | Anti-Pattern | Impact | Effort |
|---|---|---|---|
| P0 | Upload-Then-Verify | Eliminates 1-2 calls per upload (huge) | Skill change only |
| P0 | Status-Check-Before-Upload | Eliminates 1 call per session | Skill change only |
| P1 | Pandoc filename sanitization | Eliminates 3-5 call retry loops | CLI change |
| P1 | Auto-reauth on 401 | Eliminates 2 calls per auth failure | CLI change |
| P2 | Compact flag reference in skill | Eliminates 1-2 help calls per session | Skill change |
| P2 | Recursive cloud ls | Eliminates 1-2 ls calls per browse | CLI change |
| P3 | Upload --dry-run | Reduces failed uploads | CLI change |

## Estimated Reduction

If P0 and P1 changes are implemented, we estimate:
- **upload bundle**: 5.5 → 2 calls/session (remove status + verify)
- **upload md**: 4.9 → 2 calls/session (remove status + verify + filename retries)
- **cloud ls**: 2.7 → 1.5 calls/session (recursive + skill guidance)
- **status**: 1.2 → 0 calls/session (eliminated entirely)

Overall remarquee calls would drop from **2,861 → ~1,400** (≈50% reduction).

## Query Commands Reference

All analysis scripts are in `scripts/js/remarquee-analysis/` and auto-discovered via `.envrc`:

```bash
# Subcommand distribution
go-minitrace query commands remarquee-analysis 06-remarquee-subcommand-summary remarquee-subcommand-summary --archive-glob '.../*.minitrace.json'

# Failure mode breakdown
go-minitrace query commands remarquee-analysis 10-remarquee-failure-mode-summary remarquee-failure-mode-summary --archive-glob '.../*.minitrace.json'

# Per-session churn scores
go-minitrace query commands remarquee-analysis 09-remarquee-churn-metrics remarquee-churn-metrics --archive-glob '.../*.minitrace.json'

# Temporal sequences
go-minitrace query commands remarquee-analysis 05-remarquee-sequences remarquee-sequences --archive-glob '.../*.minitrace.json'

# Per-session detail
go-minitrace query commands remarquee-analysis 08-remarquee-sequence-detail remarquee-sequence-detail --archive-glob '.../*.minitrace.json' --session-id <ID>
```
