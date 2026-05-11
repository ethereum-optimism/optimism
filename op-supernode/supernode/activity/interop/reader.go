package interop

import "errors"

// ErrNotActive signals that the verifier has not produced — and will not
// produce — a VerifiedResult for the requested timestamp. Two cases:
//   - ts < activationTimestamp: interop is genuinely inactive for ts.
//   - ts < firstVerifiableTimestamp: interop is active but the verifier's
//     bootstrap floor has not reached ts and never will.
//
// Callers interpret this as "fall back to pre-interop composition" — distinct
// from ethereum.NotFound which means "interop IS active for ts and the
// verifier may eventually produce a result, but has not yet."
var ErrNotActive = errors.New("interop not active for timestamp")

// VerifiedResultReader exposes a read-only view of committed VerifiedResults
// so non-interop activities can decide whether a timestamp falls into the
// strict (verified) regime, the active-but-not-yet-verified regime
// (returns ethereum.NotFound), or the pre-interop regime (returns
// ErrNotActive).
type VerifiedResultReader interface {
	VerifiedResultAtTimestamp(ts uint64) (VerifiedResult, error)
}

// NoopVerifiedResultReader is used when the supernode is wired without an
// interop activity. Every call reports ErrNotActive, routing consumers into
// the pre-interop fallback.
type NoopVerifiedResultReader struct{}

func (NoopVerifiedResultReader) VerifiedResultAtTimestamp(uint64) (VerifiedResult, error) {
	return VerifiedResult{}, ErrNotActive
}
