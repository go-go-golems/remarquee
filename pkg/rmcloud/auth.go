package rmcloud

import (
	"strings"

	"github.com/juruen/rmapi/api"
	"github.com/pkg/errors"
)

const authRetries = 3

// AuthSettings mirrors the common rmapi authentication knobs used by remarquee commands.
//
// Note: command-layer structs may add struct tags (e.g. glazed.parameter), but this package
// stays tag-free so it can be reused from Cobra-only or Glazed-based commands.
type AuthSettings struct {
	NonInteractive bool
	Reauth         bool
}

// CreateApiCtx creates an rmapi ApiCtx using rmapi's token bootstrap logic.
func CreateApiCtx(auth AuthSettings) (*api.UserInfo, api.ApiCtx, error) {
	var lastErr error
	for i := 0; i < authRetries; i++ {
		reauth := auth.Reauth || i > 0
		httpCtx := api.AuthHttpCtx(reauth, auth.NonInteractive)
		if httpCtx.Tokens.UserToken == "" {
			lastErr = errors.New("rmapi did not return a user token; device token may be invalid, run `rmapi reset` and re-register this device")
			continue
		}
		userInfo, err := api.ParseToken(httpCtx.Tokens.UserToken)
		if err != nil {
			if strings.Contains(err.Error(), "token Expired") {
				lastErr = errors.Wrap(err, "rmapi user token expired after reauth; check system clock or re-register via `rmapi reset`")
			} else {
				lastErr = errors.Wrap(err, "failed to parse rmapi user token")
			}
			continue
		}

		apiCtx, err := api.CreateApiCtx(httpCtx, userInfo.SyncVersion)
		if err != nil {
			lastErr = errors.Wrap(err, "failed to create rmapi api context")
			continue
		}

		return userInfo, apiCtx, nil
	}

	if lastErr == nil {
		lastErr = errors.New("failed to create rmapi api context")
	}
	return nil, nil, lastErr
}

// IsAuthError returns true if the error is caused by an authentication failure
// (401 Unauthorized, 403 Forbidden, or expired token). Upload commands use this
// to decide whether to retry with reauth.
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "401") ||
		strings.Contains(msg, "unauthorized") ||
		strings.Contains(msg, "403") ||
		strings.Contains(msg, "forbidden") ||
		strings.Contains(msg, "token expired")
}

// WithAuthRetry executes fn with the given apiCtx. If fn returns an auth error
// (401/403/expired token), it re-creates the API context with reauth=true and
// retries fn once. This eliminates the common pattern where an agent's upload
// fails with 401, forcing a separate `remarquee cloud account --reauth` tool
// call before retrying the upload.
//
// Returns the re-created apiCtx (which may differ from the input) so callers
// can continue using the refreshed context for subsequent operations.
func WithAuthRetry(
	auth AuthSettings,
	apiCtx api.ApiCtx,
	fn func(api.ApiCtx) (api.ApiCtx, error),
) (api.ApiCtx, error) {
	newCtx, err := fn(apiCtx)
	if err == nil || !IsAuthError(err) {
		return newCtx, err
	}

	// Auth error: retry once with fresh context.
	reauthAuth := auth
	reauthAuth.Reauth = true
	_, freshCtx, reauthErr := CreateApiCtx(reauthAuth)
	if reauthErr != nil {
		// Return the original error (the auth failure), not the reauth failure,
		// unless reauth itself failed.
		return apiCtx, errors.Wrap(err, "auth failed and re-authentication also failed")
	}

	return fn(freshCtx)
}
