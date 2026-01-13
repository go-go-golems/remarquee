package rmcloud

import (
	"strings"

	"github.com/juruen/rmapi/api"
	"github.com/juruen/rmapi/model"
	"github.com/pkg/errors"
)

func normalizeDirPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	// Keep "/" as-is, otherwise trim trailing slashes.
	for len(p) > 1 && strings.HasSuffix(p, "/") {
		p = strings.TrimSuffix(p, "/")
	}
	return p
}

// MkdirAll ensures the given remote directory path exists and returns the node for it.
//
// The behavior is similar to `mkdir -p`: intermediate directories are created if missing.
func MkdirAll(apiCtx api.ApiCtx, dirPath string) (*model.Node, error) {
	dirPath = normalizeDirPath(dirPath)
	if dirPath == "/" {
		return apiCtx.Filetree().Root(), nil
	}

	// Fast path: exists already.
	if existing, err := apiCtx.Filetree().NodeByPath(dirPath, nil); err == nil {
		if existing.IsFile() {
			return nil, errors.Errorf("remote path is a file (expected directory): %s", dirPath)
		}
		return existing, nil
	}

	current := apiCtx.Filetree().Root()

	// Split on "/" but keep semantics very close to rmapi's NodeByPath.
	entries := strings.Split(dirPath, "/")
	for _, entry := range entries {
		if entry == "" || entry == "." {
			continue
		}
		if entry == ".." {
			if current.Parent == nil {
				current = apiCtx.Filetree().Root()
			} else {
				current = current.Parent
			}
			continue
		}

		next, err := current.FindByName(entry)
		if err == nil {
			if next.IsFile() {
				return nil, errors.Errorf("remote path segment is a file (expected directory): %s", entry)
			}
			current = next
			continue
		}

		parentId := current.Id()
		if current.IsRoot() {
			parentId = ""
		}

		doc, err := apiCtx.CreateDir(parentId, entry, true)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create remote directory %q under %q", entry, current.Name())
		}
		apiCtx.Filetree().AddDocument(doc)

		// Re-resolve after adding, to ensure we get the correct node.
		next, err = current.FindByName(entry)
		if err != nil {
			return nil, errors.Wrapf(err, "created directory %q but failed to resolve it", entry)
		}
		current = next
	}

	return current, nil
}
