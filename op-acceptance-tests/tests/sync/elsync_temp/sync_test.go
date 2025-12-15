package elsync_temp

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestInitialELSyncBump(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithTestSeq(t)
	require := t.Require()
	logger := t.Logger()

	target := uint64(3)
	dsl.CheckAll(t,
		sys.L2EL.ReachedFn(eth.Unsafe, target, 10),
		// Verifier finishes EL Sync, from genesis(block 0) to first block.
		// Example logs:
		//   t=2025-12-16T01:19:16.362+0900 lvl=info msg="Finished EL sync" sync_duration=2.780375ms finalized_block=0x45706623fec0f72515dbd3ffd92b126ee97105b9defb3383a4b99c3dd2fdc4bd:1 scope=/pkg chainID=901 kind=L2CLNode id=L2CLNode-b-901
		// Causing {unsafe, safe, finalized} bump to 1
		sys.L2ELB.ReachedFn(eth.Unsafe, target, 10),
	)
	// Safe and Finalized are bumped, even though batcher is stopped
	require.Equal(uint64(1), sys.L2ELB.SafeHead().BlockRef.Number)
	require.Equal(uint64(1), sys.L2ELB.FinalizedHead().BlockRef.Number)

	syncStatus := sys.L2CLB.SyncStatus()
	logger.Info("Verifier syncStatus", "safe", syncStatus.SafeL2, "finalized", syncStatus.FinalizedL2)
}
