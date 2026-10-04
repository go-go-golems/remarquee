package serve

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-go-golems/remarquee/pkg/rmfiles"
	"github.com/rs/zerolog/log"
)

const maxUploadBytes = 64 << 20 // 64 MiB per request

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Error().Err(err).Msg("serve: write json response")
	}
}

func statusFor(err error) int {
	switch {
	case errors.Is(err, rmfiles.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, rmfiles.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, rmfiles.ErrInvalid), errors.Is(err, rmfiles.ErrNotDir):
		return http.StatusBadRequest
	case errors.Is(err, rmfiles.ErrUnsupported):
		return http.StatusUnsupportedMediaType
	default:
		return http.StatusInternalServerError
	}
}

func codeFor(err error) string {
	switch {
	case errors.Is(err, rmfiles.ErrNotFound):
		return "not_found"
	case errors.Is(err, rmfiles.ErrConflict):
		return "conflict"
	case errors.Is(err, rmfiles.ErrInvalid):
		return "invalid"
	case errors.Is(err, rmfiles.ErrNotDir):
		return "not_a_directory"
	case errors.Is(err, rmfiles.ErrUnsupported):
		return "unsupported"
	default:
		return "internal"
	}
}

func writeErr(w http.ResponseWriter, err error) {
	status := statusFor(err)
	msg := err.Error()
	if status == http.StatusInternalServerError {
		log.Error().Err(err).Msg("serve: internal error")
		msg = "internal server error"
	}
	writeJSON(w, status, errorBody{Error: errorDetail{Code: codeFor(err), Message: msg}})
}

func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func cloneURLPath(r *http.Request, p string) *http.Request {
	r2 := r.Clone(r.Context())
	u := *r.URL
	u.Path = p
	u.RawPath = ""
	r2.URL = &u
	return r2
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.svc.Status())
}

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	dir := r.URL.Query().Get("dir")
	if dir == "" {
		dir = "/"
	}
	entries, err := s.svc.List(dir)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"path": dir, "entries": entries})
}

func (s *Server) handleTree(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"entries": s.svc.Entries()})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"query": q, "results": s.svc.Search(q, limit)})
}

func (s *Server) handleCreateFolder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ParentPath string `json:"parentPath"`
		Name       string `json:"name"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, rmfiles.ErrInvalid)
		return
	}
	entry, err := s.svc.CreateFolder(req.ParentPath, req.Name)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, entry)
}

func (s *Server) handleRename(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, rmfiles.ErrInvalid)
		return
	}
	entry, err := s.svc.Rename(id, req.Name)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (s *Server) handleMove(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		DestDir string `json:"destDir"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, rmfiles.ErrInvalid)
		return
	}
	entry, err := s.svc.Move(id, req.DestDir)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs       []string `json:"ids"`
		Confirm   string   `json:"confirm"`
		Recursive bool     `json:"recursive"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, rmfiles.ErrInvalid)
		return
	}
	if req.Confirm != "DELETE" {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: errorDetail{
			Code:    "confirmation_required",
			Message: "deletion requires confirm to be the string DELETE",
		}})
		return
	}
	// Recursive defaults to true because the UI already gates deletion behind an
	// explicit typed confirmation; non-empty folders otherwise cannot be removed.
	recursive := true
	if !req.Recursive && len(req.IDs) > 0 {
		recursive = false
	}
	n, err := s.svc.Delete(req.IDs, recursive)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"deleted": n})
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	data, name, err := s.svc.Download(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	// The service sanitizes the suggested name; FormatMediaType additionally
	// applies RFC 2231 encoding so quotes, spaces and Unicode round-trip.
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": name})
	if disposition == "" {
		disposition = "attachment; filename=\"document.rmdoc\""
	}
	w.Header().Set("Content-Disposition", disposition)
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		log.Error().Err(err).Msg("serve: write download")
	}
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: errorDetail{
			Code:    "invalid",
			Message: "invalid multipart upload: " + err.Error(),
		}})
		return
	}
	dest := r.FormValue("destDir")
	if dest == "" {
		dest = r.URL.Query().Get("destDir")
	}
	fileHeaders := r.MultipartForm.File["files"]
	if len(fileHeaders) == 0 {
		writeErr(w, rmfiles.ErrInvalid)
		return
	}
	// Optional parallel "paths" values carry folder-relative paths so a dropped
	// or selected folder tree can be recreated under destDir.
	paths := r.MultipartForm.Value["paths"]
	inputs := make([]rmfiles.UploadInput, 0, len(fileHeaders))
	for i, fh := range fileHeaders {
		f, err := fh.Open()
		if err != nil {
			writeErr(w, err)
			return
		}
		data, err := io.ReadAll(io.LimitReader(f, maxUploadBytes))
		_ = f.Close()
		if err != nil {
			writeErr(w, err)
			return
		}
		name := fh.Filename
		if i < len(paths) && strings.TrimSpace(paths[i]) != "" {
			name = paths[i]
		}
		inputs = append(inputs, rmfiles.UploadInput{Name: name, Data: data})
	}
	jobID, err := s.svc.StartUpload(inputs, dest)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"jobId": jobID})
}

func (s *Server) handleUploadStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	job, ok := s.svc.GetUpload(id)
	if !ok {
		writeErr(w, rmfiles.ErrNotFound)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.Refresh(r.Context()); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.svc.Status())
}
