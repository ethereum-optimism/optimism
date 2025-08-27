package engine

import (
	"errors"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

var ErrNoFCUNeeded = errors.New("no FCU call was needed")

// BuildTemporaryError indicates a retryable condition while building.
type BuildTemporaryError struct {
	Err     error
	FCEvent *ForkchoiceUpdateEvent
}

func (e *BuildTemporaryError) Error() string { return e.Err.Error() }
func (e *BuildTemporaryError) Unwrap() error { return e.Err }

// BuildPrestateError indicates the engine pre-state needs to be reset.
type BuildPrestateError struct {
	Err     error
	FCEvent *ForkchoiceUpdateEvent
}

func (e *BuildPrestateError) Error() string { return e.Err.Error() }
func (e *BuildPrestateError) Unwrap() error { return e.Err }

// BuildInvalidAttributesError indicates the payload attributes are invalid.
type BuildInvalidAttributesError struct {
	Attributes *derive.AttributesWithParent
	Err        error
	FCEvent    *ForkchoiceUpdateEvent
}

func (e *BuildInvalidAttributesError) Error() string { return e.Err.Error() }
func (e *BuildInvalidAttributesError) Unwrap() error { return e.Err }

// BuildCriticalError indicates an unrecoverable/unknown failure.
type BuildCriticalError struct {
	Err     error
	FCEvent *ForkchoiceUpdateEvent
}

func (e *BuildCriticalError) Error() string { return e.Err.Error() }
func (e *BuildCriticalError) Unwrap() error { return e.Err }

// SealExpiredError indicates sealing timed out or failed.
type SealExpiredError struct {
	FCEvent     *ForkchoiceUpdateEvent
	Info        eth.PayloadInfo
	Err         error
	Concluding  bool
	DerivedFrom eth.L1BlockRef
}

func (e *SealExpiredError) Error() string { return e.Err.Error() }
func (e *SealExpiredError) Unwrap() error { return e.Err }

// SealInvalidError indicates the produced payload failed sanity checks
// or could not be decoded into an L2 block ref.
type SealInvalidError struct {
	FCEvent     *ForkchoiceUpdateEvent
	Info        eth.PayloadInfo
	Err         error
	Concluding  bool
	DerivedFrom eth.L1BlockRef
}

func (e *SealInvalidError) Error() string { return e.Err.Error() }
func (e *SealInvalidError) Unwrap() error { return e.Err }

// PayloadInvalidError indicates the engine evaluated the payload as invalid.
type PayloadInvalidError struct {
	FCEvent  *ForkchoiceUpdateEvent
	Envelope *eth.ExecutionPayloadEnvelope
	Err      error
}

func (e *PayloadInvalidError) Error() string { return e.Err.Error() }
func (e *PayloadInvalidError) Unwrap() error { return e.Err }

// DepositsOnlyRequest is a control-flow signal to request a deposits-only
// attributes build when Holocene is active and a derived block is invalid.
type DepositsOnlyRequest struct {
	FCEvent     *ForkchoiceUpdateEvent
	Parent      eth.BlockID
	DerivedFrom eth.L1BlockRef
}

func (e *DepositsOnlyRequest) Error() string { return "request deposits-only attributes" }

// InvalidPrestateError indicates the initial pre-state before starting
// block building was invalid (e.g. unsafe behind finalized).
type InvalidPrestateError struct {
	Err     error
	FCEvent *ForkchoiceUpdateEvent
}

func (e *InvalidPrestateError) Error() string { return e.Err.Error() }
func (e *InvalidPrestateError) Unwrap() error { return e.Err }
