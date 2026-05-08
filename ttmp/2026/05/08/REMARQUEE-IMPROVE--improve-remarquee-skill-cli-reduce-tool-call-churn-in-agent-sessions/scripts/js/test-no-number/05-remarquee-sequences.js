// 05-remarquee-sequences.js
// Group remarquee calls into temporal "sequences" — bursts of closely-spaced
// remarquee invocations within the same session.
//
// A "sequence" is a maximal run of remarquee calls where each call is within
// `gap_seconds` of the previous one in the same session.
//
// Run:
//   go-minitrace query commands remarquee-analysis remarquee-sequences \
//     --query-repository <ticket>/scripts/js \
//     --archive-glob '<analysis>/pi-minitrace/active/*/*.minitrace.json'

__section__("sequence", {
  title: "Sequence Detection",
  fields: {
    gap_seconds: {
      type: "int",
      default: 120,
      help: "Max gap (seconds) between remarquee calls to consider them part of the same sequence",
    },
    min_sequence_length: {
      type: "int",
      default: 2,
      help: "Minimum calls in a sequence to include in output",
    },
    limit: {
      type: "int",
      default: 100,
      help: "Maximum number of sequences to return",
    },
  },
});

function remarqueeSequences(filters) {
  const mt = require("minitrace");

  // Step 1: pull all remarquee calls ordered by session + timestamp
  const calls = mt.query(`
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
        ELSE 'other'
      END AS subcommand,
      json_extract_string(tc, '$.output.success') AS success,
      json_extract(tc, '$.context.position_in_session') AS position_in_session
    FROM ${mt.tableName} s, UNNEST(s.tool_calls) AS t(tc)
    WHERE json_extract_string(tc, '$.tool_name') = 'bash'
      AND json_extract_string(tc, '$.input.command') ILIKE '%remarquee%'
    ORDER BY s.id, json_extract_string(tc, '$.timestamp')
  `);

  // Step 2: group into sequences
  const gapMs = filters.gap_seconds * 1000;
  const sequences = [];
  let current = null;

  for (const call of calls) {
    const ts = new Date(call.timestamp).getTime();

    if (!current || current.session_id !== call.session_id || (ts - current.lastTs) > gapMs) {
      // Start a new sequence
      if (current && current.calls.length >= filters.min_sequence_length) {
        sequences.push(current);
      }
      current = {
        session_id: call.session_id,
        session_title: call.session_title,
        sequence_index: sequences.length + 1,
        calls: [],
        subcommands: new Set(),
        firstTs: ts,
        lastTs: ts,
        successCount: 0,
        failCount: 0,
      };
    }

    current.calls.push(call);
    current.lastTs = ts;
    current.subcommands.add(call.subcommand);
    if (String(call.success) === "true") {
      current.successCount++;
    } else {
      current.failCount++;
    }
  }
  // Don't forget the last one
  if (current && current.calls.length >= filters.min_sequence_length) {
    sequences.push(current);
  }

  // Step 3: emit summary rows
  const rows = [];
  for (const seq of sequences) {
    const duration = seq.lastTs - seq.firstTs;
    rows.push({
      session_id: seq.session_id,
      session_title: (seq.session_title || "").substring(0, 60),
      sequence_index: seq.sequence_index,
      call_count: seq.calls.length,
      subcommands: Array.from(seq.subcommands).sort().join(", "),
      success_count: seq.successCount,
      fail_count: seq.failCount,
      duration_seconds: Math.round(duration / 1000),
      first_timestamp: new Date(seq.firstTs).toISOString(),
      commands_summary: seq.calls
        .slice(0, 5)
        .map((c) => {
          const cmd = c.command || "";
          // Extract just the remarquee subcommand + key args
          const m = cmd.match(/remarquee\s+(\S+(?:\s+--\S+)*)/);
          return m ? m[1].substring(0, 60) : cmd.substring(0, 60);
        })
        .join(" → ")
        + (seq.calls.length > 5 ? ` … (+${seq.calls.length - 5} more)` : ""),
    });

    if (rows.length >= filters.limit) break;
  }

  return rows;
}

__verb__("remarqueeSequences", {
  name: "remarquee-sequences",
  short: "Detect temporal sequences of remarquee calls within sessions",
  long: "Groups remarquee bash invocations into sequences based on a configurable time gap. Each sequence is a maximal run of remarquee calls where consecutive calls are within gap_seconds of each other in the same session. Outputs sequence metadata: call count, subcommands used, success/fail counts, duration, and a command summary.",
  fields: { filters: { bind: "sequence" } },
  tags: ["remarquee", "sequences", "analysis"],
});
