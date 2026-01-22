package upgrade

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum/go-ethereum/common"
)

// TestInteropContractsDeployed verifies that interop contracts are deployed at genesis
// without any supervisor running.
// Note: CrossL2Inbox is only deployed when there are 2+ chains in the dependency set.
// This test uses a single-chain setup, so only L2ToL2CrossDomainMessenger is deployed.
func TestInteropContractsDeployed(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimalInteropNoSupervisor(t)
	require := t.Require()
	logger := t.Logger()

	// Wait for interop activation - for interop at genesis this is block 0,
	// but the upgrade transactions run in the first block
	activationBlock := sys.L2Chain.AwaitActivation(t, forks.Interop)
	logger.Info("interop activated", "block", activationBlock.Number, "hash", activationBlock.Hash)

	client := sys.L2EL.Escape().L2EthClient()

	// Verify L2ToL2CrossDomainMessenger is deployed
	// (CrossL2Inbox is only deployed when there are 2+ chains in dependency set)
	implAddrBytes, err := client.GetStorageAt(t.Ctx(), predeploys.L2toL2CrossDomainMessengerAddr,
		genesis.ImplementationSlot, activationBlock.Hash.String())
	require.NoError(err)
	implAddr := common.BytesToAddress(implAddrBytes[:])
	require.NotEqual(common.Address{}, implAddr, "L2ToL2CrossDomainMessenger should have implementation at %s",
		predeploys.L2toL2CrossDomainMessengerAddr)

	// Verify the implementation has code
	code, err := client.CodeAtHash(t.Ctx(), implAddr, activationBlock.Hash)
	require.NoError(err)
	require.NotEmpty(code, "L2ToL2CrossDomainMessenger implementation should have code")

	logger.Info("interop contracts deployed successfully without supervisor")
}

// TestLocalFinalityWithoutSupervisor verifies that local finality and promotion work
// correctly when interop is active but no supervisor is running.
func TestLocalFinalityWithoutSupervisor(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimalInteropNoSupervisor(t)
	require := t.Require()
	logger := t.Logger()

	targetBlocks := uint64(5)

	for i := 0; i < 30; i++ {
		time.Sleep(time.Second * 2)

		status := sys.L2CL.SyncStatus()
		require.NotNil(status)

		logger.Info("chain status",
			"unsafe", status.UnsafeL2.Number,
			"safe", status.SafeL2.Number,
			"finalized", status.FinalizedL2.Number,
		)

		// Without supervisor, local promotion should work:
		// - UnsafeL2 should advance (sequencer producing blocks)
		// - SafeL2 should advance (after batches submitted to L1)

		if status.UnsafeL2.Number >= targetBlocks &&
			status.SafeL2.Number >= targetBlocks-2 {
			logger.Info("local finality working without supervisor!")
			return
		}
	}

	gt.Errorf("Expected unsafe >= %d and safe >= %d", targetBlocks, targetBlocks-2)
	gt.FailNow()
}
