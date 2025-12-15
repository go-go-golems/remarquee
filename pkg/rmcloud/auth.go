package rmcloud

import (
	"github.com/juruen/rmapi/api"
	"github.com/pkg/errors"
)

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
	// rmapi does retries in its own main.go; keep this minimal for now and bubble errors.
	httpCtx := api.AuthHttpCtx(auth.Reauth, auth.NonInteractive)
	userInfo, err := api.ParseToken(httpCtx.Tokens.UserToken)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to parse rmapi user token")
	}

	apiCtx, err := api.CreateApiCtx(httpCtx, userInfo.SyncVersion)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create rmapi api context")
	}

	return userInfo, apiCtx, nil
}
