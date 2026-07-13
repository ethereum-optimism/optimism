package config

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
)

// TestRustEngineGenesisSmoke forces the op-e2e genesis build (config.L2Allocs /
// L1Allocs) to run end-to-end for the default alloc type. With the Rust engine
// as the default, this exercises the op-e2e caller context specifically:
//   - artifacts.NewFileLocator (os.DirFS) → rustengine.ArtifactsDir reflection,
//   - rustengine.EngineBinary provisioning inside the op-e2e process, and
//   - initAllocType's concurrent (9-mode) genesis builds, each spawning L1+L2
//     engine subprocesses — a concurrency the sequential integration_test table
//     tests never covered.
//
// Spike-only smoke test: it asserts the build succeeds and produces non-empty
// allocs. Byte-for-byte Go-vs-Rust parity of these allocs is already covered by
// the op-deployer integration_test A/B gates.
func TestRustEngineGenesisSmoke(t *testing.T) {
	at := DefaultAllocType

	l1 := L1Allocs(at)
	require.NotNil(t, l1)
	require.NotEmpty(t, l1.Accounts, "L1 genesis allocs must be non-empty")
	t.Logf("L1 genesis accounts: %d", len(l1.Accounts))

	modes := []genesis.L2AllocsMode{
		genesis.L2AllocsLagoon,
		genesis.L2AllocsKarst,
		genesis.L2AllocsJovian,
		genesis.L2AllocsIsthmus,
		genesis.L2AllocsHolocene,
		genesis.L2AllocsGranite,
		genesis.L2AllocsFjord,
		genesis.L2AllocsEcotone,
		genesis.L2AllocsDelta,
	}
	for _, mode := range modes {
		mode := mode
		t.Run(string(mode), func(t *testing.T) {
			l2 := L2Allocs(at, mode)
			require.NotNil(t, l2)
			require.NotEmpty(t, l2.Accounts, "L2 genesis allocs must be non-empty for mode %s", mode)
			t.Logf("L2 genesis accounts (%s): %d", mode, len(l2.Accounts))
			require.Greater(t, len(l2.Accounts), 2000, "L2 genesis should have the full preinstall set")
		})
	}
}
