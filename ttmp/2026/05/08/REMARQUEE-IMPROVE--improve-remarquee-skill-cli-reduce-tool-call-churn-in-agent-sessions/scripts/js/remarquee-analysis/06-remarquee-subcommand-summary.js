// 06-remarquee-subcommand-summary.js
// Aggregate statistics per remarquee subcommand: count, success rate,
// common failure modes, and typical command patterns.
//
// Run:
//   go-minitrace query commands remarquee-analysis remarquee-subcommand-summary \
//     --query-repository <ticket>/scripts/js \
//     --archive-glob '<analysis>/pi-minitrace/active/*/*.minitrace.json'

__section__("filters", {
  title: "Filters",
  fields: {
    limit: {
      type: "int",
      default: 30,
      help: "Maximum rows",
    },
  },
});

function remarqueeSubcommandSummary(filters) {
  const mt = require("minitrace");

  const rows = mt.query(`
    SELECT
      CASE
        WHEN json_extract_string(tc, '$.input.command') ILIKE '%remarquee upload bundle%' THEN 'upload bundle'
        WHEN json_extract_string(tc, '$.input.command') ILIKE '%remarquee upload md%' THEN 'upload md'
        WHEN json_extract_string(tc, '$.input.command') ILIKE '%remarquee upload%' THEN 'upload (other)'
        WHEN json_extract_string(tc, '$.input.command') ILIKE '%remarquee cloud ls%' THEN 'cloud ls'
        WHEN json_extract_string(tc, '$.input.command') ILIKE '%remarquee cloud account%' THEN 'cloud account'
        WHEN json_extract_string(tc, '$.input.command') ILIKE '%remarquee cloud rm%' THEN 'cloud rm'
        WHEN json_extract_string(tc, '$.input.command') ILIKE '%remarquee cloud%' THEN 'cloud (other)'
        WHEN json_extract_string(tc, '$.input.command') ILIKE '%remarquee rmdoc%' THEN 'rmdoc'
        WHEN json_extract_string(tc, '$.input.command') ILIKE '%remarquee render%' THEN 'render'
        WHEN json_extract_string(tc, '$.input.command') ILIKE '%remarquee status%' THEN 'status'
        WHEN json_extract_string(tc, '$.input.command') ILIKE '%remarquee help%' THEN 'help'
        WHEN json_extract_string(tc, '$.input.command') ILIKE '%remarquee --%' THEN 'flag-first'
        ELSE 'other'
      END AS subcommand,
      COUNT(*) AS total_calls,
      SUM(CASE WHEN json_extract_string(tc, '$.output.success') = 'true' THEN 1 ELSE 0 END) AS success_count,
      SUM(CASE WHEN json_extract_string(tc, '$.output.success') != 'true' OR json_extract_string(tc, '$.output.success') IS NULL THEN 1 ELSE 0 END) AS fail_count,
      COUNT(DISTINCT s.id) AS session_count
    FROM ${mt.tableName} s, UNNEST(s.tool_calls) AS t(tc)
    WHERE json_extract_string(tc, '$.tool_name') = 'bash'
      AND json_extract_string(tc, '$.input.command') ILIKE '%remarquee%'
    GROUP BY 1
    ORDER BY total_calls DESC
    LIMIT ${filters.limit}
  `);

  // Cast BigInt from DuckDB aggregates to Number
  return rows.map((r) => {
    const total = Number(r.total_calls);
    const success = Number(r.success_count);
    const fail = Number(r.fail_count);
    const sessions = Number(r.session_count);
    return {
      subcommand: r.subcommand,
      total_calls: total,
      success_count: success,
      fail_count: fail,
      session_count: sessions,
      success_rate: total > 0
        ? (success / total * 100).toFixed(1) + "%"
        : "N/A",
      calls_per_session: sessions > 0
        ? (total / sessions).toFixed(1)
        : "N/A",
    };
  });
}

__verb__("remarqueeSubcommandSummary", {
  name: "remarquee-subcommand-summary",
  short: "Aggregate statistics per remarquee subcommand",
  long: "Groups remarquee invocations by fine-grained subcommand (upload bundle, upload md, cloud ls, etc.), showing total count, success/fail split, success rate, session count, and calls-per-session.",
  fields: { filters: { bind: "filters" } },
  tags: ["remarquee", "summary", "analysis"],
});
