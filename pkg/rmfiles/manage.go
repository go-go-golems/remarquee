package rmfiles

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/juruen/rmapi/api"
	"github.com/juruen/rmapi/model"
	"github.com/pkg/errors"
)

// CreateFolder creates a directory named name under parentPath.
func (s *Service) CreateFolder(parentPath, name string) (Entry, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsAny(name, "/\\") {
		return Entry{}, ErrInvalid
	}
	var entry Entry
	err := s.mutate(func(c api.ApiCtx) error {
		parent, err := s.resolveDirLocked(parentPath)
		if err != nil {
			return err
		}
		if existing, err := parent.FindByName(name); err == nil && existing != nil {
			return ErrConflict
		}
		doc, err := c.CreateDir(parent.Id(), name, true)
		if err != nil {
			return errors.Wrap(err, "create folder")
		}
		c.Filetree().AddDocument(doc)
		entry = nodeToEntry(c.Filetree().NodeById(doc.ID))
		return nil
	})
	return entry, err
}

// Rename renames an entry in place (same parent).
func (s *Service) Rename(id, newName string) (Entry, error) {
	newName = strings.TrimSpace(newName)
	if newName == "" || strings.ContainsAny(newName, "/\\") {
		return Entry{}, ErrInvalid
	}
	var entry Entry
	err := s.mutate(func(c api.ApiCtx) error {
		node, err := s.nodeByIDLocked(id)
		if err != nil {
			return err
		}
		parent := node.Parent
		if parent == nil {
			return ErrInvalid
		}
		if sib, err := parent.FindByName(newName); err == nil && sib.Id() != node.Id() {
			return ErrConflict
		}
		n, err := c.MoveEntry(node, parent, newName)
		if err != nil {
			return errors.Wrap(err, "rename entry")
		}
		c.Filetree().MoveNode(node, n)
		entry = nodeToEntry(node)
		return nil
	})
	return entry, err
}

// Move moves an entry into destDirPath, keeping its name.
func (s *Service) Move(id, destDirPath string) (Entry, error) {
	var entry Entry
	err := s.mutate(func(c api.ApiCtx) error {
		node, err := s.nodeByIDLocked(id)
		if err != nil {
			return err
		}
		dest, err := s.resolveDirLocked(destDirPath)
		if err != nil {
			return err
		}
		if isSubdir(node, dest) {
			return errors.Wrap(ErrInvalid, "cannot move an entry into itself")
		}
		if sib, err := dest.FindByName(node.Name()); err == nil && sib.Id() != node.Id() {
			return ErrConflict
		}
		n, err := c.MoveEntry(node, dest, node.Name())
		if err != nil {
			return errors.Wrap(err, "move entry")
		}
		c.Filetree().MoveNode(node, n)
		entry = nodeToEntry(node)
		return nil
	})
	return entry, err
}

// Delete removes entries by ID. Folders require recursive=true when non-empty.
func (s *Service) Delete(ids []string, recursive bool) (int, error) {
	if len(ids) == 0 {
		return 0, ErrInvalid
	}
	deleted := 0
	err := s.mutate(func(c api.ApiCtx) error {
		for _, id := range ids {
			node, err := s.nodeByIDLocked(id)
			if err != nil {
				return err
			}
			if err := c.DeleteEntry(node, recursive, true); err != nil {
				return errors.Wrapf(err, "delete %q", node.Name())
			}
			c.Filetree().DeleteNode(node)
			deleted++
		}
		return nil
	})
	return deleted, err
}

// fixedDownloadStagingName is the internal staging filename used inside the
// private per-download temporary directory. It is deliberately independent of
// cloud display names: cloud metadata may come from other clients and is not
// validated, so it must never influence a local output path.
const fixedDownloadStagingName = "document.rmdoc"

// Download fetches a document as an .rmdoc archive and returns its bytes and a
// suggested filename. The staging path is a fixed name inside a private temp
// directory; the cloud display name only feeds the separately sanitized
// suggested filename.
func (s *Service) Download(id string) ([]byte, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	node, err := s.nodeByIDLocked(id)
	if err != nil {
		return nil, "", err
	}
	if node.IsDirectory() {
		return nil, "", ErrUnsupported
	}
	tmpDir, err := os.MkdirTemp("", "rmfiles-dl-")
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()
	out := filepath.Join(tmpDir, fixedDownloadStagingName)
	if err := s.apiCtx.FetchDocument(id, out); err != nil {
		return nil, "", errors.Wrap(err, "fetch document")
	}
	data, err := os.ReadFile(out)
	if err != nil {
		return nil, "", err
	}
	return data, sanitizeAttachmentName(node.Name()) + ".rmdoc", nil
}

// sanitizeAttachmentName converts untrusted cloud display metadata into a
// safe, single-component suggested filename. It removes path separators,
// quotes and control characters so the result cannot traverse directories or
// corrupt a Content-Disposition header; legitimate Unicode and spaces are
// preserved. Degenerate results (empty, all whitespace, or pure dot
// components) fall back to "document".
func sanitizeAttachmentName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case unicode.IsControl(r):
			// Drop control characters: they can break header parsing.
		case r == '/' || r == '\\' || r == '"':
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	cleaned := b.String()
	if cleaned == "" || cleaned == "." || cleaned == ".." || strings.TrimSpace(cleaned) == "" {
		return "document"
	}
	return cleaned
}

// isSubdir reports whether child is the same node as parent or nested beneath
// it. Used to reject "move into own subtree" operations that would detach data.
func isSubdir(parent, child *model.Node) bool {
	for n := child; n != nil; n = n.Parent {
		if n.Id() == parent.Id() {
			return true
		}
	}
	return false
}
