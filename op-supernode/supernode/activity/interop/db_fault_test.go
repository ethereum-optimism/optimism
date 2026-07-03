package interop

import (
	"errors"
	"hash/fnv"
	"testing"

	"github.com/stretchr/testify/require"
)

// errInjectedDBFault is the sentinel a planted verifiedDB fault returns. The
// drive wrappers recognize it to replay past a one-shot fault; the verifier's
// own errors never wrap it.
var errInjectedDBFault = errors.New("injected verifiedDB fault")

func fnv32(data []byte) uint32 {
	h := fnv.New32a()
	_, _ = h.Write(data)
	return h.Sum32()
}

// faultMethods are the verifiedDB methods on the apply/observe path a fault can
// target. Each is recoverable: the WAL retains the pending tx and the next round
// replays it idempotently.
var faultMethods = []string{
	"Commit",
	"Get",
	"GetPendingTransition",
	"SetPendingTransition",
	"ClearPendingTransition",
	"Rewind",
}

// dbFault is a one-shot verifiedDB fault. It fires at most once, and only while
// active — the drive wrappers activate it around a single production call so the
// Dafny oracle's own DB reads between rounds never trip it.
type dbFault struct {
	target string
	skip   int // let this many matching calls pass before firing
	seen   int // matching calls observed while active (fires on seen == skip+1)
	done   bool
	active bool
}

func (f *dbFault) gate(method string) error {
	if !f.active || method != f.target {
		return nil
	}
	f.seen++
	if f.done || f.seen <= f.skip {
		return nil
	}
	f.done = true
	return errInjectedDBFault
}

// assertFiredIfReachable fails t when the target was hit enough times to fire
// yet the fault never did — a wiring or recovery regression. A no-op when no
// fault was armed, or when the target was unreachable this run (nothing to
// prove).
func (f *dbFault) assertFiredIfReachable(t *testing.T) {
	t.Helper()
	if f == nil || f.seen <= f.skip {
		return
	}
	require.True(t, f.done,
		"armed verifiedDB fault on %s was reachable (%d active hits) but never fired", f.target, f.seen)
}

// armDBFault derives a one-shot fault from the seed and installs it on the live
// DB. ~1/7 of seeds arm nothing (returns nil), so the plain fault-free path is
// still exercised.
//
// ponytail: single main-loop goroutine in the harness, so no lock is needed.
func armDBFault(i *Interop, data []byte) *dbFault {
	h := fnv32(data)
	if h%7 == 0 {
		return nil
	}
	f := &dbFault{
		target: faultMethods[h%uint32(len(faultMethods))],
		skip:   int((h >> 8) % 4),
	}
	i.verifiedDB.faultHook = f.gate
	return f
}

// recordStep runs progressAndRecord with the fault active, then replays past a
// one-shot fire (the WAL replays the pending tx). Real errors — including the
// verifier's own — pass through unchanged. f may be nil (no fault armed).
func recordStep(i *Interop, f *dbFault) (bool, error) {
	if f != nil {
		f.active = true
	}
	p, err := i.progressAndRecord()
	if f != nil {
		f.active = false
	}
	if errors.Is(err, errInjectedDBFault) {
		p, err = i.progressAndRecord()
	}
	return p, err
}

// observeStep runs progressInterop (read-only) with the fault active, retrying
// once past a one-shot fire. f may be nil.
func observeStep(i *Interop, f *dbFault) (StepOutput, RoundObservation, error) {
	if f != nil {
		f.active = true
	}
	out, obs, err := i.progressInterop()
	if f != nil {
		f.active = false
	}
	if errors.Is(err, errInjectedDBFault) {
		out, obs, err = i.progressInterop()
	}
	return out, obs, err
}
