package interop

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"testing"

	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/stretchr/testify/require"
)

// errInjectedFault is the sentinel every planted fault wraps. The drive wrappers
// recognize it to replay past a one-shot fault; the verifier's own errors never
// wrap it. Engine faults also wrap the production-relevant error (deadline,
// not-found) so the real code path behaves; the sentinel rides alongside.
var errInjectedFault = errors.New("injected fault")

func fnv32(data []byte) uint32 {
	h := fnv.New32a()
	_, _ = h.Write(data)
	return h.Sum32()
}

// faultTargets are the injectable methods. verifiedDB (WAL) targets surface the
// bare sentinel and recover via WAL replay. Engine targets are limited to the
// apply-only l2Provider methods (called only inside RewindToTimestamp), whose
// errors propagate through the rewind path and are recovered by an apply replay.
//
// L2BlockRefByNumber / PayloadByNumber are deliberately NOT here: they are also
// read during verification (LocalSafeBlockAtTimestamp, OutputV0AtBlockNumber),
// where the verifier absorbs ethereum.NotFound as control flow (wait) rather
// than propagating it — so the sentinel never reaches recordStep and the round
// silently stalls. Their not-found handling is covered by the targeted L1 tests.
var faultTargets = []struct {
	method string
	under  error // nil -> bare sentinel; else wrapped underneath it
}{
	{"Commit", nil},
	{"Get", nil},
	{"GetPendingTransition", nil},
	{"SetPendingTransition", nil},
	{"ClearPendingTransition", nil},
	{"Rewind", nil},
	{"ForkchoiceUpdate", context.DeadlineExceeded}, // RewindEngine root cause 1
	{"NewPayload", nil},                            // synthetic insert rejected
}

// fault is a one-shot fault on a single verifiedDB or engine method. It fires at
// most once, and only while active — the drive wrappers activate it around a
// single production call so the Dafny oracle's own reads between rounds never
// trip it.
type fault struct {
	target string
	under  error
	skip   int // let this many matching calls pass before firing
	seen   int // matching calls observed while active (fires on seen == skip+1)
	done   bool
	active bool
}

func (f *fault) gate(method string) error {
	if !f.active || method != f.target {
		return nil
	}
	f.seen++
	if f.done || f.seen <= f.skip {
		return nil
	}
	f.done = true
	if f.under != nil {
		return fmt.Errorf("%w: %w", errInjectedFault, f.under)
	}
	return errInjectedFault
}

// assertFiredIfReachable fails t when the target was hit enough times to fire
// yet the fault never did — a wiring or recovery regression. A no-op when no
// fault was armed, or when the target was unreachable this run.
func (f *fault) assertFiredIfReachable(t *testing.T) {
	t.Helper()
	if f == nil || f.seen <= f.skip {
		return
	}
	require.True(t, f.done,
		"armed fault on %s was reachable (%d active hits) but never fired", f.target, f.seen)
}

// armFault derives a one-shot fault from the seed and installs it on the live DB
// and every chain's engine. ~1/7 of seeds arm nothing (returns nil), so the
// plain fault-free path is still exercised. A verifiedDB target never matches an
// engine method name and vice-versa, so installing on both hooks is harmless.
//
// ponytail: single main-loop goroutine in the harness, so no lock is needed.
func armFault(i *Interop, mgr *cc.RandomChainManager, data []byte) *fault {
	h := fnv32(data)
	if h%7 == 0 {
		return nil
	}
	t := faultTargets[h%uint32(len(faultTargets))]
	f := &fault{target: t.method, under: t.under, skip: int((h >> 8) % 4)}
	i.verifiedDB.faultHook = f.gate
	for _, rc := range mgr.Chains() {
		rc.SetEngineFaultHook(f.gate)
	}
	return f
}

// firedThisCall reports whether the one-shot fault transitioned to fired during
// the just-completed call (done went false->true). Used to replay even when the
// interop layer wrapped the fault in a summary error that dropped the sentinel
// (e.g. the Invalidate aggregation at interop.go:743) — the WAL still preserved
// the pending tx, so a replay recovers now that the fault is exhausted.
func firedThisCall(f *fault, doneBefore bool) bool {
	return f != nil && f.done && !doneBefore
}

// recordStep runs progressAndRecord with the fault active, then replays past a
// one-shot fire (the WAL / pending tx replays on the retry). Real errors —
// including the verifier's own — pass through unchanged. f may be nil.
func recordStep(i *Interop, f *fault) (bool, error) {
	doneBefore := f != nil && f.done
	if f != nil {
		f.active = true
	}
	p, err := i.progressAndRecord()
	if f != nil {
		f.active = false
	}
	if err != nil && (errors.Is(err, errInjectedFault) || firedThisCall(f, doneBefore)) {
		p, err = i.progressAndRecord()
	}
	return p, err
}

// observeStep runs progressInterop (read-only) with the fault active, retrying
// once past a one-shot fire. f may be nil.
func observeStep(i *Interop, f *fault) (StepOutput, RoundObservation, error) {
	doneBefore := f != nil && f.done
	if f != nil {
		f.active = true
	}
	out, obs, err := i.progressInterop()
	if f != nil {
		f.active = false
	}
	if err != nil && (errors.Is(err, errInjectedFault) || firedThisCall(f, doneBefore)) {
		out, obs, err = i.progressInterop()
	}
	return out, obs, err
}
