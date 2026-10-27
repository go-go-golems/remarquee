package rmfiles

import "fmt"

import "github.com/pkg/errors"

// Sentinel errors returned by the service. The HTTP layer maps these to status
// codes; keep them coarse and stable.
var (
	// ErrNotFound means a referenced entry does not exist in the cloud tree.
	ErrNotFound = errors.New("entry not found")
	// ErrConflict means an entry with the requested name already exists.
	ErrConflict = errors.New("entry already exists")
	// ErrInvalid means the request is malformed (bad name, empty input, ...).
	ErrInvalid = errors.New("invalid request")
	// ErrNotDir means a path that should be a directory resolves to a file.
	ErrNotDir = errors.New("not a directory")
	// ErrUnsupported means an operation is not supported for this entry/file.
	ErrUnsupported = errors.New("unsupported")
)

// PartialDeleteError reports a delete batch that removed some records before
// failing, including failure of the final notification after all removals.
// Removed is the number of individual cloud records actually gone
// (verified against the reconciled cloud state when reconciliation ran);
// RemainingIDs are planned record IDs still present after the failure;
// FailedID is the planned record whose deletion failed. ReconcileErr is
// non-nil when re-synchronizing the local tree/index with authoritative cloud
// state itself failed, in which case the local representation may still
// disagree with the cloud and no rollback is claimed.
type PartialDeleteError struct {
	Removed      int
	RemainingIDs []string
	FailedID     string
	Err          error
	ReconcileErr error
}

func (e *PartialDeleteError) Error() string {
	msg := fmt.Sprintf("delete removed %d record(s) (%d remaining)",
		e.Removed, len(e.RemainingIDs))
	if e.FailedID != "" {
		msg += fmt.Sprintf("; failed at %q", e.FailedID)
	}
	if e.Err != nil {
		msg += fmt.Sprintf(": %v", e.Err)
	}
	if e.ReconcileErr != nil {
		msg += fmt.Sprintf("; local reconciliation failed: %v", e.ReconcileErr)
	}
	return msg
}

func (e *PartialDeleteError) Unwrap() error { return e.Err }
