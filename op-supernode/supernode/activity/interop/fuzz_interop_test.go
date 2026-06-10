package interop

import (
	"context"
	"testing"

	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/stretchr/testify/require"
)

// FuzzInteropRound generates a set of valid chains over one shared L1, wires
// them into a real Interop (logsDBs + verifiedDB in a tempdir, real L1
// consistency checker), and loops verification rounds until the safe head.
// Each advancing round seals that timestamp's blocks into the logsDBs, so the
// cross-chain executing messages of later rounds resolve against sealed init
// logs. Valid-by-construction data must verify every round cleanly.
func FuzzInteropRound(f *testing.F) {
	f.Add([]byte("seed-interop"))
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		ctx := context.Background()
		mgr := cc.NewRandomChainManager(data)
		mgr.Generate()

		for _, rc := range mgr.Chains() {
			require.NoError(t, rc.Start(ctx))
		}

		chains, err := mgr.ChainContainers()
		require.NoError(t, err)

		const activationTS = 1000 // matches the generated genesis L2 time

		i := New(testLogger(), activationTS, 0, chains, t.TempDir(), mgr.L1Source(), 0, nil)
		require.NotNil(t, i)
		t.Cleanup(func() { _ = i.Stop(ctx) })

		i.ctx = ctx
		i.verificationStartTimestamp = 1004 // block 2: first SafeDB-covered block; earlier blocks are below SafeDB history
		i.initialized.Store(true)

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
