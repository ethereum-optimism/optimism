package interop

import (
	"errors"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// ErrNotActive signals that ts is before interop activation. Callers compose
// the super root from per-chain optimistic outputs.
var ErrNotActive = errors.New("interop not active for timestamp")

// ErrBeforeVerifiedDB signals that ts is post-activation but below the
// verifier's first verifiable timestamp on this node. The timestamp is
// already covered by the safe-head startup handoff: no VerifiedResult will
// be produced here, but callers can treat optimistic outputs as canonical.
var ErrBeforeVerifiedDB = errors.New("timestamp below verified-db start")

// ErrNotStarted signals that Start has not yet populated i.ctx. Callers
// should retry after startup completes.
var ErrNotStarted = errors.New("interop activity not started")

// VerifiedResultReader exposes committed VerifiedResults to non-interop
// activities. Errors discriminate the regime:
//   - nil:                  verified entry returned
//   - ErrNotActive:         pre-activation; compose from optimistic outputs
//   - ErrBeforeVerifiedDB:  post-activation but below firstVerifiable; the
//     handoff covers ts, so compose from optimistic outputs
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

// NoopVerifiedResultReader is used when interop is not configured: every
// call returns ErrNotActive.
type NoopVerifiedResultReader struct{}

func (NoopVerifiedResultReader) VerifiedResultAtTimestamp(uint64) (VerifiedResult, eth.BlockID, error) {
	return VerifiedResult{}, eth.BlockID{}, ErrNotActive
}
