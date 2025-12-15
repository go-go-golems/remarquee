package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/remarquee/pkg/rmdoc"
)

// InspectResponse represents the JSON response for the inspect endpoint
type InspectResponse struct {
	UUID          string          `json:"uuid"`
	Schema        string          `json:"schema"`
	DocType       string          `json:"docType"`
	PageCount     int             `json:"pageCount"`
	Pages         []PageRefJSON   `json:"pages"`
	HasPayloadPDF bool            `json:"hasPayloadPDF"`
	Error         string          `json:"error,omitempty"`
}

// PageRefJSON is the JSON representation of a page reference
type PageRefJSON struct {
	Index         int    `json:"index"`
	PageID        string `json:"pageId"`
	SourcePDFPage int    `json:"sourcePdfPage"`
	Template      string `json:"template"`
	Deleted       bool   `json:"deleted"`
}

// HandleInspect handles GET /api/document/:id/inspect
func HandleInspect(testDocsPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract document ID from URL path
		// Expected: /api/document/{id}/inspect
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(pathParts) != 4 || pathParts[0] != "api" || pathParts[1] != "document" || pathParts[3] != "inspect" {
			http.Error(w, "Invalid path format, expected /api/document/{id}/inspect", http.StatusBadRequest)
			return
		}
		docID := pathParts[2]

		// Load test documents manifest to find the document path
		docPath, err := findDocumentPath(testDocsPath, docID)
		if err != nil {
			log.Printf("Failed to find document %s: %v", docID, err)
			respondJSON(w, http.StatusNotFound, InspectResponse{
				Error: fmt.Sprintf("Document not found: %s", docID),
			})
			return
		}

		// Open and inspect the document using pkg/rmdoc
		ctx := context.Background()
		doc, err := rmdoc.OpenFile(ctx, docPath)
		if err != nil {
			log.Printf("Failed to open rmdoc %s: %v", docPath, err)
			respondJSON(w, http.StatusInternalServerError, InspectResponse{
				Error: fmt.Sprintf("Failed to open document: %v", err),
			})
			return
		}

		// Convert to JSON response
		pages := make([]PageRefJSON, len(doc.Pages))
		for i, page := range doc.Pages {
			pages[i] = PageRefJSON{
				Index:         page.Index,
				PageID:        page.PageID,
				SourcePDFPage: page.SourcePDFPage,
				Template:      page.Template,
				Deleted:       page.Deleted,
			}
		}

	response := InspectResponse{
		UUID:          doc.UUID,
		Schema:        doc.Schema.String(),
		DocType:       doc.Type.String(),
		PageCount:     len(doc.Pages),
		Pages:         pages,
		HasPayloadPDF: len(doc.PayloadPDF) > 0,
	}

		respondJSON(w, http.StatusOK, response)
	}
}

// findDocumentPath looks up a document ID in the test documents manifest
func findDocumentPath(testDocsPath, docID string) (string, error) {
	data, err := readJSON(filepath.Join(testDocsPath, "test-documents.json"))
	if err != nil {
		return "", fmt.Errorf("failed to read manifest: %w", err)
	}

	docs, ok := data.([]interface{})
	if !ok {
		return "", fmt.Errorf("invalid manifest format")
	}

	for _, item := range docs {
		doc, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if id, ok := doc["id"].(string); ok && id == docID {
			if path, ok := doc["path"].(string); ok {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("document %s not found in manifest", docID)
}

