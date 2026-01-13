package filter

import (
	"fmt"
	"time"
)

// IngesterErrorReason indicates why an ingester entered an error state
type IngesterErrorReason int

const (
	// ErrorReorg indicates a true chain reorganization was detected
	ErrorReorg IngesterErrorReason = iota
	// ErrorConflict indicates a database conflict (app-level failure)
	ErrorConflict
	// ErrorValidationFailed indicates cross-unsafe validation failed
	ErrorValidationFailed
)

// String returns a human-readable name for the error reason
func (r IngesterErrorReason) String() string {
	switch r {
	case ErrorReorg:
		return "reorg"
	case ErrorConflict:
		return "conflict"
	case ErrorValidationFailed:
		return "validation_failed"
	default:
		return "unknown"
	}
}

// IngesterError represents an error state in a ChainIngester
type IngesterError struct {
	Reason    IngesterErrorReason
	Message   string
	Timestamp time.Time
}

func (e *IngesterError) Error() string {
	return fmt.Sprintf("%s: %s", e.Reason, e.Message)
}
