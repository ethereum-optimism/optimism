package follow_l2

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// TestFollowSource_P2PSync checks that a follower CL syncs unsafe blocks from the
// sequencer via P2P. After stopping and restarting the follower, it reconnects to
// the sequencer and catches up to the same unsafe head.
func TestFollowSource_P2PSync(gt *testing.T) {
	t := devtest.ParallelT(gt)

	sys := presets.NewTwoL2SupernodeFollowL2(t, 0)
	logger := sys.Log.With("Test", "TestFollowSource_P2PSync")

	logger.Info("Make sure sequencer and follower unsafe head advances")
	dsl.CheckAll(t,
		sys.L2ACL.AdvancedFn(types.LocalUnsafe, 5, 30),
		sys.L2AFollowCL.AdvancedFn(types.LocalUnsafe, 5, 30),
	)

	logger.Info("Stop follower CL")
	sys.L2AFollowCL.Stop()

	logger.Info("Make sure follower EL does not advance")
	sys.L2AFollowEL.NotAdvanced(eth.Unsafe, 5)

	logger.Info("Restart follower CL")
	sys.L2AFollowCL.Start()

	logger.Info("Reconnect follower P2P to sequencer")
	sys.L2AFollowCL.ConnectPeer(sys.L2ACL)

	logger.Info("Make sure both advance")
	dsl.CheckAll(t,
		sys.L2ACL.AdvancedFn(types.LocalUnsafe, 10, 30),
		sys.L2AFollowCL.AdvancedFn(types.LocalUnsafe, 10, 30),
	)

	logger.Info("Check sequencer and follower holds identical chain")
	sys.L2AFollowCL.Matched(sys.L2ACL, types.LocalUnsafe, 30)
}
