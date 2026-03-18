package divergence

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/depreqres/common"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum"
)

// TestCLELDivergence tests that the CL and EL diverge when the CL advances the unsafe head, due to accepting SYNCING response from the EL, but the EL cannot validate the block (yet), does not canonicalize it, and doesn't serve it.
func TestCLELDivergence(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithoutP2PWithoutCheck(t, common.ReqRespSyncDisabledOpts(sync.ELSync)...)
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
		targetBlock := sys.L2EL.BlockRefByNumber(targetNumber)

		l.Info("Sending payload ", "target", targetNumber, "startNum", startNum)
		sys.L2CLB.SignalTarget(sys.L2EL, targetNumber)

		// Canonical unsafe head never advances because of the gap
		require.Equal(startNum+1, sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number)

		// EL-sync can quickly reset the status tracker after exposing the posted
		// unsafe head, so poll tightly and without extra RPCs between the post and
		// the SyncStatus check.
		var ss *eth.SyncStatus
		require.Eventually(func() bool {
			ss = sys.L2CLB.SyncStatus()
			return ss.UnsafeL2.Number == targetNumber && ss.UnsafeL2.Hash == targetBlock.Hash
		}, 2*time.Second, 10*time.Millisecond, "L2CLB unsafe head did not expose target block")

		// Confirm that L2ELB cannot fetch the block by hash yet, because the block is not canonicalized, even though the CL reference is set to it.
		_, err := sys.L2ELB.Escape().L2EthClient().L2BlockRefByHash(t.Ctx(), ss.UnsafeL2.Hash)
		require.Error(err, ethereum.NotFound)
	}
}
