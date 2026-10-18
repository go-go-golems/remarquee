package rmfiles

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-go-golems/remarquee/pkg/mdpdf"
	"github.com/go-go-golems/remarquee/pkg/rmcloud"
	"github.com/juruen/rmapi/api"
	"github.com/juruen/rmapi/model"
	"github.com/juruen/rmapi/util"
	"github.com/pkg/errors"
)

// Upload job states.
const (
	JobQueued     = "queued"
	JobConverting = "converting"
	JobUploading  = "uploading"
	JobDone       = "done"
	JobFailed     = "failed"
)

// UploadInput is one file to upload.
type UploadInput struct {
	Name string
	Data []byte
}

// UploadItem is the per-file state of an upload job.
type UploadItem struct {
	Name    string `json:"name"`
	State   string `json:"state"`
	Error   string `json:"error,omitempty"`
	EntryID string `json:"entryId,omitempty"`
}

// UploadJob is a snapshot of an upload job.
type UploadJob struct {
	ID        string       `json:"id"`
	State     string       `json:"state"`
	DestDir   string       `json:"destDir"`
	Total     int          `json:"total"`
	Done      int          `json:"done"`
	Items     []UploadItem `json:"items"`
	CreatedAt time.Time    `json:"createdAt"`
}

type jobStore struct {
	mu   sync.Mutex
	seq  int
	jobs map[string]*uploadJob
}

type uploadJob struct {
	mu   sync.Mutex
	snap UploadJob
}

func newJobStore() *jobStore {
	return &jobStore{jobs: make(map[string]*uploadJob)}
}

func (st *jobStore) create(total int, destDir string) *uploadJob {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.seq++
	id := "upload-" + itoa(st.seq) + "-" + itoa(int(time.Now().UnixNano()%1_000_000))
	items := make([]UploadItem, total)
	for i := range items {
		items[i].State = JobQueued
	}
	j := &uploadJob{snap: UploadJob{
		ID:        id,
		State:     JobQueued,
		DestDir:   destDir,
		Total:     total,
		Items:     items,
		CreatedAt: time.Now(),
	}}
	st.jobs[id] = j
	return j
}

func (st *jobStore) get(id string) (UploadJob, bool) {
	st.mu.Lock()
	j, ok := st.jobs[id]
	st.mu.Unlock()
	if !ok {
		return UploadJob{}, false
	}
	return j.snapshot(), true
}

func (j *uploadJob) snapshot() UploadJob {
	j.mu.Lock()
	defer j.mu.Unlock()
	out := j.snap
	out.Items = append([]UploadItem(nil), j.snap.Items...)
	return out
}

func (j *uploadJob) setItem(i int, state, errMsg, entryID string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if i < 0 || i >= len(j.snap.Items) {
		return
	}
	j.snap.Items[i].State = state
	if errMsg != "" {
		j.snap.Items[i].Error = errMsg
	}
	if entryID != "" {
		j.snap.Items[i].EntryID = entryID
	}
	done := 0
	failed := 0
	pending := 0
	for _, it := range j.snap.Items {
		switch it.State {
		case JobDone:
			done++
		case JobFailed:
			failed++
		default:
			pending++
		}
	}
	j.snap.Done = done
	switch {
	case pending > 0:
		j.snap.State = JobUploading
	case failed > 0:
		j.snap.State = JobFailed
	default:
		j.snap.State = JobDone
	}
}

// GetUpload returns a snapshot of an upload job.
func (s *Service) GetUpload(id string) (UploadJob, bool) {
	return s.jobs.get(id)
}

// StartUpload begins an asynchronous upload job and returns its ID. Each input
// is converted (Markdown -> PDF) and uploaded into destDir.
func (s *Service) StartUpload(inputs []UploadInput, destDir string) (string, error) {
	if len(inputs) == 0 {
		return "", ErrInvalid
	}
	dest := normalizePath(destDir)
	if dest == "/" && s.cfg.DefaultRemoteDir != "" {
		dest = normalizePath(s.cfg.DefaultRemoteDir)
	}
	j := s.jobs.create(len(inputs), dest)
	for i, in := range inputs {
		j.mu.Lock()
		j.snap.Items[i].Name = in.Name
		j.mu.Unlock()
	}
	go s.runUpload(j, inputs, dest)
	return j.snapshot().ID, nil
}

func (s *Service) runUpload(j *uploadJob, inputs []UploadInput, dest string) {
	// A job must outlive the HTTP request that started it, so it uses its own
	// bounded context rather than the request context.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	sem := make(chan struct{}, s.cfg.Workers)
	var wg sync.WaitGroup
	for i, in := range inputs {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, in UploadInput) {
			defer wg.Done()
			defer func() { <-sem }()
			entryID, err := s.uploadOne(ctx, j, i, in, dest)
			if err != nil {
				j.setItem(i, JobFailed, err.Error(), "")
				return
			}
			j.setItem(i, JobDone, "", entryID)
		}(i, in)
	}
	wg.Wait()
}

func (s *Service) uploadOne(ctx context.Context, j *uploadJob, i int, in UploadInput, dest string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	// in.Name may be a relative path ("folder/sub/file.md") when uploading a
	// folder or a dropped directory tree; preserve the structure under dest.
	rel := sanitizeRelPath(in.Name)
	baseName := path.Base(rel)
	dirPart := path.Dir(rel)
	ext := strings.ToLower(path.Ext(baseName))
	stem := strings.TrimSuffix(baseName, path.Ext(baseName))
	safe := sanitizeStem(stem)
	targetDir := joinRemote(dest, dirPart)

	tmpDir, err := os.MkdirTemp("", "rmfiles-up-")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	var uploadPath string
	switch ext {
	case ".md", ".markdown":
		j.setItem(i, JobConverting, "", "")
		mdPath := filepath.Join(tmpDir, safe+".md")
		// safe comes from sanitizeStem, an allowlist that strips every character
		// outside [a-zA-Z0-9_.-], so it cannot carry path separators or "..".
		// codeql[go/path-injection]
		if err := os.WriteFile(mdPath, in.Data, 0o600); err != nil {
			return "", err
		}
		uploadPath = filepath.Join(tmpDir, safe+".pdf")
		opts := mdpdf.DefaultPandocOptions()
		if err := mdpdf.ConvertMarkdownFileToPDF(ctx, mdPath, uploadPath, opts); err != nil {
			return "", errors.Wrap(err, "convert markdown to PDF")
		}
	case ".pdf", ".epub":
		uploadPath = filepath.Join(tmpDir, safe+ext)
		// safe is allowlist-sanitized (see above) and ext is a fixed literal;
		// the combined path cannot escape tmpDir.
		// codeql[go/path-injection]
		if err := os.WriteFile(uploadPath, in.Data, 0o600); err != nil {
			return "", err
		}
	default:
		return "", errors.Wrapf(ErrUnsupported, "file type %q (expected .md, .pdf, or .epub)", ext)
	}

	if !util.IsFileTypeSupported(strings.TrimPrefix(filepath.Ext(uploadPath), ".")) {
		return "", errors.Wrapf(ErrUnsupported, "file type %q", filepath.Ext(uploadPath))
	}

	j.setItem(i, JobUploading, "", "")
	var entryID string
	err = s.mutate(func(c api.ApiCtx) (bool, error) {
		parent, err := ensureDir(c, targetDir)
		if err != nil {
			// MkdirAll can commit earlier folders or its failing write.
			// Conservatively reconcile; an error is not a rollback.
			return true, err
		}
		doc, err := c.UploadDocument(parent.Id(), uploadPath, true, nil, nil, nil, nil)
		if err != nil {
			return true, errors.Wrap(err, "upload document")
		}
		c.Filetree().AddDocument(doc)
		entryID = doc.ID
		return true, nil
	})
	if err != nil {
		return "", err
	}
	return entryID, nil
}

// ensureDir ensures dest exists (creating intermediate folders) and returns it.
func ensureDir(c api.ApiCtx, dest string) (*model.Node, error) {
	return rmcloud.MkdirAll(c, dest)
}

// joinRemote joins a remote directory with a relative slash path. An empty or
// "." rel returns base unchanged; a "/" base yields "/rel".
func joinRemote(base, rel string) string {
	base = strings.TrimRight(base, "/")
	rel = strings.Trim(rel, "/")
	if rel == "" || rel == "." {
		if base == "" {
			return "/"
		}
		return base
	}
	if base == "" {
		return "/" + rel
	}
	return base + "/" + rel
}

// sanitizeRelPath normalizes a client-provided relative path: backslashes become
// slashes, empty/"."/".." segments are dropped (path-traversal guard), and the
// result is a clean slash path. Returns "document" if nothing remains.
func sanitizeRelPath(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	parts := strings.Split(name, "/")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || p == "." || p == ".." {
			continue
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return "document"
	}
	return strings.Join(out, "/")
}

var (
	stemRe     = regexp.MustCompile(`[^a-zA-Z0-9_.\-]`)
	underscore = regexp.MustCompile(`_{2,}`)
)

// sanitizeStem mirrors the CLI's PDF name sanitization for the file stem.
func sanitizeStem(stem string) string {
	stem = strings.ReplaceAll(stem, " ", "_")
	stem = stemRe.ReplaceAllString(stem, "")
	stem = underscore.ReplaceAllString(stem, "_")
	stem = strings.Trim(stem, "_-")
	if stem == "" {
		stem = "document"
	}
	return stem
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
