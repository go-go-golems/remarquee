package rmcloud

import (
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
			unsafe.Pointer(schemaVersionField.UnsafeAddr()),
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
