package mail

import "github.com/emersion/go-imap/v2"

// UID is an alias for imap.UID (uint32).
type UID = imap.UID

// Flag is an alias for imap.Flag (string).
type Flag = imap.Flag

// StoreFlagsOp is an alias for imap.StoreFlagsOp.
type StoreFlagsOp = imap.StoreFlagsOp

// Common IMAP flags.
const (
	FlagSeen     = imap.FlagSeen
	FlagAnswered = imap.FlagAnswered
	FlagFlagged  = imap.FlagFlagged
	FlagDeleted  = imap.FlagDeleted
	FlagDraft    = imap.FlagDraft
)

// Store flag operations.
const (
	StoreFlagsAdd = imap.StoreFlagsAdd
	StoreFlagsDel = imap.StoreFlagsDel
	StoreFlagsSet = imap.StoreFlagsSet
)
