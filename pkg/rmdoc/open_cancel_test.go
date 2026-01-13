package rmdoc

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

type errAfterNContext struct {
	context.Context
	after int32
	n     int32
}

func (c *errAfterNContext) Err() error {
	if atomic.AddInt32(&c.n, 1) > c.after {
		return context.Canceled
	}
	return nil
}

func TestOpenReaderAtHonorsContextCancellation(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// Large enough to guarantee multiple read iterations with the 32KiB buffer.
	w, err := zw.Create("abc.content")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := w.Write(bytes.Repeat([]byte("a"), 2*1024*1024)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	r := bytes.NewReader(buf.Bytes())
	ctx := &errAfterNContext{Context: context.Background(), after: 5}

	_, err = OpenReaderAt(ctx, r, int64(r.Len()))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %T %v", err, err)
	}
}
