package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/go-go-golems/remarquee/pkg/rmdoc"
)

func main() {
	var (
		rmdocPath = flag.String("rmdoc", "", "path to .rmdoc")
		page      = flag.Int("page", 1, "1-based page number")
	)
	flag.Parse()
	if *rmdocPath == "" {
		fmt.Fprintln(os.Stderr, "--rmdoc is required")
		os.Exit(2)
	}
	if *page < 1 {
		fmt.Fprintln(os.Stderr, "--page must be >= 1")
		os.Exit(2)
	}

	ctx := context.Background()
	doc, err := rmdoc.OpenFile(ctx, *rmdocPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open rmdoc: %v\n", err)
		os.Exit(1)
	}
	idx := *page - 1
	if idx < 0 || idx >= len(doc.Pages) {
		fmt.Fprintf(os.Stderr, "page %d out of range (pages=%d)\n", *page, len(doc.Pages))
		os.Exit(1)
	}

	pageID := doc.Pages[idx].PageID
	if pageID == "" {
		fmt.Fprintln(os.Stderr, "page has no pageID")
		os.Exit(1)
	}

	rm, ok, err := rmdoc.ReadRMFileFromArchive(ctx, *rmdocPath, pageID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read rm: %v\n", err)
		os.Exit(1)
	}
	if !ok {
		fmt.Fprintln(os.Stderr, "rm file not found in archive")
		os.Exit(1)
	}

	tree, err := rmdoc.ParseRMV6SceneTree(bytesReader(rm.Bytes))
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse scene tree: %v\n", err)
		os.Exit(1)
	}

	strokes, err := rmdoc.ExtractRMV6StrokesWithAnchors(tree)
	if err != nil {
		fmt.Fprintf(os.Stderr, "extract strokes: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("page=%d pageID=%s\n", *page, pageID)

	bboxAnchored, ok := rmdoc.BBoxForStrokes(strokes, 0)
	if !ok {
		fmt.Println("no strokes")
		return
	}
	fmt.Printf("bbox (anchored) min=(%.2f, %.2f) max=(%.2f, %.2f)\n", bboxAnchored.MinX, bboxAnchored.MinY, bboxAnchored.MaxX, bboxAnchored.MaxY)

	bboxUnanchored, err := rmdoc.BBoxForRMV6SceneTree(tree, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bbox unanchored: %v\n", err)
		os.Exit(1)
	}
	if !bboxUnanchored.IsEmpty() {
		fmt.Printf("bbox (unanchored) min=(%.2f, %.2f) max=(%.2f, %.2f)\n", bboxUnanchored.MinX, bboxUnanchored.MinY, bboxUnanchored.MaxX, bboxUnanchored.MaxY)
	}
}

func bytesReader(b []byte) *bytes.Reader {
	return bytes.NewReader(b)
}
