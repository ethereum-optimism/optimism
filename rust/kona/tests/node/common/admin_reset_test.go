package node

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	node_utils "github.com/ethereum-optimism/optimism/rust/kona/tests/node/utils"
	"github.com/stretchr/testify/require"
)

func TestAdminResetPreservesSafeProgress(gt *testing.T) {
	t := devtest.ParallelT(gt)
	out := newCommonPreset(t)
	nodes := out.L2CLKonaNodes()
	require.NotEmpty(t, nodes)
	checks := make([]dsl.CheckFunc, 0, len(nodes))
	for _, node := range nodes {
		checks = append(checks, node.AdvancedFn(safety.LocalSafe, 2, 30))
	}
	dsl.CheckAll(t, checks...)

	checks = checks[:0]
	for _, node := range nodes {
		require.NoError(t, node_utils.SendRPCRequest[any](node_utils.GetNodeRPCEndpoint(&node), "admin_resetDerivationPipeline", nil), "reset %s", node.Escape().Name())
		checks = append(checks, node.AdvancedFn(safety.LocalSafe, 2, 40))
	}
	dsl.CheckAll(t, checks...)

	peers := out.L2CLNodes()
	height := peers[0].ChainSyncStatus(out.L2Chain.ChainID(), safety.LocalSafe).Number
	for _, node := range peers[1:] {
		height = min(height, node.ChainSyncStatus(out.L2Chain.ChainID(), safety.LocalSafe).Number)
	}
	reference, err := peers[0].Escape().RollupAPI().OutputAtBlock(t.Ctx(), height)
	require.NoError(t, err)
	for _, node := range peers[1:] {
		output, err := node.Escape().RollupAPI().OutputAtBlock(t.Ctx(), height)
		require.NoError(t, err)
		require.Equal(t, reference.BlockRef, output.BlockRef)
	}
}
