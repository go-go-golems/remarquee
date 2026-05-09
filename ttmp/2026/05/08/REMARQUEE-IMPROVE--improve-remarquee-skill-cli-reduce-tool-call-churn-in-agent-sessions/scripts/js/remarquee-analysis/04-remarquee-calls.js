// 04-remarquee-calls.js
// Extract every remarquee bash invocation with session context, timestamp,
// subcommand, success/failure, and output/error snippets.
// Run:
//   go-minitrace query commands remarquee-analysis remarquee-calls \
//     --query-repository <ticket>/scripts/js \
//     --archive-glob '<analysis>/pi-minitrace/active/*/*.minitrace.json'

__section__("filters", {
  title: "Filters",
  fields: {
    subcommand: {
      type: "string",
      help: "Filter by remarquee subcommand (upload, cloud, rmdoc, render, status)",
    },
    success_only: {
      type: "bool",
      default: false,
      help: "Only show successful calls",
    },
    failed_only: {
      type: "bool",
      default: false,
      help: "Only show failed calls",
    },
    limit: {
      type: "int",
      default: 200,
      help: "Maximum rows",
    },
  },
});

function remarqueeCalls(filters) {
  const mt = require("minitrace");

  let where = `json_extract_string(tc, '$.tool_name') = 'bash'
    AND json_extract_string(tc, '$.input.command') ILIKE '%remarquee%'`;

  if (filters.subcommand) {
    where += ` AND json_extract_string(tc, '$.input.command') ILIKE '%remarquee ${mt.sql.like(filters.subcommand).slice(1, -1).replace(/^%|%$/g, '')}%'`;
    // simpler: just LIKE the subcommand word
    where = `json_extract_string(tc, '$.tool_name') = 'bash'
      AND json_extract_string(tc, '$.input.command') ILIKE ${mt.sql.like("remarquee " + filters.subcommand)}`;
  }
  if (filters.success_only) {
    where += ` AND json_extract_string(tc, '$.output.success') = 'true'`;
  }
  if (filters.failed_only) {
    where += ` AND (json_extract_string(tc, '$.output.success') != 'true' OR json_extract_string(tc, '$.output.success') IS NULL)`;
  }

  const rows = mt.query(`
    SELECT
      s.id AS session_id,
      s.title AS session_title,
      json_extract_string(tc, '$.id') AS tool_call_id,
      json_extract_string(tc, '$.timestamp') AS timestamp,
      json_extract_string(tc, '$.input.command') AS command,
      CASE
        WHEN json_extract_string(tc, '$.input.command') ILIKE '%remarquee upload%' THEN 'upload'
        WHEN json_extract_string(tc, '$.input.command') ILIKE '%remarquee cloud%' THEN 'cloud'
        WHEN json_extract_string(tc, '$.input.command') ILIKE '%remarquee rmdoc%' THEN 'rmdoc'
        WHEN json_extract_string(tc, '$.input.command') ILIKE '%remarquee render%' THEN 'render'
        WHEN json_extract_string(tc, '$.input.command') ILIKE '%remarquee status%' THEN 'status'
        WHEN json_extract_string(tc, '$.input.command') ILIKE '%remarquee help%' THEN 'help'
        WHEN json_extract_string(tc, '$.input.command') ILIKE '%remarquee --%' THEN 'flag-first'
        ELSE 'other'
      END AS subcommand,
      json_extract_string(tc, '$.output.success') AS success,
      CASE
        WHEN length(json_extract_string(tc, '$.output.result')) > 300
          THEN substring(json_extract_string(tc, '$.output.result'), 1, 300) || '…[truncated]'
        ELSE json_extract_string(tc, '$.output.result')
      END AS result_preview,
      CASE
        WHEN length(json_extract_string(tc, '$.output.error')) > 300
          THEN substring(json_extract_string(tc, '$.output.error'), 1, 300) || '…[truncated]'
        ELSE json_extract_string(tc, '$.output.error')
      END AS error_preview,
      json_extract(tc, '$.output.exit_code') AS exit_code,
      json_extract(tc, '$.output.exit_code') AS exit_code,
      COALESCE(json_extract(tc, '$.context.position_in_session'), 0) AS position_in_session,
      json_extract_string(tc, '$.context.time_since_last_user') AS time_since_last_user
    FROM ${mt.tableName} s, UNNEST(s.tool_calls) AS t(tc)
    WHERE ${where}
    ORDER BY s.id, json_extract_string(tc, '$.timestamp')
    LIMIT ${filters.limit}
  `);

  return rows;
}

__verb__("remarqueeCalls", {
  name: "remarquee-calls",
  short: "Extract all remarquee bash invocations with results",
  long: "Scans tool_calls for bash commands mentioning 'remarquee', extracts the command, subcommand, success/failure, and result/error preview. Useful for understanding how remarquee is used in agent sessions.",
  fields: { filters: { bind: "filters" } },
  tags: ["remarquee", "analysis"],
});
