package cloud

import (
	"context"
	"os"

	"github.com/go-go-golems/remarquee/pkg/rmcloud"
	"github.com/juruen/rmapi/api"
)

type AuthSettings struct {
	NonInteractive bool `glazed:"non-interactive"`
	Reauth         bool `glazed:"reauth"`
}

func createApiCtx(ctx context.Context, auth AuthSettings) (*api.UserInfo, api.ApiCtx, error) {
	return rmcloud.CreateApiCtx(ctx, rmcloud.AuthSettings{
		NonInteractive: auth.NonInteractive,
		Reauth:         auth.Reauth,
		Progress:       os.Stderr,
	})
}
