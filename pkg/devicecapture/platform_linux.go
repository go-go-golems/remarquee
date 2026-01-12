//go:build linux

package devicecapture

import (
	"fmt"
	"os"
)

func getFileAndPointer() (readAtCloser, int64, ScreenInfo, error) {
	pid, err := findXochitlPID()
	if err != nil {
		return nil, 0, ScreenInfo{}, err
	}
	if pid == "" {
		return nil, 0, ScreenInfo{}, fmt.Errorf("xochitl pid not found")
	}
	file, err := os.OpenFile("/proc/"+pid+"/mem", os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return nil, 0, ScreenInfo{}, err
	}
	pointerAddr, err := getFramePointer(pid)
	if err != nil {
		if cerr := file.Close(); cerr != nil {
			return nil, 0, ScreenInfo{}, fmt.Errorf("get frame pointer: %w (close mem: %v)", err, cerr)
		}
		return nil, 0, ScreenInfo{}, err
	}
	info := deviceScreenInfo()
	return file, pointerAddr, info, nil
}
