//go:build !ci

package sync

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestL2CLResync checks that unsafe head advances after stopping and restarting sequencing.
// In the supernode architecture, sequencing is controlled via StopSequencer/StartSequencer.
func TestL2CLResync(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 0)
	logger := sys.Log.With("Test", "TestL2CLResync")

	logger.Info("Check unsafe chains are advancing")
	dsl.CheckAll(t,
		sys.L2ELA.AdvancedFn(eth.Unsafe, 5),
		sys.L2ELB.AdvancedFn(eth.Unsafe, 5),
	)

	logger.Info("Stop sequencers")
	sys.L2ACL.StopSequencer()
	sys.L2BCL.StopSequencer()

	logger.Info("Make sure L2ELs does not advance")
	dsl.CheckAll(t,
		sys.L2ELA.NotAdvancedFn(eth.Unsafe, 5),
		sys.L2ELB.NotAdvancedFn(eth.Unsafe, 5),
	)

	logger.Info("Restart sequencers")
	sys.L2ACL.StartSequencer()
	sys.L2BCL.StartSequencer()

	logger.Info("Check unsafe chains are advancing again")
	dsl.CheckAll(t,
		sys.L2ELA.AdvancedFn(eth.Unsafe, 10),
		sys.L2ELB.AdvancedFn(eth.Unsafe, 10),
	)
}
