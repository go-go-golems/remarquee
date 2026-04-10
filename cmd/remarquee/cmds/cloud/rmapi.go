package cloud

import (
	"github.com/go-go-golems/remarquee/pkg/rmcloud"
	"github.com/juruen/rmapi/api"
)

type AuthSettings struct {
	NonInteractive bool `glazed:"non-interactive"`
	Reauth         bool `glazed:"reauth"`
}

func createApiCtx(auth AuthSettings) (*api.UserInfo, api.ApiCtx, error) {
	userInfo, apiCtx, err := rmcloud.CreateApiCtx(rmcloud.AuthSettings{
		NonInteractive: auth.NonInteractive,
		Reauth:         auth.Reauth,
	})
	if err != nil {
		return nil, nil, err
	}

	return userInfo, apiCtx, nil
}
