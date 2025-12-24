package rmdoc

import (
	"archive/zip"
	"context"
	"io"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
)

// DetectRMVersionFromHeader attempts to parse an .rm header buffer and returns a
// version tag ("V3"/"V5"/"V6") when recognized.
func DetectRMVersionFromHeader(header []byte) (string, bool) {
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

type RMFile struct {
	PageID  string
	Version string
	Bytes   []byte
}

// ReadRMFileFromArchive reads the .rm file for a given pageID (matching by base name),
// returning ok=false if no such entry exists.
func ReadRMFileFromArchive(ctx context.Context, archivePath string, pageID string) (*RMFile, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	if pageID == "" {
		return nil, false, errors.New("pageID is empty")
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return nil, false, errors.Wrap(err, "open archive")
	}
	defer func() { _ = f.Close() }()

	fi, err := f.Stat()
	if err != nil {
		return nil, false, errors.Wrap(err, "stat archive")
	}

	zr, err := zip.NewReader(f, fi.Size())
	if err != nil {
		return nil, false, errors.Wrap(err, "open zip reader")
	}

	targetName := pageID + ".rm"

	var found *zip.File
	for _, zf := range zr.File {
		if err := ctx.Err(); err != nil {
			return nil, false, err
		}
		if !strings.HasSuffix(zf.Name, ".rm") {
			continue
		}
		if path.Base(zf.Name) == targetName {
			found = zf
			break
		}
	}

	if found == nil {
		return nil, false, nil
	}

	rc, err := found.Open()
	if err != nil {
		return nil, false, errors.Wrap(err, "open rm entry")
	}
	defer func() { _ = rc.Close() }()

	b, err := io.ReadAll(rc)
	if err != nil {
		return nil, false, errors.Wrap(err, "read rm entry")
	}

	version := "unknown"
	if len(b) >= 43 {
		if v, ok := DetectRMVersionFromHeader(b[:43]); ok {
			version = v
		}
	}

	return &RMFile{
		PageID:  pageID,
		Version: version,
		Bytes:   b,
	}, true, nil
}
