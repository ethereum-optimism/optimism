// Package sdmtest holds the reusable building blocks for SDM (block-production-with-refunds)
// acceptance tests: the op-reth sequencer/verifier system fixture, the repeated-slot warming
// workload, and the consensus-replay/refund RPC helpers.
//
// It is a non-test (importable) package precisely so the same fixtures can drive SDM acceptance
// tests that live outside this repository — e.g. a suite that boots a sequencer binary built from a
// separate repo (selected via OpRethWithBinary). Those external consumers depend on this monorepo by
// git rev; the monorepo never depends on them.
package sdmtest

import (
	"strings"

	sdmpkg "github.com/ethereum-optimism/optimism/op-chain-ops/pkg/sdm"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// RethSystem bundles the DSL frontends an SDM acceptance test drives: an op-reth sequencer, a
// stock op-reth verifier, the batcher, and an L2 funder, all on a single Interop/SDM chain.
type RethSystem struct {
	L1EL         *dsl.L1ELNode
	L2EL         *dsl.L2ELNode
	L2CL         *dsl.L2CLNode
	L2Network    *dsl.L2Network
	L2ELVerifier *dsl.L2ELNode
	L2CLVerifier *dsl.L2CLNode
	L2Batcher    *dsl.L2Batcher
	FunderL2     *dsl.Funder
}

// FinishRethSystem wraps a built MixedSingleChainRuntime in DSL frontends, derives the verifier
// refs + an L2 funder, and (when SDM is active) opts the sequencer in via admin_setOperatorSdmOptIn.
// Shared by the stock-op-reth builder and any external premium-sequencer builder so both produce an
// identical RethSystem regardless of which EL binary the sequencer runs.
func FinishRethSystem(t devtest.T, runtime *sysgo.MixedSingleChainRuntime, interopAtGenesis bool) *RethSystem {
	frontends := presets.NewMixedSingleChainFrontends(t, runtime)
	frontends.L2Batcher.Stop()
	t.Require().Len(frontends.Nodes, 2, "SDM op-reth system must include sequencer and verifier nodes")

	var verifierEL *dsl.L2ELNode
	var verifierCL *dsl.L2CLNode
	for _, node := range frontends.Nodes {
		if !node.Spec.IsSequencer {
			verifierEL = node.EL
			verifierCL = node.CL
			break
		}
	}
	t.Require().NotNil(verifierEL, "missing SDM verifier EL node")
	t.Require().NotNil(verifierCL, "missing SDM verifier CL node")

	wallet := dsl.NewRandomHDWallet(t, 30)
	sys := &RethSystem{
		L1EL:         frontends.L1EL,
		L2EL:         frontends.L2Network.PrimaryEL(),
		L2CL:         frontends.L2Network.PrimaryCL(),
		L2Network:    frontends.L2Network,
		L2ELVerifier: verifierEL,
		L2CLVerifier: verifierCL,
		L2Batcher:    frontends.L2Batcher,
		FunderL2:     dsl.NewFunder(wallet, frontends.FaucetL2, frontends.L2Network.PrimaryEL()),
	}

	// The protocol gate (Interop hardfork) is already scheduled by the caller when
	// interopAtGenesis is true. Local PostExec production additionally requires the
	// sequencer's op-reth to be opted in via admin_setOperatorSdmOptIn; nothing else
	// flips this on. Verifier nodes do not need to opt in — they accept PostExec
	// txs by chain spec rule alone.
	if interopAtGenesis {
		SetSDMEnabled(t, sys.L2EL, true)
	}
	return sys
}

// SetSDMEnabled toggles the local SDM PostExec production opt-in on an L2 EL via the
// admin_setOperatorSdmOptIn RPC. SDM is disabled by default on every process boot; tests that
// expect PostExec txs to flow must opt in explicitly on the sequencer's EL.
func SetSDMEnabled(t devtest.T, l2EL *dsl.L2ELNode, enabled bool) {
	rpcClient := l2EL.Escape().L2EthClient().RPC()
	err := rpcClient.CallContext(t.Ctx(), nil, "admin_setOperatorSdmOptIn", enabled)
	t.Require().NoError(err, "admin_setOperatorSdmOptIn(%v) RPC failed", enabled)
}

// VerifyOpReth checks the L2 execution layer client is op-reth by calling
// web3_clientVersion via the L2EthClient's RPC and asserting it contains "reth".
func VerifyOpReth(t devtest.T, l2EL *dsl.L2ELNode) string {
	rpcClient := l2EL.Escape().L2EthClient().RPC()
	var clientVersion string
	err := rpcClient.CallContext(t.Ctx(), &clientVersion, "web3_clientVersion")
	t.Require().NoError(err, "web3_clientVersion RPC failed — cannot verify EL client")

	lower := strings.ToLower(clientVersion)
	t.Require().True(
		strings.Contains(lower, "reth"),
		"FATAL: Expected op-reth execution client, but got: %q. "+
			"This test MUST run on op-reth. "+
			"Set DEVSTACK_L2EL_KIND=op-reth or ensure op-reth binary is available.",
		clientVersion,
	)
	t.Require().False(
		strings.Contains(lower, "geth"),
		"FATAL: Detected op-geth (%q) but this test requires op-reth.", clientVersion,
	)

	return clientVersion
}

// GetBlockWithTxs fetches a block with full transactions via eth_getBlockByNumber.
func GetBlockWithTxs(t devtest.T, l2EL *dsl.L2ELNode, blockNum uint64) *sdmpkg.RPCBlock {
	block, err := sdmpkg.GetBlockWithTxs(t.Ctx(), l2EL.Escape().L2EthClient().RPC(), blockNum)
	t.Require().NoError(err, "eth_getBlockByNumber RPC failed for block %d", blockNum)
	return block
}

// ReplayBlockWithSDM re-executes a block through debug_replaySDMBlock, returning the consensus
// replay (synthesized PostExec payload, per-tx refund breakdown, and summary totals).
func ReplayBlockWithSDM(t devtest.T, l2EL *dsl.L2ELNode, blockNum uint64) *sdmpkg.ReplaySDMBlock {
	replay, err := sdmpkg.ReplayBlockWithSDM(t.Ctx(), l2EL.Escape().L2EthClient().RPC(), blockNum, true)
	t.Require().NoError(err, "debug_replaySDMBlock RPC failed for block %d", blockNum)
	return replay
}
