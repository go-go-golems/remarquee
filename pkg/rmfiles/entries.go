package rmfiles

import (
	"sort"
	"strings"
	"time"

	"github.com/juruen/rmapi/filetree"
	"github.com/juruen/rmapi/model"
)

// Entry is the JSON-facing representation of a cloud file or folder. It is
// derived from rmapi's model.Node but carries a synthesized path and a
// stable document ID, which the UI uses as the primary key (paths change on
// rename).
type Entry struct {
	ID             string    `json:"id"`
	ParentID       string    `json:"parentId"`
	Name           string    `json:"name"`
	Path           string    `json:"path"`
	IsDir          bool      `json:"isDir"`
	Type           string    `json:"type"`
	Version        int       `json:"version"`
	ModifiedClient string    `json:"modifiedClient"`
	ModifiedTime   time.Time `json:"modifiedTime"`
}

// BuildPath synthesizes a remote path ("/A/B/name") for a node by walking its
// parent pointers. This mirrors the CLI's buildPathFromParents so the UI and
// CLI agree on paths.
func BuildPath(n *model.Node) string {
	if n == nil {
		return ""
	}
	if n.IsRoot() {
		return "/"
	}

	parts := []string{n.Name()}
	for cur := n.Parent; cur != nil && !cur.IsRoot(); cur = cur.Parent {
		parts = append(parts, cur.Name())
	}
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return "/" + strings.Join(parts, "/")
}

// nodeToEntry converts a node to an Entry.
func nodeToEntry(n *model.Node) Entry {
	e := Entry{
		ID:             n.Id(),
		Name:           n.Name(),
		Path:           BuildPath(n),
		IsDir:          n.IsDirectory(),
		Type:           n.Document.Type,
		Version:        n.Version(),
		ModifiedClient: n.Document.ModifiedClient,
	}
	if n.Parent != nil {
		e.ParentID = n.Parent.Id()
	}
	if t, err := n.LastModified(); err == nil {
		e.ModifiedTime = t
	}
	return e
}

// WalkEntries returns every visible node in the tree as a flat list, parents
// before children, sorted by path. The caller is expected to hold the service
// lock when the tree may be mutated concurrently.
func WalkEntries(root *model.Node, includeTemplates bool) []Entry {
	out := make([]Entry, 0, 64)
	var walk func(n *model.Node)
	walk = func(n *model.Node) {
		children := make([]*model.Node, 0, len(n.Children))
		for _, c := range n.Children {
			children = append(children, c)
		}
		sort.Slice(children, func(i, j int) bool { return children[i].Name() < children[j].Name() })
		for _, c := range children {
			if c.Id() == filetree.TrashID {
				continue
			}
			if !includeTemplates && c.Document.Type == model.TemplateType {
				continue
			}
			out = append(out, nodeToEntry(c))
			walk(c)
		}
	}
	if root != nil {
		walk(root)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Path == out[j].Path {
			return out[i].Name < out[j].Name
		}
		return out[i].Path < out[j].Path
	})
	return out
}

// ChildrenEntries returns the direct children of a directory node.
func ChildrenEntries(n *model.Node, includeTemplates bool) []Entry {
	out := make([]Entry, 0, len(n.Children))
	for _, c := range n.Children {
		if c.Id() == filetree.TrashID {
			continue
		}
		if !includeTemplates && c.Document.Type == model.TemplateType {
			continue
		}
		out = append(out, nodeToEntry(c))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

// normalizePath trims whitespace and trailing slashes and ensures a leading
// slash. "" and "//" become "/".
func normalizePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	for len(p) > 1 && strings.HasSuffix(p, "/") {
		p = strings.TrimSuffix(p, "/")
	}
	return p
}
