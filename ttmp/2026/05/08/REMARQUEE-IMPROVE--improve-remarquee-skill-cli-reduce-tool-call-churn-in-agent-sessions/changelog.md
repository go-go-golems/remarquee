# Changelog

## 2026-05-08

- Initial workspace created


## 2026-05-08

Steps 1-3: Created ticket, converted 458 Pi sessions to minitrace, wrote 7 JS analysis commands (04-10) + 3 SQL (01-03), ran full analysis against archive. Key finding: 2,861 remarquee calls, 6.4% failure rate, upload-then-verify anti-pattern causes most churn. Estimated 50% reduction achievable.

### Related Files

- /home/manuel/workspaces/2026-05-08/improve-tooling/remarquee/ttmp/2026/05/08/REMARQUEE-IMPROVE--improve-remarquee-skill-cli-reduce-tool-call-churn-in-agent-sessions/scripts/js/remarquee-analysis/06-remarquee-subcommand-summary.js — Subcommand distribution analysis


## 2026-05-08

Steps 4-5: Implemented P0 (skill file rewrite for remarkable-upload + remarkable-render-pdf) and P1 (sanitizePDFName in CLI + IsAuthError). Wrote 27KB Obsidian vault article teaching the transcript mining method.

### Related Files

- /home/manuel/.pi/agent/skills/remarkable-upload/SKILL.md — P0 skill rewrite eliminating status/verify/help churn


## 2026-05-08

Step 6: Updated go-minitrace docs (js-api-reference.md + troubleshooting.md) with BigInt/Number, struct-vs-JSON, and bash command path issues. Committed as 1460bba.

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/doc/js-api-reference.md — Added BigInt handling section


## 2026-05-08

Step 7: Investigated BigInt/Goja boundary. Traced data path: DuckDB → rows.Scan → NormalizeValue() → map[string]any → Goja. Found NormalizeValue() in pkg/query/engine.go is the single fix point. Stored test script as 11-test-duckdb-types.go. Cherry-picked doc commit into ./go-minitrace.

### Related Files

- /home/manuel/workspaces/2026-05-08/improve-tooling/go-minitrace/pkg/query/engine.go — NormalizeValue is the fix point for BigInt→Number coercion

