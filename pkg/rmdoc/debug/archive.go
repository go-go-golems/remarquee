package debug

import (
	"archive/zip"
	"context"
	"io"
	"math"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
)

// RMFileInfo represents metadata about a .rm file inside an rmdoc archive.
type RMFileInfo struct {
	PageID   string
	Filename string
	Size     int64
	Version  string // "V3", "V5", "V6", or "unknown"
}

// DetectRMVersionFromHeader attempts to parse an .rm header buffer and returns a
// version tag ("V3"/"V5"/"V6") when recognized.
//
// The canonical header format is:
//
//	"reMarkable .lines file, version=X      " (43 bytes total)
func DetectRMVersionFromHeader(header []byte) (string, bool) {
	// Keep this permissive: we only need to detect known versions from the header.
	s := string(header)
	switch {
	case strings.Contains(s, "version=3"):
		return "V3", true
	case strings.Contains(s, "version=5"):
		return "V5", true
	case strings.Contains(s, "version=6"):
		return "V6", true
	default:
		return "", false
	}
}

// ListArchiveFiles returns the names of every file entry inside the archive.
func ListArchiveFiles(ctx context.Context, archivePath string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return nil, errors.Wrap(err, "open archive")
	}
	defer func() { _ = f.Close() }()

	fi, err := f.Stat()
	if err != nil {
		return nil, errors.Wrap(err, "stat archive")
	}

	zr, err := zip.NewReader(f, fi.Size())
	if err != nil {
		return nil, errors.Wrap(err, "open zip reader")
	}

	out := make([]string, 0, len(zr.File))
	for _, zf := range zr.File {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		out = append(out, zf.Name)
	}
	return out, nil
}

// InspectRMFiles returns metadata for every .rm file entry inside the archive.
func InspectRMFiles(ctx context.Context, archivePath string) ([]RMFileInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return nil, errors.Wrap(err, "open archive")
	}
	defer func() { _ = f.Close() }()

	fi, err := f.Stat()
	if err != nil {
		return nil, errors.Wrap(err, "stat archive")
	}

	zr, err := zip.NewReader(f, fi.Size())
	if err != nil {
		return nil, errors.Wrap(err, "open zip reader")
	}

	var out []RMFileInfo
	for _, zf := range zr.File {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !strings.HasSuffix(zf.Name, ".rm") {
			continue
		}

		pageID := strings.TrimSuffix(path.Base(zf.Name), ".rm")
		version := "unknown"

		rc, err := zf.Open()
		if err != nil {
			return nil, errors.Wrapf(err, "open rm entry %s", zf.Name)
		}

		header := make([]byte, 43)
		n, readErr := io.ReadFull(rc, header)
		_ = rc.Close()
		if readErr == nil && n == 43 {
			if v, ok := DetectRMVersionFromHeader(header); ok {
				version = v
			}
		}

		if zf.UncompressedSize64 > math.MaxInt64 {
			return nil, errors.Errorf("rm entry %s size %d exceeds max int64", zf.Name, zf.UncompressedSize64)
		}
		out = append(out, RMFileInfo{
			PageID:   pageID,
			Filename: zf.Name,
			Size:     int64(zf.UncompressedSize64),
			Version:  version,
		})
	}

	return out, nil
}
