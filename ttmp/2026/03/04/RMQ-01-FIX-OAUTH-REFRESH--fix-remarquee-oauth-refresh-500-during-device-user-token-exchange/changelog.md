# Changelog

## 2026-03-04

- Initial workspace created


## 2026-03-04

Hardened rmapi/remarquee auth refresh: return errors instead of hard exit; add bounded retry + backoff for transient failures (incl. HTTP 500 during /user/new).

### Related Files

- /home/manuel/workspaces/2026-03-04/fix-remarquee-oauth-refresh/remarquee/pkg/rmcloud/auth.go — Backoff between retries when auth/bootstrap fails
- /home/manuel/workspaces/2026-03-04/fix-remarquee-oauth-refresh/rmapi/api/auth.go — Auth bootstrap no longer log.Fatal on user token refresh failure

