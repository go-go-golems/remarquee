package render

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/go-go-golems/remarquee/pkg/rmdoc"
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

func TestBuildBackgroundPDFHonorsContextCancellation(t *testing.T) {
	doc := &rmdoc.Document{
		Pages: make([]rmdoc.PageRef, 100),
	}
	for i := range doc.Pages {
		doc.Pages[i] = rmdoc.PageRef{
			Index:         i,
			PageID:        fmt.Sprintf("page-%03d", i),
			SourcePDFPage: rmdoc.InsertedPage,
		}
	}

	ctx := &errAfterNContext{Context: context.Background(), after: 2}
	_, err := BuildBackgroundPDF(ctx, doc, BackgroundOptions{})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %T %v", err, err)
	}
}
