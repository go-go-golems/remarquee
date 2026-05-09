package rmcloud

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"unsafe"

	"github.com/juruen/rmapi/api"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
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

// forceSchemaV4 uses reflection to set the underlying sync15 HashTree.SchemaVersion to "4"
// when it is empty. This avoids the global side-effect of os.Setenv and works around the
// rmapi bug where an empty SchemaVersion defaults to V3, causing 400 "invalid hash" from
// the reMarkable cloud.
func forceSchemaV4(apiCtx api.ApiCtx) {
	v := reflect.ValueOf(apiCtx)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return
	}
	v = v.Elem()

	hashTreeField := v.FieldByName("hashTree")
	if !hashTreeField.IsValid() || hashTreeField.IsNil() {
		return
	}

	schemaVersionField := hashTreeField.Elem().FieldByName("SchemaVersion")
	if !schemaVersionField.IsValid() {
		return
	}

	oldSchemaVersion := schemaVersionField.String()
	if oldSchemaVersion != "4" {
		// Values reached through unexported fields are not settable by default.
		// Create a new, settable Value backed by the same memory address.
		schemaVersionField = reflect.NewAt(
			schemaVersionField.Type(),
			unsafe.Pointer(schemaVersionField.UnsafeAddr()), // #nosec G103 -- rmapi keeps hashTree unexported; this targeted workaround sets SchemaVersion only.
		).Elem()
		schemaVersionField.SetString("4")
		log.Debug().Str("old_schema_version", oldSchemaVersion).Msg("rmcloud: set HashTree.SchemaVersion to 4 via reflection")
	}
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

		forceSchemaV4(apiCtx)
		WrapTransportWithLogging(apiCtx)

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

	// Auth error: retry once with fresh context. Unit tests pass a nil context
	// and have no rmapi token store; in that case, only exercise retry control flow.
	if apiCtx == nil {
		return fn(apiCtx)
	}

	fmt.Fprintln(os.Stderr, "NOTE: auth expired, re-authenticating and retrying...")

	reauthAuth := auth
	reauthAuth.Reauth = true
	_, freshCtx, reauthErr := CreateApiCtx(reauthAuth)
	if reauthErr != nil {
		return apiCtx, errors.Wrap(err, "auth failed and re-authentication also failed")
	}

	return fn(freshCtx)
}
