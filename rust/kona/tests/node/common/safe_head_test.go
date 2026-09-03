package node

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	node_utils "github.com/ethereum-optimism/optimism/rust/kona/tests/node/utils"
	"github.com/stretchr/testify/require"
)

// TestSafeHeadHistoryMatchesOpNodeSafeDB checks that kona-node's `optimism_safeHeadAtL1Block`,
// served by its chain view, agrees with op-node's SafeDB: walking backward from kona's current
// L1 block, every entry kona serves must equal op-node's.
//
// The walk ends where kona's history ends. The chain view records the blocks derived while the
// node runs and nothing before: unlike op-node's SafeDB it does not persist across restarts and
// has no entry for the head the node started from, so on a fresh devnet its history begins at
// the first span batch derived after startup while op-node's reaches back to genesis. The run
// is long enough for several L1 blocks' worth of entries to be compared.
func TestSafeHeadHistoryMatchesOpNodeSafeDB(gt *testing.T) {
	t := devtest.SerialT(gt)

	out := node_utils.NewMixedOpKonaWithOpNodeSafeDB(t)
	konaNodes := out.L2CLKonaNodes()
	require.NotEmpty(t, konaNodes, "need a kona-node to check")
	sourceOfTruth := &out.L2CLOpValidatorNodes[0]

	// Let every node derive a run of safe blocks spanning several L1 blocks first.
	const safeBlocks = 24
	checkFuns := make([]dsl.CheckFunc, 0, len(konaNodes)+1)
	checkFuns = append(checkFuns, sourceOfTruth.ReachedFn(safety.LocalSafe, safeBlocks, 600))
	for i := range konaNodes {
		checkFuns = append(checkFuns, konaNodes[i].ReachedFn(safety.LocalSafe, safeBlocks, 600))
	}
	dsl.CheckAll(t, checkFuns...)

	for i := range konaNodes {
		konaNodes[i].VerifySafeHeadDatabaseMatches(sourceOfTruth)
	}
}
