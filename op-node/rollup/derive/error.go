package derive

import (
	"errors"
	"fmt"
)

// These are the underlying errors that callers can inspect to estimate
// severity of the error and either backoff, reset pipeline or halt.
var (
	// ErrTemporary is returned when the error is temporary in nature and
	// recovery is possible, so the caller can reattempt with a backoff
	ErrTemporary = errors.New("temporary error")
	// ErrCritical is returned when there is a reorg and the pipeline MUST be
	// reset
	ErrCritical = errors.New("critical error")
	// ErrHalt is returned when there is a system misconfiguration and the node
	// needs to be halted.
	ErrHalt = errors.New("misconfiguration error")
)

// ErrorCode identifies a kind of error.
type ErrorCode int

// These constants are used to identify a specific derivation Error.
const (
	ErrFetchFailed ErrorCode = iota
	ErrL1OriginMismatch
	ErrDeriveFailed
	ErrEpochHashMismatch
	ErrInfoByHashFailed
	ErrL1InfoTxFailed
	ErrOpenDataFailed
	ErrIngestDataEmpty
	ErrL1BlockRefFailed
	ErrForkChoiceUpdateFailed
	ErrPayloadPrepare
	ErrInsertPayloadFailed
	ErrPayloadProcess
	ErrUnsafePayloadFailed
	ErrPayloadBlockRefFailed
	ErrInsertBlockFailed
	ErrL2BlockRefHeadFailed
	ErrFindL2HeadsFailed
)

// Map of ErrorCode values back to their constant names for pretty printing.
var errorCodeStrings = map[ErrorCode]string{
	ErrFetchFailed:            "ErrFetchFailed",
	ErrL1OriginMismatch:       "ErrL1OriginMismatch",
	ErrDeriveFailed:           "ErrDeriveFailed",
	ErrEpochHashMismatch:      "ErrEpochHashMismatch",
	ErrInfoByHashFailed:       "ErrInfoByHashFailed",
	ErrL1InfoTxFailed:         "ErrL1InfoTxFailed",
	ErrOpenDataFailed:         "ErrOpenDataFailed",
	ErrIngestDataEmpty:        "ErrIngestDataEmpty",
	ErrL1BlockRefFailed:       "ErrL1BlockRefFailed",
	ErrForkChoiceUpdateFailed: "ErrForkChoiceUpdateFailed",
	ErrPayloadPrepare:         "ErrPayloadPrepare",
	ErrInsertPayloadFailed:    "ErrInsertPayloadFailed",
	ErrPayloadProcess:         "ErrPayloadProcess",
	ErrUnsafePayloadFailed:    "ErrUnsafePayloadFailed",
	ErrPayloadBlockRefFailed:  "ErrPayloadBlockRefFailed",
	ErrInsertBlockFailed:      "ErrInsertBlockFailed",
	ErrL2BlockRefHeadFailed:   "ErrL2BlockRefHeadFailed",
	ErrFindL2HeadsFailed:      "ErrFindL2HeadsFailed",
}

// String returns the ErrorCode as a human-readable name.
func (e ErrorCode) String() string {
	if s := errorCodeStrings[e]; s != "" {
		return s
	}
	return fmt.Sprintf("Unknown ErrorCode (%d)", int(e))
}

// Error provides a single type for errors that can happen during derivation.
type Error struct {
	ErrorCode   ErrorCode // Describes the kind of error
	Description string    // Human readable description of the issue
	Err         error     // Underlying error
}

// Error satisfies the error interface and prints human-readable errors.
func (e Error) Error() string {
	if e.Err != nil {
		return e.Err.Error() + ": " + e.Description
	}
	return e.Description
}

// Unwrap returns the underlying error
func (e Error) Unwrap() error {
	return e.Err
}

// makeError creates an Error given a set of arguments.  The error code must
// be one of the error codes provided by this package.
func makeError(c ErrorCode, desc string, err error) Error {
	return Error{ErrorCode: c, Description: desc, Err: err}
}
