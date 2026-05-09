package rmcloud

import (
	"strings"

	"github.com/juruen/rmapi/api"
	"github.com/juruen/rmapi/model"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
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
	originalDirPath := dirPath
	dirPath = normalizeDirPath(dirPath)
	log.Debug().Str("input_dir_path", originalDirPath).Str("normalized_dir_path", dirPath).Msg("rmcloud mkdir-all: ensure remote directory")
	if dirPath == "/" {
		return apiCtx.Filetree().Root(), nil
	}

	// Fast path: exists already.
	if existing, err := apiCtx.Filetree().NodeByPath(dirPath, nil); err == nil {
		if existing.IsFile() {
			return nil, errors.Errorf("remote path is a file (expected directory): %s", dirPath)
		}
		log.Debug().Str("dir_path", dirPath).Str("node_id", existing.Id()).Msg("rmcloud mkdir-all: directory already exists")
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

		log.Debug().Str("entry", entry).Str("parent_name", current.Name()).Str("parent_id", parentId).Msg("rmcloud mkdir-all: creating remote directory")
		doc, err := apiCtx.CreateDir(parentId, entry, true)
		if err != nil {
			log.Error().Err(err).Str("entry", entry).Str("parent_name", current.Name()).Str("parent_id", parentId).Msg("rmcloud mkdir-all: create remote directory failed")
			return nil, errors.Wrapf(err, "failed to create remote directory %q under %q", entry, current.Name())
		}
		log.Debug().Str("entry", entry).Str("document_id", doc.ID).Str("document_name", doc.Name).Str("parent_id", doc.Parent).Msg("rmcloud mkdir-all: remote directory created")
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
