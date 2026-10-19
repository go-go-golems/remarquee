// Offline review probes. Run from the repository root with go run <this file>.
// These report current behavior, not assertions that the behavior is correct.
package main

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-go-golems/remarquee/cmd/remarquee/cmds/serve"
	"github.com/go-go-golems/remarquee/pkg/rmfiles"
	"github.com/juruen/rmapi/api"
	"github.com/juruen/rmapi/api/sync15"
	"github.com/juruen/rmapi/filetree"
	"github.com/juruen/rmapi/model"
)

type fake struct {
	ft  filetree.FileTreeCtx
	seq int
}

var _ api.ApiCtx = (*fake)(nil)

func newFake() *fake { return &fake{ft: filetree.CreateFileTreeCtx()} }
func (f *fake) add(parent, name, typ string) *model.Document {
	f.seq++
	d := &model.Document{ID: fmt.Sprintf("probe-%d", f.seq), Parent: parent, Name: name, Type: typ}
	f.ft.AddDocument(d)
	return d
}
func (f *fake) Filetree() *filetree.FileTreeCtx                { return &f.ft }
func (f *fake) SyncComplete() error                            { return nil }
func (f *fake) Nuke() error                                    { return nil }
func (f *fake) Refresh() (string, int64, error)                { return "", 0, nil }
func (f *fake) ReplaceDocumentFile(string, string, bool) error { return nil }
func (f *fake) FetchDocument(_, p string) error                { return os.WriteFile(p, []byte("fake archive"), 0600) }
func (f *fake) CreateDir(p, n string, _ bool) (*model.Document, error) {
	return f.add(p, n, model.DirectoryType), nil
}
func (f *fake) UploadDocument(p, src string, _ bool, _, _, _ *int, _ *string) (*model.Document, error) {
	f.seq++
	n := filepath.Base(src)
	n = strings.TrimSuffix(n, filepath.Ext(n))
	return &model.Document{ID: fmt.Sprintf("probe-%d", f.seq), Parent: p, Name: n, Type: model.DocumentType}, nil
}
func (f *fake) MoveEntry(src, dst *model.Node, n string) (*model.Node, error) {
	d := *src.Document
	d.Name = n
	d.Parent = dst.Id()
	return &model.Node{Document: &d, Parent: dst}, nil
}
func (f *fake) DeleteEntry(*model.Node, bool, bool) error { return nil }
func must(err error) {
	if err != nil {
		panic(err)
	}
}
func contains(es []rmfiles.Entry, id string) bool {
	for _, e := range es {
		if e.ID == id {
			return true
		}
	}
	return false
}
func wait(s *rmfiles.Service, id string) rmfiles.UploadJob {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		j, ok := s.GetUpload(id)
		if ok && (j.State == rmfiles.JobDone || j.State == rmfiles.JobFailed) {
			return j
		}
		time.Sleep(time.Millisecond)
	}
	panic("fake job did not finish")
}
func main() {
	// Origin/Host are supplied by a browser, not authenticated by the mux.
	f := newFake()
	s := rmfiles.NewService(f, rmfiles.Config{})
	h := serve.NewServer(s, serve.Options{}).Handler()
	r := httptest.NewRequest("POST", "http://127.0.0.1:8080/api/folders", bytes.NewBufferString(`{"parentPath":"/","name":"foreign-origin"}`))
	r.Header.Set("Origin", "https://attacker.invalid")
	r.Header.Set("Sec-Fetch-Site", "cross-site")
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	fmt.Printf("foreign Origin + text/plain create-folder: HTTP %d\n", w.Code)
	// Legitimate root should not be mistaken for an omitted destination.
	f = newFake()
	f.add("", "ai", model.DirectoryType)
	s = rmfiles.NewService(f, rmfiles.Config{DefaultRemoteDir: "/ai"})
	id, err := s.StartUpload([]rmfiles.UploadInput{{Name: "root.pdf", Data: []byte("fake")}}, "/")
	must(err)
	j := wait(s, id)
	fmt.Printf("explicit root upload with default /ai: dest=%s state=%s\n", j.DestDir, j.State)
	// Two source names map to the same sanitized document name.
	id, err = s.StartUpload([]rmfiles.UploadInput{{Name: "a b.pdf", Data: []byte("fake")}, {Name: "a_b.pdf", Data: []byte("fake")}}, "/ai")
	must(err)
	wait(s, id)
	es, err := s.List("/ai")
	must(err)
	count := 0
	for _, e := range es {
		if e.Name == "a_b" {
			count++
		}
	}
	fmt.Printf("sanitized upload collisions: %d sibling documents named a_b\n", count)
	// A batch can mutate the tree and then fail before reindexing.
	f = newFake()
	d := f.add("", "gone", model.DocumentType)
	s = rmfiles.NewService(f, rmfiles.Config{})
	n, err := s.Delete([]string{d.ID, "missing"}, false)
	es, listErr := s.List("/")
	must(listErr)
	fmt.Printf("partial delete: deleted=%d error=%v listContains=%t indexContains=%t\n", n, err, contains(es, d.ID), contains(s.Entries(), d.ID))
	// ID lookup survives DeleteNode in the pinned filetree implementation.
	res, err := s.Download(d.ID)
	if res != nil {
		defer func() { _ = res.Close() }()
	}
	dlBytes := int64(0)
	if err == nil {
		dlBytes = res.Size()
	}
	fmt.Printf("download deleted ID using fake storage: bytes=%d error=%v\n", dlBytes, err)
	// Directly exercise the actual pinned hash-tree Remove, with no transport.
	ht := sync15.HashTree{SchemaVersion: sync15.SchemaVersionV4, Docs: []*sync15.BlobDoc{
		sync15.NewBlobDoc("folder", "parent", model.DirectoryType, ""), sync15.NewBlobDoc("child", "child", model.DocumentType, "parent")}}
	for _, doc := range ht.Docs {
		doc.Hash = strings.Repeat("0", 64) // Valid placeholder shape; no blob transport is used.
	}
	must(ht.Remove("parent"))
	_, err = ht.FindDoc("child")
	fmt.Printf("pinned HashTree.Remove(parent): remaining=%d childStillPresent=%t\n", len(ht.Docs), err == nil)
	rebuilt := sync15.DocumentsFileTree(&ht)
	child := rebuilt.NodeById("child")
	fmt.Printf("rebuild after removing parent: child path=%s\n", rmfiles.BuildPath(child))
	// Verify successful recursive service deletion still leaves ID lookup alive.
	f = newFake()
	p := f.add("", "folder", model.DirectoryType)
	c := f.add(p.ID, "child", model.DocumentType)
	s = rmfiles.NewService(f, rmfiles.Config{})
	_, err = s.Delete([]string{p.ID}, true)
	must(err)
	fmt.Printf("after recursive service delete: childIDLookup=%t visibleIndex=%d\n", f.ft.NodeById(c.ID) != nil, len(s.Entries()))
	// Path-shaped names make created folders inaccessible by their own display path.
	_, err = s.CreateFolder("/", "..")
	fmt.Printf("create folder named '..': error=%v\n", err)
	// Also show decoder accepts trailing JSON, independent of media type.
	r = httptest.NewRequest("POST", "/api/folders", bytes.NewBufferString(`{"parentPath":"/","name":"trailing-json"} {"ignored":true}`))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	fmt.Printf("create-folder with trailing JSON: HTTP %d\n", w.Code)
}
