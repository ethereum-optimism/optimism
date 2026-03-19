package preinterop

import (
	"testing"

	sfp "github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/superfaultproofs"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestPreinteropFaultProofs(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newSimpleInteropPreinterop(t)
	sfp.RunSuperFaultProofTest(t, sys)
}

func TestPreinteropFaultProofs_TraceExtensionActivation(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newSimpleInteropPreinterop(t)
	sfp.RunTraceExtensionActivationTest(t, sys)
}

func TestPreinteropFaultProofs_UnsafeProposal(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newSimpleInteropPreinterop(t)
	sfp.RunUnsafeProposalTest(t, sys)
}

func TestPreinteropFaultProofs_VariedBlockTimes(gt *testing.T) {
	t := devtest.SerialT(gt)
	// TODO(#19010): Unskip once varied block time fault proofs are stable.
	t.Skip("Skipping flaky varied block time fault proof test")
	sys := presets.NewSimpleInteropIsthmusSuper(
		t,
		presets.WithChallengerCannonKonaEnabled(),
		presets.WithL2BlockTimes(map[eth.ChainID]uint64{
			sysgo.DefaultL2AID: 1,
			sysgo.DefaultL2BID: 2,
		}),
	)
	sfp.RunVariedBlockTimesTest(t, sys)
}

func TestPreinteropFaultProofs_VariedBlockTimes_FasterChainB(gt *testing.T) {
	t := devtest.SerialT(gt)
	// TODO(#19010): Unskip once varied block time fault proofs are stable.
	t.Skip("Skipping flaky varied block time fault proof test")
	sys := presets.NewSimpleInteropIsthmusSuper(
		t,
		presets.WithChallengerCannonKonaEnabled(),
		presets.WithL2BlockTimes(map[eth.ChainID]uint64{
			sysgo.DefaultL2AID: 2,
			sysgo.DefaultL2BID: 1,
		}),
	)
	sfp.RunVariedBlockTimesTest(t, sys)
}
