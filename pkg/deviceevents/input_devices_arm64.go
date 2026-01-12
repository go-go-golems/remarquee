//go:build linux && arm64

package deviceevents

const (
	penInputDevice   = "/dev/input/event2"
	touchInputDevice = "/dev/input/event3"
)
