//go:build linux

package devicecapture

import (
	"fmt"
	"os"
	"path/filepath"
)

func findXochitlPID() (string, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid := entry.Name()
		procEntries, err := os.ReadDir(filepath.Join("/proc", pid))
		if err != nil {
			continue
		}
		for _, procEntry := range procEntries {
			info, err := procEntry.Info()
			if err != nil {
				continue
			}
			if info.Mode()&os.ModeSymlink == 0 {
				continue
			}
			orig, err := os.Readlink(filepath.Join("/proc", pid, procEntry.Name()))
			if err != nil {
				continue
			}
			if orig == "/usr/bin/xochitl" {
				return pid, nil
			}
		}
	}
	return "", fmt.Errorf("xochitl process not found")
}
