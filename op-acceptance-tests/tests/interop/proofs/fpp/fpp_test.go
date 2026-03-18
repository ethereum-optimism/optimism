package fpp

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestFPP(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInteropSupernodeProofs(t, presets.WithChallengerCannonKonaEnabled())

	startTimestamp := max(sys.L2ChainA.Escape().RollupConfig().TimestampForBlock(1), sys.L2ChainB.Escape().RollupConfig().TimestampForBlock(1))
	endTimestamp := sys.L2ChainA.Escape().RollupConfig().TimestampForBlock(5)
	sys.SuperRoots.AwaitValidatedTimestamp(endTimestamp)

	dgf := sys.DisputeGameFactory()
	dgf.RunFPP(startTimestamp, endTimestamp)
}

func TestNextSuperRootNotFound(gt *testing.T) {
	t := devtest.SerialT(gt)
	// TODO(#19180): Unskip this once supernode is updated.
	t.Skip("Supernode does not yet return optimistic blocks until blocks are fully validated")
	sys := presets.NewSimpleInteropSupernodeProofs(t, presets.WithChallengerCannonKonaEnabled())
	blockTime := sys.L2ChainA.Escape().RollupConfig().BlockTime

	// Need to setup situation where the next super root is not found but the next block is safe on the first chain, but not safe on the second.
	// Wait for at least 1 block to be fully validated on both chains so we have a good starting point.
	initTimestamp := max(sys.L2ChainA.Escape().RollupConfig().TimestampForBlock(1), sys.L2ChainB.Escape().RollupConfig().TimestampForBlock(1))
	sys.SuperRoots.AwaitValidatedTimestamp(initTimestamp)

	// Stop the second sequencer so we have a point where new blocks aren't available (and thus no super root is found)
	chainBLastBlockHash := sys.L2CLB.StopSequencer()
	defer sys.L2CLB.StartSequencer() // Start the sequencer again for other tests.

	chainBLastBlock := sys.L2ELB.BlockRefByHash(chainBLastBlockHash)

	// Wait for data to be fully validated up to the last block on second chain.
	sys.SuperRoots.AwaitValidatedTimestamp(chainBLastBlock.Time)

	// Wait for safe head to advance on first chain to be sure the next block is also safe.
	sys.L2CLA.Advanced(types.LocalSafe, 1, 10)

	startTimestamp := chainBLastBlock.Time
	endTimestamp := startTimestamp + blockTime

	// Verify we have a super root at the last block timestamp
	resp := sys.SuperRoots.SuperRootAtTimestamp(startTimestamp)
	t.Require().NotNil(resp.Data)

	// But not at the next block
	resp = sys.SuperRoots.SuperRootAtTimestamp(endTimestamp)
	t.Require().Nil(resp.Data)

	// Run FPP from timestamp of safe head on second chain, to 2 seconds later.
	dgf := sys.DisputeGameFactory()
	dgf.RunFPP(startTimestamp, endTimestamp)
}
