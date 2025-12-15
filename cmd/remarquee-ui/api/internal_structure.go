package api

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/remarquee/pkg/rmdoc"
	"github.com/rs/zerolog/log"
)

// RMFileInfo represents metadata about a .rm file
type RMFileInfo struct {
	PageID   string `json:"pageId"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	Version  string `json:"version,omitempty"`
}

// InternalStructureResponse represents the detailed internal structure
type InternalStructureResponse struct {
	ContentJSON  string       `json:"contentJson"`
	MetadataJSON string       `json:"metadataJson"`
	RMFiles      []RMFileInfo `json:"rmFiles"`
	AllFiles     []string     `json:"allFiles"`
	Error        string       `json:"error,omitempty"`
}

// HandleInternalStructure handles GET /api/document/:id/structure
func HandleInternalStructure(testDocsPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract document ID from URL path
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		log.Info().Strs("pathParts", pathParts).Msg("Parsing URL path")
		if len(pathParts) != 4 || pathParts[0] != "api" || pathParts[1] != "document" || pathParts[3] != "structure" {
			log.Warn().Strs("pathParts", pathParts).Msg("Invalid path format")
			http.Error(w, "Invalid path format, expected /api/document/{id}/structure", http.StatusBadRequest)
			return
		}
		docID := pathParts[2]
		log.Info().Str("docID", docID).Msg("Processing internal structure request")

		// Load test documents manifest to find the document path
		docPath, err := findDocumentPath(testDocsPath, docID)
		if err != nil {
			log.Printf("Failed to find document %s: %v", docID, err)
			respondJSON(w, http.StatusNotFound, InternalStructureResponse{
				Error: fmt.Sprintf("Document not found: %s", docID),
			})
			return
		}

		// Open the rmdoc to get metadata
		ctx := context.Background()
		doc, err := rmdoc.OpenFile(ctx, docPath)
		if err != nil {
			log.Printf("Failed to open rmdoc %s: %v", docPath, err)
			respondJSON(w, http.StatusInternalServerError, InternalStructureResponse{
				Error: fmt.Sprintf("Failed to open document: %v", err),
			})
			return
		}

		// Open the zip file to inspect all files
		zipFile, err := os.Open(docPath)
		if err != nil {
			log.Printf("Failed to open zip file %s: %v", docPath, err)
			respondJSON(w, http.StatusInternalServerError, InternalStructureResponse{
				Error: fmt.Sprintf("Failed to open zip file: %v", err),
			})
			return
		}
		defer zipFile.Close()

		zipStat, err := zipFile.Stat()
		if err != nil {
			log.Printf("Failed to stat zip file: %v", err)
			respondJSON(w, http.StatusInternalServerError, InternalStructureResponse{
				Error: fmt.Sprintf("Failed to stat zip file: %v", err),
			})
			return
		}

		zipReader, err := zip.NewReader(zipFile, zipStat.Size())
		if err != nil {
			log.Printf("Failed to create zip reader: %v", err)
			respondJSON(w, http.StatusInternalServerError, InternalStructureResponse{
				Error: fmt.Sprintf("Failed to read zip: %v", err),
			})
			return
		}

		// Collect all files and .rm file info
		var rmFiles []RMFileInfo
		var allFiles []string

		for _, file := range zipReader.File {
			allFiles = append(allFiles, file.Name)

			if strings.HasSuffix(file.Name, ".rm") {
				pageID := strings.TrimSuffix(filepath.Base(file.Name), ".rm")
				
				// Try to read first 43 bytes to get version
				// Format: "reMarkable .lines file, version=X      " (43 bytes total)
				version := "unknown"
				if rc, err := file.Open(); err == nil {
					header := make([]byte, 43)
					if n, err := rc.Read(header); err == nil && n == 43 {
						headerStr := string(header)
						log.Debug().Str("header", headerStr).Str("pageID", pageID).Msg("RM file header")
						if strings.Contains(headerStr, "version=3") {
							version = "V3"
						} else if strings.Contains(headerStr, "version=5") {
							version = "V5"
						} else if strings.Contains(headerStr, "version=6") {
							version = "V6"
						} else {
							log.Warn().Str("header", headerStr).Str("pageID", pageID).Msg("Unknown .rm version format")
						}
					} else {
						log.Warn().Int("bytesRead", n).Str("pageID", pageID).Msg("Failed to read full .rm header")
					}
					rc.Close()
				} else {
					log.Warn().Err(err).Str("pageID", pageID).Msg("Failed to open .rm file for version detection")
				}

				rmFiles = append(rmFiles, RMFileInfo{
					PageID:   pageID,
					Filename: file.Name,
					Size:     int64(file.UncompressedSize64),
					Version:  version,
				})
			}
		}

		// Pretty-print JSONs
		var contentJSONPretty, metadataJSONPretty string

		if len(doc.ContentJSON) > 0 {
			var contentObj map[string]interface{}
			if err := json.Unmarshal(doc.ContentJSON, &contentObj); err == nil {
				if prettyBytes, err := json.MarshalIndent(contentObj, "", "  "); err == nil {
					contentJSONPretty = string(prettyBytes)
				}
			}
		}

		if len(doc.MetadataJSON) > 0 {
			var metadataObj map[string]interface{}
			if err := json.Unmarshal(doc.MetadataJSON, &metadataObj); err == nil {
				if prettyBytes, err := json.MarshalIndent(metadataObj, "", "  "); err == nil {
					metadataJSONPretty = string(prettyBytes)
				}
			}
		}

		log.Info().
			Str("docID", docID).
			Int("rmFileCount", len(rmFiles)).
			Int("totalFiles", len(allFiles)).
			Msg("Successfully processed internal structure")

		respondJSON(w, http.StatusOK, InternalStructureResponse{
			ContentJSON:  contentJSONPretty,
			MetadataJSON: metadataJSONPretty,
			RMFiles:      rmFiles,
			AllFiles:     allFiles,
		})
	}
}

