package rmcloud

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/juruen/rmapi/util"
	"github.com/pkg/errors"
)

// DownloadedDocument describes a remote document fetched to a local .rmdoc file.
type DownloadedDocument struct {
	RemotePath string
	LocalPath  string
	DocumentID string
	Name       string
}

// DownloadDocumentByPath downloads a remote document to outDir and returns the local path.
func DownloadDocumentByPath(ctx context.Context, auth AuthSettings, remotePath string, outDir string) (*DownloadedDocument, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	remotePath = strings.TrimSpace(remotePath)
	if remotePath == "" {
		return nil, errors.New("remote path is empty")
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, errors.Wrap(err, "ensure output dir")
	}

	_, apiCtx, err := CreateApiCtx(auth)
	if err != nil {
		return nil, err
	}

	node, err := apiCtx.Filetree().NodeByPath(remotePath, nil)
	if err != nil || node.IsDirectory() {
		return nil, errors.New("file doesn't exist")
	}

	localPath := filepath.Join(outDir, fmt.Sprintf("%s.%s", node.Name(), util.RMDOC))
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := apiCtx.FetchDocument(node.Document.ID, localPath); err != nil {
		return nil, errors.Wrapf(err, "failed to download %s", remotePath)
	}

	return &DownloadedDocument{
		RemotePath: remotePath,
		LocalPath:  localPath,
		DocumentID: node.Document.ID,
		Name:       node.Name(),
	}, nil
}
