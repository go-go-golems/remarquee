package rmcloud

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCacheNotice(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tree.cache")
	if got := cacheNoticeAt(path); !strings.Contains(got, "initial sync may take a while") {
		t.Fatal(got)
	}
	// Existence is not validity; never promise that an existing cache is warm.
	if err := os.WriteFile(path, []byte("invalid cache"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := cacheNoticeAt(path); !strings.Contains(got, "full resync may still be needed") {
		t.Fatal(got)
	}
	// A regular file used as a directory gives a portable stat error (ENOTDIR).
	if got := cacheNoticeAt(filepath.Join(path, "child")); !strings.Contains(got, "Cannot inspect") {
		t.Fatal(got)
	}
}

func TestProgressHeartbeatAndTerminalOutcomes(t *testing.T) {
	for _, tty := range []bool{false, true} {
		for _, tc := range []struct {
			name string
			err  error
			want string
		}{
			{"success", nil, "Cloud tree synchronized."},
			{"error", errors.New("private document name"), "synchronization failed."},
			{"cancel", context.Canceled, "synchronization canceled."},
			{"deadline", context.DeadlineExceeded, "synchronization timed out."},
		} {
			t.Run(tc.name+map[bool]string{true: "-tty", false: "-pipe"}[tty], func(t *testing.T) {
				var buf bytes.Buffer
				a := &syncActivity{}
				a.responses.Store(42)
				a.active.Store(3)
				started := time.Now()
				ticks := make(chan time.Time)
				result := make(chan error)
				done := make(chan struct{})
				go func() {
					runSyncProgress(&buf, tty, started, a, ticks, result)
					close(done)
				}()
				// Unbuffered ticks ensure both heartbeats run, with no new activity.
				ticks <- started.Add(5 * time.Second)
				ticks <- started.Add(10 * time.Second)
				result <- tc.err
				<-done
				got := buf.String()
				for _, want := range []string{"5s | 42 HTTP responses, 3 active requests", "10s | 42 HTTP responses, 3 active requests", tc.want} {
					if !strings.Contains(got, want) {
						t.Errorf("missing %q in %q", want, got)
					}
				}
				if strings.Contains(got, "private") || strings.Contains(got, "%") || strings.Contains(got, "documents") {
					t.Fatalf("misleading or private output: %q", got)
				}
				if tty != strings.Contains(got, "\r\x1b[2K") || !strings.HasSuffix(got, "\n") {
					t.Fatalf("wrong terminal formatting: %q", got)
				}
			})
		}
	}
}

func TestProgressFinishJoinsAndNilWriter(t *testing.T) {
	isolatedCache(t)
	startSyncProgress(nil, &syncActivity{})(nil)
	var buf bytes.Buffer
	finish := startSyncProgress(&buf, &syncActivity{})
	finish(context.Canceled)
	// Safe immediately after finish even under the race detector: no late writes.
	if got := buf.String(); !strings.Contains(got, "Synchronizing cloud tree") || !strings.Contains(got, "canceled") || strings.Contains(got, "synchronized.") {
		t.Fatal(got)
	}
}
