-- 01-tool-name-distribution.sql
-- Quick survey of all tool names across the archive to understand what's available
-- and confirm remarquee appears only via bash invocations.
SELECT
  json_extract_string(tc, '$.tool_name') AS tool_name,
  COUNT(*) AS cnt
FROM sessions_base, UNNEST(tool_calls) AS t(tc)
GROUP BY 1
ORDER BY cnt DESC
LIMIT 30;
