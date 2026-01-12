//go:build linux

package deviceevents

import (
	"context"
	"fmt"
	"os"
	"unsafe"
)

// EventScanner reads pen/touch input events and publishes them.
type EventScanner struct {
	pen   *os.File
	touch *os.File
}

// NewEventScanner opens input devices for event scanning.
func NewEventScanner() (*EventScanner, error) {
	if penInputDevice == "" || touchInputDevice == "" {
		return nil, fmt.Errorf("input devices not configured for this architecture")
	}
	pen, err := os.OpenFile(penInputDevice, os.O_RDONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to read pen input: %w", err)
	}
	touch, err := os.OpenFile(touchInputDevice, os.O_RDONLY, 0o644)
	if err != nil {
		pen.Close()
		return nil, fmt.Errorf("failed to read touch input: %w", err)
	}
	return &EventScanner{pen: pen, touch: touch}, nil
}

// StartAndPublish begins the event loops.
func (e *EventScanner) StartAndPublish(ctx context.Context, pubsub *PubSub) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				ev, err := readEvent(e.pen)
				if err != nil {
					continue
				}
				pubsub.Publish(InputEventFromSource{Source: Pen, InputEvent: ev})
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				ev, err := readEvent(e.touch)
				if err != nil {
					continue
				}
				pubsub.Publish(InputEventFromSource{Source: Touch, InputEvent: ev})
			}
		}
	}()
}

func readEvent(inputDevice *os.File) (InputEvent, error) {
	var ev InputEvent
	_, err := inputDevice.Read((*(*[unsafe.Sizeof(ev)]byte)(unsafe.Pointer(&ev)))[:])
	return ev, err
}
