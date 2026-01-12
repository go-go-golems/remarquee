//go:build linux && !arm && !arm64

package devicecapture

import "fmt"

func deviceScreenInfo() ScreenInfo {
	return ScreenInfo{}
}

func getFramePointer(_ string) (int64, error) {
	return 0, fmt.Errorf("framebuffer capture only supported on arm/arm64")
}
