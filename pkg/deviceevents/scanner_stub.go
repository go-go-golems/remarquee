//go:build !linux

package deviceevents

import (
	"context"
	"fmt"
)

// EventScanner is a stub for non-linux builds.
type EventScanner struct{}

// NewEventScanner returns an error on non-linux platforms.
func NewEventScanner() (*EventScanner, error) {
	return nil, fmt.Errorf("device events are only supported on linux")
}

// StartAndPublish is a no-op for the stub.
func (e *EventScanner) StartAndPublish(_ context.Context, _ *PubSub) {}
