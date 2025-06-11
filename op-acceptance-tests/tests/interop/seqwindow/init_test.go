package seqwindow

import (
	"testing"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

func TestMain(m *testing.M) {
	presets.DoMain(m,
		presets.WithSimpleInterop(),
		// Short enough that we can run the test,
		// long enough that the batcher can still submit something before we make things expire.
		presets.WithSequencingWindow(10, 30),
		stack.MakeCommon(sysgo.WithBatcherOption(func(id stack.L2BatcherID, cfg *bss.CLIConfig) {
			// Span-batches during recovery don't appear to align well with the starting-point.
			// It can be off by ~6 L2 blocks, possibly due to off-by-one
			// in L1 block sync considerations in batcher stop or start.
			// So we end up having to encode block by block, so the full batch data does not get dropped.
			cfg.BatchType = derive.SingularBatchType
		})))
}
