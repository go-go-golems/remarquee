package rmfiles

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/juruen/rmapi/api"
	"github.com/juruen/rmapi/model"
)

// Each injected error follows an authoritative record mutation, not a local
// tree mutation. Refresh must rebuild the local tree to discover the commit.
type uncertainAPI struct {
	*cloudFake
	createErr, moveErr, uploadErr, syncErr  error
	createCalls, failCreateAt, refreshCalls int
}

var _ api.ApiCtx = (*uncertainAPI)(nil)

func (f *uncertainAPI) CreateDir(parent, name string, sync bool) (*model.Document, error) {
	f.createCalls++
	doc, err := f.cloudFake.CreateDir(parent, name, sync)
	if f.createErr != nil && (f.failCreateAt == 0 || f.createCalls == f.failCreateAt) {
		return nil, f.createErr
	}
	return doc, err
}
func (f *uncertainAPI) MoveEntry(src, dst *model.Node, name string) (*model.Node, error) {
	node, err := f.cloudFake.MoveEntry(src, dst, name)
	if f.moveErr != nil {
		return nil, f.moveErr
	}
	return node, err
}
func (f *uncertainAPI) UploadDocument(parent, source string, sync bool, p, x, y *int, font *string) (*model.Document, error) {
	doc, err := f.cloudFake.UploadDocument(parent, source, sync, p, x, y, font)
	if f.uploadErr != nil {
		return nil, f.uploadErr
	}
	return doc, err
}
func (f *uncertainAPI) SyncComplete() error { return f.syncErr }
func (f *uncertainAPI) Refresh() (string, int64, error) {
	f.refreshCalls++
	return f.cloudFake.Refresh()
}

func assertViewsCurrent(t *testing.T, s *Service) {
	t.Helper()
	want := WalkEntries(s.apiCtx.Filetree().Root(), s.cfg.IncludeTemplates)
	if got := s.Entries(); !slices.Equal(got, want) {
		t.Fatalf("index differs from current tree: got=%v want=%v", got, want)
	}
	for _, e := range want {
		found := false
		for _, hit := range s.Search(e.Name, 1000) {
			if hit.Entry.ID == e.ID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Search missing current entry %q", e.ID)
		}
		if e.IsDir {
			got, err := s.List(e.Path)
			if err != nil {
				t.Fatal(err)
			}
			node := s.apiCtx.Filetree().NodeById(e.ID)
			if expected := ChildrenEntries(node, s.cfg.IncludeTemplates); !reflect.DeepEqual(got, expected) {
				t.Fatalf("List(%q) differs from current tree", e.Path)
			}
		}
	}
}

func TestUncertainMutationReconciles(t *testing.T) {
	for _, kind := range []string{"create", "rename", "move", "upload", "nested-folder"} {
		t.Run(kind, func(t *testing.T) {
			f := &uncertainAPI{cloudFake: newCloudFake(
				cloudRec{id: "doc", name: "Document", typ: model.DocumentType},
				cloudRec{id: "dest", name: "Destination", typ: model.DirectoryType},
			)}
			s := NewService(f, Config{})
			injected := errors.New("connection reset after commit")
			var err error
			switch kind {
			case "create":
				f.createErr = injected
				_, err = s.CreateFolder("/", "Created")
			case "rename":
				f.moveErr = injected
				_, err = s.Rename("doc", "Renamed")
			case "move":
				f.moveErr = injected
				_, err = s.Move("doc", "/Destination")
			case "upload":
				f.uploadErr = injected
				_, err = s.uploadOne(context.Background(), newJobStore().create(1, "/"), 0, UploadInput{Name: "Uploaded.pdf", Data: []byte("%PDF-1.4\n")}, "/")
			case "nested-folder":
				f.createErr = injected
				f.failCreateAt = 2
				_, err = s.uploadOne(context.Background(), newJobStore().create(1, "/"), 0, UploadInput{Name: "First/Second/Uploaded.pdf", Data: []byte("%PDF-1.4\n")}, "/")
			}
			if !errors.Is(err, injected) {
				t.Fatalf("error=%v want original cloud error", err)
			}
			if f.refreshCalls != 1 {
				t.Fatalf("Refresh calls=%d want 1 after uncertain %s", f.refreshCalls, kind)
			}
			assertViewsCurrent(t, s)
			switch kind {
			case "rename":
				if f.ft.NodeById("doc").Name() != "Renamed" {
					t.Fatal("committed rename missing")
				}
			case "move":
				if f.ft.NodeById("doc").Parent.Id() != "dest" {
					t.Fatal("committed move missing")
				}
			default:
				name := map[string]string{"create": "Created", "upload": "Uploaded", "nested-folder": "Second"}[kind]
				found := false
				for _, e := range s.Entries() {
					if e.Name == name {
						found = true
					}
				}
				if !found {
					t.Fatalf("committed %s missing", kind)
				}
			}
		})
	}
}

func TestSyncNotificationFailureKeepsViewsCurrent(t *testing.T) {
	for _, kind := range []string{"create", "rename", "move", "upload", "delete"} {
		t.Run(kind, func(t *testing.T) {
			injected := errors.New("notification failed")
			f := &uncertainAPI{cloudFake: newCloudFake(cloudRec{id: "doc", name: "Document", typ: model.DocumentType}, cloudRec{id: "dest", name: "Destination", typ: model.DirectoryType}), syncErr: injected}
			s := NewService(f, Config{})
			var err error
			switch kind {
			case "create":
				_, err = s.CreateFolder("/", "Created")
			case "rename":
				_, err = s.Rename("doc", "Renamed")
			case "move":
				_, err = s.Move("doc", "/Destination")
			case "upload":
				_, err = s.uploadOne(context.Background(), newJobStore().create(1, "/"), 0, UploadInput{Name: "Uploaded.pdf", Data: []byte("%PDF-1.4\n")}, "/")
			case "delete":
				n, e := s.Delete([]string{"doc"}, false)
				err = e
				var partial *PartialDeleteError
				if !errors.As(err, &partial) || partial.Removed != 1 || n != 1 || len(partial.RemainingIDs) != 0 {
					t.Fatalf("notification failure lost committed deletion: count=%d error=%v", n, err)
				}
			}
			if !errors.Is(err, injected) {
				t.Fatalf("error=%v want notification error", err)
			}
			assertViewsCurrent(t, s)
		})
	}
}

func TestUncertainMutationReportsReconciliationFailure(t *testing.T) {
	f := &uncertainAPI{cloudFake: newCloudFake(), createErr: errors.New("commit uncertain")}
	f.failRefresh = errors.New("mirror failed")
	s := NewService(f, Config{})
	_, err := s.CreateFolder("/", "Created")
	if err == nil || !errors.Is(err, f.createErr) || f.refreshCalls != 1 {
		t.Fatalf("error=%v Refresh=%d", err, f.refreshCalls)
	}
	if !strings.Contains(err.Error(), "reconciliation failed") {
		t.Fatalf("missing reconciliation failure: %v", err)
	}
	assertViewsCurrent(t, s)
}
