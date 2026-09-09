package rmcloud

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/juruen/rmapi/api"
	"github.com/juruen/rmapi/model"
	"github.com/juruen/rmapi/transport"
)

func isolatedCache(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CACHE_HOME", dir)
	t.Setenv("LocalAppData", dir)
	cache, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(cache, "rmapi", "tree.cache")
}

func TestInitializeTreeColdWarmAndCorruptCache(t *testing.T) {
	cache := isolatedCache(t)
	calls := 0
	newHTTP := func() *transport.HttpClientCtx {
		client := transport.CreateHttpClientCtx(model.AuthTokens{UserToken: "fake-token"})
		client.Client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			body := `{"hash":"root-hash","generation":1}`
			// rmapi assigns lowercase header map keys directly (before wire canonicalization).
			if strings.HasSuffix(req.URL.Path, "/root-hash") {
				body = "4\n0:.:0:0\n"
			}
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
		})
		return &client
	}
	for _, tc := range []struct {
		name, notice string
		calls        int
	}{
		{"cold", "No cached tree found", 2},
		{"warm", "Cached tree found", 1},
		{"corrupt", "full resync may still be needed", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "corrupt" {
				if err := os.WriteFile(cache, []byte("not json"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			calls = 0
			var progress bytes.Buffer
			result, err := initializeTree(context.Background(), newHTTP(), api.Version15, &progress)
			if err != nil {
				t.Fatal(err)
			}
			if result == nil || result.Filetree() == nil {
				t.Fatal("missing tree")
			}
			if calls != tc.calls {
				t.Fatalf("got %d requests, want %d", calls, tc.calls)
			}
			got := progress.String()
			if !strings.Contains(got, tc.notice) || !strings.Contains(got, "Cloud tree synchronized.") {
				t.Fatal(got)
			}
			if strings.Contains(got, "fake-token") || strings.Contains(got, "root-hash") {
				t.Fatal("private metadata in progress")
			}
		})
	}
}

func TestInitializeTreeCancellationAndFailure(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		t.Run(map[bool]string{false: "failure", true: "cancel"}[canceled], func(t *testing.T) {
			cache := isolatedCache(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			client := transport.CreateHttpClientCtx(model.AuthTokens{})
			client.Client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if canceled {
					cancel()
					<-req.Context().Done()
					return nil, req.Context().Err()
				}
				return nil, errors.New("network unavailable")
			})
			var progress bytes.Buffer
			_, err := initializeTree(ctx, &client, api.Version15, &progress)
			if err == nil {
				t.Fatal("expected failure")
			}
			if canceled && !errors.Is(err, context.Canceled) {
				t.Fatalf("lost cancellation identity: %v", err)
			}
			if calls != 1 || strings.Contains(progress.String(), "synchronized.") {
				t.Fatal(progress.String())
			}
			if _, err := os.Stat(cache); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("failed tree should not be saved")
			}
		})
	}
}

func TestCancellationDoesNotAuthenticateOrRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// This must return before rmapi accesses config or prompts for real tokens.
	if _, _, err := CreateApiCtx(ctx, AuthSettings{}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	calls := 0
	if _, err := WithAuthRetry(ctx, AuthSettings{}, nil, func(a api.ApiCtx) (api.ApiCtx, error) {
		calls++
		return a, nil
	}); !errors.Is(err, context.Canceled) || calls != 0 {
		t.Fatal("ran canceled operation")
	}

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	_, err := WithAuthRetry(ctx, AuthSettings{}, nil, func(a api.ApiCtx) (api.ApiCtx, error) {
		calls++
		cancel()
		return a, errors.New("HTTP 401 Unauthorized")
	})
	if !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatalf("retried canceled operation: %v, calls=%d", err, calls)
	}
}
