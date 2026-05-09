package rmcloud

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/juruen/rmapi/api/sync15"
	"github.com/rs/zerolog/log"
)

// maxLoggedBody is the maximum number of bytes we'll read from a response body for logging.
const maxLoggedBody = 4096

// loggingRoundTripper wraps an http.RoundTripper and logs every request/response at debug level.
type loggingRoundTripper struct {
	base http.RoundTripper
}

func (l *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Log request basics
	logger := log.Debug().
		Str("method", req.Method).
		Str("url", req.URL.String())

	for k, v := range req.Header {
		// Skip auth header values for security
		if strings.EqualFold(k, "Authorization") {
			logger = logger.Strs("header_"+k, []string{"<redacted>"})
		} else {
			logger = logger.Strs("header_"+k, v)
		}
	}
	logger.Msg("rmapi HTTP request")

	resp, err := l.base.RoundTrip(req)
	if err != nil {
		log.Debug().Str("url", req.URL.String()).Err(err).Msg("rmapi HTTP request failed")
		return resp, err
	}

	// Read full body so downstream rmapi code sees the complete response,
	// then log only the first maxLoggedBody bytes as a preview.
	var bodyStr string
	if resp.Body != nil {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr == nil {
			if len(bodyBytes) > maxLoggedBody {
				bodyStr = string(bodyBytes[:maxLoggedBody]) + "…"
			} else {
				bodyStr = string(bodyBytes)
			}
		}
		// Reconstruct the body from the FULL read so the caller isn't truncated.
		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	log.Debug().
		Str("url", req.URL.String()).
		Int("status", resp.StatusCode).
		Str("status_text", resp.Status).
		Str("response_body", bodyStr).
		Msg("rmapi HTTP response")

	return resp, nil
}

// WrapTransportWithLogging wraps the underlying http.Client.Transport of an rmapi
// sync15.ApiCtx so that every HTTP request/response is logged at debug level.
func WrapTransportWithLogging(apiCtx interface{}) {
	syncCtx, ok := apiCtx.(*sync15.ApiCtx)
	if !ok {
		log.Warn().Msg("WrapTransportWithLogging: apiCtx is not *sync15.ApiCtx, skipping")
		return
	}
	if syncCtx.Http == nil || syncCtx.Http.Client == nil {
		log.Warn().Msg("WrapTransportWithLogging: no HTTP client available, skipping")
		return
	}
	client := syncCtx.Http.Client
	if client.Transport == nil {
		client.Transport = http.DefaultTransport
	}
	client.Transport = &loggingRoundTripper{base: client.Transport}
}
