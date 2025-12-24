package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-go-golems/remarquee/pkg/rmdoc"
	rmdocdebug "github.com/go-go-golems/remarquee/pkg/rmdoc/debug"
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

		docID := r.PathValue("id")
		if docID == "" {
			log.Warn().Msg("Missing document id in path")
			http.Error(w, "Missing document id in path", http.StatusBadRequest)
			return
		}
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
		ctx := r.Context()
		doc, err := rmdoc.OpenFile(ctx, docPath)
		if err != nil {
			log.Printf("Failed to open rmdoc %s: %v", docPath, err)
			respondJSON(w, http.StatusInternalServerError, InternalStructureResponse{
				Error: fmt.Sprintf("Failed to open document: %v", err),
			})
			return
		}

		allFiles, err := rmdocdebug.ListArchiveFiles(ctx, docPath)
		if err != nil {
			log.Printf("Failed to list archive files in %s: %v", docPath, err)
			respondJSON(w, http.StatusInternalServerError, InternalStructureResponse{
				Error: fmt.Sprintf("Failed to list archive files: %v", err),
			})
			return
		}

		rmFilesDebug, err := rmdocdebug.InspectRMFiles(ctx, docPath)
		if err != nil {
			log.Printf("Failed to inspect .rm files in %s: %v", docPath, err)
			respondJSON(w, http.StatusInternalServerError, InternalStructureResponse{
				Error: fmt.Sprintf("Failed to inspect .rm files: %v", err),
			})
			return
		}

		rmFiles := make([]RMFileInfo, 0, len(rmFilesDebug))
		for _, f := range rmFilesDebug {
			rmFiles = append(rmFiles, RMFileInfo{
				PageID:   f.PageID,
				Filename: f.Filename,
				Size:     f.Size,
				Version:  f.Version,
			})
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
