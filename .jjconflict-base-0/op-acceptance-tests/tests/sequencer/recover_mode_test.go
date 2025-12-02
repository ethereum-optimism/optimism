package sequencer

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/stretchr/testify/require"
)

// TestRecoverModeWhenChainHealthy checks that the chain
// can progress as normal when recover mode is activated.
// Recover mode is designed to recover from a sequencing
// window expiry when there are ample L1 blocks to eagerly
// progress the l1 origin to. But when the l1 origin is
// close to the tip of the l1 chain, the eagerness would cause
// a delay in unsafe block production while the sequencer waits
// for the next l1 origin to become available. Recover mode
// has since been patched, and the sequencer will not demand the
// next l1 origin until it is actually available. This tests
// protects against a regeression in that behavior.
func TestRecoverModeWhenChainHealthy(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimal(t)
	tracer := t.Tracer()
	ctx := t.Ctx()

	err := sys.L2CL.SetSequencerRecoverMode(true)
	require.NoError(t, err)
	blockTime := sys.L2Chain.Escape().RollupConfig().BlockTime
	numL2Blocks := uint64(20)
	waitTime := time.Duration(blockTime*numL2Blocks+5) * time.Second

	num := sys.L2CL.SyncStatus().UnsafeL2.Number
	new_num := num
	require.Eventually(t, func() bool {
		ctx, span := tracer.Start(ctx, "check head")
		defer span.End()

		new_num, num = sys.L2CL.SyncStatus().UnsafeL2.Number, new_num
		t.Logger().InfoContext(ctx, "unsafe head", "number", new_num, "safe head", sys.L2CL.SyncStatus().SafeL2.Number)
		return new_num >= numL2Blocks
	}, waitTime, time.Duration(blockTime)*time.Second)
}
