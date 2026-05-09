package api

import (
	"net/http"
	"path/filepath"
	"strings"
)

// HandleOutputs handles GET /api/outputs/{filename}
func HandleOutputs(outputsDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		filename := r.PathValue("filename")
		if filename == "" {
			http.Error(w, "Missing filename in path", http.StatusBadRequest)
			return
		}

		// Security: prevent path traversal
		if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
			http.Error(w, "Invalid filename", http.StatusBadRequest)
			return
		}

		// Serve the file. The filename path variable is restricted to a basename above.
		filePath := filepath.Join(outputsDir, filename)

		// Set content type based on extension
		if strings.HasSuffix(filename, ".pdf") {
			w.Header().Set("Content-Type", "application/pdf")
		}

		http.ServeFile(w, r, filePath) // #nosec G703 -- filename is rejected when it contains path separators or "..".
	}
}
