// 10-remarquee-failure-mode-summary.js
// Aggregate failure counts by classified failure mode.
// Companion to 07-remarquee-failures.js which gives raw rows.
//
// Run:
//   go-minitrace query commands remarquee-analysis remarquee-failure-mode-summary \
//     --archive-glob '<analysis>/pi-minitrace/active/*/*.minitrace.json'

__section__("filters", {
  title: "Filters",
  fields: {
    limit: {
      type: "int",
      default: 20,
      help: "Maximum rows",
    },
  },
});

function remarqueeFailureModeSummary(filters) {
  const mt = require("minitrace");

  const failures = mt.query(`
    SELECT
      json_extract_string(tc, '$.input.command') AS command,
      json_extract_string(tc, '$.output.success') AS success,
      COALESCE(json_extract_string(tc, '$.output.error'), '') AS error_text,
      COALESCE(json_extract_string(tc, '$.output.result'), '') AS result_text
    FROM ${mt.tableName} s, UNNEST(s.tool_calls) AS t(tc)
    WHERE json_extract_string(tc, '$.tool_name') = 'bash'
      AND json_extract_string(tc, '$.input.command') ILIKE '%remarquee%'
      AND (json_extract_string(tc, '$.output.success') != 'true'
           OR json_extract_string(tc, '$.output.success') IS NULL)
  `);

  // Classify and count
  const modeCounts = {};

  for (const f of failures) {
    const text = (f.error_text + " " + f.result_text).toLowerCase();
    let mode = "unknown";
    if (text.includes("authentication") || text.includes("unauthorized") || text.includes("401") || text.includes("403")) {
      mode = "auth";
    } else if (text.includes("timeout") || text.includes("timed out")) {
      mode = "timeout";
    } else if (text.includes("not found") || text.includes("no such file") || text.includes("404")) {
      mode = "not-found";
    } else if (text.includes("network") || text.includes("connection") || text.includes("dns") || text.includes("refused")) {
      mode = "network";
    } else if (text.includes("already exists") || text.includes("conflict") || text.includes("409")) {
      mode = "conflict";
    } else if (text.includes("permission denied") || text.includes("access denied")) {
      mode = "permission";
    } else if (text.includes("pandoc") || text.includes("xelatex") || text.includes("latex") || text.includes("exit status 43")) {
      mode = "pandoc-pdf";
    } else if (text.includes("status 400") || text.includes("bad request")) {
      mode = "http-400";
    } else if (text.includes("error:") || text.includes("panic") || text.includes("fatal")) {
      mode = "runtime-error";
    } else if (text.includes("usage:") || text.includes("flag") || text.includes("unknown command") || text.includes("unsupported file type")) {
      mode = "cli-usage";
    }

    modeCounts[mode] = (modeCounts[mode] || 0) + 1;
  }

  const total = Object.values(modeCounts).reduce((a, b) => a + b, 0);

  return Object.entries(modeCounts)
    .map(([mode, count]) => ({
      failure_mode: mode,
      count: Number(count),
      percentage: total > 0 ? (count / total * 100).toFixed(1) + "%" : "N/A",
    }))
    .sort((a, b) => b.count - a.count)
    .slice(0, filters.limit);
}

__verb__("remarqueeFailureModeSummary", {
  name: "remarquee-failure-mode-summary",
  short: "Aggregate failure counts by classified failure mode",
  long: "Classifies all failed remarquee calls into failure modes (auth, pandoc-pdf, http-400, cli-usage, runtime-error, etc.) and shows counts and percentages. Helps identify the most impactful failure patterns to fix.",
  fields: { filters: { bind: "filters" } },
  tags: ["remarquee", "failures", "summary", "analysis"],
});
