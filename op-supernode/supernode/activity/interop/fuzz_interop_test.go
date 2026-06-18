package interop

import (
	"context"
	"testing"

	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/stretchr/testify/require"
)

// buildInterop generates a chain set from data and wires it into a real Interop
// (logsDBs + verifiedDB in a tempdir, real L1 consistency checker), started and
// positioned at the first verifiable slot.
func buildInterop(t *testing.T, data []byte) (*Interop, *cc.RandomChainManager) {
	ctx := context.Background()
	mgr := cc.NewRandomChainManager(data)
	mgr.Generate()

	for _, rc := range mgr.Chains() {
		require.NoError(t, rc.Start(ctx))
	}

	dir := t.TempDir()
	chains, err := mgr.ChainContainers(dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = mgr.Close() }) // releases each chain's denyList bbolt handle

	i := New(testLogger(), cc.GenInteropActivation, cc.GenExpiryWindow, chains, dir, mgr.L1Source(), 0, nil)
	require.NotNil(t, i)
	t.Cleanup(func() { _ = i.Stop(ctx) })

	i.ctx = ctx
	i.verificationStartTimestamp = mgr.FirstVerifiableTimestamp()
	i.initialized.Store(true)
	return i, mgr
}

// FuzzInteropRound loops verification rounds over a valid chain set to the safe
// head. Each round seals its blocks, so later rounds' executing messages
// resolve against sealed init logs. Valid data must verify cleanly.
func FuzzInteropRound(f *testing.F) {
	f.Add([]byte("seed-interop"))
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		i, mgr := buildInterop(t, data)

		for n := 0; ; n++ {
			require.Less(t, n, 100, "verification loop did not terminate")
			progress, err := i.progressAndRecord()
			require.NoError(t, err)
			if !progress {
				break
			}
		}

		last, ok := i.verifiedDB.LastTimestamp()
		require.True(t, ok, "no timestamp verified")
		// The verifier steps timestamps by 1s while blocks span 2s, so the safe
		// block's slot verifies at both its own timestamp and the next.
		require.Equal(t, mgr.MinSafeTimestamp()+1, last, "must verify up to the safe head's slot")
	})
}

// FuzzInteropInvalid plants one violation (exec-msg corruption, L1 divergence,
// or safeDB front gap), drives rounds up to the bad timestamp, and asserts the
// verifier rejects correctly.
func FuzzInteropInvalid(f *testing.F) {
	f.Add([]byte("seed-invalid"))
	f.Add([]byte("seed-expiry"))
	f.Add([]byte("seed-exp2"))
	f.Add([]byte("seed-l1div"))
	f.Add([]byte("seed-l1div2"))
	f.Add([]byte("seed-safedb-gap"))
	f.Add([]byte("seed-gap2"))

	f.Fuzz(func(t *testing.T, data []byte) {
		i, mgr := buildInterop(t, data)

		plan, ok := mgr.BreakOne()
		if !ok {
			t.Skip("no violation site reachable")
		}

		next := func() uint64 {
			if last, ok := i.verifiedDB.LastTimestamp(); ok {
				return last + 1
			}
			// Use the frozen start; BreakOneSafeDBFrontGap shifts the live
			// FirstVerifiableTimestamp but AssertTS was captured before that.
			return i.verificationStartTimestamp
		}

		for n := 0; ; n++ {
			require.Less(t, n, 1000, "did not reach the bad block")
			if next() == plan.AssertTS {
				out, _, err := i.progressInterop()
				switch plan.Reject {
				case cc.RejectInvalidHead:
					require.NoError(t, err)
					require.Equal(t, DecisionInvalidate, out.Decision)
					require.Contains(t, out.Result.InvalidHeads, plan.Chain)
				case cc.RejectWait:
					require.NoError(t, err)
					require.Equal(t, DecisionWait, out.Decision)
				case cc.RejectHistoryUnavailable:
					require.ErrorIs(t, err, cc.ErrHistoryUnavailable)
				}
				return
			}
			progress, err := i.progressAndRecord()
			require.NoError(t, err)
			require.True(t, progress, "valid rounds before the bad block must advance")
		}
	})
}
