package serve

import (
	"bytes"
	"encoding/json"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-go-golems/remarquee/pkg/rmfiles"
	"github.com/juruen/rmapi/api"
	"github.com/juruen/rmapi/filetree"
	"github.com/juruen/rmapi/model"
)

type fakeApiCtx struct {
	ft     filetree.FileTreeCtx
	mu     sync.Mutex
	nextID int
	bytes  int
}

var _ api.ApiCtx = (*fakeApiCtx)(nil)

func newFake() *fakeApiCtx { return &fakeApiCtx{ft: filetree.CreateFileTreeCtx()} }

func (f *fakeApiCtx) id() string {
	f.nextID++
	return "id-" + string(rune('a'+f.nextID))
}

func (f *fakeApiCtx) add(parentID, name, typ string) *model.Document {
	d := &model.Document{ID: f.id(), Name: name, Type: typ, Parent: parentID}
	f.ft.AddDocument(d)
	return d
}

func (f *fakeApiCtx) Filetree() *filetree.FileTreeCtx                { return &f.ft }
func (f *fakeApiCtx) SyncComplete() error                            { return nil }
func (f *fakeApiCtx) Nuke() error                                    { return nil }
func (f *fakeApiCtx) Refresh() (string, int64, error)                { return "", 0, nil }
func (f *fakeApiCtx) ReplaceDocumentFile(string, string, bool) error { return nil }
func (f *fakeApiCtx) FetchDocument(_, dst string) error {
	return os.WriteFile(dst, []byte("archive"), 0o600)
}
func (f *fakeApiCtx) CreateDir(parentID, name string, _ bool) (*model.Document, error) {
	return f.add(parentID, name, model.DirectoryType), nil
}
func (f *fakeApiCtx) UploadDocument(parentID, source string, _ bool, _, _, _ *int, _ *string) (*model.Document, error) {
	stem := filepath.Base(source)
	stem = strings.TrimSuffix(stem, filepath.Ext(stem))
	return &model.Document{ID: f.id(), Name: stem, Type: model.DocumentType, Parent: parentID}, nil
}
func (f *fakeApiCtx) MoveEntry(src, dst *model.Node, name string) (*model.Node, error) {
	doc := *src.Document
	doc.Name = name
	doc.Parent = dst.Id()
	return &model.Node{Document: &doc, Parent: dst}, nil
}
func (f *fakeApiCtx) DeleteEntry(*model.Node, bool, bool) error { return nil }

func newTestServer(t *testing.T) (*Server, string, string) {
	t.Helper()
	f := newFake()
	ai := f.add("", "ai", model.DirectoryType)
	notes := f.add(ai.ID, "Notes", model.DirectoryType)
	doc := f.add(notes.ID, "Project Plan", model.DocumentType)
	svc := rmfiles.NewService(f, rmfiles.Config{DefaultRemoteDir: "/ai"})
	return NewServer(svc, Options{}), doc.ID, notes.ID
}

func do(t *testing.T, s *Server, method, target string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		r = httptest.NewRequest(method, target, bytes.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, target, nil)
	}
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, r)
	return rr
}

func TestHealthAndStatus(t *testing.T) {
	s, _, _ := newTestServer(t)
	if rr := do(t, s, http.MethodGet, "/api/health", nil); rr.Code != 200 {
		t.Fatalf("health = %d", rr.Code)
	}
	rr := do(t, s, http.MethodGet, "/api/status", nil)
	if rr.Code != 200 {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "authenticated") {
		t.Fatalf("status body = %s", rr.Body.String())
	}
}

func TestListFiles(t *testing.T) {
	s, _, _ := newTestServer(t)
	rr := do(t, s, http.MethodGet, "/api/files?dir=/ai/Notes", nil)
	if rr.Code != 200 {
		t.Fatalf("list = %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "Project Plan") {
		t.Fatalf("body = %s", rr.Body.String())
	}
}

func TestSearchEndpoint(t *testing.T) {
	s, _, _ := newTestServer(t)
	rr := do(t, s, http.MethodGet, "/api/search?q=proj", nil)
	if rr.Code != 200 {
		t.Fatalf("search = %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Project Plan") {
		t.Fatalf("body = %s", rr.Body.String())
	}
}

func TestCreateFolderEndpoint(t *testing.T) {
	s, _, notesID := newTestServer(t)
	rr := do(t, s, http.MethodPost, "/api/folders", map[string]string{"parentPath": "/ai/Notes", "name": "Ideas"})
	if rr.Code != 201 {
		t.Fatalf("create folder = %d body=%s", rr.Code, rr.Body.String())
	}
	_ = notesID
}

func TestRenameEndpoint(t *testing.T) {
	s, docID, _ := newTestServer(t)
	rr := do(t, s, http.MethodPatch, "/api/entries/"+docID, map[string]string{"name": "Renamed"})
	if rr.Code != 200 {
		t.Fatalf("rename = %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "Renamed") {
		t.Fatalf("body = %s", rr.Body.String())
	}
}

func TestDeleteRequiresConfirmation(t *testing.T) {
	s, docID, _ := newTestServer(t)
	rr := do(t, s, http.MethodDelete, "/api/entries", map[string]any{"ids": []string{docID}, "confirm": "nope"})
	if rr.Code != 400 {
		t.Fatalf("delete without confirm = %d, want 400", rr.Code)
	}
	rr = do(t, s, http.MethodDelete, "/api/entries", map[string]any{"ids": []string{docID}, "confirm": "DELETE", "recursive": true})
	if rr.Code != 200 {
		t.Fatalf("delete = %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestDownloadEndpoint(t *testing.T) {
	s, docID, _ := newTestServer(t)
	rr := do(t, s, http.MethodGet, "/api/entries/"+docID+"/download", nil)
	if rr.Code != 200 {
		t.Fatalf("download = %d", rr.Code)
	}
	if rr.Body.Len() == 0 {
		t.Fatal("expected download bytes")
	}
}

// TestDownloadAttachmentHeaderIsSanitized proves adversarial cloud display
// names (simulating metadata imported from other cloud clients, bypassing UI
// validation) produce Content-Disposition headers that parse cleanly and
// never carry separators, quotes, or control characters into the suggested
// filename. Table cases cover traversal, quotes, control characters, spaces
// (RFC 2231 encoding), and Unicode.
func TestDownloadAttachmentHeaderIsSanitized(t *testing.T) {
	cases := []struct {
		name         string // raw cloud display name
		wantFilename string // expected parsed filename parameter
	}{
		{
			name:         `..\..\evil "quoted"` + "\x07/slash",
			wantFilename: ".._.._evil _quoted__slash.rmdoc",
			// control \x07 dropped; / and \\ and " become _; space kept
		},
		{
			name:         "unicode é中",
			wantFilename: "unicode é中.rmdoc",
		},
		{
			name:         "sp ace.rmdoc",
			wantFilename: "sp ace.rmdoc.rmdoc",
		},
		{
			name:         "line\r\nbreak\"q\"",
			wantFilename: "linebreak_q_.rmdoc",
		},
		{
			name:         `a"b`,
			wantFilename: "a_b.rmdoc",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFake()
			ai := f.add("", "ai", model.DirectoryType)
			doc := f.add(ai.ID, tc.name, model.DocumentType)
			s := NewServer(rmfiles.NewService(f, rmfiles.Config{DefaultRemoteDir: "/ai"}), Options{})
			rr := do(t, s, http.MethodGet, "/api/entries/"+doc.ID+"/download", nil)
			if rr.Code != 200 {
				t.Fatalf("download = %d body=%s", rr.Code, rr.Body.String())
			}
			raw := rr.Header().Get("Content-Disposition")
			if raw == "" {
				t.Fatal("missing Content-Disposition header")
			}
			// The raw header value must never contain a carriage return or
			// newline, which would enable header injection.
			if strings.ContainsAny(raw, "\r\n") {
				t.Fatalf("header value %q contains CR/LF", raw)
			}
			mediatype, params, err := mime.ParseMediaType(raw)
			if err != nil {
				t.Fatalf("Content-Disposition %q does not parse: %v", raw, err)
			}
			if mediatype != "attachment" {
				t.Fatalf("disposition type = %q, want attachment", mediatype)
			}
			name, ok := params["filename"]
			if !ok {
				t.Fatalf("Content-Disposition %q has no filename parameter", raw)
			}
			if name != tc.wantFilename {
				t.Fatalf("filename = %q, want %q", name, tc.wantFilename)
			}
			if !strings.HasSuffix(name, ".rmdoc") {
				t.Fatalf("filename %q lacks .rmdoc suffix", name)
			}
			if strings.ContainsAny(name, `/\\"`) {
				t.Fatalf("filename %q contains separators or quotes", name)
			}
			for _, r := range name {
				if r < 0x20 || r == 0x7f {
					t.Fatalf("filename %q contains control character %q", name, r)
				}
			}
		})
	}
}

func TestUploadEndpoint(t *testing.T) {
	s, _, _ := newTestServer(t)
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("files", "note.pdf")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fw.Write([]byte("%PDF-1.4 test"))
	_ = mw.WriteField("destDir", "/ai")
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/uploads", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != 202 {
		t.Fatalf("upload = %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "jobId") {
		t.Fatalf("body = %s", rr.Body.String())
	}
}

func TestUploadFolderPaths(t *testing.T) {
	s, _, _ := newTestServer(t)
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for _, rel := range []string{"MyFolder/sub/a.pdf", "MyFolder/b.pdf"} {
		fw, err := mw.CreateFormFile("files", filepath.Base(rel))
		if err != nil {
			t.Fatal(err)
		}
		_, _ = fw.Write([]byte("%PDF-1.4 test"))
		if err := mw.WriteField("paths", rel); err != nil {
			t.Fatal(err)
		}
	}
	_ = mw.WriteField("destDir", "/")
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/uploads", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != 202 {
		t.Fatalf("upload = %d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		JobID string `json:"jobId"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	// Wait for the job to finish, then assert the relative path is preserved.
	for i := 0; i < 100; i++ {
		job, ok := s.svc.GetUpload(resp.JobID)
		if ok && (job.State == rmfiles.JobDone || job.State == rmfiles.JobFailed) {
			if job.Items[0].Name != "MyFolder/sub/a.pdf" {
				t.Fatalf("item name = %q, want MyFolder/sub/a.pdf", job.Items[0].Name)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("upload job did not finish")
}

func TestStaticServesIndexAndFallsBack(t *testing.T) {
	s, _, _ := newTestServer(t)
	rr := do(t, s, http.MethodGet, "/", nil)
	if rr.Code != 200 || !strings.Contains(rr.Body.String(), "REMARQUEE FILES") {
		t.Fatalf("index = %d body=%s", rr.Code, rr.Body.String())
	}
	rr = do(t, s, http.MethodGet, "/some/client/route", nil)
	if rr.Code != 200 || !strings.Contains(rr.Body.String(), "REMARQUEE FILES") {
		t.Fatalf("spa fallback = %d", rr.Code)
	}
}

func TestUnknownAPI404(t *testing.T) {
	s, _, _ := newTestServer(t)
	rr := do(t, s, http.MethodGet, "/api/nope", nil)
	if rr.Code != 404 {
		t.Fatalf("unknown api = %d, want 404", rr.Code)
	}
}
