package serve

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/go-go-golems/remarquee/pkg/rmfiles"
)

// Options configures the HTTP server.
type Options struct {
	// Dev disables static-asset caching (useful while editing the frontend).
	Dev bool
}

// Server wires the rmfiles service to HTTP handlers.
type Server struct {
	svc  *rmfiles.Service
	opts Options
	mux  *http.ServeMux
}

// NewServer builds the HTTP routing for the files UI.
func NewServer(svc *rmfiles.Service, opts Options) *Server {
	s := &Server{svc: svc, opts: opts, mux: http.NewServeMux()}
	s.routes()
	return s
}

// Handler returns the root HTTP handler.
func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/status", s.handleStatus)
	s.mux.HandleFunc("GET /api/files", s.handleFiles)
	s.mux.HandleFunc("GET /api/files/tree", s.handleTree)
	s.mux.HandleFunc("GET /api/search", s.handleSearch)
	s.mux.HandleFunc("POST /api/folders", s.handleCreateFolder)
	s.mux.HandleFunc("PATCH /api/entries/{id}", s.handleRename)
	s.mux.HandleFunc("POST /api/entries/{id}/move", s.handleMove)
	s.mux.HandleFunc("DELETE /api/entries", s.handleDelete)
	s.mux.HandleFunc("GET /api/entries/{id}/download", s.handleDownload)
	s.mux.HandleFunc("POST /api/uploads", s.handleUpload)
	s.mux.HandleFunc("GET /api/uploads/{id}", s.handleUploadStatus)
	s.mux.HandleFunc("POST /api/refresh", s.handleRefresh)
	s.mux.HandleFunc("GET /", s.handleStatic)
}

// handleStatic serves the embedded single-page app with an index.html fallback
// for client-side routes. /api/ paths are never handled here.
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}
	root, err := frontendRoot()
	if err != nil {
		http.Error(w, "frontend unavailable", http.StatusInternalServerError)
		return
	}
	if s.opts.Dev {
		w.Header().Set("Cache-Control", "no-store")
	} else if strings.HasSuffix(r.URL.Path, ".css") || strings.HasSuffix(r.URL.Path, ".js") {
		w.Header().Set("Cache-Control", "public, max-age=3600")
	}

	// SPA fallback: unknown non-asset paths return index.html. Asset-like
	// requests (anything with a file extension) must 404 instead: the fallback
	// document references root-relative assets, and serving HTML for a missing
	// stylesheet or module makes the browser drop it (MIME type text/html).
	p := strings.TrimPrefix(r.URL.Path, "/")
	if p == "" {
		p = "index.html"
	}
	if _, statErr := fs.Stat(root, p); statErr != nil {
		if isAssetLikePath(r.URL.Path) {
			http.NotFound(w, r)
			return
		}
		r = cloneURLPath(r, "/")
	}
	http.FileServer(http.FS(root)).ServeHTTP(w, r)
}

// isAssetLikePath reports whether the URL names a static asset rather than a
// client-side route: the last path segment carries a file extension.
func isAssetLikePath(p string) bool {
	return path.Ext(path.Base(p)) != ""
}
