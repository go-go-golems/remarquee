//go:build linux && arm64

package devicecapture

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	deviceModel       = "RemarkablePaperPro"
	screenWidth       = 1632
	screenHeight      = 2154
	bytesPerPixel     = 4
	screenSizeBytes   = screenWidth * screenHeight * bytesPerPixel
	penInputDevice    = "/dev/input/event2"
	touchInputDevice  = "/dev/input/event3"
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
	startAddress, err := getMemoryRange(pid)
	if err != nil {
		return 0, fmt.Errorf("failed to get memory range: %w", err)
	}
	framePointer, err := calculateFramePointer(pid, startAddress)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate frame pointer: %w", err)
	}
	return framePointer, nil
}

func getMemoryRange(pid string) (int64, error) {
	file, err := os.Open(fmt.Sprintf("/proc/%s/maps", pid))
	if err != nil {
		return 0, fmt.Errorf("cannot open maps file: %w", err)
	}
	defer file.Close()

	var memoryRange string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "/dev/dri/card0") {
			memoryRange = line
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error reading maps file: %w", err)
	}
	if memoryRange == "" {
		return 0, fmt.Errorf("no mapping found for /dev/dri/card0")
	}

	fields := strings.Fields(memoryRange)
	rangeField := fields[0]
	startEnd := strings.Split(rangeField, "-")
	if len(startEnd) != 2 {
		return 0, fmt.Errorf("invalid memory range format")
	}
	end, err := strconv.ParseInt(startEnd[1], 16, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse end address: %w", err)
	}
	return end, nil
}

func calculateFramePointer(pid string, startAddress int64) (int64, error) {
	file, err := os.Open(fmt.Sprintf("/proc/%s/mem", pid))
	if err != nil {
		return 0, fmt.Errorf("cannot open memory file: %w", err)
	}
	defer file.Close()

	var offset int64
	length := 2
	for length < screenSizeBytes {
		offset += int64(length - 2)
		if _, err := file.Seek(startAddress+offset+8, 0); err != nil {
			return 0, fmt.Errorf("error seeking memory header: %w", err)
		}
		header := make([]byte, 8)
		_, err := file.Read(header)
		if err != nil {
			return 0, fmt.Errorf("error reading memory header: %w", err)
		}
		length = int(int64(header[0]) | int64(header[1])<<8 | int64(header[2])<<16 | int64(header[3])<<24)
	}
	return startAddress + offset, nil
}
