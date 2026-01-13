package device

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-go-golems/remarquee/pkg/devicecapture"
)

const (
	defaultStreamRate = 200 * time.Millisecond
	minStreamRate     = 100 * time.Millisecond
)

type streamLimiter struct {
	sem chan struct{}
}

func newStreamLimiter(size int) *streamLimiter {
	sem := make(chan struct{}, size)
	for i := 0; i < size; i++ {
		sem <- struct{}{}
	}
	return &streamLimiter{sem: sem}
}

func (l *streamLimiter) Acquire() bool {
	select {
	case <-l.sem:
		return true
	default:
		return false
	}
}

func (l *streamLimiter) Release() {
	l.sem <- struct{}{}
}

func newStreamHandler(reader devicecapture.FramebufferReader) http.Handler {
	limiter := newStreamLimiter(1)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Acquire() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		defer limiter.Release()

		rate := defaultStreamRate
		if rateStr := r.URL.Query().Get("rate"); rateStr != "" {
			val, err := strconv.Atoi(rateStr)
			if err != nil {
				http.Error(w, "Invalid rate parameter", http.StatusBadRequest)
				return
			}
			rate = time.Duration(val) * time.Millisecond
		}
		if rate < minStreamRate {
			http.Error(w, "rate value is too low", http.StatusBadRequest)
			return
		}

		info := reader.ScreenInfo()
		buf := make([]byte, info.ScreenSizeBytes)

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "close")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.Header().Set("X-Device-Model", info.Model)
		w.Header().Set("X-Screen-Width", fmt.Sprintf("%d", info.Width))
		w.Header().Set("X-Screen-Height", fmt.Sprintf("%d", info.Height))
		w.Header().Set("X-Bytes-Per-Pixel", fmt.Sprintf("%d", info.BytesPerPixel))

		ticker := time.NewTicker(rate)
		defer ticker.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case <-ticker.C:
				if err := reader.ReadFrame(buf); err != nil {
					http.Error(w, "failed to read framebuffer", http.StatusInternalServerError)
					return
				}
				if _, err := w.Write(buf); err != nil {
					return
				}
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
			}
		}
	})
}
