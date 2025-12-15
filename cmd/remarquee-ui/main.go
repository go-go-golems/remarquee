package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-go-golems/remarquee/cmd/remarquee-ui/api"
)

var (
	devMode bool
	port    string
)

func init() {
	flag.BoolVar(&devMode, "dev", false, "Run in development mode (proxy to Vite)")
	flag.StringVar(&port, "port", "8080", "HTTP server port")
}

func main() {
	flag.Parse()

	// Directories
	testDocsPath := "testdata"
	outputsDir := "outputs"
	ticketDir := "../../ttmp/2025/12/15/RMQ-RMDOC-WEB-001--build-remarquee-ui-web-validation-tool-for-rmdoc-rendering"

	// Ensure outputs directory exists
	if err := os.MkdirAll(outputsDir, 0755); err != nil {
		log.Fatalf("Failed to create outputs directory: %v", err)
	}

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/test-documents", handleTestDocuments)
	mux.HandleFunc("/api/document/", api.HandleInspect(testDocsPath))
	mux.HandleFunc("/api/render/background", api.HandleRenderBackground(testDocsPath, outputsDir))
	mux.HandleFunc("/api/render/legacy", api.HandleRenderLegacy(testDocsPath, outputsDir))
	mux.HandleFunc("/api/outputs/", api.HandleOutputs(outputsDir))
	mux.HandleFunc("/api/validation", api.HandleValidation(ticketDir))

	// Static assets (future: serve embedded frontend in prod mode)
	if devMode {
		log.Println("Running in DEV mode: expecting Vite dev server on :5173")
	} else {
		log.Println("Running in PROD mode: serving embedded assets (not yet implemented)")
	}

	addr := ":" + port
	log.Printf("remarquee-ui server listening on %s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleTestDocuments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read testdata/test-documents.json
	manifestPath := filepath.Join("testdata", "test-documents.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		log.Printf("Failed to read test documents manifest: %v", err)
		http.Error(w, fmt.Sprintf("Failed to read manifest: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

