package filter

import (
	"errors"
	"fmt"
)

// ErrInvalidLog indicates a malformed executing message log was encountered.
// This is a sentinel error that can be checked with errors.Is.
var ErrInvalidLog = errors.New("invalid executing message log")

// IngesterErrorReason indicates why an ingester entered an error state.
// These are ingestion-level errors (reorg, DB conflicts).
type IngesterErrorReason int

const (
	// ErrorReorg indicates a true chain reorganization was detected
	ErrorReorg IngesterErrorReason = iota
	// ErrorConflict indicates a database conflict (app-level failure)
	ErrorConflict
	// ErrorDataCorruption indicates a database I/O or corruption error
	ErrorDataCorruption
	// ErrorInvalidExecutingMessage indicates a malformed executing message log from the chain
	ErrorInvalidExecutingMessage
)

// String returns a human-readable name for the error reason
func (r IngesterErrorReason) String() string {
	switch r {
	case ErrorReorg:
		return "reorg"
	case ErrorConflict:
		return "conflict"
	case ErrorDataCorruption:
		return "data_corruption"
	case ErrorInvalidExecutingMessage:
		return "invalid_log"
	default:
		return "unknown"
	}
}

// IngesterError represents an error state in a ChainIngester.
// These are ingestion-level errors tracked per-chain.
type IngesterError struct {
	Reason  IngesterErrorReason
	Message string
}

func (e *IngesterError) Error() string {
	return fmt.Sprintf("%s: %s", e.Reason, e.Message)
}

// ValidatorError represents an error state in a CrossValidator.
// These are validation-level errors (invalid executing messages).
type ValidatorError struct {
	Message string
}

func (e *ValidatorError) Error() string {
	return e.Message
}
