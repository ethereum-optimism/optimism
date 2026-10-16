package node

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	node_utils "github.com/ethereum-optimism/optimism/rust/kona/tests/node/utils"
)

// Check that an observed unsafe block becomes safe without changing its hash.
func TestSyncUnsafeBecomesSafe(gt *testing.T) {
	t := devtest.ParallelT(gt)
	out := newCommonPreset(t)
	nodes := out.L2CLKonaNodes()
	var checks []dsl.CheckFunc
	for _, node := range nodes {
		checks = append(checks, node.AdvancedFn(safety.LocalUnsafe, 20, 80))
	}
	dsl.CheckAll(t, checks...)

	status, err := node_utils.RollupWS(t, &nodes[0]).SyncStatus(t.Ctx())
	t.Require().NoError(err)
	checkHeadOnPeers(t, nodes, status.UnsafeL2, safety.CrossSafe)
}

// Unsafe heads agree on this single-sequencer network without reorgs.
func TestSyncUnsafe(gt *testing.T) {
	t := devtest.ParallelT(gt)
	out := newCommonPreset(t)
	nodes := out.L2CLKonaNodes()
	var checks []dsl.CheckFunc
	for _, node := range nodes {
		checks = append(checks, node.AdvancedFn(safety.LocalUnsafe, 20, 80))
	}
	dsl.CheckAll(t, checks...)

	status, err := node_utils.RollupWS(t, &nodes[0]).SyncStatus(t.Ctx())
	t.Require().NoError(err)
	checkHeadOnPeers(t, nodes, status.UnsafeL2, safety.LocalUnsafe)
}

// Safe heads agree on this network with a single DA layer.
func TestSyncSafe(gt *testing.T) {
	t := devtest.ParallelT(gt)
	out := newCommonPreset(t)
	nodes := out.L2CLKonaNodes()
	var checks []dsl.CheckFunc
	for _, node := range nodes {
		checks = append(checks, node.AdvancedFn(safety.CrossSafe, 20, 80))
	}
	dsl.CheckAll(t, checks...)

	status, err := node_utils.RollupWS(t, &nodes[0]).SyncStatus(t.Ctx())
	t.Require().NoError(err)
	checkHeadOnPeers(t, nodes, status.SafeL2, safety.CrossSafe)
}

// Every node finalizes a new block, and its contents agree across peers.
func TestSyncFinalized(gt *testing.T) {
	t := devtest.ParallelT(gt)
	out := newCommonPreset(t)
	nodes := out.L2CLKonaNodes()
	var checks []dsl.CheckFunc
	for _, node := range nodes {
		checks = append(checks, node.AdvancedFn(safety.Finalized, 1, 120))
	}
	dsl.CheckAll(t, checks...)

	status, err := node_utils.RollupWS(t, &nodes[0]).SyncStatus(t.Ctx())
	t.Require().NoError(err)
	checkHeadOnPeers(t, nodes, status.FinalizedL2, safety.Finalized)
}

func checkHeadOnPeers(t devtest.T, nodes []dsl.L2CLNode, head eth.L2BlockRef, level safety.Level) {
	t.Helper()
	t.Require().NotEmpty(nodes)
	t.Require().Greater(head.Number, uint64(0), "expected a non-genesis block")
	for _, node := range nodes {
		node.Reached(level, head.Number, 120)
		output, err := node_utils.RollupWS(t, &node).OutputAtBlock(t.Ctx(), head.Number)
		t.Require().NoError(err)
		t.Require().Equal(head, output.BlockRef, "block mismatch on %s", node.Escape().Name())
	}
}
