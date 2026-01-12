package device

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-go-golems/remarquee/pkg/deviceevents"
)

type gesture struct {
	LeftDistance  int `json:"left"`
	RightDistance int `json:"right"`
	UpDistance    int `json:"up"`
	DownDistance  int `json:"down"`
}

func (g *gesture) sum() int {
	return g.LeftDistance + g.RightDistance + g.UpDistance + g.DownDistance
}

func (g *gesture) reset() {
	g.LeftDistance = 0
	g.RightDistance = 0
	g.UpDistance = 0
	g.DownDistance = 0
}

func newGesturesHandler(bus *deviceevents.PubSub) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if bus == nil {
			http.Error(w, "gestures not available", http.StatusServiceUnavailable)
			return
		}
		eventC := bus.Subscribe()
		defer bus.Unsubscribe(eventC)

		const (
			codeXAxis          uint16        = 54
			codeYAxis          uint16        = 53
			gestureMaxInterval time.Duration = 150 * time.Millisecond
		)

		tick := time.NewTicker(gestureMaxInterval)
		defer tick.Stop()
		current := &gesture{}
		lastX := deviceevents.InputEventFromSource{}
		lastY := deviceevents.InputEventFromSource{}
		hasLastX := false
		hasLastY := false

		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		enc := json.NewEncoder(w)

		for {
			select {
			case <-r.Context().Done():
				return
			case <-tick.C:
				if current.sum() != 0 {
					if err := enc.Encode(current); err != nil {
						http.Error(w, "failed to encode gesture", http.StatusInternalServerError)
						return
					}
					if flusher, ok := w.(http.Flusher); ok {
						flusher.Flush()
					}
				}
				current.reset()
				lastX = deviceevents.InputEventFromSource{}
				lastY = deviceevents.InputEventFromSource{}
				hasLastX = false
				hasLastY = false
			case event := <-eventC:
				if event.Source != deviceevents.Touch {
					continue
				}
				if event.Type != deviceevents.EvAbs {
					continue
				}
				switch event.Code {
				case codeXAxis:
					if !hasLastX {
						lastX = event
						hasLastX = true
						continue
					}
					distance := event.Value - lastX.Value
					if distance < 0 {
						current.RightDistance += int(-distance)
					} else {
						current.LeftDistance += int(distance)
					}
					lastX = event
				case codeYAxis:
					if !hasLastY {
						lastY = event
						hasLastY = true
						continue
					}
					distance := event.Value - lastY.Value
					if distance < 0 {
						current.UpDistance += int(-distance)
					} else {
						current.DownDistance += int(distance)
					}
					lastY = event
				}
				tick.Reset(gestureMaxInterval)
			}
		}
	})
}
