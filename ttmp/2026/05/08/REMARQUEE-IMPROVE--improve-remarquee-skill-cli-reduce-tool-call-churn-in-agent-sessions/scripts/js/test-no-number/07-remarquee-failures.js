// 07-remarquee-failures.js
// Deep dive into remarquee failures: extract error messages, classify
// failure modes, and identify the most common failure patterns.
//
// Run:
//   go-minitrace query commands remarquee-analysis remarquee-failures \
//     --query-repository <ticket>/scripts/js \
//     --archive-glob '<analysis>/pi-minitrace/active/*/*.minitrace.json'

__section__("filters", {
  title: "Filters",
  fields: {
    subcommand: {
      type: "string",
      help: "Filter by remarquee subcommand (upload, cloud, rmdoc, render, status)",
    },
    limit: {
      type: "int",
      default: 100,
      help: "Maximum rows",
    },
  },
});

function remarqueeFailures(filters) {
  const mt = require("minitrace");

  let subcommandFilter = "";
  if (filters.subcommand) {
    subcommandFilter = `AND json_extract_string(tc, '$.input.command') ILIKE ${mt.sql.like("remarquee " + filters.subcommand)}`;
  }

  const failures = mt.query(`
    SELECT
      s.id AS session_id,
      s.title AS session_title,
      json_extract_string(tc, '$.id') AS tool_call_id,
      json_extract_string(tc, '$.timestamp') AS timestamp,
      json_extract_string(tc, '$.input.command') AS command,
      CASE
        WHEN json_extract_string(tc, '$.input.command') ILIKE '%remarquee upload bundle%' THEN 'upload bundle'
        WHEN json_extract_string(tc, '$.input.command') ILIKE '%remarquee upload md%' THEN 'upload md'
        WHEN json_extract_string(tc, '$.input.command') ILIKE '%remarquee upload%' THEN 'upload (other)'
        WHEN json_extract_string(tc, '$.input.command') ILIKE '%remarquee cloud ls%' THEN 'cloud ls'
        WHEN json_extract_string(tc, '$.input.command') ILIKE '%remarquee cloud account%' THEN 'cloud account'
        WHEN json_extract_string(tc, '$.input.command') ILIKE '%remarquee cloud%' THEN 'cloud (other)'
        WHEN json_extract_string(tc, '$.input.command') ILIKE '%remarquee rmdoc%' THEN 'rmdoc'
        WHEN json_extract_string(tc, '$.input.command') ILIKE '%remarquee render%' THEN 'render'
        WHEN json_extract_string(tc, '$.input.command') ILIKE '%remarquee status%' THEN 'status'
        ELSE 'other'
      END AS subcommand,
      CASE
        WHEN length(json_extract_string(tc, '$.output.error')) > 500
          THEN substring(json_extract_string(tc, '$.output.error'), 1, 500) || '…'
        ELSE COALESCE(json_extract_string(tc, '$.output.error'), '')
      END AS error_text,
      CASE
        WHEN length(json_extract_string(tc, '$.output.result')) > 300
          THEN substring(json_extract_string(tc, '$.output.result'), 1, 300) || '…'
        ELSE COALESCE(json_extract_string(tc, '$.output.result'), '')
      END AS result_text,
      json_extract(tc, '$.output.exit_code') AS exit_code
    FROM ${mt.tableName} s, UNNEST(s.tool_calls) AS t(tc)
    WHERE json_extract_string(tc, '$.tool_name') = 'bash'
      AND json_extract_string(tc, '$.input.command') ILIKE '%remarquee%'
      AND (json_extract_string(tc, '$.output.success') != 'true'
           OR json_extract_string(tc, '$.output.success') IS NULL)
      ${subcommandFilter}
    ORDER BY s.id, json_extract_string(tc, '$.timestamp')
    LIMIT ${filters.limit}
  `);

  // Classify failure mode from error/result text
  return failures.map((f) => {
    const text = (f.error_text + " " + f.result_text).toLowerCase();
    let failure_mode = "unknown";
    if (text.includes("authentication") || text.includes("auth") || text.includes("unauthorized") || text.includes("401") || text.includes("403")) {
      failure_mode = "auth";
    } else if (text.includes("timeout") || text.includes("timed out")) {
      failure_mode = "timeout";
    } else if (text.includes("not found") || text.includes("no such file") || text.includes("404")) {
      failure_mode = "not-found";
    } else if (text.includes("network") || text.includes("connection") || text.includes("dns") || text.includes("refused")) {
      failure_mode = "network";
    } else if (text.includes("already exists") || text.includes("conflict") || text.includes("409")) {
      failure_mode = "conflict";
    } else if (text.includes("permission denied") || text.includes("access denied")) {
      failure_mode = "permission";
    } else if (text.includes("error:") || text.includes("panic") || text.includes("fatal")) {
      failure_mode = "runtime-error";
    } else if (text.includes("usage:") || text.includes("flag") || text.includes("unknown command")) {
      failure_mode = "cli-usage";
    }

    return {
      session_id: f.session_id,
      session_title: (f.session_title || "").substring(0, 50),
      timestamp: f.timestamp,
      subcommand: f.subcommand,
      failure_mode,
      exit_code: f.exit_code,
      error_snippet: f.error_text.substring(0, 200),
      result_snippet: f.result_text.substring(0, 200),
    };
  });
}

__verb__("remarqueeFailures", {
  name: "remarquee-failures",
  short: "Extract and classify remarquee call failures",
  long: "Pulls all failed remarquee bash invocations and classifies the failure mode (auth, timeout, network, not-found, conflict, permission, runtime-error, cli-usage, unknown). Useful for identifying the most impactful failure patterns to fix.",
  fields: { filters: { bind: "filters" } },
  tags: ["remarquee", "failures", "analysis"],
});
