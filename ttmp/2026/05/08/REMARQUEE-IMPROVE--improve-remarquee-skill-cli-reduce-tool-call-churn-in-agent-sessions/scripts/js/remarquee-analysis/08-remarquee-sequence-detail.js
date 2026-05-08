// 08-remarquee-sequence-detail.js
// For a given session, emit the full timeline of remarquee calls with
// inter-call gaps, so you can see the exact sequencing pattern.
// This is the "zoom in" companion to 05-remarquee-sequences.js.
//
// Run:
//   go-minitrace query commands remarquee-analysis remarquee-sequence-detail \
//     --query-repository <ticket>/scripts/js \
//     --session-id <SESSION_ID> \
//     --archive-glob '<analysis>/pi-minitrace/active/*/*.minitrace.json'

__section__("filters", {
  title: "Filters",
  fields: {
    session_id: {
      type: "string",
      help: "Specific session ID to examine (required for detail view)",
    },
    gap_threshold_seconds: {
      type: "int",
      default: 120,
      help: "Gap larger than this marks a new sequence boundary",
    },
  },
});

function remarqueeSequenceDetail(filters) {
  const mt = require("minitrace");

  if (!filters.session_id) {
    return [{ error: "Please provide --session-id to see detailed sequence timeline" }];
  }

  const calls = mt.query(`
    SELECT
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
        ELSE 'other'
      END AS subcommand,
      json_extract_string(tc, '$.output.success') AS success,
      CASE
        WHEN length(json_extract_string(tc, '$.output.result')) > 150
          THEN substring(json_extract_string(tc, '$.output.result'), 1, 150) || '…'
        ELSE COALESCE(json_extract_string(tc, '$.output.result'), '')
      END AS result_preview,
      CASE
        WHEN length(json_extract_string(tc, '$.output.error')) > 150
          THEN substring(json_extract_string(tc, '$.output.error'), 1, 150) || '…'
        ELSE COALESCE(json_extract_string(tc, '$.output.error'), '')
      END AS error_preview,
      json_extract(tc, '$.context.position_in_session') AS position_in_session
    FROM ${mt.tableName} s, UNNEST(s.tool_calls) AS t(tc)
    WHERE s.id = ${mt.sql.string(filters.session_id)}
      AND json_extract_string(tc, '$.tool_name') = 'bash'
      AND json_extract_string(tc, '$.input.command') ILIKE '%remarquee%'
    ORDER BY json_extract_string(tc, '$.timestamp')
  `);

  const gapMs = filters.gap_threshold_seconds * 1000;
  const rows = [];
  let prevTs = null;
  let sequenceNum = 0;

  for (const call of calls) {
    const ts = new Date(call.timestamp).getTime();
    const gapSeconds = prevTs ? Math.round((ts - prevTs) / 1000) : 0;
    const isNewSequence = !prevTs || gapSeconds > filters.gap_threshold_seconds;

    if (isNewSequence) sequenceNum++;

    rows.push({
      sequence: sequenceNum,
      timestamp: call.timestamp,
      gap_from_previous_s: gapSeconds,
      is_sequence_start: isNewSequence,
      subcommand: call.subcommand,
      success: String(call.success),
      command_preview: (call.command || "").substring(0, 100),
      result_preview: call.result_preview,
      error_preview: call.error_preview,
      position_in_session: Number(call.position_in_session),
    });

    prevTs = ts;
  }

  return rows;
}

__verb__("remarqueeSequenceDetail", {
  name: "remarquee-sequence-detail",
  short: "Detailed per-session remarquee call timeline with inter-call gaps",
  long: "For a specific session, shows every remarquee call in order with the gap to the previous call, which sequence it belongs to, success/failure, and result/error previews. Use this to understand the exact flow of remarquee usage in a single session.",
  fields: { filters: { bind: "filters" } },
  tags: ["remarquee", "sequences", "detail", "analysis"],
});
