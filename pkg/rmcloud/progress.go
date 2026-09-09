package rmcloud

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"golang.org/x/term"
)

// syncActivity counts HTTP activity, not documents. Headers count as a response;
// a request remains active until its body reaches EOF/errors or is closed.
type syncActivity struct {
	responses atomic.Int64
	active    atomic.Int64
}

func cacheNotice() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "Cache location unavailable; synchronizing cloud tree."
	}
	return cacheNoticeAt(filepath.Join(dir, "rmapi", "tree.cache"))
}

func cacheNoticeAt(path string) string {
	_, err := os.Stat(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return "No cached tree found; initial sync may take a while."
	case err != nil:
		return "Cannot inspect tree cache; synchronization will report any cache error."
	default:
		return "Cached tree found; checking for remote changes (a full resync may still be needed)."
	}
}

// startSyncProgress owns its ticker and joins its goroutine on finish. A nil
// writer opts library callers out of CLI progress entirely.
func startSyncProgress(w io.Writer, activity *syncActivity) func(error) {
	if w == nil {
		return func(error) {}
	}
	tty := false
	if f, ok := w.(*os.File); ok {
		tty = term.IsTerminal(int(f.Fd()))
	}
	interval := 5 * time.Second
	if tty {
		interval = time.Second
	}
	fmt.Fprintln(w, cacheNotice())
	started := time.Now()
	renderSyncProgress(w, tty, started, started, activity, "", false)
	ticker := time.NewTicker(interval)
	result := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer ticker.Stop()
		runSyncProgress(w, tty, started, activity, ticker.C, result)
	}()
	return func(err error) {
		result <- err
		<-done
	}
}

func runSyncProgress(w io.Writer, tty bool, started time.Time, activity *syncActivity, ticks <-chan time.Time, result <-chan error) {
	for {
		select {
		case now := <-ticks:
			renderSyncProgress(w, tty, started, now, activity, "", false)
		case err := <-result:
			outcome := "Cloud tree synchronized."
			if err != nil {
				outcome = "Cloud tree synchronization failed."
				if kind := errorKind(err); kind == "canceled" {
					outcome = "Cloud tree synchronization canceled."
				} else if kind == "deadline" {
					outcome = "Cloud tree synchronization timed out."
				}
			}
			renderSyncProgress(w, tty, started, time.Now(), activity, outcome, true)
			return
		}
	}
}

func renderSyncProgress(w io.Writer, tty bool, started, now time.Time, activity *syncActivity, outcome string, final bool) {
	prefix, suffix := "", "\n"
	if tty {
		prefix = "\r\x1b[2K"
		if !final {
			suffix = ""
		}
	}
	elapsed := now.Sub(started).Truncate(time.Second)
	if final {
		fmt.Fprintf(w, "%s%s %s | %d HTTP responses%s", prefix, outcome, elapsed, activity.responses.Load(), suffix)
		return
	}
	fmt.Fprintf(w, "%sSynchronizing cloud tree… %s | %d HTTP responses, %d active requests%s",
		prefix, elapsed, activity.responses.Load(), activity.active.Load(), suffix)
}
