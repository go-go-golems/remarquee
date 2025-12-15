package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
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

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/test-documents", handleTestDocuments)
	mux.HandleFunc("/api/health", handleHealth)

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

