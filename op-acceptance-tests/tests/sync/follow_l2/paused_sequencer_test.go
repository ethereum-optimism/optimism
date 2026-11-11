package follow_l2

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// TestFollowL2_LabelsMatchELByHashWithSequencerPaused stops the sequencer, waits until the
// follow-source verifier is safe and finalized at the paused head, and then requires its
// consensus-layer labels to name the same blocks, by hash, as its execution layer's labels and
// the sequencer's paused head. With the sequencer paused the labels stop moving, so the two
// layers are compared at rest rather than racing new blocks.
func TestFollowL2_LabelsMatchELByHashWithSequencerPaused(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newSingleChainTwoVerifiersFollowL2(t)
	require := t.Require()
	// With the sequencer paused only the waited label itself can move, and it moves in jumps: when a
	// batch lands on L1 for local-safe, and when that L1 block finalizes, about two minutes later,
	// for finalized. The stall bound covers one such jump; a label stuck past it fails fast.
	const maxWait = 8 * time.Minute
	const stallTimeout = 4 * time.Minute

	// Local-safe above genesis means kona finished engine sync and follows the source. Pausing
	// earlier strands kona-node at genesis (#24030), where every label would match trivially.
	require.NoError(sys.L2CLC.ReachedWithProgressFn(safety.LocalSafe, safety.LocalUnsafe, 1, maxWait, stallTimeout)())

	pausedHead := sys.L2EL.BlockRefByHash(sys.L2CL.StopSequencer())
	t.Logger().Info("Paused the sequencer", "head", pausedHead)
	for _, label := range []struct {
		consensusLayer safety.Level
		executionLayer eth.BlockLabel
	}{
		{safety.LocalSafe, eth.Safe},
		{safety.Finalized, eth.Finalized},
	} {
		dsl.CheckAll(t,
			sys.L2CLC.ReachedWithProgressFn(label.consensusLayer, label.consensusLayer, pausedHead.Number, maxWait, stallTimeout),
			sys.L2ELC.ReachedWithProgressFn(label.executionLayer, label.executionLayer, pausedHead.Number, maxWait, stallTimeout),
		)
		require.Equal(pausedHead.ID(), sys.L2CLC.HeadBlockRef(label.consensusLayer).ID(), "%s label must reach the paused head itself", label.consensusLayer)
	}

	status := sys.L2CLC.SyncStatus()
	require.Equal(status.SafeL2, status.LocalSafeL2, "delegated local-safe and safe must match outside interop")
	for _, label := range []struct {
		consensusLayer eth.L2BlockRef
		executionLayer eth.BlockLabel
	}{
		{status.UnsafeL2, eth.Unsafe},
		{status.SafeL2, eth.Safe},
		{status.FinalizedL2, eth.Finalized},
	} {
		require.Equal(pausedHead, label.consensusLayer, "%s label must stop at the paused head", label.executionLayer)
		require.Equal(label.consensusLayer, sys.L2ELC.BlockRefByLabel(label.executionLayer), "%s label differs between the layers", label.executionLayer)
	}
	// Every label is the paused head, so it is on the sequencer's chain as long as the sequencer
	// still has the paused head as its own head.
	require.Equal(pausedHead, sys.L2EL.BlockRefByLabel(eth.Unsafe), "the sequencer's head moved while paused")
}
