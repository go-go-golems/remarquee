-- 03-remarquee-total-count.sql
-- Total count of remarquee-related bash invocations across the archive.
SELECT
  COUNT(*) AS total_remq_calls
FROM sessions_base, UNNEST(tool_calls) AS t(tc)
WHERE json_extract_string(tc, '$.tool_name') = 'bash'
  AND json_extract_string(tc, '$.input.command') ILIKE '%remarquee%';
