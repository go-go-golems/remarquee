package device

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-go-golems/remarquee/pkg/deviceevents"
)

func newEventsHandler(bus *deviceevents.PubSub) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if bus == nil {
			http.Error(w, "events not available", http.StatusServiceUnavailable)
			return
		}
		eventC := bus.Subscribe()
		defer bus.Unsubscribe(eventC)

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		for {
			select {
			case <-r.Context().Done():
				return
			case event := <-eventC:
				if event.Source != deviceevents.Pen {
					continue
				}
				if event.Type != deviceevents.EvAbs {
					continue
				}
				payload, err := json.Marshal(event)
				if err != nil {
					http.Error(w, "failed to encode event", http.StatusInternalServerError)
					return
				}
				_, _ = fmt.Fprintf(w, "data: %s\n\n", payload)
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
			}
		}
	})
}
