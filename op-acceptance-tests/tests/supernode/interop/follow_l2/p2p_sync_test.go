package follow_l2

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum/go-ethereum/common"
)

// TestFollowSource_P2PSync checks that a follower CL syncs unsafe blocks from the
// sequencer via P2P. After stopping and restarting the follower, it reconnects to
// the sequencer and catches up to the same unsafe head.
func TestFollowSource_P2PSync(gt *testing.T) {
	t := devtest.SerialT(gt)
	require := t.Require()

	sys := presets.NewTwoL2SupernodeFollowL2(t, 0)
	logger := sys.Log.With("Test", "TestFollowSource_P2PSync")

	logger.Info("Make sure both sequencers and followers advance")
	dsl.CheckAll(t,
		sys.L2ACL.AdvancedFn(safety.LocalUnsafe, 5, 30),
		sys.L2AFollowCL.AdvancedFn(safety.LocalUnsafe, 5, 30),
		sys.L2BCL.AdvancedFn(safety.LocalUnsafe, 5, 30),
		sys.L2BFollowCL.AdvancedFn(safety.LocalUnsafe, 5, 30),
	)
	beforeA := sys.L2AFollowCL.HeadBlockRef(safety.LocalUnsafe)
	beforeB := sys.L2BFollowCL.HeadBlockRef(safety.LocalUnsafe)

	logger.Info("Stop chain A follower CL")
	sys.L2AFollowCL.Stop()

	logger.Info("Make sure chain A stays stopped while chain B remains live")
	dsl.CheckAll(t,
		sys.L2AFollowEL.NotAdvancedFn(eth.Unsafe, 5),
		sys.L2BCL.AdvancedFn(safety.LocalUnsafe, 5, 30),
		sys.L2BFollowCL.AdvancedFn(safety.LocalUnsafe, 5, 30),
	)
	duringB := sys.L2BFollowCL.HeadBlockRef(safety.LocalUnsafe)
	canonicalB := make(map[uint64]common.Hash, duringB.Number-beforeB.Number)
	for height := beforeB.Number + 1; height <= duringB.Number; height++ {
		canonicalB[height] = sys.L2BFollowEL.BlockRefByNumber(height).Hash
	}

	logger.Info("Restart chain A follower CL")
	sys.L2AFollowCL.Start()

	logger.Info("Reconnect follower P2P to sequencer")
	sys.L2AFollowCL.ConnectPeer(sys.L2ACL)
	sys.L2AFollowCL.Reached(safety.LocalUnsafe, beforeA.Number, 30)
	restoredA := sys.L2AFollowCL.HeadBlockRef(safety.LocalUnsafe)
	require.GreaterOrEqual(restoredA.Number, beforeA.Number,
		"chain A follower did not restore its pre-restart head")
	require.Equal(beforeA.Hash, sys.L2AFollowEL.BlockRefByNumber(beforeA.Number).Hash,
		"chain A follower rewrote its pre-restart head")

	logger.Info("Make sure both advance")
	dsl.CheckAll(t,
		sys.L2ACL.AdvancedFn(safety.LocalUnsafe, 10, 30),
		sys.L2AFollowCL.AdvancedFn(safety.LocalUnsafe, 10, 30),
	)

	logger.Info("Check sequencer and follower converged on the same canonical chain")
	sys.L2AFollowCL.InSync(sys.L2ACL, safety.LocalUnsafe, 30)
	for height, expected := range canonicalB {
		require.Equal(expected, sys.L2BFollowEL.BlockRefByNumber(height).Hash,
			"chain B follower rewrote height %d during chain A restart", height)
		require.Equal(expected, sys.L2ELB.BlockRefByNumber(height).Hash,
			"chain B source disagreed at height %d after chain A restart", height)
	}
}
