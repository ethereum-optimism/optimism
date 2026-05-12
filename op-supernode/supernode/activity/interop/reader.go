package interop

import "errors"

// ErrNotActive signals that ts is before interop activation. Callers compose
// the super root from per-chain optimistic outputs.
var ErrNotActive = errors.New("interop not active for timestamp")

// ErrBeforeVerifiedDB signals that ts is post-activation but below the
// verifier's first verifiable timestamp on this node. No VerifiedResult will
// ever be produced for ts here, and the deny-list state needed to reproduce
// the canonical super root from optimistic data is not available either. The
// supernode cannot answer the call.
var ErrBeforeVerifiedDB = errors.New("timestamp below verified-db start")

// VerifiedResultReader exposes committed VerifiedResults to non-interop
// activities. Errors discriminate the regime:
//   - nil:                  verified entry returned
//   - ErrNotActive:         pre-activation; compose from optimistic outputs
//   - ErrBeforeVerifiedDB:  post-activation but below firstVerifiable; not
//     answerable on this node
//   - ethereum.NotFound:    verifier may eventually produce a result but has
//     not yet — return Data = nil and let CurrentL1 communicate progress
type VerifiedResultReader interface {
	VerifiedResultAtTimestamp(ts uint64) (VerifiedResult, error)
}

// NoopVerifiedResultReader is used when interop is not configured: every
// call returns ErrNotActive.
type NoopVerifiedResultReader struct{}

func (NoopVerifiedResultReader) VerifiedResultAtTimestamp(uint64) (VerifiedResult, error) {
	return VerifiedResult{}, ErrNotActive
}
