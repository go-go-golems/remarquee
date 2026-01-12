package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-go-golems/remarquee/cmd/remarquee-ui/api"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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

	// Setup zerolog console output
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if devMode {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	// Directories
	testDocsPath := "testdata"
	outputsDir := "outputs"
	ticketDir := "../../ttmp/2025/12/15/RMQ-RMDOC-WEB-001--build-remarquee-ui-web-validation-tool-for-rmdoc-rendering"

	// Ensure outputs directory exists
	if err := os.MkdirAll(outputsDir, 0755); err != nil {
		log.Fatal().Err(err).Msg("Failed to create outputs directory")
	}

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/test-documents", handleTestDocuments)
	mux.HandleFunc("/api/document/{id}/inspect", api.HandleInspect(testDocsPath))
	mux.HandleFunc("/api/document/{id}/structure", api.HandleInternalStructure(testDocsPath))
	mux.HandleFunc("/api/render/background", api.HandleRenderBackground(testDocsPath, outputsDir))
	mux.HandleFunc("/api/render/legacy", api.HandleRenderLegacy(testDocsPath, outputsDir))
	mux.HandleFunc("/api/outputs/{filename}", api.HandleOutputs(outputsDir))
	mux.HandleFunc("/api/validation", api.HandleValidation(ticketDir))

	// Static assets
	if devMode {
		log.Info().Msg("Running in DEV mode: expecting Vite dev server on :5173")
	} else {
		log.Info().Msg("Running in PROD mode: serving embedded assets")
		frontendFS, err := GetFrontendFS()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get frontend filesystem")
		}
		fileServer := http.FileServer(http.FS(frontendFS))
		mux.Handle("/", fileServer)
	}

	addr := ":" + port
	log.Info().Str("addr", addr).Msg("remarquee-ui server listening")
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal().Err(err).Msg("Server failed")
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Error().Err(err).Msg("Failed to write health response")
	}
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
	if _, err := w.Write(data); err != nil {
		log.Error().Err(err).Msg("Failed to write test documents response")
	}
}
