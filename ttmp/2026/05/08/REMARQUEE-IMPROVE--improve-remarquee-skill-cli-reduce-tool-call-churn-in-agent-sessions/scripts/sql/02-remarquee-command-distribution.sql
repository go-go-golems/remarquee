-- 02-remarquee-command-distribution.sql
-- Distribution of exact remarquee bash commands across all sessions.
-- Helps identify the most common remarquee subcommands and patterns.
SELECT
  json_extract_string(tc, '$.input.command') AS cmd,
  COUNT(*) AS cnt
FROM sessions_base, UNNEST(tool_calls) AS t(tc)
WHERE json_extract_string(tc, '$.tool_name') = 'bash'
  AND json_extract_string(tc, '$.input.command') ILIKE '%remarquee%'
GROUP BY 1
ORDER BY cnt DESC
LIMIT 50;
