package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// ValidationReview matches the frontend ValidationReview interface
type ValidationReview struct {
	Status string `json:"status"` // "pass" | "fail" | "unknown"
	Notes  string `json:"notes"`
}

// ValidationSession matches the frontend ValidationSession interface
type ValidationSession struct {
	DocumentID string           `json:"documentId"`
	Actions    []string         `json:"actions"`
	Review     ValidationReview `json:"review"`
	Timestamp  int64            `json:"timestamp"`
}

// ValidationResponse is the JSON response for the validation endpoint
type ValidationResponse struct {
	SessionID  string   `json:"session_id"`
	SavedPaths []string `json:"saved_paths"`
	Error      string   `json:"error,omitempty"`
}

// HandleValidation handles POST /api/validation
func HandleValidation(ticketDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var session ValidationSession
		if err := json.NewDecoder(r.Body).Decode(&session); err != nil {
			respondJSON(w, http.StatusBadRequest, ValidationResponse{
				Error: fmt.Sprintf("Invalid request body: %v", err),
			})
			return
		}

		// Generate session ID from timestamp
		sessionID := fmt.Sprintf("validation-%d", time.Now().Unix())

		// Create validation directory if it doesn't exist
		validationDir := filepath.Join(ticketDir, "reference", "validation")
		if err := os.MkdirAll(validationDir, 0755); err != nil {
			log.Printf("Failed to create validation directory: %v", err)
			respondJSON(w, http.StatusInternalServerError, ValidationResponse{
				Error: fmt.Sprintf("Failed to create validation directory: %v", err),
			})
			return
		}

		var savedPaths []string

		// Write JSON file
		jsonPath := filepath.Join(validationDir, sessionID+".json")
		jsonData, err := json.MarshalIndent(session, "", "  ")
		if err != nil {
			log.Printf("Failed to marshal validation session: %v", err)
			respondJSON(w, http.StatusInternalServerError, ValidationResponse{
				Error: fmt.Sprintf("Failed to marshal session: %v", err),
			})
			return
		}
		if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
			log.Printf("Failed to write JSON file: %v", err)
			respondJSON(w, http.StatusInternalServerError, ValidationResponse{
				Error: fmt.Sprintf("Failed to write JSON file: %v", err),
			})
			return
		}
		savedPaths = append(savedPaths, jsonPath)

		// Write Markdown file
		mdPath := filepath.Join(validationDir, sessionID+".md")
		mdContent := formatValidationMarkdown(session, sessionID)
		if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
			log.Printf("Failed to write Markdown file: %v", err)
			respondJSON(w, http.StatusInternalServerError, ValidationResponse{
				Error: fmt.Sprintf("Failed to write Markdown file: %v", err),
			})
			return
		}
		savedPaths = append(savedPaths, mdPath)

		log.Printf("Saved validation session %s: %v", sessionID, savedPaths)
		respondJSON(w, http.StatusOK, ValidationResponse{
			SessionID:  sessionID,
			SavedPaths: savedPaths,
		})
	}
}

// formatValidationMarkdown generates a Markdown representation of the validation session
func formatValidationMarkdown(session ValidationSession, sessionID string) string {
	timestamp := time.Unix(session.Timestamp/1000, 0).Format(time.RFC3339)

	md := fmt.Sprintf(`# Validation Session: %s

**Document ID:** %s  
**Timestamp:** %s  
**Status:** %s

## Actions Performed

`, sessionID, session.DocumentID, timestamp, session.Review.Status)

	if len(session.Actions) == 0 {
		md += "- (none)\n"
	} else {
		for _, action := range session.Actions {
			md += fmt.Sprintf("- %s\n", action)
		}
	}

	md += "\n## Review Notes\n\n"
	if session.Review.Notes == "" {
		md += "(no notes)\n"
	} else {
		md += session.Review.Notes + "\n"
	}

	return md
}
