package divergence

import (
	"testing"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum"
)

func TestMain(m *testing.M) {
	// No ELP2P, CLP2P to control the supply of unsafe payload to the CL
	presets.DoMain(m, presets.WithSingleChainMultiNodeWithoutP2P(),
		presets.WithCompatibleTypes(compat.SysGo),
		presets.WithExecutionLayerSyncOnVerifiers(),
		presets.WithReqRespSyncDisabled(),
		presets.WithNoDiscovery(),
		stack.MakeCommon(sysgo.WithBatcherOption(func(id stack.L2BatcherID, cfg *bss.CLIConfig) {
			cfg.Stopped = true
		})),
	)
}

// TestCLELDivergence tests that the CL and EL diverge when the CL advances the unsafe head, due to accepting SYNCING response from the EL, but the EL cannot validate the block (yet), does not canonicalize it, and doesn't serve it.
func TestCLELDivergence(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithoutCheck(t)
	require := t.Require()
	l := t.Logger()

	sys.L2CL.Advanced(types.LocalUnsafe, 8, 30)

	// batcher down so safe not advanced
	require.Equal(uint64(0), sys.L2CL.HeadBlockRef(types.LocalSafe).Number)
	require.Equal(uint64(0), sys.L2CLB.HeadBlockRef(types.LocalSafe).Number)

	startNum := sys.L2CLB.HeadBlockRef(types.LocalUnsafe).Number

	// Finish EL sync by supplying the first block
	// EL Sync finished because underlying EL has states to validate the payload for block startNum+1
	sys.L2CLB.SignalTarget(sys.L2EL, startNum+1)
	require.Equal(startNum+1, sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number)

	for _, delta := range []uint64{3, 4, 5} {
		targetNumber := startNum + delta
		l.Info("Sending payload ", "target", targetNumber, "startNum", startNum)
		sys.L2CLB.SignalTarget(sys.L2EL, targetNumber)

		// Canonical unsafe head never advances because of the gap
		require.Equal(startNum+1, sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number)

		// Unsafe head on CL advanced, but on EL we cannot fetch state for the unsafe block hash yet
		targetBlock := sys.L2EL.BlockRefByNumber(targetNumber)

		// Confirm that L2CLB SyncStatus returns the newest unsafe block number and hash
		ss := sys.L2CLB.SyncStatus()
		require.Equal(targetNumber, ss.UnsafeL2.Number)
		require.Equal(targetBlock.Hash, ss.UnsafeL2.Hash)

		// Confirm that L2ELB cannot fetch the block by hash yet, because the block is not canonicalized, even though the CL reference is set to it.
		_, err := sys.L2ELB.Escape().L2EthClient().L2BlockRefByHash(t.Ctx(), ss.UnsafeL2.Hash)
		require.Error(err, ethereum.NotFound)
	}
}
