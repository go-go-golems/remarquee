package api

import (
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

// HandleOutputs handles GET /api/outputs/:filename
func HandleOutputs(outputsDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract filename from URL path
		// Expected: /api/outputs/{filename}
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(pathParts) != 3 || pathParts[0] != "api" || pathParts[1] != "outputs" {
			http.Error(w, "Invalid path format, expected /api/outputs/{filename}", http.StatusBadRequest)
			return
		}
		filename := pathParts[2]

		// Security: prevent path traversal
		if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
			http.Error(w, "Invalid filename", http.StatusBadRequest)
			return
		}

		// Serve the file
		filePath := filepath.Join(outputsDir, filename)
		log.Printf("Serving output file: %s", filePath)

		// Set content type based on extension
		if strings.HasSuffix(filename, ".pdf") {
			w.Header().Set("Content-Type", "application/pdf")
		}

		http.ServeFile(w, r, filePath)
	}
}

