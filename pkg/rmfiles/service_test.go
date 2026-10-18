package rmfiles

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/juruen/rmapi/api"
	"github.com/juruen/rmapi/api/sync15"
	"github.com/juruen/rmapi/filetree"
	"github.com/juruen/rmapi/model"
)

// fakeApiCtx is a minimal in-memory implementation of api.ApiCtx for tests.
type fakeApiCtx struct {
	ft        filetree.FileTreeCtx
	nextID    int
	syncCount int
	moved     int
	deleted   int
	// lastFetchPath records the destination path of the last FetchDocument so
	// tests can prove staging destinations are independent of cloud names.
	lastFetchPath string
	// failFetch makes the next FetchDocument calls fail, to exercise error
	// paths without a network or real filesystem escapes.
	failFetch bool
}

var _ api.ApiCtx = (*fakeApiCtx)(nil)

func newFake() *fakeApiCtx {
	f := &fakeApiCtx{ft: filetree.CreateFileTreeCtx()}
	return f
}

func (f *fakeApiCtx) id() string {
	f.nextID++
	return "id-" + itoa(f.nextID)
}

func (f *fakeApiCtx) add(parentID, name, typ string) *model.Document {
	doc := &model.Document{ID: f.id(), Name: name, Type: typ, Parent: parentID}
	f.ft.AddDocument(doc)
	return doc
}

func (f *fakeApiCtx) Filetree() *filetree.FileTreeCtx { return &f.ft }
func (f *fakeApiCtx) SyncComplete() error             { f.syncCount++; return nil }
func (f *fakeApiCtx) Nuke() error                     { return nil }
func (f *fakeApiCtx) Refresh() (string, int64, error) { return "", 0, nil }
func (f *fakeApiCtx) ReplaceDocumentFile(string, string, bool) error {
	return nil
}
func (f *fakeApiCtx) FetchDocument(docId, dstPath string) error {
	f.lastFetchPath = dstPath
	if f.failFetch {
		return errors.New("injected fetch failure")
	}
	return os.WriteFile(dstPath, []byte("rmdoc-bytes"), 0o600)
}

func (f *fakeApiCtx) CreateDir(parentId, name string, _ bool) (*model.Document, error) {
	return f.add(parentId, name, model.DirectoryType), nil
}

func (f *fakeApiCtx) UploadDocument(parentId, sourceDocPath string, _ bool, _, _, _ *int, _ *string) (*model.Document, error) {
	stem := filepath.Base(sourceDocPath)
	stem = stem[:len(stem)-len(filepath.Ext(stem))]
	return &model.Document{ID: f.id(), Name: stem, Type: model.DocumentType, Parent: parentId}, nil
}

func (f *fakeApiCtx) MoveEntry(src, dstDir *model.Node, name string) (*model.Node, error) {
	f.moved++
	doc := *src.Document
	doc.Name = name
	doc.Parent = dstDir.Id()
	return &model.Node{Document: &doc, Parent: dstDir}, nil
}

func (f *fakeApiCtx) DeleteEntry(node *model.Node, _, _ bool) error {
	f.deleted++
	return nil
}

// tree builds a small tree and returns the fake plus useful IDs.
func tree(t *testing.T) (*fakeApiCtx, string, string, string) {
	t.Helper()
	f := newFake()
	ai := f.add("", "ai", model.DirectoryType)
	notes := f.add(ai.ID, "Notes", model.DirectoryType)
	doc := f.add(notes.ID, "Project Plan", model.DocumentType)
	// a template document that must be hidden by default
	f.add(notes.ID, "hidden template", model.TemplateType)
	return f, ai.ID, notes.ID, doc.ID
}

func TestWalkEntriesPathsAndFiltering(t *testing.T) {
	f, _, notesID, docID := tree(t)
	entries := WalkEntries(f.ft.Root(), false)
	byID := map[string]Entry{}
	for _, e := range entries {
		byID[e.ID] = e
	}
	if _, ok := byID[notesID]; !ok {
		t.Fatalf("expected notes folder in entries: %+v", entries)
	}
	if got := byID[docID].Path; got != "/ai/Notes/Project Plan" {
		t.Fatalf("path = %q, want /ai/Notes/Project Plan", got)
	}
	for _, e := range entries {
		if e.Name == "hidden template" {
			t.Fatalf("template should be hidden by default")
		}
	}
	if with := WalkEntries(f.ft.Root(), true); len(with) != len(entries)+1 {
		t.Fatalf("include-templates length = %d, want %d", len(with), len(entries)+1)
	}
}

func TestSearchRanksNamePrefixFirst(t *testing.T) {
	f, _, _, _ := tree(t)
	entries := WalkEntries(f.ft.Root(), false)
	got := Search(entries, "proj", 10)
	if len(got) == 0 {
		t.Fatal("expected a match for proj")
	}
	if got[0].Entry.Name != "Project Plan" {
		t.Fatalf("top result = %q, want Project Plan", got[0].Entry.Name)
	}
	if len(Search(entries, "", 10)) != len(entries) {
		t.Fatal("empty query should return all entries")
	}
}

func TestCreateFolderAndReindex(t *testing.T) {
	f, _, notesID, _ := tree(t)
	s := NewService(f, Config{})
	if _, err := s.CreateFolder("/ai/Notes", "Ideas"); err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	if got := len(s.Entries()); got != 4 { // ai, Notes, Project Plan, Ideas
		t.Fatalf("entries = %d, want 4", got)
	}
	// duplicate must conflict
	if _, err := s.CreateFolder("/ai/Notes", "Ideas"); err != ErrConflict {
		t.Fatalf("duplicate error = %v, want ErrConflict", err)
	}
	_ = notesID
}

func TestRenameAndMove(t *testing.T) {
	f, aiID, _, docID := tree(t)
	s := NewService(f, Config{})
	renamed, err := s.Rename(docID, "Renamed Plan")
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if renamed.Name != "Renamed Plan" {
		t.Fatalf("renamed name = %q", renamed.Name)
	}
	// rename to a sibling's name conflicts
	if _, err := s.Rename(docID, "hidden template"); err != ErrConflict {
		t.Fatalf("conflict error = %v, want ErrConflict", err)
	}
	moved, err := s.Move(docID, "/")
	if err != nil {
		t.Fatalf("Move: %v", err)
	}
	if moved.ParentID != "" {
		t.Fatalf("moved parent = %q, want root", moved.ParentID)
	}
	_ = aiID
}

func TestMoveIntoSubtreeRejected(t *testing.T) {
	f, _, notesID, _ := tree(t)
	s := NewService(f, Config{})
	// ai is not in the fixture return; resolve it.
	ai, err := s.List("/")
	if err != nil {
		t.Fatal(err)
	}
	var aiID string
	for _, e := range ai {
		if e.Name == "ai" {
			aiID = e.ID
		}
	}
	if _, err := s.Move(aiID, "/ai/Notes"); err == nil {
		t.Fatalf("moving a folder into its own subtree should fail")
	}
	_ = notesID
}

func TestDelete(t *testing.T) {
	f, _, _, docID := tree(t)
	s := NewService(f, Config{})
	n, err := s.Delete([]string{docID}, false)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if n != 1 || f.deleted != 1 {
		t.Fatalf("deleted = %d (fake %d), want 1", n, f.deleted)
	}
	if _, err := s.Delete(nil, false); err != ErrInvalid {
		t.Fatalf("empty delete error = %v, want ErrInvalid", err)
	}
}

// TestDownloadStagingPathIsFixed proves a traversal-shaped, control-laden
// cloud display name (as imported by another client) can never influence the
// local FetchDocument destination or the suggested filename, and that the
// private staging directory is removed afterwards.
func TestDownloadStagingPathIsFixed(t *testing.T) {
	f, _, _, docID := tree(t)
	evil := f.add("", `..\..\evil "quoted"`+"\x07/slash", model.DocumentType)
	s := NewService(f, Config{})
	data, name, err := s.Download(evil.ID)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if string(data) != "rmdoc-bytes" {
		t.Fatalf("download bytes = %q", data)
	}
	if f.lastFetchPath == "" {
		t.Fatal("FetchDocument was never called")
	}
	if got := filepath.Base(f.lastFetchPath); got != "document.rmdoc" {
		t.Fatalf("staging filename = %q, want fixed document.rmdoc", got)
	}
	// The private staging directory must be cleaned up after the call.
	stagingDir := filepath.Dir(f.lastFetchPath)
	if _, err := os.Stat(stagingDir); !os.IsNotExist(err) {
		t.Fatalf("staging dir %s still exists after Download (err=%v)", stagingDir, err)
	}
	// The suggested filename must be a safe single component.
	if !strings.HasSuffix(name, ".rmdoc") {
		t.Fatalf("suggested name %q lacks .rmdoc suffix", name)
	}
	if strings.ContainsAny(name, `/\"`) {
		t.Fatalf("suggested name %q contains separators or quotes", name)
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			t.Fatalf("suggested name %q contains control character %q", name, r)
		}
	}
	base := strings.TrimSuffix(name, ".rmdoc")
	if base == "" || base == "." || base == ".." {
		t.Fatalf("suggested base name %q is a path component hazard", base)
	}
	if name == evil.Name+".rmdoc" {
		t.Fatal("suggested name still equals the raw cloud name")
	}
	// Normal names must still round-trip as the suggestion.
	_, normalName, err := s.Download(docID)
	if err != nil {
		t.Fatalf("Download normal: %v", err)
	}
	if normalName != "Project Plan.rmdoc" {
		t.Fatalf("normal suggested name = %q, want Project Plan.rmdoc", normalName)
	}
}

// TestSanitizeAttachmentName pins the exact sanitization contract for
// untrusted cloud names.
func TestSanitizeAttachmentName(t *testing.T) {
	cases := map[string]string{
		"../../etc/passwd": ".._.._etc_passwd",
		"..\\..\\win":      ".._.._win",
		`a"b`:              "a_b",
		"ctl\x07name":      "ctlname",
		"del\x7fete":       "delete",
		"":                 "document",
		"   ":              "document",
		"..":               "document",
		".":                "document",
		"unicode é中":       "unicode é中",
		"sp ace.rmdoc":     "sp ace.rmdoc",
		"/":                "_",
		"\\":               "_",
	}
	for in, want := range cases {
		if got := sanitizeAttachmentName(in); got != want {
			t.Errorf("sanitizeAttachmentName(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestDownloadStagingCleanupOnFetchFailure proves the private staging
// directory is removed even when FetchDocument fails, without any network or
// real path escapes.
func TestDownloadStagingCleanupOnFetchFailure(t *testing.T) {
	f, _, _, docID := tree(t)
	f.failFetch = true
	s := NewService(f, Config{})
	data, name, err := s.Download(docID)
	if err == nil {
		t.Fatalf("Download should fail when FetchDocument fails (data=%q name=%q)", data, name)
	}
	if f.lastFetchPath == "" {
		t.Fatal("FetchDocument was never called")
	}
	stagingDir := filepath.Dir(f.lastFetchPath)
	if _, statErr := os.Stat(stagingDir); !os.IsNotExist(statErr) {
		t.Fatalf("staging dir %s still exists after failed Download (err=%v)", stagingDir, statErr)
	}
}

func TestStatus(t *testing.T) {
	f, _, _, _ := tree(t)
	s := NewService(f, Config{DefaultRemoteDir: "/ai"})
	st := s.Status()
	if !st.Authenticated {
		t.Fatal("expected authenticated")
	}
	if st.DefaultRemoteDir != "/ai" {
		t.Fatalf("default dir = %q", st.DefaultRemoteDir)
	}
}

func TestSanitizeStem(t *testing.T) {
	cases := map[string]string{
		"My Notes":          "My_Notes",
		"a/b:c":             "abc",
		"__weird--":         "weird",
		"":                  "document",
		"2026-10-04 report": "2026-10-04_report",
	}
	for in, want := range cases {
		if got := sanitizeStem(in); got != want {
			t.Errorf("sanitizeStem(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSanitizeRelPath(t *testing.T) {
	cases := map[string]string{
		"folder/sub/file.md": "folder/sub/file.md",
		"a\\b\\c.pdf":        "a/b/c.pdf",
		"./a/../b":           "a/b",
		"../../etc/passwd":   "etc/passwd",
		"folder//double//x":  "folder/double/x",
		"":                   "document",
		"/leading/slash.md":  "leading/slash.md",
	}
	for in, want := range cases {
		if got := sanitizeRelPath(in); got != want {
			t.Errorf("sanitizeRelPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestJoinRemote(t *testing.T) {
	cases := []struct{ base, rel, want string }{
		{"/", ".", "/"},
		{"/ai", "", "/ai"},
		{"/ai", "sub/dir", "/ai/sub/dir"},
		{"", "sub", "/sub"},
		{"/ai/", "/sub/", "/ai/sub"},
	}
	for _, c := range cases {
		if got := joinRemote(c.base, c.rel); got != c.want {
			t.Errorf("joinRemote(%q,%q) = %q, want %q", c.base, c.rel, got, c.want)
		}
	}
}

// ---------- dependency-faithful cloud-record fake ----------

// cloudRec is one authoritative cloud record for cloudFake.
type cloudRec struct{ id, name, parent, typ string }

// cloudFake keeps the authoritative cloud record inventory in a real pinned
// sync15.HashTree. DeleteEntry removes exactly one record from it (matching
// pinned semantics: the recursive flag only bypasses the non-empty check and
// the hash tree removes a single record), and Refresh rebuilds the file tree
// with the real sync15.DocumentsFileTree rebuild. Tests can therefore prove
// absence after a refresh/rebuild rather than only a detached local edge.
type cloudFake struct {
	ht *sync15.HashTree
	ft *filetree.FileTreeCtx

	nextID int

	// failDelete makes DeleteEntry fail without touching the record.
	failDelete map[string]error
	// ghostDelete removes the record first, then fails: a committed side
	// effect followed by an error, the uncertainty reconcile must resolve.
	ghostDelete map[string]error
	failRefresh error

	deletedIDs []string // records actually removed from the inventory, in order
}

var _ api.ApiCtx = (*cloudFake)(nil)

func newCloudFake(recs ...cloudRec) *cloudFake {
	ht := &sync15.HashTree{SchemaVersion: sync15.SchemaVersionV4}
	for _, r := range recs {
		ht.Docs = append(ht.Docs, sync15.NewBlobDoc(r.name, r.id, r.typ, r.parent))
	}
	f := &cloudFake{
		ht:          ht,
		ft:          sync15.DocumentsFileTree(ht),
		failDelete:  map[string]error{},
		ghostDelete: map[string]error{},
	}
	return f
}

func (f *cloudFake) id() string {
	f.nextID++
	return "cid-" + itoa(f.nextID)
}

func (f *cloudFake) addRecord(parent, name, typ string) *model.Document {
	doc := sync15.NewBlobDoc(name, f.id(), typ, parent)
	f.ht.Docs = append(f.ht.Docs, doc)
	return doc.ToDocument()
}

func (f *cloudFake) Filetree() *filetree.FileTreeCtx { return f.ft }
func (f *cloudFake) SyncComplete() error             { return nil }
func (f *cloudFake) Nuke() error                     { return nil }

func (f *cloudFake) Refresh() (string, int64, error) {
	if f.failRefresh != nil {
		return "", 0, f.failRefresh
	}
	f.ft = sync15.DocumentsFileTree(f.ht)
	return "", f.ht.Generation, nil
}

func (f *cloudFake) DeleteEntry(node *model.Node, _, _ bool) error {
	id := node.Id()
	if err, ok := f.failDelete[id]; ok {
		return err
	}
	if err := f.ht.Remove(id); err != nil {
		return err
	}
	f.deletedIDs = append(f.deletedIDs, id)
	if err, ok := f.ghostDelete[id]; ok {
		return err // side effect committed, then the call still failed
	}
	return nil
}

func (f *cloudFake) CreateDir(parentId, name string, _ bool) (*model.Document, error) {
	return f.addRecord(parentId, name, model.DirectoryType), nil
}

func (f *cloudFake) UploadDocument(parentId, sourceDocPath string, _ bool, _, _, _ *int, _ *string) (*model.Document, error) {
	stem := filepath.Base(sourceDocPath)
	stem = stem[:len(stem)-len(filepath.Ext(stem))]
	return f.addRecord(parentId, stem, model.DocumentType), nil
}

func (f *cloudFake) MoveEntry(src, dstDir *model.Node, name string) (*model.Node, error) {
	for _, d := range f.ht.Docs {
		if d.DocumentID == src.Id() {
			d.Metadata.Parent = dstDir.Id()
			d.Metadata.DocName = name
			break
		}
	}
	doc := *src.Document
	doc.Name = name
	doc.Parent = dstDir.Id()
	return &model.Node{Document: &doc, Children: src.Children, Parent: dstDir}, nil
}

func (f *cloudFake) ReplaceDocumentFile(string, string, bool) error { return nil }

func (f *cloudFake) FetchDocument(docId, dstPath string) error {
	return os.WriteFile(dstPath, []byte("rmdoc-bytes"), 0o600)
}

// hasRecord reports whether the authoritative cloud inventory still holds id.
func (f *cloudFake) hasRecord(id string) bool {
	_, err := f.ht.FindDoc(id)
	return err == nil
}

// deepTree builds root > Top > Mid > {Leaf, Leaf2} and returns the fake plus
// the record IDs.
func deepTree(t *testing.T) (*cloudFake, string, string, string, string) {
	t.Helper()
	f := newCloudFake(
		cloudRec{id: "top", name: "Top", parent: "", typ: model.DirectoryType},
		cloudRec{id: "mid", name: "Mid", parent: "top", typ: model.DirectoryType},
		cloudRec{id: "leaf", name: "Leaf", parent: "mid", typ: model.DocumentType},
		cloudRec{id: "leaf2", name: "Leaf2", parent: "mid", typ: model.DocumentType},
	)
	return f, "top", "mid", "leaf", "leaf2"
}

// assertAgreement verifies List, Entries, and Search all reflect the same
// visible set and none of the gone IDs is anywhere.
func assertAgreement(t *testing.T, f *cloudFake, s *Service, gone ...string) {
	t.Helper()
	entries := s.Entries()
	listed, err := s.List("/")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	indexed := map[string]struct{}{}
	for _, e := range entries {
		indexed[e.ID] = struct{}{}
	}
	for _, e := range listed {
		if _, ok := indexed[e.ID]; !ok {
			t.Fatalf("List contains %q absent from Entries", e.ID)
		}
	}
	for _, g := range gone {
		if _, ok := indexed[g]; ok {
			t.Fatalf("Entries still contains deleted %q", g)
		}
		if hits := s.Search("a", 100); len(hits) > 0 {
			for _, h := range hits {
				if h.Entry.ID == g {
					t.Fatalf("Search still returns deleted %q", g)
				}
			}
		}
	}
}

// ---------- F01: descendant-first deletion ----------

func TestDeleteRecursiveRemovesEveryCloudRecord(t *testing.T) {
	f, top, _, leaf, leaf2 := deepTree(t)
	s := NewService(f, Config{})
	n, err := s.Delete([]string{top}, true)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if n != 4 {
		t.Fatalf("removed = %d, want all 4 records", n)
	}
	if got := strings.Join(f.deletedIDs, ","); got != "leaf,leaf2,mid,top" {
		t.Fatalf("removal order = %q, want children-before-parent leaf,leaf2,mid,top", got)
	}
	for _, id := range []string{top, "mid", leaf, leaf2} {
		if f.hasRecord(id) {
			t.Fatalf("cloud record %q still present after recursive delete", id)
		}
	}
	// Rebuild from the authoritative inventory: absence must survive a
	// refresh, not just a detached local edge.
	if err := s.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	assertAgreement(t, f, s, top, "mid", leaf, leaf2)
	if f.ft.NodeById(leaf) != nil {
		t.Fatal("rebuilt file tree still maps the deleted leaf ID")
	}
}

func TestDeletePlanDeduplicatesOverlappingSelections(t *testing.T) {
	f, top, mid, leaf, _ := deepTree(t)
	s := NewService(f, Config{})
	n, err := s.Delete([]string{top, mid, leaf, mid, top}, true)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if n != 4 {
		t.Fatalf("removed = %d, want 4 unique records for overlapping selection", n)
	}
	seen := map[string]int{}
	for _, id := range f.deletedIDs {
		seen[id]++
	}
	for id, c := range seen {
		if c != 1 {
			t.Fatalf("record %q removed %d times", id, c)
		}
	}
}

func TestDeleteNonrecursiveNonemptyRejectedBeforeSideEffects(t *testing.T) {
	f, top, _, _, _ := deepTree(t)
	s := NewService(f, Config{})
	n, err := s.Delete([]string{top}, false)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("error = %v, want ErrInvalid for nonrecursive nonempty folder", err)
	}
	if n != 0 || len(f.deletedIDs) != 0 {
		t.Fatalf("preflight rejection must not delete (removed=%d cloudDeletes=%v)", n, f.deletedIDs)
	}
	if !f.hasRecord(top) {
		t.Fatal("cloud record must remain after preflight rejection")
	}
	// A nonrecursive delete of an empty folder still works.
	if _, err := s.CreateFolder("/", "Empty"); err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	var emptyID string
	for _, e := range s.Entries() {
		if e.Name == "Empty" {
			emptyID = e.ID
		}
	}
	n, err = s.Delete([]string{emptyID}, false)
	if err != nil || n != 1 {
		t.Fatalf("nonrecursive empty-folder delete = (%d, %v)", n, err)
	}
	if f.hasRecord(emptyID) {
		t.Fatal("empty folder record not removed")
	}
}

func TestDeleteProtectedRootsAndUnknownIDsPreflight(t *testing.T) {
	f, top, _, _, _ := deepTree(t)
	s := NewService(f, Config{})
	for _, bad := range []string{"", filetree.TrashID, "missing-id"} {
		n, err := s.Delete([]string{bad}, true)
		if err == nil {
			t.Fatalf("Delete(%q) must fail", bad)
		}
		if n != 0 || len(f.deletedIDs) != 0 {
			t.Fatalf("preflight failure for %q must not delete (removed=%d cloudDeletes=%v)", bad, n, f.deletedIDs)
		}
	}
	if !f.hasRecord(top) {
		t.Fatal("tree must be untouched after preflight failures")
	}
}

// ---------- F05: live-ID integrity and partial outcomes ----------

func TestDeletePartialFailureReportsOutcomes(t *testing.T) {
	f, top, mid, leaf, _ := deepTree(t)
	f.failDelete["leaf2"] = errors.New("cloud 500")
	s := NewService(f, Config{})
	n, err := s.Delete([]string{top}, true)
	var partial *PartialDeleteError
	if !errors.As(err, &partial) {
		t.Fatalf("error = %v, want *PartialDeleteError", err)
	}
	if partial.FailedID != "leaf2" {
		t.Fatalf("FailedID = %q, want leaf2", partial.FailedID)
	}
	if partial.ReconcileErr != nil {
		t.Fatalf("unexpected reconciliation failure: %v", partial.ReconcileErr)
	}
	if partial.Removed != 1 {
		t.Fatalf("Removed = %d, want 1 (leaf)", partial.Removed)
	}
	if got := strings.Join(partial.RemainingIDs, ","); got != "leaf2,mid,top" {
		t.Fatalf("RemainingIDs = %q, want leaf2,mid,top", got)
	}
	if n != partial.Removed {
		t.Fatalf("returned count %d disagrees with outcome %d", n, partial.Removed)
	}
	// Cloud truth: exactly the pre-failure records are gone.
	if f.hasRecord(leaf) {
		t.Fatal("leaf record should be gone from the cloud inventory")
	}
	for _, id := range []string{"leaf2", mid, top} {
		if !f.hasRecord(id) {
			t.Fatalf("record %q should remain in the cloud inventory", id)
		}
	}
	// Reconciled views agree: List, Entries, Search, and lookups.
	assertAgreement(t, f, s, leaf)
	if _, err := s.List("/Top/Mid"); err != nil {
		t.Fatalf("List of surviving folder: %v", err)
	}
}

func TestDeleteSideEffectThenErrorReconciles(t *testing.T) {
	f, top, _, leaf, _ := deepTree(t)
	f.ghostDelete["leaf"] = errors.New("connection reset after commit")
	s := NewService(f, Config{})
	_, err := s.Delete([]string{top}, true)
	var partial *PartialDeleteError
	if !errors.As(err, &partial) {
		t.Fatalf("error = %v, want *PartialDeleteError", err)
	}
	if partial.ReconcileErr != nil {
		t.Fatalf("unexpected reconciliation failure: %v", partial.ReconcileErr)
	}
	// The failed record was actually committed: reconciliation must report
	// it as removed rather than pretending it remains or claiming rollback.
	if partial.Removed != 1 {
		t.Fatalf("Removed = %d, want 1 (ghost-committed leaf)", partial.Removed)
	}
	if got := strings.Join(partial.RemainingIDs, ","); got != "leaf2,mid,top" {
		t.Fatalf("RemainingIDs = %q, want leaf2,mid,top", got)
	}
	if f.hasRecord(leaf) {
		t.Fatal("ghost-committed record missing from cloud inventory check")
	}
	assertAgreement(t, f, s, leaf)
	if _, err := s.List("/Top/Mid"); err != nil {
		t.Fatalf("List of surviving folder: %v", err)
	}
}

func TestDeleteReconciliationFailureReported(t *testing.T) {
	f, top, _, _, _ := deepTree(t)
	f.failDelete["leaf2"] = errors.New("cloud 500")
	f.failRefresh = errors.New("mirror failed")
	s := NewService(f, Config{})
	_, err := s.Delete([]string{top}, true)
	var partial *PartialDeleteError
	if !errors.As(err, &partial) {
		t.Fatalf("error = %v, want *PartialDeleteError", err)
	}
	if partial.ReconcileErr == nil {
		t.Fatal("reconciliation failure must be reported, not swallowed")
	}
	if !strings.Contains(partial.Error(), "reconciliation failed") {
		t.Fatalf("error %q does not mention reconciliation failure", partial.Error())
	}
	// Even without reconciliation, list/search/index must agree with the
	// current tree edges (leaf was detached before the failure).
	assertAgreement(t, f, s, "leaf")
	if partial.Removed != 1 {
		t.Fatalf("Removed = %d, want 1 (locally detached leaf)", partial.Removed)
	}
	if got := strings.Join(partial.RemainingIDs, ","); got != "leaf2,mid,top" {
		t.Fatalf("RemainingIDs = %q, want leaf2,mid,top", got)
	}
}

func TestLiveLookupRejectsDeletedAndDescendantIDs(t *testing.T) {
	f, top, mid, leaf, _ := deepTree(t)
	s := NewService(f, Config{})
	if _, err := s.Delete([]string{top}, true); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	// The pinned filetree still returns stale idToNode nodes and stale
	// parent pointers; the service must reject them anyway.
	if f.ft.NodeById(leaf) == nil {
		t.Fatal("precondition: pinned idToNode still holds the deleted leaf")
	}
	for _, id := range []string{top, mid, leaf} {
		if _, _, err := s.Download(id); !errors.Is(err, ErrNotFound) {
			t.Fatalf("Download(%q) error = %v, want ErrNotFound", id, err)
		}
		if _, err := s.Rename(id, "x"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("Rename(%q) error = %v, want ErrNotFound", id, err)
		}
		if _, err := s.Move(id, "/"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("Move(%q) error = %v, want ErrNotFound", id, err)
		}
		if _, err := s.Delete([]string{id}, true); !errors.Is(err, ErrNotFound) {
			t.Fatalf("Delete(%q) error = %v, want ErrNotFound", id, err)
		}
	}
	assertAgreement(t, f, s, top, mid, leaf)
}

func TestTrashRemainsVisibleToOperationsButNotDeletable(t *testing.T) {
	f, _, _, _, _ := deepTree(t)
	s := NewService(f, Config{})
	if _, err := s.Delete([]string{filetree.TrashID}, true); !errors.Is(err, ErrInvalid) {
		t.Fatalf("trash delete error = %v, want ErrInvalid", err)
	}
	// trash must not appear in visible listings either.
	for _, e := range s.Entries() {
		if e.ID == filetree.TrashID {
			t.Fatal("trash leaked into Entries")
		}
	}
}

func TestDeleteRetryAfterPartialResult(t *testing.T) {
	f, top, mid, _, leaf2 := deepTree(t)
	f.failDelete["leaf2"] = errors.New("cloud 500")
	s := NewService(f, Config{})
	n, err := s.Delete([]string{top}, true)
	var partial *PartialDeleteError
	if !errors.As(err, &partial) || partial.Removed != 1 {
		t.Fatalf("first attempt = (%d, %v), want partial with Removed=1", n, err)
	}
	// Retry the whole selection after the transient failure: remaining
	// records are removed; the service must not pretend the first attempt
	// was atomic or duplicate already-removed records.
	delete(f.failDelete, "leaf2")
	n, err = s.Delete([]string{top}, true)
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if n != 3 {
		t.Fatalf("retry removed = %d, want the 3 remaining records", n)
	}
	for _, id := range []string{top, mid, leaf2} {
		if f.hasRecord(id) {
			t.Fatalf("record %q still present after retry", id)
		}
	}
	if err := s.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	assertAgreement(t, f, s, top, mid, leaf2)
}
