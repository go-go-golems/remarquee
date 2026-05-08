// 09-remarquee-churn-metrics.js
// Compute per-session churn metrics for remarquee usage:
//   - total remarquee calls per session
//   - ratio of remarquee calls to total bash calls
//   - number of distinct remarquee sequences
//   - "churn score" = sequences * (1 + fail_count / total)
// This identifies sessions with the most remarquee tool-call overhead.
//
// Run:
//   go-minitrace query commands remarquee-analysis remarquee-churn-metrics \
//     --query-repository <ticket>/scripts/js \
//     --archive-glob '<analysis>/pi-minitrace/active/*/*.minitrace.json'

__section__("filters", {
  title: "Filters",
  fields: {
    min_remq_calls: {
      type: "int",
      default: 3,
      help: "Minimum remarquee calls in a session to include",
    },
    limit: {
      type: "int",
      default: 50,
      help: "Maximum sessions to return",
    },
  },
});

function remarqueeChurnMetrics(filters) {
  const mt = require("minitrace");

  // Get per-session remarquee call counts and bash totals
  const stats = mt.query(`
    SELECT
      s.id AS session_id,
      s.title AS session_title,
      CAST(s.metrics->>'tool_call_count' AS INT) AS total_tool_calls,
      CAST(s.metrics->>'turn_count' AS INT) AS total_turns,
      SUM(CASE WHEN json_extract_string(tc, '$.tool_name') = 'bash'
               AND json_extract_string(tc, '$.input.command') ILIKE '%remarquee%'
          THEN 1 ELSE 0 END) AS remq_call_count,
      SUM(CASE WHEN json_extract_string(tc, '$.tool_name') = 'bash'
               AND json_extract_string(tc, '$.input.command') ILIKE '%remarquee%'
               AND json_extract_string(tc, '$.output.success') = 'true'
          THEN 1 ELSE 0 END) AS remq_success_count,
      SUM(CASE WHEN json_extract_string(tc, '$.tool_name') = 'bash'
               AND json_extract_string(tc, '$.input.command') ILIKE '%remarquee%'
               AND (json_extract_string(tc, '$.output.success') != 'true'
                    OR json_extract_string(tc, '$.output.success') IS NULL)
          THEN 1 ELSE 0 END) AS remq_fail_count,
      SUM(CASE WHEN json_extract_string(tc, '$.tool_name') = 'bash' THEN 1 ELSE 0 END) AS total_bash_calls
    FROM ${mt.tableName} s, UNNEST(s.tool_calls) AS t(tc)
    GROUP BY s.id, s.title, s.metrics
  `);

  // Filter to sessions with enough remarquee calls (cast BigInt)
  const filtered = stats.filter((s) => s.remq_call_count >= filters.min_remq_calls);

  // Now get remarquee call timestamps per qualifying session to count sequences
  const sessionIds = filtered.map((s) => s.session_id);
  if (sessionIds.length === 0) return [];

  const callTimeline = mt.query(`
    SELECT
      s.id AS session_id,
      json_extract_string(tc, '$.timestamp') AS timestamp
    FROM ${mt.tableName} s, UNNEST(s.tool_calls) AS t(tc)
    WHERE json_extract_string(tc, '$.tool_name') = 'bash'
      AND json_extract_string(tc, '$.input.command') ILIKE '%remarquee%'
      AND s.id IN (${mt.sql.stringIn(sessionIds.slice(0, 500))})
    ORDER BY s.id, json_extract_string(tc, '$.timestamp')
  `);

  // Count sequences per session (gap > 120s = new sequence)
  const sequenceCounts = {};
  let prevSession = null;
  let prevTs = null;

  for (const row of callTimeline) {
    const ts = new Date(row.timestamp).getTime();
    if (row.session_id !== prevSession || !prevTs || (ts - prevTs) > 120000) {
      sequenceCounts[row.session_id] = (sequenceCounts[row.session_id] || 0) + 1;
    }
    prevSession = row.session_id;
    prevTs = ts;
  }

  // Build output rows with churn score (cast all BigInt from DuckDB)
  return filtered
    .map((s) => {
      const remqCalls = s.remq_call_count;
      const remqSuccesses = s.remq_success_count;
      const remqFailures = s.remq_fail_count;
      const totalBash = s.total_bash_calls;
      const totalTools = s.total_tool_calls;
      const totalTurns = s.total_turns;
      const seqs = sequenceCounts[s.session_id] || 1;
      const failRatio = remqCalls > 0
        ? remqFailures / remqCalls
        : 0;
      const churnScore = Math.round(seqs * (1 + failRatio) * 10) / 10;
      const remqRatio = totalBash > 0
        ? (remqCalls / totalBash * 100).toFixed(1) + "%"
        : "N/A";

      return {
        session_id: s.session_id,
        session_title: (s.session_title || "").substring(0, 60),
        remq_calls: remqCalls,
        remq_successes: remqSuccesses,
        remq_failures: remqFailures,
        total_bash: totalBash,
        total_tools: totalTools,
        total_turns: totalTurns,
        remq_bash_ratio: remqRatio,
        sequences: seqs,
        churn_score: churnScore,
      };
    })
    .sort((a, b) => b.churn_score - a.churn_score)
    .slice(0, filters.limit);
}

__verb__("remarqueeChurnMetrics", {
  name: "remarquee-churn-metrics",
  short: "Per-session remarquee churn metrics and scores",
  long: "Computes per-session metrics: remarquee call count, success/fail split, ratio to total bash calls, number of temporal sequences, and a churn score (sequences × (1 + fail_ratio)). Higher churn scores indicate sessions where remarquee caused the most tool-call overhead.",
  fields: { filters: { bind: "filters" } },
  tags: ["remarquee", "churn", "metrics", "analysis"],
});
