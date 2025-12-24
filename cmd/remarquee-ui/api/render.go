package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-go-golems/remarquee/pkg/rmdoc"
	"github.com/go-go-golems/remarquee/pkg/rmdoc/render"
	"github.com/juruen/rmapi/annotations"
)

// RenderRequest is the JSON body for render endpoints
type RenderRequest struct {
	DocumentID string `json:"document_id"`
}

// RenderResponse is the JSON response for render endpoints
type RenderResponse struct {
	JobID      string `json:"job_id"`
	OutputPath string `json:"output_path"`
	Error      string `json:"error,omitempty"`
}

// HandleRenderBackground handles POST /api/render/background
func HandleRenderBackground(testDocsPath, outputsDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req RenderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, RenderResponse{
				Error: fmt.Sprintf("Invalid request body: %v", err),
			})
			return
		}

		if req.DocumentID == "" {
			respondJSON(w, http.StatusBadRequest, RenderResponse{
				Error: "document_id is required",
			})
			return
		}

		// Find document path
		docPath, err := findDocumentPath(testDocsPath, req.DocumentID)
		if err != nil {
			log.Printf("Failed to find document %s: %v", req.DocumentID, err)
			respondJSON(w, http.StatusNotFound, RenderResponse{
				Error: fmt.Sprintf("Document not found: %s", req.DocumentID),
			})
			return
		}

		// Open document
		ctx := r.Context()
		doc, err := rmdoc.OpenFile(ctx, docPath)
		if err != nil {
			log.Printf("Failed to open rmdoc %s: %v", docPath, err)
			respondJSON(w, http.StatusInternalServerError, RenderResponse{
				Error: fmt.Sprintf("Failed to open document: %v", err),
			})
			return
		}

		// Render background PDF
		pdfBytes, err := render.BuildBackgroundPDF(ctx, doc, render.BackgroundOptions{})
		if err != nil {
			log.Printf("Failed to build background PDF: %v", err)
			respondJSON(w, http.StatusInternalServerError, RenderResponse{
				Error: fmt.Sprintf("Failed to build background PDF: %v", err),
			})
			return
		}

		// Write output
		jobID := fmt.Sprintf("%s-background-%d", req.DocumentID, time.Now().Unix())
		outputFilename := fmt.Sprintf("%s.pdf", jobID)
		outputPath := filepath.Join(outputsDir, outputFilename)

		if err := os.WriteFile(outputPath, pdfBytes, 0644); err != nil {
			log.Printf("Failed to write output PDF: %v", err)
			respondJSON(w, http.StatusInternalServerError, RenderResponse{
				Error: fmt.Sprintf("Failed to write output: %v", err),
			})
			return
		}

		log.Printf("Generated background PDF: %s", outputPath)
		respondJSON(w, http.StatusOK, RenderResponse{
			JobID:      jobID,
			OutputPath: outputFilename,
		})
	}
}

// HandleRenderLegacy handles POST /api/render/legacy
func HandleRenderLegacy(testDocsPath, outputsDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req RenderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, RenderResponse{
				Error: fmt.Sprintf("Invalid request body: %v", err),
			})
			return
		}

		if req.DocumentID == "" {
			respondJSON(w, http.StatusBadRequest, RenderResponse{
				Error: "document_id is required",
			})
			return
		}

		// Find document path
		docPath, err := findDocumentPath(testDocsPath, req.DocumentID)
		if err != nil {
			log.Printf("Failed to find document %s: %v", req.DocumentID, err)
			respondJSON(w, http.StatusNotFound, RenderResponse{
				Error: fmt.Sprintf("Document not found: %s", req.DocumentID),
			})
			return
		}

		// Verify it's a legacy document
		ctx := r.Context()
		doc, err := rmdoc.OpenFile(ctx, docPath)
		if err != nil {
			log.Printf("Failed to open rmdoc %s: %v", docPath, err)
			respondJSON(w, http.StatusInternalServerError, RenderResponse{
				Error: fmt.Sprintf("Failed to open document: %v", err),
			})
			return
		}

		if doc.Schema != rmdoc.SchemaLegacy {
			respondJSON(w, http.StatusBadRequest, RenderResponse{
				Error: fmt.Sprintf("Document is not legacy (schema: %s), use /api/render/background instead", doc.Schema.String()),
			})
			return
		}

		// Generate legacy PDF using rmapi
		jobID := fmt.Sprintf("%s-legacy-%d", req.DocumentID, time.Now().Unix())
		outputFilename := fmt.Sprintf("%s.pdf", jobID)
		outputPath := filepath.Join(outputsDir, outputFilename)

		options := annotations.PdfGeneratorOptions{
			AddPageNumbers:  false,
			AllPages:        true,
			AnnotationsOnly: false,
		}

		generator := annotations.CreatePdfGenerator(docPath, outputPath, options)
		if err := generator.Generate(); err != nil {
			log.Printf("Failed to generate legacy PDF: %v", err)
			respondJSON(w, http.StatusInternalServerError, RenderResponse{
				Error: fmt.Sprintf("Failed to generate legacy PDF: %v", err),
			})
			return
		}

		log.Printf("Generated legacy PDF: %s", outputPath)
		respondJSON(w, http.StatusOK, RenderResponse{
			JobID:      jobID,
			OutputPath: outputFilename,
		})
	}
}
