package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/remarquee/cmd/remarquee-ui/api"
)

func newTestMux(t *testing.T, outputsDir string) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()

	// Keep the same registration style as main.go.
	testDocsPath := "testdata"
	ticketDir := "../../ttmp/2025/12/15/RMQ-RMDOC-WEB-001--build-remarquee-ui-web-validation-tool-for-rmdoc-rendering"

	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/test-documents", handleTestDocuments)
	mux.HandleFunc("/api/document/{id}/inspect", api.HandleInspect(testDocsPath))
	mux.HandleFunc("/api/document/{id}/structure", api.HandleInternalStructure(testDocsPath))
	mux.HandleFunc("/api/render/background", api.HandleRenderBackground(testDocsPath, outputsDir))
	mux.HandleFunc("/api/render/legacy", api.HandleRenderLegacy(testDocsPath, outputsDir))
	mux.HandleFunc("/api/outputs/{filename}", api.HandleOutputs(outputsDir))
	mux.HandleFunc("/api/validation", api.HandleValidation(ticketDir))

	return mux
}

func TestRouting_DocumentInspect_OK(t *testing.T) {
	outputsDir := t.TempDir()
	mux := newTestMux(t, outputsDir)

	req := httptest.NewRequest(http.MethodGet, "/api/document/cpage-pdf/inspect", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v body=%s", err, rr.Body.String())
	}
	if payload["error"] != nil {
		t.Fatalf("unexpected error field: %v", payload["error"])
	}
}

func TestRouting_DocumentInspect_UnknownDoc_404(t *testing.T) {
	outputsDir := t.TempDir()
	mux := newTestMux(t, outputsDir)

	req := httptest.NewRequest(http.MethodGet, "/api/document/does-not-exist/inspect", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestRouting_DocumentInspect_ExtraSegment_404(t *testing.T) {
	outputsDir := t.TempDir()
	mux := newTestMux(t, outputsDir)

	req := httptest.NewRequest(http.MethodGet, "/api/document/cpage-pdf/inspect/extra", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestRouting_DocumentInspect_WrongMethod_405(t *testing.T) {
	outputsDir := t.TempDir()
	mux := newTestMux(t, outputsDir)

	req := httptest.NewRequest(http.MethodPost, "/api/document/cpage-pdf/inspect", bytes.NewReader([]byte("{}")))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestRouting_Outputs_OK(t *testing.T) {
	outputsDir := t.TempDir()
	mux := newTestMux(t, outputsDir)

	filename := "test.pdf"
	content := []byte("%PDF-1.4\n% test\n")
	if err := os.WriteFile(filepath.Join(outputsDir, filename), content, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/outputs/"+filename, nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Content-Type"); got != "application/pdf" {
		t.Fatalf("Content-Type=%q", got)
	}
}

func TestRouting_Outputs_PathTraversal_400(t *testing.T) {
	outputsDir := t.TempDir()
	mux := newTestMux(t, outputsDir)

	// Note: a literal ".." path segment is normalized/redirected by the stdlib mux,
	// so we test a traversal-like filename that still reaches the handler.
	req := httptest.NewRequest(http.MethodGet, "/api/outputs/..evil.pdf", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
