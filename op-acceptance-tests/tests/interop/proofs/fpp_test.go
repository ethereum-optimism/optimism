package proofs

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestFPP(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)

	startTimestamp := max(sys.L2ChainA.Escape().RollupConfig().TimestampForBlock(1), sys.L2ChainB.Escape().RollupConfig().TimestampForBlock(1))
	endTimestamp := sys.L2ChainA.Escape().RollupConfig().TimestampForBlock(5)
	sys.SuperRoots.AwaitValidatedTimestamp(endTimestamp)

	dgf := sys.DisputeGameFactory()
	dgf.RunFPP(startTimestamp, endTimestamp)
}
