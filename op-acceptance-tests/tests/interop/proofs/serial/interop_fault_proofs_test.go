package serial

import (
	"testing"

	sfp "github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/superfaultproofs"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestInteropFaultProofs(gt *testing.T) {
	t := devtest.ParallelT(gt)
	// TODO(#19180): Unskip this once supernode is updated.
	t.Skip("Supernode does not yet return optimistic blocks until blocks are fully validated")
	sys := presets.NewSimpleInteropSupernodeProofs(t, presets.WithChallengerCannonKonaEnabled())
	sfp.RunSuperFaultProofTest(t, sys)
}

func TestInteropFaultProofs_ConsolidateValidCrossChainMessage(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInteropSupernodeProofs(t, presets.WithChallengerCannonKonaEnabled())
	sfp.RunConsolidateValidCrossChainMessageTest(t, sys)
}

func TestInteropFaultProofs_VariedBlockTimes(gt *testing.T) {
	t := devtest.SerialT(gt)
	// TODO(#19010): Unskip once varied block time fault proofs are stable.
	t.Skip("Skipping flaky varied block time fault proof test")
	sys := presets.NewSimpleInteropSupernodeProofs(
		t,
		presets.WithChallengerCannonKonaEnabled(),
		presets.WithL2BlockTimes(map[eth.ChainID]uint64{
			sysgo.DefaultL2AID: 1,
			sysgo.DefaultL2BID: 2,
		}),
	)
	sfp.RunVariedBlockTimesTest(t, sys)
}

func TestInteropFaultProofs_VariedBlockTimes_FasterChainB(gt *testing.T) {
	t := devtest.SerialT(gt)
	// TODO(#19010): Unskip once varied block time fault proofs are stable.
	t.Skip("Skipping flaky varied block time fault proof test")
	sys := presets.NewSimpleInteropSupernodeProofs(
		t,
		presets.WithChallengerCannonKonaEnabled(),
		presets.WithL2BlockTimes(map[eth.ChainID]uint64{
			sysgo.DefaultL2AID: 2,
			sysgo.DefaultL2BID: 1,
		}),
	)
	sfp.RunVariedBlockTimesTest(t, sys)
}
