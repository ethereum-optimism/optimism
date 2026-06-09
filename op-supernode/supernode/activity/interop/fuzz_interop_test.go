package interop

import (
	"context"
	"testing"

	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/stretchr/testify/require"
)

// FuzzInteropRound generates a set of valid chains over one shared L1, wires
// them into a real Interop (logsDBs + verifiedDB in a tempdir, real L1
// consistency checker), and runs one verification round. Since the data is
// valid by construction the round must advance with no invalid heads.
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
		i.verificationStartTimestamp = 1006 // a mid-chain block, past genesis and <= safe head
		i.initialized.Store(true)

		output, _, err := i.progressInterop()
		require.NoError(t, err)
		require.Equal(t, DecisionAdvance, output.Decision)
		require.Empty(t, output.Result.InvalidHeads)
	})
}
