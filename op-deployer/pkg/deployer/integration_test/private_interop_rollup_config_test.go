package integration_test

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// Producing the two rollup configs of a private interop pair.
//
// There is no bespoke generator, and there deliberately isn't one: both configs are STOCK, and the
// only thing that makes them a pair is that they share a chain ID. The design's earlier alt-DA
// shape -- a permissioned second inbox with a DA-server adapter, which WOULD have needed a
// non-stock `alt_da` block on the private side -- is retired. The private chain's safe label is
// claim-driven now: a range is safe once its claim has landed in an L1 batch the rendering has
// derived. Nothing about that reaches rollup.json.
//
// So the runbook is two ordinary op-deployer runs over the same L2 chain ID, differing only in the
// chain's privateInterop stanza and its custom gas token:
//
//	# the private half
//	op-deployer apply --intent intent-private.toml   --state state-private.json
//	op-deployer inspect genesis --state state-private.json <chain-id> > genesis-private.json
//	op-deployer inspect rollup  --state state-private.json <chain-id> > rollup-private.json
//
//	# the public rendering, same <chain-id>
//	op-deployer apply --intent intent-rendering.toml --state state-rendering.json
//	op-deployer inspect genesis --state state-rendering.json <chain-id> > genesis-rendering.json
//	op-deployer inspect rollup  --state state-rendering.json <chain-id> > rollup-rendering.json
//
// where intent-private.toml carries
//
//	[[chains]]
//	  [chains.customGasToken]  name = "..." symbol = "..."
//	  [chains.privateInterop]  role = "private"  operator = "0x..."  operatorBalance = "0x..."
//	                           counterpartyChainID = ...  lockVault = "0x..."
//
// and intent-rendering.toml carries the same chain ID with the same `l1StartBlockHash` -- see the
// timestamp note in the block-for-block subtest below for why that pinning is load-bearing --
//
//	[[chains]]
//	  [chains.privateInterop]  role = "rendering"  operator = "0x..."  operatorBalance = "0x..."
//
// This test IS that runbook, executed: it renders both configs through the same code path
// `op-deployer inspect rollup` uses and asserts the properties the pair depends on.
//
// Not covered here: cross-parsing the rendering's config with kona. The Rust tree is in-tree
// (rust/kona) and op-devstack/shared/rustbin already builds kona binaries from Go tests -- see
// op-deployer/pkg/deployer/integration_test/cli/pcd_boot.go, which boots op-reth from a committed
// op-deployer genesis artifact the same way -- but there is no existing Go test that parses a
// config with a Rust binary, and standing one up means compiling kona in CI. Worth doing; it is
// not cheap, and it would be testing stock config parsing rather than anything private interop
// introduces, since neither config has a non-stock field in it.
func TestPrivateInteropRollupConfigs(t *testing.T) {
	op_e2e.InitParallel(t)

	rendering := renderPrivateInteropRollupConfig(t, state.PrivateInteropRendering)
	private := renderPrivateInteropRollupConfig(t, state.PrivateInteropPrivateChain)

	t.Run("both configs are valid", func(t *testing.T) {
		// RenderGenesisAndRollup already calls Check, but assert it here too: this is the property
		// the deliverable is about, and it should fail in this test rather than in a helper.
		require.NoError(t, rendering.Check())
		require.NoError(t, private.Check())
	})

	t.Run("both configs are stock", func(t *testing.T) {
		// The alt-DA design is retired. A stray alt_da block would put a private-interop node into
		// alt-DA derivation, waiting on a DA server that no longer exists.
		require.Nil(t, rendering.AltDAConfig, "the rendering carries no alt_da block")
		require.Nil(t, private.AltDAConfig, "the private chain carries no alt_da block")
	})

	t.Run("one chain ID, two configs", func(t *testing.T) {
		// The rendering IS the private chain's identity in the dependency set, so a counterparty
		// judging the rendering and a relayer reading the private chain are talking about the same
		// chain.
		require.Zero(t, rendering.L2ChainID.Cmp(private.L2ChainID))
	})

	t.Run("two genesis states", func(t *testing.T) {
		// Same ID, different content: the rendering's genesis carries the replay messenger and the
		// claim registry, the private chain's carries the mint bridge and a custom gas token. The
		// two must never be confused for one another, and the genesis hash is what tells them
		// apart. This is also why the two sides must never gossip-peer.
		require.NotEqual(t, rendering.Genesis.L2.Hash, private.Genesis.L2.Hash)
	})

	t.Run("block-for-block correspondence is possible", func(t *testing.T) {
		// The rendering synthesizes one public block per private block at the same number and
		// timestamp. That is only expressible if both chains agree on block time and on where
		// numbering starts.
		require.Equal(t, private.BlockTime, rendering.BlockTime)
		require.Equal(t, private.Genesis.L2.Number, rendering.Genesis.L2.Number)

		// Genesis TIMESTAMPS are deliberately not compared here, and the reason is a real
		// deployment constraint rather than a gap in this test. Each run below seals its own dev
		// L1 and takes its L2 genesis timestamp from that L1 block, so the two differ by however
		// long the first run took. A real pair must pin both runs to the SAME L1 start block --
		// `l1StartBlockHash` on the chain intent -- or the halves start their numbering at
		// different times and the builder can never emit a block at both the private chain's
		// number and its timestamp. Worth an explicit check wherever a pair is actually deployed.
	})

	t.Run("the private chain's batch inbox is vestigial but required", func(t *testing.T) {
		// The private chain never derives from L1. Its LightCL runs in follow mode, pointed at the
		// claim-follower sidecar, and takes its safe head from claims that land on the RENDERING.
		// Nothing on the private side ever reads a batch out of an inbox.
		//
		// rollup.Config.Check still rejects a zero batch inbox address, so the field has to hold
		// something. It holds the SAME address as the rendering: the pair posts exactly one batch
		// stream, to one inbox, and the private chain's config naming that same inbox is the
		// honest statement of where its data went -- as opposed to a second address, which would
		// imply a second stream that does not exist, or a placeholder, which would read as a
		// deployment mistake to anyone comparing the two files.
		require.NotEqual(t, common.Address{}, private.BatchInboxAddress)
		require.Equal(t, rendering.BatchInboxAddress, private.BatchInboxAddress)
	})
}

func renderPrivateInteropRollupConfig(t *testing.T, role state.PrivateInteropRole) *rollup.Config {
	t.Helper()

	gen, applied := generatePrivateInteropPair(t, role)
	_, cfg, err := pipeline.RenderGenesisAndRollup(applied.st, gen.chainIntent.ID, applied.intent)
	require.NoError(t, err)
	return cfg
}
