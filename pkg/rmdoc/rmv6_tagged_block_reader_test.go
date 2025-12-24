package rmdoc

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func remarqueeRootFromThisFile(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	// This file lives at: <repo>/remarquee/pkg/rmdoc/...
	// Module root is: <repo>/remarquee
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}

func readFirstRMFileFromRMDoc(t *testing.T, rmdocPath string) []byte {
	t.Helper()

	f, err := os.Open(rmdocPath)
	if err != nil {
		t.Fatalf("open rmdoc fixture: %v", err)
	}
	defer func() { _ = f.Close() }()

	fi, err := f.Stat()
	if err != nil {
		t.Fatalf("stat rmdoc fixture: %v", err)
	}

	zr, err := zip.NewReader(f, fi.Size())
	if err != nil {
		t.Fatalf("zip.NewReader: %v", err)
	}

	for _, zf := range zr.File {
		if !strings.HasSuffix(zf.Name, ".rm") {
			continue
		}
		rc, err := zf.Open()
		if err != nil {
			t.Fatalf("open zip entry: %v", err)
		}
		defer func() { _ = rc.Close() }()

		b, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("read .rm entry: %v", err)
		}
		return b
	}

	t.Fatalf("no .rm files found in fixture %s", rmdocPath)
	return nil
}

func TestRMV6TaggedBlockReader_ReadHeaderAndBlocks(t *testing.T) {
	root := remarqueeRootFromThisFile(t)
	fixture := filepath.Join(root, "cmd", "remarquee-ui", "testdata", "cpage-pdf.rmdoc")

	rmBytes := readFirstRMFileFromRMDoc(t, fixture)
	r := newRMV6TaggedBlockReader(bytes.NewReader(rmBytes))

	if err := r.readHeader(); err != nil {
		t.Fatalf("readHeader: %v", err)
	}

	var (
		blockCount    int
		subblockCount int
	)

	for {
		blk, err := r.readBlockHeader()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("readBlockHeader: %v", err)
		}
		blockCount++

		// Best-effort: scan within the block for a Length4 subblock boundary and exercise subblock parsing.
		for {
			rem, err := r.bytesRemainingInBlock()
			if err != nil {
				t.Fatalf("bytesRemainingInBlock: %v", err)
			}
			if rem <= 0 {
				break
			}

			idx, tt, ok, err := r.peekTag()
			if err != nil {
				t.Fatalf("peekTag: %v", err)
			}
			if !ok {
				break
			}

			if tt == rmV6TagTypeLength4 {
				sb, err := r.readSubBlock(idx)
				if err != nil {
					t.Fatalf("readSubBlock: %v", err)
				}
				if err := r.discardSubBlockRemainder(sb); err != nil {
					t.Fatalf("discardSubBlockRemainder: %v", err)
				}
				subblockCount++
				break
			}

			_, tt2, err := r.readTag()
			if err != nil {
				break
			}
			if err := r.skipTaggedValue(tt2); err != nil {
				break
			}
		}

		if err := r.endBlock(blk); err != nil {
			t.Fatalf("endBlock: %v", err)
		}
	}

	if blockCount == 0 {
		t.Fatalf("expected at least 1 block, got %d", blockCount)
	}
	if subblockCount == 0 {
		t.Fatalf("expected to find at least 1 Length4 subblock, got %d", subblockCount)
	}
}
