package rmcloud

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// loggingRoundTripper binds rmapi's context-free requests to the command and
// records metadata without buffering bodies or logging cloud/account contents.
// Install it before tree initialization, not after CreateApiCtx returns.
type loggingRoundTripper struct {
	base     http.RoundTripper
	ctx      context.Context
	activity *syncActivity
}

func (l *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := l.ctx.Err(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(req.Context())
	stop := context.AfterFunc(l.ctx, cancel)
	// AfterFunc runs asynchronously; close the preflight cancellation race.
	if l.ctx.Err() != nil {
		cancel()
	}
	started := time.Now()
	l.activity.active.Add(1)
	finish := sync.OnceFunc(func() {
		l.activity.active.Add(-1)
		stop()
		cancel()
	})
	log.Debug().Str("method", req.Method).Msg("rmapi HTTP request started")
	base := l.base
	if base == nil {
		base = http.DefaultTransport
	}
	resp, err := base.RoundTrip(req.Clone(ctx))
	if err != nil {
		finish()
		log.Debug().Str("method", req.Method).Str("outcome", errorKind(err)).Dur("elapsed", time.Since(started)).Msg("rmapi HTTP request failed")
		return resp, err
	}
	l.activity.responses.Add(1)
	log.Debug().Str("method", req.Method).Int("status", resp.StatusCode).Dur("elapsed", time.Since(started)).Msg("rmapi HTTP response headers received")
	if resp.Body == nil {
		finish()
	} else {
		resp.Body = &activityBody{ReadCloser: resp.Body, finish: finish}
	}
	return resp, nil
}

// CloseIdleConnections preserves http.Client cleanup through this wrapper.
func (l *loggingRoundTripper) CloseIdleConnections() {
	base := l.base
	if base == nil {
		base = http.DefaultTransport
	}
	if closer, ok := base.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
}

// Preserve the original body's Close and read errors. Context cancellation must
// remain attached after headers arrive, until the caller has consumed the body.
type activityBody struct {
	io.ReadCloser
	finish func()
}

func (b *activityBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if err != nil {
		b.finish()
	}
	return n, err
}

func (b *activityBody) Close() error {
	defer b.finish()
	return b.ReadCloser.Close()
}

// Transport errors can embed credential-bearing URLs. Log a category instead.
func errorKind(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline"
	default:
		return "error"
	}
}
