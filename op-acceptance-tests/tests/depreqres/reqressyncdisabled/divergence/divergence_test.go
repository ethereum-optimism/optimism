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

// TestCLELDivergence verifies that the CL and EL can temporarily diverge.
// This happens when the CL advances its unsafe head after receiving a SYNCING
// response from the EL, while the EL itself cannot yet validate or canonicalize
// the corresponding block, and therefore does not serve it.
func TestCLELDivergence(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainMultiNodeWithoutP2PWithoutCheck(t, common.ReqRespSyncDisabledOpts(sync.ELSync)...)
	require := t.Require()
	l := t.Logger()

	startNum := sys.L2CLB.HeadBlockRef(types.LocalUnsafe).Number

	// Wait for the sequencer to produce the next block so the verifier initial EL sync can complete.
	sys.L2CL.Reached(types.LocalUnsafe, startNum+1, 30)

	// Complete initial EL sync by providing the first missing block.
	// At this point, the EL has sufficient state to validate block startNum+1.
	sys.L2CLB.SignalTarget(sys.L2EL, startNum+1)
	require.Equal(startNum+1, sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number)

	// Ensure at least one L1 block is processed to ensure a derivation pipeline reset.
	// Without this, a pipeline reset could interfere with observing divergence behavior.
	sys.L2CLB.AwaitMinL1Processed(sys.L2CLB.SyncStatus().CurrentL1.Number + 1)

	// Choose a future EL sync target for which the EL lacks state to validate.
	delta := uint64(5)
	targetNumber := startNum + delta
	sys.L2CL.Advanced(types.LocalUnsafe, targetNumber, 30)
	targetBlock := sys.L2EL.BlockRefByNumber(targetNumber)

	// The CL advances its unsafe head to the target block, even though there is a gap.
	var ss *eth.SyncStatus
	require.Eventually(func() bool {
		l.Info("Sending payload", "target", targetNumber, "startNum", startNum)
		sys.L2CLB.SignalTarget(sys.L2EL, targetNumber)
		ss = sys.L2CLB.SyncStatus()
		return targetBlock.ID() == ss.UnsafeL2.ID()
	}, 10*time.Second, 2*time.Second, "L2CLB unsafe head did not expose target block")

	// The EL unsafe head remains unchanged because it cannot validate the block due to the gap.
	require.Equal(startNum+1, sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number)

	// Verify that the EL cannot retrieve the block by hash.
	// Although the CL references it, the block is not canonicalized or served by the EL.
	_, err := sys.L2ELB.Escape().L2EthClient().L2BlockRefByHash(t.Ctx(), ss.UnsafeL2.Hash)
	require.Error(err, ethereum.NotFound)
}
