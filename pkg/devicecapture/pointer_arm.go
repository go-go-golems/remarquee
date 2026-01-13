//go:build linux && arm

package devicecapture

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

const (
	deviceModel      = "Remarkable2"
	screenWidth      = 1872
	screenHeight     = 1404
	bytesPerPixel    = 2
	screenSizeBytes  = screenWidth * screenHeight * bytesPerPixel
	penInputDevice   = "/dev/input/event1"
	touchInputDevice = "/dev/input/event2"
)

func deviceScreenInfo() ScreenInfo {
	return ScreenInfo{
		Model:           deviceModel,
		Width:           screenWidth,
		Height:          screenHeight,
		BytesPerPixel:   bytesPerPixel,
		ScreenSizeBytes: screenSizeBytes,
	}
}

func getFramePointer(pid string) (int64, error) {
	file, err := os.OpenFile("/proc/"+pid+"/maps", os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanWords)
	scanAddr := false
	var addr int64
	for scanner.Scan() {
		if scanAddr {
			hex := strings.Split(scanner.Text(), "-")[0]
			addr, err = strconv.ParseInt("0x"+hex, 0, 64)
			break
		}
		if scanner.Text() == "/dev/fb0" {
			scanAddr = true
		}
	}
	if err != nil {
		return 0, err
	}
	return addr + 8, nil
}
