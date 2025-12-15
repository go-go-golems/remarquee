package api

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/remarquee/pkg/rmdoc"
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
		if len(pathParts) != 4 || pathParts[0] != "api" || pathParts[1] != "document" || pathParts[3] != "structure" {
			http.Error(w, "Invalid path format, expected /api/document/{id}/structure", http.StatusBadRequest)
			return
		}
		docID := pathParts[2]

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
				
				// Try to read first 32 bytes to get version
				version := "unknown"
				if rc, err := file.Open(); err == nil {
					header := make([]byte, 32)
					if n, err := rc.Read(header); err == nil && n >= 32 {
						// Check for V3, V5, or V6 markers
						headerStr := string(header[:32])
						if strings.Contains(headerStr, "reMarkable") {
							if header[32-1] == 3 {
								version = "V3"
							} else if header[32-1] == 5 {
								version = "V5"
							} else if header[32-1] == 6 {
								version = "V6"
							}
						}
					}
					rc.Close()
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

		respondJSON(w, http.StatusOK, InternalStructureResponse{
			ContentJSON:  contentJSONPretty,
			MetadataJSON: metadataJSONPretty,
			RMFiles:      rmFiles,
			AllFiles:     allFiles,
		})
	}
}

