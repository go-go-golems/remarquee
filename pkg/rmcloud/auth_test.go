package rmcloud

import (
	"errors"
	"testing"

	"github.com/juruen/rmapi/api"
)

func TestIsAuthError(t *testing.T) {
	cases := []struct {
		name  string
		err   error
		want  bool
	}{
		{"nil", nil, false},
		{"unrelated", errors.New("connection refused"), false},
		{"401", errors.New("HTTP 401 Unauthorized"), true},
		{"403", errors.New("HTTP 403 Forbidden"), true},
		{"unauthorized lowercase", errors.New("unauthorized access"), true},
		{"forbidden mixed case", errors.New("Request Forbidden"), true},
		{"token expired", errors.New("token Expired at 2026-01-01"), true},
		{"401 in middle", errors.New("got 401 from cloud API"), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsAuthError(tc.err)
			if got != tc.want {
				t.Errorf("IsAuthError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// TestWithAuthRetry_NoError verifies that fn is called exactly once when it succeeds.
func TestWithAuthRetry_NoError(t *testing.T) {
	calls := 0
	_, err := WithAuthRetry(AuthSettings{}, nil, func(ctx api.ApiCtx) (api.ApiCtx, error) {
		calls++
		return ctx, nil
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

// TestWithAuthRetry_NonAuthError verifies that fn is NOT retried on non-auth errors.
func TestWithAuthRetry_NonAuthError(t *testing.T) {
	calls := 0
	_, err := WithAuthRetry(AuthSettings{}, nil, func(ctx api.ApiCtx) (api.ApiCtx, error) {
		calls++
		return ctx, errors.New("connection refused")
	})
	if err == nil {
		t.Error("expected error, got nil")
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retry), got %d", calls)
	}
}

// TestWithAuthRetry_AuthErrorThenSuccess verifies retry-on-auth-error behavior.
// Since we can't create a real api.ApiCtx in unit tests, we use nil as the context
// and verify the call count increases when the first call returns an auth error.
func TestWithAuthRetry_AuthErrorThenSuccess(t *testing.T) {
	calls := 0
	_, err := WithAuthRetry(AuthSettings{NonInteractive: true}, nil, func(ctx api.ApiCtx) (api.ApiCtx, error) {
		calls++
		if calls == 1 {
			return ctx, errors.New("HTTP 401 Unauthorized")
		}
		return ctx, nil
	})
	if err != nil {
		t.Errorf("expected no error after retry, got %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls (initial + retry), got %d", calls)
	}
}
