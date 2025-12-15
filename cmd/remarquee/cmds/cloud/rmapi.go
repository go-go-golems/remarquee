package cloud

import (
	"github.com/juruen/rmapi/api"
	"github.com/pkg/errors"
)

type AuthSettings struct {
	NonInteractive bool `glazed.parameter:"non-interactive"`
	Reauth         bool `glazed.parameter:"reauth"`
}

func createApiCtx(auth AuthSettings) (*api.UserInfo, api.ApiCtx, error) {
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
