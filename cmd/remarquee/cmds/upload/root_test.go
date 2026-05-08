package upload

import (
	"testing"
)

func TestSanitizePDFName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Spaces become underscores (the most common 400 trigger).
		{"Big Brother Review.pdf", "Big_Brother_Review.pdf"},
		{"My Document Name.pdf", "My_Document_Name.pdf"},

		// Special characters removed.
		{"CSSVD-FLEX-JS-API - Guide.pdf", "CSSVD-FLEX-JS-API_-_Guide.pdf"},
		{"doc (v2).pdf", "doc_v2.pdf"},
		{"doc:v1.pdf", "docv1.pdf"},

		// Already clean names pass through.
		{"simple-name.pdf", "simple-name.pdf"},
		{"My_Document.pdf", "My_Document.pdf"},

		// Dashes are allowed (they're safe for rmapi).
		{"a -- b.pdf", "a_--_b.pdf"},

		// Leading/trailing underscores and dashes stripped.
		{"_leading.pdf", "leading.pdf"},
		{"-leading.pdf", "leading.pdf"},
		{"trailing_.pdf", "trailing.pdf"},

		// Empty after sanitization becomes "document".
		{"(:).pdf", "document.pdf"},

		// Path component — only basename is sanitized.
		{"/tmp/My Doc.pdf", "My_Doc.pdf"},
	}

	for _, tc := range tests {
		got := sanitizePDFName(tc.input)
		if got != tc.expected {
			t.Errorf("sanitizePDFName(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}
