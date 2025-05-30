package sync

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestL2CLResync checks that unsafe head advances after restarting L2CL.
// Resync is only possible when supervisor and L2CL reconnects.
func TestL2CLResync(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)
	logger := sys.Log.With("Test", "TestL2CLResync")

	logger.Info("Check unsafe chains are advancing")
	dsl.CheckAll(t,
		sys.L2ELA.AdvancedFn(eth.Unsafe, 5),
		sys.L2ELB.AdvancedFn(eth.Unsafe, 5),
	)

	logger.Info("Stop L2CL nodes")
	sys.L2CLA.Stop()
	sys.L2CLB.Stop()

	logger.Info("Make sure L2ELs does not advance")
	dsl.CheckAll(t,
		sys.L2ELA.NotAdvancedFn(eth.Unsafe),
		sys.L2ELB.NotAdvancedFn(eth.Unsafe),
	)

	logger.Info("Restart L2CL nodes")
	sys.L2CLA.Start()
	sys.L2CLB.Start()

	// L2CL may advance a few blocks without supervisor connection, but eventually it will stop without the connection
	// we must check that unsafe head is advancing due to reconnection
	logger.Info("Boot up L2CL nodes")
	dsl.CheckAll(t,
		sys.L2ELA.AdvancedFn(eth.Unsafe, 10),
		sys.L2ELB.AdvancedFn(eth.Unsafe, 10),
	)

	// supervisor will attempt to reconnect with L2CLs at this point because L2CL ws endpoint is recovered
	logger.Info("Check unsafe chains are advancing again")
	dsl.CheckAll(t,
		sys.L2ELA.AdvancedFn(eth.Unsafe, 10),
		sys.L2ELB.AdvancedFn(eth.Unsafe, 10),
	)

	// supervisor successfully connected with managed L2CLs
}
