package rmcloud

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type probeBody struct {
	reads, closes int
	err           error
}

func (b *probeBody) Read([]byte) (int, error) { b.reads++; return 0, b.err }
func (b *probeBody) Close() error             { b.closes++; return nil }

func TestTransportStreamsAndRedacts(t *testing.T) {
	for _, level := range []zerolog.Level{zerolog.DebugLevel, zerolog.Disabled} {
		t.Run(level.String(), func(t *testing.T) {
			var logs bytes.Buffer
			previous := log.Logger
			log.Logger = zerolog.New(&logs).Level(level)
			t.Cleanup(func() { log.Logger = previous })
			bodyErr := errors.New("body failed")
			body := &probeBody{err: bodyErr}
			activity := &syncActivity{}
			var requestCtx context.Context
			tr := &loggingRoundTripper{ctx: context.Background(), activity: activity, base: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				requestCtx = r.Context()
				return &http.Response{StatusCode: 200, Body: body}, nil
			})}
			req, err := http.NewRequest(http.MethodGet, "https://user:password@example.test/private-document?signed=secret", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Authorization", "secret-token")
			req.Header.Set("Cookie", "private-cookie")
			req.Header.Set("Rm-Filename", "private-document")
			resp, err := tr.RoundTrip(req)
			if err != nil {
				t.Fatal(err)
			}
			if body.reads != 0 || body.closes != 0 || requestCtx.Err() != nil || activity.active.Load() != 1 || activity.responses.Load() != 1 {
				t.Fatal("body consumed/closed or context canceled before caller read")
			}
			if _, err := resp.Body.Read(make([]byte, 1)); !errors.Is(err, bodyErr) {
				t.Fatalf("lost body error: %v", err)
			}
			if err := resp.Body.Close(); err != nil {
				t.Fatal(err)
			}
			if body.closes != 1 || activity.active.Load() != 0 || requestCtx.Err() == nil {
				t.Fatal("incorrect body/context ownership")
			}
			got := logs.String()
			for _, secret := range []string{"password", "secret", "private", "example.test"} {
				if strings.Contains(got, secret) {
					t.Fatalf("secret in logs: %q", got)
				}
			}
			if level == zerolog.DebugLevel && !strings.Contains(got, "headers received") {
				t.Fatal(got)
			}
			if level == zerolog.Disabled && got != "" {
				t.Fatal(got)
			}
		})
	}
}

func TestTransportFailureLogsCategoryOnly(t *testing.T) {
	var logs bytes.Buffer
	previous := log.Logger
	log.Logger = zerolog.New(&logs)
	defer func() { log.Logger = previous }()
	a := &syncActivity{}
	tr := &loggingRoundTripper{ctx: context.Background(), activity: a, base: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("https://secret-token@example.test/private")
	})}
	req, _ := http.NewRequest(http.MethodGet, "https://example.test", nil)
	if _, err := tr.RoundTrip(req); err == nil {
		t.Fatal("expected error")
	}
	if a.active.Load() != 0 || a.responses.Load() != 0 || strings.Contains(logs.String(), "secret") {
		t.Fatal(logs.String())
	}
}

func TestTransportCancellationDuringHeadersAndBody(t *testing.T) {
	for _, body := range []bool{false, true} {
		for _, original := range []bool{false, true} {
			t.Run(fmt.Sprintf("body=%v/original=%v", body, original), func(t *testing.T) {
				entered := make(chan struct{})
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if body {
						w.WriteHeader(http.StatusOK)
						w.(http.Flusher).Flush()
					}
					close(entered)
					<-r.Context().Done()
				}))
				defer server.Close()
				commandCtx, cancelCommand := context.WithCancel(context.Background())
				defer cancelCommand()
				requestCtx, cancelRequest := context.WithCancel(context.Background())
				defer cancelRequest()
				a := &syncActivity{}
				client := &http.Client{Transport: &loggingRoundTripper{base: server.Client().Transport, ctx: commandCtx, activity: a}, Timeout: 3 * time.Second}
				defer client.CloseIdleConnections()
				req, _ := http.NewRequestWithContext(requestCtx, http.MethodGet, server.URL, nil)
				result := make(chan error, 1)
				headers := make(chan struct{})
				go func() {
					resp, err := client.Do(req)
					if err == nil {
						close(headers)
						_, err = io.ReadAll(resp.Body)
						_ = resp.Body.Close()
					}
					result <- err
				}()
				select {
				case <-entered:
				case <-time.After(2 * time.Second):
					t.Fatal("request not received")
				}
				if body {
					select {
					case <-headers:
					case <-time.After(2 * time.Second):
						t.Fatal("body was eagerly buffered")
					}
					if a.active.Load() != 1 {
						t.Fatal("body no longer active")
					}
				}
				if original {
					cancelRequest()
				} else {
					cancelCommand()
				}
				select {
				case err := <-result:
					if !errors.Is(err, context.Canceled) {
						t.Fatalf("expected cancellation: %v", err)
					}
				case <-time.After(time.Second):
					t.Fatal("cancellation did not unblock HTTP")
				}
				if a.active.Load() != 0 {
					t.Fatal("leaked active request")
				}
			})
		}
	}
}

func TestTransportConcurrentActivityAndEarlyCancel(t *testing.T) {
	a := &syncActivity{}
	ctx, cancel := context.WithCancel(context.Background())
	tr := &loggingRoundTripper{ctx: ctx, activity: a, base: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("data"))}, nil
	})}
	var wg sync.WaitGroup
	for range 40 {
		wg.Go(func() {
			req, _ := http.NewRequest(http.MethodGet, "https://example.test", nil)
			resp, err := tr.RoundTrip(req)
			if err != nil {
				t.Error(err)
				return
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		})
	}
	wg.Wait()
	if a.responses.Load() != 40 || a.active.Load() != 0 {
		t.Fatal("incorrect concurrent counters")
	}
	cancel()
	req, _ := http.NewRequest(http.MethodGet, "https://example.test", nil)
	if _, err := tr.RoundTrip(req); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if a.responses.Load() != 40 {
		t.Fatal("canceled request reached transport")
	}
}
