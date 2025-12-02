package base

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/stretchr/testify/require"
)

func TestCLAdvance(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimal(t)
	tracer := t.Tracer()
	ctx := t.Ctx()

	blockTime := sys.L2Chain.Escape().RollupConfig().BlockTime
	waitTime := time.Duration(blockTime+1) * time.Second

	num := sys.L2CL.SyncStatus().UnsafeL2.Number
	new_num := num
	require.Eventually(t, func() bool {
		ctx, span := tracer.Start(ctx, "check head")
		defer span.End()

		new_num, num = sys.L2CL.SyncStatus().UnsafeL2.Number, new_num
		t.Logger().InfoContext(ctx, "unsafe head", "number", new_num)
		return new_num > num
	}, 30*time.Second, waitTime)
}
