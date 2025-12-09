package sync

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// TestL2CLResync checks that unsafe head advances after restarting L2CL.
// Resync is only possible when supervisor and L2CL reconnects.
// Acceptance Test: https://github.com/ethereum-optimism/optimism/blob/develop/op-acceptance-tests/tests/interop/sync/simple_interop/interop_sync_test.go
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
		sys.L2ELA.AdvancedFn(eth.Unsafe, 30),
		sys.L2ELB.AdvancedFn(eth.Unsafe, 30),
	)

	// supervisor will attempt to reconnect with L2CLs at this point because L2CL ws endpoint is recovered
	logger.Info("Check unsafe chains are advancing again")
	dsl.CheckAll(t,
		sys.L2ELA.AdvancedFn(eth.Unsafe, 10),
		sys.L2ELB.AdvancedFn(eth.Unsafe, 10),
	)

	// supervisor successfully connected with managed L2CLs
}

// TestSupervisorResync checks that heads advances after restarting the Supervisor.
func TestSupervisorResync(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)
	logger := sys.Log.With("Test", "TestSupervisorResync")

	logger.Info("Check unsafe chains are advancing")

	for _, level := range []types.SafetyLevel{types.LocalUnsafe, types.LocalSafe, types.CrossUnsafe, types.CrossSafe} {
		sys.Supervisor.WaitForL2HeadToAdvance(sys.L2ChainA.ChainID(), 2, level, 20)
		sys.Supervisor.WaitForL2HeadToAdvance(sys.L2ChainB.ChainID(), 2, level, 20)
	}

	logger.Info("Stop Supervisor node")
	sys.Supervisor.Stop()

	logger.Info("Restart Supervisor node")
	sys.Supervisor.Start()

	logger.Info("Boot up Supervisor node")

	// Re check syncing is not blocked
	for _, level := range []types.SafetyLevel{types.LocalUnsafe, types.LocalSafe, types.CrossUnsafe, types.CrossSafe} {
		sys.Supervisor.WaitForL2HeadToAdvance(sys.L2ChainA.ChainID(), 2, level, 20)
		sys.Supervisor.WaitForL2HeadToAdvance(sys.L2ChainB.ChainID(), 2, level, 20)
	}
}
