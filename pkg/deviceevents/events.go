package deviceevents

import "syscall"

const (
	// Input event types (see Linux input-event-codes).
	EvSyn      = 0
	EvKey      = 1
	EvRel      = 2
	EvAbs      = 3
	EvMsc      = 4
	EvSw       = 5
	EvLed      = 17
	EvSnd      = 18
	EvRep      = 20
	EvFf       = 21
	EvPwr      = 22
	EvFfStatus = 23
)

const (
	// Pen input source.
	Pen int = 1
	// Touch input source.
	Touch int = 2
)

// InputEvent represents a Linux input event structure.
type InputEvent struct {
	Time  syscall.Timeval `json:"-"`
	Type  uint16          `json:"type"`
	Code  uint16          `json:"code"`
	Value int32           `json:"value"`
}

// InputEventFromSource wraps input events with a source label.
type InputEventFromSource struct {
	Source int `json:"source"`
	InputEvent
}
