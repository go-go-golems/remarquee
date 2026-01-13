//go:build !linux

package devicecapture

import "fmt"

func getFileAndPointer() (readAtCloser, int64, ScreenInfo, error) {
	return nil, 0, ScreenInfo{}, fmt.Errorf("device capture only supported on linux devices")
}
