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
	sys := presets.NewSimpleInteropSupernodeProofs(t)
	sfp.RunSuperFaultProofTest(t, sys)
}

func TestInteropFaultProofs_PreForkActivation(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInteropSupernodeProofs(t, presets.WithSuggestedInteropActivationOffset(365*24*60*60))
	sfp.RunPreForkActivationTest(t, sys)
}

func TestInteropFaultProofs_ActivationBoundary(gt *testing.T) {
	t := devtest.SerialT(gt)
	// Set interop activation ~6s (3 blocks) after genesis. A small offset keeps
	// the fork reachable within CI timeouts while still having pre-interop blocks.
	sys := presets.NewSimpleInteropSupernodeProofs(t,
		presets.WithSuggestedInteropActivationOffset(6),
	)
	sfp.RunInteropActivationBoundaryTest(t, sys)
}

func TestInteropFaultProofs_ConsolidateValidCrossChainMessage(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInteropSupernodeProofs(t)
	sfp.RunConsolidateValidCrossChainMessageTest(t, sys)
}

func TestInteropFaultProofs_DepositMessage(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInteropSupernodeProofs(t)
	sfp.RunDepositMessageTest(t, sys)
}

func TestInteropFaultProofs_VariedBlockTimes(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInteropSupernodeProofs(
		t,
		presets.WithL2BlockTimes(map[eth.ChainID]uint64{
			sysgo.DefaultL2AID: 1,
			sysgo.DefaultL2BID: 2,
		}),
	)
	sfp.RunVariedBlockTimesTest(t, sys)
}

func TestInteropFaultProofs_VariedBlockTimes_FasterChainB(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInteropSupernodeProofs(
		t,
		presets.WithL2BlockTimes(map[eth.ChainID]uint64{
			sysgo.DefaultL2AID: 2,
			sysgo.DefaultL2BID: 1,
		}),
	)
	sfp.RunVariedBlockTimesTest(t, sys)
}

func TestInteropFaultProofs_InvalidBlock(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInteropSupernodeProofs(t)
	sfp.RunInvalidBlockTest(t, sys)
}

func TestInteropFaultProofs_IntraBlock(gt *testing.T) {
	for _, tc := range sfp.IntraBlockCases() {
		gt.Run(tc.Name, func(gt *testing.T) {
			t := devtest.SerialT(gt)
			sys := presets.NewSimpleInteropSupernodeProofs(t)
			sfp.RunIntraBlockConsolidationTest(t, sys, tc)
		})
	}
}

func TestInteropFaultProofs_DepositMessage_InvalidExecution(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInteropSupernodeProofs(t)
	sfp.RunDepositMessageInvalidExecutionTest(t, sys)
}

func TestInteropFaultProofs_MessageExpiry(gt *testing.T) {
	t := devtest.SerialT(gt)
	const messageExpiryWindow = uint64(12) // 12 seconds for fast test
	sys := presets.NewSimpleInteropSupernodeProofs(t,
		presets.WithMessageExpiryWindow(messageExpiryWindow),
	)
	sfp.RunMessageExpiryTest(t, sys, messageExpiryWindow)
}
