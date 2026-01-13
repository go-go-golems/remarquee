package rmcloud

import (
	"github.com/juruen/rmapi/api"
	"github.com/pkg/errors"
	"strings"
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
