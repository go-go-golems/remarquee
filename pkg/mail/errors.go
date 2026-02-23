package mail

import "fmt"

// MailError is a structured error returned by IMAP/Sieve operations.
type MailError struct {
	Name    string                 `json:"name"`
	Message string                 `json:"message"`
	Code    string                 `json:"code,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
	Source  string                 `json:"source"` // "imap" or "sieve"
}

func (e *MailError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("[%s/%s] %s", e.Source, e.Name, e.Message)
	}
	return fmt.Sprintf("[%s/%s] %s", e.Source, e.Name, e.Message)
}

// NewMailError creates a MailError with the given name, message, and source.
func NewMailError(name, message, source string) *MailError {
	return &MailError{Name: name, Message: message, Source: source}
}

// NewSieveError creates a Sieve-specific error.
func NewSieveError(name, message string) *MailError {
	return &MailError{Name: name, Message: message, Source: "sieve"}
}

// NewIMAPError creates an IMAP-specific error.
func NewIMAPError(name, message string) *MailError {
	return &MailError{Name: name, Message: message, Source: "imap"}
}
