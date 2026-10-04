package rmfiles

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/juruen/rmapi/api"
	"github.com/juruen/rmapi/filetree"
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
	err := s.mutate(func(c api.ApiCtx) (bool, error) {
		parent, err := s.resolveDirLocked(parentPath)
		if err != nil {
			return false, err
		}
		if existing, err := parent.FindByName(name); err == nil && existing != nil {
			return false, ErrConflict
		}
		doc, err := c.CreateDir(parent.Id(), name, true)
		if err != nil {
			return true, errors.Wrap(err, "create folder")
		}
		c.Filetree().AddDocument(doc)
		entry = nodeToEntry(c.Filetree().NodeById(doc.ID))
		return true, nil
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
	err := s.mutate(func(c api.ApiCtx) (bool, error) {
		node, err := s.nodeByIDLocked(id)
		if err != nil {
			return false, err
		}
		parent := node.Parent
		if parent == nil {
			return false, ErrInvalid
		}
		if sib, err := parent.FindByName(newName); err == nil && sib.Id() != node.Id() {
			return false, ErrConflict
		}
		n, err := c.MoveEntry(node, parent, newName)
		if err != nil {
			return true, errors.Wrap(err, "rename entry")
		}
		c.Filetree().MoveNode(node, n)
		entry = nodeToEntry(node)
		return true, nil
	})
	return entry, err
}

// Move moves an entry into destDirPath, keeping its name.
func (s *Service) Move(id, destDirPath string) (Entry, error) {
	var entry Entry
	err := s.mutate(func(c api.ApiCtx) (bool, error) {
		node, err := s.nodeByIDLocked(id)
		if err != nil {
			return false, err
		}
		dest, err := s.resolveDirLocked(destDirPath)
		if err != nil {
			return false, err
		}
		if isSubdir(node, dest) {
			return false, errors.Wrap(ErrInvalid, "cannot move an entry into itself")
		}
		if sib, err := dest.FindByName(node.Name()); err == nil && sib.Id() != node.Id() {
			return false, ErrConflict
		}
		n, err := c.MoveEntry(node, dest, node.Name())
		if err != nil {
			return true, errors.Wrap(err, "move entry")
		}
		c.Filetree().MoveNode(node, n)
		entry = nodeToEntry(node)
		return true, nil
	})
	return entry, err
}

// deletePlan is a validated, deduplicated removal plan. IDs are in postorder
// (every descendant before its ancestors) so executing it children-first
// removes every selected cloud record, which the pinned DeleteEntry never
// does: its recursive flag only bypasses the non-empty check and HashTree
// removes exactly one record.
type deletePlan struct {
	roots int           // distinct selected root entries
	nodes []*model.Node // postorder, unique, children before parents
}

// planDeleteLocked validates the selection against live, reachable nodes and
// builds the removal plan. It must be called before any side effect so an
// invalid request cannot partially delete. Callers must hold s.mu.
func (s *Service) planDeleteLocked(ids []string, recursive bool) (*deletePlan, error) {
	if len(ids) == 0 {
		return nil, ErrInvalid
	}
	plan := &deletePlan{}
	roots := make([]*model.Node, 0, len(ids))
	seenRoots := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" || id == filetree.TrashID {
			return nil, errors.Wrapf(ErrInvalid, "cannot delete protected entry %q", id)
		}
		node, err := s.nodeByIDLocked(id) // live + reachable
		if err != nil {
			return nil, err
		}
		if _, dup := seenRoots[id]; dup {
			continue
		}
		seenRoots[id] = struct{}{}
		plan.roots++
		if !recursive && node.IsDirectory() && len(node.Children) > 0 {
			return nil, errors.Wrapf(ErrInvalid, "folder %q is not empty and recursive was not requested", node.Name())
		}
		roots = append(roots, node)
	}
	// Expand each distinct selected root into a unique postorder ID plan:
	// every descendant is planned before its ancestors, and nodes shared by
	// overlapping selections are planned exactly once. Children are visited
	// in ID order so the plan (and its failure boundaries) is deterministic
	// despite Go's randomized map iteration order.
	planned := make(map[string]struct{})
	var expand func(n *model.Node)
	expand = func(n *model.Node) {
		if _, dup := planned[n.Id()]; dup {
			return
		}
		children := make([]*model.Node, 0, len(n.Children))
		for _, c := range n.Children {
			if c.Id() == filetree.TrashID {
				continue
			}
			children = append(children, c)
		}
		sort.Slice(children, func(i, j int) bool { return children[i].Id() < children[j].Id() })
		for _, c := range children {
			expand(c)
		}
		planned[n.Id()] = struct{}{}
		plan.nodes = append(plan.nodes, n)
	}
	for _, node := range roots {
		expand(node)
	}
	return plan, nil
}

// Delete removes the selected entries and, for folders, every descendant
// record. The pinned DeleteEntry only removes the single record it is given
// regardless of its recursive flag, so this expands the selection into an
// explicit children-before-parent plan and deletes every record. IDs are
// validated and deduplicated and the nonrecursive non-empty guard is applied
// before any side effect; a preflight failure removes nothing. The returned
// count is individual removed cloud records (selected roots may expand to
// more); a failure after some records were removed returns a
// *PartialDeleteError with per-outcome data, not a rollback claim.
func (s *Service) Delete(ids []string, recursive bool) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.apiCtx == nil {
		return 0, ErrInvalid
	}
	plan, err := s.planDeleteLocked(ids, recursive)
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, node := range plan.nodes {
		if err := s.apiCtx.DeleteEntry(node, true, true); err != nil {
			delErr := errors.Wrapf(err, "delete %q", node.Name())
			// Local edges may already have changed (removed descendants);
			// keep the index consistent with the tree.
			s.reindexLocked()
			// The cloud method commits independently: the failed record may
			// actually be gone (side effect before error). Reconcile with
			// authoritative cloud state and report the true outcome.
			reconcileErr := s.reconcileLocked()
			// After reconciliation (or, if it failed, against current local
			// edges), recompute which planned records are really gone.
			removedNow, remaining := s.outcomeForPlannedLocked(plan.nodes)
			partial := &PartialDeleteError{
				Removed:      removedNow,
				RemainingIDs: remaining,
				FailedID:     node.Id(),
				Err:          delErr,
				ReconcileErr: reconcileErr,
			}
			return removedNow, partial
		}
		s.apiCtx.Filetree().DeleteNode(node)
		removed++
	}
	s.reindexLocked()
	if err := s.apiCtx.SyncComplete(); err != nil {
		// Record removal is already committed. Notification failure must
		// preserve that outcome rather than become a bare generic error.
		reconcileErr := s.reconcileLocked()
		removedNow, remaining := s.outcomeForPlannedLocked(plan.nodes)
		return removedNow, &PartialDeleteError{
			Removed: removedNow, RemainingIDs: remaining,
			Err: errors.Wrap(err, "sync cloud tree"), ReconcileErr: reconcileErr,
		}
	}
	return removed, nil
}

// outcomeForPlannedLocked reports the real outcome for the planned records
// against the current tree: how many are gone (no longer reachable) and which
// IDs remain, in plan order. Stale node pointers are only used for their IDs.
func (s *Service) outcomeForPlannedLocked(planned []*model.Node) (int, []string) {
	removed := 0
	var remaining []string
	for _, node := range planned {
		cur := s.apiCtx.Filetree().NodeById(node.Id())
		if cur == nil || !s.reachableByEdgesLocked(cur) {
			removed++
			continue
		}
		remaining = append(remaining, node.Id())
	}
	return removed, remaining
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
