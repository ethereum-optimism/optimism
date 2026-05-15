package interop

import (
	"errors"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// ErrNotActive signals that ts is before interop activation.
var ErrNotActive = errors.New("interop not active for timestamp")

// ErrInteropDisabled signals that the interop verifier is not configured.
// Callers may use the pre-interop optimistic composition path.
var ErrInteropDisabled = errors.New("interop verifier disabled")

// ErrBeforeHandoff signals that ts is post-activation but older than the
// local startup handoff window. This node intentionally does not serve
// optimistic superroots for the timestamp because doing so would require
// SafeDB history that the startup backfill did not cover.
var ErrBeforeHandoff = errors.New("timestamp below startup handoff")

// ErrBeforeVerifiedDB signals that ts is post-activation but below the
// verifier's first verifiable timestamp on this node but within the startup
// handoff window. The timestamp is covered by backfilled logs and available
// SafeDB history: no VerifiedResult will be produced here, but callers can
// treat optimistic outputs as canonical.
var ErrBeforeVerifiedDB = errors.New("timestamp below verified-db start")

// ErrNotStarted signals that Start has not yet populated i.ctx. Callers
// should retry after startup completes.
var ErrNotStarted = errors.New("interop activity not started")

// VerifiedResultReader exposes committed VerifiedResults to non-interop
// activities. Errors discriminate the regime:
//   - nil:                  verified entry returned
//   - ErrInteropDisabled:   interop not configured; compose from optimistic outputs
//   - ErrNotActive:         pre-activation; do not query interop superroots
//   - ErrBeforeHandoff:     post-activation but below local backfill coverage
//   - ErrBeforeVerifiedDB:  post-activation and inside startup handoff; compose
//     from optimistic outputs
//   - ethereum.NotFound:    verifier may eventually produce a result but has
//     not yet — return Data = nil and let CurrentL1 communicate progress
//
// currentL1 is the verifier's CurrentL1 observed atomically with the
// verifiedDB read. Callers must combine it with their own CurrentL1 view
// (via min) so the response cannot report a CurrentL1 that is inconsistent
// with the verifiedDB observation.
type VerifiedResultReader interface {
	VerifiedResultAtTimestamp(ts uint64) (result VerifiedResult, currentL1 eth.BlockID, err error)
}

// NoopVerifiedResultReader is used when interop is not configured.
type NoopVerifiedResultReader struct{}

func (NoopVerifiedResultReader) VerifiedResultAtTimestamp(uint64) (VerifiedResult, eth.BlockID, error) {
	return VerifiedResult{}, eth.BlockID{}, ErrInteropDisabled
}
