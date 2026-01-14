package node

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	node_utils "github.com/op-rs/kona/node/utils"
)

// Check that all the nodes in the network are synced to the local safe block and can catch up to the sequencer node.
func TestL2SafeSync(gt *testing.T) {
	t := devtest.ParallelT(gt)

	out := node_utils.NewMixedOpKona(t)

	sequencer := out.L2CLSequencerNodes()[0]
	nodes := out.L2CLValidatorNodes()

	checkFuns := make([]dsl.CheckFunc, 0, 2*len(nodes))

	for _, node := range nodes {
		checkFuns = append(checkFuns, node.ReachedFn(types.LocalSafe, 20, 40))
		checkFuns = append(checkFuns, node_utils.MatchedWithinRange(t, node, sequencer, 5, types.LocalSafe, 100))
	}

	dsl.CheckAll(t, checkFuns...)
}

// Check that all the nodes in the network are synced to the local unsafe block and can catch up to the sequencer node.
func TestL2UnsafeSync(gt *testing.T) {
	t := devtest.ParallelT(gt)

	out := node_utils.NewMixedOpKona(t)

	nodes := out.L2CLNodes()

	checkFuns := make([]dsl.CheckFunc, 0, len(nodes))

	for _, node := range nodes {
		checkFuns = append(checkFuns, node.ReachedFn(types.LocalUnsafe, 40, 80))
	}

	dsl.CheckAll(t, checkFuns...)
}

// Check that all the kona nodes in the network are synced to the finalized block.
func TestL2FinalizedSync(gt *testing.T) {
	t := devtest.ParallelT(gt)
	t.Skip("Skipping finalized sync test")

	out := node_utils.NewMixedOpKona(t)

	nodes := out.L2CLNodes()

	checkFuns := make([]dsl.CheckFunc, 0, len(nodes))

	for _, node := range nodes {
		checkFuns = append(checkFuns, node.ReachedFn(types.Finalized, 10, 600))
	}

	dsl.CheckAll(t, checkFuns...)
}
