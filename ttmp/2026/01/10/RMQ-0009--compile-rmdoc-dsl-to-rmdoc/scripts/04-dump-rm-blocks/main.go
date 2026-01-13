package main

// Usage:
//   go run ./ttmp/2026/01/10/RMQ-0009--compile-rmdoc-dsl-to-rmdoc/scripts/04-dump-rm-blocks/main.go \
//     --rmdoc /path/to/file.rmdoc

import (
	"bytes"
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"io"

	"github.com/go-go-golems/remarquee/pkg/rmdoc"
)

type blockInfo struct {
	Count int
	MinV  uint8
	CurV  uint8
}

func main() {
	rmdocPath := flag.String("rmdoc", "", "Path to .rmdoc")
	flag.Parse()

	if *rmdocPath == "" {
		fmt.Println("--rmdoc is required")
		return
	}

	ctx := context.Background()
	doc, err := rmdoc.OpenFile(ctx, *rmdocPath)
	if err != nil {
		fmt.Printf("open rmdoc: %v\n", err)
		return
	}
	if len(doc.Pages) == 0 {
		fmt.Println("no pages in rmdoc")
		return
	}

	rm, ok, err := rmdoc.ReadRMFileFromArchive(ctx, *rmdocPath, doc.Pages[0].PageID)
	if err != nil || !ok {
		fmt.Printf("read rm: ok=%v err=%v\n", ok, err)
		return
	}

	blocks, err := scanBlocks(bytes.NewReader(rm.Bytes))
	if err != nil {
		fmt.Printf("scan blocks: %v\n", err)
		return
	}

	fmt.Printf("rmdoc=%s page=%s blocks=%d\n", *rmdocPath, doc.Pages[0].PageID, len(blocks))
	for typ, info := range blocks {
		fmt.Printf("  type=0x%02X name=%s count=%d min=%d cur=%d\n", typ, blockTypeName(typ), info.Count, info.MinV, info.CurV)
	}
}

func scanBlocks(r io.ReadSeeker) (map[uint8]blockInfo, error) {
	if err := readHeader(r); err != nil {
		return nil, err
	}

	out := map[uint8]blockInfo{}
	for {
		length, err := readUint32LE(r)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		unknown, err := readUint8(r)
		if err != nil {
			return nil, err
		}
		_ = unknown
		minV, err := readUint8(r)
		if err != nil {
			return nil, err
		}
		curV, err := readUint8(r)
		if err != nil {
			return nil, err
		}
		blockType, err := readUint8(r)
		if err != nil {
			return nil, err
		}

		if _, err := r.Seek(int64(length), io.SeekCurrent); err != nil {
			return nil, err
		}

		info := out[blockType]
		info.Count++
		info.MinV = minV
		info.CurV = curV
		out[blockType] = info
	}
	return out, nil
}

func readHeader(r io.Reader) error {
	head := make([]byte, len(rmv6Header))
	if _, err := io.ReadFull(r, head); err != nil {
		return err
	}
	if !bytes.Equal(head, rmv6Header) {
		return fmt.Errorf("wrong header: %q", string(head))
	}
	return nil
}

func readUint8(r io.Reader) (uint8, error) {
	var b [1]byte
	_, err := io.ReadFull(r, b[:])
	return b[0], err
}

func readUint32LE(r io.Reader) (uint32, error) {
	var out uint32
	if err := binary.Read(r, binary.LittleEndian, &out); err != nil {
		if err == io.EOF {
			return 0, io.EOF
		}
		return 0, err
	}
	return out, nil
}

func blockTypeName(t uint8) string {
	switch t {
	case 0x00:
		return "MigrationInfo"
	case 0x01:
		return "SceneTree"
	case 0x02:
		return "TreeNode"
	case 0x03:
		return "SceneGlyphItem"
	case 0x04:
		return "SceneGroupItem"
	case 0x05:
		return "SceneLineItem"
	case 0x06:
		return "SceneTextItem"
	case 0x07:
		return "RootText"
	case 0x08:
		return "SceneTombstone"
	case 0x0A:
		return "PageInfo"
	default:
		return "Unknown"
	}
}

var rmv6Header = []byte("reMarkable .lines file, version=6          ")
